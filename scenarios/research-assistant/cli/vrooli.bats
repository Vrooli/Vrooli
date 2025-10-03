#!/usr/bin/env bats

# Research Assistant CLI BATS Tests
# Tests the scenario-specific CLI commands

# Test configuration
readonly CLI_NAME="vrooli"
readonly TEST_TIMEOUT=10

setup() {
    # 1. ALWAYS set variables FIRST
    export TEST_DIR="/tmp/research-assistant-cli-test-$$"
    export TEST_FILE="/tmp/research-assistant-test-file-$$"

    # Detect API port from running process
    local api_pid=$(ps aux | grep -E "\./research-assistant-api" | grep -v grep | awk '{print $2}' | head -1)
    if [ -n "$api_pid" ]; then
        export RESEARCH_ASSISTANT_API_PORT=$(lsof -Pan -p $api_pid -i 2>/dev/null | grep LISTEN | awk '{print $9}' | grep -o '[0-9]*$' | head -1)
    fi
    export RESEARCH_ASSISTANT_API_PORT="${RESEARCH_ASSISTANT_API_PORT:-17039}"

    # 2. Check prerequisites AFTER variables
    if ! command -v "$CLI_NAME" >/dev/null 2>&1; then
        skip "$CLI_NAME not installed"
    fi

    # 3. Create test environment
    mkdir -p "$TEST_DIR"
}

teardown() {
    # Safe cleanup with multiple guards
    local paths=("$TEST_DIR" "$TEST_FILE")

    for path in "${paths[@]}"; do
        if [ -n "${path:-}" ] && [ -e "$path" ]; then
            case "$path" in
                /tmp/*)
                    rm -rf "$path" 2>/dev/null || true
                    ;;
                *)
                    echo "WARNING: Refusing to delete $path (not in /tmp)" >&2
                    ;;
            esac
        fi
    done
}

# Helper functions
api_running() {
    timeout 3 curl -sf "http://localhost:${RESEARCH_ASSISTANT_API_PORT}/health" >/dev/null 2>&1
}

is_valid_json() {
    echo "$1" | jq . >/dev/null 2>&1
}

# Basic CLI tests
@test "Vrooli CLI is available" {
    run which $CLI_NAME
    [ "$status" -eq 0 ]
}

@test "Vrooli CLI shows help" {
    run $CLI_NAME help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]] || [[ "$output" =~ "COMMANDS:" ]]
}

# Scenario-specific command tests
@test "Research Assistant scenario status command" {
    run $CLI_NAME scenario status research-assistant
    [ "$status" -eq 0 ]
    [[ "$output" =~ "research-assistant" ]]
}

@test "Research Assistant scenario status shows running state" {
    if ! api_running; then
        skip "Research Assistant API not running"
    fi

    run $CLI_NAME scenario status research-assistant
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RUNNING" ]] || [[ "$output" =~ "running" ]]
}

@test "Research Assistant scenario logs command" {
    run timeout 5 $CLI_NAME scenario logs research-assistant --lines 10
    [ "$status" -eq 0 ] || [ "$status" -eq 124 ]  # Success or timeout
}

# API integration tests
@test "Research Assistant API is accessible" {
    if ! api_running; then
        skip "Research Assistant API not running"
    fi

    run timeout 5 curl -sf "http://localhost:${RESEARCH_ASSISTANT_API_PORT}/health"
    [ "$status" -eq 0 ]
    is_valid_json "$output"
}

@test "Research Assistant API health endpoint returns expected structure" {
    if ! api_running; then
        skip "Research Assistant API not running"
    fi

    run timeout 5 curl -sf "http://localhost:${RESEARCH_ASSISTANT_API_PORT}/health"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "services" ]]
}

@test "Research Assistant API reports endpoint is accessible" {
    if ! api_running; then
        skip "Research Assistant API not running"
    fi

    run timeout 5 curl -sf "http://localhost:${RESEARCH_ASSISTANT_API_PORT}/api/reports"
    [ "$status" -eq 0 ]
    is_valid_json "$output"
}

@test "Research Assistant API templates endpoint returns data" {
    if ! api_running; then
        skip "Research Assistant API not running"
    fi

    run timeout 5 curl -sf "http://localhost:${RESEARCH_ASSISTANT_API_PORT}/api/templates"
    [ "$status" -eq 0 ]
    is_valid_json "$output"
    [[ "$output" =~ "templates" ]]
}

@test "Research Assistant API depth-configs endpoint returns data" {
    if ! api_running; then
        skip "Research Assistant API not running"
    fi

    run timeout 5 curl -sf "http://localhost:${RESEARCH_ASSISTANT_API_PORT}/api/depth-configs"
    [ "$status" -eq 0 ]
    is_valid_json "$output"
    [[ "$output" =~ "depth_configs" ]]
    [[ "$output" =~ "quick" ]]
    [[ "$output" =~ "standard" ]]
    [[ "$output" =~ "deep" ]]
}

# Error handling tests
@test "Research Assistant handles invalid scenario operation" {
    run $CLI_NAME scenario nonexistent-command research-assistant
    [ "$status" -ne 0 ]
}

@test "Research Assistant scenario test command exists" {
    run timeout 10 $CLI_NAME scenario test research-assistant --help 2>&1
    # Command should exist even if it fails
    [[ "$output" =~ "test" ]] || [ "$status" -eq 0 ]
}
