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
resource-vault inject openrouter_api_key "sk-or-v1-xxxxx"

# Store multiple AI provider keys
resource-vault inject ai/openai_key "sk-xxxxx"
resource-vault inject ai/anthropic_key "sk-ant-xxxxx"
resource-vault inject ai/gemini_key "AIzaxxxxx"

# Retrieve for use in scripts
OPENROUTER_KEY=$(resource-vault get-secret openrouter_api_key)
export OPENROUTER_API_KEY="$OPENROUTER_KEY"
EOF
echo

# Pattern 2: Database Credentials
echo "Pattern 2: Database Connection Strings"
echo "--------------------------------------"
cat << 'EOF'
# Store PostgreSQL connection details
resource-vault inject postgres/connection '{
  "host": "localhost",
  "port": 5432,
  "database": "vrooli",
  "user": "vrooli_user",
  "password": "secure_password"
}'

# Retrieve and construct connection string
DB_INFO=$(resource-vault get-secret postgres/connection)
# Parse JSON and build connection string (example with jq)
# DB_URL="postgresql://$(echo $DB_INFO | jq -r '.user'):$(echo $DB_INFO | jq -r '.password')@$(echo $DB_INFO | jq -r '.host'):$(echo $DB_INFO | jq -r '.port')/$(echo $DB_INFO | jq -r '.database')"
EOF
echo

# Pattern 3: OAuth and Social Login
echo "Pattern 3: OAuth Application Credentials"
echo "----------------------------------------"
cat << 'EOF'
# Store OAuth app credentials
resource-vault inject oauth/github '{
  "client_id": "Iv1.xxxxx",
  "client_secret": "xxxxx",
  "redirect_uri": "http://localhost:3000/auth/github/callback"
}'

resource-vault inject oauth/google '{
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
resource-vault inject webhooks/stripe_signing "whsec_xxxxx"
resource-vault inject webhooks/github_webhook "sha256=xxxxx"
resource-vault inject webhooks/slack_signing "xxxxx"

# Verify webhook signatures in your app
STRIPE_WEBHOOK_SECRET=$(resource-vault get-secret webhooks/stripe_signing)
EOF
echo

# Pattern 5: Encryption Keys
echo "Pattern 5: Encryption and JWT Keys"
echo "----------------------------------"
cat << 'EOF'
# Store encryption keys and salts
resource-vault inject encryption/jwt_secret "your-256-bit-secret"
resource-vault inject encryption/cookie_secret "cookie-encryption-key"
resource-vault inject encryption/data_key "AES-256-encryption-key"

# Generate and store a new key
NEW_KEY=$(openssl rand -hex 32)
resource-vault inject encryption/session_key "$NEW_KEY"
EOF
echo

# Pattern 6: Service Account Credentials
echo "Pattern 6: Service Account Files"
echo "--------------------------------"
cat << 'EOF'
# Store Google service account JSON
SERVICE_ACCOUNT=$(cat /path/to/service-account.json)
resource-vault inject gcp/service_account "$SERVICE_ACCOUNT"

# Store AWS credentials
resource-vault inject aws/credentials '{
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
resource-vault inject config/dev/redis_url "redis://localhost:6379"
resource-vault inject config/staging/redis_url "redis://staging.example.com:6379"
resource-vault inject config/prod/redis_url "redis://prod.example.com:6379"

# Retrieve based on environment
ENV=${ENV:-dev}
REDIS_URL=$(resource-vault get-secret "config/$ENV/redis_url")
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