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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"telemetruum/agent/modules"
	"time"

	"github.com/alecthomas/kingpin/v2"
	prom_client "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/exporters/prometheus"
	metric2 "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

var (
	bindAddress          = kingpin.Flag("bind", "Bind address").Default(":2545").String()
	dockerEnabled        = kingpin.Flag("docker", "Enable Docker Provider").Default("true").Bool()
	kubernetesEnabled    = kingpin.Flag("kubernetes", "Enable Kubernetes Provider").Default("true").Bool()
	systemEnabled        = kingpin.Flag("system", "Enable System Provider").Default("true").Bool()
	hostInfoInterval     = kingpin.Flag("host-info-interval", "Interval for Host Info Metrics").Default("5m").String()
	orchInfoInterval     = kingpin.Flag("orch-info-interval", "Interval for Orchestrator Info Metrics").Default("2m").String()
	workloadInfoInterval = kingpin.Flag("workload-info-interval", "Interval for Workload Info Metrics").Default("1m").String()
	nodeMountedInterval  = kingpin.Flag("node-mount-interval", "Interval for Node Mounted Metrics").Default("1m").String()
)

func serveMetrics(logger zerolog.Logger) {
	logger.Info().Msgf("serving metrics at %s/metrics", *bindAddress)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(*bindAddress, nil) //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
	if err != nil {
		fmt.Printf("error serving http: %v", err)
		return
	}
}

func setupOtel() metric2.Meter {

	prom_client.Unregister(collectors.NewGoCollector())
	prom_client.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	exporter, err := prometheus.New(
		prometheus.WithoutTargetInfo(),
		prometheus.WithoutScopeInfo())
	if err != nil {
		log.Fatal()
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("telemetruum-agent")

	return meter
}

func main() {
	kingpin.Parse()

	logger := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC822},
	).Level(zerolog.TraceLevel).With().Timestamp().Logger()

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	meter := setupOtel()

	// Setup Metric Collectors
	hicr_interval, _ := time.ParseDuration(*hostInfoInterval)
	hicr := &modules.AsyncCollectorRunner[*modules.HostInfoCollector]{
		Collector: &modules.HostInfoCollector{},
		Interval:  hicr_interval,
		Logger:    logger.With().Str("Collector", "HostInfo").Logger()}

	oicr_interval, _ := time.ParseDuration(*orchInfoInterval)
	oicr := &modules.AsyncCollectorRunner[*modules.OrchInfoCollector]{
		Collector: &modules.OrchInfoCollector{},
		Interval:  oicr_interval,
		Logger:    logger.With().Str("Collector", "OrchInfo").Logger()}

	wicr_interval, _ := time.ParseDuration(*workloadInfoInterval)
	wicr := &modules.AsyncCollectorRunner[*modules.WorkloadInfoCollector]{
		Collector: &modules.WorkloadInfoCollector{},
		Interval:  wicr_interval,
		Logger:    logger.With().Str("Collector", "WorkloadInfo").Logger()}

	pecr_interval, _ := time.ParseDuration(*nodeMountedInterval)
	pecr := &modules.AsyncCollectorRunner[*modules.NodeMountedCollector]{
		Collector: &modules.NodeMountedCollector{},
		Interval:  pecr_interval,
		Logger:    logger.With().Str("Collector", "NodeMounted").Logger()}
	// Setup Providers

	var kubernetesProvider *modules.KubernetesProvider
	var systemProvider *modules.SystemProvider
	var dockerProvider *modules.DockerProvider
	var providerErr error

	if *kubernetesEnabled {
		kubernetesProvider, providerErr = modules.InizializeKubernetesProvider(logger.With().Str("Provider", "Kubernetes").Logger())
		if providerErr != nil {
			logger.Warn().Msgf("Error initializing Kubernetes (\"%s\"). The Kubernetes provider will not be used", providerErr)
		} else {

			kubernetesProvider.Start(ctx, wg)
			logger.Info().Msg("Kubernetes Provider successfully started")

			oicr.AppendAsyncDataProvider(kubernetesProvider.ProvideOCMOrchInfo)
			wicr.AppendAsyncDataProvider(kubernetesProvider.ProvideWorkloadInfo)
			oicr.AppendAsyncDataProvider(kubernetesProvider.ProvideNuvlaOrchestratorInfo)
		}

	} else {
		logger.Debug().Msg("Kubernetes Provider disabled")
	}

	if *systemEnabled {
		systemProvider = &modules.SystemProvider{
			BaseProvider: modules.BaseProvider{Logger: logger.With().Str("Provider", "System").Logger()}}

		systemProvider.Start(ctx, wg)
		logger.Info().Msg("System Provider successfully started")

		hicr.AppendAsyncDataProvider(systemProvider.ProvideHostInfo)
		wicr.AppendAsyncDataProvider(systemProvider.ProvideWorkloadInfoLabels)

	} else {
		logger.Debug().Msg("System Provider disabled")
	}

	if *dockerEnabled {
		dockerProvider, providerErr = modules.InizializeDockerProvider(logger.With().Str("Provider", "Docker").Logger())
		if providerErr != nil {
			logger.Warn().Msgf("Error initializing Docker (\"%s\"). The Docker provider will not be used", providerErr)
		} else {
			dockerProvider.Start(ctx, wg)
			logger.Info().Msg("Docker Provider successfully started")

			wicr.AppendAsyncDataProvider(dockerProvider.ProvideWorkloadInfo)
			oicr.AppendAsyncDataProvider(dockerProvider.ProvideNuvlaOrchestratorInfo)
			pecr.AppendAsyncDataProvider(dockerProvider.ProvideNuvlaAttachedPeripherals)
		}
	} else {
		logger.Debug().Msg("Docker Provider disabled")
	}

	// Start Metrics Collectors

	hicr.Init(meter)
	hicr.Start(context.TODO())
	oicr.Init(meter)
	oicr.Start(context.TODO())
	wicr.Init(meter)
	wicr.Start(context.TODO())
	pecr.Init(meter)
	pecr.Start(context.TODO())

	go serveMetrics(logger)

	<-ch
	cancel()

	wg.Wait()
}
