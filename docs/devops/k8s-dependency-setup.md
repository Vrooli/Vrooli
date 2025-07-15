# Kubernetes Dependencies Setup Guide

This guide provides comprehensive instructions for setting up all external dependencies required for Vrooli deployment to Kubernetes.

## Quick Dependency Check

Before starting, run the enhanced dependency checker to identify what needs to be set up:

```bash
# Check all dependencies for your environment
bash scripts/helpers/deploy/k8s-dependencies-check.sh --environment dev
bash scripts/helpers/deploy/k8s-dependencies-check.sh --environment prod
```

This script will provide specific setup instructions for any missing dependencies.

## Dependencies Overview

### Required Components

1. **Kubernetes Operators** (PGO, Redis Operator, VSO)
2. **Vault** (for secrets management)
3. **Container Registry Access** (Docker Hub or alternative)
4. **Storage Classes** (for persistent volumes)
5. **Networking** (ingress controllers for production)

## Automated Setup (Development)

For development environments, use the automated setup:

```bash
# Enable Vault-based secrets management
export SECRETS_SOURCE=vault

# Full automated setup including all operators and Vault
bash scripts/main/develop.sh --target k8s-cluster
```

This automatically installs and configures:
- kubectl, Helm, Minikube
- All three required operators
- Development Vault instance with policies
- Vault Kubernetes authentication

## Manual Setup Instructions

### 1. Kubernetes Operators

#### Option A: Automated Installation
```bash
# Install all operators automatically
bash scripts/helpers/deploy/k8s-prerequisites.sh --yes

# Check installation status
bash scripts/helpers/deploy/k8s-prerequisites.sh --check-only
```

#### Option B: Manual Installation

**CrunchyData PostgreSQL Operator v5.8.2:**
```bash
helm repo add crunchydata https://charts.crunchydata.com
helm repo update crunchydata
helm install pgo crunchydata/pgo \
  --version 5.8.2 \
  --namespace postgres-operator \
  --create-namespace \
  --wait --timeout 10m

# Verify installation
kubectl get pods -n postgres-operator
kubectl get crd postgresclusters.postgres-operator.crunchydata.com
```

**Spotahome Redis Operator v1.2.4:**
```bash
helm repo add redis-operator https://spotahome.github.io/redis-operator
helm repo update redis-operator
helm install spotahome-redis-operator redis-operator/redis-operator \
  --version 1.2.4 \
  --namespace redis-operator \
  --create-namespace \
  --wait --timeout 10m

# Verify installation
kubectl get pods -n redis-operator
kubectl get crd redisfailovers.databases.spotahome.com
```

**Vault Secrets Operator (latest):**
```bash
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update hashicorp
helm install vault-secrets-operator hashicorp/vault-secrets-operator \
  --namespace vault-secrets-operator-system \
  --create-namespace \
  --wait --timeout 5m

# Verify installation
kubectl get pods -n vault-secrets-operator-system
kubectl get crd vaultsecrets.secrets.hashicorp.com
```

### 2. Vault Setup

#### Development Vault (In-Cluster)

**Automated:**
```bash
export SECRETS_SOURCE=vault
bash scripts/helpers/setup/target/k8s_cluster.sh
```

**Manual:**
```bash
# Add HashiCorp Helm repository
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update hashicorp

# Install Vault with development configuration
helm install vault hashicorp/vault \
  --namespace vault \
  --create-namespace \
  -f k8s/dev-support/vault-values.yaml \
  --wait --timeout 10m

# Configure Vault for Kubernetes authentication
kubectl exec -n vault vault-0 -- vault auth enable kubernetes

# Apply Vault policies
for policy in k8s/dev-support/vault-policies/*.hcl; do
  kubectl exec -n vault vault-0 -- vault policy write \
    $(basename $policy .hcl) - < $policy
done

# Create Vault role for VSO
kubectl exec -n vault vault-0 -- vault write auth/kubernetes/role/vrooli-vso-sync-role \
  bound_service_account_names=default \
  bound_service_account_namespaces='*' \
  policies=vrooli-config-shared-all-read,vrooli-secrets-shared-server-jobs-read,vrooli-secrets-postgres-read,vrooli-secrets-redis-read,vrooli-secrets-dockerhub-read \
  ttl=24h

# Enable KV secrets engine
kubectl exec -n vault vault-0 -- vault secrets enable -path=secret kv-v2
```

#### Production Vault (External)

**Requirements:**
- External Vault instance (managed service or self-hosted)
- Network accessibility from Kubernetes cluster
- Admin access to configure policies and authentication

**Setup Steps:**

1. **Configure Environment Variables:**
```bash
export VAULT_ADDR=https://your-vault.example.com:8200
export VAULT_TOKEN=your-admin-token
```

2. **Enable Kubernetes Authentication:**
```bash
# Get Kubernetes cluster information
K8S_HOST=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.server}')
K8S_CA_CERT=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}' | base64 -d)

# Enable and configure Kubernetes auth
vault auth enable kubernetes
vault write auth/kubernetes/config \
  kubernetes_host="$K8S_HOST" \
  kubernetes_ca_cert="$K8S_CA_CERT"
```

3. **Apply Vault Policies:**
```bash
for policy in k8s/dev-support/vault-policies/*.hcl; do
  vault policy write $(basename $policy .hcl) $policy
done
```

4. **Create VSO Role:**
```bash
vault write auth/kubernetes/role/vrooli-vso-sync-role \
  bound_service_account_names=default \
  bound_service_account_namespaces=production \
  policies=vrooli-config-shared-all-read,vrooli-secrets-shared-server-jobs-read,vrooli-secrets-postgres-read,vrooli-secrets-redis-read,vrooli-secrets-dockerhub-read \
  ttl=24h
```

