# HashiCorp Vault - Secret Management Service

A secure, API-driven secret management solution that provides encrypted storage, dynamic secrets, data encryption, and more. This implementation provides both development and production modes with seamless integration into the Vrooli resource ecosystem.

## 🎯 **Use Cases**

### **Secret Management**
- API keys and credentials for external services
- Database connection strings and passwords  
- OAuth tokens and JWT secrets
- SSL/TLS certificates and private keys

### **Client Credential Isolation**
- Multi-tenant secret namespacing
- Client-specific API key management
- Temporary project credentials
- Secure credential handoff workflows

### **Development & Production**
- Environment-specific configuration
- Secure secret migration between environments
- Automated secret rotation
- Audit logging and compliance

### **Integration with Vrooli Resources**
- n8n workflow credentials
- Node-RED flow secrets
- Agent-S2 website authentication
- Automated CI/CD pipeline secrets

## 🚀 **Quick Start**

### **Development Setup**
```bash
# Initialize Vault in development mode (auto-unsealed, in-memory storage)
./cli.sh content init-dev

# Store a secret (positional arguments)
./cli.sh content add "environments/dev/database-url" "postgresql://user:pass@localhost/db"

# Store a secret (named arguments)
./cli.sh content add --path "environments/dev/database-url" --value "postgresql://user:pass@localhost/db"

# Retrieve a secret (positional)
./cli.sh content get "environments/dev/database-url"

# Retrieve a secret (named arguments)
./cli.sh content get --path "environments/dev/database-url"

# List secrets in a path
./cli.sh content list "environments/dev"
```

### **Production Setup**
```bash
# Initialize Vault in production mode (manual unsealing, persistent storage)
./cli.sh content init-prod

# Unseal Vault after initialization
./cli.sh content unseal

# Store production secrets (both syntaxes work)
./cli.sh content add "environments/prod/stripe-key" "sk_live_..."
# or:
./cli.sh content add --path "environments/prod/stripe-key" --value "sk_live_..."
```

## 📋 **Available Actions**

### **Lifecycle Management**
- `install` - Install Vault container and dependencies
- `uninstall` - Remove Vault container (optionally remove data)
- `start` - Start Vault service
- `stop` - Stop Vault service  
- `restart` - Restart Vault service

### **Initialization & Setup**
- `init-dev` - Initialize in development mode (recommended for local work)
- `init-prod` - Initialize in production mode (manual unsealing required)
- `unseal` - Unseal Vault (production mode only)

### **Secret Operations**
- `content add <path> <value>` or `content add --path <path> --value <value>` - Store a secret
- `content get <path>` or `content get --path <path>` - Retrieve a secret  
- `content list <path>` or `content list --path <path>` - List secrets at path
- `content remove <path>` or `content remove --path <path>` - Delete a secret

### **Migration & Backup**
- `content migrate-env <file> <prefix>` or `content migrate-env --env-file <file> --vault-prefix <prefix>` - Migrate .env file to Vault
- `backup [--backup-file <file>]` - Backup Vault data
- `restore --backup-file <file>` - Restore Vault data

### **Monitoring & Diagnostics**
- `status` - Show comprehensive status report
- `logs [--lines N] [--follow]` - Show container logs
- `diagnose` - Run comprehensive diagnostics
- `monitor [--interval N]` - Continuous status monitoring

## 🗂️ **Secret Organization**

### **Recommended Namespace Structure**
```
/vrooli/
├── environments/
│   ├── development/
│   ├── staging/
│   └── production/
├── clients/
│   ├── acme-corp/
│   ├── globex-ltd/
│   └── initech/
├── resources/
│   ├── n8n-api-key
│   ├── agent-s2-credentials
│   └── minio-access-key
└── ephemeral/
    ├── temp-oauth-tokens/
    └── workflow-secrets/
```

## 🔧 **Configuration**

### **Environment Variables**
```bash
# Service configuration
VAULT_PORT=8200                    # Vault API port
VAULT_MODE=dev                     # dev or prod
VAULT_DATA_DIR=~/.vault/data       # Data storage directory
VAULT_CONFIG_DIR=~/.vault/config   # Configuration directory

# Development mode
VAULT_DEV_ROOT_TOKEN_ID=myroot     # Development root token

# Production mode  
VAULT_TLS_DISABLE=1                # Disable TLS (1 for dev, 0 for prod)
VAULT_STORAGE_TYPE=file            # Storage backend type
```

### **Modes**

#### **Development Mode (`--mode dev`)**
- ✅ Auto-initialized and unsealed
- ✅ In-memory storage (no persistence)  
- ✅ Fixed root token for convenience
- ⚠️  **Not secure for production use**

