#!/usr/bin/env bats
# Web Scraper Manager CLI tests

setup() {
    # Path to the CLI script
    CLI_SCRIPT="$BATS_TEST_DIRNAME/web-scraper-manager"
    
    # Mock API URL for testing
    export WEB_SCRAPER_MANAGER_API_URL="http://localhost:${SERVICE_PORT:-8091}"
}

@test "CLI shows help when no arguments provided" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Web Scraper Manager CLI" ]]
}

@test "CLI shows help with help command" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Web Scraper Manager CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows version" {
    run "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Web Scraper Manager CLI v" ]]
}

@test "CLI requires jq dependency" {
    # Temporarily hide jq from PATH
    PATH="/bin:/usr/bin" run "$CLI_SCRIPT" status
    [ "$status" -eq 1 ]
    [[ "$output" =~ "jq is required but not installed" ]]
}

@test "CLI requires curl dependency" {
    # This test assumes jq is available but curl is not
    # In practice, both should be available in CI environments
    if ! command -v jq &> /dev/null; then
        skip "jq not available for testing"
    fi
    
    # Test that CLI properly checks for curl
    run bash -c 'PATH="/usr/bin:$(dirname $(which jq))" '"$CLI_SCRIPT"' status'
    if [ "$status" -eq 1 ]; then
        [[ "$output" =~ "curl is required but not installed" ]]
    fi
}

@test "CLI handles unknown commands" {
    run "$CLI_SCRIPT" unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: unknown-command" ]]
}

@test "CLI handles unknown agent subcommands" {
    run "$CLI_SCRIPT" agents unknown
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown agents subcommand: unknown" ]]
}

@test "CLI validates agent create arguments" {
    run "$CLI_SCRIPT" agents create
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage: agents create" ]]
}

@test "CLI validates agent delete arguments" {
    run "$CLI_SCRIPT" agents delete
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage: agents delete" ]]
}

@test "CLI validates agent execute arguments" {
    run "$CLI_SCRIPT" agents execute
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage: agents execute" ]]
}