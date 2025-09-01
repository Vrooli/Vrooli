#!/usr/bin/env bash
# Example: Migrating .env files to Vault

# This script demonstrates how to migrate environment files to Vault
# for different environments and use cases

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_CLI="resource-vault"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Example 1: Migrate development environment
migrate_development() {
    log "Migrating development environment..."
    
    # Create example .env.development file
    cat > /tmp/example.env.development << 'EOF'
DATABASE_URL=postgresql://localhost:5432/vrooli_dev
REDIS_URL=redis://localhost:6379
STRIPE_SECRET_KEY=sk_test_12345
SENDGRID_API_KEY=SG.dev_key_here
JWT_SECRET=dev-jwt-secret-key
OAUTH_GITHUB_CLIENT_ID=dev_github_client_id
OAUTH_GITHUB_CLIENT_SECRET=dev_github_client_secret
EOF

    # Migrate to Vault using v2.0 CLI patterns
    "$VAULT_CLI" content execute migrate-env \
        --env-file /tmp/example.env.development \
        --vault-prefix "environments/development"
    
    # Clean up
    rm /tmp/example.env.development
    
    log "Development environment migrated successfully"
}

# Example 2: Migrate client-specific secrets
migrate_client_secrets() {
    local client_name="$1"
    
    log "Migrating secrets for client: $client_name"
    
    # Create example client secrets file
    cat > "/tmp/client-${client_name}.env" << EOF
STRIPE_API_KEY=sk_live_client_${client_name}_key
SALESFORCE_CLIENT_ID=${client_name}_salesforce_id
SALESFORCE_CLIENT_SECRET=${client_name}_salesforce_secret
WEBHOOK_SECRET=${client_name}_webhook_secret_key
API_RATE_LIMIT=1000
EOF

    # Migrate to Vault with client namespace using v2.0 CLI patterns
    "$VAULT_CLI" content execute migrate-env \
        --env-file "/tmp/client-${client_name}.env" \
        --vault-prefix "clients/${client_name}"
    
    # Clean up
    rm "/tmp/client-${client_name}.env"
    
    log "Client $client_name secrets migrated successfully"
}

# Example 3: Migrate production environment (with extra caution)
migrate_production() {
    warn "CAUTION: This will migrate production secrets!"
    warn "Ensure Vault is properly secured and backed up."
    
    read -p "Are you sure you want to proceed? (yes/no): " confirm
    if [[ "$confirm" != "yes" ]]; then
        log "Migration cancelled"
        return 0
    fi
    
    # In real use, this would be your actual .env.production file
    if [[ ! -f ".env.production" ]]; then
        error ".env.production file not found"
        return 1
    fi
    
    # Create backup first
    log "Creating Vault backup before migration..."
    "$VAULT_CLI" manage backup
    
    # Migrate production secrets using v2.0 CLI patterns
    "$VAULT_CLI" content execute migrate-env \
        --env-file ".env.production" \
        --vault-prefix "environments/production"
    
    log "Production environment migrated successfully"
    warn "Remember to update your deployment scripts to use Vault!"
}

# Example 4: Migrate resource credentials
migrate_resource_credentials() {
    log "Migrating resource credentials..."
    
    # Create example resource credentials
    cat > /tmp/resource-credentials.env << 'EOF'
N8N_API_KEY=n8n-api-key-12345
OLLAMA_API_KEY=ollama-api-key-67890
MINIO_ACCESS_KEY=minio-access-key
MINIO_SECRET_KEY=minio-secret-key
BROWSERLESS_TOKEN=browserless-token-abcdef
EOF

    # Migrate to Vault using v2.0 CLI patterns
    "$VAULT_CLI" content execute migrate-env \
        --env-file /tmp/resource-credentials.env \
        --vault-prefix "resources"
    
    # Clean up
    rm /tmp/resource-credentials.env
    
    log "Resource credentials migrated successfully"
}

# Example 5: Show migrated secrets
show_migrated_secrets() {
    log "Showing migrated secrets structure..."
    
    echo -e "\n${BLUE}Environments:${NC}"
    "$VAULT_CLI" content list --path "environments/" --format list 2>/dev/null || echo "  No environment secrets found"
    
    echo -e "\n${BLUE}Clients:${NC}"
    "$VAULT_CLI" content list --path "clients/" --format list 2>/dev/null || echo "  No client secrets found"
    
    echo -e "\n${BLUE}Resources:${NC}"
    "$VAULT_CLI" content list --path "resources/" --format list 2>/dev/null || echo "  No resource secrets found"
}

# Example 6: Test secret retrieval
test_secret_retrieval() {
    log "Testing secret retrieval..."
    
    # Try to retrieve a development secret using v2.0 CLI patterns
    if database_url=$("$VAULT_CLI" content get --path "environments/development/database_url" 2>/dev/null); then
        log "Successfully retrieved database URL: ${database_url:0:20}..."
    else
        warn "Could not retrieve development database URL"
    fi
    
    # Try to retrieve a client secret
    if stripe_key=$("$VAULT_CLI" content get --path "clients/acme-corp/stripe_api_key" 2>/dev/null); then
        log "Successfully retrieved client Stripe key: ${stripe_key:0:10}..."
    else
        warn "Could not retrieve client Stripe key"
    fi
}

# Main execution
main() {
    log "Vault Migration Example Script"
    log "This script demonstrates various migration patterns"
    echo
    
    # Check if Vault is ready using v2.0 CLI patterns
    if ! "$VAULT_CLI" status >/dev/null 2>&1; then
        error "Vault is not ready. Please initialize Vault first:"
        error "  $VAULT_CLI manage init-dev"
        exit 1
    fi
    
    case "${1:-demo}" in
        "development")
            migrate_development
            ;;
        "client")
            if [[ -z "${2:-}" ]]; then
                error "Client name required: $0 client <client-name>"
                exit 1
            fi
            migrate_client_secrets "$2"
            ;;
        "production")
            migrate_production
            ;;
        "resources")
            migrate_resource_credentials
            ;;
        "show")
            show_migrated_secrets
            ;;
        "test")
            test_secret_retrieval
            ;;
        "demo")
            log "Running full demonstration..."
            migrate_development
            migrate_client_secrets "acme-corp"
            migrate_client_secrets "globex-ltd"
            migrate_resource_credentials
            echo
            show_migrated_secrets
            echo
            test_secret_retrieval
            ;;
        *)
            echo "Usage: $0 [development|client <name>|production|resources|show|test|demo]"
            echo
            echo "Examples:"
            echo "  $0 development              # Migrate development environment"
            echo "  $0 client acme-corp         # Migrate client secrets"
            echo "  $0 production               # Migrate production environment"
            echo "  $0 resources                # Migrate resource credentials"  
            echo "  $0 show                     # Show migrated secrets"
            echo "  $0 test                     # Test secret retrieval"
            echo "  $0 demo                     # Run full demonstration"
            exit 1
            ;;
    esac
    
    log "Migration example completed successfully"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi