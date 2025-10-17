#!/usr/bin/env bash
################################################################################
# SQLite Resource - Test Library
#
# Test implementations for SQLite resource validation
################################################################################

set -euo pipefail

# Smoke test - quick health check
sqlite::test::smoke() {
    log::info "Running SQLite smoke test..."
    
    # Check if sqlite3 is available
    if ! command -v sqlite3 &> /dev/null; then
        log::error "SQLite3 not installed"
        return 1
    fi
    
    # Check version
    local version
    version=$(sqlite3 --version | awk '{print $1}')
    log::info "SQLite version: $version"
    
    # Create and test a temporary database
    local test_db="/tmp/sqlite_smoke_test_$$.db"
    
    # Create test database with sample data
    sqlite3 "$test_db" <<EOF
CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT);
INSERT INTO test (name) VALUES ('smoke_test');
SELECT COUNT(*) FROM test;
.quit
EOF
    
    local result=$?
    
    # Cleanup
    rm -f "$test_db"
    
    if [[ $result -eq 0 ]]; then
        log::info "Smoke test passed"
    else
        log::error "Smoke test failed"
    fi
    
    return $result
}

# Integration test - full functionality
sqlite::test::integration() {
    log::info "Running SQLite integration test..."
    
    local test_db="integration_test_$$"
    local errors=0
    
    # Test 1: Create database
    log::info "Test 1: Creating database..."
    if sqlite::content::create "$test_db"; then
        log::info "✓ Database creation successful"
    else
        log::error "✗ Database creation failed"
        ((errors++))
    fi
    
    # Test 2: Execute queries
    log::info "Test 2: Executing queries..."
    local db_path="${SQLITE_DATABASE_PATH}/${test_db}.db"
    
    # Create table
    if sqlite3 "$db_path" "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT UNIQUE);"; then
        log::info "✓ Table creation successful"
    else
        log::error "✗ Table creation failed"
        ((errors++))
    fi
    
    # Insert data
    if sqlite3 "$db_path" "INSERT INTO users (name, email) VALUES ('Test User', 'test@example.com');"; then
        log::info "✓ Data insertion successful"
    else
        log::error "✗ Data insertion failed"
        ((errors++))
    fi
    
    # Query data
    local count
    count=$(sqlite3 "$db_path" "SELECT COUNT(*) FROM users;")
    if [[ "$count" == "1" ]]; then
        log::info "✓ Data query successful"
    else
        log::error "✗ Data query failed (expected 1, got $count)"
        ((errors++))
    fi
    
    # Test 3: Backup database
    log::info "Test 3: Backing up database..."
    if sqlite::content::backup "$test_db"; then
        log::info "✓ Backup successful"
    else
        log::error "✗ Backup failed"
        ((errors++))
    fi
    
    # Test 4: List databases
    log::info "Test 4: Listing databases..."
    if sqlite::content::list | grep -q "${test_db}.db"; then
        log::info "✓ Database listing successful"
    else
        log::error "✗ Database listing failed"
        ((errors++))
    fi
    
    # Test 5: WAL mode
    log::info "Test 5: Testing WAL mode..."
    local journal_mode
    journal_mode=$(sqlite3 "$db_path" "PRAGMA journal_mode;")
    if [[ "$journal_mode" == "wal" ]]; then
        log::info "✓ WAL mode enabled"
    else
        log::error "✗ WAL mode not enabled (got: $journal_mode)"
        ((errors++))
    fi
    
    # Test 6: Concurrent access
    log::info "Test 6: Testing concurrent access..."
    # Use retry logic with busy timeout for concurrent inserts
    (
        for i in {1..5}; do
            # Retry up to 3 times with busy timeout set
            for attempt in {1..3}; do
                if sqlite3 "$db_path" "PRAGMA busy_timeout = 10000; INSERT INTO users (name, email) VALUES ('User$i', 'user$i@example.com');" 2>/dev/null; then
                    break
                fi
                sleep 0.1
            done
        done
    ) &
    (
        for i in {6..10}; do
            # Retry up to 3 times with busy timeout set
            for attempt in {1..3}; do
                if sqlite3 "$db_path" "PRAGMA busy_timeout = 10000; INSERT INTO users (name, email) VALUES ('User$i', 'user$i@example.com');" 2>/dev/null; then
                    break
                fi
                sleep 0.1
            done
        done
    ) &
    wait
    
    local final_count
    final_count=$(sqlite3 "$db_path" "SELECT COUNT(*) FROM users;")
    if [[ "$final_count" == "11" ]]; then
        log::info "✓ Concurrent access successful"
    else
        log::error "✗ Concurrent access failed (expected 11, got $final_count)"
        ((errors++))
    fi
    
    # Cleanup
    log::info "Cleaning up test database..."
    echo "y" | sqlite::content::remove "$test_db"
    
    if [[ $errors -eq 0 ]]; then
        log::info "Integration test passed"
        return 0
    else
        log::error "Integration test failed with $errors errors"
        return 1
    fi
}

# Unit test - test individual functions
sqlite::test::unit() {
    log::info "Running SQLite unit tests..."
    
    local errors=0
    
    # Test ensure_directories
    log::info "Testing ensure_directories..."
    if sqlite::ensure_directories; then
        if [[ -d "${SQLITE_DATABASE_PATH}" ]] && [[ -d "${SQLITE_BACKUP_PATH}" ]]; then
            log::info "✓ ensure_directories passed"
        else
            log::error "✗ ensure_directories failed - directories not created"
            ((errors++))
        fi
    else
        log::error "✗ ensure_directories failed"
        ((errors++))
    fi
    
    # Test status function
    log::info "Testing status function..."
    if sqlite::status > /dev/null; then
        log::info "✓ status function passed"
    else
        log::error "✗ status function failed"
        ((errors++))
    fi
    
    # Test info function
    log::info "Testing info function..."
    if sqlite::info > /dev/null; then
        log::info "✓ info function passed"
    else
        log::error "✗ info function failed"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        log::info "Unit tests passed"
        return 0
    else
        log::error "Unit tests failed with $errors errors"
        return 1
    fi
}

# Replication test - verify replication functionality
sqlite::test::replication() {
    log::info "Running SQLite replication test..."
    
    # Execute the replication test script
    local test_script="${SQLITE_CLI_DIR}/test/phases/test-replication.sh"
    
    if [[ -f "$test_script" ]]; then
        if bash "$test_script"; then
            log::info "Replication test passed"
            return 0
        else
            log::error "Replication test failed"
            return 1
        fi
    else
        log::warning "Replication test script not found"
        return 1
    fi
}

# Run all tests
sqlite::test::all() {
    log::info "Running all SQLite tests..."
    
    local errors=0
    
    if ! sqlite::test::smoke; then
        ((errors++))
    fi
    
    if ! sqlite::test::unit; then
        ((errors++))
    fi
    
    if ! sqlite::test::integration; then
        ((errors++))
    fi
    
    # Run replication test if available
    if [[ -f "${SQLITE_CLI_DIR}/test/phases/test-replication.sh" ]]; then
        if ! sqlite::test::replication; then
            ((errors++))
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        log::info "All tests passed"
        return 0
    else
        log::error "Tests failed with $errors test suite failures"
        return 1
    fi
}