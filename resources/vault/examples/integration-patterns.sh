#!/usr/bin/env bash
# Example: Common Vault integration patterns for Vrooli resources

set -e

echo "=== Vault Integration Patterns Example ==="
echo

# Pattern 1: Storing AI Provider Keys
echo "Pattern 1: AI Provider API Keys"
echo "--------------------------------"
cat << 'EOF'
# Store OpenRouter API key
resource-vault content add --path "openrouter_api_key" --value "sk-or-v1-xxxxx"

# Store multiple AI provider keys
resource-vault content add --path "ai/openai_key" --value "sk-xxxxx"
resource-vault content add --path "ai/anthropic_key" --value "sk-ant-xxxxx"
resource-vault content add --path "ai/gemini_key" --value "AIzaxxxxx"

# Retrieve for use in scripts
OPENROUTER_KEY=$(resource-vault content get --path "openrouter_api_key")
export OPENROUTER_API_KEY="$OPENROUTER_KEY"
EOF
echo

# Pattern 2: Database Credentials
echo "Pattern 2: Database Connection Strings"
echo "--------------------------------------"
cat << 'EOF'
# Store PostgreSQL connection details
resource-vault content add --path "postgres/connection" --value '{
  "host": "localhost",
  "port": 5432,
  "database": "vrooli",
  "user": "vrooli_user",
  "password": "secure_password"
}'

# Retrieve and construct connection string
DB_INFO=$(resource-vault content get --path "postgres/connection")
# Parse JSON and build connection string (example with jq)
# DB_URL="postgresql://$(echo $DB_INFO | jq -r '.user'):$(echo $DB_INFO | jq -r '.password')@$(echo $DB_INFO | jq -r '.host'):$(echo $DB_INFO | jq -r '.port')/$(echo $DB_INFO | jq -r '.database')"
EOF
echo

# Pattern 3: OAuth and Social Login
echo "Pattern 3: OAuth Application Credentials"
echo "----------------------------------------"
cat << 'EOF'
# Store OAuth app credentials
resource-vault content add --path "oauth/github" --value '{
  "client_id": "Iv1.xxxxx",
  "client_secret": "xxxxx",
  "redirect_uri": "http://localhost:3000/auth/github/callback"
}'

resource-vault content add --path "oauth/google" --value '{
  "client_id": "xxxxx.apps.googleusercontent.com",
  "client_secret": "GOCSPX-xxxxx",
  "redirect_uri": "http://localhost:3000/auth/google/callback"
}'
EOF
echo

# Pattern 4: Webhook Secrets
echo "Pattern 4: Webhook Verification Secrets"
echo "---------------------------------------"
cat << 'EOF'
# Store webhook signing secrets
resource-vault content add --path "webhooks/stripe_signing" --value "whsec_xxxxx"
resource-vault content add --path "webhooks/github_webhook" --value "sha256=xxxxx"
resource-vault content add --path "webhooks/slack_signing" --value "xxxxx"

# Verify webhook signatures in your app
STRIPE_WEBHOOK_SECRET=$(resource-vault content get --path "webhooks/stripe_signing")
EOF
echo

# Pattern 5: Encryption Keys
echo "Pattern 5: Encryption and JWT Keys"
echo "----------------------------------"
cat << 'EOF'
# Store encryption keys and salts
resource-vault content add --path "encryption/jwt_secret" --value "your-256-bit-secret"
resource-vault content add --path "encryption/cookie_secret" --value "cookie-encryption-key"
resource-vault content add --path "encryption/data_key" --value "AES-256-encryption-key"

# Generate and store a new key
NEW_KEY=$(openssl rand -hex 32)
resource-vault content add --path "encryption/session_key" --value "$NEW_KEY"
EOF
echo

# Pattern 6: Service Account Credentials
echo "Pattern 6: Service Account Files"
echo "--------------------------------"
cat << 'EOF'
# Store Google service account JSON
SERVICE_ACCOUNT=$(cat /path/to/service-account.json)
resource-vault content add --path "gcp/service_account" --value "$SERVICE_ACCOUNT"

# Store AWS credentials
resource-vault content add --path "aws/credentials" --value '{
  "access_key_id": "AKIA...",
  "secret_access_key": "xxxxx",
  "region": "us-east-1"
}'
EOF
echo

# Pattern 7: Environment-Specific Configs
echo "Pattern 7: Environment Configurations"
echo "------------------------------------"
cat << 'EOF'
# Store environment-specific settings
resource-vault content add --path "config/dev/redis_url" --value "redis://localhost:6379"
resource-vault content add --path "config/staging/redis_url" --value "redis://staging.example.com:6379"
resource-vault content add --path "config/prod/redis_url" --value "redis://prod.example.com:6379"

# Retrieve based on environment
ENV=${ENV:-dev}
REDIS_URL=$(resource-vault content get --path "config/$ENV/redis_url")
EOF
echo

echo
echo "=== Best Practices ==="
echo "1. Use hierarchical paths to organize secrets (e.g., oauth/, ai/, config/)"
echo "2. Store complex credentials as JSON for easier parsing"
echo "3. Never log or print full secret values"
echo "4. Use environment variables for runtime configuration"
echo "5. Rotate secrets regularly and version them if needed"
echo "6. Clean up test/temporary secrets after use"
echo "7. Use Vault's audit logging to track access"