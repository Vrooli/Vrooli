#!/usr/bin/env bash
################################################################################
# Unstructured.io Unit Test - v2.0 Contract Compliant
# 
# Library function validation for unstructured-io resource
# Must complete in <60s as per universal.yaml requirements
#
################################################################################

set -uo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source CLI framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Source resource libraries
for lib in common core install status api process cache-simple validate test content; do
    lib_file="${RESOURCE_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

log::header "Unstructured.io Unit Test"

# Track test results
tests_passed=0
tests_failed=0

# Test helper function
test_function_exists() {
    local func_name="$1"
    local description="${2:-Function $func_name}"
    
    if command -v "$func_name" &>/dev/null; then
        log::success "✓ $description exists"
        ((tests_passed++))
        return 0
    else
        log::error "✗ $description missing"
        ((tests_failed++))
        return 1
    fi
}

# Test helper for function execution
test_function_runs() {
    local func_name="$1"
    local description="${2:-Function $func_name}"
    local expected_exit="${3:-0}"
    
    if $func_name &>/dev/null; then
        if [[ $? -eq $expected_exit ]]; then
            log::success "✓ $description runs successfully"
            ((tests_passed++))
            return 0
        else
            log::error "✗ $description returned unexpected exit code"
            ((tests_failed++))
            return 1
        fi
    else
        log::error "✗ $description failed to execute"
        ((tests_failed++))
        return 1
    fi
}

# Test 1: Core lifecycle functions
log::info "Test 1: Core lifecycle functions..."
test_function_exists "unstructured_io::install" "Install function"
test_function_exists "unstructured_io::uninstall" "Uninstall function"
test_function_exists "unstructured_io::start" "Start function"
test_function_exists "unstructured_io::stop" "Stop function"
test_function_exists "unstructured_io::restart" "Restart function"

# Test 2: Status and health functions
log::info "Test 2: Status and health functions..."
test_function_exists "unstructured_io::status" "Status function"
test_function_exists "unstructured_io::check_health" "Health check function"
test_function_exists "unstructured_io::container_exists" "Container exists check"
test_function_exists "unstructured_io::container_running" "Container running check"

# Test 3: Content management functions
log::info "Test 3: Content management functions..."
test_function_exists "unstructured_io::content::add" "Content add function"
test_function_exists "unstructured_io::content::list" "Content list function"
test_function_exists "unstructured_io::content::get" "Content get function"
test_function_exists "unstructured_io::content::remove" "Content remove function"
test_function_exists "unstructured_io::process_document" "Document processing function"

# Test 4: API functions
log::info "Test 4: API functions..."
test_function_exists "unstructured_io::api::process_file" "API process file function"
test_function_exists "unstructured_io::api::get_supported_formats" "Get supported formats function"

# Test 5: Cache functions
log::info "Test 5: Cache functions..."
test_function_exists "unstructured_io::cache::get" "Cache get function"
test_function_exists "unstructured_io::cache::set" "Cache set function"
test_function_exists "unstructured_io::cache::clear" "Cache clear function"
test_function_exists "unstructured_io::cache::generate_key" "Cache key generation function"

# Test 6: Validation functions
log::info "Test 6: Validation functions..."
test_function_exists "unstructured_io::validate_installation" "Installation validation function"
test_function_exists "unstructured_io::validate::file_type" "File type validation function"
test_function_exists "unstructured_io::validate::file_size" "File size validation function"

# Test 7: Test default configuration values
log::info "Test 7: Configuration variables..."
if [[ -n "${UNSTRUCTURED_IO_PORT:-}" ]]; then
    log::success "✓ Port configured: $UNSTRUCTURED_IO_PORT"
    ((tests_passed++))
else
    log::error "✗ Port not configured"
    ((tests_failed++))
fi

if [[ -n "${UNSTRUCTURED_IO_CONTAINER_NAME:-}" ]]; then
    log::success "✓ Container name configured: $UNSTRUCTURED_IO_CONTAINER_NAME"
    ((tests_passed++))
else
    log::error "✗ Container name not configured"
    ((tests_failed++))
fi

if [[ -n "${UNSTRUCTURED_IO_BASE_URL:-}" ]]; then
    log::success "✓ Base URL configured: $UNSTRUCTURED_IO_BASE_URL"
    ((tests_passed++))
else
    log::error "✗ Base URL not configured"
    ((tests_failed++))
fi

# Test 8: CLI command handlers
log::info "Test 8: CLI command handlers..."
if [[ -n "${CLI_COMMAND_HANDLERS[manage::install]:-}" ]]; then
    log::success "✓ Install handler registered"
    ((tests_passed++))
else
    log::error "✗ Install handler not registered"
    ((tests_failed++))
fi

if [[ -n "${CLI_COMMAND_HANDLERS[test::smoke]:-}" ]]; then
    log::success "✓ Test smoke handler registered"
    ((tests_passed++))
else
    log::error "✗ Test smoke handler not registered"
    ((tests_failed++))
fi

# Summary
log::info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
log::info "Unit Test Summary:"
log::success "  Passed: $tests_passed"
if [[ $tests_failed -gt 0 ]]; then
    log::error "  Failed: $tests_failed"
else
    log::info "  Failed: 0"
fi

if [[ $tests_failed -eq 0 ]]; then
    log::success "All unit tests passed!"
    exit 0
else
    log::error "Unit tests failed"
    exit 1
fi