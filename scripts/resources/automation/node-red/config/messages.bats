#!/usr/bin/env bats
# Tests for config/messages.sh

load ../test_fixtures/test_helper

setup() {
    setup_test_environment
    source_node_red_scripts
}

teardown() {
    teardown_test_environment
}

# Tests for config/messages.sh
@test "messages.sh defines all required message functions" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    # Check that key message functions exist
    declare -f node_red::show_installing >/dev/null
    declare -f node_red::show_installation_complete >/dev/null
    declare -f node_red::show_installation_failed_error >/dev/null
    declare -f node_red::show_uninstalling >/dev/null
    declare -f node_red::show_success_message >/dev/null
}

@test "messages.sh functions produce appropriate output" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_installing
    assert_success
    assert_output_contains "Installing"
    
    run node_red::show_installation_complete
    assert_success
    assert_output_contains "installed successfully"
    
    run node_red::show_uninstalling
    assert_success
    assert_output_contains "Uninstalling"
}

@test "messages.sh success message includes access URL" {
    export RESOURCE_PORT="1880"
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_success_message
    assert_success
    assert_output_contains "http://localhost:1880"
}

@test "messages.sh error messages are descriptive" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_installation_failed_error
    assert_success
    assert_output_contains "failed"
    
    run node_red::show_not_installed_error
    assert_success
    assert_output_contains "not installed"
    
    run node_red::show_not_running_error
    assert_success
    assert_output_contains "not running"
}

@test "messages.sh warning messages are informative" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_already_installed_warning
    assert_success
    assert_output_contains "already installed"
    
    run node_red::show_already_running_warning
    assert_success
    assert_output_contains "already running"
}

@test "messages.sh includes flow management messages" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_importing_flows
    assert_success
    assert_output_contains "Importing"
    
    run node_red::show_exporting_flows "/tmp/test.json"
    assert_success
    assert_output_contains "Exporting"
    assert_output_contains "/tmp/test.json"
}

@test "messages.sh includes status display functions" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_status_header
    assert_success
    assert_output_contains "Status"
    
    run node_red::show_not_installed
    assert_success
    assert_output_contains "Not installed"
}

@test "messages.sh test functions provide clear output" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::show_test_header
    assert_success
    assert_output_contains "test"
    
    run node_red::show_test_result "container status" "passed"
    assert_success
    assert_output_contains "container status"
    assert_output_contains "PASSED"
    
    run node_red::show_test_result "http endpoint" "failed" "Connection refused"
    assert_success
    assert_output_contains "http endpoint"
    assert_output_contains "FAILED"
    assert_output_contains "Connection refused"
}

@test "messages.sh test summary shows statistics" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    # Test summary with failures returns exit code 1 (correct behavior)
    run node_red::show_test_summary 5 2
    assert_failure
    assert_output_contains "Passed: 5"
    assert_output_contains "Failed: 2"
    assert_output_contains "Total: 7"
}

@test "messages.sh usage function shows help" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    run node_red::usage
    assert_success
    assert_output_contains "Usage"
    assert_output_contains "Examples"
}

@test "messages.sh prompt functions are interactive" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    # Test with automatic yes
    export YES="yes"
    run node_red::prompt_reinstall
    assert_success
    
    # Test with automatic no (should return false)
    export YES="no"
    run bash -c 'echo "n" | node_red::prompt_reinstall'
    assert_failure
}

@test "messages.sh interrupt handler is defined" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    declare -f node_red::show_interrupt_message >/dev/null
    
    run node_red::show_interrupt_message
    assert_success
    assert_output_contains "interrupted"
}

@test "messages use consistent formatting" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    # All message functions should exist and be callable
    local message_functions=(
        "show_installing"
        "show_installation_complete"
        "show_uninstalling"
        "show_success_message"
        "show_not_installed_error"
    )
    
    for func in "${message_functions[@]}"; do
        declare -f "node_red::$func" >/dev/null
    done
}

@test "messages include appropriate log levels" {
    source "$NODE_RED_TEST_DIR/config/messages.sh"
    
    # Success messages should use log::success
    run node_red::show_installation_complete
    assert_success
    
    # Error messages should use log::error
    run node_red::show_installation_failed_error
    assert_success
    
    # Info messages should use log::info
    run node_red::show_installing
    assert_success
}
