#!/usr/bin/env bats
# Campaign Content Studio CLI Tests - Ultra-thin API wrapper

setup() {
    export CLI_PATH="./campaign-content-studio-cli.sh"
    export CAMPAIGN_CONTENT_STUDIO_API_BASE="http://localhost:8090"
    export CAMPAIGN_CONTENT_STUDIO_TOKEN="test-token"
}

# Helper function to run CLI command
run_cli() {
    bash "$CLI_PATH" "$@"
}

# Tests for the CLI commands: health, campaigns, create, documents, search, generate, help

@test "CLI shows help" {
    run run_cli help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Campaign Content Studio CLI - Ultra-thin API wrapper"* ]]
    [[ "$output" == *"Commands:"* ]]
}

@test "CLI shows help with --help" {
    run run_cli --help
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Campaign Content Studio CLI - Ultra-thin API wrapper"* ]]
}

@test "CLI shows help with no args" {
    run run_cli
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Campaign Content Studio CLI - Ultra-thin API wrapper"* ]]
}

@test "CLI shows version" {
    run run_cli version
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"campaign-content-studio CLI version"* ]]
}

@test "CLI shows version with --version" {
    run run_cli --version
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"campaign-content-studio CLI version"* ]]
}

@test "CLI shows API base URL" {
    run run_cli api
    [[ "$status" -eq 0 ]]
    [[ "$output" == "http://localhost:8090" ]]
}

# Mock successful curl responses for API tests
mock_curl_success() {
    function curl() {
        case "$*" in
            *"/health"*) echo '{"status":"healthy","timestamp":"2024-01-01T00:00:00Z"}' ;;
            *"/campaigns"* && *"GET"*) echo '[{"id":"550e8400-e29b-41d4-a716-446655440001","name":"Test Campaign"}]' ;;
            *"/campaigns"* && *"POST"*) echo '{"id":"550e8400-e29b-41d4-a716-446655440001","name":"New Campaign"}' ;;
            *"/campaigns/*/documents"*) echo '[{"id":"doc-1","filename":"test.pdf"}]' ;;
            *"/campaigns/*/search"*) echo '{"results":[{"excerpt":"relevant content","score":0.9}]}' ;;
            *"/generate"*) echo '{"generated_content":"Generated blog post content","status":"success"}' ;;
            *) return 1 ;;
        esac
        return 0
    }
    export -f curl
}

@test "CLI health command calls correct endpoint" {
    mock_curl_success
    
    run run_cli health
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"healthy"* ]]
}

@test "CLI campaigns command calls campaigns endpoint" {
    mock_curl_success
    
    run run_cli campaigns
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Test Campaign"* ]]
}

@test "CLI list command calls campaigns endpoint" {
    mock_curl_success
    
    run run_cli list
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Test Campaign"* ]]
}

@test "CLI create requires name argument" {
    run run_cli create
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH create <name> [description]"* ]]
}

@test "CLI create calls correct endpoint with name" {
    mock_curl_success
    
    run run_cli create "New Campaign"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"New Campaign"* ]]
}

@test "CLI create accepts optional description" {
    mock_curl_success
    
    run run_cli create "New Campaign" "Test description"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"New Campaign"* ]]
}

@test "CLI documents requires campaign_id argument" {
    run run_cli documents
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH documents <campaign_id>"* ]]
}

@test "CLI documents calls correct endpoint" {
    mock_curl_success
    
    run run_cli documents "550e8400-e29b-41d4-a716-446655440001"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"test.pdf"* ]]
}

@test "CLI search requires campaign_id and query arguments" {
    run run_cli search
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH search <campaign_id> <query>"* ]]
    
    run run_cli search "550e8400-e29b-41d4-a716-446655440001"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH search <campaign_id> <query>"* ]]
}

@test "CLI search calls correct endpoint with query" {
    mock_curl_success
    
    run run_cli search "550e8400-e29b-41d4-a716-446655440001" "productivity tools"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"results"* ]]
}

@test "CLI generate requires campaign_id, content_type, and prompt arguments" {
    run run_cli generate
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH generate <campaign_id> <content_type> <prompt> [include_images]"* ]]
    
    run run_cli generate "550e8400-e29b-41d4-a716-446655440001"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH generate <campaign_id> <content_type> <prompt> [include_images]"* ]]
    
    run run_cli generate "550e8400-e29b-41d4-a716-446655440001" "blog_post"
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Usage: $CLI_PATH generate <campaign_id> <content_type> <prompt> [include_images]"* ]]
}

@test "CLI generate calls correct endpoint with required params" {
    mock_curl_success
    
    run run_cli generate "550e8400-e29b-41d4-a716-446655440001" "blog_post" "Write about AI tools"
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Generated blog post content"* ]]
}

@test "CLI generate accepts optional include_images parameter" {
    mock_curl_success
    
    run run_cli generate "550e8400-e29b-41d4-a716-446655440001" "social_media" "Create Twitter post" true
    [[ "$status" -eq 0 ]]
    [[ "$output" == *"Generated blog post content"* ]]
}

@test "CLI handles API connection failure gracefully" {
    # Use invalid port
    export CAMPAIGN_CONTENT_STUDIO_API_BASE="http://localhost:99999"
    
    run run_cli health
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Failed to connect to API"* ]]
}

@test "CLI handles unknown command" {
    run run_cli unknown-command
    [[ "$status" -eq 1 ]]
    [[ "$output" == *"Unknown command: unknown-command"* ]]
}