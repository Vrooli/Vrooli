#!/usr/bin/env bash
################################################################################
# Mifos Core Library
# 
# Core functions for managing Mifos X platform
################################################################################

set -euo pipefail

# ==============================================================================
# HEALTH CHECK
# ==============================================================================
mifos::health_check() {
    local timeout="${1:-5}"
    local port="${MIFOS_PORT:-8030}"
    
    log::info "Checking Mifos health on port ${port}..."
    
    if timeout "${timeout}" curl -sf "http://localhost:${port}/fineract-provider/api/v1/offices" \
        -H "Fineract-Platform-TenantId: ${FINERACT_TENANT:-default}" \
        -H "Accept: application/json" &>/dev/null; then
        log::success "Mifos is healthy"
        return 0
    else
        log::error "Mifos health check failed"
        return 1
    fi
}

# ==============================================================================
# STATUS
# ==============================================================================
mifos::status() {
    log::header "Mifos X Status"
    
    # Check if containers are running
    local backend_status="$(docker ps --filter "name=${MIFOS_CONTAINER_PREFIX}-backend" --format "table {{.Status}}" | tail -n +2 || echo "Not running")"
    local webapp_status="$(docker ps --filter "name=${MIFOS_CONTAINER_PREFIX}-webapp" --format "table {{.Status}}" | tail -n +2 || echo "Not running")"
    local db_status="$(docker ps --filter "name=${MIFOS_CONTAINER_PREFIX}-db" --format "table {{.Status}}" | tail -n +2 || echo "Not running")"
    
    echo "Backend: ${backend_status:-Not running}"
    echo "Web App: ${webapp_status:-Not running}"
    echo "Database: ${db_status:-Not running}"
    echo ""
    echo "API URL: ${MIFOS_API_URL:-http://localhost:8030}"
    echo "Web UI: http://localhost:${MIFOS_WEBAPP_PORT:-8031}"
    echo "Tenant: ${FINERACT_TENANT:-default}"
    
    # Try health check
    echo ""
    if mifos::health_check 2; then
        echo "Health: ✓ Healthy"
    else
        echo "Health: ✗ Unhealthy"
    fi
    
    return 0
}

# ==============================================================================
# CREDENTIALS
# ==============================================================================
mifos::core::credentials() {
    log::header "Mifos X Credentials"
    
    echo "Default Credentials:"
    echo "  Username: ${MIFOS_DEFAULT_USER:-mifos}"
    echo "  Password: ${MIFOS_DEFAULT_PASSWORD:-password}"
    echo "  Tenant: ${FINERACT_TENANT:-default}"
    echo ""
    echo "API Endpoints:"
    echo "  Base URL: ${MIFOS_API_URL:-http://localhost:8030}/fineract-provider/api/${MIFOS_API_VERSION:-v1}"
    echo "  Health: http://localhost:${MIFOS_PORT:-8030}/fineract-provider/api/v1/offices"
    echo "  Clients: http://localhost:${MIFOS_PORT:-8030}/fineract-provider/api/v1/clients"
    echo "  Loans: http://localhost:${MIFOS_PORT:-8030}/fineract-provider/api/v1/loans"
    echo "  Savings: http://localhost:${MIFOS_PORT:-8030}/fineract-provider/api/v1/savingsaccounts"
    echo ""
    echo "Web UI:"
    echo "  URL: http://localhost:${MIFOS_WEBAPP_PORT:-8031}"
    echo ""
    echo "Database:"
    echo "  Host: ${FINERACT_DB_HOST:-postgres}"
    echo "  Port: ${FINERACT_DB_PORT:-5433}"
    echo "  Database: ${FINERACT_DB_NAME:-fineract}"
    echo "  User: ${FINERACT_DB_USER:-fineract}"
    
    return 0
}

