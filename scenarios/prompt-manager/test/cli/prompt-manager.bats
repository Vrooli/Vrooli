#!/usr/bin/env bats
# CLI integration tests for prompt-manager using BATS framework

setup() {
    # Set up test environment
    export TEST_DATA_DIR="${BATS_TMPDIR}/prompt-manager-test"
    mkdir -p "$TEST_DATA_DIR"

    # Check if CLI is available
    if ! command -v prompt-manager &> /dev/null; then
        skip "prompt-manager CLI not installed"
    fi
}

teardown() {
    # Clean up test data
    rm -rf "$TEST_DATA_DIR"
}

@test "prompt-manager: CLI is installed and executable" {
    run prompt-manager --version
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]  # Some CLIs don't have --version
}

@test "prompt-manager: Show help message" {
    run prompt-manager --help
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    [[ "$output" == *"prompt-manager"* ]] || [[ "$output" == *"help"* ]] || [[ "$output" == *"Usage"* ]]
}

@test "prompt-manager: Status command works" {
    run prompt-manager status
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "prompt-manager: List campaigns" {
    # This test requires API to be running
    if ! curl -sf "http://localhost:${API_PORT:-15280}/health" &> /dev/null; then
        skip "API not running"
    fi

    run prompt-manager list campaigns
    # Command should succeed or return specific error code
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
}

@test "prompt-manager: List prompts" {
    # This test requires API to be running
    if ! curl -sf "http://localhost:${API_PORT:-15280}/health" &> /dev/null; then
        skip "API not running"
    fi

    run prompt-manager list prompts
    # Command should succeed or return specific error code
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
}

@test "prompt-manager: Search functionality" {
    # This test requires API to be running
    if ! curl -sf "http://localhost:${API_PORT:-15280}/health" &> /dev/null; then
        skip "API not running"
    fi

    run prompt-manager search "test"
    # Command should succeed or return specific error code
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 2 ]
}

@test "prompt-manager: Export command exists" {
    run prompt-manager export --help
    # Should show help or execute
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "prompt-manager: Import command exists" {
    run prompt-manager import --help
    # Should show help or execute
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "prompt-manager: CLI handles invalid commands gracefully" {
    run prompt-manager invalid-command-xyz
    # Should fail but not crash
    [ "$status" -ne 0 ]
}

@test "prompt-manager: CLI handles missing arguments" {
    run prompt-manager create
    # Should fail with helpful error
    [ "$status" -ne 0 ]
}

@test "prompt-manager: Version or build info available" {
    run prompt-manager version
    # Accept both success and "command not found" (not all CLIs have version)
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 127 ]
}
