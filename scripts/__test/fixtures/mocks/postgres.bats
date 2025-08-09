#!/usr/bin/env bats
# PostgreSQL Mock Test Suite
#
# Comprehensive tests for the PostgreSQL mock implementation
# Tests all PostgreSQL commands, state management, error injection,
# and BATS compatibility features

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/postgres-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure PostgreSQL mock state directory
    export POSTGRES_MOCK_STATE_DIR="$TEST_DIR/postgres-state"
    mkdir -p "$POSTGRES_MOCK_STATE_DIR"
    
    # Source the PostgreSQL mock
    source "$MOCK_DIR/postgres.sh"
    
    # Reset PostgreSQL mock to clean state
    mock::postgres::reset
}

teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Helper functions for assertions
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got status $status" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_output() {
    local expected="$1"
    if [[ "$1" == "--partial" ]]; then
        expected="$2"
        if [[ ! "$output" =~ "$expected" ]]; then
            echo "Expected output to contain: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    elif [[ "$1" == "--regexp" ]]; then
        expected="$2"
        if [[ ! "$output" =~ $expected ]]; then
            echo "Expected output to match: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    else
        if [[ "$output" != "$expected" ]]; then
            echo "Expected: $expected" >&2
            echo "Actual: $output" >&2
            return 1
        fi
    fi
}

refute_output() {
    local pattern="$2"
    if [[ "$1" == "--partial" ]] || [[ "$1" == "--regexp" ]]; then
        if [[ "$output" =~ "$pattern" ]]; then
            echo "Expected output NOT to contain: $pattern" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    fi
}

assert_line() {
    local index expected
    if [[ "$1" == "--index" ]]; then
        index="$2"
        expected="$3"
        local lines=()
        while IFS= read -r line; do
            lines+=("$line")
        done <<< "$output"
        
        if [[ "${lines[$index]}" != "$expected" ]]; then
            echo "Line $index mismatch" >&2
            echo "Expected: $expected" >&2
            echo "Actual: ${lines[$index]}" >&2
            return 1
        fi
    fi
}

# =============================================================================
# Basic psql connectivity tests
# =============================================================================

@test "psql: version command" {
    run psql --version
    assert_success
    assert_output --regexp "psql \(PostgreSQL\) [0-9]+\.[0-9]+"
}

@test "psql: help command" {
    run psql --help
    assert_success
    assert_output --partial "psql is the PostgreSQL interactive terminal"
    assert_output --partial "Usage:"
    assert_output --partial "Connection options:"
}

@test "psql: basic version query" {
    run psql -c "SELECT version();"
    assert_success
    assert_output --partial "PostgreSQL"
    assert_output --partial "(1 row)"
}

@test "psql: connection refused when disconnected" {
    mock::postgres::set_connected "false"
    
    run psql -c "SELECT 1;"
    assert_failure
    assert_output --partial "Connection refused"
}

@test "psql: error injection - connection timeout" {
    mock::postgres::set_error "connection_timeout"
    
    run psql -c "SELECT 1;"
    assert_failure
    assert_output --partial "connection to server timed out"
}

@test "psql: error injection - auth failed" {
    mock::postgres::set_error "auth_failed"
    
    run psql -c "SELECT 1;"
    assert_failure
    assert_output --partial "password authentication failed"
}

@test "psql: error injection - database not exist" {
    mock::postgres::set_error "database_not_exist"
    
    run psql -c "SELECT 1;"
    assert_failure
    assert_output --partial "database \"testdb\" does not exist"
}

@test "psql: error injection - too many connections" {
    mock::postgres::set_error "too_many_connections"
    
    run psql -c "SELECT 1;"
    assert_failure
    assert_output --partial "too many clients already"
}

@test "psql: interactive mode not supported" {
    run psql
    assert_failure
    assert_output --partial "Interactive mode not supported in tests"
}

# =============================================================================
# SQL command execution tests
# =============================================================================

@test "psql: simple SELECT query" {
    run psql -c "SELECT 1;"
    assert_success
    assert_output --partial "?column?"
    assert_output --partial "1"
    assert_output --partial "(1 row)"
}

@test "psql: tuples-only mode" {
    run psql -t -c "SELECT 1;"
    assert_success
    assert_output "1"
}

@test "psql: quiet mode" {
    run psql -q -c "INSERT INTO test VALUES (1);"
    assert_success
    # Should not contain extra output in quiet mode
    refute_output --partial "Command executed successfully"
}

