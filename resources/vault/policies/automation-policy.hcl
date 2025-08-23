# Vault Policy for Automation Systems (n8n, Node-RED, etc.)
# Provides read-only access to specific secret paths needed for automation workflows

# Allow reading environment-specific secrets
path "secret/data/vrooli/environments/development/*" {
  capabilities = ["read"]
}

path "secret/data/vrooli/environments/staging/*" {
  capabilities = ["read"]
}

# Limited production access (only specific automation secrets)
path "secret/data/vrooli/environments/production/automation/*" {
  capabilities = ["read"]
}

# Allow reading resource credentials
path "secret/data/vrooli/resources/*" {
  capabilities = ["read"]
}

# Allow reading client secrets for automation workflows
path "secret/data/vrooli/clients/*/automation/*" {
  capabilities = ["read"]
}

# Allow reading ephemeral/temporary secrets
path "secret/data/vrooli/ephemeral/*" {
  capabilities = ["read", "create", "update", "delete"]
}

# Allow listing secret paths (for discovery)
path "secret/metadata/vrooli/environments/" {
  capabilities = ["list"]
}

path "secret/metadata/vrooli/resources/" {
  capabilities = ["list"]
}

path "secret/metadata/vrooli/clients/" {
  capabilities = ["list"]
}

path "secret/metadata/vrooli/ephemeral/" {
  capabilities = ["list"]
}

# Allow reading own token information
path "auth/token/lookup-self" {
  capabilities = ["read"]
}

# Allow renewing own token
path "auth/token/renew-self" {
  capabilities = ["update"]
}