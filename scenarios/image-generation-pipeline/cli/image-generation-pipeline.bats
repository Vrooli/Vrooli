#!/usr/bin/env bats
# Tests for Image Generation Pipeline CLI

setup() {
    # Set test environment
    export IMAGE_GENERATION_PIPELINE_API_BASE="http://localhost:${API_PORT:-8090}"
    
    # CLI location
    CLI_SCRIPT="$BATS_TEST_DIRNAME/image-generation-pipeline"
    
    # Skip if service is not running
    if ! curl -sf "$IMAGE_GENERATION_PIPELINE_API_BASE/health" >/dev/null 2>&1; then
        skip "Image Generation Pipeline API is not running"
    fi
}

@test "CLI shows help message" {
    run bash "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Image Generation Pipeline CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI status command works" {
    run bash "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Service is healthy" ]]
}

@test "CLI can list campaigns" {
    run bash "$CLI_SCRIPT" campaigns
    [ "$status" -eq 0 ]
    # Should return JSON array (even if empty)
    [[ "$output" =~ "\[" ]]
}

@test "CLI can list brands" {
    run bash "$CLI_SCRIPT" brands
    [ "$status" -eq 0 ]
    # Should return JSON array (even if empty)
    [[ "$output" =~ "\[" ]]
}

@test "CLI can list generations" {
    run bash "$CLI_SCRIPT" generations
    [ "$status" -eq 0 ]
    # Should return JSON array (even if empty)
    [[ "$output" =~ "\[" ]]
}

@test "CLI handles unknown commands gracefully" {
    run bash "$CLI_SCRIPT" nonexistent-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI can create and list a brand" {
    # Create a brand
    run bash "$CLI_SCRIPT" brands create "Test Brand CLI" "Test brand description"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Brand created successfully" ]]
    
    # Verify it appears in the list
    run bash "$CLI_SCRIPT" brands
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Test Brand CLI" ]]
}

@test "CLI can create and list a campaign" {
    # First create a brand to reference
    run bash "$CLI_SCRIPT" brands create "Test Brand for Campaign" "Brand for testing campaigns"
    [ "$status" -eq 0 ]
    
    # Extract brand ID from response (assuming jq is available)
    if command -v jq >/dev/null 2>&1; then
        brand_id=$(echo "$output" | tail -n 1 | jq -r '.id')
        
        # Create a campaign
        run bash "$CLI_SCRIPT" campaigns create "Test Campaign CLI" "$brand_id" "Test campaign description"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Campaign created successfully" ]]
        
        # Verify it appears in the list
        run bash "$CLI_SCRIPT" campaigns
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Test Campaign CLI" ]]
    else
        skip "jq not available for JSON parsing"
    fi
}

@test "CLI handles invalid brand creation" {
    run bash "$CLI_SCRIPT" brands create
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI handles invalid campaign creation" {
    run bash "$CLI_SCRIPT" campaigns create
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI handles invalid generation request" {
    run bash "$CLI_SCRIPT" generate
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI handles missing voice brief file" {
    run bash "$CLI_SCRIPT" voice-brief --file nonexistent.wav
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}
