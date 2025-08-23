#!/usr/bin/env bats
# App Personalizer CLI Tests

setup() {
    # Set test environment
    export APP_PERSONALIZER_API_BASE="http://localhost:${SERVICE_PORT:-8300}"
    CLI_SCRIPT="$(dirname "$BATS_TEST_FILENAME")/app-personalizer-cli.sh"
}

@test "CLI shows version" {
    run bash "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "app-personalizer CLI version" ]]
}

@test "CLI shows help" {
    run bash "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "App Personalizer CLI" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI handles unknown command" {
    run bash "$CLI_SCRIPT" unknown_command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI shows API base URL" {
    run bash "$CLI_SCRIPT" api
    [ "$status" -eq 0 ]
    [[ "$output" =~ "http://localhost:" ]]
}

@test "CLI requires parameters for register command" {
    run bash "$CLI_SCRIPT" register
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI requires parameters for analyze command" {
    run bash "$CLI_SCRIPT" analyze
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI requires parameters for personalize command" {
    run bash "$CLI_SCRIPT" personalize
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Types:" ]]
}

@test "CLI requires parameters for backup command" {
    run bash "$CLI_SCRIPT" backup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI requires parameters for validate command" {
    run bash "$CLI_SCRIPT" validate
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

# Integration tests (run only if API is available)
@test "CLI can check health if API is running" {
    if curl -sf "$APP_PERSONALIZER_API_BASE/health" >/dev/null 2>&1; then
        run bash "$CLI_SCRIPT" health
        [ "$status" -eq 0 ]
        [[ "$output" =~ "status" ]]
    else
        skip "API not running"
    fi
}

@test "CLI can list apps if API is running" {
    if curl -sf "$APP_PERSONALIZER_API_BASE/health" >/dev/null 2>&1; then
        run bash "$CLI_SCRIPT" list
        [ "$status" -eq 0 ]
        # Output should be JSON array (empty or with apps)
        [[ "$output" =~ "\[" ]]
    else
        skip "API not running"
    fi
}