#!/usr/bin/env bats
# Agent Metareasoning Manager CLI Tests - Ultra-thin API wrapper

setup() {
    export CLI_PATH="./agent-metareasoning-manager-cli.sh"
    export AGENT_METAREASONING_MANAGER_API_BASE="http://localhost:8090"
    export AGENT_METAREASONING_MANAGER_TOKEN="test-token"
}

# Helper function to run CLI command
run_cli() {
    bash "$CLI_PATH" "$@"
}

# Tests for the 5 CLI commands: health, list, search, analyze, help

@test "CLI shows help" {
    run run_cli help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Agent Metareasoning Manager CLI - Ultra-thin API wrapper"* ]]
    [[ "$output" == *"Commands:"* ]]
}

@test "CLI shows help with --help" {
    run run_cli --help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Agent Metareasoning Manager CLI - Ultra-thin API wrapper"* ]]
}

@test "CLI shows help with no args" {
    run run_cli
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Agent Metareasoning Manager CLI - Ultra-thin API wrapper"* ]]
}

@test "CLI shows version" {
    run run_cli version
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"agent-metareasoning-manager CLI version"* ]]
}

@test "CLI shows version with --version" {
    run run_cli --version
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"agent-metareasoning-manager CLI version"* ]]
}

@test "CLI shows API base URL" {
    run run_cli api
    [[ "$status" -eq 0 ]]
    [[ "$output" == "http://localhost:8090" ]]
}

# Mock successful curl responses for API tests
mock_curl_success() {
    function curl() {
        local payload="$*"
        if [[ "$payload" == *"/health"* ]]; then
            echo '{"status":"healthy","timestamp":"2024-01-01T00:00:00Z"}'
            return 0
        fi

        if [[ "$payload" == *"/workflows/search"* ]]; then
            echo '{"results":[{"name":"risk-assessment","score":0.9}]}'
            return 0
        fi

        if [[ "$payload" == *"/workflows"* ]]; then
            echo '{"workflows":[{"name":"pros-cons-analyzer","platform":"n8n"}]}'
            return 0
        fi

        if [[ "$payload" == *"/analyze"* ]]; then
            echo '{"analysis":"complete","type":"pros-cons"}'
            return 0
        fi

        return 1
    }
    export -f curl
}

@test "CLI health command calls correct endpoint" {
    mock_curl_success
    
    run run_cli health
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"healthy"* ]]
}

@test "CLI list command calls workflows endpoint" {
    mock_curl_success
    
    run run_cli list
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"workflows"* ]]
}

@test "CLI workflows command calls workflows endpoint" {
    mock_curl_success
    
    run run_cli workflows
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"workflows"* ]]
}

@test "CLI search requires query argument" {
    run run_cli search
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH search <query>"* ]]
}

@test "CLI search calls correct endpoint with query" {
    mock_curl_success
    
    run run_cli search "risk assessment"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"results"* ]]
}

@test "CLI analyze requires type and input arguments" {
    run run_cli analyze
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH analyze <type> <input> [context]"* ]]
    
    run run_cli analyze pros-cons
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH analyze <type> <input> [context]"* ]]
}

@test "CLI analyze calls correct endpoint with payload" {
    mock_curl_success
    
    run run_cli analyze pros-cons "Should we adopt GraphQL?"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"analysis"* ]]
}

@test "CLI analyze accepts optional context" {
    mock_curl_success
    
    run run_cli analyze swot "New product" "competitive market"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"analysis"* ]]
}

@test "CLI handles API connection failure gracefully" {
    # Use invalid port
    export AGENT_METAREASONING_MANAGER_API_BASE="http://localhost:99999"
    
    run run_cli health
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Failed to connect to API"* ]]
}

@test "CLI handles unknown command" {
    run run_cli unknown-command
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Unknown command: unknown-command"* ]]
}
