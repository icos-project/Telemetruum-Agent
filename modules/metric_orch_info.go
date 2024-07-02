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
	"log"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	api "go.opentelemetry.io/otel/metric"
)

type OrchInfoCollector struct {
	Type      string
	AgentId   string
	AgentName string
	ClusterId string
	gauge     metric.Int64ObservableGauge
	gaugeOld  metric.Int64ObservableGauge
}

func (c *OrchInfoCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_orch_info", api.WithDescription("info about the orchestrator"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge

		gaugeOld, err := meter.Int64ObservableGauge("tlum_ocm_agent_info", api.WithDescription("info about the orchestrator. Legacy, do not use"))
		if err != nil {
			log.Fatal(err)
		}
		c.gaugeOld = gaugeOld
	}

	return []metric.Observable{c.gauge, c.gaugeOld}
}

func (c *OrchInfoCollector) CreateObservations(ctx context.Context, o api.Observer, logger zerolog.Logger) {
	if c.Type != "" {
		opt := api.WithAttributes(
			attribute.Key("type").String(c.Type),
			attribute.Key("agent-id").String(c.AgentId),
			attribute.Key("agent-name").String(c.AgentName),
			attribute.Key("cluster_id").String(c.ClusterId))

		o.ObserveInt64(c.gauge, 1, opt)

		// TODO: remove if the Aggregator is not using it anymore
		if c.Type == "ocm" {
			optOld := api.WithAttributes(
				attribute.Key("id").String(c.AgentId),
				attribute.Key("name").String(c.AgentName))

			o.ObserveInt64(c.gaugeOld, 1, optOld)
		}
	}
}
