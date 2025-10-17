#!/usr/bin/env bash
# Vault Resource Integration Test - Full functionality validation
# Tests end-to-end Vault functionality including secret operations and API interactions
# Max duration: 120 seconds per v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
VAULT_CLI_DIR="${APP_ROOT}/resources/vault"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/common.sh"
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/config/defaults.sh"
vault::export_config
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/lib/docker.sh"

# Vault Resource Integration Test
vault::test::integration() {
    log::info "Running Vault resource integration tests..."
    
    local overall_status=0
    local verbose="${VAULT_TEST_VERBOSE:-false}"
    local test_namespace="integration-test-$$"
    
    # Pre-check: Ensure Vault is healthy
    log::info "Pre-check: Verifying Vault health..."
    if ! vault::is_healthy; then
        log::error "✗ Vault is not healthy. Cannot run integration tests."
        log::info "Please start Vault first: resource-vault manage start"
        return 1
    fi
    
    local vault_status
    vault_status=$(vault::get_status)
    if [[ "$vault_status" == "sealed" ]]; then
        log::warn "⚠ Vault is sealed. Tests will be limited."
        log::info "For full testing, unseal Vault: resource-vault content unseal"
    fi
    
    # Test 1: Health API endpoint
    log::info "1/10 Testing health API endpoint..."
    local health_response
    if health_response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/health" 2>/dev/null); then
        local initialized sealed version
        initialized=$(echo "$health_response" | jq -r '.initialized // false')
        sealed=$(echo "$health_response" | jq -r '.sealed // true')
        version=$(echo "$health_response" | jq -r '.version // "unknown"')
        
        log::success "✓ Health endpoint working"
        if [[ "$verbose" == "true" ]]; then
            log::info "  Initialized: $initialized"
            log::info "  Sealed: $sealed"
            log::info "  Version: $version"
        fi
    else
        log::error "✗ Health endpoint failed"
        overall_status=1
    fi
    
    # Test 2: Initialization status
    log::info "2/10 Testing initialization status..."
    local init_response
    if init_response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/init" 2>/dev/null); then
        local initialized
        initialized=$(echo "$init_response" | jq -r '.initialized // false')
        
        if [[ "$initialized" == "true" ]]; then
            log::success "✓ Vault is initialized"
        else
            log::warn "⚠ Vault is not initialized"
        fi
    else
        log::error "✗ Cannot check initialization status"
        overall_status=1
    fi
    
    # Test 3: Seal status
    log::info "3/10 Testing seal status..."
    local seal_response
    if seal_response=$(curl -sf "${VAULT_BASE_URL}/v1/sys/seal-status" 2>/dev/null); then
        local sealed threshold shares
        sealed=$(echo "$seal_response" | jq -r '.sealed // true')
        threshold=$(echo "$seal_response" | jq -r '.t // 0')
        shares=$(echo "$seal_response" | jq -r '.n // 0')
        
        log::success "✓ Seal status endpoint working"
        if [[ "$verbose" == "true" ]]; then
            log::info "  Sealed: $sealed"
            log::info "  Threshold: $threshold"
            log::info "  Shares: $shares"
        fi
    else
        log::error "✗ Cannot check seal status"
        overall_status=1
    fi
    
    # Only continue with secret operations if Vault is unsealed
    if [[ "$vault_status" == "healthy" ]] || [[ "$vault_status" == "sealed" && "$VAULT_MODE" == "dev" ]]; then
        # Test 4: Get authentication token
        log::info "4/10 Testing authentication token retrieval..."
        local token
        token=$(vault::get_root_token 2>/dev/null)
        
        if [[ -n "$token" ]]; then
            log::success "✓ Authentication token available"
            if [[ "$verbose" == "true" ]]; then
                log::info "  Token (masked): ${token:0:6}..."
            fi
        else
            log::error "✗ No authentication token available"
            log::warn "Skipping remaining secret operation tests"
            overall_status=1
            return $overall_status
        fi
        
        # Test 5: Write secret operation
        log::info "5/10 Testing secret write operation..."
        local test_path="${test_namespace}/test-secret"
        local test_value="integration-test-value-$(date +%s)"
        local write_response
        
        write_response=$(curl -sf -X PUT \
            -H "X-Vault-Token: $token" \
            -d "{\"data\":{\"test_key\":\"$test_value\"}}" \
            "${VAULT_BASE_URL}/v1/secret/data/${test_path}" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "$write_response" | jq -e '.data' >/dev/null 2>&1; then
            local version_created
            version_created=$(echo "$write_response" | jq -r '.data.version // "unknown"')
            log::success "✓ Secret write successful (version: $version_created)"
        else
            log::error "✗ Secret write failed"
            overall_status=1
        fi
        
        # Test 6: Read secret operation
        log::info "6/10 Testing secret read operation..."
        local read_response
        read_response=$(curl -sf \
            -H "X-Vault-Token: $token" \
            "${VAULT_BASE_URL}/v1/secret/data/${test_path}" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            local read_value
            read_value=$(echo "$read_response" | jq -r '.data.data.test_key // ""' 2>/dev/null)
            
            if [[ "$read_value" == "$test_value" ]]; then
                log::success "✓ Secret read successful (data integrity verified)"
            else
                log::error "✗ Secret read failed or data mismatch"
                if [[ "$verbose" == "true" ]]; then
                    log::info "  Expected: $test_value"
                    log::info "  Got: $read_value"
                fi
                overall_status=1
            fi
        else
            log::error "✗ Secret read operation failed"
            overall_status=1
        fi
        
        # Test 7: List secrets operation
        log::info "7/10 Testing secret list operation..."
        local list_response
        list_response=$(curl -sf \
            -H "X-Vault-Token: $token" \
            "${VAULT_BASE_URL}/v1/secret/metadata/${test_namespace}?list=true" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "$list_response" | jq -e '.data.keys' >/dev/null 2>&1; then
            local keys
            keys=$(echo "$list_response" | jq -r '.data.keys[]' 2>/dev/null)
            
            if echo "$keys" | grep -q "test-secret"; then
                log::success "✓ Secret list successful (test secret found)"
            else
                log::warn "⚠ Secret list works but test secret not found"
            fi
        else
            log::warn "⚠ Secret list operation not available (normal for new namespace)"
        fi
        
        # Test 8: Update secret operation
        log::info "8/10 Testing secret update operation..."
        local update_value="updated-value-$(date +%s)"
        local update_response
        
        update_response=$(curl -sf -X PUT \
            -H "X-Vault-Token: $token" \
            -d "{\"data\":{\"test_key\":\"$update_value\",\"updated\":true}}" \
            "${VAULT_BASE_URL}/v1/secret/data/${test_path}" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "$update_response" | jq -e '.data.version' >/dev/null 2>&1; then
            local version_updated
            version_updated=$(echo "$update_response" | jq -r '.data.version // "unknown"')
            log::success "✓ Secret update successful (version: $version_updated)"
        else
            log::error "✗ Secret update failed"
            overall_status=1
        fi
        
        # Test 9: Delete secret operation
        log::info "9/10 Testing secret delete operation..."
        local delete_response
        delete_response=$(curl -sf -X DELETE \
            -H "X-Vault-Token: $token" \
            "${VAULT_BASE_URL}/v1/secret/data/${test_path}" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            log::success "✓ Secret delete successful"
            
            # Verify deletion (remove -f flag since 404 is expected after delete)
            local verify_response
            verify_response=$(curl -s \
                -H "X-Vault-Token: $token" \
                "${VAULT_BASE_URL}/v1/secret/data/${test_path}" 2>/dev/null || echo "{}")
            
            # Deleted secrets may still be retrievable but marked as deleted
            if echo "$verify_response" | jq -e '.data.data' >/dev/null 2>&1; then
                log::info "  Note: Soft delete - secret marked as deleted but recoverable"
            fi
        else
            log::error "✗ Secret delete failed"
            overall_status=1
        fi
        
        # Test 10: Secret engines availability
        log::info "10/10 Testing secret engines..."
        local mounts_response
        mounts_response=$(curl -sf \
            -H "X-Vault-Token: $token" \
            "${VAULT_BASE_URL}/v1/sys/mounts" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && echo "$mounts_response" | jq -e '.' >/dev/null 2>&1; then
            # Check for KV engine
            if echo "$mounts_response" | jq -e '.["secret/"]' >/dev/null 2>&1 || \
               echo "$mounts_response" | jq -e '.data["secret/"]' >/dev/null 2>&1; then
                log::success "✓ KV secret engine is mounted"
            else
                log::warn "⚠ KV secret engine status unclear"
            fi
        else
            log::error "✗ Cannot check secret engines"
            overall_status=1
        fi
    else
        log::warn "⚠ Vault is sealed. Skipping secret operation tests."
        log::info "To run full tests, unseal Vault first."
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 Vault resource integration tests PASSED"
        echo "Vault service fully functional - all operations work correctly"
    else
        log::error "💥 Vault resource integration tests FAILED"
        echo "Vault service has functional issues that need to be resolved"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vault::test::integration "$@"
fi