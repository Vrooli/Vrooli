#!/usr/bin/env bash
# PostgreSQL Smoke Tests
# Quick health validation (<30s per v2.0 contract)

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

# Test 1: Container is running
test_container_running() {
    log::info "Testing: Container is running..."
    
    if postgres::common::is_running "main"; then
        test_pass "Container is running"
    else
        test_fail "Container is not running"
        return 1
    fi
}

# Test 2: Health check passes
test_health_check() {
    log::info "Testing: Health check..."
    
    if postgres::common::health_check "main"; then
        test_pass "Health check passed"
    else
        test_fail "Health check failed"
        return 1
    fi
}

# Test 3: Can connect to database
test_database_connection() {
    log::info "Testing: Database connection..."
    
    local instance_user=$(postgres::common::get_instance_config "main" "user" 2>/dev/null || echo "${POSTGRES_DEFAULT_USER}")
    local instance_password=$(postgres::common::get_instance_config "main" "password" 2>/dev/null || echo "${POSTGRES_DEFAULT_PASSWORD:-}")
    local instance_database=$(postgres::common::get_instance_config "main" "database" 2>/dev/null || echo "${POSTGRES_DEFAULT_DB}")
    
    if PGPASSWORD="${instance_password}" timeout 5 docker exec "vrooli-postgres-main" \
        psql -h localhost -U "${instance_user}" -d "${instance_database}" -c "SELECT version();" >/dev/null 2>&1; then
        test_pass "Database connection successful"
    else
        test_fail "Database connection failed"
        return 1
    fi
}

# Test 4: Port is accessible
test_port_accessible() {
    log::info "Testing: Port accessibility..."
    
    local port=$(postgres::common::get_instance_config "main" "port" 2>/dev/null || echo "${POSTGRES_DEFAULT_PORT}")
    
    if timeout 5 nc -zv localhost "$port" >/dev/null 2>&1; then
        test_pass "Port $port is accessible"
    else
        test_fail "Port $port is not accessible"
        return 1
    fi
}

# Main test execution
main() {
    log::header "PostgreSQL Smoke Tests"
    
    # Run tests
    test_container_running
    test_health_check
    test_database_connection
    test_port_accessible
    
    # Summary
    log::info ""
    log::info "Test Summary:"
    log::success "  Passed: $TESTS_PASSED"
    [[ $TESTS_FAILED -gt 0 ]] && log::error "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "Some smoke tests failed"
        return 1
    fi
}

# Only run main if script is executed directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi