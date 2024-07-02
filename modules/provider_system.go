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
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/alecthomas/kingpin/v2"
)

type SystemProvider struct {
	BaseProvider
}

var (
	ipHint = kingpin.Flag("ip-hint", "An ip:port to use to help identify the device's ip (the specified endpoint is never called)").Default("8.8.8.8:80").String()
)

func (p *SystemProvider) Start(context.Context, *sync.WaitGroup) {

}

func (p *SystemProvider) ProvideWorkloadInfoLabels(ctx context.Context, wic *WorkloadInfoCollector) {

	b, err := os.ReadFile(filepath.Join(*pathRootFs, "/etc/machine-id")) // just pass the file name
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*pathRootFs, "/etc/machine-id"))
		fmt.Print(err)
	}

	wic.HostId = strings.Trim(string(b), "\n")
}

func (p *SystemProvider) ProvideHostInfo(ctx context.Context, hic *HostInfoCollector) {
	p.Logger.Debug().Msg("Collecting host metrics")

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(*pathRootFs, "/etc/machine-id")) // just pass the file name
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*pathRootFs, "/etc/machine-id"))
		fmt.Print(err)
	}
	loc := p.getMachineLocation()

	hic.Os = runtime.GOOS
	hic.Arch = runtime.GOARCH
	hic.Ip = p.getOutboundIP().String()
	hic.Latitutde = loc[0]
	hic.Longitude = loc[1]
	hic.Hostname = hostname
	hic.Id = strings.Trim(string(b), "\n")
}

func (p *SystemProvider) getMachineLocation() []string {
	content, err := os.ReadFile(filepath.Join(*pathRootFs, "/etc/machine-location")) // just pass the file name
	if err != nil {
		p.Logger.Warn().Msgf("Cannot find %s file", filepath.Join(*pathRootFs, "/etc/machine-location"))
		return []string{"", ""}
	}
	return strings.Split(strings.Trim(string(content), "\n"), ":")
}

// Get preferred outbound ip of this machine
func (p *SystemProvider) getOutboundIP() net.IP {
	conn, err := net.Dial("udp", *ipHint)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
