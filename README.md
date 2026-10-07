<div align="center">
  <h1>EvoNHI Collector</h1>
  <p><b>Secure Runtime Telemetry and State Ingestion Agent</b></p>
</div>

<br/>

The **EvoNHI Collector** is a read-only, lightweight agent designed to securely map Kubernetes RBAC state and Non-Human Identity (NHI) configurations. It extracts workload context, anonymizes sensitive metadata, and transmits a payload to the EvoNHI Control Plane for evolutionary attack path analysis.

## Core Principles
* **Read-Only Execution:** Requires minimal RBAC permissions (`get`, `list` on core and rbac resources). It cannot mutate cluster state.
* **Stateless:** Does not store historical data on the cluster.
* **Authenticated transport:** Payloads are signed with HMAC-SHA256 over `timestamp || body` (replay window enforced server-side) and must be sent over TLS. The payload itself is not encrypted at the application layer.

## What it sends (schema 1.1)

Contract: [`evonhi-core/contracts/cluster-snapshot.v1.schema.json`](https://github.com/gnazinu/evonhi-core/blob/main/contracts/cluster-snapshot.v1.schema.json),
frozen by `internal/k8s/testdata/snapshot.golden.json` and validated against the published schema in CI.

* RBAC: service accounts (with `automountServiceAccountToken`), roles, cluster roles and their bindings.
* Workloads: service account, readable secrets (volumes, projected volumes, `env`, `envFrom` — never
  `imagePullSecrets`) and **external exposure** (LoadBalancer/NodePort Services, external IPs, Ingress backends).
* Secret metadata only. `kubectl.kubernetes.io/last-applied-configuration` is never forwarded and Secret
  annotations are allow-listed, because they can embed the secret's data.
* `usage: null` until an audit-log source is wired, so the control plane reports telemetry confidence 0.

Required RBAC (`deploy/helm`): `get`/`list` on exactly the resources above, enforced by
`TestClusterRoleCoversEverythingTheAgentLists`.

## Development

### Prerequisites
* Go 1.25+
* A running Kubernetes cluster (e.g., Minikube, Kind)
* Valid `~/.kube/config`

### Running the Agent Locally
Currently, the agent authenticates using your local kubeconfig to test extraction logic.

```bash
go run cmd/agent/main.go

Part of the EvoNHI ecosystem — the payload is consumed by the private platform and translated by the open
[`evonhi-core`](https://github.com/gnazinu/evonhi-core) engine.
