/*
ICOS Telemetruum Agent
Copyright © 2022-2024 Engineering Ingegneria Informatica S.p.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This work has received funding from the European Union's HORIZON research
and innovation programme under grant agreement No. 101070177.
*/

package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

var (
	kubeConfig = kingpin.Flag("kube-config", "Kubernetes Configuration file").String()
	m1         = regexp.MustCompile(`(.+).icos.eu/(.+)`)
)

type KubernetesProvider struct {
	BaseProvider
	KubernetesClient *kubernetes.Clientset
	iAmTheLeader     bool
	Id               string
}

func InizializeKubernetesProvider(logger zerolog.Logger) (*KubernetesProvider, error) {

	var clientset *kubernetes.Clientset

	if *kubeConfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
		if err != nil {
			return nil, err
		}

		clientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}

	} else {

		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		clientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}
	}

	kubeSystemNS, err := clientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error reading cluster id from \"kube-system\" namespace: %s", err)
	}

	logger.Debug().Msgf("Initialed Kubernetes Provider for cluster with Id: %s", string(kubeSystemNS.UID))

	return &KubernetesProvider{Id: string(kubeSystemNS.UID), KubernetesClient: clientset, BaseProvider: BaseProvider{Logger: logger}}, nil
}

func (kp *KubernetesProvider) Start(ctx context.Context, wg *sync.WaitGroup) {
	kp.leaderElectionControlLoop(ctx, wg)
}

func (kp *KubernetesProvider) ProvideWorkloadInfo(ctx context.Context, c *WorkloadInfoCollector) {
	nodeName := os.Getenv("NODE_NAME")
	kp.Logger.Debug().Msgf("Listing pods in node \"%s\"\n", nodeName)

	pods, _ := kp.KubernetesClient.
		CoreV1().
		Pods("").
		List(ctx, metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + nodeName,
		})

	res := []*WorkloadInfo{}
	for _, p := range pods.Items {
		wi := &WorkloadInfo{Name: p.ObjectMeta.Name, Annotations: map[string]string{}}
		res = append(res, wi)

		for k, v := range p.ObjectMeta.Annotations {
			if m1.MatchString(k) {
				newK := m1.ReplaceAllString(k, "icos.$1.$2")
				wi.Annotations[newK] = v
			}
		}
	}

	c.RunningWorkloads = res
	c.ClusterId = kp.Id
}

func (kp *KubernetesProvider) ProvideNuvlaOrchestratorInfo(ctx context.Context, oic *OrchInfoCollector) {

	nodeName := os.Getenv("NODE_NAME")

	nuvlaEdgePodList, _ := kp.KubernetesClient.CoreV1().Pods("").List(context.TODO(),
		metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=nuvlaedge,component=agent"})

	if len(nuvlaEdgePodList.Items) == 0 {
		kp.Logger.Warn().Msgf("No NuvlaEdge pod found.\n")
		return
	}

	if len(nuvlaEdgePodList.Items) > 1 {
		kp.Logger.Warn().Msgf("More than on pod found for NuvlaEdge. This should never happen. Do not extract Nuvla info.\n")
		return
	}

	nuvlaEdgePod := nuvlaEdgePodList.Items[0]
	kp.Logger.Debug().Msgf("NuvlaEdge pod found: %s", nuvlaEdgePod.ObjectMeta.Name)

	if nodeName != nuvlaEdgePod.Spec.NodeName {
		kp.Logger.Warn().Msg("We are in a different node from the one NuvlaEdge is running. Not extracting info from Nuvla context file\n")
		return
	}

	nuvlaContextFile := filepath.Join(*pathRootFs, fmt.Sprintf("/var/lib/nuvlaedge/%s/.context", nuvlaEdgePod.ObjectMeta.Namespace))

	if _, err := os.Stat(nuvlaContextFile); !os.IsNotExist(err) {
		kp.Logger.Debug().Msgf("Nuvla Context file found at %s", nuvlaContextFile)
		CommonProvideNuvlaOrchestratorInfo(ctx, nuvlaContextFile, oic, kp.Logger)
	} else {
		kp.Logger.Warn().Msgf("Nuvla Context file not found at %s", nuvlaContextFile)
	}
}

func (kp *KubernetesProvider) ProvideOCMOrchInfo(ctx context.Context, c *OrchInfoCollector) {

	if !kp.iAmTheLeader {
		return
	}

	kp.Logger.Debug().Msg("Getting OCM Agent info from Klusterlet pod...")

	pods, _ := kp.KubernetesClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	for _, p := range pods.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, "klusterlet-work-agent") {
			ocm_agent_name := ""
			ocm_agent_id := ""
			for _, a := range p.Spec.Containers[0].Args {
				if strings.HasPrefix(a, "--spoke-cluster-name") {
					ocm_agent_name = a[21:]
				}
				if strings.HasPrefix(a, "--agent-id") {
					ocm_agent_id = a[11:]
				}

			}

			if ocm_agent_name != "" && ocm_agent_id != "" {
				c.Type = "ocm"
				c.AgentName = ocm_agent_name
				c.AgentId = ocm_agent_id
				c.ClusterId = kp.Id
			} else {
				kp.Logger.Warn().Msgf("Couldn't determine OCM Agent Id and/or Name from the container args: \"%s\"", p.Spec.Containers[0].Args)
			}
		}
	}
}

func (kp *KubernetesProvider) leaderElectionControlLoop(ctx context.Context, wg *sync.WaitGroup) {

	wg.Add(1)

	go func() {
		namespace := os.Getenv("NAMESPACE")
		podName := os.Getenv("POD_NAME")

		lock := &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      "telemetruum-agent-kubernetes-collector",
				Namespace: namespace,
			},
			Client: kp.KubernetesClient.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: podName,
			},
		}

		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   60 * time.Second,
			RenewDeadline:   15 * time.Second,
			RetryPeriod:     5 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {

				},
				OnStoppedLeading: func() {
					kp.Logger.Info().Msg("[kubernetes] Stopping leading. Stopping collecting metrics")
					kp.iAmTheLeader = false
					wg.Done()
				},
				OnNewLeader: func(identity string) {
					kp.Logger.Info().Msgf("[kubernetes] New leader is \"%s\"", identity)
					if identity == podName {
						kp.Logger.Info().Msgf("[kubernetes] We are the new leader. Starting collecting metrics")
						kp.iAmTheLeader = true
					}
				},
			},
		})

	}()
}
