#!/usr/bin/env bash
# PostgreSQL Integration Tests
# End-to-end functionality tests (<120s per v2.0 contract)

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

# Test 1: Create and query table
test_create_table() {
    log::info "Testing: Create and query table..."
    
    local instance_user=$(postgres::common::get_instance_config "main" "user" 2>/dev/null || echo "${POSTGRES_DEFAULT_USER}")
    local instance_password=$(postgres::common::get_instance_config "main" "password" 2>/dev/null || echo "${POSTGRES_DEFAULT_PASSWORD:-}")
    local instance_database=$(postgres::common::get_instance_config "main" "database" 2>/dev/null || echo "${POSTGRES_DEFAULT_DB}")
    
    # Drop table if exists to ensure clean test
    PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
        psql -h localhost -U "${instance_user}" -d "${instance_database}" \
        -c "DROP TABLE IF EXISTS test_table;" >/dev/null 2>&1
    
    # Create test table
    if PGPASSWORD="${instance_password}" timeout 10 docker exec "vrooli-postgres-main" \
        psql -h localhost -U "${instance_user}" -d "${instance_database}" \
        -c "CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(100), created_at TIMESTAMP DEFAULT NOW());" >/dev/null 2>&1; then
        
        # Insert test data
        if PGPASSWORD="${instance_password}" timeout 10 docker exec "vrooli-postgres-main" \
            psql -h localhost -U "${instance_user}" -d "${instance_database}" \
            -c "INSERT INTO test_table (name) VALUES ('test_entry_1'), ('test_entry_2');" >/dev/null 2>&1; then
            
            # Query data
            local count=$(PGPASSWORD="${instance_password}" timeout 10 docker exec "vrooli-postgres-main" \
                psql -h localhost -U "${instance_user}" -d "${instance_database}" -t \
                -c "SELECT COUNT(*) FROM test_table;" 2>/dev/null | tr -d ' ')
            
            if [[ "$count" == "2" ]]; then
                test_pass "Table creation and querying works"
                
                # Cleanup
                PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
                    psql -h localhost -U "${instance_user}" -d "${instance_database}" \
                    -c "DROP TABLE test_table;" >/dev/null 2>&1
            else
                test_fail "Query returned unexpected count: $count"
            fi
        else
            test_fail "Failed to insert data"
        fi
    else
        test_fail "Failed to create table"
    fi
}

# Test 2: Database backup and restore
test_backup_restore() {
    log::info "Testing: Backup and restore functionality..."
    
    local instance_user=$(postgres::common::get_instance_config "main" "user" 2>/dev/null || echo "${POSTGRES_DEFAULT_USER}")
    local instance_password=$(postgres::common::get_instance_config "main" "password" 2>/dev/null || echo "${POSTGRES_DEFAULT_PASSWORD:-}")
    local instance_database=$(postgres::common::get_instance_config "main" "database" 2>/dev/null || echo "${POSTGRES_DEFAULT_DB}")
    local backup_file="/tmp/postgres_test_backup_$$.sql"
    
    # Create test data
    PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
        psql -h localhost -U "${instance_user}" -d "${instance_database}" \
        -c "CREATE TABLE IF NOT EXISTS backup_test (id INT PRIMARY KEY, data TEXT);" >/dev/null 2>&1
    
    PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
        psql -h localhost -U "${instance_user}" -d "${instance_database}" \
        -c "INSERT INTO backup_test VALUES (1, 'test_data') ON CONFLICT DO NOTHING;" >/dev/null 2>&1
    
    # Create backup
    if PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
        pg_dump -h localhost -U "${instance_user}" -d "${instance_database}" > "$backup_file" 2>/dev/null; then
        
        # Drop table
        PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
            psql -h localhost -U "${instance_user}" -d "${instance_database}" \
            -c "DROP TABLE IF EXISTS backup_test;" >/dev/null 2>&1
        
        # Restore from backup
        if docker exec -i "vrooli-postgres-main" \
            psql -h localhost -U "${instance_user}" -d "${instance_database}" < "$backup_file" 2>/dev/null; then
            
            # Verify restored data
            local data=$(PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
                psql -h localhost -U "${instance_user}" -d "${instance_database}" -t \
                -c "SELECT data FROM backup_test WHERE id = 1;" 2>/dev/null | tr -d ' ')
            
            if [[ "$data" == "test_data" ]]; then
                test_pass "Backup and restore works"
            else
                test_fail "Restored data doesn't match: $data"
            fi
        else
            test_fail "Failed to restore backup"
        fi
        
        # Cleanup
        rm -f "$backup_file"
        PGPASSWORD="${instance_password}" docker exec "vrooli-postgres-main" \
            psql -h localhost -U "${instance_user}" -d "${instance_database}" \
            -c "DROP TABLE IF EXISTS backup_test;" >/dev/null 2>&1
    else
        test_fail "Failed to create backup"
    fi
}

# Test 3: Multiple connections
test_concurrent_connections() {
    log::info "Testing: Concurrent connections..."
    
    local instance_user=$(postgres::common::get_instance_config "main" "user" 2>/dev/null || echo "${POSTGRES_DEFAULT_USER}")
    local instance_password=$(postgres::common::get_instance_config "main" "password" 2>/dev/null || echo "${POSTGRES_DEFAULT_PASSWORD:-}")
    local instance_database=$(postgres::common::get_instance_config "main" "database" 2>/dev/null || echo "${POSTGRES_DEFAULT_DB}")
    
    # Start multiple concurrent connections
    local pids=()
    for i in {1..5}; do
        (PGPASSWORD="${instance_password}" timeout 5 docker exec "vrooli-postgres-main" \
            psql -h localhost -U "${instance_user}" -d "${instance_database}" \
            -c "SELECT pg_sleep(1);" >/dev/null 2>&1) &
        pids+=($!)
    done
    
    # Wait for all connections to complete
    local failed=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        test_pass "Concurrent connections handled successfully"
    else
        test_fail "$failed concurrent connections failed"
    fi
}

# Main test execution
main() {
    log::header "PostgreSQL Integration Tests"
    
    # Run tests
    test_create_table
    test_backup_restore
    test_concurrent_connections
    
    # Summary
    log::info ""
    log::info "Test Summary:"
    log::success "  Passed: $TESTS_PASSED"
    [[ $TESTS_FAILED -gt 0 ]] && log::error "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "Some integration tests failed"
        return 1
    fi
}

# Only run main if script is executed directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi