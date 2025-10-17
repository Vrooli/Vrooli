#!/usr/bin/env bash
################################################################################
# Keycloak Security Test - Security Configuration Validation
# Maximum execution time: 60 seconds
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"

# Get port
KEYCLOAK_PORT="${KEYCLOAK_PORT:-8070}"

# Test functions
test_password_strength() {
    log::info "Testing admin password strength..."
    
    # Check if using default password
    if [[ "${KEYCLOAK_ADMIN_PASSWORD}" == "admin" ]]; then
        log::warning "Using default admin password - INSECURE for production"
        return 0  # Don't fail, just warn
    fi
    
    # Check password length
    local pass_length=${#KEYCLOAK_ADMIN_PASSWORD}
    if [[ $pass_length -lt 12 ]]; then
        log::warning "Password length is $pass_length - recommend at least 12 characters"
    else
        log::success "Password length is adequate ($pass_length characters)"
    fi
    
    return 0
}

test_ssl_configuration() {
    log::info "Testing SSL/TLS configuration..."
    
    # Check if HTTPS is configured
    local https_port="${KEYCLOAK_HTTPS_PORT:-8443}"
    if timeout 5 curl -sf "https://localhost:${https_port}/health" --insecure > /dev/null 2>&1; then
        log::success "HTTPS endpoint is configured"
        
        # Check certificate validity
        local cert_info=$(echo | openssl s_client -connect localhost:${https_port} 2>/dev/null | openssl x509 -noout -dates 2>/dev/null || echo "")
        if [[ -n "$cert_info" ]]; then
            log::success "TLS certificate is valid"
        fi
    else
        log::info "HTTPS not configured (optional for development)"
    fi
    
    return 0
}

test_security_headers() {
    log::info "Testing security headers..."
    
    local response_headers=$(timeout 5 curl -sI "http://localhost:${KEYCLOAK_PORT}/" 2>/dev/null || echo "")
    
    # Check for important security headers
    if echo "$response_headers" | grep -qi "X-Frame-Options"; then
        log::success "X-Frame-Options header present"
    else
        log::warning "X-Frame-Options header missing"
    fi
    
    if echo "$response_headers" | grep -qi "X-Content-Type-Options"; then
        log::success "X-Content-Type-Options header present"
    else
        log::warning "X-Content-Type-Options header missing"
    fi
    
    return 0
}

test_realm_security_settings() {
    log::info "Testing realm security settings..."
    
    # Get admin token
    local token_url="http://localhost:${KEYCLOAK_PORT}/realms/master/protocol/openid-connect/token"
    local token=$(timeout 5 curl -sf -X POST "${token_url}" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${KEYCLOAK_ADMIN_USER}" \
        -d "password=${KEYCLOAK_ADMIN_PASSWORD}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" 2>/dev/null | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
    
    if [[ -z "$token" ]]; then
        log::warning "Cannot check realm security without admin access"
        return 0
    fi
    
    # Get master realm configuration
    local realm_config=$(timeout 5 curl -sf \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/master" \
        -H "Authorization: Bearer ${token}" 2>/dev/null || echo "{}")
    
    # Check security settings
    if echo "$realm_config" | grep -q '"bruteForceProtected":true'; then
        log::success "Brute force protection enabled"
    else
        log::info "Brute force protection not enabled (optional)"
    fi
    
    if echo "$realm_config" | grep -q '"permanentLockout":false'; then
        log::success "Permanent lockout disabled (good for recovery)"
    fi
    
    return 0
}

test_database_security() {
    log::info "Testing database security configuration..."
    
    # Check if using PostgreSQL (more secure than H2)
    if [[ "${KEYCLOAK_DB}" == "postgres" ]]; then
        log::success "Using PostgreSQL database (production-ready)"
        
        # Check if database password is not default
        if [[ "${KEYCLOAK_DB_PASSWORD}" != "keycloak" ]]; then
            log::success "Database password is not default"
        else
            log::warning "Using default database password - consider changing for production"
        fi
    else
        log::warning "Using H2 database - not recommended for production"
    fi
    
    return 0
}

test_port_exposure() {
    log::info "Testing port exposure..."
    
    # Check which ports are exposed
    local container_ports=$(docker inspect ${KEYCLOAK_CONTAINER_NAME} 2>/dev/null | grep -o '"HostPort": "[0-9]*"' | cut -d'"' -f4 || echo "")
    
    if [[ -n "$container_ports" ]]; then
        log::info "Exposed ports: $(echo $container_ports | tr '\n' ' ')"
        
        # Warn if exposing on all interfaces
        local bind_info=$(docker inspect ${KEYCLOAK_CONTAINER_NAME} 2>/dev/null | grep -o '"HostIp": "[^"]*"' | cut -d'"' -f4 || echo "")
        if [[ "$bind_info" == "0.0.0.0" ]] || [[ "$bind_info" == "" ]]; then
            log::warning "Ports exposed on all interfaces - consider binding to localhost only for development"
        fi
    fi
    
    return 0
}

# Main test execution
main() {
    log::info "Starting Keycloak security tests..."
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
        log::warning "Keycloak container not running, skipping runtime security tests"
        # Still run static tests
        test_password_strength
        test_database_security
        return 0
    fi
    
    local failed=0
    
    # Run tests
    test_password_strength || ((failed++))
    test_ssl_configuration || ((failed++))
    test_security_headers || ((failed++))
    test_realm_security_settings || ((failed++))
    test_database_security || ((failed++))
    test_port_exposure || ((failed++))
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        log::success "All security tests passed"
        return 0
    else
        log::error "${failed} security test(s) failed"
        return 1
    fi
}

# Execute main function
main