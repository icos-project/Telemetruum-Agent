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

type WorkloadInfo struct {
	Name        string
	Type        string
	Annotations map[string]string
}

type WorkloadInfoCollector struct {
	RunningWorkloads []*WorkloadInfo
	HostId           string
	ClusterId        string
	gauge            metric.Int64ObservableGauge
}

func (c *WorkloadInfoCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_workload_info", api.WithDescription("info about the workloads running in the node"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *WorkloadInfoCollector) CreateObservations(ctx context.Context, o api.Observer, logger zerolog.Logger) {

	for _, w := range c.RunningWorkloads {

		var annotationAttributes []attribute.KeyValue

		annotationAttributes = append(annotationAttributes, attribute.Key("name").String(w.Name))
		annotationAttributes = append(annotationAttributes, attribute.Key("cluster_id").String(c.ClusterId))
		annotationAttributes = append(annotationAttributes, attribute.Key("host_id").String(c.HostId))

		for k, v := range w.Annotations {
			annotationAttributes = append(annotationAttributes, attribute.Key(k).String(v))
		}

		opt := api.WithAttributeSet(attribute.NewSet(annotationAttributes...))

		o.ObserveInt64(c.gauge, 1, opt)
	}
}