#### **Production Mode (`--mode prod`)**
- 🔐 Manual initialization required
- 🔐 Manual unsealing after restarts
- 💾 Persistent file storage
- 🔒 Secure unseal keys and root token
- 📝 Audit logging enabled

## 🔗 **Integration Examples**

### **n8n Workflow Integration**
```javascript
// Custom n8n node to retrieve Vault secrets
const vaultConfig = await getResourceConfig('vault');
const secret = await fetch(`${vaultConfig.baseUrl}/v1/secret/data/clients/acme/stripe-key`, {
  headers: { 'X-Vault-Token': vaultConfig.token }
});
```

### **Node-RED Flow Integration**
```javascript
// Node-RED vault-secret node
msg.payload = await vaultClient.getSecret('environments/prod/database-url');
```

### **Environment Variable Migration**
```bash
# Migrate existing .env file to Vault (positional)
./cli.sh content migrate-env .env "environments/development"

# Migrate using named arguments
./cli.sh content migrate-env --env-file .env --vault-prefix "environments/development"

# Migrate production environment
./cli.sh content migrate-env --env-file .env.production --vault-prefix "environments/production"
```

### **CI/CD Pipeline Integration**
```bash
# Store deployment secrets
./cli.sh content add --path "ci/docker-registry-token" --value "ghp_xxxx"
./cli.sh content add --path "ci/aws-access-key" --value "AKIAXXXX"

# Retrieve in deployment scripts
DB_URL=$(./cli.sh content get --path "environments/prod/database-url")
```

## 🔒 **Security Best Practices**

### **Development**
- Use development mode for local testing only
- Never use development tokens in shared environments
- Regularly rotate development credentials

### **Production**
- Always use production mode for sensitive data
- Store unseal keys securely and separately
- Enable audit logging for compliance
- Implement secret rotation policies
- Use least-privilege access controls

### **Secret Management**
- Use descriptive, hierarchical paths
- Store related secrets together
- Document secret purposes and owners
- Implement automated rotation where possible

## 🚨 **Important Security Notes**

### **Development Mode Warnings**
- Root token is predictable and logged
- Data is stored in memory (lost on restart)
- No encryption at rest
- Suitable for development only

### **Production Mode Requirements**
- Secure storage of unseal keys (minimum 3 required)
- Secure storage of root token
- Regular backups of Vault data
- Network security and access controls
- Audit log monitoring

## 📊 **Monitoring & Troubleshooting**

### **Health Checks**
```bash
# Quick status check
./cli.sh status

# Comprehensive diagnostics
./cli.sh diagnose

# Continuous monitoring
./cli.sh monitor --interval 30
```

### **Common Issues**

#### **Vault Sealed (Production Mode)**
```bash
# Check seal status
./cli.sh status

# Unseal if needed
./cli.sh content unseal
```

#### **Permission Issues**
```bash
# Fix data directory permissions
sudo chown -R $(whoami):$(whoami) ~/.vault

# Restart with fresh configuration
./cli.sh manage restart
```

#### **Port Conflicts**
```bash
# Check what's using the port
netstat -tlnp | grep 8200

# Use custom port
VAULT_PORT=8201 ./cli.sh manage start
```

### **Log Analysis**
```bash
# Show recent logs
./cli.sh logs --lines 100

# Follow logs in real-time
./cli.sh logs --follow

# Check for specific errors
./cli.sh logs --lines 200 | grep -i error
```

## 🔄 **Backup & Recovery**

### **Backup Operations**
```bash
# Create backup
./cli.sh content backup --backup-file vault-backup-$(date +%Y%m%d).tar.gz

# Automated backup
./cli.sh content backup  # Uses timestamp filename
```

### **Restore Operations**
```bash
# Restore from backup
./cli.sh manage stop
./cli.sh content restore --backup-file vault-backup-20240128.tar.gz
./cli.sh manage start
```

## 🌐 **API Reference**

### **Direct API Usage**
```bash
# Health check
curl ${VAULT_BASE_URL}/v1/sys/health

# Store secret (requires authentication)
curl -X POST ${VAULT_BASE_URL}/v1/secret/data/myapp/config \
  -H "X-Vault-Token: ${VAULT_TOKEN}" \
  -d '{"data": {"api_key": "secret123"}}'

# Retrieve secret
curl -X GET ${VAULT_BASE_URL}/v1/secret/data/myapp/config \
  -H "X-Vault-Token: ${VAULT_TOKEN}"
```

## 📚 **Additional Resources**

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [Vault API Reference](https://www.vaultproject.io/api-docs)
- [Secret Engines](https://www.vaultproject.io/docs/secrets)
- [Authentication Methods](https://www.vaultproject.io/docs/auth)

---

**🔐 Vault provides enterprise-grade secret management capabilities while maintaining ease of use for development workflows. Always follow security best practices appropriate for your environment.**