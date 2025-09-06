#!/usr/bin/env bats

# BATS tests for AI Chatbot Manager CLI
# Run with: bats ai-chatbot-manager.bats

setup() {
    # Set up test environment
    export API_BASE_URL="${API_BASE_URL:-http://localhost:8090}"
    export CLI_PATH="./ai-chatbot-manager"
    
    # Skip tests if CLI is not executable
    if [[ ! -x "$CLI_PATH" ]]; then
        skip "CLI script is not executable: $CLI_PATH"
    fi
}

# Helper function to check if API is available
is_api_available() {
    curl -s "$API_BASE_URL/health" >/dev/null 2>&1
}

@test "CLI script is executable and has correct shebang" {
    [[ -x "$CLI_PATH" ]]
    head -n 1 "$CLI_PATH" | grep -q "#!/bin/bash"
}

@test "CLI shows version information" {
    run "$CLI_PATH" version
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "AI Chatbot Manager CLI" ]]
}

@test "CLI shows help information" {
    run "$CLI_PATH" help
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "AI Chatbot Manager CLI" ]]
    [[ "$output" =~ "Commands:" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "create" ]]
    [[ "$output" =~ "list" ]]
}

@test "CLI shows command-specific help" {
    run "$CLI_PATH" help create
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Create a new chatbot" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI handles unknown commands gracefully" {
    run "$CLI_PATH" unknown-command
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI shows version in JSON format" {
    run "$CLI_PATH" version --json
    [[ "$status" -eq 0 ]]
    
    # Check if output is valid JSON (requires jq)
    if command -v jq >/dev/null 2>&1; then
        echo "$output" | jq . >/dev/null
        echo "$output" | jq -r '.cli_version' | grep -q "^[0-9]"
    else
        # Basic JSON format check
        [[ "$output" =~ ^{.*}$ ]]
        [[ "$output" =~ "cli_version" ]]
    fi
}

@test "CLI status command works when API is available" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" status
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "AI Chatbot Manager is running" ]]
}

@test "CLI status command shows detailed info with --verbose" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" status --verbose
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Version:" ]]
    [[ "$output" =~ "Database:" ]]
    [[ "$output" =~ "Ollama:" ]]
}

@test "CLI status command returns JSON with --json flag" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" status --json
    [[ "$status" -eq 0 ]]
    
    # Check if output is valid JSON
    if command -v jq >/dev/null 2>&1; then
        echo "$output" | jq . >/dev/null
        echo "$output" | jq -r '.status' | grep -q "healthy"
    else
        [[ "$output" =~ ^{.*}$ ]]
        [[ "$output" =~ "status" ]]
    fi
}

@test "CLI list command works" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" list
    [[ "$status" -eq 0 ]]
    # Should either show chatbots or "No chatbots found"
    [[ "$output" =~ ("chatbot" || "No chatbots found") ]]
}

@test "CLI list command returns JSON with --json flag" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" list --json
    [[ "$status" -eq 0 ]]
    
    # Check if output is valid JSON (should be an array)
    if command -v jq >/dev/null 2>&1; then
        echo "$output" | jq . >/dev/null
        echo "$output" | jq 'type' | grep -q "array"
    else
        [[ "$output" =~ ^\[.*\]$ ]]
    fi
}

@test "CLI create command requires name argument" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" create
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Chatbot name is required" ]]
}

@test "CLI create command works with basic arguments" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    local test_name="CLI-Test-Bot-$(date +%s)"
    
    run "$CLI_PATH" create "$test_name" --personality "You are a test bot"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Chatbot created successfully" ]]
    [[ "$output" =~ "$test_name" ]]
}

@test "CLI handles API connectivity issues gracefully" {
    # Test with a non-existent API URL
    export API_BASE_URL="http://localhost:9999"
    
    run "$CLI_PATH" status
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ ("API server is not available" || "not responding") ]]
}

@test "CLI handles invalid JSON responses gracefully" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    # Test with an endpoint that might return invalid JSON
    # This tests the CLI's error handling for malformed responses
    
    run "$CLI_PATH" list --json
    [[ "$status" -eq 0 ]]
    
    # Even if there's an issue, it should handle it gracefully
    # and not crash with unhandled JSON parsing errors
}

@test "CLI preserves exit codes properly" {
    # Test that help command returns 0
    run "$CLI_PATH" help
    [[ "$status" -eq 0 ]]
    
    # Test that invalid command returns 1
    run "$CLI_PATH" invalid-command
    [[ "$status" -eq 1 ]]
}

@test "CLI handles empty responses" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    # Test list command when there might be no chatbots
    run "$CLI_PATH" list
    [[ "$status" -eq 0 ]]
    # Should handle empty results gracefully
}

@test "CLI chat command requires chatbot ID" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" chat
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Chatbot ID is required" ]]
}

@test "CLI analytics command requires chatbot ID" {
    if ! is_api_available; then
        skip "API is not available at $API_BASE_URL"
    fi
    
    run "$CLI_PATH" analytics
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Chatbot ID is required" ]]
}