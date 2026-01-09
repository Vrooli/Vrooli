# Legacy: Production Vault Setup for Kubernetes Deployment

> **Status:** Historical reference from the previous Kubernetes workflow. Use for inspiration when implementing Tier 4/5 secret automation, but follow the new [Secrets Guide](../guides/secrets-management.md) going forward.

This guide covers setting up HashiCorp Vault for production Kubernetes deployments with the Vault Secrets Operator (VSO).

## Prerequisites

- HashiCorp Vault instance accessible from your Kubernetes cluster
- Vault admin access or ability to configure policies and authentication
- Kubernetes cluster with VSO installed
- `vault` CLI tool installed and configured

## Quick Start Options

### Option 1: In-Cluster Vault (Development/Testing)
```bash
# Deploy Vault in your cluster using the built-in automation
cd /root/Vrooli
./scripts/helpers/setup/target/k8s_cluster.sh --environment development --yes yes
```

### Option 2: External Vault (Production Recommended)
Configure your external Vault instance following this guide.

## 1. Environment-Specific Vault Paths

Vrooli uses environment-specific paths in Vault to isolate secrets:

```
Development:  secret/data/vrooli/dev/...
Staging:      secret/data/vrooli/staging/...  
Production:   secret/data/vrooli-prod/...
```

## 2. Required Vault Policies

Create the following policies for production deployment:

### 2.1 Shared Configuration Policy
```bash
vault policy write vrooli-prod-config-shared-all-read - <<EOF
path "secret/data/vrooli-prod/config/shared-all" {
  capabilities = ["read"]
}
EOF
```

### 2.2 Server/Jobs Secrets Policy
```bash
vault policy write vrooli-prod-secrets-shared-server-jobs-read - <<EOF
path "secret/data/vrooli-prod/secrets/shared-server-jobs" {
  capabilities = ["read"]
}
EOF
```

### 2.3 PostgreSQL Secrets Policy
```bash
vault policy write vrooli-prod-secrets-postgres-read - <<EOF
path "secret/data/vrooli-prod/secrets/postgres" {
  capabilities = ["read"]
}
EOF
```

### 2.4 Redis Secrets Policy
```bash
vault policy write vrooli-prod-secrets-redis-read - <<EOF
path "secret/data/vrooli-prod/secrets/redis" {
  capabilities = ["read"]
}
EOF
```

### 2.5 Docker Hub Secrets Policy
```bash
vault policy write vrooli-prod-secrets-dockerhub-read - <<EOF
path "secret/data/vrooli-prod/dockerhub/*" {
  capabilities = ["read"]
}
EOF
```

## 3. Authentication Methods

You can use either Kubernetes authentication (recommended for production) or AppRole authentication.

### Option A: Kubernetes Authentication (Recommended)

#### 3.1 Enable Kubernetes Auth Method
```bash
vault auth enable -path=kubernetes-prod kubernetes
```

### 3.2 Configure Kubernetes Auth
```bash
# Get your Kubernetes cluster's details
K8S_HOST=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.server}')
K8S_CA_CERT=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}' | base64 -d)
TOKEN_REVIEW_JWT=$(kubectl get secret -n kube-system $(kubectl get serviceaccount -n kube-system default -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 -d)

# Configure the auth method
vault write auth/kubernetes-prod/config \
    token_reviewer_jwt="$TOKEN_REVIEW_JWT" \
    kubernetes_host="$K8S_HOST" \
    kubernetes_ca_cert="$K8S_CA_CERT"
```

### 3.3 Create VSO Role
```bash
vault write auth/kubernetes-prod/role/vrooli-vso-sync-role \
    bound_service_account_names=default \
    bound_service_account_namespaces=production \
    policies=vrooli-prod-config-shared-all-read,vrooli-prod-secrets-shared-server-jobs-read,vrooli-prod-secrets-postgres-read,vrooli-prod-secrets-redis-read,vrooli-prod-secrets-dockerhub-read \
    ttl=24h
```

### Option B: AppRole Authentication

#### 3.4 Enable AppRole Auth Method
```bash
vault auth enable approle
```

#### 3.5 Create AppRole for VSO
```bash
vault write auth/approle/role/vso-role \
    policies=vrooli-prod-config-shared-all-read,vrooli-prod-secrets-shared-server-jobs-read,vrooli-prod-secrets-postgres-read,vrooli-prod-secrets-redis-read,vrooli-prod-secrets-dockerhub-read \
    token_ttl=1h \
    token_max_ttl=4h
```

#### 3.6 Get AppRole Credentials
```bash
# Get Role ID (safe to store in values files)
ROLE_ID=$(vault read -field=role_id auth/approle/role/vso-role/role-id)
echo "Role ID: $ROLE_ID"

# Generate Secret ID (sensitive - must be stored in Kubernetes secret)
SECRET_ID=$(vault write -field=secret_id -f auth/approle/role/vso-role/secret-id)
echo "Secret ID: $SECRET_ID"

# Create Kubernetes secret for Secret ID
kubectl create secret generic vault-approle \
    --from-literal=id="$SECRET_ID" \
    --namespace=default
```

