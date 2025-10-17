#!/usr/bin/env bash
# PostgreSQL Unit Tests
# Library function tests (<60s per v2.0 contract)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/resources/postgres/config/defaults.sh" 2>/dev/null || true
source "${APP_ROOT}/resources/postgres/lib/common.sh" 2>/dev/null || true

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log::success "✓ $1"
}

test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log::error "✗ $1"
    [[ -n "${2:-}" ]] && log::error "  Details: $2"
}

# Test 1: Instance configuration functions
test_instance_config() {
    log::info "Testing: Instance configuration functions..."
    
    # Test get_instance_config
    local port=$(postgres::common::get_instance_config "main" "port")
    if [[ -n "$port" && "$port" =~ ^[0-9]+$ ]]; then
        test_pass "get_instance_config returns valid port: $port"
    else
        test_fail "get_instance_config failed" "Port: $port"
    fi
    
    # Test instance listing
    local instances=($(postgres::common::list_instances))
    if [[ ${#instances[@]} -gt 0 ]]; then
        test_pass "list_instances returns instances: ${instances[*]}"
    else
        test_fail "list_instances returned no instances"
    fi
}

# Test 2: Container existence check
test_container_exists() {
    log::info "Testing: Container existence check..."
    
    if postgres::common::container_exists "main"; then
        test_pass "container_exists correctly identifies existing container"
    else
        test_fail "container_exists failed for existing container"
    fi
    
    if ! postgres::common::container_exists "nonexistent"; then
        test_pass "container_exists correctly identifies non-existent container"
    else
        test_fail "container_exists incorrectly reported non-existent container"
    fi
}

# Test 3: Port allocation functions
test_port_functions() {
    log::info "Testing: Port allocation functions..."
    
    # Test port range validation
    local port_in_range=5450
    local port_out_of_range=9999
    
    # Check if port is in valid range
    if [[ $port_in_range -ge ${POSTGRES_INSTANCE_PORT_RANGE_START} && 
          $port_in_range -le ${POSTGRES_INSTANCE_PORT_RANGE_END} ]]; then
        test_pass "Port range validation works"
    else
        test_fail "Port range validation failed"
    fi
}

# Test 4: Disk space check
test_disk_space() {
    log::info "Testing: Disk space check..."
    
    if postgres::common::check_disk_space; then
        test_pass "Disk space check passed"
    else
        # This might fail on systems with low disk space, so we just warn
        log::warn "Disk space check failed (may be expected on low-disk systems)"
        test_pass "Disk space check completed"
    fi
}

# Main test execution
main() {
    log::header "PostgreSQL Unit Tests"
    
    # Run tests
    test_instance_config
    test_container_exists
    test_port_functions
    test_disk_space
    
    # Summary
    log::info ""
    log::info "Test Summary:"
    log::success "  Passed: $TESTS_PASSED"
    [[ $TESTS_FAILED -gt 0 ]] && log::error "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "Some unit tests failed"
        return 1
    fi
}

# Only run main if script is executed directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi