#!/usr/bin/env bash
# K6 Resource Unit Test - Library function validation
# Tests individual K6 library functions work correctly

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
K6_CLI_DIR="${APP_ROOT}/resources/k6"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091  
source "${K6_CLI_DIR}/lib/core.sh"
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/lib/docker.sh"

# K6 Resource Unit Test
k6::test::unit() {
    log::info "Running K6 resource unit tests..."
    
    local overall_status=0
    local tests_run=0
    local tests_passed=0
    
    # Helper function for unit tests
    test_function() {
        local test_name="$1"
        local test_func="$2"
        shift 2
        
        tests_run=$((tests_run + 1))
        log::info "Testing: $test_name"
        
        if $test_func "$@" >/dev/null 2>&1; then
            log::success "✓ $test_name"
            tests_passed=$((tests_passed + 1))
        else
            log::error "✗ $test_name"
            overall_status=1
        fi
    }
    
    # Test configuration functions
    test_function "k6::core::init function" k6::core::init
    test_function "k6::export_config function" k6::export_config
    
    # Test Docker status (should work if container exists)
    if docker ps -a | grep -q "vrooli-k6"; then
        test_function "k6::docker::status function" k6::docker::status
    else
        log::warn "K6 container not found - skipping Docker tests"
    fi
    
    # Test directory creation
    test_function "Directory creation" test -d "$K6_DATA_DIR"
    test_function "Scripts directory" test -d "$K6_SCRIPTS_DIR"  
    test_function "Results directory" test -d "$K6_RESULTS_DIR"
    
    # Test configuration validation
    test_function "K6_NAME variable set" test -n "$K6_NAME"
    test_function "K6_IMAGE variable set" test -n "$K6_IMAGE"
    test_function "K6_PORT variable set" test -n "$K6_PORT"
    
    # Test CLI framework integration
    if declare -f cli::dispatch >/dev/null 2>&1; then
        log::success "✓ CLI framework integration available"
        tests_passed=$((tests_passed + 1))
    else
        log::error "✗ CLI framework integration missing"
        overall_status=1
    fi
    tests_run=$((tests_run + 1))
    
    # Test library sourcing
    local required_functions=(
        "k6::core::init"
        "k6::docker::status"
        "k6::content::add"
        "k6::content::list"
        "k6::content::execute"
    )
    
    for func in "${required_functions[@]}"; do
        test_function "Function $func exists" declare -f "$func"
    done
    
    echo ""
    log::info "Unit Test Summary:"
    log::info "Tests Run: $tests_run"
    log::info "Tests Passed: $tests_passed"
    log::info "Tests Failed: $((tests_run - tests_passed))"
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 K6 resource unit tests PASSED"
        echo "All K6 library functions working correctly"
    else
        log::error "💥 K6 resource unit tests FAILED"
        echo "Some K6 library functions have issues"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    k6::test::unit "$@"
fi