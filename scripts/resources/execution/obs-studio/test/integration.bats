#!/usr/bin/env bats

# OBS Studio Integration Tests

setup() {
    # Load bats helpers
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    # Set up test variables
    export TEST_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    export OBS_CLI="${TEST_DIR}/../cli.sh"
    export OBS_DATA_DIR="${HOME}/.vrooli/obs-studio"
    
    # Ensure OBS Studio is installed for tests
    if ! "$OBS_CLI" status | grep -q "Installed: true"; then
        "$OBS_CLI" install
    fi
}

@test "OBS Studio CLI exists and is executable" {
    [ -f "$OBS_CLI" ]
    [ -x "$OBS_CLI" ]
}

@test "OBS Studio status returns valid output" {
    run "$OBS_CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OBS Studio Status" ]]
    [[ "$output" =~ "Installed:" ]]
    [[ "$output" =~ "Running:" ]]
    [[ "$output" =~ "Health:" ]]
}

@test "OBS Studio status --json returns valid JSON" {
    run "$OBS_CLI" status --json
    [ "$status" -eq 0 ]
    
    # Validate JSON structure
    echo "$output" | jq '.' >/dev/null
    [ $? -eq 0 ]
    
    # Check required fields
    echo "$output" | jq -e '.status' >/dev/null
    echo "$output" | jq -e '.installed' >/dev/null
    echo "$output" | jq -e '.running' >/dev/null
    echo "$output" | jq -e '.health' >/dev/null
}

@test "OBS Studio can start and stop" {
    # Start OBS Studio
    run "$OBS_CLI" start
    [ "$status" -eq 0 ]
    
    # Check it's running
    run "$OBS_CLI" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Running: true" ]]
    
    # Stop OBS Studio
    run "$OBS_CLI" stop
    [ "$status" -eq 0 ]
}

@test "OBS Studio configuration directory exists" {
    [ -d "$OBS_DATA_DIR" ]
    [ -d "$OBS_DATA_DIR/config" ]
    [ -d "$OBS_DATA_DIR/scenes" ]
    [ -d "$OBS_DATA_DIR/recordings" ]
}

@test "OBS Studio WebSocket configuration exists" {
    "$OBS_CLI" start
    [ -f "$OBS_DATA_DIR/config/websocket.json" ]
    
    # Check WebSocket config has required fields
    run cat "$OBS_DATA_DIR/config/websocket.json"
    [ "$status" -eq 0 ]
    echo "$output" | jq -e '.server_port' >/dev/null
    echo "$output" | jq -e '.server_enabled' >/dev/null
}

@test "OBS Studio help command works" {
    run "$OBS_CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OBS Studio Resource CLI" ]]
    [[ "$output" =~ "Commands:" ]]
    [[ "$output" =~ "Recording Commands:" ]]
    [[ "$output" =~ "Streaming Commands:" ]]
}

@test "OBS Studio list-scenes command works" {
    "$OBS_CLI" start
    run "$OBS_CLI" list-scenes
    [ "$status" -eq 0 ]
}

@test "OBS Studio config command shows configuration" {
    run "$OBS_CLI" config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OBS Studio Configuration:" ]]
    [[ "$output" =~ "Config Directory:" ]]
    [[ "$output" =~ "WebSocket Port:" ]]
}

# Run tests
run_tests() {
    echo "[INFO] Running OBS Studio integration tests..."
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Run bats tests and capture output
    if bats "$TEST_DIR/integration.bats" 2>&1; then
        echo "[SUCCESS] All tests passed"
        test_status="passed"
    else
        echo "[WARNING] Some tests failed"
        test_status="failed"
    fi
    
    # Update test timestamp
    echo "$timestamp" > "$OBS_DATA_DIR/.last-test"
    echo "$test_status" > "$OBS_DATA_DIR/.test-status"
    
    return 0
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_tests
fi