#\!/usr/bin/env bats

# Setup function that runs before each test
setup() {
    # Set up test environment
    CLI_SCRIPT="$BATS_TEST_DIRNAME/smart-file-photo-manager"
    export SMART_FILE_MANAGER_API_URL="http://localhost:8090"
}

@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI shows help when no arguments provided" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: help" ]] || [[ "$output" =~ "Smart File Photo Manager CLI" ]]
}

@test "CLI shows help with --help flag" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Smart File Photo Manager CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
}

@test "CLI shows version with --version flag" {
    run "$CLI_SCRIPT" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Smart File Photo Manager CLI v" ]]
}

@test "CLI help command works" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Smart File Photo Manager CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI validates unknown commands" {
    run "$CLI_SCRIPT" nonexistent-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command: nonexistent-command" ]]
}

@test "CLI files command exists" {
    # This test will fail when API is not available, but validates command structure
    run "$CLI_SCRIPT" files list --help
    # We expect this to fail due to API unavailability, but not due to unknown command
    [ "$status" -ne 0 ]
    [[ \! "$output" =~ "Unknown command" ]]
}

@test "CLI search command exists" {
    run "$CLI_SCRIPT" search text "test" --help 2>/dev/null || true
    # Command should be recognized even if API fails
    [[ \! "$output" =~ "Unknown command" ]]
}

@test "CLI accepts API URL override" {
    run "$CLI_SCRIPT" --api-url "http://example.com:9999" health
    [ "$status" -ne 0 ]  # Will fail due to unavailable API, but command should be recognized
    [[ \! "$output" =~ "Unknown global option" ]]
}
