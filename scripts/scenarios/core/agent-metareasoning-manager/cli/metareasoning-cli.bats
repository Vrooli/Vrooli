#!/usr/bin/env bats
# Agent Metareasoning Manager CLI Tests

setup() {
    export CLI_PATH="./metareasoning-cli.sh"
    export METAREASONING_API_BASE="http://localhost:8093/api"
    export TEST_CONFIG_DIR="$(mktemp -d)"
    export HOME="$TEST_CONFIG_DIR"
}

teardown() {
    if [[ -d "$TEST_CONFIG_DIR" ]]; then
        rm -rf "$TEST_CONFIG_DIR"
    fi
}

# Helper function to run CLI command
run_cli() {
    bash "$CLI_PATH" "$@"
}

# Basic tests
@test "CLI shows version" {
    run run_cli --version
    [[ "$status" -eq 0 ]]
    [[ "$output" == "1.0.0" ]]
}

@test "CLI shows help" {
    run run_cli --help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Agent Metareasoning Manager CLI"* ]]
    [[ "$output" == *"Usage:"* ]]
}

@test "CLI shows usage with no args" {
    run run_cli
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Usage:"* ]]
}

# Configuration tests
@test "CLI creates default config on first run" {
    run run_cli config show
    [[ "$status" -eq 0 ]]
    [[ -f "$HOME/.metareasoning/config.json" ]]
}

@test "CLI sets configuration value" {
    run run_cli config set api_base "http://localhost:9999/api"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Configuration updated"* ]]
    
    # Verify the value was set
    run run_cli config show
    [[ "$output" == *"http://localhost:9999/api"* ]]
}

@test "CLI resets configuration" {
    # First set a custom value
    run run_cli config set api_base "http://custom:9999/api"
    [[ "$status" -eq 0 ]]
    
    # Then reset
    run run_cli config reset
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Configuration reset to defaults"* ]]
    
    # Verify it's back to default
    run run_cli config show
    [[ "$output" == *"http://localhost:8093/api"* ]]
}

# Command validation tests
@test "CLI requires subcommand for prompt" {
    run run_cli prompt
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Unknown prompt subcommand"* ]]
}

@test "CLI requires subcommand for workflow" {
    run run_cli workflow
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Unknown workflow subcommand"* ]]
}

@test "CLI requires input for analyze commands" {
    run run_cli analyze decision
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Analysis input required"* ]]
}

# Format option tests
@test "CLI accepts format option" {
    # Mock curl to return JSON
    function curl() {
        echo '{"status":"healthy","timestamp":"2024-01-01T00:00:00Z"}'
        return 0
    }
    export -f curl
    
    run run_cli --format json health
    [[ "$status" -eq 0 ]]
}

# API base override test
@test "CLI accepts api-base override" {
    run run_cli --api-base "http://custom:8093/api" config show
    [[ "$status" -eq 0 ]]
}

# Analysis command structure tests
@test "CLI structures decision analysis payload correctly" {
    # Mock curl to capture the payload
    function curl() {
        for arg in "$@"; do
            if [[ "$arg" == -d* ]]; then
                echo "$arg" | grep -q '"scenario"' && echo '{"status":"success"}'
                return 0
            fi
        done
        return 1
    }
    export -f curl
    
    run run_cli analyze decision "Should we migrate?"
    [[ "$status" -eq 0 ]]
}

@test "CLI structures pros-cons analysis payload correctly" {
    function curl() {
        for arg in "$@"; do
            if [[ "$arg" == -d* ]]; then
                echo "$arg" | grep -q '"topic"' && echo '{"status":"success"}'
                return 0
            fi
        done
        return 1
    }
    export -f curl
    
    run run_cli analyze pros-cons "Remote work"
    [[ "$status" -eq 0 ]]
}

@test "CLI structures SWOT analysis payload correctly" {
    function curl() {
        for arg in "$@"; do
            if [[ "$arg" == -d* ]]; then
                echo "$arg" | grep -q '"target"' && echo '{"status":"success"}'
                return 0
            fi
        done
        return 1
    }
    export -f curl
    
    run run_cli analyze swot "New product launch"
    [[ "$status" -eq 0 ]]
}

@test "CLI structures risk analysis payload correctly" {
    function curl() {
        for arg in "$@"; do
            if [[ "$arg" == -d* ]]; then
                echo "$arg" | grep -q '"proposal"' && echo '{"status":"success"}'
                return 0
            fi
        done
        return 1
    }
    export -f curl
    
    run run_cli analyze risk "System migration"
    [[ "$status" -eq 0 ]]
}

# Prompt command tests
@test "CLI creates prompt with required name" {
    run run_cli prompt create
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Prompt name required"* ]]
}

@test "CLI tests prompt with required ID and input" {
    run run_cli prompt test
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Prompt ID required"* ]]
    
    run run_cli prompt test "prompt-123"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Test input required"* ]]
}

# Workflow command tests
@test "CLI runs workflow with required ID" {
    run run_cli workflow run
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Workflow ID or name required"* ]]
}

@test "CLI checks workflow status with required IDs" {
    run run_cli workflow status
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Workflow ID and execution ID required"* ]]
    
    run run_cli workflow status "workflow-123"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Workflow ID and execution ID required"* ]]
}

# Health check test (when API is not running)
@test "CLI handles API connection failure gracefully" {
    # Use a port where nothing is running
    export METAREASONING_API_BASE="http://localhost:99999/api"
    
    run run_cli health
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Failed to connect to API"* ]]
    [[ "$output" == *"Is the metareasoning API server running?"* ]]
}

# Error handling tests
@test "CLI handles unknown command" {
    run run_cli unknown-command
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Unknown command: unknown-command"* ]]
}

@test "CLI handles unknown analysis type" {
    run run_cli analyze unknown-type "input"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Unknown analysis type: unknown-type"* ]]
}

# Dependency check tests
@test "CLI requires jq" {
    # Mock missing jq
    function command() {
        if [[ "$2" == "jq" ]]; then
            return 1
        fi
        return 0
    }
    export -f command
    
    run run_cli health
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"jq is required"* ]]
}

@test "CLI requires curl" {
    # Mock missing curl
    function command() {
        if [[ "$2" == "curl" ]]; then
            return 1
        fi
        return 0
    }
    export -f command
    
    run run_cli health
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"curl is required"* ]]
}