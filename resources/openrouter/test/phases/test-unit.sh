#!/usr/bin/env bash
################################################################################
# OpenRouter Unit Tests - Library Function Validation
# 
# Tests individual functions in the OpenRouter libraries
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
        ((TESTS_PASSED++)) || true
        return 0
    else
        echo "$(format::error "✗")"
        ((TESTS_FAILED++)) || true
        return 1
    fi
}

# Unit tests
main() {
    log::info "Running OpenRouter unit tests..."
    
    # Test 1: Config defaults are set
    run_test "OPENROUTER_API_BASE defined" "[[ -n '${OPENROUTER_API_BASE:-}' ]]"
    run_test "OPENROUTER_DEFAULT_MODEL defined" "[[ -n '${OPENROUTER_DEFAULT_MODEL:-}' ]]"
    run_test "OPENROUTER_TIMEOUT defined" "[[ -n '${OPENROUTER_TIMEOUT:-}' ]]"
    
    # Test 2: Library files can be sourced
    run_test "Source core.sh" "source '${RESOURCE_DIR}/lib/core.sh'"
    run_test "Source install.sh" "source '${RESOURCE_DIR}/lib/install.sh'"
    run_test "Source configure.sh" "source '${RESOURCE_DIR}/lib/configure.sh'"
    run_test "Source status.sh" "source '${RESOURCE_DIR}/lib/status.sh'"
    run_test "Source content.sh" "source '${RESOURCE_DIR}/lib/content.sh'"
    
    # Test 3: Core functions exist after sourcing
    source "${RESOURCE_DIR}/lib/core.sh" 2>/dev/null || true
    run_test "openrouter::init function exists" "type -t openrouter::init"
    run_test "openrouter::test_connection function exists" "type -t openrouter::test_connection"
    run_test "openrouter::list_models function exists" "type -t openrouter::list_models"
    
    # Test 4: Install functions exist
    source "${RESOURCE_DIR}/lib/install.sh" 2>/dev/null || true
    run_test "openrouter::install function exists" "type -t openrouter::install"
    run_test "openrouter::uninstall function exists" "type -t openrouter::uninstall"
    
    # Test 5: Status functions exist
    source "${RESOURCE_DIR}/lib/status.sh" 2>/dev/null || true
    run_test "openrouter::status function exists" "type -t openrouter::status"
    
    # Test 6: Content functions exist  
    source "${RESOURCE_DIR}/lib/content.sh" 2>/dev/null || true
    run_test "openrouter::content::add function exists" "type -t openrouter::content::add"
    run_test "openrouter::content::list function exists" "type -t openrouter::content::list"
    
    # Test 7: Cloudflare functions exist
    source "${RESOURCE_DIR}/lib/cloudflare.sh" 2>/dev/null || true
    run_test "openrouter::cloudflare::is_configured function exists" "type -t openrouter::cloudflare::is_configured"
    run_test "openrouter::cloudflare::configure function exists" "type -t openrouter::cloudflare::configure"
    run_test "openrouter::cloudflare::status function exists" "type -t openrouter::cloudflare::status"
    run_test "openrouter::cloudflare::cli function exists" "type -t openrouter::cloudflare::cli"
    
    # Summary
    echo
    log::info "Test Results: $(format::success "$TESTS_PASSED passed"), $(format::error "$TESTS_FAILED failed")"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        exit 1
    fi
    
    exit 0
}

main "$@"