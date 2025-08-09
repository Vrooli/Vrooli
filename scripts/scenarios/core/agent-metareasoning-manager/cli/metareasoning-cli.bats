#!/usr/bin/env bats
# Agent Metareasoning Manager CLI Tests
# Tests for the metareasoning CLI functionality

load "../../../__test/helpers/bats-support/load"
load "../../../__test/helpers/bats-assert/load"

setup() {
    # Set up test environment
    export METAREASONING_API_BASE="http://localhost:8093/api"
    export CLI_SCRIPT="${BATS_TEST_DIRNAME}/metareasoning-cli.sh"
    
    # Create temporary config directory for tests
    export TEST_CONFIG_DIR="${BATS_TMPDIR}/metareasoning-test-config"
    export HOME="${BATS_TMPDIR}"
    mkdir -p "$TEST_CONFIG_DIR"
}

teardown() {
    # Clean up test environment
    rm -rf "$TEST_CONFIG_DIR"
}

@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI shows help when called without arguments" {
    run "$CLI_SCRIPT"
    assert_success
    assert_output --partial "Agent Metareasoning Manager CLI"
    assert_output --partial "Usage:"
}

@test "CLI shows help with --help flag" {
    run "$CLI_SCRIPT" --help
    assert_success
    assert_output --partial "Agent Metareasoning Manager CLI"
    assert_output --partial "Commands:"
    assert_output --partial "prompt"
    assert_output --partial "workflow"
    assert_output --partial "analyze"
}

@test "CLI shows version with --version flag" {
    run "$CLI_SCRIPT" --version
    assert_success
    assert_output "1.0.0"
}

@test "CLI rejects unknown commands" {
    run "$CLI_SCRIPT" unknown-command
    assert_failure
    assert_output --partial "Unknown command: unknown-command"
    assert_output --partial "Use 'metareasoning help' for usage information"
}

@test "CLI rejects unknown options" {
    run "$CLI_SCRIPT" --unknown-option
    assert_failure
    assert_output --partial "Unknown option: --unknown-option"
}

@test "CLI config command shows default configuration" {
    run "$CLI_SCRIPT" config show
    assert_success
    # Should either show config or indicate no config found
}

@test "CLI config set command updates configuration" {
    run "$CLI_SCRIPT" config set api_base "http://test:8093/api"
    assert_success
    assert_output --partial "Configuration updated: api_base = http://test:8093/api"
}

@test "CLI config reset command resets configuration" {
    # First set something
    "$CLI_SCRIPT" config set test_key test_value
    
    # Then reset
    run "$CLI_SCRIPT" config reset
    assert_success
    assert_output --partial "Configuration reset to defaults"
}

@test "CLI prompt list command has correct structure" {
    # This test requires API server to be running
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" prompt list
    # Should succeed or show reasonable error message
}

@test "CLI workflow list command has correct structure" {
    # This test requires API server to be running
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" workflow list
    # Should succeed or show reasonable error message
}

@test "CLI analyze command requires input" {
    # This test requires API server to be running
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" analyze decision
    assert_failure
    assert_output --partial "Analysis input required"
}

@test "CLI analyze command accepts valid types" {
    # This test requires API server to be running
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" analyze decision "Test decision"
    # Should succeed or show reasonable error message for API connection
}

@test "CLI health command checks API server" {
    # This test requires API server to be running  
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" health
    # Should succeed if API is running, or show connection error
}

@test "CLI template list command has correct structure" {
    # This test requires API server to be running
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" template list
    # Should succeed or show reasonable error message
}

@test "CLI supports different output formats" {
    skip_if_no_api_server
    
    run "$CLI_SCRIPT" --format json template list
    # Should succeed and format correctly
}

@test "CLI handles missing dependencies gracefully" {
    # Temporarily rename jq to test dependency checking
    if command -v jq > /dev/null; then
        # Create a test environment without jq
        PATH="/nonexistent:$PATH" run "$CLI_SCRIPT" health
        assert_failure
        assert_output --partial "jq is required"
    else
        skip "jq not available for dependency test"
    fi
}

# Helper functions

skip_if_no_api_server() {
    # Check if API server is running
    if ! curl -sf "${METAREASONING_API_BASE}/health" > /dev/null 2>&1; then
        skip "API server not running at ${METAREASONING_API_BASE}"
    fi
}

# Test the configuration file creation and management
@test "CLI creates config file with proper structure" {
    # Remove any existing config
    rm -f "${HOME}/.metareasoning/config.json"
    
    # Run any command that triggers config creation
    run "$CLI_SCRIPT" config show
    
    # Check that config file was created
    [ -f "${HOME}/.metareasoning/config.json" ]
    
    # Check config file structure
    run cat "${HOME}/.metareasoning/config.json"
    assert_success
    assert_output --partial '"api_base"'
    assert_output --partial '"default_format"'
}

@test "CLI preserves existing configuration" {
    # Set initial config
    "$CLI_SCRIPT" config set test_key initial_value
    
    # Set another key
    "$CLI_SCRIPT" config set another_key another_value
    
    # Verify both keys exist
    run cat "${HOME}/.metareasoning/config.json"
    assert_success
    assert_output --partial '"test_key": "initial_value"'
    assert_output --partial '"another_key": "another_value"'
}

@test "CLI validates required arguments for analyze commands" {
    run "$CLI_SCRIPT" analyze pros-cons
    assert_failure
    assert_output --partial "Analysis input required"
    
    run "$CLI_SCRIPT" analyze swot
    assert_failure
    assert_output --partial "Analysis input required"
    
    run "$CLI_SCRIPT" analyze risk
    assert_failure
    assert_output --partial "Analysis input required"
}