# GRPC xds example

## Kubernetes environment:

- Kubernetes version v1.24.0 (Use [Kind](https://kind.sigs.k8s.io/) to create a local Kubernetes cluster)
- Monitoring
  operator: [Kube prometheus stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
- Go version: go1.18.3

## Configurations

You can configure grpc client and grpc xds client by change the command line arguments in deployment file

- --target (-t): uri of the target grpc server. It must begin with `xds:///` prefix for xds client.
  Example: `xds:///localhost:50051`
- --concurrency (-c): Number of concurrent requests to run the client service. Example: 100
- --duration (-d): A duration to send requests to the server. Duration is a possibly signed sequence of decimal numbers.
  Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.). Ex:  `300ms`, `-1.5h` or `2h45m`.

## Installation

### Applications

- Create namespace

```bash
kubectl apply -f ./deploy/namespace.yml
```

- Deploy grpc server

```bash
kubectl apply -f ./deploy/server.yml 
```

- Deploy xds server

```bash
kubectl apply -f ./deploy/xds.yml
```

- Deploy grpc client for testing grpc client load balancing

```bash
kubectl apply -f ./deploy/client.yml 
```

- Deploy xds client for testing grpc client load balancing through xds protocol

```bash
kubectl apply -f ./deploy/xds-client.yml
```

### Grafana dashboard

Open grafana dashboard and import [this file](./deploy/grafana/dashboard.json) to load the dashboard from