@test "psql: list databases command" {
    run psql -c "\\l"
    assert_success
    assert_output --partial "List of databases"
    assert_output --partial "postgres"
    assert_output --partial "template0"
    assert_output --partial "template1"
    assert_output --partial "testdb"
}

@test "psql: list tables command when no tables exist" {
    run psql -c "\\dt"
    assert_success
    assert_output "Did not find any relations."
}

@test "psql: list tables command with tables" {
    mock::postgres::add_table "users"
    mock::postgres::add_table "products"
    
    run psql -c "\\dt"
    assert_success
    assert_output --partial "List of relations"
    assert_output --partial "users"
    assert_output --partial "products"
    assert_output --partial "(2 rows)"
}

@test "psql: connect to database" {
    run psql -c "\\c mydb"
    assert_success
    assert_output --partial "You are now connected to database \"mydb\""
}

@test "psql: empty command" {
    run psql -c ""
    assert_success
}

@test "psql: quit command" {
    run psql -c "\\q"
    assert_success
}

# =============================================================================
# Database DDL operations
# =============================================================================

@test "psql: CREATE DATABASE" {
    run psql -c "CREATE DATABASE testdb123;"
    assert_success
    assert_output "CREATE DATABASE"
    
    # Verify database was added by checking psql list command
    run psql -c "\\l"
    assert_success
    assert_output --partial "testdb123"
}

@test "psql: DROP DATABASE" {
    mock::postgres::add_database "dropme"
    
    run psql -c "DROP DATABASE dropme;"
    assert_success
    assert_output "DROP DATABASE"
    
    # Reload state from file after subshell execution
    mock::postgres::load_state
    
    # Verify database was removed from mock state (assertion should fail since db is gone)
    ! mock::postgres::assert_database_exists "dropme" 2>/dev/null
}

@test "psql: CREATE TABLE" {
    run psql -c "CREATE TABLE employees (id INT, name VARCHAR(100));"
    assert_success
    assert_output "CREATE TABLE"
    
    # Verify table was added by checking psql list command
    run psql -c "\\dt"
    assert_success
    assert_output --partial "employees"
}

@test "psql: DROP TABLE" {
    mock::postgres::add_table "dropme_table"
    
    run psql -c "DROP TABLE dropme_table;"
    assert_success
    assert_output "DROP TABLE"
    
    # Reload state from file after subshell execution
    mock::postgres::load_state
    
    # Verify table was removed from mock state (assertion should fail since table is gone)
    ! mock::postgres::assert_table_exists "dropme_table" 2>/dev/null
}

# =============================================================================
# DML operations
# =============================================================================

@test "psql: INSERT statement" {
    run psql -c "INSERT INTO users (name) VALUES ('John Doe');"
    assert_success
    assert_output "INSERT 0 1"
}

@test "psql: UPDATE statement" {
    run psql -c "UPDATE users SET name = 'Jane Doe' WHERE id = 1;"
    assert_success
    assert_output "UPDATE 1"
}

@test "psql: DELETE statement" {
    run psql -c "DELETE FROM users WHERE id = 1;"
    assert_success
    assert_output "DELETE 1"
}

@test "psql: SELECT statement with no results" {
    run psql -c "SELECT * FROM users WHERE id = 999;"
    assert_success
    assert_output "(No rows)"
}

# =============================================================================
# Custom query results
# =============================================================================

@test "psql: custom query result" {
    mock::postgres::set_query_result "SELECT COUNT(*) FROM users;" "count\n-----\n   42\n(1 row)"
    
    run psql -c "SELECT COUNT(*) FROM users;"
    assert_success
    assert_output --partial "count"
    assert_output --partial "42"
    assert_output --partial "(1 row)"
}

# =============================================================================
# File operations
# =============================================================================

@test "psql: execute SQL file" {
    echo "SELECT 1;" > test.sql
    
    run psql -f test.sql
    assert_success
    assert_output --partial "Executing SQL file: test.sql"
}

@test "psql: execute non-existent SQL file" {
    run psql -f nonexistent.sql
    assert_failure
    assert_output --partial "No such file or directory"
}

@test "psql: output to file" {
    run psql -c "SELECT 1;" -o output.txt
    assert_success
    
    # Check that file was created (simulated)
    [[ -f output.txt ]]
    
    # Check file contents
    run cat output.txt
    assert_success
    assert_output --partial "?column?"
    assert_output --partial "1"
}

# =============================================================================
# pg_isready tests
# =============================================================================

