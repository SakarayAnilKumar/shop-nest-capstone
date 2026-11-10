# Kubernetes Node & Pod Health Monitoring Collector

## 1. Project Overview

The Node & Pod Health Monitoring Collector is a Go-based application that collects resource utilization metrics from a Kubernetes cluster.

It uses the Kubernetes Client and Metrics API to monitor CPU and memory consumption across cluster nodes and pods.

The application evaluates resource utilization against configurable thresholds and reports health status.

## 2. Objectives

- Monitor Kubernetes node health.
- Collect node CPU and memory utilization.
- Monitor pod CPU and memory consumption.
- Compare pod resource usage against configured requests and limits.
- Identify healthy, warning, and critical resource conditions.
- Provide visibility into cluster resource utilization.

## 3. Technology Stack

| Technology | Purpose |
|---|---|
| Go | Monitoring application |
| Kubernetes | Container orchestration |
| kind | Local Kubernetes cluster |
| Metrics Server | Resource metrics collection |
| Kubernetes Client-go | Kubernetes API communication |
| Kubernetes Metrics Client | Node and pod metrics retrieval |

## 4. Project Structure

```text
node-health-collector/
├── cmd/
│   └── main.go
├── go.mod
├── go.sum
└── README.md
```

## 5. Kubernetes Environment

The project uses a local kind cluster with three nodes:

- shop-nest-control-plane
- shop-nest-worker
- shop-nest-worker2

Verify cluster nodes:

```bash
kubectl get nodes
```

All nodes should be in the `Ready` state.

## 6. Metrics Server Setup

Metrics Server provides CPU and memory usage metrics to Kubernetes.

Verify its deployment:

```bash
kubectl get deployment metrics-server -n kube-system
```

Verify node metrics:

```bash
kubectl top nodes
```

Verify pod metrics:

```bash
kubectl top pods -A
```

**Local kind environment note:** The Metrics Server configuration uses `--kubelet-insecure-tls` to work around kubelet certificate verification issues in this local lab environment. This setting is not recommended for production clusters.

## 7. Application Dependencies

The application uses:

- `k8s.io/client-go`
- `k8s.io/api`
- `k8s.io/apimachinery`
- `k8s.io/metrics`

Install or synchronize dependencies:

```bash
go mod tidy
```

## 8. Running the Collector

Navigate to the project directory:

```powershell
cd monitoring\node-health-collector
```

Run the application:

```powershell
go run .\cmd\
```

The application uses the local kubeconfig to connect to the Kubernetes cluster.

## 9. Monitoring Features

### Node Monitoring

The collector retrieves:

- Node name
- Node readiness
- CPU capacity
- Memory capacity
- CPU usage
- Memory usage
- CPU utilization percentage
- Memory utilization percentage

### Pod Monitoring

The collector retrieves:

- Pod namespace
- Pod name
- CPU usage
- Memory usage
- CPU request utilization
- CPU limit utilization
- Memory request utilization
- Memory limit utilization

Pod resource utilization is calculated using:

```text
CPU Request Utilization =
CPU Usage / CPU Request * 100

CPU Limit Utilization =
CPU Usage / CPU Limit * 100

Memory Request Utilization =
Memory Usage / Memory Request * 100

Memory Limit Utilization =
Memory Usage / Memory Limit * 100
```

If a request or limit is not configured, the corresponding utilization is reported as `N/A`.

### Health Evaluation

The collector evaluates resource utilization using configurable thresholds.

| Resource | Warning | Critical |
|---|---:|---:|
| CPU | 70% | 85% |
| Memory | 75% | 90% |

Health statuses:

- **HEALTHY:** Utilization is below the warning threshold.
- **WARNING:** Utilization reaches the warning threshold.
- **CRITICAL:** Utilization reaches the critical threshold.
- **N/A:** No applicable request or limit is configured for pod evaluation.

For pod health evaluation, the collector uses the configured limit when available; otherwise, it uses the request.

## 10. Verification

Run the collector and verify:

- All three nodes are listed.
- Node CPU and memory utilization are displayed.
- Node health status is displayed.
- Pod metrics are collected across namespaces.
- Pod request and limit utilization percentages are displayed.
- Pod health status is evaluated.
- Threshold tests return the expected statuses.

## 11. Threshold Testing

The following test cases validate the threshold evaluation logic:

| Test | Input | Expected Result |
|---|---:|---|
| CPU | 75% | WARNING |
| CPU | 90% | CRITICAL |
| Memory | 80% | WARNING |
| Memory | 95% | CRITICAL |

## 12. Current Project Status

- [x] kind cluster setup with three nodes
- [x] Metrics Server integration
- [x] Node metrics collection
- [x] Node utilization calculation
- [x] Node health evaluation
- [x] Pod metrics collection
- [x] Pod request and limit utilization
- [x] Pod health evaluation
- [x] Threshold validation

## 13. Future Enhancements

Potential enhancements for subsequent development stages:

- Export metrics in Prometheus format.
- Integrate Grafana dashboards.
- Containerize the Go monitoring application.
- Deploy the collector inside Kubernetes.
- Add automated alerting.
- Integrate monitoring into the CI/CD pipeline.

---

**Project:** Shop-Nest Capstone  
**Component:** Kubernetes Node & Pod Health Monitoring Collector  
**Language:** Go