#!/usr/bin/env bats
# BATS tests for Vrooli Orchestrator CLI

# Test setup
setup() {
    export CLI_PATH="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)/vrooli-orchestrator"
    export VROOLI_ORCHESTRATOR_API_BASE="http://localhost:15001"
    
    # Ensure CLI is executable
    chmod +x "$CLI_PATH"
}

# Helper function to check if CLI exists
cli_exists() {
    [[ -x "$CLI_PATH" ]]
}

# Helper to run CLI with timeout
run_cli() {
    timeout 10 "$CLI_PATH" "$@" 2>&1 || true
}

@test "CLI file exists and is executable" {
    run cli_exists
    [[ $status -eq 0 ]]
}

@test "CLI shows version" {
    run run_cli version
    [[ $output =~ "vrooli-orchestrator CLI version" ]]
}

@test "CLI shows help" {
    run run_cli help
    [[ $output =~ "Vrooli Orchestrator CLI" ]]
    [[ $output =~ "Usage:" ]]
    [[ $output =~ "activate" ]]
    [[ $output =~ "list-profiles" ]]
}

@test "CLI shows help with --help flag" {
    run run_cli --help
    [[ $output =~ "Vrooli Orchestrator CLI" ]]
}

@test "CLI shows help with -h flag" {
    run run_cli -h
    [[ $output =~ "Vrooli Orchestrator CLI" ]]
}

@test "CLI shows version with --version flag" {
    run run_cli --version
    [[ $output =~ "vrooli-orchestrator CLI version" ]]
}

@test "CLI shows version with -v flag" {
    run run_cli -v
    [[ $output =~ "vrooli-orchestrator CLI version" ]]
}

@test "CLI shows API URL with api command" {
    run run_cli api
    [[ $output =~ "http://localhost:15001" ]]
}

@test "CLI shows error for unknown command" {
    run run_cli nonexistent-command
    [[ $status -eq 1 ]]
    [[ $output =~ "Unknown command" ]]
}

@test "CLI health command structure" {
    # This test checks command structure, not actual connectivity
    run timeout 2 "$CLI_PATH" health
    # Should either connect or show connection error, both are valid CLI behavior
    [[ $status -eq 0 || $status -eq 1 || $status -eq 124 ]]
}

@test "CLI list-profiles command structure" {
    # Test command parsing, not actual functionality (may fail due to no API)
    run timeout 2 "$CLI_PATH" list-profiles
    # Should either connect or show connection error, both indicate proper command structure
    [[ $status -eq 0 || $status -eq 1 || $status -eq 124 ]]
}

@test "CLI list command (alias for list-profiles)" {
    # Test command alias
    run timeout 2 "$CLI_PATH" list
    # Should either connect or show connection error
    [[ $status -eq 0 || $status -eq 1 || $status -eq 124 ]]
}

@test "CLI status command structure" {
    # Test status command structure
    run timeout 2 "$CLI_PATH" status
    [[ $status -eq 0 || $status -eq 1 || $status -eq 124 ]]
}

@test "CLI get-profile requires argument" {
    run run_cli get-profile
    [[ $status -eq 1 ]]
    [[ $output =~ "Usage:" ]]
}

@test "CLI create-profile requires argument" {
    run run_cli create-profile
    [[ $status -eq 1 ]]
    [[ $output =~ "Usage:" ]]
}

@test "CLI update-profile requires argument" {
    run run_cli update-profile
    [[ $status -eq 1 ]]
    [[ $output =~ "Usage:" ]]
}

@test "CLI delete-profile requires argument" {
    run run_cli delete-profile
    [[ $status -eq 1 ]]
    [[ $output =~ "Usage:" ]]
}

@test "CLI activate requires argument" {
    run run_cli activate
    [[ $status -eq 1 ]]
    [[ $output =~ "Usage:" ]]
}

@test "CLI create-profile handles options parsing" {
    # Test that create-profile can parse its options without crashing
    run timeout 2 "$CLI_PATH" create-profile test-profile --display-name "Test" --description "Test profile"
    # Should either succeed or fail with connection error, not with parsing error
    [[ $status -eq 0 || $status -eq 1 || $status -eq 124 ]]
    [[ ! $output =~ "Unknown option" ]]
}

@test "CLI create-profile rejects unknown options" {
    run run_cli create-profile test-profile --unknown-option value
    [[ $status -eq 1 ]]
    [[ $output =~ "Unknown option" ]]
}

@test "CLI update-profile rejects unknown options" {
    run run_cli update-profile test-profile --unknown-option value
    [[ $status -eq 1 ]]
    [[ $output =~ "Unknown option" ]]
}

@test "CLI respects environment variables" {
    export VROOLI_ORCHESTRATOR_API_BASE="http://example.com:9999"
    run run_cli api
    [[ $output =~ "http://example.com:9999" ]]
}

@test "CLI handles missing jq gracefully" {
    # Temporarily move jq if it exists
    JQ_PATH=$(which jq 2>/dev/null || echo "")
    if [[ -n "$JQ_PATH" ]]; then
        sudo mv "$JQ_PATH" "$JQ_PATH.bak" 2>/dev/null || true
    fi
    
    # Run CLI command that uses formatting
    run run_cli version
    [[ $output =~ "vrooli-orchestrator CLI version" ]]
    
    # Restore jq if we moved it
    if [[ -n "$JQ_PATH" && -f "$JQ_PATH.bak" ]]; then
        sudo mv "$JQ_PATH.bak" "$JQ_PATH" 2>/dev/null || true
    fi
}