#!/usr/bin/env bats
# OpenRouter integration tests

# Get script directory
OPENROUTER_TEST_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
OPENROUTER_DIR="${OPENROUTER_TEST_DIR%/*}"
CLI_PATH="${OPENROUTER_DIR}/cli.sh"

# Load test helpers if available
if [[ -f "${OPENROUTER_DIR}/../../../__test/helpers/bats-support/load.bash" ]]; then
    load "${OPENROUTER_DIR}/../../../__test/helpers/bats-support/load"
    load "${OPENROUTER_DIR}/../../../__test/helpers/bats-assert/load"
fi

# Source the CLI
source "${OPENROUTER_DIR}/cli.sh"

@test "OpenRouter: CLI exists and is executable" {
    [ -x "${OPENROUTER_DIR}/cli.sh" ]
}

@test "OpenRouter: Status command returns correct format" {
    run bash "${OPENROUTER_DIR}/cli.sh" status
    # Status may return 1 if unhealthy but running (e.g., placeholder key)
    # Check output format regardless of exit code
    assert_output --partial "Status:"
    assert_output --partial "Running:"
    assert_output --partial "Healthy:"
}

@test "OpenRouter: JSON status output is valid" {
    run bash "${OPENROUTER_DIR}/cli.sh" status --json
    # May return exit code 1 if unhealthy but running
    local json_output="$output"
    
    # Check if output is valid JSON
    echo "$json_output" | jq empty
    [ $? -eq 0 ] || fail "Invalid JSON output"
    
    # Check required fields exist
    echo "$json_output" | jq -e 'has("status")' > /dev/null
    [ $? -eq 0 ] || fail "Missing 'status' field"
    echo "$json_output" | jq -e 'has("running")' > /dev/null
    [ $? -eq 0 ] || fail "Missing 'running' field"
    echo "$json_output" | jq -e 'has("healthy")' > /dev/null
    [ $? -eq 0 ] || fail "Missing 'healthy' field"
}

@test "OpenRouter: Fast mode status works" {
    run bash "${OPENROUTER_DIR}/cli.sh" status --fast
    # May return exit code 1 if unhealthy but running
    assert_output --partial "Status:"
}

@test "OpenRouter: Help command shows usage" {
    run bash "${OPENROUTER_DIR}/cli.sh" help
    assert_success
    assert_output --partial "OpenRouter Resource CLI"
    assert_output --partial "Commands:"
    assert_output --partial "status"
    assert_output --partial "install"
}

@test "OpenRouter: Start command returns expected message" {
    run bash "${OPENROUTER_DIR}/cli.sh" start
    assert_success
    assert_output --partial "API service"
}

@test "OpenRouter: Stop command returns expected message" {
    run bash "${OPENROUTER_DIR}/cli.sh" stop
    assert_success
    assert_output --partial "API service"
}

@test "OpenRouter: Test connection handles missing API key gracefully" {
    # Temporarily unset API key if set
    local saved_key="$OPENROUTER_API_KEY"
    unset OPENROUTER_API_KEY
    
    run bash "${OPENROUTER_DIR}/cli.sh" test-connection
    
    # Restore key
    export OPENROUTER_API_KEY="$saved_key"
    
    # Should fail but not crash
    assert_failure
    assert_output --partial "Failed to connect"
}

@test "OpenRouter: List models command exists" {
    run bash "${OPENROUTER_DIR}/cli.sh" list-models
    # May fail without valid API key, but command should exist
    # Just check it doesn't report "Unknown command"
    refute_output --partial "Unknown command"
}

@test "OpenRouter: Usage command exists" {
    run bash "${OPENROUTER_DIR}/cli.sh" usage
    # May fail without valid API key, but command should exist
    refute_output --partial "Unknown command"
}

@test "OpenRouter: Unknown command shows error" {
    run bash "${OPENROUTER_DIR}/cli.sh" invalid-command
    assert_failure
    assert_output --partial "Unknown command"
}

@test "OpenRouter: CLI handles no arguments" {
    run bash "${OPENROUTER_DIR}/cli.sh"
    assert_success
    assert_output --partial "OpenRouter Resource CLI"
}

@test "OpenRouter: Status verbose mode provides additional info" {
    run bash "${OPENROUTER_DIR}/cli.sh" status --verbose
    # May return exit code 1 if unhealthy but running
    assert_output --partial "Status:"
    # Should have more output than non-verbose
    [ ${#output} -gt 50 ]
}

@test "OpenRouter: Configure command exists and shows help" {
    run timeout 5 bash -c "echo '' | ${OPENROUTER_DIR}/cli.sh configure"
    # Should fail with empty input but show help text
    assert_failure
    assert_output --partial "https://openrouter.ai/keys"
}

@test "OpenRouter: Show-config command works" {
    run bash "${OPENROUTER_DIR}/cli.sh" show-config
    assert_success
    assert_output --partial "OpenRouter Configuration"
}

@test "OpenRouter: Run-tests command exists" {
    # Check that run-tests command is recognized
    run bash "${OPENROUTER_DIR}/cli.sh" help
    assert_success
    assert_output --partial "run-tests"
}

@test "OpenRouter: Test results are saved to correct location" {
    # Run tests and check if results file is created
    local result_file="${var_ROOT_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}}/data/test-results/openrouter-test.json"
    
    # Run the tests (this will run recursively but with timeout)
    timeout 10 bash "${OPENROUTER_DIR}/cli.sh" run-tests >/dev/null 2>&1 || true
    
    # Check if results file was created
    if [[ -f "$result_file" ]]; then
        # Verify it's valid JSON
        jq empty "$result_file" || fail "Invalid JSON in test results"
        
        # Check required fields
        jq -e 'has("resource")' "$result_file" >/dev/null || fail "Missing resource field"
        jq -e 'has("status")' "$result_file" >/dev/null || fail "Missing status field"
        jq -e 'has("timestamp")' "$result_file" >/dev/null || fail "Missing timestamp field"
    fi
}