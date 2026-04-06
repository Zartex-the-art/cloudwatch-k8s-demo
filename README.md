# CloudWatch Container Insights for Self-Managed K8s

<div align="center">

![DevOps](https://img.shields.io/badge/DevOps-Project-orange?style=for-the-badge)
![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go)
![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.29.15-326CE5?style=for-the-badge&logo=kubernetes)
![AWS](https://img.shields.io/badge/AWS-CloudWatch-FF9900?style=for-the-badge&logo=amazonaws)
![Jenkins](https://img.shields.io/badge/Jenkins-CI%2FCD-D24939?style=for-the-badge&logo=jenkins)
![Docker](https://img.shields.io/badge/Docker-Hub-2496ED?style=for-the-badge&logo=docker)
![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?style=for-the-badge&logo=prometheus)
![Grafana](https://img.shields.io/badge/Grafana-Dashboards-F46800?style=for-the-badge&logo=grafana)

**Build · Test · Deploy · Monitor — Fully Automated**

[Problem Statement](#problem-statement) • [Architecture](#architecture) • [Tech Stack](#tech-stack) • [Setup](#setup-guide) • [Demo](#demo-outputs) • [Team](#team)

</div>

---

## Problem Statement

Companies running microservices on Kubernetes have **zero visibility** into what their containers are doing at runtime. When a pod crashes at 3 AM, engineers don't know:
- Which pod failed
- What CPU/memory it was consuming
- What logs it produced — because the pod restarted and evidence is gone

On **self-managed Kubernetes (kubeadm) on EC2**, the problem is worse. None of the AWS managed observability comes pre-configured. No automatic CloudWatch integration, no Container Insights, no log shipping. You have to wire everything yourself.

> In infrastructure disruption scenarios — regional outages, DDoS attacks, cascading failures — **observability is not optional. It is survival.** Mean Time To Detection of 30 minutes means 30 minutes of user-facing downtime. Our stack reduces that to under 5 minutes.

---

## Solution

A **production-grade observability stack** built from scratch on a self-managed kubeadm cluster:

| Layer | Tool | What It Does |
|-------|------|-------------|
| Application | Go HTTP Service | `/health`, `/metrics`, Prometheus-format endpoints |
| CI/CD | Jenkins + GitHub | Auto build → test → Docker push on every git push |
| Container Registry | Docker Hub | Versioned images (`zartu/cloudwatch-k8s-demo`) |
| Orchestration | Kubernetes (kubeadm) | Self-managed 2-node cluster on AWS EC2 |
| Log Shipping | Fluent Bit DaemonSet | Ships pod logs to CloudWatch before pods restart |
| Container Metrics | CloudWatch Agent DaemonSet | CPU, memory, network per pod into Container Insights |
| App Metrics | Prometheus + ServiceMonitor | Scrapes `/metrics` every 15 seconds |
| Visualization | Grafana | Live dashboards — requests/sec, uptime, total requests |
| Alerting | CloudWatch Alarm + SNS | Email when pod CPU > 50% for 5 minutes |

---

## Architecture

```
Developer
    │
    └─► GitHub (dev branch)
            │
            ▼
        Jenkins CI/CD Pipeline
        ├── go test ./...
        ├── go build
        ├── docker build -t zartu/cloudwatch-k8s-demo:vN
        └── docker push → Docker Hub
                                │
                                ▼
                    Kubernetes Cluster (kubeadm on EC2)
                    ├── Control Plane: ip-172-31-37-188
                    └── Worker Node:   ip-172-31-43-19
                            │
                            ├── Go App (2 replicas, NodePort 30080)
                            │       └── /health  /metrics
                            │
                            ├── Fluent Bit DaemonSet
                            │       └── → CloudWatch Logs
                            │               └── /cloudwatch-k8s-demo/containers
                            │
                            ├── CloudWatch Agent DaemonSet
                            │       └── → CloudWatch Container Insights
                            │               └── /aws/containerinsights/cloudwatch-k8s-demo/performance
                            │
                            └── Prometheus + Grafana (kube-prometheus-stack)
                                    ├── Scrapes /metrics every 15s
                                    ├── Grafana dashboards on :32000
                                    └── Prometheus on :32001

CloudWatch Alarm
    └── pod_cpu_utilization_over_pod_limit > 50%
            └── SNS Topic: go-app-alerts
                    └── Email → thummaabhishek465@gmail.com
```

---

## Tech Stack

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22.0 | HTTP service — `/health`, `/metrics`, unit tests |
| Docker | 29.3.1 | Multi-stage containerization (7.22 MB final image) |
| Jenkins | 2.541.3 (WAR) | 6-stage CI/CD pipeline |
| Kubernetes | v1.29.15 | Self-managed container orchestration (kubeadm) |
| Calico | Latest | CNI — pod-to-pod networking via BGP |
| Helm | v3.20.1 | Installed kube-prometheus-stack |
| Prometheus | v3.11.0 | Metrics scraping and time-series storage |
| Grafana | 12.4.2 | Dashboards — Go App Metrics, Node Exporter |
| Fluent Bit | aws-for-fluent-bit | Log shipping to CloudWatch |
| CloudWatch Agent | CWAgent/1.300064 | EMF metrics → Container Insights |
| AWS EC2 | c7i-flex.large x2 | Compute — ap-south-1 (Mumbai) |
| AWS CloudWatch | ap-south-1 | Logs + Container Insights + Alarms |
| AWS SNS | go-app-alerts | Email notification on alarm |
| Docker Hub | zartu account | Container registry |
| WSL 2 | Ubuntu 22.04 | Local development environment |
| containerd | runtime | Container runtime on K8s nodes |

---

## Project Structure

```
cloudwatch-k8s-demo/
├── main.go                          # Go HTTP service (3 endpoints)
├── main_test.go                     # Unit tests
├── go.mod
├── Dockerfile                       # Multi-stage build
├── k8s/
│   ├── deployment.yaml              # Go app — 2 replicas, NodePort 30080
│   ├── service.yaml
│   └── servicemonitor.yaml          # Prometheus scrape config
├── monitoring/
│   ├── fluent-bit-config.yaml       # Fluent Bit DaemonSet
│   ├── cwagent-config.yaml          # CloudWatch Agent ConfigMap
│   └── cloudwatch-namespace.yaml
└── jenkins/
    └── Jenkinsfile                  # 6-stage CI/CD pipeline
```

---

## Go HTTP Service

Three endpoints:

```
GET /        → plain text confirmation
GET /health  → JSON: { status, service, uptime }
GET /metrics → Prometheus format:
               app_requests_total
               app_uptime_seconds
               app_info{version, service}
```

Unit tests:
```bash
go test ./... -v
# TestHealthHandler --- PASS
# TestMetricsHandler --- PASS
```

---

## Jenkins CI/CD Pipeline

6 stages triggered on every push to `dev` branch:

```
Checkout SCM → Checkout → Test → Build Binary → Build Docker Image → Push to Docker Hub → Cleanup
```

```groovy
// Jenkinsfile highlights
stage('Test') {
    sh 'go test ./... -v'
}
stage('Build Docker Image') {
    sh "docker build -t zartu/cloudwatch-k8s-demo:v${BUILD_NUMBER} ."
}
stage('Push to Docker Hub') {
    sh "docker push zartu/cloudwatch-k8s-demo:v${BUILD_NUMBER}"
    sh "docker push zartu/cloudwatch-k8s-demo:latest"
}
```

Jenkins runs as WAR file (not systemd):
```bash
java -jar ~/jenkins.war --httpPort=8080
```

---

## Kubernetes Cluster

Self-managed kubeadm cluster — **not EKS**:

```bash
# Control plane init
kubeadm init --pod-network-cidr=192.168.0.0/16

# Worker join
kubeadm join <CP_IP>:6443 --token <token> --discovery-token-ca-cert-hash <hash>

# Calico CNI
kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/calico.yaml
```

Go app deployment:
```bash
kubectl apply -f k8s/deployment.yaml   # 2 replicas, NodePort 30080
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/servicemonitor.yaml
```

---

## CloudWatch Container Insights

### Why it's hard on self-managed K8s

On EKS, Container Insights is a checkbox. On kubeadm, the **kubelet serving certificate** only has the hostname in SANs — not the IP address. The CloudWatch Agent connects to kubelet over HTTPS, and TLS verification fails silently.

### Fix applied

```bash
# On worker node — enable TLS bootstrapping
echo "serverTLSBootstrap: true" >> /var/lib/kubelet/config.yaml
sudo rm /var/lib/kubelet/pki/kubelet.crt /var/lib/kubelet/pki/kubelet.key
sudo systemctl restart kubelet

# On control plane — approve the new CSR
kubectl get csr
kubectl certificate approve <csr-name>
```

### Deploy CloudWatch Agent

```bash
# Namespace
kubectl apply -f monitoring/cloudwatch-namespace.yaml

# ConfigMap
kubectl apply -f monitoring/cwagent-config.yaml

# RBAC + DaemonSet
kubectl apply -f https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/main/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/cwagent/cwagent-serviceaccount.yaml
kubectl apply -f https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/main/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/cwagent/cwagent-daemonset.yaml
```

### CloudWatch Agent Config

```json
{
  "agent": {
    "region": "ap-south-1"
  },
  "logs": {
    "metrics_collected": {
      "kubernetes": {
        "cluster_name": "cloudwatch-k8s-demo",
        "metrics_collection_interval": 60
      }
    },
    "force_flush_interval": 5
  }
}
```

Metrics appear in CloudWatch under namespace `ContainerInsights`:
- `pod_cpu_utilization`
- `pod_memory_utilization`
- `pod_cpu_utilization_over_pod_limit`
- `node_cpu_utilization`
- `namespace_number_of_running_pods`

---

## Fluent Bit Log Shipping

```bash
kubectl apply -f monitoring/fluent-bit-config.yaml
```

Log group created: `/cloudwatch-k8s-demo/containers`

Every `fmt.Println` from every pod is permanently stored in CloudWatch — even after pod restarts.

---

## Prometheus + Grafana

```bash
# Install via Helm
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace

# Access
kubectl port-forward svc/kube-prometheus-stack-grafana 32000:80 -n monitoring
kubectl port-forward svc/kube-prometheus-stack-prometheus 32001:9090 -n monitoring
```

Grafana dashboard **"Go App Metrics"** — 3 panels:
- Total Requests (`app_requests_total`)
- Uptime (`app_uptime_seconds`)
- Requests per second (`rate(app_requests_total[5m])`)

---

## CloudWatch Alarm

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name "go-app-pod-cpu-high" \
  --metric-name pod_cpu_utilization_over_pod_limit \
  --namespace ContainerInsights \
  --dimensions Name=ClusterName,Value=cloudwatch-k8s-demo \
  --statistic Average \
  --period 300 \
  --threshold 50 \
  --comparison-operator GreaterThanThreshold \
  --evaluation-periods 1 \
  --alarm-actions arn:aws:sns:ap-south-1:866454486611:go-app-alerts \
  --region ap-south-1
```

When triggered → SNS → email with full alarm details, metric value, and timestamp.

---

## Setup Guide

### Prerequisites
- AWS account with EC2 access
- WSL 2 (Ubuntu 22.04) on Windows or Linux
- Go 1.22+, Docker, kubectl, AWS CLI, Helm installed

### Step 1 — Clone and test locally
```bash
git clone https://github.com/Zartex-the-art/cloudwatch-k8s-demo.git
cd cloudwatch-k8s-demo
go test ./... -v
go run main.go
curl localhost:8080/health
```

### Step 2 — Build and push Docker image
```bash
docker build -t zartu/cloudwatch-k8s-demo:v1 .
docker push zartu/cloudwatch-k8s-demo:v1
```

### Step 3 — Provision EC2 and set up kubeadm cluster
Launch 2x EC2 instances (c7i-flex.large or t3.medium), open ports: 6443, 2379-2380, 10250-10252, 30000-32767, 179 (Calico BGP).

```bash
# On both nodes
sudo apt-get install -y kubeadm kubelet kubectl
sudo systemctl enable kubelet

# On control plane only
sudo kubeadm init --pod-network-cidr=192.168.0.0/16 \
  --apiserver-cert-extra-sans=<PUBLIC_IP>
```

### Step 4 — Deploy application
```bash
kubectl apply -f k8s/
kubectl get pods --all-namespaces
```

### Step 5 — Deploy monitoring stack
```bash
# Fluent Bit
kubectl apply -f monitoring/fluent-bit-config.yaml

# CloudWatch Agent
kubectl apply -f monitoring/cwagent-config.yaml

# Prometheus + Grafana
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
```

### Step 6 — Configure Jenkins
```bash
java -jar ~/jenkins.war --httpPort=8080
# Add GitHub webhook, configure dockerhub-creds credential
# Pipeline picks up jenkins/Jenkinsfile automatically
```

---

## Demo Outputs

| Output | Description |
|--------|-------------|
| `GET /health` | `{"service":"cloudwatch-k8s-demo","status":"ok","uptime":"Xh"}` |
| `GET /metrics` | `app_requests_total`, `app_uptime_seconds`, `app_info` |
| `kubectl get nodes` | 2 nodes Ready — control-plane + worker |
| `kubectl get pods -A` | 23 pods Running across all namespaces |
| Jenkins Build #2 | All 6 stages green, completed in 1m 3s |
| Docker Hub | `zartu/cloudwatch-k8s-demo:v1`, `v2`, `latest` |
| CloudWatch Logs | `/cloudwatch-k8s-demo/containers` — live pod logs |
| Container Insights | CPU 22.9%, Memory 54%, pod graphs — `cloudwatch-k8s-demo` cluster |
| CloudWatch Alarm | `go-app-pod-cpu-high` — fired and emailed during CPU stress test |
| Grafana | Go App Metrics dashboard — 3 panels live |
| Prometheus | `serviceMonitor/monitoring/go-app/0` — 2/2 UP |

---

## Key Challenges Solved

### 1. Kubelet TLS Certificate SAN Bug
CloudWatch Agent couldn't connect to kubelet — cert missing IP in SANs. Fixed via `serverTLSBootstrap: true` and CSR approval. This is a known production issue that trips up senior engineers.

### 2. Jenkins GPG Key Failure
Jenkins apt repository GPG key expired. Switched to WAR file installation — more reliable and portable.

### 3. Docker Hub Authentication
Password auth deprecated by Docker Hub. Switched to Personal Access Token stored as Jenkins credential.

### 4. kubectl Remote TLS Mismatch
API server cert missing public IP SAN — kubectl from WSL couldn't connect. Fixed by regenerating cert with `--apiserver-cert-extra-sans` on every session (IPs change on EC2 restart).

### 5. CloudWatch Agent Config Format
Newer CloudWatch Agent (v0.124, OTEL-based) requires explicit `region` field in config. EC2 cluster tag `kubernetes.io/cluster/cloudwatch-k8s-demo=owned` also required for cluster name detection.

---

## Impact

| Metric | Before | After |
|--------|--------|-------|
| Mean Time To Detection | ~30 minutes | < 5 minutes |
| Log availability after pod crash | Lost | Permanent (CloudWatch) |
| CPU/Memory visibility | None | Real-time (Container Insights) |
| Deployment process | Manual | Fully automated (Jenkins) |
| Incident notification | Manual check | Automated email (SNS) |

---

## What Makes This Unique

1. **Self-managed kubeadm, not EKS** — we understand what managed services abstract away
2. **Dual observability** — CloudWatch for ops/alerting + Prometheus+Grafana for developers
3. **Real production bug solved** — kubelet TLS SAN issue that breaks Container Insights on kubeadm
4. **End-to-end alerting demonstrated live** — CPU stress test → alarm → email in under 5 minutes
5. **Full CI/CD** — zero manual steps from git push to running pod

---

## Team

| Member | Role |
|--------|------|
| Abhishek | DevOps Lead — K8s cluster, CloudWatch Agent, CI/CD pipeline |
| HariCharan | Infrastructure — EC2 setup, Docker, Fluent Bit, alarm configuration |
| Hrushikesh | Monitoring — Prometheus, Grafana, dashboards, testing |

**Domain:** zartex.tech  
**Target:** FAANG & high-growth startups  
**Stack:** Go · Docker · Jenkins · Kubernetes · AWS · Prometheus · Grafana

---

## Repository Structure

```
main branch   → production-ready code
dev branch    → triggers Jenkins CI/CD pipeline on every push
```

---

<div align="center">

**Built from scratch. No managed services. No shortcuts.**

*CloudWatch Container Insights · kubeadm · Self-Managed K8s · AWS EC2*

</div>
