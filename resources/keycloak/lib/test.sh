#!/usr/bin/env bash
# Keycloak test functions for v2.0 CLI

keycloak::test::smoke() {
    log::info "Running Keycloak smoke test..."
    
    # Check if Keycloak is running
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME:-vrooli-keycloak}"; then
        log::error "Keycloak container is not running"
        return 1
    fi
    
    # Check health endpoint - use OIDC discovery endpoint as health check
    local health_url="http://localhost:${KEYCLOAK_PORT:-8070}/realms/master/.well-known/openid-configuration"
    if timeout 5 curl -sf "${health_url}" > /dev/null 2>&1; then
        log::success "Keycloak health check passed"
        return 0
    else
        log::error "Keycloak health check failed"
        return 1
    fi
}

keycloak::logs() {
    local lines="${1:-50}"
    log::info "Showing last ${lines} lines of Keycloak logs..."
    docker logs "${KEYCLOAK_CONTAINER_NAME:-vrooli-keycloak}" --tail "${lines}" 2>&1
}

keycloak::credentials() {
    # Source defaults to get actual password
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${script_dir}/../config/defaults.sh"
    
    local password_display="$KEYCLOAK_ADMIN_PASSWORD"
    
    # Check if using default password and warn
    if [[ "$KEYCLOAK_ADMIN_PASSWORD" == "admin" ]]; then
        log::warning "Using DEFAULT admin password. Consider setting KEYCLOAK_ADMIN_PASSWORD for production."
    fi
    
    cat << EOF
Keycloak Admin Credentials:
===========================
URL:      http://localhost:${KEYCLOAK_PORT:-8070}
Admin UI: http://localhost:${KEYCLOAK_PORT:-8070}/admin
Username: ${KEYCLOAK_ADMIN_USER:-admin}
Password: ${password_display}

Note: For production use, set KEYCLOAK_ADMIN_PASSWORD environment variable
EOF
}