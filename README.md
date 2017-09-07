# kubernetes-grafana-updater

`kubernetes-grafana-updater` creates/updates [Grafana](https://grafana.com/) 
dashboards and datasources from Kubernetes
[ConfigMaps](https://kubernetes.io/docs/tasks/configure-pod-container/configmap/) 
and [Services](https://kubernetes.io/docs/concepts/services-networking/service/).

## Status 

`kubernetes-grafana-updater` is functional. It is under active development and
features are subject to change at anytime.

## Motivation

We run multiple Prometheus instances in Kubernetes clusters and want to
automatically add those as datasources to Grafana.  Also, we have many dashboards.
Some other tools[https://github.com/coreos/prometheus-operator/tree/master/contrib/grafana-watcher]
allow one to add dashboards using a ConfigMap, however, we wanted to use multiple
ConfigMaps owned by multiple teams.

## Installation

This requires a working [local Go](https://golang.org/doc/install) evironment.

Clone this repository into your GOPATH and run `./script/build`;

```shell
$ mkdir -p $GOPATH/src/github.com/bakins
$ cd $GOPATH/src/github.com/bakins
$ git clone https://github.com/bakins/kubernetes-grafana-updater.git
$ cd kubernetes-grafana-updater
$ ./script/build
$ ls kubernetes-grafana-updater.*
kubernetes-grafana-updater.darwin.amd64	kubernetes-grafana-updater.linux.amd64
```

Binary releases are availible under [Releases](https://github.com/bakins/kubernetes-grafana-updater/releases).

This is also availible as a [Docker image](https://quay.io/repository/bakins/kubernetes-grafana-updater?tab=tags).


## Usage

```shell
$ ./kubernetes-grafana-updater.darwin.amd64 --help
update grafana datasources and dashboards

Usage:
  kubernetes-grafana-updater [command]

Available Commands:
  dashboards  syncronize dashboards
  datasources syncronize datasources
  help        Help about any command

Flags:
  -h, --help                 help for kubernetes-grafana-updater
  -l, --log-level logLevel   log level (default info)

Use "kubernetes-grafana-updater [command] --help" for more information about a command.
```

### Datasources

`kubernetes-grafana-updater datasources` will search for all services within a 
Kubernetes cluster that match the label selector and add them as datasources in
Grafana.  It will watch the Kubernetes API and ensure new services are added.

```shell
./kubernetes-grafana-updater.darwin.amd64 datasources --help
syncronize datasources

Usage:
  kubernetes-grafana-updater datasources [flags]

Flags:
      --apiserver string     override Kubernetes API server. default is to use value from kubeconfig or in cluster value
      --grafana-url string   grafana url (default "http://localhost:3000")
  -h, --help                 help for datasources
      --kubeconfig string    path to kubeconfig. default is in cluster.
      --namespace string     namespace to search. Default is all namespaces
      --selector string      label selector (default "app=prometheus")

Global Flags:
  -l, --log-level logLevel   log level (default info)
```

### Dashboards

`kubernetes-grafana-updater dashboards` will search for all configmaps within a 
Kubernetes cluster that match the label selector and add each item in each
configmap as a dashboard in Grafana. It will watch the Kubernetes API and ensure 
new dashboards are added.

```shell
$ ./kubernetes-grafana-updater.darwin.amd64 dashboards --help
syncronize dashboards

Usage:
  kubernetes-grafana-updater dashboards [flags]

Flags:
      --apiserver string     override Kubernetes API server. default is to use value from kubeconfig or in cluster value
      --grafana-url string   grafana url (default "http://localhost:3000")
  -h, --help                 help for dashboards
      --kubeconfig string    path to kubeconfig. default is in cluster.
      --namespace string     namespace to search. Default is all namespaces
      --selector string      label selector (default "type=grafana-dashboard")

Global Flags:
  -l, --log-level logLevel   log level (default info)
```

### Typical Usage

`kubernetes-grafana-updater` is usually ran as two side-cars in a Pod with 
Grafana.  One side car is ran in datasource mode and the other in dashboard mode.

## Issues/TODO

- ConfigMaps or keys within ConfigMaps that are deleted do not cause the
corresponding dashboard to be deleted. Modifications to existing dashbaords
are handled.

## LICENSE

See [./LICENSE]
