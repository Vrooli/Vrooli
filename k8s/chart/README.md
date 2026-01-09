# Vrooli Helm Chart

This Helm chart deploys the Vrooli application stack to Kubernetes, including all microservices, databases, and supporting infrastructure.

## Prerequisites

1. **Kubernetes Cluster** (v1.25+)
2. **Required Operators**:
   - PostgreSQL Operator (CrunchyData PGO or Zalando)
   - Redis Operator (Spotahome)
   - Vault Secrets Operator (VSO)
3. **HashiCorp Vault** (for secrets management)
4. **Helm** (v3.10+)

## Quick Start

### 1. Install Required Operators

```bash
# Automated installation (includes all operators and Vault)
cd /root/Vrooli
./scripts/helpers/setup/target/k8s_cluster.sh --environment development --yes yes
```

### 2. Set Up Vault

Follow the [Legacy Vault Setup Guide](/docs/deployment/history/vault-legacy.md) for background, but prefer the new secrets workflows described in the Deployment Hub:

```bash
# Populate secrets from .env-prod
./scripts/populate-vault-secrets.sh
```

### 3. Deploy Vrooli

#### Development Deployment
```bash
helm install vrooli k8s/chart/ \
  --values k8s/chart/values-dev.yaml
```

#### Production Deployment with AppRole Auth
```bash
# Get AppRole ID
ROLE_ID=$(vault read -field=role_id auth/approle/role/vso-role/role-id)

# Deploy
helm install vrooli k8s/chart/ \
  --values k8s/chart/values-prod.yaml \
  --set vaultAddr="http://vault.vault.svc.cluster.local:8200" \
  --set productionDomain="yourproductiondomain.com" \
  --set vso.appRoleId="$ROLE_ID" \
  --set vso.appRoleSecretRef="vault-approle"
```

#### Production Deployment with Kubernetes Auth
```bash
helm install vrooli k8s/chart/ \
  --values k8s/chart/values-prod.yaml \
  --set vaultAddr="http://vault.vault.svc.cluster.local:8200" \
  --set productionDomain="yourproductiondomain.com" \
  --set vso.k8sAuthMount="kubernetes-prod" \
  --set vso.k8sAuthRole="vrooli-vso-sync-role"
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `productionDomain` | Public-facing domain | `""` |
| `vaultAddr` | Vault instance address | `"http://vault.vault.svc.cluster.local:8200"` |
| `tlsSecretName` | TLS certificate secret name | `""` |
| `vso.enabled` | Enable Vault Secrets Operator | `true` |
| `vso.appRoleId` | AppRole ID for Vault auth | `""` |
| `vso.appRoleSecretRef` | K8s secret with AppRole Secret ID | `"vault-approle"` |
| `pgoPostgresql.enabled` | Deploy PostgreSQL cluster | `true` |
| `spotahomeRedis.enabled` | Deploy Redis cluster | `true` |

### Environment-Specific Values

- `values.yaml` - Base configuration
- `values-dev.yaml` - Development overrides
- `values-prod.yaml` - Production overrides

## Services Deployed

### Core Services
- **UI** - React frontend (PWA)
- **Server** - Express.js API server
- **Jobs** - Background job processor

### Optional Services
- **NSFW Detector** - Content moderation service
- **Adminer** - Database management UI (dev only)

### Data Services
- **PostgreSQL** - Primary database (via operator)
- **Redis** - Cache and pub/sub (via operator)

## Secrets Management

All secrets are managed through HashiCorp Vault and synchronized to Kubernetes via VSO:

1. **Shared Config** - Non-sensitive configuration
2. **Server/Jobs Secrets** - API keys, tokens
3. **Database Credentials** - PostgreSQL connection info
4. **Redis Credentials** - Redis authentication
5. **Docker Hub** - Registry pull credentials

## Monitoring

Check deployment status:
```bash
# Overall status
helm status vrooli

# Pod status
kubectl get pods -l app.kubernetes.io/instance=vrooli

# Vault secrets sync status
kubectl get vaultstaticsecret
kubectl get secrets | grep vrooli
```

## Troubleshooting

### Secrets Not Syncing
```bash
# Check VSO logs
kubectl logs -n vault-secrets-operator-system -l app.kubernetes.io/name=vault-secrets-operator

# Check VaultAuth status
kubectl describe vaultauth vrooli-vault-auth

# Check VaultStaticSecret status
kubectl describe vaultstaticsecret
```

### Pod Startup Issues
```bash
# Check pod events
kubectl describe pod <pod-name>

# Check container logs
kubectl logs <pod-name> -c <container-name>
```

## Uninstalling

```bash
# Remove deployment
helm uninstall vrooli

# Clean up PVCs (careful - this deletes data!)
kubectl delete pvc -l app.kubernetes.io/instance=vrooli
```

## Additional Resources

- [Legacy Vault Setup Guide](/docs/deployment/history/vault-legacy.md)
- [Complete Vault Setup Guide](/docs/scratch/vault-setup-complete-guide.md)
- [Kubernetes Deployment Guide](/docs/devops/kubernetes.md)
- [Environment Management](/docs/devops/environment-management.md)
