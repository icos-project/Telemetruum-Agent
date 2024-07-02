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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var ()

type DockerProvider struct {
	BaseProvider
	DockerClient *client.Client
	Id           string
}

func InizializeDockerProvider(logger zerolog.Logger) (*DockerProvider, error) {
	cli, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	info, err := cli.Info(context.TODO())
	if err != nil {
		logger.Debug().Msgf("Error initializing Docker with message: %s", err)
		return nil, err
	}

	return &DockerProvider{Id: info.Swarm.NodeID, DockerClient: cli, BaseProvider: BaseProvider{Logger: logger}}, nil
}

func (kd *DockerProvider) Start(ctx context.Context, wg *sync.WaitGroup) {
}

func (kd *DockerProvider) ProvideWorkloadInfo(ctx context.Context, c *WorkloadInfoCollector) {
	containers, err := kd.DockerClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		panic(err)
	}
	res := []*WorkloadInfo{}

	for _, ctr := range containers {
		wi := &WorkloadInfo{Name: ctr.Names[0], Annotations: map[string]string{}}
		res = append(res, wi)

		for k, v := range ctr.Labels {
			if m1.MatchString(k) {
				newK := m1.ReplaceAllString(k, "icos.$1.$2")
				wi.Annotations[newK] = v
			}
		}

	}

	c.RunningWorkloads = res
	c.ClusterId = kd.Id
}

func (kd *DockerProvider) ProvideNuvlaOrchestratorInfo(ctx context.Context, oic *OrchInfoCollector) {
	CommonProvideNuvlaOrchestratorInfo(ctx, filepath.Join(*pathRootFs, "/nuvla_peripherals/.context"), oic, kd.Logger)
}

type NuvlaPeripheralFileStruct struct {
	Identifier string `json:"identifier"`
	Available  bool   `json:"available"`
	Interface  string `json:"interface"`
	DevicePath string `json:"device-path"`
	Name       string `json:"name"`
}

func (kd *DockerProvider) ProvideNuvlaAttachedPeripherals(ctx context.Context, oic *NodeMountedCollector) {

	peripheral_files := filepath.Join(*pathRootFs, "/nuvla_peripherals/.peripherals/local_peripherals.json")

	nuvlaFile, err := os.ReadFile(peripheral_files) // just pass the file name
	if err != nil {
		kd.Logger.Warn().Msgf("Error reading Nuvla context file at %s: %s\n", peripheral_files, err)
		return
	}

	var nuvlaPeripherals map[string]NuvlaPeripheralFileStruct

	err = json.Unmarshal(nuvlaFile, &nuvlaPeripherals)

	if err != nil {
		kd.Logger.Warn().Msgf("Error unmarshalling Nuvla peripherals file: %s", err)
		return
	}

	res := []*Peripheral{}
	for _, v := range nuvlaPeripherals {

		if v.Interface != "USB" {
			continue
		}

		device := strings.ToLower(strings.ReplaceAll(v.Name, " ", "-")) + "_" + strings.ToLower(strings.ReplaceAll(v.Identifier, ":", "_"))

		p := &Peripheral{Device: device, Available: v.Available, ResourcePath: v.DevicePath}

		res = append(res, p)
	}

	oic.AttachedPeripherals = res
}
