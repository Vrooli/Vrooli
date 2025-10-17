#!/usr/bin/env bash

#######################################
# Test script for Browserless Adapter Pattern
# 
# Verifies that the adapter framework is working correctly
# and that adapters can be loaded and executed.
#######################################

set -euo pipefail

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TEST_DIR="${APP_ROOT}/resources/browserless/test"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${BROWSERLESS_DIR}/lib/common.sh"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

#######################################
# Run a test and track results
#######################################
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing $test_name... "
    
    if eval "$test_command" >/dev/null 2>&1; then
        echo "✅ PASSED"
        ((TESTS_PASSED++))
        return 0
    else
        echo "❌ FAILED"
        ((TESTS_FAILED++))
        return 1
    fi
}

#######################################
# Test adapter framework loading
#######################################
test_adapter_framework() {
    source "${BROWSERLESS_DIR}/adapters/common.sh"
    source "${BROWSERLESS_DIR}/adapters/registry.sh"
    
    # Check if key functions are available
    declare -f adapter::init >/dev/null && \
    declare -f adapter::list >/dev/null && \
    declare -f registry::init >/dev/null
}

#######################################
# Test n8n adapter loading
#######################################
test_n8n_adapter() {
    source "${BROWSERLESS_DIR}/adapters/n8n/api.sh"
    
    # Check if n8n functions are available
    declare -f n8n::init >/dev/null && \
    declare -f n8n::dispatch >/dev/null
}

#######################################
# Test vault adapter loading
#######################################
test_vault_adapter() {
    source "${BROWSERLESS_DIR}/adapters/vault/api.sh"
    
    # Check if vault functions are available
    declare -f vault::init >/dev/null && \
    declare -f vault::dispatch >/dev/null
}

#######################################
# Test CLI integration
#######################################
test_cli_integration() {
    source "${BROWSERLESS_DIR}/cli.sh"
    
    # Check if adapter command is registered
    declare -f browserless_adapter >/dev/null
}

#######################################
# Test adapter discovery
#######################################
test_adapter_discovery() {
    source "${BROWSERLESS_DIR}/adapters/common.sh"
    source "${BROWSERLESS_DIR}/adapters/registry.sh"
    
    # Initialize registry
    registry::init
    
    # Check if we can list adapters
    local adapters_dir="${BROWSERLESS_DIR}/adapters"
    [[ -d "$adapters_dir/n8n" ]] && [[ -d "$adapters_dir/vault" ]]
}

#######################################
# Test backward compatibility
#######################################
test_backward_compatibility() {
    source "${BROWSERLESS_DIR}/cli.sh"
    
    # Check if legacy command still exists
    declare -f browserless_execute_workflow_legacy >/dev/null
}

#######################################
# Main test execution
#######################################
main() {
    log::header "🧪 Testing Browserless Adapter Pattern"
    echo
    
    # Run tests
    run_test "Adapter Framework Loading" test_adapter_framework
    run_test "N8N Adapter Loading" test_n8n_adapter
    run_test "Vault Adapter Loading" test_vault_adapter
    run_test "CLI Integration" test_cli_integration
    run_test "Adapter Discovery" test_adapter_discovery
    run_test "Backward Compatibility" test_backward_compatibility
    
    echo
    echo "─────────────────────────────────"
    echo "Test Results:"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    echo "─────────────────────────────────"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "✅ All tests passed!"
        
        echo
        echo "You can now use the adapter pattern:"
        echo "  resource-browserless for n8n execute-workflow <workflow-id>"
        echo "  resource-browserless for vault add-secret <path> <key>=<value>"
        echo "  resource-browserless for --help"
        
        return 0
    else
        log::error "❌ Some tests failed"
        return 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi