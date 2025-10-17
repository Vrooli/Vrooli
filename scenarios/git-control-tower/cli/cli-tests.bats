#!/usr/bin/env bats
# Tests for git-control-tower CLI

# Test configuration
readonly TEST_CLI="./git-control-tower"
# Force git-control-tower's actual port (18700)
export API_PORT=18700
readonly API_BASE="http://localhost:${API_PORT}"

# Setup - verify API is running
setup() {
    # Export API_PORT for CLI to use (override any existing value)
    export API_PORT=18700

    # Check if API is responding
    if ! curl -sf "${API_BASE}/health" > /dev/null 2>&1; then
        skip "API not running on ${API_BASE}"
    fi
}

# Test: CLI exists and is executable
@test "CLI script exists and is executable" {
    [[ -f "$TEST_CLI" ]]
    [[ -x "$TEST_CLI" ]]
}

# Test: Help command
@test "help command displays usage information" {
    run $TEST_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Git Control Tower" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

# Test: Health command
@test "health command shows service status" {
    run $TEST_CLI health
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]] || [[ "$output" =~ "degraded" ]]
}

# Test: Status command
@test "status command shows repository status" {
    run $TEST_CLI status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Branch:" ]] || [[ "$output" =~ "branch" ]]
}

# Test: Status JSON output
@test "status --json returns valid JSON" {
    run $TEST_CLI status --json
    [ "$status" -eq 0 ]
    # Verify JSON is parseable
    echo "$output" | jq . > /dev/null
}

# Test: Branches command
@test "branches command lists branches" {
    run $TEST_CLI branches
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Branches:" ]] || [[ "$output" =~ "*" ]]
}

# Test: Diff command (without arguments)
@test "diff command without file shows error" {
    run $TEST_CLI diff
    [ "$status" -eq 1 ]
}

# Test: Preview command
@test "preview command shows change summary" {
    run $TEST_CLI preview
    [ "$status" -eq 0 ]
    # Should show either "No staged changes" or actual preview
    [[ "$output" =~ "Change Preview" ]] || [[ "$output" =~ "No staged" ]] || [[ "$output" =~ "Staged Files" ]]
}

# Test: Stage command validation
@test "stage command without files shows error" {
    run $TEST_CLI stage
    [ "$status" -eq 1 ]
}

# Test: Unstage command validation
@test "unstage command without files shows error" {
    run $TEST_CLI unstage
    [ "$status" -eq 1 ]
}

# Test: Commit command validation
@test "commit command without message shows error" {
    run $TEST_CLI commit
    [ "$status" -eq 1 ]
}

# Test: Commit with non-conventional message
@test "commit command validates conventional commit format" {
    # Try to commit with invalid message (should fail validation)
    run $TEST_CLI commit "invalid message format"
    # Command should fail (non-zero exit code)
    [ "$status" -ne 0 ]
}

# Test: Invalid command
@test "invalid command shows error" {
    run $TEST_CLI invalid_command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown" ]] || [[ "$output" =~ "Error" ]]
}

# Test: API connectivity
@test "CLI can connect to API" {
    run $TEST_CLI health
    [ "$status" -eq 0 ]
}
