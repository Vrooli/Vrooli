#!/usr/bin/env bash
################################################################################
# OBS Studio Unit Tests - Library Function Validation (<60s)
#
# Tests individual library functions and utilities
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Unit test implementation
main() {
    local start_time=$(date +%s)
    local failed=0
    
    log::header "🧪 OBS Studio Unit Tests"
    log::info "Testing library functions (<60s)"
    
    # Test 1: Configuration loading
    log::info "Test 1: Configuration loading"
    
    # Check defaults are loaded
    if [[ -n "${OBS_PORT:-}" ]]; then
        log::success "✅ Port configuration loaded"
    else
        log::error "❌ Port configuration not loaded"
        ((failed++))
    fi
    
    if [[ -n "${OBS_CONFIG_DIR:-}" ]]; then
        log::success "✅ Config directory set"
    else
        log::error "❌ Config directory not set"
        ((failed++))
    fi
    
    # Test 2: Runtime configuration validation
    log::info "Test 2: Runtime configuration"
    
    local runtime_file="${RESOURCE_DIR}/config/runtime.json"
    if [[ -f "$runtime_file" ]]; then
        # Validate JSON structure
        if command -v jq &>/dev/null; then
            if jq -e '.startup_order' "$runtime_file" &>/dev/null; then
                log::success "✅ Runtime config has startup_order"
            else
                log::error "❌ Runtime config missing startup_order"
                ((failed++))
            fi
            
            if jq -e '.dependencies' "$runtime_file" &>/dev/null; then
                log::success "✅ Runtime config has dependencies"
            else
                log::error "❌ Runtime config missing dependencies"
                ((failed++))
            fi
            
            if jq -e '.startup_timeout' "$runtime_file" &>/dev/null; then
                log::success "✅ Runtime config has startup_timeout"
            else
                log::error "❌ Runtime config missing startup_timeout"
                ((failed++))
            fi
        else
            log::warning "⚠️  jq not available, skipping JSON validation"
        fi
    else
        log::error "❌ Runtime configuration file missing"
        ((failed++))
    fi
    
    # Test 3: Schema validation
    log::info "Test 3: Schema validation"
    
    local schema_file="${RESOURCE_DIR}/config/schema.json"
    if [[ -f "$schema_file" ]]; then
        if command -v jq &>/dev/null; then
            if jq -e '."$schema"' "$schema_file" &>/dev/null; then
                log::success "✅ Schema file is valid JSON Schema"
            else
                log::error "❌ Schema file is not valid JSON Schema"
                ((failed++))
            fi
        fi
        log::success "✅ Schema file exists"
    else
        log::error "❌ Schema file missing"
        ((failed++))
    fi
    
    # Test 4: Port allocation
    log::info "Test 4: Port allocation"
    
    # Check port is within valid range
    local port="${OBS_PORT:-4455}"
    if [[ $port -ge 1024 ]] && [[ $port -le 65535 ]]; then
        log::success "✅ Port $port is in valid range"
    else
        log::error "❌ Port $port is out of valid range"
        ((failed++))
    fi
    
    # Test 5: Directory creation functions
    log::info "Test 5: Directory functions"
    
    local test_dir="/tmp/obs-test-$$"
    if mkdir -p "$test_dir"; then
        log::success "✅ Can create directories"
        
        if [[ -w "$test_dir" ]]; then
            log::success "✅ Directory is writable"
        else
            log::error "❌ Directory not writable"
            ((failed++))
        fi
        
        rm -rf "$test_dir"
    else
        log::error "❌ Cannot create directories"
        ((failed++))
    fi
    
    # Test 6: Mock mode detection
    log::info "Test 6: Mock mode detection"
    
    if [[ "${OBS_MOCK_MODE:-true}" == "true" ]]; then
        log::success "✅ Running in mock mode (testing environment)"
    else
        log::success "✅ Running in real mode (production environment)"
    fi
    
    # Test 7: Library function availability
    log::info "Test 7: Library functions"
    
    # Check core functions exist
    if type -t obs::is_installed &>/dev/null; then
        log::success "✅ obs::is_installed function available"
    else
        log::error "❌ obs::is_installed function missing"
        ((failed++))
    fi
    
    if type -t obs::is_running &>/dev/null; then
        log::success "✅ obs::is_running function available"
    else
        log::error "❌ obs::is_running function missing"
        ((failed++))
    fi
    
    if type -t obs::start &>/dev/null; then
        log::success "✅ obs::start function available"
    else
        log::error "❌ obs::start function missing"
        ((failed++))
    fi
    
    if type -t obs::stop &>/dev/null; then
        log::success "✅ obs::stop function available"
    else
        log::error "❌ obs::stop function missing"
        ((failed++))
    fi
    
    # Test 8: Error handling
    log::info "Test 8: Error handling"
    
    # Test with invalid input
    set +e  # Temporarily disable exit on error
    
    # This should fail gracefully
    obs::content::get "" 2>/dev/null
    if [[ $? -ne 0 ]]; then
        log::success "✅ Handles empty input correctly"
    else
        log::error "❌ Does not handle empty input"
        ((failed++))
    fi
    
    set -e  # Re-enable exit on error
    
    # Test 9: Test function availability
    log::info "Test 9: Test functions"
    
    if type -t obs::test::smoke &>/dev/null; then
        log::success "✅ Smoke test function available"
    else
        log::error "❌ Smoke test function missing"
        ((failed++))
    fi
    
    if type -t obs::test::integration &>/dev/null; then
        log::success "✅ Integration test function available"
    else
        log::error "❌ Integration test function missing"
        ((failed++))
    fi
    
    if type -t obs::test::unit &>/dev/null; then
        log::success "✅ Unit test function available"
    else
        log::error "❌ Unit test function missing"
        ((failed++))
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Check time limit
    if [[ $duration -gt 60 ]]; then
        log::warning "⚠️  Unit tests took ${duration}s (exceeds 60s limit)"
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed (${duration}s)"
        exit 0
    else
        log::error "$failed unit tests failed (${duration}s)"
        exit 1
    fi
}

main "$@"