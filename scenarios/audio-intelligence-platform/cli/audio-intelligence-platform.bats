#!/usr/bin/env bats
# Audio Intelligence Platform CLI tests

# Setup test environment
setup() {
    # Point to local CLI script
    CLI_SCRIPT="$BATS_TEST_DIRNAME/audio-intelligence-platform-cli.sh"
    export AUDIO_INTELLIGENCE_PLATFORM_API_BASE="http://localhost:${API_PORT:-8090}"
}

@test "CLI shows version" {
    run "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "audio-intelligence-platform CLI version" ]]
}

@test "CLI shows help" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Audio Intelligence Platform CLI" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI handles unknown command" {
    run "$CLI_SCRIPT" unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI shows API base URL" {
    run "$CLI_SCRIPT" api
    [ "$status" -eq 0 ]
    [[ "$output" =~ "http://localhost:" ]]
}

@test "CLI requires arguments for upload" {
    run "$CLI_SCRIPT" upload
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI requires arguments for analyze" {
    run "$CLI_SCRIPT" analyze
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI requires arguments for search" {
    run "$CLI_SCRIPT" search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Usage:" ]]
}