@test "pg_isready: basic connection test" {
    run pg_isready
    assert_success
    assert_output "localhost:5432 - accepting connections"
}

@test "pg_isready: with custom host and port" {
    run pg_isready -h myhost -p 5433
    assert_success
    assert_output "myhost:5433 - accepting connections"
}

@test "pg_isready: connection refused when server stopped" {
    mock::postgres::set_server_status "stopped"
    
    run pg_isready
    [[ "$status" -eq 2 ]]
    assert_output "localhost:5432 - no response"
}

@test "pg_isready: quiet mode" {
    run pg_isready -q
    assert_success
    assert_output ""
}

@test "pg_isready: quiet mode with failure" {
    mock::postgres::set_connected "false"
    
    run pg_isready -q
    [[ "$status" -eq 2 ]]
    assert_output ""
}

@test "pg_isready: help command" {
    run pg_isready --help
    assert_success
    assert_output --partial "pg_isready tests whether a PostgreSQL server is ready"
    assert_output --partial "Usage: pg_isready"
}

@test "pg_isready: version command" {
    run pg_isready --version
    assert_success
    assert_output --regexp "pg_isready \(PostgreSQL\) [0-9]+\.[0-9]+"
}

# =============================================================================
# pg_dump tests
# =============================================================================

@test "pg_dump: basic database dump" {
    mock::postgres::add_table "users"
    mock::postgres::add_table "products"
    
    run pg_dump testdb
    assert_success
    assert_output --partial "PostgreSQL database dump"
    assert_output --partial "CREATE DATABASE testdb"
    assert_output --partial "CREATE TABLE public.users"
    assert_output --partial "CREATE TABLE public.products"
    assert_output --partial "COPY public.users"
}

@test "pg_dump: schema-only dump" {
    mock::postgres::add_table "users"
    
    run pg_dump -s testdb
    assert_success
    assert_output --partial "CREATE TABLE public.users"
    refute_output --partial "COPY public.users"
}

@test "pg_dump: data-only dump" {
    mock::postgres::add_table "users"
    
    run pg_dump -a testdb
    assert_success
    assert_output --partial "COPY public.users"
    refute_output --partial "CREATE TABLE"
}

@test "pg_dump: output to file" {
    mock::postgres::add_table "users"
    
    run pg_dump -f backup.sql testdb
    assert_success
    assert_output --partial "Database dump completed successfully to backup.sql"
    
    # Check that file was created (simulated)
    [[ -f backup.sql ]]
}

@test "pg_dump: connection failure" {
    mock::postgres::set_connected "false"
    
    run pg_dump testdb
    assert_failure
    assert_output --partial "could not connect to database"
}

@test "pg_dump: help command" {
    run pg_dump --help
    assert_success
    assert_output --partial "pg_dump dumps a database"
    assert_output --partial "Usage: pg_dump"
}

@test "pg_dump: version command" {
    run pg_dump --version
    assert_success
    assert_output --regexp "pg_dump \(PostgreSQL\) [0-9]+\.[0-9]+"
}

# =============================================================================
# createdb tests
# =============================================================================

@test "createdb: basic database creation" {
    run createdb newdb
    assert_success
    
    # Verify database was created by checking psql list command
    run psql -c "\\l"
    assert_success
    assert_output --partial "newdb"
}

@test "createdb: database already exists" {
    mock::postgres::add_database "existing"
    
    run createdb existing
    assert_failure
    assert_output --partial "database \"existing\" already exists"
}

@test "createdb: connection failure" {
    mock::postgres::set_connected "false"
    
    run createdb testdb123
    assert_failure
    assert_output --partial "could not connect to database template1"
}

@test "createdb: with options" {
    run createdb -O myuser -T template0 -E UTF8 customdb
    assert_success
    
    # Verify database was created by checking psql list command
    run psql -c "\\l"
    assert_success
    assert_output --partial "customdb"
}

@test "createdb: help command" {
    run createdb --help
    assert_success
    assert_output --partial "createdb creates a PostgreSQL database"
    assert_output --partial "Usage: createdb"
}

@test "createdb: version command" {
    run createdb --version
    assert_success
    assert_output --regexp "createdb \(PostgreSQL\) [0-9]+\.[0-9]+"
}

@test "createdb: invalid option" {
    run createdb --invalid-option
    assert_failure
    assert_output --partial "invalid option"
}

@test "createdb: missing database name uses username" {
    mock::postgres::set_config "user" "testuser"
    
    run createdb
    assert_success
    
    # Should create database with username - verify by checking psql list
    run psql -c "\\l"
    assert_success
    assert_output --partial "testuser"
}