# ==============================================================================
# VALIDATE CONFIGURATION
# ==============================================================================
mifos::core::validate_config() {
    log::header "Validating Mifos Configuration"
    
    local errors=0
    
    # Check required environment variables
    for var in MIFOS_PORT FINERACT_DB_HOST FINERACT_DB_PORT; do
        if [[ -z "${!var:-}" ]]; then
            log::error "Required variable ${var} is not set"
            ((errors++))
        else
            log::success "${var}=${!var}"
        fi
    done
    
    # Check port availability
    if lsof -Pi :"${MIFOS_PORT:-8030}" -sTCP:LISTEN -t >/dev/null 2>&1; then
        log::error "Port ${MIFOS_PORT:-8030} is already in use"
        ((errors++))
    else
        log::success "Port ${MIFOS_PORT:-8030} is available"
    fi
    
    # Check Docker
    if command -v docker &>/dev/null; then
        log::success "Docker is installed"
    else
        log::error "Docker is not installed"
        ((errors++))
    fi
    
    # Check Docker Compose
    if command -v docker-compose &>/dev/null || docker compose version &>/dev/null; then
        log::success "Docker Compose is available"
    else
        log::error "Docker Compose is not available"
        ((errors++))
    fi
    
    # Check PostgreSQL dependency
    if vrooli resource postgres status &>/dev/null; then
        log::success "PostgreSQL resource is running"
    else
        log::warning "PostgreSQL resource is not running (will be started if needed)"
    fi
    
    if [[ ${errors} -gt 0 ]]; then
        log::error "Configuration validation failed with ${errors} error(s)"
        return 1
    fi
    
    log::success "Configuration validation passed"
    return 0
}

# ==============================================================================
# API AUTHENTICATION
# ==============================================================================
mifos::core::authenticate() {
    local username="${1:-${MIFOS_DEFAULT_USER}}"
    local password="${2:-${MIFOS_DEFAULT_PASSWORD}}"
    local tenant="${3:-${FINERACT_TENANT:-default}}"
    
    local auth_response
    auth_response=$(curl -sf -X POST \
        "http://localhost:${MIFOS_PORT:-8030}/fineract-provider/api/v1/authentication" \
        -H "Content-Type: application/json" \
        -H "Fineract-Platform-TenantId: ${tenant}" \
        -d "{\"username\": \"${username}\", \"password\": \"${password}\"}" 2>/dev/null || echo "{}")
    
    # Check if authenticated (mock server returns token directly)
    local auth_token
    auth_token=$(echo "${auth_response}" | jq -r '.token // .base64EncodedAuthenticationKey // empty')
    
    if [[ -n "${auth_token}" ]]; then
        # Return a base64 encoded token for compatibility
        echo -n "${username}:${password}" | base64
        return 0
    else
        log::error "Authentication failed for user ${username}"
        return 1
    fi
}

# ==============================================================================
# API REQUEST HELPER
# ==============================================================================
mifos::core::api_request() {
    local method="${1}"
    local endpoint="${2}"
    local data="${3:-}"
    local auth_token="${4:-}"
    
    # Get auth token if not provided (optional for mock server)
    if [[ -z "${auth_token}" ]]; then
        auth_token=$(mifos::core::authenticate 2>/dev/null || echo "dGVzdDp0ZXN0")
    fi
    
    # Ensure endpoint starts with /
    [[ "${endpoint:0:1}" != "/" ]] && endpoint="/${endpoint}"
    
    local url="http://localhost:${MIFOS_PORT:-8030}/fineract-provider/api/v1${endpoint}"
    local curl_opts=(-sf -X "${method}" -H "Accept: application/json" 
                     -H "Fineract-Platform-TenantId: ${FINERACT_TENANT:-default}")
    
    # Add auth header if available
    [[ -n "${auth_token}" ]] && curl_opts+=(-H "Authorization: Basic ${auth_token}")
    
    if [[ -n "${data}" ]]; then
        curl_opts+=(-H "Content-Type: application/json" -d "${data}")
    fi
    
    curl "${curl_opts[@]}" "${url}"
}