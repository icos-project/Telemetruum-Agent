# Telemetruum Agent

The Telemetruum Agent is a component of the ICOS Logging and Telemetry subsystem. This component is deployed in the ICOS Edge nodes and generates some metrics (see below) that are useful to ICOS for the matchmaking and orchestration processes.


## Metrics


| name          | labels                                          | meaning                                                                                         |
| ------------- | ----------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| host_info     | os, ip, arch, latitude, longitude, hostname, id | publish information about the host                                                              |
| orch_info     | type, agent-id, agent-name, cluster_id          | publish information about the multi-cluster orchestrator (i.e. Nuvla or OCM)                    |
| workload_info | name, cluster_id, host_id                       | publish information on the workloads (i.e. containers) running in the host                      |
| node_mounted  | device, resource_path                           | publish information about the peripherals attached to this host. This is enabled for Nuvla only |



## Build

The component can be built uisng the `go` tool:

```bash
go build -o ./output/telemetruum-agent-local
```

and packaged with Docker:

```bash
docker build --build-arg CUSTOM_PLATFORM_SLUG=local .
```


## Usage

The Telmetruum Agent can be configured passing arguments in its command line. The flags can be used to enable/disable specific features (e.g. Kubernetes provider) and to configure some aspects of the provides (e.g. Kubernetes API Server endpoint). 

```
usage: main [<flags>]


Flags:
  --[no-]help                    Show context-sensitive help (also try --help-long and --help-man).
  --path-rootfs="/"              Path of the root fs
  --kube-config=KUBE-CONFIG      Kubernetes Configuration file
  --ip-hint="8.8.8.8:80"         An ip:port to use to help identify the device's ip (the specified endpoint is never called)
  --bind=":2545"                 Bind address
  --[no-]docker                  Enable Docker Provider
  --[no-]kubernetes              Enable Kubernetes Provider
  --[no-]system                  Enable System Provider
  --host-info-interval="5m"      Interval for Host Info Metrics
  --orch-info-interval="2m"      Interval for Orchestrator Info Metrics
  --workload-info-interval="1m"  Interval for Workload Info Metrics
  --node-mount-interval="1m"     Interval for Node Mounted Metrics
```


# Legal
The Telemetruum Agent is released under the Apache 2.0 license.
Copyright © 2022-2024 Engineering Ingegneria Informatica S.p.A. All rights reserved.

🇪🇺 This work has received funding from the European Union's HORIZON research and innovation programme under grant agreement No. 101070177.
