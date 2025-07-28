#!/usr/bin/env bats
# Tests for Huginn config/messages.sh

load ../test-fixtures/test_helper

setup() {
    setup_test_environment
    source "$RESOURCES_DIR/common.sh"
    source "$HUGINN_TEST_DIR/config/messages.sh"
}

teardown() {
    teardown_test_environment
}

@test "messages.sh: all display functions exist" {
    # Installation messages
    declare -f huginn::show_installing >/dev/null
    declare -f huginn::show_already_installed >/dev/null
    declare -f huginn::show_install_success >/dev/null
    declare -f huginn::show_install_failed >/dev/null
    
    # Status messages
    declare -f huginn::show_not_installed >/dev/null
    declare -f huginn::show_not_running >/dev/null
    declare -f huginn::show_waiting_message >/dev/null
    
    # Header messages
    declare -f huginn::show_status_header >/dev/null
    declare -f huginn::show_agents_header >/dev/null
    declare -f huginn::show_scenarios_header >/dev/null
    declare -f huginn::show_test_header >/dev/null
}

@test "messages.sh: show_installing displays correct message" {
    run huginn::show_installing
    assert_success
    assert_output_contains "Installing Huginn"
}

@test "messages.sh: show_already_installed displays warning" {
    run huginn::show_already_installed
    assert_success
    assert_output_contains "already installed"
}

@test "messages.sh: show_not_installed displays error" {
    run huginn::show_not_installed
    assert_success
    assert_output_contains "not installed"
}

@test "messages.sh: show_not_running displays info" {
    run huginn::show_not_running
    assert_success
    assert_output_contains "not running"
}

@test "messages.sh: show_status_header displays header" {
    run huginn::show_status_header
    assert_success
    assert_output_contains "Huginn Status"
}

@test "messages.sh: show_agents_header displays header" {
    run huginn::show_agents_header
    assert_success
    assert_output_contains "Agents"
}

@test "messages.sh: show_scenarios_header displays header" {
    run huginn::show_scenarios_header
    assert_success
    assert_output_contains "Scenarios"
}

@test "messages.sh: show_install_success displays success" {
    run huginn::show_install_success
    assert_success
    assert_output_contains "successfully"
    assert_output_contains "http://localhost:4111"
}

@test "messages.sh: show_docker_error displays error" {
    run huginn::show_docker_error
    assert_success
    assert_output_contains "Docker"
}

@test "messages.sh: show_health_check_failed displays error" {
    run huginn::show_health_check_failed
    assert_success
    assert_output_contains "Health check failed"
}

@test "messages.sh: show_test_result handles passed status" {
    run huginn::show_test_result "example test" "passed"
    assert_success
    assert_output_contains "✅"
    assert_output_contains "example test"
}

@test "messages.sh: show_test_result handles failed status" {
    run huginn::show_test_result "example test" "failed" "Error message"
    assert_success
    assert_output_contains "❌"
    assert_output_contains "example test"
    assert_output_contains "Error message"
}

@test "messages.sh: usage function exists and shows help" {
    declare -f huginn::usage >/dev/null
    run huginn::usage
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "Actions:"
}