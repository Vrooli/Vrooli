#!/usr/bin/env bash
################################################################################
# Keycloak Multi-Realm Tenant Isolation Tests
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test configuration
TEST_TENANT_NAME="Test Tenant $(date +%s)"
TEST_TENANT_ID="test-tenant-$(date +%s)"
TEST_ADMIN_EMAIL="admin@testtenant.com"
TEST_ADMIN_PASSWORD="TestPass123!"

# Helper functions
cleanup_test_tenant() {
    log::info "Cleaning up test tenant..."
    "${RESOURCE_DIR}/cli.sh" realm delete-tenant "$TEST_TENANT_ID" &>/dev/null || true
}

# Main test execution
main() {
    log::info "Starting Keycloak multi-realm tenant isolation tests..."
    
    # Ensure Keycloak is running
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME:-vrooli-keycloak}"; then
        log::error "Keycloak container is not running"
        return 1
    fi
    
    # Test 1: Create tenant realm
    log::info "Testing tenant realm creation..."
    if "${RESOURCE_DIR}/cli.sh" realm create-tenant "$TEST_TENANT_NAME" "$TEST_ADMIN_EMAIL" "$TEST_ADMIN_PASSWORD"; then
        log::success "Tenant realm creation successful"
    else
        log::error "Failed to create tenant realm"
        return 1
    fi
    
    # Test 2: List tenants
    log::info "Testing tenant listing..."
    local tenant_list
    tenant_list=$("${RESOURCE_DIR}/cli.sh" realm list-tenants 2>&1)
    if echo "$tenant_list" | grep -q "$TEST_TENANT_NAME"; then
        log::success "Tenant appears in list"
    else
        log::error "Tenant not found in list"
        cleanup_test_tenant
        return 1
    fi
    
    # Test 3: Get tenant details
    log::info "Testing get tenant details..."
    local realm_id
    realm_id=$(echo "$TEST_TENANT_NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
    if "${RESOURCE_DIR}/cli.sh" realm get-tenant "$realm_id" &>/dev/null; then
        log::success "Retrieved tenant details successfully"
    else
        log::error "Failed to get tenant details"
        cleanup_test_tenant
        return 1
    fi
    
    # Test 4: Export tenant configuration
    log::info "Testing tenant export..."
    local export_file="/tmp/keycloak-test-export.json"
    if "${RESOURCE_DIR}/cli.sh" realm export-tenant "$realm_id" "$export_file"; then
        if [[ -f "$export_file" ]]; then
            log::success "Tenant export successful"
            rm -f "$export_file"
        else
            log::error "Export file not created"
            cleanup_test_tenant
            return 1
        fi
    else
        log::error "Failed to export tenant"
        cleanup_test_tenant
        return 1
    fi
    
    # Test 5: Delete tenant
    log::info "Testing tenant deletion..."
    if "${RESOURCE_DIR}/cli.sh" realm delete-tenant "$realm_id"; then
        log::success "Tenant deletion successful"
    else
        log::error "Failed to delete tenant"
        return 1
    fi
    
    # Verify deletion
    tenant_list=$("${RESOURCE_DIR}/cli.sh" realm list-tenants 2>&1)
    if ! echo "$tenant_list" | grep -q "$TEST_TENANT_NAME"; then
        log::success "Tenant successfully removed from list"
    else
        log::error "Tenant still appears in list after deletion"
        return 1
    fi
    
    log::success "All multi-realm tests passed"
    return 0
}

# Run tests
main "$@"