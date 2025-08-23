#\!/usr/bin/env bats
# BATS test suite for App Monitor CLI

setup() {
    # Test configuration
    export APP_MONITOR_API_URL="${APP_MONITOR_API_URL:-http://localhost:8090}"
    
    # Check if API is available
    if \! curl -sf "$APP_MONITOR_API_URL/health" > /dev/null 2>&1; then
        skip "App Monitor API not available for testing"
    fi
}

@test "CLI displays help when no arguments provided" {
    run ./app-monitor
    [ "$status" -eq 0 ]
    [[ "$output" == *"App Monitor CLI"* ]]
    [[ "$output" == *"Usage:"* ]]
}

@test "CLI displays help with help command" {
    run ./app-monitor help
    [ "$status" -eq 0 ]
    [[ "$output" == *"App Monitor CLI"* ]]
    [[ "$output" == *"Commands:"* ]]
}

@test "CLI status command works" {
    run ./app-monitor status
    [ "$status" -eq 0 ]
    [[ "$output" == *"App Monitor"* ]]
}

@test "CLI list command works" {
    run ./app-monitor list
    [ "$status" -eq 0 ]
    # Should either show apps or indicate no apps
    [[ "$output" == *"apps"* ]] || [[ "$output" == *"No apps"* ]]
}

@test "CLI docker command works" {
    run ./app-monitor docker
    [ "$status" -eq 0 ]
    [[ "$output" == *"Containers"* ]] || [[ "$output" == *"Server Version"* ]]
}

@test "CLI fails gracefully with invalid command" {
    run ./app-monitor invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown command"* ]]
}

@test "CLI start command requires app ID" {
    run ./app-monitor start
    [ "$status" -eq 1 ]
    [[ "$output" == *"App ID required"* ]]
}

@test "CLI stop command requires app ID" {
    run ./app-monitor stop
    [ "$status" -eq 1 ]
    [[ "$output" == *"App ID required"* ]]
}

@test "CLI logs command requires app ID" {
    run ./app-monitor logs
    [ "$status" -eq 1 ]
    [[ "$output" == *"App ID required"* ]]
}

@test "CLI metrics command requires app ID" {
    run ./app-monitor metrics
    [ "$status" -eq 1 ]
    [[ "$output" == *"App ID required"* ]]
}
EOF < /dev/null
