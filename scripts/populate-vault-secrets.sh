#!/bin/bash
# Automated script to populate Vault with production secrets from .env-prod
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Default values
VAULT_ADDR="${VAULT_ADDR:-http://vault.vault.svc.cluster.local:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-root}"
ENV_FILE="${PROJECT_DIR}/.env-prod"

# Check if .env-prod exists
if [ ! -f "$ENV_FILE" ]; then
    log_error ".env-prod file not found at $ENV_FILE"
    exit 1
fi

log_info "🚀 Populating Vault with production secrets..."
log_info "Vault Address: $VAULT_ADDR"
log_info "Environment File: $ENV_FILE"

# Export Vault configuration
export VAULT_ADDR
export VAULT_TOKEN

# Test Vault connection
log_info "Testing Vault connection..."
if ! vault status > /dev/null 2>&1; then
    log_error "Cannot connect to Vault at $VAULT_ADDR"
    log_error "Make sure Vault is running and VAULT_ADDR/VAULT_TOKEN are correct"
    exit 1
fi
log_success "Connected to Vault"

# Enable KV v2 secrets engine if not already enabled
log_info "Ensuring KV v2 secrets engine is enabled..."
if ! vault secrets list | grep -q "secret/"; then
    vault secrets enable -path=secret kv-v2
    log_success "KV v2 secrets engine enabled"
else
    log_info "KV v2 secrets engine already enabled"
fi

# Source .env-prod to get variables
log_info "Loading variables from $ENV_FILE..."
set -a
source "$ENV_FILE"
set +a

# 1. Shared Configuration (non-sensitive)
log_info "Writing shared configuration..."
vault kv put secret/vrooli-prod/config/shared-all \
    PROJECT_DIR="${PROJECT_DIR:-/srv/app}" \
    SITE_IP="${SITE_IP}" \
    API_URL="${API_URL}" \
    VIRTUAL_HOST="${VIRTUAL_HOST}" \
    PORT_DB="${PORT_DB}" \
    PORT_JOBS="${PORT_JOBS}" \
    PORT_SERVER="${PORT_SERVER}" \
    PORT_REDIS="${PORT_REDIS}" \
    PORT_UI="${PORT_UI}" \
    CREATE_MOCK_DATA="${CREATE_MOCK_DATA}" \
    DB_PULL="${DB_PULL}" \
    NODE_ENV="${NODE_ENV}" \
    VITE_GOOGLE_ADSENSE_PUBLISHER_ID="${VITE_GOOGLE_ADSENSE_PUBLISHER_ID}" \
    VITE_GOOGLE_TRACKING_ID="${VITE_GOOGLE_TRACKING_ID}" \
    VITE_STRIPE_PUBLISHABLE_KEY="${VITE_STRIPE_PUBLISHABLE_KEY}" \
    WORKER_ID="1" \
    VITE_SERVER_LOCATION="local"
log_success "Shared configuration written"

# 2. PostgreSQL Credentials
log_info "Writing PostgreSQL credentials..."
vault kv put secret/vrooli-prod/secrets/postgres \
    DB_NAME="${DB_NAME}" \
    DB_USER="${DB_USER}" \
    DB_PASSWORD="${DB_PASSWORD}"
log_success "PostgreSQL credentials written"

# 3. Redis Credentials
log_info "Writing Redis credentials..."
vault kv put secret/vrooli-prod/secrets/redis \
    REDIS_PASSWORD="${REDIS_PASSWORD}"
log_success "Redis credentials written"

# 4. Docker Hub Credentials
log_info "Writing Docker Hub credentials..."
vault kv put secret/vrooli-prod/dockerhub/pull-credentials \
    username="${DOCKERHUB_USERNAME}" \
    password="${DOCKERHUB_TOKEN}"
log_success "Docker Hub credentials written"

# 5. Shared Server/Jobs Secrets (API keys, etc.)
log_info "Writing server/jobs secrets..."
vault kv put secret/vrooli-prod/secrets/shared-server-jobs \
    VAPID_PUBLIC_KEY="${VAPID_PUBLIC_KEY}" \
    VAPID_PRIVATE_KEY="${VAPID_PRIVATE_KEY}" \
    SITE_EMAIL_FROM="${SITE_EMAIL_FROM}" \
    SITE_EMAIL_USERNAME="${SITE_EMAIL_USERNAME}" \
    SITE_EMAIL_PASSWORD="${SITE_EMAIL_PASSWORD}" \
    SITE_EMAIL_ALIAS="${SITE_EMAIL_ALIAS}" \
    STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY}" \
    STRIPE_WEBHOOK_SECRET="${STRIPE_WEBHOOK_SECRET}" \
    TWILIO_ACCOUNT_SID="${TWILIO_ACCOUNT_SID}" \
    TWILIO_AUTH_TOKEN="${TWILIO_AUTH_TOKEN}" \
    TWILIO_PHONE_NUMBER="${TWILIO_PHONE_NUMBER}" \
    AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
    AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
    AWS_REGION="${AWS_REGION}" \
    OPENAI_API_KEY="${OPENAI_API_KEY}" \
    ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
    MISTRAL_API_KEY="${MISTRAL_API_KEY}" \
    ADMIN_WALLET="${ADMIN_WALLET}" \
    ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
    VALYXA_PASSWORD="${VALYXA_PASSWORD}" \
    SECRET_KEY="${SECRET_KEY}"
log_success "Server/jobs secrets written"

# 6. JWT Keys (if they exist)
JWT_PRIV_FILE="${PROJECT_DIR}/jwt_priv_production.pem"
JWT_PUB_FILE="${PROJECT_DIR}/jwt_pub_production.pem"

if [ -f "$JWT_PRIV_FILE" ] && [ -f "$JWT_PUB_FILE" ]; then
    log_info "Adding JWT keys..."
    JWT_PRIV_CONTENT=$(cat "$JWT_PRIV_FILE" | sed ':a;N;$!ba;s/\n/\\n/g')
    JWT_PUB_CONTENT=$(cat "$JWT_PUB_FILE" | sed ':a;N;$!ba;s/\n/\\n/g')
    
    vault kv patch secret/vrooli-prod/secrets/shared-server-jobs \
        JWT_PRIV="$JWT_PRIV_CONTENT" \
        JWT_PUB="$JWT_PUB_CONTENT"
    log_success "JWT keys added"
else
    log_warning "JWT key files not found at $JWT_PRIV_FILE and $JWT_PUB_FILE"
fi

# Verify secrets were written
log_info "Verifying secrets..."
vault kv get secret/vrooli-prod/config/shared-all > /dev/null && log_success "✓ Shared config"
vault kv get secret/vrooli-prod/secrets/postgres > /dev/null && log_success "✓ PostgreSQL secrets"
vault kv get secret/vrooli-prod/secrets/redis > /dev/null && log_success "✓ Redis secrets"
vault kv get secret/vrooli-prod/dockerhub/pull-credentials > /dev/null && log_success "✓ Docker Hub secrets"
vault kv get secret/vrooli-prod/secrets/shared-server-jobs > /dev/null && log_success "✓ Server/jobs secrets"

log_success "🎉 All secrets successfully populated in Vault!"
log_info ""
log_info "Next steps:"
log_info "1. Update your deployment to use vaultAddr: \"$VAULT_ADDR\""
log_info "2. Deploy your application - VSO will automatically sync secrets"
log_info ""
log_info "Vault UI: http://localhost:8200 (if port-forwarded)"
log_info "Token: $VAULT_TOKEN"