# =============================================================================
# dropdb tests
# =============================================================================

@test "dropdb: basic database drop" {
    mock::postgres::add_database "dropme"
    
    run dropdb dropme
    assert_success
    
    # Reload state from file after subshell execution
    mock::postgres::load_state
    
    # Verify database was dropped (assertion should fail since db is gone)
    ! mock::postgres::assert_database_exists "dropme" 2>/dev/null
}

@test "dropdb: database does not exist" {
    run dropdb nonexistent
    assert_failure
    assert_output --partial "database \"nonexistent\" does not exist"
}

@test "dropdb: connection failure" {
    mock::postgres::set_connected "false"
    
    run dropdb testdb
    assert_failure
    assert_output --partial "could not connect to database template1"
}

@test "dropdb: interactive mode" {
    mock::postgres::add_database "interactive_drop"
    
    run dropdb -i interactive_drop
    assert_success
    assert_output --partial "Database \"interactive_drop\" will be permanently removed"
    assert_output --partial "Are you sure? (y/N) y"
    
    # Reload state from file after subshell execution
    mock::postgres::load_state
    
    # Verify database was dropped (assertion should fail since db is gone)
    ! mock::postgres::assert_database_exists "interactive_drop" 2>/dev/null
}

@test "dropdb: help command" {
    run dropdb --help
    assert_success
    assert_output --partial "dropdb removes a PostgreSQL database"
    assert_output --partial "Usage: dropdb"
}

@test "dropdb: version command" {
    run dropdb --version
    assert_success
    assert_output --regexp "dropdb \(PostgreSQL\) [0-9]+\.[0-9]+"
}

@test "dropdb: missing database name" {
    run dropdb
    assert_failure
    assert_output --partial "missing required argument database name"
}

@test "dropdb: invalid option" {
    run dropdb --invalid-option
    assert_failure
    assert_output --partial "invalid option"
}

# =============================================================================
# State persistence tests
# =============================================================================

@test "postgres state persistence across subshells" {
    # Set data in parent shell
    mock::postgres::add_database "persistdb"
    mock::postgres::add_table "persisttable"
    mock::postgres::save_state
    
    # Verify in subshell
    output=$(
        source "$MOCK_DIR/postgres.bash"
        mock::postgres::assert_database_exists "persistdb" && \
        mock::postgres::assert_table_exists "persisttable" && \
        echo "persistence_success"
    )
    [[ "$output" == "persistence_success" ]]
}

@test "postgres state file creation and loading" {
    mock::postgres::add_database "statedb"
    mock::postgres::add_table "statetable"
    mock::postgres::set_config "port" "5433"
    mock::postgres::save_state
    
    # Check state file exists
    [[ -f "$POSTGRES_MOCK_STATE_DIR/postgres-state.sh" ]]
    
    # Reset without saving state, then reload
    mock::postgres::reset false
    mock::postgres::load_state
    
    # Test assertions without run to avoid subshell issues
    mock::postgres::assert_database_exists "statedb"
    mock::postgres::assert_table_exists "statetable"
    mock::postgres::assert_config_value "port" "5433"
}

# =============================================================================
# Test helper functions
# =============================================================================

@test "mock::postgres::reset clears all data" {
    mock::postgres::add_database "testdb1"
    mock::postgres::add_table "testtable1"
    mock::postgres::set_error "connection_timeout"
    mock::postgres::set_config "port" "5433"
    
    mock::postgres::reset
    
    # Error mode should be cleared
    run psql -c "SELECT 1;"
    assert_success
    
    # Config should be reset (without run to avoid subshell)
    mock::postgres::assert_config_value "port" "5432"
    
    # Data should be cleared (expect failures)
    run mock::postgres::assert_database_exists "testdb1"
    assert_failure
    
    run mock::postgres::assert_table_exists "testtable1"
    assert_failure
}

@test "mock::postgres::assert_database_exists" {
    mock::postgres::add_database "mydb"
    
    # Test success case (without run to avoid subshell)
    mock::postgres::assert_database_exists "mydb"
    
    # Test failure case
    run mock::postgres::assert_database_exists "nonexistent"
    assert_failure
    assert_output --partial "Database 'nonexistent' does not exist"
}

