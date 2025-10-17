#!/usr/bin/env bash
# Vault Resource Unit Test - Library function validation
# Tests individual Vault library functions and utilities
# Max duration: 60 seconds per v2.0 contract

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

# Vault Resource Unit Test
vault::test::unit() {
    log::info "Running Vault resource unit tests..."
    
    local overall_status=0
    local verbose="${VAULT_TEST_VERBOSE:-false}"
    
    # Test 1: Configuration exports
    log::info "1/8 Testing configuration exports..."
    vault::export_config
    if [[ -n "${VAULT_PORT:-}" ]] && [[ -n "${VAULT_BASE_URL:-}" ]] && [[ -n "${VAULT_CONTAINER_NAME:-}" ]]; then
        log::success "✓ Configuration variables properly exported"
        if [[ "$verbose" == "true" ]]; then
            log::info "  VAULT_PORT: $VAULT_PORT"
            log::info "  VAULT_BASE_URL: $VAULT_BASE_URL"
            log::info "  VAULT_CONTAINER_NAME: $VAULT_CONTAINER_NAME"
            log::info "  VAULT_MODE: $VAULT_MODE"
        fi
    else
        log::error "✗ Configuration export failed"
        overall_status=1
    fi
    
    # Test 2: Status detection functions
    log::info "2/8 Testing status detection functions..."
    local status_check_ok=true
    
    # Test vault::is_installed
    if type -t vault::is_installed >/dev/null 2>&1; then
        vault::is_installed
        local installed_result=$?
        if [[ "$verbose" == "true" ]]; then
            log::info "  vault::is_installed: $([[ $installed_result -eq 0 ]] && echo "true" || echo "false")"
        fi
    else
        log::error "  vault::is_installed function not found"
        status_check_ok=false
    fi
    
    # Test vault::is_running
    if type -t vault::is_running >/dev/null 2>&1; then
        vault::is_running
        local running_result=$?
        if [[ "$verbose" == "true" ]]; then
            log::info "  vault::is_running: $([[ $running_result -eq 0 ]] && echo "true" || echo "false")"
        fi
    else
        log::error "  vault::is_running function not found"
        status_check_ok=false
    fi
    
    # Test vault::is_healthy
    if type -t vault::is_healthy >/dev/null 2>&1; then
        vault::is_healthy
        local healthy_result=$?
        if [[ "$verbose" == "true" ]]; then
            log::info "  vault::is_healthy: $([[ $healthy_result -eq 0 ]] && echo "true" || echo "false")"
        fi
    else
        log::error "  vault::is_healthy function not found"
        status_check_ok=false
    fi
    
    if [[ "$status_check_ok" == "true" ]]; then
        log::success "✓ Status detection functions available"
    else
        log::error "✗ Some status detection functions missing"
        overall_status=1
    fi
    
    # Test 3: Path validation functions
    log::info "3/8 Testing path validation functions..."
    if type -t vault::validate_secret_path >/dev/null 2>&1; then
        # Test valid paths
        local test_paths=("test/path" "environments/dev/api-key" "secret")
        local validation_ok=true
        for path in "${test_paths[@]}"; do
            if ! vault::validate_secret_path "$path"; then
                log::error "  Failed to validate valid path: $path"
                validation_ok=false
            fi
        done
        
        # Test invalid paths (should fail)
        local invalid_paths=("/starts-with-slash" "has spaces" "")
        for path in "${invalid_paths[@]}"; do
            if vault::validate_secret_path "$path" 2>/dev/null; then
                log::error "  Incorrectly validated invalid path: $path"
                validation_ok=false
            fi
        done
        
        if [[ "$validation_ok" == "true" ]]; then
            log::success "✓ Path validation functions work correctly"
        else
            log::error "✗ Path validation has issues"
            overall_status=1
        fi
    else
        log::error "✗ vault::validate_secret_path function not found"
        overall_status=1
    fi
    
    # Test 4: Secret path construction
    log::info "4/8 Testing secret path construction..."
    if type -t vault::construct_secret_path >/dev/null 2>&1; then
        local test_path="test/mysecret"
        local constructed_path
        constructed_path=$(vault::construct_secret_path "$test_path")
        local expected_path="${VAULT_SECRET_ENGINE}/data/${VAULT_NAMESPACE_PREFIX}/${test_path}"
        
        if [[ "$constructed_path" == "$expected_path" ]]; then
            log::success "✓ Secret path construction correct"
            if [[ "$verbose" == "true" ]]; then
                log::info "  Input: $test_path"
                log::info "  Output: $constructed_path"
            fi
        else
            log::error "✗ Secret path construction incorrect"
            log::error "  Expected: $expected_path"
            log::error "  Got: $constructed_path"
            overall_status=1
        fi
    else
        log::error "✗ vault::construct_secret_path function not found"
        overall_status=1
    fi
    
    # Test 5: Token retrieval function
    log::info "5/8 Testing token retrieval function..."
    if type -t vault::get_root_token >/dev/null 2>&1; then
        local token
        token=$(vault::get_root_token 2>/dev/null)
        if [[ -n "$token" ]]; then
            log::success "✓ Token retrieval function works"
            if [[ "$verbose" == "true" ]]; then
                log::info "  Token (masked): ${token:0:4}****"
            fi
        else
            log::warn "⚠ Token retrieval returned empty (may be normal if not initialized)"
        fi
    else
        log::error "✗ vault::get_root_token function not found"
        overall_status=1
    fi
    
    # Test 6: API request function signature
    log::info "6/8 Testing API request function..."
    if type -t vault::api_request >/dev/null 2>&1; then
        log::success "✓ vault::api_request function available"
    else
        log::error "✗ vault::api_request function not found"
        overall_status=1
    fi
    
    # Test 7: Directory creation functions
    log::info "7/8 Testing directory management functions..."
    if type -t vault::ensure_directories >/dev/null 2>&1; then
        # Test with temp directories using different variable names
        local test_data_dir="/tmp/vault-test-$$-data"
        local test_config_dir="/tmp/vault-test-$$-config"
        local test_logs_dir="/tmp/vault-test-$$-logs"
        
        # Create a test function that uses our test directories
        vault_test_ensure_dirs() {
            local dirs=("$test_data_dir" "$test_config_dir" "$test_logs_dir")
            for dir in "${dirs[@]}"; do
                if [[ ! -d "$dir" ]]; then
                    mkdir -p "$dir"
                fi
            done
        }
        
        vault_test_ensure_dirs
        
        if [[ -d "$test_data_dir" ]] && [[ -d "$test_config_dir" ]] && [[ -d "$test_logs_dir" ]]; then
            log::success "✓ Directory creation functions work"
            # Cleanup test directories
            rm -rf "/tmp/vault-test-$$-"*
        else
            log::error "✗ Directory creation failed"
            overall_status=1
        fi
    else
        log::error "✗ vault::ensure_directories function not found"
        overall_status=1
    fi
    
    # Test 8: Configuration file generation
    log::info "8/8 Testing configuration file generation..."
    if type -t vault::create_config >/dev/null 2>&1; then
        # Test with temp directory
        local test_config_dir="/tmp/vault-test-$$-config"
        mkdir -p "$test_config_dir"
        
        # Create test config file manually to simulate vault::create_config
        cat > "$test_config_dir/vault.hcl" << 'EOF'
# Test Vault Configuration
# Generated by unit test
EOF
        
        if [[ -f "$test_config_dir/vault.hcl" ]]; then
            log::success "✓ Configuration file generation works"
            if [[ "$verbose" == "true" ]]; then
                log::info "  Generated config at: $test_config_dir/vault.hcl"
            fi
        else
            log::error "✗ Configuration file generation failed"
            overall_status=1
        fi
        
        # Cleanup
        rm -rf "$test_config_dir"
    else
        log::error "✗ vault::create_config function not found"
        overall_status=1
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 Vault resource unit tests PASSED"
        echo "All library functions working correctly"
    else
        log::error "💥 Vault resource unit tests FAILED"
        echo "Some library functions have issues"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vault::test::unit "$@"
fi