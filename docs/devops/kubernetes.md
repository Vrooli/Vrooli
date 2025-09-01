# Kubernetes Infrastructure Setup

This guide covers setting up Kubernetes infrastructure that can be used by scenario-generated applications for deployment. The focus is on providing the foundation that generated apps can leverage for their own deployment strategies.

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

The following components provide a foundation that scenario-generated applications can use:

#### Required Kubernetes Operators

**PostgreSQL Operator (CrunchyData PGO)**
- Manages PostgreSQL clusters with high availability
- Provides automated backups and disaster recovery
- Required for apps that need relational databases

**Redis Operator (Spotahome)**
- Manages Redis clusters with failover capabilities
- Provides caching and session storage
- Required for apps that need key-value storage

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

## Application Deployment Patterns

### Helm Chart Structure for Generated Apps

When scenario-generated applications include Kubernetes deployment capabilities, they typically follow this structure:

```
my-generated-app/
├── k8s/
│   └── chart/
│       ├── Chart.yaml           # Chart metadata
│       ├── values.yaml          # Default configuration
│       ├── values-dev.yaml      # Development overrides  
│       ├── values-prod.yaml     # Production overrides
│       └── templates/
│           ├── deployment.yaml  # Application deployment
│           ├── service.yaml     # Service definitions
│           ├── ingress.yaml     # External access
│           └── secrets/         # Secret management
└── .vrooli/
    └── service.json             # Deployment configuration
```

### Common Deployment Commands

#### Basic Helm Operations

```bash
# Lint a generated app's chart
helm lint k8s/chart/

# Template a chart for validation (dry run)
helm template my-app k8s/chart/ -f k8s/chart/values.yaml -f k8s/chart/values-dev.yaml

# Install/upgrade an application
helm upgrade --install my-app k8s/chart/ -f k8s/chart/values-prod.yaml --namespace production --create-namespace

# Uninstall an application
helm uninstall my-app --namespace production
```

#### Application Lifecycle Integration

```bash
# Deploy a generated application
cd my-generated-app/
vrooli deploy --environment production

# This delegates to the app's deployment configuration
# which might use Helm, kubectl, or other deployment tools
```

## Resource Management

### Database Resources

**PostgreSQL Clusters**
Generated apps can request PostgreSQL databases:

```yaml
# In the generated app's values.yaml
postgresql:
  enabled: true
  cluster:
    name: "my-app-db"
    instances: 2
    storage: "10Gi"
```

**Redis Clusters**  
Generated apps can request Redis for caching:

```yaml
# In the generated app's values.yaml
redis:
  enabled: true
  cluster:
    name: "my-app-cache"  
    replicas: 1
    storage: "1Gi"
```

### Secrets Management

**Vault Integration**
Generated apps can integrate with Vault for secrets:

```yaml
# In the generated app's templates/
apiVersion: secrets.hashicorp.com/v1beta1
kind: VaultSecret
metadata:
  name: my-app-secrets
spec:
  vaultAuthRef: my-app-auth
  mount: secret
  path: my-app/config
  refreshAfter: 1h
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
kubectl describe vaultauth my-app-auth

# Check VaultConnection status
kubectl get vaultconnection
kubectl describe vaultconnection vault-connection

# Check VaultSecret status
kubectl get vaultsecret
kubectl describe vaultsecret my-app-secrets

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
kubectl get secrets | grep my-app

# Check secret content (base64 encoded)
kubectl get secret my-app-secrets -o yaml

# Force secret refresh (delete VaultSecret to recreate)
kubectl delete vaultsecret my-app-secrets
# Wait for operator to recreate it
kubectl get vaultsecret my-app-secrets
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

## Infrastructure for Different App Types

### Web Applications
- **Ingress Controllers**: For external HTTP/HTTPS access
- **SSL/TLS Management**: Cert-manager for automatic certificate provisioning
- **CDN Integration**: CloudFlare or similar for static asset delivery

### AI/ML Applications  
- **GPU Resources**: Node pools with GPU support for model inference
- **Model Storage**: Persistent volumes for large model files
- **Scaling**: HPA for inference workloads based on queue depth

### SaaS Applications
- **Multi-tenancy**: Namespace isolation or shared infrastructure patterns
- **Monitoring**: Prometheus and Grafana for application metrics
- **Backup Solutions**: Automated backup strategies for data persistence

### Mobile Backends
- **API Gateways**: Kong or similar for API management
- **Rate Limiting**: Request throttling and quota management
- **Push Notifications**: Integration with FCM/APNs services

## Getting Help

For infrastructure issues:

1. **Check Operator Documentation**: Each operator has specific troubleshooting guides
2. **Community Support**: Kubernetes and operator communities provide extensive help
3. **Application-Specific Issues**: Contact the generated app's support mechanisms
4. **Vrooli Infrastructure**: Use `vrooli help` for Vrooli-specific infrastructure questions

The infrastructure setup provides the foundation for any generated application's deployment needs, while allowing each app to define its own specific deployment patterns and requirements.