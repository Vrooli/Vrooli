# Environment Management & Secrets

> **Prerequisites**: See [Prerequisites Guide](./getting-started/prerequisites.md) for required tools installation.

This guide covers Vrooli's sophisticated environment management system, which supports multiple secrets sources, automatic environment detection, and seamless integration with HashiCorp Vault.

> **Note**: For practical development environment setup, see [Development Environment](./development-environment.md).

## Overview

Vrooli's environment management system provides:

- 🏗️ **Multi-Environment Support**: Development and production configurations
- 🔐 **Multiple Secrets Sources**: File-based secrets and HashiCorp Vault integration
- 🎯 **Automatic Detection**: Location and environment auto-detection
- 🔄 **Seamless Integration**: Works across all deployment targets (local-services, docker-daemon, k8s-cluster)
- 🛡️ **Security-First**: Encrypted secrets, JWT key management, and access controls

## Environment Structure

### Environment Files

```
.env-dev          # Development environment variables
.env-prod         # Production environment variables
```

### Key Management

```
jwt_priv_dev.pem         # Development JWT private key
jwt_pub_dev.pem          # Development JWT public key
jwt_priv_production.pem  # Production JWT private key
jwt_pub_production.pem   # Production JWT public key
```

## Configuration Sources

### 1. File-Based Secrets (Default)

The traditional approach using `.env` files:

```bash
# .env-dev example
SECRETS_SOURCE=file
DB_USER=vrooli_dev
DB_PASSWORD=dev_password_123
REDIS_PASSWORD=redis_dev_123
API_URL=http://localhost:5329
```

### 2. HashiCorp Vault Integration

Advanced secrets management with centralized control:

```bash
# .env-dev for Vault
SECRETS_SOURCE=vault
VAULT_ADDR=http://127.0.0.1:8200
VAULT_AUTH_METHOD=approle
VAULT_ROLE_ID=your-role-id
VAULT_SECRET_ID=your-secret-id
VAULT_SECRET_PATH=secret/data/vrooli/config/shared-all
```

## Vault Integration Deep Dive

> **Note**: For Vault troubleshooting, see [Troubleshooting Guide](./troubleshooting.md#vault-integration-issues).

### Supported Authentication Methods

#### 1. Token Authentication
```bash
VAULT_AUTH_METHOD=token
VAULT_TOKEN=your-vault-token
```

#### 2. AppRole Authentication
```bash
VAULT_AUTH_METHOD=approle
VAULT_ROLE_ID=your-role-id
VAULT_SECRET_ID=your-secret-id
```

#### 3. Kubernetes Authentication
```bash
VAULT_AUTH_METHOD=kubernetes
VAULT_K8S_ROLE=vrooli-vso-sync-role
K8S_JWT_PATH=/var/run/secrets/kubernetes.io/serviceaccount/token
```

## Environment Detection

### Automatic Location Detection

The system automatically detects whether you're running locally or on a remote server:

```bash
# Manual override (optional)
export LOCATION=local   # or 'remote'

# Or use script arguments
vrooli develop --location remote
```

### Environment Classification

```bash
# Automatic detection based on NODE_ENV and ENVIRONMENT
export ENVIRONMENT=development  # development, production
export NODE_ENV=development     # Overrides environment for Node.js apps
```

## Environment-Specific Workflows

### Development Workflow

> **Note**: For complete development setup instructions, see [Development Environment](./development-environment.md).

```bash
# 1. Set up local environment
cp .env-example .env-dev

# 2. Start with file-based secrets (recommended for development)
echo "SECRETS_SOURCE=file" >> .env-dev

# 3. Or use local Vault for testing (see current Vault resource setup)
# Configure Vault as needed for your setup

# 4. Start development environment
vrooli develop --target local-services
```

### Production Deployment with Vault

> **Note**: For server deployment details, see [Server Deployment](./server-deployment.md).

```bash
# 1. Set up production Vault (external service)
# Configure policies, AppRole, and authentication

# 2. Create production environment file with Vault config
cat > .env-prod << EOF
SECRETS_SOURCE=vault
VAULT_ADDR=https://vault.example.com
VAULT_AUTH_METHOD=approle
VAULT_ROLE_ID=prod-role-id
VAULT_SECRET_ID=prod-secret-id
EOF

# 3. Deploy with Vault integration
./scripts/manage.sh deploy --source docker --environment production
```

### Kubernetes Deployment with Vault

> **Note**: For Kubernetes details, see [Kubernetes Deployment](./kubernetes.md).

```bash
# 1. Vault Secrets Operator (VSO) is automatically set up
# 2. Secrets are synchronized from Vault to Kubernetes secrets
# 3. Applications consume secrets as environment variables

# Deploy with Kubernetes + Vault integration
./scripts/manage.sh deploy --source k8s --environment production
```

## Troubleshooting

> **Complete Troubleshooting**: For comprehensive environment and secrets troubleshooting, see [Troubleshooting Guide](./troubleshooting.md#environment--secrets-issues).

### Common Issues

#### 1. Vault Connection Failed
```bash
# Check Vault status
vault status

# Verify network connectivity
curl -k $VAULT_ADDR/v1/sys/health

# Check authentication
vault auth list
```

#### 2. Secrets Not Loading
```bash
# Verify SECRETS_SOURCE setting
echo $SECRETS_SOURCE

# Check environment file exists
ls -la .env-*

# Test Vault authentication
vault login -method=approle role_id=$VAULT_ROLE_ID secret_id=$VAULT_SECRET_ID
```

#### 3. JWT Keys Missing
```bash
# Check key files exist
ls -la jwt_*.pem

# Regenerate if missing
./scripts/manage.sh setup

# Verify keys are loaded
echo "JWT_PRIV length: ${#JWT_PRIV}"
```

### Debug Commands

```bash
# Environment debugging
env | grep -E "(VAULT_|DB_|REDIS_|JWT_|SECRETS_|ENVIRONMENT|NODE_ENV)"

# Vault debugging
vault status
vault token lookup
vault kv list secret/

# Connection testing
bash scripts/helpers/utils/domainCheck.sh
```

## Migration Guides

### From File-Based to Vault

1. **Set up Vault server** (production) or use local dev Vault
2. **Create appropriate policies** for your services
3. **Migrate secrets** from `.env` files to Vault paths
4. **Update environment configuration** to use Vault
5. **Test authentication** and secret retrieval
6. **Deploy with new configuration**

### Between Environments

```bash
# Copy secrets between environments (be careful!)
vault kv get -format=json secret/data/vrooli/config/shared-all | \
vault kv put secret/data/vrooli-dev/config/shared-all -
```

This comprehensive environment management system provides the flexibility to use simple file-based secrets for development while supporting enterprise-grade Vault integration for production deployments. 