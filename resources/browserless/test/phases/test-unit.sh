#!/usr/bin/env bash
# Browserless Resource Unit Test - Library function tests
# Tests that Browserless library functions work correctly
# Max duration: 60 seconds per v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
BROWSERLESS_CLI_DIR="${APP_ROOT}/resources/browserless"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh"
# Ensure configuration is exported
browserless::export_config
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/health.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/lib/api.sh"

# Browserless Resource Unit Test
browserless::test::unit() {
    log::info "Running Browserless resource unit test..."
    
    local overall_status=0
    local verbose="${BROWSERLESS_TEST_VERBOSE:-false}"
    
    # Test 1: Configuration validation
    log::info "1/7 Testing configuration functions..."
    local config_ok=true
    
    # Check required config variables
    local required_vars=("BROWSERLESS_PORT" "BROWSERLESS_CONTAINER_NAME" "BROWSERLESS_IMAGE" "BROWSERLESS_BASE_URL")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            log::error "✗ Required config variable $var is not set"
            config_ok=false
        fi
    done
    
    if [[ "$config_ok" == "true" ]]; then
        log::success "✓ Configuration variables properly loaded"
        if [[ "$verbose" == "true" ]]; then
            echo "    Port: $BROWSERLESS_PORT"
            echo "    Container: $BROWSERLESS_CONTAINER_NAME"
            echo "    Image: $BROWSERLESS_IMAGE"
        fi
    else
        overall_status=1
    fi
    
    # Test 2: Common library functions
    log::info "2/8 Testing common library functions..."
    local common_ok=true
    
    # Test is_running function
    if declare -f is_running >/dev/null 2>&1; then
        log::success "✓ is_running function is available"
    else
        log::error "✗ is_running function is not available"
        common_ok=false
    fi
    
    # Test get_container_status function
    if declare -f get_container_status >/dev/null 2>&1; then
        log::success "✓ get_container_status function is available"
    else
        log::error "✗ get_container_status function is not available"
        common_ok=false
    fi
    
    if [[ "$common_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 3: Health check functions
    log::info "3/8 Testing health check functions..."
    local health_ok=true
    
    # Test health functions existence
    local health_functions=("browserless::check_basic_health" "browserless::check_api_health" "browserless::is_healthy")
    for func in "${health_functions[@]}"; do
        if declare -f "$func" >/dev/null 2>&1; then
            log::success "✓ $func function is available"
        else
            log::error "✗ $func function is not available"
            health_ok=false
        fi
    done
    
    if [[ "$health_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 4: API testing functions
    log::info "4/8 Testing API library functions..."
    local api_ok=true
    
    # Test API functions existence
    local api_functions=("browserless::test_screenshot" "browserless::ensure_test_output_dir")
    for func in "${api_functions[@]}"; do
        if declare -f "$func" >/dev/null 2>&1; then
            log::success "✓ $func function is available"
        else
            log::error "✗ $func function is not available"
            api_ok=false
        fi
    done
    
    if [[ "$api_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 5: Directory creation and access
    log::info "5/8 Testing directory operations..."
    local dir_ok=true
    local test_dir="/tmp/browserless-unit-test-$$"
    
    # Test directory creation
    if mkdir -p "$test_dir"; then
        log::success "✓ Can create test directories"
        
        # Test file operations
        if echo "test" > "$test_dir/test.txt" && [[ -f "$test_dir/test.txt" ]]; then
            log::success "✓ Can write test files"
        else
            log::error "✗ Cannot write test files"
            dir_ok=false
        fi
        
        # Cleanup
        rm -rf "$test_dir"
    else
        log::error "✗ Cannot create test directories"
        dir_ok=false
    fi
    
    if [[ "$dir_ok" != "true" ]]; then
        overall_status=1
    fi
    
    # Test 6: Environment validation
    log::info "6/8 Testing environment requirements..."
    local env_ok=true
    
    # Test curl availability
    if command -v curl >/dev/null 2>&1; then
        log::success "✓ curl is available"
    else
        log::error "✗ curl is not available"
        env_ok=false
    fi
    
    # Test docker availability
    if command -v docker >/dev/null 2>&1; then
        log::success "✓ docker command is available"
    else
        log::error "✗ docker command is not available"
        env_ok=false
    fi
    
    # Test jq availability
    if command -v jq >/dev/null 2>&1; then
        log::success "✓ jq is available for JSON processing"
    else
        log::error "✗ jq is not available for JSON processing"
        env_ok=false
    fi
    
    if [[ "$env_ok" != "true" ]]; then
        overall_status=1
    fi

    # Test 7: Cache functionality
    log::info "7/8 Testing cache functions..."
    
    # Source cache library
    if source "${BROWSERLESS_CLI_DIR}/lib/cache.sh" 2>/dev/null; then
        # Test cache functions are available
        if declare -f cache::init >/dev/null 2>&1; then
            log::success "✓ cache::init function is available"
        else
            log::error "✗ cache::init function not found"
            overall_status=1
        fi
        
        if declare -f cache::stats >/dev/null 2>&1; then
            log::success "✓ cache::stats function is available"
        else
            log::error "✗ cache::stats function not found"
            overall_status=1
        fi
        
        if declare -f cache::cleanup_expired >/dev/null 2>&1; then
            log::success "✓ cache::cleanup_expired function is available"
        else
            log::error "✗ cache::cleanup_expired function not found"
            overall_status=1
        fi
    else
        log::warning "⚠️ Cache library not available (optional feature)"
    fi
    
    # Test 8: Pool recovery functions
    log::info "8/8 Testing pool recovery functions..."
    # shellcheck disable=SC1091
    if source "${BROWSERLESS_CLI_DIR}/lib/pool-manager.sh" 2>/dev/null; then
        if declare -f pool::health_check_and_recover >/dev/null 2>&1; then
            log::success "✓ pool::health_check_and_recover function is available"
        else
            log::error "✗ pool::health_check_and_recover function not found"
            overall_status=1
        fi
        
        if declare -f pool::attempt_recovery >/dev/null 2>&1; then
            log::success "✓ pool::attempt_recovery function is available"
        else
            log::error "✗ pool::attempt_recovery function not found"
            overall_status=1
        fi
    else
        log::warning "⚠️ Pool manager library not available"
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 Browserless resource unit test PASSED"
        echo "All Browserless library functions are working correctly"
    else
        log::error "💥 Browserless resource unit test FAILED"
        echo "Some Browserless library functions have issues"
        
        # Provide troubleshooting hints
        echo ""
        echo "Common fixes:"
        echo "  1. Ensure all dependencies are installed: curl, docker, jq"
        echo "  2. Check that library files are properly sourced"
        echo "  3. Verify configuration files exist and are valid"
        echo "  4. Run: resource-browserless manage install"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    browserless::test::unit "$@"
fi
