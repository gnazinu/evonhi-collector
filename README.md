<div align="center">
  <h1>EvoNHI Collector</h1>
  <p><b>Secure Runtime Telemetry and State Ingestion Agent</b></p>
</div>

<br/>

The **EvoNHI Collector** is a read-only, lightweight agent designed to securely map Kubernetes RBAC state and Non-Human Identity (NHI) configurations. It extracts workload context, anonymizes sensitive metadata, and transmits a payload to the EvoNHI Control Plane for evolutionary attack path analysis.

## Core Principles
* **Read-Only Execution:** Requires minimal RBAC permissions (`get`, `list` on core and rbac resources). It cannot mutate cluster state.
* **Stateless:** Does not store historical data on the cluster.
* **Cryptographically Secure:** Payloads are signed and encrypted before leaving the cluster boundary.

## Development

### Prerequisites
* Go 1.21+
* A running Kubernetes cluster (e.g., Minikube, Kind)
* Valid `~/.kube/config`

### Running the Agent Locally
Currently, the agent authenticates using your local kubeconfig to test extraction logic.

```bash
go run cmd/agent/main.go