5. **Enable KV Secrets Engine:**
```bash
vault secrets enable -path=secret kv-v2
```

6. **Populate Required Secrets:**
```bash
# Shared configuration (non-sensitive)
vault kv put secret/vrooli/config/shared-all \
  VITE_GOOGLE_TRACKING_ID=your-ga-id \
  VITE_STRIPE_PUBLISHABLE_KEY=pk_live_your_key

# Sensitive server/jobs secrets
vault kv put secret/vrooli/secrets/shared-server-jobs \
  OPENAI_API_KEY=your-openai-key \
  ANTHROPIC_API_KEY=your-anthropic-key \
  STRIPE_SECRET_KEY=sk_live_your_key

# Database credentials
vault kv put secret/vrooli/secrets/postgres \
  DB_NAME=vrooli_prod \
  DB_USER=vrooli_user \
  DB_PASSWORD=secure-password

# Redis credentials
vault kv put secret/vrooli/secrets/redis \
  REDIS_PASSWORD=secure-redis-password

# Docker Hub credentials
vault kv put secret/vrooli/dockerhub/pull-credentials \
  username=your-dockerhub-username \
  password=your-dockerhub-password
```

### 3. Container Registry Setup

#### Docker Hub (Default)

**Option A: Manual Secret Creation:**
```bash
kubectl create secret docker-registry vrooli-dockerhub-pull-secret \
  --docker-server=docker.io \
  --docker-username=your-username \
  --docker-password=your-password \
  --docker-email=your-email@example.com \
  --namespace your-deployment-namespace
```

**Option B: VSO Management (Recommended):**
The secret is automatically created by VSO when Vault contains the credentials (see Vault setup above).

#### Alternative Registry

Update `k8s/chart/values-prod.yaml`:
```yaml
image:
  registry: "your-registry.com/your-namespace"

# Update image pull secret if needed
imagePullSecrets:
  - name: "your-custom-pull-secret"
```

### 4. Storage Configuration

#### Check Available Storage Classes:
```bash
kubectl get storageclass
```

#### For Local Development (Minikube/Kind):
```bash
# Install local-path provisioner
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.24/deploy/local-path-storage.yaml

# Set as default if needed
kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

#### For Cloud Providers:
- **AWS**: Use `gp2` or `gp3` storage classes
- **GCP**: Use `standard-rwo` or `ssd` storage classes  
- **Azure**: Use `default` or `managed-premium` storage classes

#### Custom Storage Class Configuration:
Update `k8s/chart/values.yaml` if you need specific storage classes:
```yaml
pgoPostgresql:
  instances:
    storage:
      storageClass: "your-fast-ssd-class"

spotahomeRedis:
  redis:
    storage:
      persistentVolumeClaim:
        spec:
          storageClassName: "your-storage-class"
```

### 5. Networking Setup

#### Development:
No ingress controller required. Use port-forwarding:
```bash
kubectl port-forward svc/vrooli-dev-ui 3000:3000 -n dev
```

#### Production Ingress Controller:

**Nginx Ingress Controller:**
```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.replicaCount=2 \
  --set controller.nodeSelector."kubernetes\\.io/os"=linux \
  --set defaultBackend.nodeSelector."kubernetes\\.io/os"=linux
```

**Traefik (Alternative):**
```bash
helm repo add traefik https://helm.traefik.io/traefik
helm repo update traefik
helm install traefik traefik/traefik \
  --namespace traefik-system \
  --create-namespace
```

#### Configure DNS:
Point your domain to the ingress controller's external IP:
```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

Update `k8s/chart/values-prod.yaml`:
```yaml
ingress:
  enabled: true
  hosts:
    - host: your-domain.com
      paths:
        - path: /
          pathType: Prefix
```

## Verification

After setup, verify all dependencies:

```bash
# Comprehensive dependency check
bash scripts/helpers/deploy/k8s-dependencies-check.sh --environment your-env

# Test deployment
bash scripts/main/deploy.sh --source k8s --environment your-env
```

## Troubleshooting

### Common Issues

1. **Storage Class Not Found:**
   - Check available storage classes: `kubectl get storageclass`
   - Set a default storage class or specify in values files

2. **Vault Connection Issues:**
   - Verify Vault accessibility: `curl $VAULT_ADDR/v1/sys/health`
   - Check VSO logs: `kubectl logs -n vault-secrets-operator-system -l app.kubernetes.io/name=vault-secrets-operator`

3. **Image Pull Failures:**
   - Verify secret exists: `kubectl get secret vrooli-dockerhub-pull-secret`
   - Check secret format: `kubectl get secret vrooli-dockerhub-pull-secret -o yaml`

4. **Operator Issues:**
   - Check operator pods: `kubectl get pods -n operator-namespace`
   - Verify CRDs: `kubectl get crd | grep operator-name`

### Useful Commands

```bash
# Check all operator health
kubectl get pods -A | grep -E "(postgres-operator|redis-operator|vault-secrets-operator)"

# Check Vault status
kubectl exec -n vault vault-0 -- vault status

# Test VSO secret synchronization
kubectl get secrets | grep vrooli

# View operator logs
kubectl logs -n postgres-operator -l postgres-operator.crunchydata.com/control-plane=postgres-operator
kubectl logs -n redis-operator -l app.kubernetes.io/name=redis-operator
kubectl logs -n vault-secrets-operator-system -l app.kubernetes.io/name=vault-secrets-operator
```

## Support

For additional help:
1. Run the dependency checker for specific guidance
2. Check operator documentation links in [kubernetes.md](./kubernetes.md)
3. Review logs for specific error messages
4. Consult the troubleshooting section in [kubernetes.md](./kubernetes.md#troubleshooting-kubernetes-operators)