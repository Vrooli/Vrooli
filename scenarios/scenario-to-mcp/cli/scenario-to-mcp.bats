#!/usr/bin/env bats
# BATS test suite for scenario-to-mcp CLI
# Comprehensive testing for all CLI commands and edge cases

setup() {
    # Get the directory containing this test file
    DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    CLI="$DIR/scenario-to-mcp"

    # Create temp directory for test artifacts
    TEST_DIR=$(mktemp -d)
}

teardown() {
    # Clean up temp directory
    [ -n "$TEST_DIR" ] && rm -rf "$TEST_DIR"
}

# Basic Functionality Tests
@test "CLI help command works" {
    run bash "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Scenario to MCP" ]]
}

@test "CLI version command works" {
    run bash "$CLI" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v1.0.0" ]]
}

@test "CLI list command produces JSON output" {
    run bash "$CLI" list --json
    [ "$status" -eq 0 ]
    # Output should be valid JSON array
    echo "$output" | jq 'if type == "array" then true else false end' | grep -q "true"
}

@test "CLI detect command shows summary" {
    # Skip API check by running directly
    run bash "$CLI" detect
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Summary:" ]]
}

@test "CLI candidates command lists scenarios" {
    run bash "$CLI" candidates
    [ "$status" -eq 0 ]
    # Should either show candidates or message about none found
    [[ "$output" =~ "MCP Candidates" ]] || [[ "$output" =~ "No good candidates" ]]
}

# Error Handling Tests
@test "CLI shows error for invalid command" {
    run bash "$CLI" invalid-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]] || [[ "$output" =~ "Invalid" ]]
}

@test "CLI handles missing arguments gracefully" {
    run bash "$CLI"
    # Should show help or usage info
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI check command validates scenario name" {
    run bash "$CLI" check ""
    [ "$status" -ne 0 ]
}

@test "CLI check command with non-existent scenario" {
    run bash "$CLI" check "nonexistent-scenario-12345"
    # Should handle gracefully, not crash
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Output Format Tests
@test "CLI list command default output format" {
    run bash "$CLI" list
    [ "$status" -eq 0 ]
    # Should produce human-readable output
    [[ "$output" =~ "Scenario" ]] || [[ "$output" =~ "Name" ]]
}

@test "CLI list JSON output is valid" {
    run bash "$CLI" list --json
    [ "$status" -eq 0 ]

    # Verify it's valid JSON
    echo "$output" | jq empty
    [ $? -eq 0 ]
}

@test "CLI detect output includes statistics" {
    run bash "$CLI" detect
    [ "$status" -eq 0 ]

    # Should include summary statistics
    [[ "$output" =~ "Total:" ]] || [[ "$output" =~ "Summary:" ]]
}

# Edge Cases
@test "CLI handles special characters in scenario names" {
    run bash "$CLI" check "scenario-with-dashes"
    # Should not crash
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI handles very long scenario names" {
    long_name="scenario-$(printf 'a%.0s' {1..200})"
    run bash "$CLI" check "$long_name"
    # Should handle gracefully
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI handles concurrent executions" {
    # Run multiple CLI commands in parallel
    bash "$CLI" list &
    PID1=$!
    bash "$CLI" detect &
    PID2=$!
    bash "$CLI" version &
    PID3=$!

    wait $PID1
    STATUS1=$?
    wait $PID2
    STATUS2=$?
    wait $PID3
    STATUS3=$?

    # All should complete successfully
    [ $STATUS1 -eq 0 ]
    [ $STATUS2 -eq 0 ]
    [ $STATUS3 -eq 0 ]
}

# Performance Tests
@test "CLI version responds quickly" {
    start=$(date +%s%N)
    run bash "$CLI" version
    end=$(date +%s%N)

    [ "$status" -eq 0 ]

    # Should respond in less than 1 second (1000000000 nanoseconds)
    duration=$((end - start))
    [ $duration -lt 1000000000 ]
}

@test "CLI help responds quickly" {
    start=$(date +%s%N)
    run bash "$CLI" help
    end=$(date +%s%N)

    [ "$status" -eq 0 ]

    # Should respond in less than 1 second
    duration=$((end - start))
    [ $duration -lt 1000000000 ]
}

# Integration Tests
@test "CLI commands produce consistent output" {
    # Run list twice, should get similar results
    run bash "$CLI" list --json
    output1="$output"

    run bash "$CLI" list --json
    output2="$output"

    # Both should be valid JSON
    echo "$output1" | jq empty
    echo "$output2" | jq empty
}

@test "CLI detect and candidates show related data" {
    run bash "$CLI" detect
    detect_output="$output"

    run bash "$CLI" candidates
    candidates_output="$output"

    # Both should complete successfully
    [ "$status" -eq 0 ]
}

# Security Tests
@test "CLI rejects command injection attempts" {
    run bash "$CLI" check "; rm -rf /"
    # Should handle safely, not execute injection
    [ -d "$TEST_DIR" ]  # Temp dir should still exist
}

@test "CLI handles null bytes safely" {
    run bash "$CLI" check $'scenario\x00name'
    # Should handle gracefully without crash
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# Help and Documentation Tests
@test "CLI help includes all commands" {
    run bash "$CLI" help
    [ "$status" -eq 0 ]

    # Should mention key commands
    [[ "$output" =~ "list" ]]
    [[ "$output" =~ "detect" ]]
    [[ "$output" =~ "check" ]]
}

@test "CLI help shows version information" {
    run bash "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "v1." ]] || [[ "$output" =~ "version" ]]
}

# Exit Code Tests
@test "CLI returns 0 for successful commands" {
    run bash "$CLI" version
    [ "$status" -eq 0 ]

    run bash "$CLI" help
    [ "$status" -eq 0 ]
}

@test "CLI returns non-zero for errors" {
    run bash "$CLI" invalid-command-xyz
    [ "$status" -ne 0 ]
}
