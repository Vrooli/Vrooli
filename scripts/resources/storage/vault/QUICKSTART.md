# Vault Quick Start Guide

This guide will help you get started with HashiCorp Vault in the Vrooli ecosystem.

## Installation and Setup

### 1. Install Vault (Development Mode)
```bash
cd scripts/resources/storage/vault
./manage.sh --action install
./manage.sh --action init-dev
```

This will:
- Pull the Vault Docker image
- Start Vault in development mode (automatically unsealed)
- Create a root token stored in `/tmp/vault-token`
- Set up the KV v2 secret engine
- Create default namespace paths

### 2. Check Status
```bash
./manage.sh --action status
```

You should see:
- Status: healthy
- Port: 8200
- Mode: dev
- Docker container running

## Basic Usage

### 1. Store a Secret
```bash
# Store a simple key-value pair
./manage.sh --action put-secret --path "environments/dev/api-key" --value "my-secret-api-key"

# Store with custom key name
./manage.sh --action put-secret --path "environments/dev/database" --value "postgres://user:pass@host:5432/db" --key "connection_string"
```

### 2. Retrieve a Secret
```bash
# Get a specific secret
./manage.sh --action get-secret --path "environments/dev/api-key"

# Get as JSON (useful for multiple keys)
./manage.sh --action get-secret --path "environments/dev/database" --format json
```

### 3. List Secrets
```bash
# List all secrets in a path
./manage.sh --action list-secrets --path "environments/"

# List as JSON
./manage.sh --action list-secrets --path "environments/" --format json
```

### 4. Delete a Secret
```bash
./manage.sh --action delete-secret --path "environments/dev/api-key"
```

## Advanced Features

### 1. Bulk Operations
```bash
# Store multiple secrets from JSON
echo '{"api_key": "secret123", "db_password": "pass456"}' > secrets.json
./manage.sh --action bulk-put --json-file secrets.json --base-path "environments/dev"
rm secrets.json

# Export secrets to JSON
./manage.sh --action export-secrets --path "environments/dev" > exported-secrets.json
```

### 2. Environment Migration
```bash
# Migrate .env file to Vault
./manage.sh --action migrate-env --env-file .env --vault-prefix "environments/dev"
```

### 3. Secret Rotation
```bash
# Generate new random value
./manage.sh --action rotate-secret --path "environments/dev/api-key" --type random

# Generate new API key format
./manage.sh --action rotate-secret --path "environments/dev/service-key" --type api-key
```

### 4. Monitoring
```bash
# Show detailed diagnostics
./manage.sh --action diagnose

# Monitor health continuously
./manage.sh --action monitor --interval 5

# Follow container logs
./manage.sh --action logs --follow yes
```

## Integration Examples

### Node.js/TypeScript
```typescript
// Using node-vault client
const vault = require('node-vault')({
  endpoint: 'http://localhost:8200',
  token: process.env.VAULT_TOKEN || fs.readFileSync('/tmp/vault-token', 'utf8').trim()
});

// Write secret
await vault.write('secret/data/myapp/config', {
  data: {
    api_key: 'secret123',
    db_url: 'postgres://localhost/mydb'
  }
});

// Read secret
const result = await vault.read('secret/data/myapp/config');
const secrets = result.data.data;
console.log(secrets.api_key); // 'secret123'
```

### Bash/Shell
```bash
# Set token
export VAULT_TOKEN=$(cat /tmp/vault-token)
export VAULT_ADDR="http://localhost:8200"

# Using curl directly
curl -H "X-Vault-Token: $VAULT_TOKEN" \
     -X POST \
     -d '{"data": {"password": "secret123"}}' \
     $VAULT_ADDR/v1/secret/data/myapp/db

# Read with curl
curl -H "X-Vault-Token: $VAULT_TOKEN" \
     $VAULT_ADDR/v1/secret/data/myapp/db | jq '.data.data'
```

### Python
```python
import hvac

# Initialize client
client = hvac.Client(
    url='http://localhost:8200',
    token=open('/tmp/vault-token').read().strip()
)

# Write secret
client.secrets.kv.v2.create_or_update_secret(
    path='myapp/config',
    secret={'api_key': 'secret123'}
)

# Read secret
response = client.secrets.kv.v2.read_secret_version(path='myapp/config')
secret_value = response['data']['data']['api_key']
```

## Production Setup

For production use:

```bash
# Initialize in production mode
./manage.sh --action init-prod

# This will:
# - Initialize Vault with 5 unseal keys
# - Save keys to /tmp/vault-unseal-keys (SAVE THESE SECURELY!)
# - Save root token to /tmp/vault-token
# - Require manual unsealing after restarts

# Unseal after restart (needs 3 of 5 keys)
./manage.sh --action unseal

# Check seal status
./manage.sh --action status
```

## Namespace Organization

Vault is pre-configured with these namespace paths:
- `environments/dev/` - Development environment secrets
- `environments/staging/` - Staging environment secrets  
- `environments/prod/` - Production environment secrets
- `resources/` - Resource-specific secrets (API keys, etc.)

## Troubleshooting

### Common Issues

1. **Container not running**
   ```bash
   ./manage.sh --action start
   ```

2. **Vault is sealed (prod mode)**
   ```bash
   ./manage.sh --action unseal
   ```

3. **Port conflict**
   ```bash
   # Check what's using port 8200
   lsof -i :8200
   
   # Stop Vault and change port
   ./manage.sh --action stop
   export VAULT_PORT=8201
   ./manage.sh --action start
   ```

4. **View logs**
   ```bash
   ./manage.sh --action logs --lines 50
   ```

5. **Full diagnostics**
   ```bash
   ./manage.sh --action diagnose
   ```

## Security Best Practices

1. **Development Mode**
   - Root token is stored in `/tmp/vault-token`
   - Vault is automatically unsealed
   - Data is stored in memory (lost on restart)
   - Good for testing and development

2. **Production Mode**
   - Store unseal keys in separate secure locations
   - Never store all unseal keys together
   - Use proper authentication methods (LDAP, GitHub, etc.)
   - Enable audit logging
   - Use TLS/HTTPS
   - Implement proper access policies

3. **Secret Management**
   - Use descriptive paths: `environments/prod/database/primary`
   - Rotate secrets regularly
   - Don't commit tokens or keys to git
   - Use environment-specific namespaces

## Next Steps

1. Explore the Vault UI: http://localhost:8200 (use root token to login)
2. Set up authentication methods for team access
3. Create policies for least-privilege access
4. Integrate with your CI/CD pipeline
5. Set up secret rotation schedules