#!/usr/bin/env bash
################################################################################
# OpenRouter Integration Tests - End-to-End Functionality
# 
# Tests OpenRouter integration with the Vrooli system
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$SCRIPT_DIR"
TEST_DIR="$(dirname "$PHASES_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Define format functions if not available
if ! command -v format::success &>/dev/null; then
    format::success() { echo -e "\033[32m$*\033[0m"; }
    format::error() { echo -e "\033[31m$*\033[0m"; }
fi

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "Testing $test_name... "
    
    if eval "$test_cmd" &>/dev/null; then
        echo "$(format::success "✓")"
        ((TESTS_PASSED++))
    else
        echo "$(format::error "✗")"
        ((TESTS_FAILED++))
    fi
}

# Integration tests
main() {
    log::info "Running OpenRouter integration tests..."
    
    # Test 1: Install command
    run_test "Install command" "resource-openrouter manage install --force"
    
    # Test 2: Info command shows runtime config
    run_test "Info command" "resource-openrouter info | grep -q 'startup_order'"
    
    # Test 3: Content management - list
    run_test "Content list" "resource-openrouter content list"
    
    # Test 4: Test CLI framework integration
    run_test "CLI framework v2.0" "resource-openrouter help | grep -q 'manage'"
    
    # Test 5: Configuration commands
    run_test "Show config" "resource-openrouter show-config"
    
    # Skip API-dependent tests if no key configured
    if [[ -n "${OPENROUTER_API_KEY:-}" ]] || openrouter::init 2>/dev/null; then
        # Test 6: List models
        run_test "List models" "resource-openrouter content models | head -5"
        
        # Test 7: Test connection
        run_test "Test connection" "resource-openrouter test smoke"
    else
        echo "Skipping API-dependent tests (no API key configured)"
    fi
    
    # Test 8: Manage group commands
    run_test "Manage start (noop)" "resource-openrouter manage start"
    run_test "Manage stop (noop)" "resource-openrouter manage stop"
    run_test "Manage restart (noop)" "resource-openrouter manage restart"
    
    # Summary
    echo
    log::info "Test Results: $(format::success "$TESTS_PASSED passed"), $(format::error "$TESTS_FAILED failed")"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        exit 1
    fi
    
    exit 0
}

main "$@"