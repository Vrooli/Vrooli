#!/usr/bin/env bats
# Tests for monitor.sh

# Source test setup infrastructure  
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Mock external dependencies
    curl() { echo "Mock curl response"; return 0; }
    pg_isready() { echo "Mock pg_isready"; return 0; }
    psql() { echo "Mock psql"; return 0; }
    top() { echo "Cpu(s):  5.2%us"; }
    free() { echo -e "              total        used        free      shared  buff/cache   available\nMem:        8000000     2000000     6000000           0           0     6000000"; }
    df() { echo -e "Filesystem     1K-blocks    Used Available Use% Mounted on\n/dev/sda1       1000000  250000   750000  25% /"; }
    kill() { return 0; }
    export -f curl pg_isready psql top free df kill
}

teardown() {
    vrooli_cleanup_test
}

@test "monitor script sources var.sh correctly" {
    # Test that var.sh is sourced and variables are available
    source "$BATS_TEST_DIRNAME/monitor.sh"
    
    # Check that var_ variables are available
    [[ -n "$var_ROOT_DIR" ]]
    [[ -n "$var_SCRIPTS_DIR" ]]
}

@test "monitor::log_monitor creates proper log entries" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/monitor.sh"
    
    # Set up test environment
    export MONITOR_LOG="$VROOLI_TEST_TMPDIR/monitor.log"
    
    # Run logging function
    run monitor::log_monitor "Test message"
    
    # Verify log entry was created
    assert_success
    assert_output_contains "[MONITOR] Test message"
    [[ -f "$MONITOR_LOG" ]]
}

@test "monitor script handles command line arguments" {
    # Test help command
    run bash "$BATS_TEST_DIRNAME/monitor.sh" help
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "Commands:"
    
    # Test unknown command
    run bash "$BATS_TEST_DIRNAME/monitor.sh" unknown
    assert_failure
    assert_output_contains "Unknown command: unknown"
}

@test "monitor::main function exists and accepts parameters" {
    # Source the script
    source "$BATS_TEST_DIRNAME/monitor.sh"
    
    # Test that the main function exists
    declare -f monitor::main > /dev/null
    
    # Test help parameter (should not start actual monitoring in test)
    run monitor::main help
    assert_success
    assert_output_contains "Usage:"
}

@test "script uses relative paths via var.sh variables" {
    # Source the script
    source "$BATS_TEST_DIRNAME/monitor.sh" 
    
    # Check that log files use var_ROOT_DIR instead of hardcoded /tmp
    [[ "$MONITOR_LOG" =~ $var_ROOT_DIR ]]
    [[ "$PID_FILE" =~ $var_ROOT_DIR ]]
}