@test "mock::postgres::assert_table_exists" {
    mock::postgres::add_table "mytable"
    
    # Test success case (without run to avoid subshell)
    mock::postgres::assert_table_exists "mytable"
    
    # Test failure case
    run mock::postgres::assert_table_exists "nonexistent"
    assert_failure
    assert_output --partial "Table 'nonexistent' does not exist"
}

@test "mock::postgres::assert_config_value" {
    mock::postgres::set_config "host" "myhost"
    
    # Test success case (without run to avoid subshell)
    mock::postgres::assert_config_value "host" "myhost"
    
    # Test failure case
    run mock::postgres::assert_config_value "host" "wronghost"
    assert_failure
    assert_output --partial "Config 'host' value mismatch"
    assert_output --partial "Expected: 'wronghost'"
    assert_output --partial "Actual: 'myhost'"
}

@test "mock::postgres::dump_state shows current state" {
    mock::postgres::add_database "db1"
    mock::postgres::add_table "table1"
    mock::postgres::set_config "port" "5433"
    mock::postgres::set_query_result "SELECT 1;" "custom result"
    
    run mock::postgres::dump_state
    assert_success
    assert_output --partial "PostgreSQL Mock State"
    assert_output --partial "port: 5433"
    assert_output --partial "db1: true"
    assert_output --partial "table1: true"
    assert_output --partial "SELECT 1;: custom result"
}

# =============================================================================
# Complex scenario tests
# =============================================================================

@test "postgres: complete database lifecycle" {
    # Create database
    run createdb lifecycle_test
    assert_success
    
    # Verify it exists using psql command (this will work with state persistence)
    run psql -c "\\l"
    assert_success
    assert_output --partial "lifecycle_test"
    
    # Create table in database
    run psql -d lifecycle_test -c "CREATE TABLE users (id INT, name VARCHAR(100));"
    assert_success
    
    # Insert data
    run psql -d lifecycle_test -c "INSERT INTO users VALUES (1, 'John Doe');"
    assert_success
    assert_output "INSERT 0 1"
    
    # Dump database - use default database since lifecycle_test won't be in dump context
    run pg_dump
    assert_success
    assert_output --partial "PostgreSQL database dump"
    assert_output --partial "CREATE TABLE public.users"
    
    # Drop database
    run dropdb lifecycle_test
    assert_success
    
    # Verify it's gone (expect failure)
    run mock::postgres::assert_database_exists "lifecycle_test"
    assert_failure
}

@test "postgres: connection parameter handling" {
    # Test various connection parameters
    run psql -h myhost -p 5433 -U myuser -d mydb -c "SELECT version();"
    assert_success
    assert_output --partial "PostgreSQL"
    
    # Test connection with custom parameters in pg_isready
    run pg_isready -h myhost -p 5433 -U myuser -d mydb
    assert_success
    assert_output "myhost:5433 - accepting connections"
    
    # Test pg_dump with connection parameters
    run pg_dump -h myhost -p 5433 -U myuser mydb
    assert_success
    assert_output --partial "PostgreSQL database dump"
}

@test "postgres: error handling with multiple commands" {
    # Set error mode
    mock::postgres::set_error "auth_failed"
    
    # All commands should fail with auth error
    run psql -c "SELECT 1;"
    assert_failure
    assert_output --partial "password authentication failed"
    
    run pg_dump testdb
    assert_failure
    assert_output --partial "could not connect"
    
    run createdb newdb
    assert_failure
    assert_output --partial "could not connect"
    
    # Clear error mode
    mock::postgres::set_error ""
    
    # Commands should work again
    run psql -c "SELECT 1;"
    assert_success
}

# =============================================================================
# Edge cases and error conditions
# =============================================================================

@test "postgres: empty SQL commands" {
    run psql -c ""
    assert_success
}

@test "postgres: commands with special characters" {
    run psql -c "SELECT 'hello world';"
    assert_success
    assert_output --partial "(No rows)"
}

@test "postgres: long commands" {
    local long_sql="SELECT 1 AS very_long_column_name_that_goes_on_and_on_and_on;"
    run psql -c "$long_sql"
    assert_success
    assert_output --partial "(No rows)"
}

@test "postgres: invalid SQL commands" {
    # Mock handles invalid SQL gracefully
    run psql -c "INVALID SQL SYNTAX HERE;"
    assert_success
    assert_output --partial "Command executed successfully"
}

@test "postgres: case sensitivity in commands" {
    run psql -c "select version();"
    assert_success
    assert_output --partial "(No rows)"
    
    run psql -c "SELECT VERSION();"
    assert_success
    assert_output --partial "(No rows)"
}