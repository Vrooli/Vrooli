# Kubernetes Infrastructure Setup

This guide covers setting up Kubernetes infrastructure for Vrooli scenarios and their resource dependencies. The focus is on providing the foundation that scenarios can leverage when running directly in production environments.

## What is Kubernetes?

Kubernetes (K8s) is an open-source container orchestration platform that automates the deployment, scaling, and management of containerized applications. It provides:

- **Container Orchestration**: Automated deployment and management of containers
- **Service Discovery**: Built-in networking and load balancing 
- **Auto-scaling**: Horizontal and vertical scaling based on resource usage
- **Self-healing**: Automatic replacement of failed containers
- **Configuration Management**: Secrets and configuration management

## What is Helm?

Helm is a package manager for Kubernetes that uses "charts" to define, install, and upgrade complex Kubernetes applications. Benefits include:

- **Templating**: Parameterized Kubernetes manifests
- **Versioning**: Track and rollback application deployments
- **Dependency Management**: Handle complex application dependencies
- **Reusability**: Share and reuse deployment configurations

## Infrastructure Components

### Core Infrastructure Setup

The following components provide a foundation that Vrooli scenarios can use when deployed to production:

#### Required Kubernetes Operators

**PostgreSQL Operator (CrunchyData PGO)**
- Manages PostgreSQL clusters with high availability
- Provides automated backups and disaster recovery
- Required for scenarios that need relational databases

**Redis Operator (Spotahome)**
- Manages Redis clusters with failover capabilities
- Provides caching and session storage
- Required for scenarios that need key-value storage

**Vault Secrets Operator (VSO)**
- Integrates HashiCorp Vault with Kubernetes
- Manages application secrets securely
- Required for production deployments with sensitive data

#### Infrastructure Setup Commands

```bash
# Setup development Kubernetes environment
vrooli develop --target k8s-cluster

# This automatically:
# - Sets up Minikube (if not present)
# - Installs required operators
# - Configures basic networking
# - Sets up development databases
```

#### Manual Operator Installation

If you need to install operators manually:

```bash
# Install CrunchyData PostgreSQL Operator v5.8.2
kubectl apply --server-side -k https://github.com/CrunchyData/postgres-operator/tree/v5.8.2/kustomize/install/namespace
kubectl apply --server-side -k https://github.com/CrunchyData/postgres-operator/tree/v5.8.2/kustomize/install/default

# Install Spotahome Redis Operator v1.2.4
kubectl apply -f https://github.com/spotahome/redis-operator/releases/download/v1.2.4/operator.yaml

# Install Vault Secrets Operator (latest stable)
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update
helm install vault-secrets-operator hashicorp/vault-secrets-operator --version 0.8.1

# Verify installations
kubectl get pods -n postgres-operator
kubectl get pods -n redis-operator  
kubectl get pods -n vault-secrets-operator
```

#### Complete Development Environment Setup

```bash
# Set secrets source to use Vault (optional for dev)
export SECRETS_SOURCE=vault

# Complete setup including Minikube, operators, and Vault
vrooli develop --target k8s-cluster

# Alternative: Just the Kubernetes setup portion
./scripts/manage.sh setup --target k8s-cluster
```

## Scenario Deployment Patterns

### Kubernetes Structure for Scenarios

When scenarios are deployed to Kubernetes for production, they leverage Vrooli's shared infrastructure and maintain this structure:

```
scenarios/my-scenario/
├── .vrooli/service.json         # Resource dependencies and configuration
├── test.sh                      # Integration validation
├── README.md                    # Business documentation (optional)
├── initialization/              # Startup data and workflows
│   ├── database/               # Database schemas and seed data
│   ├── workflows/              # N8n/Windmill automation definitions
│   └── configuration/          # Runtime settings
└── deployment/                 # Production orchestration
    ├── startup.sh              # Scenario initialization
    └── monitor.sh              # Health checks and monitoring
```

### Common Deployment Commands

#### Scenario Deployment

```bash
# Deploy a scenario to Kubernetes production
vrooli scenario run research-assistant --target k8s-cluster

# Test scenario integration in Kubernetes
vrooli scenario test research-assistant --target k8s-cluster

# Package multiple scenarios for customer deployment
./scripts/deployment/package-scenario-deployment.sh \
    "customer-suite" ~/deployments/customer \
    research-assistant invoice-generator

# Deploy packaged scenarios to production cluster
kubectl apply -f ~/deployments/customer/k8s/
```

#### Resource Management in Kubernetes

```bash
# Start shared Vrooli resources in Kubernetes
vrooli resource start-all --target k8s-cluster

# Check resource health in cluster
kubectl get pods -l app.kubernetes.io/part-of=vrooli

# Monitor scenario execution
kubectl logs -l scenario=research-assistant -f
```

## Resource Management

### Database Resources

**PostgreSQL Clusters**
Scenarios can declare PostgreSQL requirements in their service.json:

```json
// In scenarios/my-scenario/.vrooli/service.json
{
  "dependencies": {
    "resources": {
      "postgres": {
        "enabled": true,
        "required": true,
        "cluster": {
          "name": "my-scenario-db",
          "instances": 2,
          "storage": "10Gi"
        }
      }
    }
  }
}
```

**Redis Clusters**  
Scenarios can declare Redis requirements for caching:

```json
// In scenarios/my-scenario/.vrooli/service.json
{
  "dependencies": {
    "resources": {
      "redis": {
        "enabled": true,
        "required": true,
        "cluster": {
          "name": "my-scenario-cache",
          "replicas": 1,
          "storage": "1Gi"
        }
      }
    }
  }
}
```

### Secrets Management

