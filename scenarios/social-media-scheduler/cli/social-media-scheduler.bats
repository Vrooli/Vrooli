#!/usr/bin/env bats

# Social Media Scheduler CLI Test Suite
# These tests verify CLI functionality without requiring authentication

setup() {
    # Set up test environment
    export CLI_PATH="$(dirname "$BATS_TEST_FILENAME")/social-media-scheduler"
    export API_PORT="${API_PORT:-18000}"
    export API_URL="http://localhost:${API_PORT}"
    
    # Ensure CLI is executable
    [ -x "$CLI_PATH" ]
}

# Basic CLI tests
@test "CLI shows help when no arguments provided" {
    run "$CLI_PATH"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Social Media Scheduler CLI"* ]]
    [[ "$output" == *"USAGE:"* ]]
}

@test "CLI shows help with help command" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Social Media Scheduler CLI"* ]]
    [[ "$output" == *"AUTHENTICATION:"* ]]
    [[ "$output" == *"POST MANAGEMENT:"* ]]
}

@test "CLI shows version information" {
    run "$CLI_PATH" version
    [ "$status" -eq 0 ]
    [[ "$output" == *"version 1.0.0"* ]]
    [[ "$output" == *"Social Media Scheduler CLI"* ]]
}

@test "CLI shows error for unknown command" {
    run "$CLI_PATH" invalid_command
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown command"* ]]
}

# Authentication tests (without actual API)
@test "Login command requires email and password" {
    run "$CLI_PATH" login
    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"login <email> <password>"* ]]
}

@test "Register command requires email and password" {
    run "$CLI_PATH" register
    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"register <email> <password>"* ]]
}

@test "Schedule command requires all parameters" {
    run "$CLI_PATH" schedule
    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"schedule <title> <content> <platforms> <scheduled_date>"* ]]
}

@test "Status command requires post ID" {
    run "$CLI_PATH" status
    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"status <post_id>"* ]]
}

# Dependency checks
@test "CLI checks for curl dependency" {
    # Temporarily hide curl if it exists
    if command -v curl >/dev/null 2>&1; then
        skip "curl is installed, cannot test missing dependency"
    fi
    
    run "$CLI_PATH" health
    [ "$status" -eq 1 ]
    [[ "$output" == *"Missing required dependencies"* ]]
    [[ "$output" == *"curl"* ]]
}

@test "CLI checks for jq dependency" {
    # Temporarily hide jq if it exists
    if command -v jq >/dev/null 2>&1; then
        skip "jq is installed, cannot test missing dependency"
    fi
    
    run "$CLI_PATH" health
    [ "$status" -eq 1 ]
    [[ "$output" == *"Missing required dependencies"* ]]
    [[ "$output" == *"jq"* ]]
}

# API connection tests (when API is not running)
@test "CLI handles API server not running gracefully" {
    # This test assumes API is not running on a random port
    export API_PORT="19999"  # Use unlikely port
    
    run "$CLI_PATH" health
    [ "$status" -eq 1 ]
    [[ "$output" == *"API server is not running"* ]]
}

@test "CLI provides helpful message when API is down" {
    export API_PORT="19999"  # Use unlikely port
    
    run "$CLI_PATH" platforms
    [ "$status" -eq 1 ]
    [[ "$output" == *"Please start the API server first"* ]]
    [[ "$output" == *"cd api && go run ."* ]]
}

# Authentication state tests
@test "Commands requiring authentication show login message" {
    export API_PORT="19999"  # Ensure API calls fail
    
    # Clear any existing token
    rm -f "$HOME/.social-media-scheduler/token"
    
    run "$CLI_PATH" whoami
    [ "$status" -eq 1 ]
    [[ "$output" == *"Not logged in"* ]] || [[ "$output" == *"Please login first"* ]]
}

@test "List command requires authentication" {
    # Clear any existing token
    rm -f "$HOME/.social-media-scheduler/token"
    
    run "$CLI_PATH" list
    [ "$status" -eq 1 ]
    [[ "$output" == *"Please login first"* ]]
}

@test "Accounts command requires authentication" {
    # Clear any existing token
    rm -f "$HOME/.social-media-scheduler/token"
    
    run "$CLI_PATH" accounts
    [ "$status" -eq 1 ]
    [[ "$output" == *"Please login first"* ]]
}

# Token management tests
@test "Token file is created with proper permissions" {
    # Create a dummy token for testing
    mkdir -p "$HOME/.social-media-scheduler"
    echo "test_token" > "$HOME/.social-media-scheduler/token"
    chmod 600 "$HOME/.social-media-scheduler/token"
    
    # Check permissions
    local perms=$(stat -c "%a" "$HOME/.social-media-scheduler/token" 2>/dev/null || stat -f "%A" "$HOME/.social-media-scheduler/token" 2>/dev/null)
    [[ "$perms" == "600" ]]
    
    # Clean up
    rm -f "$HOME/.social-media-scheduler/token"
}

@test "Logout clears token file" {
    # Create a dummy token
    mkdir -p "$HOME/.social-media-scheduler"
    echo "test_token" > "$HOME/.social-media-scheduler/token"
    
    # Logout should clear the token
    run "$CLI_PATH" logout
    [ "$status" -eq 0 ]
    [[ "$output" == *"Logged out successfully"* ]]
    
    # Token file should be gone
    [ ! -f "$HOME/.social-media-scheduler/token" ]
}

# Help text validation
@test "Help includes all main command categories" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    
    # Check for all major sections
    [[ "$output" == *"AUTHENTICATION:"* ]]
    [[ "$output" == *"POST MANAGEMENT:"* ]]
    [[ "$output" == *"PLATFORM MANAGEMENT:"* ]]
    [[ "$output" == *"SYSTEM:"* ]]
    [[ "$output" == *"EXAMPLES:"* ]]
}

@test "Help includes example commands" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" == *"login user@example.com"* ]]
    [[ "$output" == *"schedule"* ]]
    [[ "$output" == *"twitter,linkedin"* ]]
}

# Configuration tests
@test "CLI respects API_PORT environment variable" {
    export API_PORT="19998"
    
    run "$CLI_PATH" health
    [ "$status" -eq 1 ]
    # The error message should reference the custom port
    [[ "$output" == *"19998"* ]]
}

# Input validation tests
@test "Schedule command validates date format info in error" {
    # Clear token to avoid auth issues
    rm -f "$HOME/.social-media-scheduler/token"
    
    run "$CLI_PATH" schedule "title" "content" "twitter" "invalid-date"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Please login first"* ]] || [[ "$output" == *"Invalid"* ]]
}

# Cleanup
teardown() {
    # Clean up any test tokens
    rm -f "$HOME/.social-media-scheduler/token"
}