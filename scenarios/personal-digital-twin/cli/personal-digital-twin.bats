#!/usr/bin/env bats

# Test suite for Personal Digital Twin CLI

setup() {
    # Set up test environment
    export API_BASE_URL="http://localhost:8200"
    export CHAT_BASE_URL="http://localhost:8201"
    CLI_SCRIPT="${BATS_TEST_FILENAME%/*}/personal-digital-twin"
    
    # Skip tests if API is not running
    if ! curl -sf "$API_BASE_URL/health" > /dev/null 2>&1; then
        skip "API server not running at $API_BASE_URL"
    fi
}

@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "Show help when no arguments provided" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Personal Digital Twin CLI" ]]
}

@test "Show help with --help flag" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "Status command shows service status" {
    run "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Checking service status" ]]
}

@test "Persona list command works" {
    run "$CLI_SCRIPT" persona list
    [ "$status" -eq 0 ]
    # Should return either empty array or list of personas
}

@test "Invalid command shows error" {
    run "$CLI_SCRIPT" invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "Persona create requires name argument" {
    run "$CLI_SCRIPT" persona create
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "Chat command requires persona_id and message" {
    run "$CLI_SCRIPT" chat
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "Search command requires persona_id and query" {
    run "$CLI_SCRIPT" search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "Train start command requires all parameters" {
    run "$CLI_SCRIPT" train start
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "Token create command requires persona_id and name" {
    run "$CLI_SCRIPT" token create
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "Datasource connect requires persona_id and type" {
    run "$CLI_SCRIPT" datasource connect
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

# Integration test - create a persona and test operations
@test "Full workflow: create persona, connect datasource, and chat" {
    # Skip if chat API is not running
    if ! curl -sf "$CHAT_BASE_URL/health" > /dev/null 2>&1; then
        skip "Chat API server not running"
    fi
    
    # Create a test persona
    run "$CLI_SCRIPT" persona create "TestPersona" "Test persona for BATS"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Persona created with ID:" ]]
    
    # Extract persona ID from output
    persona_id=$(echo "$output" | grep -o '"id": "[^"]*"' | cut -d'"' -f4)
    
    # Test persona show
    run "$CLI_SCRIPT" persona show "$persona_id"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "TestPersona" ]]
    
    # Test datasource connect
    run "$CLI_SCRIPT" datasource connect "$persona_id" "local_files" '{"path":"/test"}'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Data source connected" ]]
    
    # Test chat
    run "$CLI_SCRIPT" chat "$persona_id" "Hello, test message"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "TestPersona" ]]
    
    # Test search
    run "$CLI_SCRIPT" search "$persona_id" "test query" 5
    [ "$status" -eq 0 ]
}