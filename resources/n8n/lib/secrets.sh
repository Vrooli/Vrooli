#!/usr/bin/env bash
################################################################################
# n8n Secrets Management - Vault Integration
# 
# Handles secure credential management through Vault or environment variables
################################################################################

set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_DIR="${APP_ROOT}/resources/n8n"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Secret validation and loading
################################################################################

n8n::secrets::load() {
    log::info "Loading n8n secrets configuration..."
    
    # Check if Vault is available
    if docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "vrooli-vault"; then
        log::info "Vault detected - attempting to load secrets from Vault"
        n8n::secrets::load_from_vault
    else
        log::info "Vault not available - using environment variables"
        n8n::secrets::load_from_env
    fi
}

n8n::secrets::load_from_vault() {
    # This would integrate with the Vault resource to fetch secrets
    # For now, we'll implement a placeholder that falls back to env vars
    
    log::warn "Vault integration pending - falling back to environment variables"
    n8n::secrets::load_from_env
}

n8n::secrets::load_from_env() {
    local missing_required=()
    
    # Check required secrets from environment
    if [[ -z "${DB_POSTGRESDB_PASSWORD:-}" ]]; then
        # Try to get from PostgreSQL resource if available
        if command -v resource-postgres &>/dev/null; then
            DB_POSTGRESDB_PASSWORD=$(resource-postgres credentials --format=env | grep POSTGRES_PASSWORD | cut -d= -f2)
            export DB_POSTGRESDB_PASSWORD
        else
            missing_required+=("DB_POSTGRESDB_PASSWORD")
        fi
    fi
    
    # Generate encryption key if not provided
    if [[ -z "${N8N_ENCRYPTION_KEY:-}" ]]; then
        log::info "Generating n8n encryption key..."
        N8N_ENCRYPTION_KEY=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | base64 | tr -d '\n')
        export N8N_ENCRYPTION_KEY
        log::success "Generated encryption key for n8n"
    fi
    
    # Report missing required secrets
    if [[ ${#missing_required[@]} -gt 0 ]]; then
        log::warn "Missing required secrets: ${missing_required[*]}"
        log::info "n8n will attempt to use default values or prompt during setup"
    else
        log::success "All required secrets loaded"
    fi
}

n8n::secrets::validate() {
    log::info "Validating n8n secrets configuration..."
    
    local validation_passed=true
    
    # Check encryption key
    if [[ -z "${N8N_ENCRYPTION_KEY:-}" ]]; then
        log::error "Missing encryption key"
        validation_passed=false
    elif [[ ${#N8N_ENCRYPTION_KEY} -lt 32 ]]; then
        log::error "Encryption key too short (minimum 32 characters)"
        validation_passed=false
    fi
    
    # Check database password
    if [[ -z "${DB_POSTGRESDB_PASSWORD:-}" ]]; then
        log::warn "Database password not set - will use default or prompt"
    fi
    
    # Check optional secrets and report status
    if [[ -n "${N8N_BASIC_AUTH_USER:-}" ]] && [[ -n "${N8N_BASIC_AUTH_PASSWORD:-}" ]]; then
        log::info "Basic authentication configured"
    fi
    
    if [[ -n "${OPENAI_API_KEY:-}" ]]; then
        log::info "OpenAI integration configured"
    fi
    
    if [[ -n "${N8N_SMTP_HOST:-}" ]]; then
        log::info "SMTP email integration configured"
    fi
    
    if [[ "$validation_passed" == true ]]; then
        log::success "Secrets validation passed"
        return 0
    else
        log::error "Secrets validation failed"
        return 1
    fi
}

n8n::secrets::show() {
    local show_secrets="${1:-false}"
    
    log::info "n8n Secrets Configuration:"
    echo ""
    
    # Required secrets
    echo "Required Secrets:"
    if [[ -n "${N8N_ENCRYPTION_KEY:-}" ]]; then
        if [[ "$show_secrets" == "true" ]]; then
            echo "  Encryption Key: $N8N_ENCRYPTION_KEY"
        else
            echo "  Encryption Key: [SET]"
        fi
    else
        echo "  Encryption Key: [NOT SET]"
    fi
    
    if [[ -n "${DB_POSTGRESDB_PASSWORD:-}" ]]; then
        if [[ "$show_secrets" == "true" ]]; then
            echo "  Database Password: $DB_POSTGRESDB_PASSWORD"
        else
            echo "  Database Password: [SET]"
        fi
    else
        echo "  Database Password: [NOT SET]"
    fi
    
    echo ""
    echo "Optional Secrets:"
    
    # Basic auth
    if [[ -n "${N8N_BASIC_AUTH_USER:-}" ]]; then
        echo "  Basic Auth User: ${N8N_BASIC_AUTH_USER}"
        if [[ -n "${N8N_BASIC_AUTH_PASSWORD:-}" ]]; then
            if [[ "$show_secrets" == "true" ]]; then
                echo "  Basic Auth Password: ${N8N_BASIC_AUTH_PASSWORD}"
            else
                echo "  Basic Auth Password: [SET]"
            fi
        fi
    else
        echo "  Basic Auth: [DISABLED]"
    fi
    
    # API integrations
    if [[ -n "${OPENAI_API_KEY:-}" ]]; then
        if [[ "$show_secrets" == "true" ]]; then
            echo "  OpenAI API Key: ${OPENAI_API_KEY}"
        else
            echo "  OpenAI API Key: [SET]"
        fi
    else
        echo "  OpenAI API Key: [NOT SET]"
    fi
    
    # SMTP
    if [[ -n "${N8N_SMTP_HOST:-}" ]]; then
        echo "  SMTP Host: ${N8N_SMTP_HOST}"
        echo "  SMTP Port: ${N8N_SMTP_PORT:-587}"
        if [[ -n "${N8N_SMTP_USER:-}" ]]; then
            echo "  SMTP User: ${N8N_SMTP_USER}"
        fi
    else
        echo "  SMTP: [NOT CONFIGURED]"
    fi
}

n8n::secrets::init_prompt() {
    log::info "n8n Secrets Configuration Wizard"
    echo ""
    
    # Encryption key
    if [[ -z "${N8N_ENCRYPTION_KEY:-}" ]]; then
        read -p "Generate encryption key automatically? (y/n): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            N8N_ENCRYPTION_KEY=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | base64 | tr -d '\n')
            export N8N_ENCRYPTION_KEY
            log::success "Generated encryption key"
        fi
    fi
    
    # Basic auth
    read -p "Enable basic authentication for n8n UI? (y/n): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Username: " N8N_BASIC_AUTH_USER
        read -s -p "Password: " N8N_BASIC_AUTH_PASSWORD
        echo ""
        export N8N_BASIC_AUTH_USER
        export N8N_BASIC_AUTH_PASSWORD
        export N8N_BASIC_AUTH_ACTIVE=true
        log::success "Basic authentication configured"
    fi
    
    # OpenAI integration
    read -p "Configure OpenAI API integration? (y/n): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "OpenAI API Key: " OPENAI_API_KEY
        export OPENAI_API_KEY
        log::success "OpenAI integration configured"
    fi
    
    log::success "Secrets configuration complete"
}

################################################################################
# Export functions
################################################################################

export -f n8n::secrets::load
export -f n8n::secrets::load_from_vault
export -f n8n::secrets::load_from_env
export -f n8n::secrets::validate
export -f n8n::secrets::show
export -f n8n::secrets::init_prompt