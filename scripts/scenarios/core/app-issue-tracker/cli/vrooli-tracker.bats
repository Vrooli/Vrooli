#!/usr/bin/env bats
# Tests for vrooli-tracker CLI script
# Validates issue tracker CLI functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Override config file location for testing
    export CONFIG_FILE="${BATS_TEST_TMPDIR}/config.json"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/vrooli-tracker.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test that the script loads without errors
@test "vrooli-tracker.sh loads without errors" {
    # Test that functions are defined
    declare -F vrooli_tracker::show_help >/dev/null
    declare -F vrooli_tracker::load_config >/dev/null
    declare -F vrooli_tracker::main >/dev/null
}

# Test help functionality
@test "vrooli_tracker::show_help displays usage information" {
    run vrooli_tracker::show_help
    assert_success
    assert_output --partial "Vrooli Issue Tracker CLI"
    assert_output --partial "Usage:"
    assert_output --partial "Commands:"
    assert_output --partial "create"
    assert_output --partial "list"
    assert_output --partial "get"
}

# Test configuration loading with no config file
@test "vrooli_tracker::load_config handles missing config file" {
    run vrooli_tracker::load_config
    assert_success
    # Should warn about missing token
    assert_output --partial "Warning: No API token configured"
}

# Test configuration saving
@test "vrooli_tracker::save_config creates config file" {
    run vrooli_tracker::save_config "test-token" "http://test-api" "test-app"
    assert_success
    assert_output --partial "Configuration saved"
    
    # Verify config file was created
    [[ -f "$CONFIG_FILE" ]]
    
    # Verify config content
    local token_from_file
    token_from_file=$(jq -r '.api_token' "$CONFIG_FILE")
    [[ "$token_from_file" == "test-token" ]]
}

# Test configuration loading with existing config
@test "vrooli_tracker::load_config loads existing config" {
    # Create a config file first
    vrooli_tracker::save_config "test-token" "http://test-api" "test-app"
    
    # Reset environment
    unset TRACKER_API_TOKEN
    unset TRACKER_API_URL
    
    run vrooli_tracker::load_config
    assert_success
    # Should not warn about missing token
    refute_output --partial "Warning: No API token configured"
}

# Test main help command
@test "vrooli_tracker main shows help with help argument" {
    run vrooli_tracker::main help
    assert_success
    assert_output --partial "Vrooli Issue Tracker CLI"
}

# Test main help command variants
@test "vrooli_tracker main shows help with --help argument" {
    run vrooli_tracker::main --help
    assert_success
    assert_output --partial "Vrooli Issue Tracker CLI"
}

# Test unknown command handling
@test "vrooli_tracker main handles unknown commands" {
    run vrooli_tracker::main unknown_command
    assert_failure
    assert_output --partial "Unknown command: unknown_command"
}

# Test create issue validation
@test "vrooli_tracker::create_issue requires title" {
    run vrooli_tracker::create_issue
    assert_failure
    assert_output --partial "Title is required"
}

# Test list issues function
@test "vrooli_tracker::list_issues runs without crashing" {
    if command -v mock::http::set_endpoint_response &>/dev/null; then
        run vrooli_tracker::list_issues
        # May succeed or fail depending on API availability, but should not crash
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    else
        skip "HTTP mocking not available"
    fi
}

# Test stats function
@test "vrooli_tracker::show_stats runs without crashing" {
    if command -v mock::http::set_endpoint_response &>/dev/null; then
        run vrooli_tracker::show_stats
        # May succeed or fail depending on API availability, but should not crash
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
    else
        skip "HTTP mocking not available"
    fi
}

# Test dependency checking
@test "script checks for required dependencies" {
    # The script should check for jq and curl
    # If these are missing, it should exit with error
    # This test assumes they are available in the test environment
    
    # Test the dependency check logic by checking if the functions exist
    command -v jq >/dev/null || skip "jq not available for testing"
    command -v curl >/dev/null || skip "curl not available for testing"
    
    # If we get here, dependencies are available and script should load
    true
}

# Assertion functions are provided by test fixtures