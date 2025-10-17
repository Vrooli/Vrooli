# Vault Quick Start Guide

This guide will help you get started with HashiCorp Vault in the Vrooli ecosystem.

## Installation and Setup

### 1. Install Vault (Development Mode)
```bash
cd resources/vault
./cli.sh manage install
./cli.sh content init-dev
```

This will:
- Pull the Vault Docker image
- Start Vault in development mode (automatically unsealed)
- Create a root token stored in `/tmp/vault-token`
- Set up the KV v2 secret engine
- Create default namespace paths

### 2. Check Status
```bash
./cli.sh status
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
./cli.sh content add --path "environments/dev/api-key" --value "my-secret-api-key"

# Store with custom key name
./cli.sh content add --path "environments/dev/database" --value "postgres://user:pass@host:5432/db" --key "connection_string"
```

### 2. Retrieve a Secret
```bash
# Get a specific secret
./cli.sh content get --path "environments/dev/api-key"

# Get as JSON (useful for multiple keys)
./cli.sh content get --path "environments/dev/database" --format json
```

### 3. List Secrets
```bash
# List all secrets in a path
./cli.sh content list --path "environments/"

# List as JSON
./cli.sh content list --path "environments/" --format json
```

### 4. Delete a Secret
```bash
./cli.sh content remove --path "environments/dev/api-key"
```

## Advanced Features

### 1. Bulk Operations
```bash
# Store multiple secrets from JSON (Note: This feature may need implementation)
echo '{"api_key": "secret123", "db_password": "pass456"}' > secrets.json
# Individual secret storage for now:
./cli.sh content add --path "environments/dev/api_key" --value "secret123"
./cli.sh content add --path "environments/dev/db_password" --value "pass456"
rm secrets.json

# Export secrets (Note: Manual export for now)
./cli.sh content get --path "environments/dev/api_key" > api_key.txt
```

### 2. Environment Migration
```bash
# Migrate .env file to Vault
./cli.sh content migrate-env --env-file .env --vault-prefix "environments/dev"
```

### 3. Secret Rotation
```bash
# Note: Manual rotation for now - generate new values and store them
# Generate new random value
NEW_SECRET=$(openssl rand -hex 32)
./cli.sh content add --path "environments/dev/api-key" --value "$NEW_SECRET"
```

### 4. Monitoring
```bash
# Show detailed diagnostics
./cli.sh diagnose

# Monitor health continuously
./cli.sh monitor --interval 5

# Follow container logs
./cli.sh logs --follow
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
./cli.sh content init-prod

# This will:
# - Initialize Vault with 5 unseal keys
# - Save keys to vault config directory (SAVE THESE SECURELY!)
# - Save root token to vault config directory
# - Require manual unsealing after restarts

# Unseal after restart (needs 3 of 5 keys)
./cli.sh content unseal

# Check seal status
./cli.sh status
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
   ./cli.sh manage start
   ```

2. **Vault is sealed (prod mode)**
   ```bash
   ./cli.sh content unseal
   ```

3. **Port conflict**
   ```bash
   # Check what's using port 8200
   lsof -i :8200
   
   # Stop Vault and change port
   ./cli.sh manage stop
   export VAULT_PORT=8201
   ./cli.sh manage start
   ```

4. **View logs**
   ```bash
   ./cli.sh logs --lines 50
   ```

5. **Full diagnostics**
   ```bash
   ./cli.sh diagnose
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