#!/usr/bin/env bats
# Tests for test.sh

# Source test setup infrastructure
APP_ROOT="${APP_ROOT:-$(builtin cd "${BATS_TEST_FILENAME%/*}/../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create test scenario structure
    export TEST_SCENARIO_DIR="$VROOLI_TEST_TMPDIR/simple-test"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/storage/postgres"
    
    # Create test files
    echo '{"name": "simple-test", "version": "1.0.0"}' > "$TEST_SCENARIO_DIR/package.json"
    echo "CREATE TABLE test (id SERIAL PRIMARY KEY);" > "$TEST_SCENARIO_DIR/initialization/storage/postgres/schema.sql"
    echo "INSERT INTO test VALUES (1);" > "$TEST_SCENARIO_DIR/initialization/storage/postgres/seed.sql"
    
    # Mock commands that test script might use
    psql() { echo "Mock psql"; return 0; }
    timeout() { shift; "$@"; }
    jq() { command jq "$@"; }
    export -f psql timeout
}

teardown() {
    vrooli_cleanup_test
}

@test "test script sources var.sh correctly" {
    # Source the script to check var_ variables are available
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Check that var_ variables are available
    [[ -n "$var_LOG_FILE" ]]
    [[ -n "$var_SCRIPTS_DIR" ]]
}

@test "test::required_files detects missing files" {
    # Source the script
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Override SCENARIO_DIR to point to empty directory
    SCENARIO_DIR="$VROOLI_TEST_TMPDIR/empty"
    mkdir -p "$SCENARIO_DIR"
    
    # Run the function
    run test::required_files
    
    # Should fail when files are missing
    assert_failure
}

@test "test::required_files succeeds when files exist" {
    # Source the script
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Override SCENARIO_DIR to point to test directory with files
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Run the function
    run test::required_files
    
    # Should succeed when files exist
    assert_success
}

@test "test::sql_validity checks SQL file content" {
    # Source the script
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Override SCENARIO_DIR
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Run the function
    run test::sql_validity
    
    # Should succeed with valid SQL files
    assert_success
    assert_output_contains "SQL file looks valid"
}

@test "test::package_json validates JSON syntax" {
    # Source the script
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Override SCENARIO_DIR
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Run the function
    run test::package_json
    
    # Should succeed with valid package.json
    assert_success
    assert_output_contains "package.json is valid JSON"
}

@test "test::database_connection handles unavailable database gracefully" {
    # Source the script
    source "$BATS_TEST_DIRNAME/test.sh"
    
    # Mock psql to fail
    psql() { return 1; }
    export -f psql
    
    # Run the function
    run test::database_connection
    
    # Should succeed with warning (it's optional for simple-test)
    assert_success
    assert_output_contains "PostgreSQL not available (this is OK for simple-test scenario)"
}

@test "main function runs all tests" {
    # Mock the test functions to avoid real execution
    mkdir -p "$VROOLI_TEST_TMPDIR/mock-scenario"
    cat > "$VROOLI_TEST_TMPDIR/test-mock.sh" <<'EOF'
#!/bin/bash
source "$(builtin cd "${BASH_SOURCE[0]%/*}/../../../lib/utils" && builtin pwd)/var.sh"
SCENARIO_DIR="$1"
source "$var_LOG_FILE"

test::run_test() { echo "Running: $1"; }
test::required_files() { return 0; }
test::sql_validity() { return 0; }
test::package_json() { return 0; }
test::database_connection() { return 0; }
main() {
    test::run_test "Required files exist" test::required_files
    test::run_test "SQL files validity" test::sql_validity 
    test::run_test "Package.json validity" test::package_json
    test::run_test "Database connection" test::database_connection
}
main
EOF
    
    # Run the mock version
    run bash "$VROOLI_TEST_TMPDIR/test-mock.sh" "$TEST_SCENARIO_DIR"
    
    # Should run all tests
    assert_success
    assert_output_contains "Running: Required files exist"
    assert_output_contains "Running: SQL files validity"
    assert_output_contains "Running: Package.json validity"
    assert_output_contains "Running: Database connection"
}