**⚠️ Security Notes:**
- **Role ID**: Can be stored in values files or passed as Helm parameter
- **Secret ID**: Must NEVER be stored in values files - only in Kubernetes secrets
- **Rotation**: Secret IDs should be rotated regularly in production
- **Alternative**: Consider using Kubernetes auth for production (no secret management needed)

## 4. Populate Production Secrets

### 4.1 Automated Secret Population
```bash
# Use the provided script to populate all secrets from .env-prod
cd /root/Vrooli
./scripts/populate-vault-secrets.sh
```

### 4.2 Manual Secret Population

#### 4.2.1 Shared Configuration (Non-Sensitive)
```bash
vault kv put secret/vrooli-prod/config/shared-all \
    PROJECT_DIR="/srv/app" \
    SITE_IP="YOUR_PRODUCTION_IP" \
    UI_URL="https://yourproductiondomain.com" \
    API_URL="https://yourproductiondomain.com/api" \
    PORT_DB="5432" \
    PORT_JOBS="4001" \
    PORT_SERVER="5329" \
    PORT_REDIS="6379" \
    PORT_UI="3000" \
    CREATE_MOCK_DATA="false" \
    DB_PULL="false" \
    NODE_ENV="production" \
    VITE_GOOGLE_ADSENSE_PUBLISHER_ID="ca-pub-YOUR_REAL_ID" \
    VITE_GOOGLE_TRACKING_ID="G-YOUR_REAL_ID" \
    VITE_STRIPE_PUBLISHABLE_KEY="pk_live_YOUR_REAL_KEY"
```

### 4.2 Database Credentials
```bash
vault kv put secret/vrooli-prod/secrets/postgres \
    DB_NAME="vrooli" \
    DB_USER="vrooli_app_user" \
    DB_PASSWORD="YOUR_SECURE_DB_PASSWORD"
```

### 4.3 Redis Credentials
```bash
vault kv put secret/vrooli-prod/secrets/redis \
    REDIS_PASSWORD="YOUR_SECURE_REDIS_PASSWORD"
```

### 4.4 Server/Jobs Secrets
```bash
vault kv put secret/vrooli-prod/secrets/shared-server-jobs \
    VAPID_PUBLIC_KEY="YOUR_VAPID_PUBLIC_KEY" \
    VAPID_PRIVATE_KEY="YOUR_VAPID_PRIVATE_KEY" \
    SITE_EMAIL_FROM="noreply@yourproductiondomain.com" \
    SITE_EMAIL_USERNAME="your_email_username" \
    SITE_EMAIL_PASSWORD="your_email_password" \
    SITE_EMAIL_ALIAS="Vrooli Production" \
    STRIPE_SECRET_KEY="sk_live_YOUR_REAL_SECRET_KEY" \
    STRIPE_WEBHOOK_SECRET="whsec_YOUR_REAL_WEBHOOK_SECRET" \
    TWILIO_ACCOUNT_SID="YOUR_TWILIO_SID" \
    TWILIO_AUTH_TOKEN="YOUR_TWILIO_TOKEN" \
    TWILIO_PHONE_NUMBER="YOUR_TWILIO_PHONE" \
    TWILIO_DOMAIN_VERIFICATION_CODE="YOUR_TWILIO_VERIFICATION" \
    AWS_ACCESS_KEY_ID="YOUR_AWS_ACCESS_KEY" \
    AWS_SECRET_ACCESS_KEY="YOUR_AWS_SECRET_KEY" \
    AWS_REGION="us-east-1" \
    S3_BUCKET_NAME="your-production-s3-bucket" \
    OPENAI_API_KEY="sk-YOUR_OPENAI_KEY" \
    ANTHROPIC_API_KEY="sk-ant-YOUR_ANTHROPIC_KEY" \
    MISTRAL_API_KEY="YOUR_MISTRAL_KEY" \
    GOOGLE_CLIENT_ID="YOUR_GOOGLE_CLIENT_ID" \
    GOOGLE_CLIENT_SECRET="YOUR_GOOGLE_CLIENT_SECRET" \
    MCP_API_KEY="YOUR_MCP_API_KEY" \
    ADMIN_WALLET="YOUR_ADMIN_WALLET" \
    ADMIN_PASSWORD="YOUR_SECURE_ADMIN_PASSWORD" \
    VALYXA_PASSWORD="YOUR_SECURE_VALYXA_PASSWORD" \
    EXTERNAL_SITE_KEY="YOUR_EXTERNAL_SITE_KEY" \
    SECRET_KEY="YOUR_SECURE_SECRET_KEY_FOR_PRODUCTION"
```

