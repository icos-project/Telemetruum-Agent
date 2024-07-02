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
	"log"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/metric"
	api "go.opentelemetry.io/otel/metric"
)

type AsyncCollectorRunner[T AsyncCollector] struct {
	Collector T
	Interval  time.Duration
	Providers []func(context.Context, T)
	Logger    zerolog.Logger
}

func (c *AsyncCollectorRunner[T]) AppendAsyncDataProvider(dp func(context.Context, T)) {
	c.Providers = append(c.Providers, dp)
}

func (c *AsyncCollectorRunner[T]) Init(meter metric.Meter) {

	if c.Interval == 0 {
		c.Interval = 60 * time.Second
		c.Logger.Warn().Msg("Interval was 0: set to 60s")
	}

	_, err := meter.RegisterCallback(func(ctx context.Context, o api.Observer) error {
		c.Collector.CreateObservations(ctx, o, c.Logger)

		return nil
	}, c.Collector.GetMetrics(meter)...)

	if err != nil {
		log.Fatal(err)
	}

}

func (c *AsyncCollectorRunner[T]) Start(ctx context.Context) {
	go func() {
		for {
			for _, exec := range c.Providers {
				exec(ctx, c.Collector)
			}
			time.Sleep(c.Interval)
		}
	}()
}

type AsyncCollector interface {
	CreateObservations(context.Context, api.Observer, zerolog.Logger)
	GetMetrics(metric.Meter) []metric.Observable
}

type Provider interface {
	Start(context.Context, *sync.WaitGroup)
}

type BaseProvider struct {
	Logger zerolog.Logger
}

type NuvlaContext struct {
	Id    string `json:"id"`
	State string `json:"state"`
}

func CommonProvideNuvlaOrchestratorInfo(ctx context.Context, nuvlaContextFile string, oic *OrchInfoCollector, logger zerolog.Logger) {

	nuvlaFile, err := os.ReadFile(nuvlaContextFile) // just pass the file name
	if err != nil {
		logger.Warn().Msgf("Error reading Nuvla context file at %s: %s\n", nuvlaContextFile, err)
		return
	}

	oic.Type = "nuvla"

	nuvlaObj := &NuvlaContext{}
	err = json.Unmarshal(nuvlaFile, &nuvlaObj)

	if err != nil {
		logger.Warn().Msgf("Error unmarshalling Nuvla context file: %s", err)
		return
	}

	oic.Type = "nuvla"
	oic.AgentId = nuvlaObj.Id
	oic.AgentName = nuvlaObj.Id
	oic.ClusterId = nuvlaObj.Id

}