**Vault Integration**
Scenarios can integrate with Vault for secrets through Vrooli's shared Vault resource:

```json
// In scenarios/my-scenario/.vrooli/service.json
{
  "dependencies": {
    "resources": {
      "vault": {
        "enabled": true,
        "required": true,
        "secrets": {
          "mount": "secret",
          "path": "scenarios/my-scenario",
          "refreshAfter": "1h"
        }
      }
    }
  }
}
```

## Troubleshooting Kubernetes Operators

### Operator Installation Issues

#### Check Operator Status

```bash
# Verify all operators are running
kubectl get pods -n postgres-operator
kubectl get pods -n redis-operator-system
kubectl get pods -n vault-secrets-operator-system

# Check operator logs if pods are failing
kubectl logs -n postgres-operator deployment/postgres-operator
kubectl logs -n redis-operator-system deployment/redis-operator
kubectl logs -n vault-secrets-operator-system deployment/vault-secrets-operator
```

#### Verify Custom Resource Definitions (CRDs)

```bash
# Check if CRDs are properly installed
kubectl get crd | grep postgres
kubectl get crd | grep redis  
kubectl get crd | grep vault

# If CRDs are missing, reinstall the operator
kubectl delete -k operator-manifests/
kubectl apply -k operator-manifests/
```

### PostgreSQL Operator (PGO) Troubleshooting

#### Check PostgreSQL Cluster Status

```bash
# List PostgreSQL clusters
kubectl get postgrescluster

# Check cluster status and events
kubectl describe postgrescluster my-cluster
kubectl get events --sort-by='.lastTimestamp'

# Check PGO operator logs
kubectl logs -n postgres-operator deployment/postgres-operator

# Check PostgreSQL pod logs
kubectl logs my-cluster-instance1-abcd-0
```

#### Common PGO Issues

- **Storage Issues**: Verify StorageClass availability and PVC binding
- **Resource Limits**: Check CPU/memory limits in cluster spec
- **Backup Configuration**: Verify pgBackRest settings and storage access

### Redis Operator Troubleshooting

#### Check Redis Failover Status

```bash
# List Redis failover instances
kubectl get redisjailover

# Check failover status
kubectl describe redisjailover my-redis-cluster

# Check Redis pods
kubectl get pods -l app.kubernetes.io/name=redis-failover
kubectl logs redis-my-cluster-0
kubectl logs sentinel-my-cluster-0
```

#### Redis Authentication Issues

- **Password Secrets**: Verify Redis password secret creation
- **Sentinel Configuration**: Check sentinel discovery settings
- **Network Policies**: Ensure pod-to-pod communication is allowed

### Vault Secrets Operator (VSO) Troubleshooting

#### Check VSO Custom Resources

```bash
# Check VaultAuth status  
kubectl get vaultauth
kubectl describe vaultauth my-scenario-auth

# Check VaultConnection status
kubectl get vaultconnection
kubectl describe vaultconnection vault-connection

# Check VaultSecret status
kubectl get vaultsecret
kubectl describe vaultsecret my-scenario-secrets

# Check VSO operator logs
kubectl logs -n vault-secrets-operator-system deployment/vault-secrets-operator
```

#### Common VSO Issues

- **Vault Connectivity**: Verify Vault server is accessible from cluster
- **Authentication**: Check Kubernetes service account and RBAC permissions  
- **Secret Paths**: Verify Vault secret paths exist and are readable
- **Policy Permissions**: Ensure Vault policies allow required operations

#### VSO Secret Sync Debugging

```bash
# Check if secrets are being created
kubectl get secrets | grep my-scenario

# Check secret content (base64 encoded)
kubectl get secret my-scenario-secrets -o yaml

# Force secret refresh (delete VaultSecret to recreate)
kubectl delete vaultsecret my-scenario-secrets
# Wait for operator to recreate it
kubectl get vaultsecret my-scenario-secrets
```

### Quick Diagnosis Script

```bash
# Run comprehensive infrastructure check
./scripts/manage.sh status --target k8s-cluster

# This script validates:
# - CRD installations  
# - Operator pod status
# - Vault connectivity (if applicable)
# - Basic functionality tests
```

## Infrastructure for Different Scenario Types

### AI-Powered Business Scenarios
- **GPU Resources**: Node pools with GPU support for Ollama and ComfyUI
- **Vector Storage**: Persistent volumes for Qdrant collections
- **Model Storage**: Shared volumes for AI model artifacts

### Automation-Heavy Scenarios
- **Workflow Orchestration**: N8n and Windmill shared infrastructure
- **Event Processing**: Node-RED and Redis pub/sub capabilities
- **External Integration**: Network policies for third-party API access

### Customer-Facing Scenarios
- **Ingress Controllers**: For external HTTP/HTTPS access
- **SSL/TLS Management**: Cert-manager for automatic certificate provisioning
- **CDN Integration**: CloudFlare or similar for static asset delivery
- **Multi-tenancy**: Namespace isolation for customer data segregation

### Data-Intensive Scenarios
- **Time-series Infrastructure**: QuestDB for metrics and analytics
- **Object Storage**: MinIO for file management and artifacts
- **Backup Solutions**: Automated backup strategies for scenario data

## Getting Help

For infrastructure issues:

1. **Check Operator Documentation**: Each operator has specific troubleshooting guides
2. **Community Support**: Kubernetes and operator communities provide extensive help
3. **Scenario-Specific Issues**: Check scenario test.sh output and logs
4. **Vrooli Infrastructure**: Use `vrooli help` for Vrooli-specific infrastructure questions

The infrastructure setup provides the foundation for Vrooli scenarios to run in production, with shared resources and isolated execution environments for each scenario deployment.
