#!/usr/bin/env bash
################################################################################
# Keycloak Smoke Test - Quick Health Validation
# Maximum execution time: 30 seconds
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

# Get port - use default if not set
KEYCLOAK_PORT="${KEYCLOAK_PORT:-8070}"

# Test functions
test_container_running() {
    log::info "Checking if Keycloak container is running..."
    
    if docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
        log::success "Container ${KEYCLOAK_CONTAINER_NAME} is running"
        return 0
    else
        log::error "Container ${KEYCLOAK_CONTAINER_NAME} is not running"
        return 1
    fi
}

test_health_endpoint() {
    log::info "Testing health endpoint..."
    
    # Keycloak doesn't have a dedicated /health endpoint, but the OIDC discovery endpoint
    # serves as a good health check since it requires core services to be running
    local health_url="http://localhost:${KEYCLOAK_PORT}/realms/master/.well-known/openid-configuration"
    
    # Use timeout as required by v2.0 contract
    if timeout 5 curl -sf "${health_url}" > /dev/null 2>&1; then
        log::success "Health endpoint responded successfully"
        return 0
    else
        log::error "Health endpoint failed to respond"
        return 1
    fi
}

test_ready_endpoint() {
    log::info "Testing readiness endpoint..."
    
    local ready_url="http://localhost:${KEYCLOAK_PORT}/health/ready"
    
    if timeout 5 curl -sf "${ready_url}" > /dev/null 2>&1; then
        log::success "Ready endpoint responded successfully"
        return 0
    else
        log::warning "Ready endpoint not available (may be normal for this version)"
        # Don't fail on ready endpoint as it may not be available in all versions
        return 0
    fi
}

test_admin_console() {
    log::info "Testing admin console availability..."
    
    local admin_url="http://localhost:${KEYCLOAK_PORT}/admin/"
    
    # Check if we get a redirect or 200 response
    local response_code=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "${admin_url}" 2>/dev/null || echo "000")
    
    if [[ "$response_code" == "200" ]] || [[ "$response_code" == "302" ]] || [[ "$response_code" == "303" ]]; then
        log::success "Admin console is accessible (HTTP ${response_code})"
        return 0
    else
        log::error "Admin console not accessible (HTTP ${response_code})"
        return 1
    fi
}

# Main test execution
main() {
    log::info "Starting Keycloak smoke tests..."
    
    local failed=0
    
    # Run tests
    test_container_running || ((failed++))
    test_health_endpoint || ((failed++))
    test_ready_endpoint || true  # Don't count as failure
    test_admin_console || ((failed++))
    
    # Report results
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed"
        return 0
    else
        log::error "${failed} smoke test(s) failed"
        return 1
    fi
}

# Execute main function
main