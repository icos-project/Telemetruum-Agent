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

type Peripheral struct {
	Device       string
	ResourcePath string
	Available    bool
}

type NodeMountedCollector struct {
	AttachedPeripherals []*Peripheral
	gauge               metric.Int64ObservableGauge
}

func (c *NodeMountedCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("node_mounted", api.WithDescription("info about the attached peripherals"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *NodeMountedCollector) CreateObservations(ctx context.Context, o api.Observer, logger zerolog.Logger) {

	for _, p := range c.AttachedPeripherals {
		opt := api.WithAttributes(
			attribute.Key("device").String(p.Device),
			attribute.Key("resource_path").String(p.ResourcePath))

		var val int64 = 1

		if !p.Available {
			val = 0
		}

		o.ObserveInt64(c.gauge, val, opt)
	}
}