### 4.5 Docker Hub Credentials
```bash
vault kv put secret/vrooli-prod/dockerhub/pull-credentials \
    username="your_dockerhub_username" \
    password="your_dockerhub_token_or_password"
```

## 5. Update Helm Values

Ensure your `k8s/chart/values-prod.yaml` points to the correct Vault paths:

```yaml
vso:
  enabled: true
  vaultAddr: "https://your-production-vault.com"
  k8sAuthMount: "kubernetes-prod"
  k8sAuthRole: "vrooli-vso-sync-role"
  
  secrets:
    sharedConfigAll:
      vaultPath: "secret/data/vrooli-prod/config/shared-all"
    sharedSecretsServerJobs:
      vaultPath: "secret/data/vrooli-prod/secrets/shared-server-jobs"
    postgres:
      vaultPath: "secret/data/vrooli-prod/secrets/postgres"
    redis:
      vaultPath: "secret/data/vrooli-prod/secrets/redis"
    dockerhubPullSecret:
      vaultPath: "secret/data/vrooli-prod/dockerhub/pull-credentials"
```

## 6. Helm Deployment

### 6.1 Using Kubernetes Authentication
```bash
helm install vrooli k8s/chart/ \
    --values k8s/chart/values-prod.yaml \
    --set vaultAddr="http://vault.vault.svc.cluster.local:8200" \
    --set productionDomain="yourproductiondomain.com" \
    --set vso.k8sAuthMount="kubernetes-prod" \
    --set vso.k8sAuthRole="vrooli-vso-sync-role"
```

### 6.2 Using AppRole Authentication
```bash
# First, get your Role ID
ROLE_ID=$(vault read -field=role_id auth/approle/role/vso-role/role-id)

# Deploy with AppRole configuration
helm install vrooli k8s/chart/ \
    --values k8s/chart/values-prod.yaml \
    --set vaultAddr="http://vault.vault.svc.cluster.local:8200" \
    --set productionDomain="yourproductiondomain.com" \
    --set vso.appRoleId="$ROLE_ID" \
    --set vso.appRoleSecretRef="vault-approle" \
    --set vso.appRoleMount="approle"
```

### 6.3 Important Deployment Parameters
- `vaultAddr`: Your Vault instance address (internal or external)
- `productionDomain`: Your production domain name
- `tlsSecretName`: Name of your TLS certificate secret (if using HTTPS)
- `vso.appRoleId`: The Role ID from Vault (AppRole auth only)
- `vso.appRoleSecretRef`: Name of k8s secret containing Secret ID (AppRole auth only)

## 7. Verification

### 7.1 Test Vault Access
```bash
# Test from within cluster
kubectl run vault-test --rm -i --tty --image=alpine/curl -- sh
# Inside the pod:
wget -qO- http://vault.vault.svc.cluster.local:8200/v1/sys/health
```

### 7.2 Verify VSO Secret Sync
```bash
# Check if VSO created the secrets
kubectl get secrets -n production | grep vrooli

# Check secret content (base64 encoded)
kubectl get secret vrooli-config-shared-all -n production -o yaml
```

### 7.3 Test Application Startup
```bash
# Deploy and check logs
kubectl logs -f deployment/vrooli-prod-server -n production
```

## 7. Security Considerations

1. **Least Privilege**: Each policy only grants read access to specific paths
2. **Environment Isolation**: Production secrets are in separate paths
3. **Regular Rotation**: Implement secret rotation for sensitive credentials
4. **Audit Logging**: Enable Vault audit logging for production
5. **Backup Strategy**: Implement Vault backup and disaster recovery

## 8. Troubleshooting

### Common Issues:

1. **VSO Authentication Failed**
   - Verify Kubernetes auth configuration
   - Check service account permissions
   - Ensure correct auth mount path

2. **Secret Sync Failed** 
   - Check Vault paths include `/data/` for KVv2
   - Verify policy permissions
   - Check VSO operator logs

3. **Pod Startup Failed**
   - Verify secret names match deployment expectations
   - Check secret key names in Vault data
   - Review pod logs for specific errors

### Debug Commands:
```bash
# Check VSO resources
kubectl get vaultauth,vaultconnection,vaultsecret -n production

# Check VSO operator logs
kubectl logs -n vault-secrets-operator-system -l app.kubernetes.io/name=vault-secrets-operator

# Test secret access manually
kubectl exec -it deployment/vrooli-prod-server -n production -- env | grep -E "(DB_|REDIS_|API_)"
```

## 9. Production Checklist

- [ ] Vault policies created and tested
- [ ] Kubernetes authentication configured  
- [ ] All production secrets populated in Vault
- [ ] VSO role created with correct permissions
- [ ] Helm values updated with production Vault paths
- [ ] Secret sync verified in target namespace
- [ ] Application pods starting successfully
- [ ] Health checks passing
- [ ] Monitoring and alerting configured
- [ ] Backup and disaster recovery procedures documented
