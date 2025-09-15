#!/usr/bin/env bash
################################################################################
# n8n Unit Test - Library Function Validation
# 
# Validates individual n8n library functions (60s max)
################################################################################

set -euo pipefail

# Get directory paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
N8N_DIR="${APP_ROOT}/resources/n8n"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

log::info "Starting n8n unit tests..."

# Test counter
tests_passed=0
tests_failed=0

# Helper function for test assertions
assert_equals() {
    local expected="$1"
    local actual="$2"
    local test_name="$3"
    
    if [[ "$expected" == "$actual" ]]; then
        log::success "✓ $test_name"
        tests_passed=$((tests_passed + 1))
    else
        log::error "✗ $test_name - Expected: '$expected', Got: '$actual'"
        tests_failed=$((tests_failed + 1))
    fi
}

assert_true() {
    local condition="$1"
    local test_name="$2"
    
    if eval "$condition"; then
        log::success "✓ $test_name"
        tests_passed=$((tests_passed + 1))
    else
        log::error "✗ $test_name - Condition failed: $condition"
        tests_failed=$((tests_failed + 1))
    fi
}

# Test 1: Configuration loading
log::info "Test 1: Configuration functions..."
if [[ -f "${N8N_DIR}/config/defaults.sh" ]]; then
    source "${N8N_DIR}/config/defaults.sh"
fi

assert_true "declare -f n8n::export_config > /dev/null 2>&1" "Configuration export function exists"
assert_true "[[ -n \${N8N_DEFAULT_PORT:-} ]]" "Default port is defined"

# Test 2: Core functions
log::info "Test 2: Core library functions..."
if [[ -f "${N8N_DIR}/lib/core.sh" ]]; then
    source "${N8N_DIR}/lib/core.sh"
    
    assert_true "declare -f n8n::install > /dev/null" "Install function exists"
    assert_true "declare -f n8n::start > /dev/null" "Start function exists"
    assert_true "declare -f n8n::stop > /dev/null" "Stop function exists"
    assert_true "declare -f n8n::status > /dev/null" "Status function exists"
else
    log::error "Core library not found"
    tests_failed=$((tests_failed + 1))
fi

# Test 3: Content management functions
log::info "Test 3: Content management functions..."
if [[ -f "${N8N_DIR}/lib/content.sh" ]]; then
    source "${N8N_DIR}/lib/content.sh"
    
    assert_true "declare -f n8n::content::add > /dev/null" "Content add function exists"
    assert_true "declare -f n8n::content::list > /dev/null" "Content list function exists"
    assert_true "declare -f n8n::content::get > /dev/null" "Content get function exists"
    assert_true "declare -f n8n::content::remove > /dev/null" "Content remove function exists"
    assert_true "declare -f n8n::content::execute > /dev/null" "Content execute function exists"
else
    log::error "Content library not found"
    tests_failed=$((tests_failed + 1))
fi

# Test 4: API functions
log::info "Test 4: API library functions..."
if [[ -f "${N8N_DIR}/lib/api.sh" ]]; then
    source "${N8N_DIR}/lib/api.sh"
    
    assert_true "declare -f n8n::api::request > /dev/null" "API request function exists"
    assert_true "declare -f n8n::api::list_workflows > /dev/null" "List workflows function exists"
    assert_true "declare -f n8n::api::activate_workflow > /dev/null" "Activate workflow function exists"
else
    log::warn "API library not found - skipping API tests"
fi

# Test 5: Health check functions
log::info "Test 5: Health check functions..."
if [[ -f "${N8N_DIR}/lib/health.sh" ]]; then
    source "${N8N_DIR}/lib/health.sh"
    
    assert_true "declare -f n8n::health::check > /dev/null" "Health check function exists"
    assert_true "declare -f n8n::health::wait_ready > /dev/null" "Wait ready function exists"
else
    log::warn "Health library not found - skipping health tests"
fi

# Test 6: Credential management functions
log::info "Test 6: Credential management functions..."
if [[ -f "${N8N_DIR}/lib/auto-credentials.sh" ]]; then
    source "${N8N_DIR}/lib/auto-credentials.sh"
    
    assert_true "declare -f n8n::auto_manage_credentials > /dev/null" "Auto credentials function exists"
    assert_true "declare -f n8n::list_discoverable_resources > /dev/null" "List discoverable function exists"
else
    log::warn "Auto-credentials library not found - skipping credential tests"
fi

# Test 7: Test functions
log::info "Test 7: Test orchestration functions..."
if [[ -f "${N8N_DIR}/lib/test.sh" ]]; then
    source "${N8N_DIR}/lib/test.sh"
    
    assert_true "declare -f n8n::test::all > /dev/null" "Test all function exists"
    assert_true "declare -f n8n::test::smoke > /dev/null" "Test smoke function exists"
    assert_true "declare -f n8n::test::integration > /dev/null" "Test integration function exists"
    assert_true "declare -f n8n::test::unit > /dev/null" "Test unit function exists"
else
    log::error "Test library not found"
    tests_failed=$((tests_failed + 1))
fi

# Summary
log::info "Unit test summary: $tests_passed passed, $tests_failed failed"

if [[ $tests_failed -gt 0 ]]; then
    log::error "Unit tests failed"
    exit 1
else
    log::success "All unit tests passed"
    exit 0
fi