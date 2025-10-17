# Vrooli Resource Secrets Standard v1.0

## Overview
This document defines the standard for how Vrooli resources declare and manage their secrets through integration with Vault.

## File Structure

Each resource that requires secrets MUST define them in `config/secrets.yaml`:

```yaml
# config/secrets.yaml - Resource Secrets Declaration
version: "1.0"
resource: "resource-name"
description: "Brief description of what secrets this resource needs"

# Secrets are organized by category for clarity
secrets:
  # API credentials
  api_keys:
    - name: "primary_api_key"
      path: "secret/resources/{resource}/api/primary"  # Vault path
      description: "Main API key for service authentication"
      required: true
      format: "string"
      validation:
        pattern: "^[A-Za-z0-9_-]{32,}$"  # Optional regex validation
      default_env: "RESOURCE_API_KEY"  # Environment variable to export
      example: "sk-proj-abc123..."  # Example value (never real!)
      
  # Database credentials  
  database:
    - name: "postgres_password"
      path: "secret/resources/{resource}/db/postgres"
      description: "PostgreSQL database password"
      required: false  # Optional secret
      format: "string"
      default_env: "POSTGRES_PASSWORD"
      fallback: "env:POSTGRES_PASSWORD"  # Try env var if not in Vault
      
  # OAuth/tokens
  tokens:
    - name: "github_token"
      path: "secret/resources/{resource}/tokens/github"
      description: "GitHub personal access token"
      required: false
      format: "string"
      default_env: "GITHUB_TOKEN"
      ttl: "24h"  # Token lifetime hint
      renewable: true  # Can be renewed
      
  # Certificates/keys
  certificates:
    - name: "tls_cert"
      path: "secret/resources/{resource}/certs/tls"
      description: "TLS certificate for HTTPS"
      required: false
      format: "pem"
      fields:  # Multi-field secret
        - cert: "certificate"
        - key: "private_key"
        - ca: "ca_bundle"
      default_env: "TLS_CERT_PATH"  # Points to file path
      
# Secret dependencies on other resources
dependencies:
  - resource: "postgres"
    secrets: ["connection_string"]
    purpose: "Database connectivity"
    
# Initialization hints for vault
initialization:
  auto_generate:  # Secrets Vault can auto-generate
    - name: "internal_token"
      type: "uuid"
      path: "secret/resources/{resource}/internal/token"
      
  prompt_user:  # Secrets that need user input
    - name: "primary_api_key"
      prompt: "Enter your API key from https://provider.com/api/keys"
      validation: "Test connection to API endpoint"
      
# Health check that validates secrets are properly configured
health_check:
  endpoint: "/health/secrets"  # Optional endpoint to verify secrets work
  required_secrets: ["primary_api_key"]  # Must have these to be healthy
```

## Vault Integration Commands

### For Vault Resource:

```bash
# Scan all resources for secrets requirements
resource-vault secrets scan
# Output: Lists all resources with secrets.yaml and their status

# Check specific resource secrets
resource-vault secrets check <resource-name>
# Output: Shows which secrets are set/missing for that resource

# Initialize secrets for a resource
resource-vault secrets init <resource-name>
# Interactive: Prompts for required secrets, generates auto ones

# Validate all secrets are properly set
resource-vault secrets validate
# Output: Full report of all resources and secret status

# Export secrets as environment variables
resource-vault secrets export <resource-name>
# Output: Shell commands to set environment variables
```

### For Individual Resources:

```bash
# Generate secrets.yaml template for a resource
resource-<name> secrets init
# Creates config/secrets.yaml with resource-specific template

# Check if secrets are configured
resource-<name> secrets check
# Output: Status of each required secret

# Test secrets work
resource-<name> secrets test
# Runs health check with actual secrets
```

## Implementation Pattern

### 1. Resource Declares Secrets (config/secrets.yaml)
```yaml
secrets:
  api_keys:
    - name: "openai_key"
      path: "secret/resources/openrouter/api/openai"
      required: true
      default_env: "OPENAI_API_KEY"
```

### 2. Vault Manages Secrets
```bash
# Vault stores at standardized paths
/secret/resources/{resource-name}/{category}/{secret-name}

# Examples:
/secret/resources/openrouter/api/openai
/secret/resources/postgres/db/password
/secret/resources/n8n/tokens/webhook
```

### 3. Resource Loads Secrets
```bash
# In resource's lib/core.sh or lib/common.sh
load_secrets() {
    # Check if Vault is available
    if resource-vault status &>/dev/null; then
        # Load from Vault
        eval "$(resource-vault secrets export ${RESOURCE_NAME})"
    else
        # Fall back to environment variables
        log_warn "Vault not available, using environment variables"
    fi
}
```

## Security Best Practices

1. **Never commit real secrets** - Only examples and patterns
2. **Use least privilege** - Resources only access their namespace
3. **Rotate regularly** - Support TTL and renewal
4. **Audit access** - Log all secret retrievals
5. **Validate format** - Enforce patterns for API keys
6. **Secure defaults** - Fail closed if secrets missing
7. **Clear documentation** - How to obtain each secret

## Migration Path

For existing resources:
1. Run `resource-vault secrets discover <resource>` to scan for hardcoded secrets
2. Generate template with `resource-<name> secrets init`
3. Move secrets to Vault with `resource-vault secrets migrate <resource>`
4. Update resource to use `load_secrets()` function
5. Test with `resource-<name> secrets test`

## Standard Paths in Vault

```
/secret/resources/{resource}/
├── api/          # API keys and tokens
├── db/           # Database credentials
├── certs/        # Certificates and keys
├── tokens/       # OAuth/JWT tokens
├── internal/     # Internal service tokens
└── config/       # Sensitive configuration
```

## Benefits

1. **Centralized Management** - Single source of truth for all secrets
2. **Standardized Access** - Consistent pattern across all resources
3. **Automatic Discovery** - Vault can scan and report on all secrets
4. **Easy Onboarding** - New users can initialize all secrets at once
5. **Security Compliance** - Audit trail and access control
6. **Environment Flexibility** - Dev/staging/prod separation
7. **Dependency Awareness** - Know which resources need which secrets

## Example: OpenRouter Resource

```yaml
# resources/openrouter/config/secrets.yaml
version: "1.0"
resource: "openrouter"
description: "API keys for various AI model providers"

secrets:
  api_keys:
    - name: "openrouter_key"
      path: "secret/resources/openrouter/api/main"
      description: "Main OpenRouter API key"
      required: true
      format: "string"
      validation:
        pattern: "^sk-or-v1-[a-f0-9]{64}$"
      default_env: "OPENROUTER_API_KEY"
      example: "sk-or-v1-abc123..."
      
initialization:
  prompt_user:
    - name: "openrouter_key"
      prompt: "Enter your OpenRouter API key from https://openrouter.ai/keys"
      validation: "Test with a simple completion request"
      
health_check:
  required_secrets: ["openrouter_key"]
```

This standard ensures that:
- Every resource clearly declares its secret requirements
- Vault can automatically manage and validate secrets
- Users have a consistent experience across all resources
- Security is maintained through centralized management
- The system can self-document its secret dependencies