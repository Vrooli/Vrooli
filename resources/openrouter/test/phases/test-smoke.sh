#!/usr/bin/env bash
################################################################################
# OpenRouter Smoke Tests - Quick Health Validation
# 
# Must complete in <30 seconds per v2.0 contract
################################################################################

set -uo pipefail

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

# Smoke tests
main() {
    log::info "Running OpenRouter smoke tests..."
    
    # Test 1: Check if CLI is accessible
    run_test "CLI accessibility" "command -v resource-openrouter"
    
    # Test 2: Check help command
    run_test "Help command" "resource-openrouter help >/dev/null 2>&1"
    
    # Test 3: Check status command
    run_test "Status command" "resource-openrouter status >/dev/null 2>&1"
    
    # Test 4: Check configuration files exist
    run_test "Config files exist" "[[ -f '${RESOURCE_DIR}/config/defaults.sh' ]]"
    
    # Test 5: Check runtime.json exists
    run_test "Runtime config exists" "[[ -f '${RESOURCE_DIR}/config/runtime.json' ]]"
    
    # Test 6: Check API connectivity (if API key is configured)
    if [[ -n "${OPENROUTER_API_KEY:-}" ]] || openrouter::init 2>/dev/null; then
        run_test "API connectivity" "timeout 5 curl -sf -H 'Authorization: Bearer $OPENROUTER_API_KEY' '${OPENROUTER_API_BASE}/models' >/dev/null"
    else
        echo "Skipping API connectivity test (no API key configured)"
    fi
    
    # Test 7: Check lib files exist
    run_test "Core library exists" "[[ -f '${RESOURCE_DIR}/lib/core.sh' ]]"
    run_test "Test library exists" "[[ -f '${RESOURCE_DIR}/lib/test.sh' ]]"
    
    # Summary
    echo
    log::info "Test Results: $(format::success "$TESTS_PASSED passed"), $(format::error "$TESTS_FAILED failed")"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        exit 1
    fi
    
    exit 0
}

main "$@"