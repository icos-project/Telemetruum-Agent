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

	"github.com/alecthomas/kingpin/v2"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	api "go.opentelemetry.io/otel/metric"
)

var (
	pathRootFs = kingpin.Flag("path-rootfs", "Path of the root fs").Default("/").String()
)

type HostInfoCollector struct {
	Os        string
	Ip        string
	Arch      string
	Latitutde string
	Longitude string
	Hostname  string
	Id        string

	gauge metric.Int64ObservableGauge
}

func (c *HostInfoCollector) GetMetrics(meter metric.Meter) []metric.Observable {

	if c.gauge == nil {
		gauge, err := meter.Int64ObservableGauge("tlum_host_info", api.WithDescription("info about the host"))
		if err != nil {
			log.Fatal(err)
		}
		c.gauge = gauge
	}

	return []metric.Observable{c.gauge}
}

func (c *HostInfoCollector) CreateObservations(ctx context.Context, o api.Observer, logger zerolog.Logger) {
	opt := api.WithAttributes(
		attribute.Key("os").String(c.Os),
		attribute.Key("ip").String(c.Ip),
		attribute.Key("arch").String(c.Arch),
		attribute.Key("latitude").String(c.Latitutde),
		attribute.Key("longitude").String(c.Longitude),
		attribute.Key("hostname").String(c.Hostname),
		attribute.Key("id").String(c.Id))

	o.ObserveInt64(c.gauge, 1, opt)
}
