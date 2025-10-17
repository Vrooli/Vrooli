#!/usr/bin/env bash
################################################################################
# SQLite Resource - Replication Test
#
# Tests basic replication functionality
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQLITE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
APP_ROOT="$(cd "$SQLITE_DIR/../.." && pwd)"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${SQLITE_DIR}/config/defaults.sh"

# Set default data path if not defined
SQLITE_DATA_PATH="${SQLITE_DATA_PATH:-${VROOLI_DATA:-${HOME}/.vrooli/data}/sqlite}"

source "${SQLITE_DIR}/lib/core.sh"
source "${SQLITE_DIR}/lib/replication.sh"

# Test setup
TEST_DB="replication_test_$$.db"
REPLICA_DIR="/tmp/sqlite_replica_test_$$"
REPLICA_PATH="${REPLICA_DIR}/replica.db"

# Cleanup function
cleanup() {
    log::info "Cleaning up test files..."
    
    # Remove test database
    if [[ -f "${SQLITE_DATA_PATH}/databases/${TEST_DB}" ]]; then
        rm -f "${SQLITE_DATA_PATH}/databases/${TEST_DB}"
    fi
    
    # Remove replica directory
    if [[ -d "$REPLICA_DIR" ]]; then
        rm -rf "$REPLICA_DIR"
    fi
    
    # Remove test replica configuration
    sqlite::replication::remove_replica "$TEST_DB" "$REPLICA_PATH" 2>/dev/null || true
}

# Trap cleanup on exit
trap cleanup EXIT

# Run replication tests
log::info "Running SQLite replication test..."

# Test 1: Create test database
log::info "Test 1: Creating test database..."
sqlite::ensure_directories

# Create database and add test data
DB_PATH="${SQLITE_DATA_PATH}/databases/${TEST_DB}"
sqlite3 "$DB_PATH" <<EOF
CREATE TABLE test_table (
    id INTEGER PRIMARY KEY,
    name TEXT,
    value INTEGER
);

INSERT INTO test_table (name, value) VALUES ('test1', 100);
INSERT INTO test_table (name, value) VALUES ('test2', 200);
INSERT INTO test_table (name, value) VALUES ('test3', 300);
EOF

if [[ -f "$DB_PATH" ]]; then
    log::success "✓ Test database created"
else
    log::error "✗ Failed to create test database"
    exit 1
fi

# Test 2: Initialize replication
log::info "Test 2: Initializing replication..."
sqlite::replication::init
if [[ -d "${SQLITE_DATA_PATH}/replication" ]]; then
    log::success "✓ Replication initialized"
else
    log::error "✗ Failed to initialize replication"
    exit 1
fi

# Test 3: Add replica
log::info "Test 3: Adding replica..."
mkdir -p "$REPLICA_DIR"
sqlite::replication::add_replica "$TEST_DB" "$REPLICA_PATH" 30
log::success "✓ Replica added"

# Test 4: List replicas
log::info "Test 4: Listing replicas..."
output=$(sqlite::replication::list 2>&1)
if echo "$output" | grep -q "$TEST_DB"; then
    log::success "✓ Replica listed"
else
    log::error "✗ Replica not found in list"
    exit 1
fi

# Test 5: Sync database to replica
log::info "Test 5: Syncing to replica..."
sqlite::replication::sync "$TEST_DB"

if [[ -f "$REPLICA_PATH" ]]; then
    # Verify data was replicated
    count=$(sqlite3 "$REPLICA_PATH" "SELECT COUNT(*) FROM test_table;")
    if [[ "$count" == "3" ]]; then
        log::success "✓ Database synced successfully"
    else
        log::error "✗ Replica data mismatch (expected 3 rows, got $count)"
        exit 1
    fi
else
    log::error "✗ Replica file not created"
    exit 1
fi

# Test 6: Verify consistency
log::info "Test 6: Verifying consistency..."
if sqlite::replication::verify "$TEST_DB"; then
    log::success "✓ Replica consistency verified"
else
    log::error "✗ Replica consistency check failed"
    exit 1
fi

# Test 7: Modify source and re-sync
log::info "Test 7: Testing incremental sync..."
sqlite3 "$DB_PATH" "INSERT INTO test_table (name, value) VALUES ('test4', 400);"
sqlite::replication::sync "$TEST_DB"

# Verify new data in replica
new_count=$(sqlite3 "$REPLICA_PATH" "SELECT COUNT(*) FROM test_table;")
if [[ "$new_count" == "4" ]]; then
    log::success "✓ Incremental sync successful"
else
    log::error "✗ Incremental sync failed (expected 4 rows, got $new_count)"
    exit 1
fi

# Test 8: Toggle replica
log::info "Test 8: Disabling replica..."
sqlite::replication::toggle "$TEST_DB" "$REPLICA_PATH" false
log::success "✓ Replica disabled"

# Test 9: Remove replica
log::info "Test 9: Removing replica..."
sqlite::replication::remove_replica "$TEST_DB" "$REPLICA_PATH"
log::success "✓ Replica removed"

log::success "Replication test passed"
exit 0