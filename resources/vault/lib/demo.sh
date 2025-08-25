#!/usr/bin/env bash
# Vault Demo Script - Shows Vault capabilities

set -euo pipefail

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_LIB_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
cd "$VAULT_LIB_DIR"

# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

echo -e "${BLUE}=== HashiCorp Vault Demo ===${NC}"
echo "This demo will show you how Vault manages secrets"
echo

# Function to pause
pause() {
    echo
    echo -e "${YELLOW}Press Enter to continue...${NC}"
    read -r
}

# 1. Check status
echo -e "${GREEN}1. Checking Vault Status${NC}"
./manage.sh --action status | head -20
pause

# 2. Store some secrets
echo -e "${GREEN}2. Storing Application Secrets${NC}"
echo "Let's store some typical application secrets..."
echo

echo "Storing database credentials..."
./manage.sh --action put-secret \
    --path "environments/dev/database" \
    --value "postgres://appuser:SecurePass123!@localhost:5432/myapp" \
    --key "connection_string"

echo "Storing API key..."
./manage.sh --action put-secret \
    --path "environments/dev/stripe" \
    --value "sk_test_4eC39HqLyjWDarjtT1zdp7dc"

echo "Storing JWT secret..."
./manage.sh --action put-secret \
    --path "environments/dev/auth" \
    --value "super-secret-jwt-signing-key-$(date +%s)" \
    --key "jwt_secret"

echo -e "${GREEN}✓ Secrets stored successfully${NC}"
pause

# 3. List secrets
echo -e "${GREEN}3. Listing Secrets in environments/dev/${NC}"
./manage.sh --action list-secrets --path "environments/dev/"
pause

# 4. Retrieve secrets
echo -e "${GREEN}4. Retrieving Secrets${NC}"
echo
echo "Database connection string:"
./manage.sh --action get-secret --path "environments/dev/database" --key "connection_string"
echo
echo "Stripe API key:"
./manage.sh --action get-secret --path "environments/dev/stripe"
echo
echo "JWT secret (as JSON):"
./manage.sh --action get-secret --path "environments/dev/auth" --format json
pause

# 5. Rotate a secret
echo -e "${GREEN}5. Rotating API Key${NC}"
echo "Let's generate a new API key..."
./manage.sh --action rotate-secret --path "environments/dev/api-key" --type api-key
echo
echo "New API key:"
./manage.sh --action get-secret --path "environments/dev/api-key"
pause

# 6. Environment migration
echo -e "${GREEN}6. Migrating .env File${NC}"
echo "Creating a sample .env file..."
cat > /tmp/demo.env << EOF
# Database
DATABASE_URL=postgres://localhost:5432/oldapp
DATABASE_POOL_SIZE=10

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=oldpassword

# External APIs  
SENDGRID_API_KEY=SG.old-key-here
TWILIO_ACCOUNT_SID=AC1234567890
TWILIO_AUTH_TOKEN=old-auth-token

# App Settings
JWT_SECRET=old-jwt-secret
SESSION_SECRET=old-session-secret
EOF

echo "Contents of .env file:"
cat /tmp/demo.env
echo
echo "Migrating to Vault..."
./manage.sh --action migrate-env --env-file /tmp/demo.env --vault-prefix "demo/migrated"
echo -e "${GREEN}✓ Migration complete${NC}"
echo
echo "Migrated secrets:"
./manage.sh --action list-secrets --path "demo/migrated/"
trash::safe_remove /tmp/demo.env --temp
pause

# 7. Bulk operations
echo -e "${GREEN}7. Bulk Secret Management${NC}"
echo "Creating multiple secrets at once..."
cat > /tmp/demo-bulk.json << EOF
{
  "github_token": "ghp_1234567890abcdef",
  "npm_token": "npm_abcdef1234567890",
  "docker_password": "secure-docker-pass",
  "ssh_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
}
EOF

./manage.sh --action bulk-put --json-file /tmp/demo-bulk.json --base-path "demo/tokens"
echo -e "${GREEN}✓ Bulk import complete${NC}"
echo
echo "Imported tokens:"
./manage.sh --action list-secrets --path "demo/tokens/"
trash::safe_remove /tmp/demo-bulk.json --temp
pause

# 8. Export secrets
echo -e "${GREEN}8. Exporting Secrets${NC}"
echo "Exporting all demo/tokens secrets to JSON..."
./manage.sh --action export-secrets --path "demo/tokens" > /tmp/exported-tokens.json
echo "Exported to /tmp/exported-tokens.json:"
cat /tmp/exported-tokens.json | jq '.'
trash::safe_remove /tmp/exported-tokens.json --temp
pause

# 9. Cleanup
echo -e "${GREEN}9. Cleanup Demo Secrets${NC}"
echo "Removing demo secrets..."
for path in "environments/dev/database" "environments/dev/stripe" "environments/dev/auth" "environments/dev/api-key"; do
    ./manage.sh --action delete-secret --path "$path" 2>/dev/null || true
done

# Clean up migrated and bulk secrets
./manage.sh --action list-secrets --path "demo/" 2>/dev/null | while read -r secret; do
    [[ -n "$secret" ]] && ./manage.sh --action delete-secret --path "demo/$secret" 2>/dev/null || true
done

echo -e "${GREEN}✓ Demo cleanup complete${NC}"
echo
echo -e "${BLUE}=== Demo Complete ===${NC}"
echo
echo "You've seen how to:"
echo "  ✓ Store and retrieve secrets"
echo "  ✓ List and organize secrets by namespace"
echo "  ✓ Rotate secrets with automatic generation"
echo "  ✓ Migrate from .env files"
echo "  ✓ Perform bulk operations"
echo "  ✓ Export secrets for backup"
echo
echo "Next steps:"
echo "  - Access the Vault UI: http://localhost:8200"
echo "  - Read the quickstart guide: cat QUICKSTART.md"
echo "  - Integrate Vault into your applications"