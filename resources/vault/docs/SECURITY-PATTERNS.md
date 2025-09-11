# Vault Security Integration Patterns

## Overview
This document provides security patterns and best practices for integrating Vault with Vrooli scenarios and resources.

## Table of Contents
- [Secret Management Patterns](#secret-management-patterns)
- [Audit Logging](#audit-logging)
- [Access Control](#access-control)
- [Security Health Monitoring](#security-health-monitoring)
- [Integration Examples](#integration-examples)

## Secret Management Patterns

### Basic Secret Storage
Store and retrieve API keys, credentials, and sensitive configuration:

```bash
# Store a secret using v2.0 compliant command
vrooli resource vault content add --name api/openrouter --data '{"key":"sk-or-v1-..."}'

# Retrieve a secret
vrooli resource vault content get --name api/openrouter

# List all secrets
vrooli resource vault content list

# Remove a secret
vrooli resource vault content remove --name api/openrouter
```

### Structured Secret Management
Organize secrets hierarchically for better management:

```bash
# Store database credentials
vrooli resource vault content add --name db/postgres/prod --data '{
  "host": "localhost",
  "port": "5432",
  "username": "app_user",
  "password": "secure_pass123"
}'

# Store API keys by provider
vrooli resource vault content add --name ai/openai/key --data '{"api_key":"sk-..."}'
vrooli resource vault content add --name ai/anthropic/key --data '{"api_key":"sk-ant-..."}'
```

### Dynamic Secret Retrieval in Scripts
```bash
#!/usr/bin/env bash
# Example: Retrieve database credentials dynamically

DB_CREDS=$(vrooli resource vault content get --name db/postgres/prod)
DB_HOST=$(echo "$DB_CREDS" | jq -r '.host')
DB_PORT=$(echo "$DB_CREDS" | jq -r '.port')
DB_USER=$(echo "$DB_CREDS" | jq -r '.username')
DB_PASS=$(echo "$DB_CREDS" | jq -r '.password')

# Use credentials
psql "postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/mydb"
```

## Audit Logging

### Enable Audit Logging
Track all secret access for compliance and security monitoring:

```bash
# Enable file-based audit logging
vrooli resource vault audit enable

# Enable with custom path
vrooli resource vault audit enable --device custom --path /vault/audit/custom.log

# List audit devices
vrooli resource vault audit list

# Disable audit logging
vrooli resource vault audit disable
```

### Analyze Audit Logs
```bash
# View recent secret access
vrooli resource vault audit analyze --recent 100

# Filter by operation type
vrooli resource vault audit analyze --operation read --path secret/*

# Generate audit report
vrooli resource vault audit analyze --report > audit-report.json
```

## Access Control

### Create Access Policies
Define fine-grained access control for different scenarios:

```bash
# Create read-only policy for AI scenarios
cat > ai-readonly.hcl <<EOF
path "secret/data/ai/*" {
  capabilities = ["read", "list"]
}
EOF
vrooli resource vault access create-policy ai-readonly ai-readonly.hcl

# Create admin policy
cat > admin.hcl <<EOF
path "*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
EOF
vrooli resource vault access create-policy admin admin.hcl
```

### Generate Scoped Tokens
```bash
# Create token with specific policy
vrooli resource vault access create-token --policy ai-readonly --ttl 1h

# Create token for scenario use
vrooli resource vault access create-token \
  --policy scenario-x \
  --display-name "scenario-x-token" \
  --ttl 24h
```

## Security Health Monitoring

### Comprehensive Security Check
Monitor Vault's security posture:

```bash
# Run full security health check
vrooli resource vault security-health

# Monitor security events in real-time
vrooli resource vault security-monitor

# Generate security audit report
vrooli resource vault security-audit > security-audit.json
```

### Automated Health Checks in Smoke Tests
The enhanced smoke tests now include security validation:
- Encryption status verification
- Seal status monitoring
- Audit logging status
- Access control policy validation
- Token expiration checks
- TLS configuration assessment

## Integration Examples

### Scenario Integration Pattern
```javascript
// Example: Node.js scenario retrieving secrets
const { exec } = require('child_process');
const util = require('util');
const execPromise = util.promisify(exec);

async function getSecret(path) {
  const { stdout } = await execPromise(
    `vrooli resource vault content get --name ${path}`
  );
  return JSON.parse(stdout);
}

// Usage in scenario
async function initializeAI() {
  const apiKeys = await getSecret('ai/openai/key');
  process.env.OPENAI_API_KEY = apiKeys.api_key;
}
```

### Docker Compose Integration
```yaml
version: '3.8'
services:
  my-app:
    image: my-app:latest
    environment:
      - VAULT_ADDR=http://vault:8200
      - VAULT_TOKEN=${VAULT_TOKEN}
    command: |
      sh -c "
        # Retrieve secrets at startup
        DB_PASS=$$(vault kv get -field=password secret/db/postgres)
        API_KEY=$$(vault kv get -field=api_key secret/api/service)
        # Start application with secrets
        exec my-app --db-pass=$$DB_PASS --api-key=$$API_KEY
      "
```

### N8n Workflow Integration
The Vault resource includes an N8n integration example at:
`examples/n8n-vault-integration.json`

This workflow demonstrates:
- Retrieving secrets from Vault
- Using secrets in API calls
- Storing workflow results back to Vault
- Audit logging of workflow secret access

### Migration from .env Files
Migrate existing environment files to Vault:

```bash
# Migrate .env file to Vault
vrooli resource vault content migrate-env --file .env --prefix app/config

# Example .env file:
# DATABASE_URL=postgresql://user:pass@localhost/db
# API_KEY=sk-123456
# SECRET_TOKEN=abc123

# Results in Vault:
# app/config/DATABASE_URL
# app/config/API_KEY
# app/config/SECRET_TOKEN
```

## Best Practices

### 1. Namespace Organization
- Use hierarchical paths: `environment/service/key`
- Example: `prod/api/openai/key`, `dev/db/postgres/password`

### 2. Token Management
- Use short-lived tokens (1-24 hours)
- Implement token renewal for long-running processes
- Never commit tokens to version control

### 3. Audit Requirements
- Enable audit logging in production
- Regularly review audit logs
- Set up alerts for suspicious access patterns

### 4. Secret Rotation
- Implement regular rotation schedules
- Use dynamic secrets where possible
- Monitor secret age and usage

### 5. Access Control
- Follow principle of least privilege
- Create specific policies per scenario
- Review and update policies regularly

## Security Compliance Checklist

- [ ] Audit logging enabled
- [ ] Access policies defined
- [ ] Tokens have appropriate TTLs
- [ ] No hardcoded secrets in code
- [ ] TLS enabled for production
- [ ] Regular security health checks
- [ ] Secret rotation implemented
- [ ] Backup and recovery tested

## Troubleshooting

### Common Issues

**Issue**: "Vault is sealed"
```bash
# Check seal status
vrooli resource vault status

# Unseal Vault (dev mode auto-unseals)
vrooli resource vault content unseal
```

**Issue**: "Permission denied"
```bash
# Check token validity
vrooli resource vault test auth

# Renew token
export VAULT_TOKEN=$(vrooli resource vault access create-token --ttl 1h)
```

**Issue**: "Audit logs not appearing"
```bash
# Verify audit is enabled
vrooli resource vault audit list

# Re-enable if needed
vrooli resource vault audit disable
vrooli resource vault audit enable
```

## Additional Resources

- [Vault Best Practices](https://developer.hashicorp.com/vault/docs/internals/security)
- [Vault Security Model](https://developer.hashicorp.com/vault/docs/internals/security)
- [Vrooli Vault PRD](../PRD.md)
- [Vault API Documentation](https://developer.hashicorp.com/vault/api-docs)