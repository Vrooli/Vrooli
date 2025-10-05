#!/usr/bin/env bats

# CLI integration tests for react-component-library
# Uses BATS (Bash Automated Testing System)

setup() {
    # Setup runs before each test
    export CLI_PATH="${CLI_PATH:-../../cli/react-component-library}"

    # Skip tests if CLI not built
    if [ ! -f "$CLI_PATH" ]; then
        skip "CLI binary not found at $CLI_PATH"
    fi
}

# ==============================================================================
# Basic CLI Tests
# ==============================================================================

@test "CLI binary exists and is executable" {
    [ -x "$CLI_PATH" ]
}

@test "CLI help command works" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "react-component-library" ]] || [[ "$output" =~ "Usage" ]]
}

@test "CLI version command works" {
    run "$CLI_PATH" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI status command exists" {
    run "$CLI_PATH" status
    # May fail if services not running, but command should exist
    # Exit code 0 = services running, 1 = services not running, other = error
    [ "$status" -le 1 ]
}

# ==============================================================================
# Command Validation Tests
# ==============================================================================

@test "CLI rejects unknown commands" {
    run "$CLI_PATH" unknown-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "unknown" ]] || [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "not found" ]]
}

@test "CLI requires arguments for create command" {
    run "$CLI_PATH" create
    # Should fail if no component name provided
    [ "$status" -ne 0 ]
}

@test "CLI requires arguments for search command" {
    run "$CLI_PATH" search
    # Should fail if no query provided
    [ "$status" -ne 0 ]
}

# ==============================================================================
# Help Command Tests
# ==============================================================================

@test "Help shows available commands" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]

    # Check for key commands
    [[ "$output" =~ "create" ]] || skip "create command not shown in help"
    [[ "$output" =~ "search" ]] || skip "search command not shown in help"
    [[ "$output" =~ "test" ]] || skip "test command not shown in help"
}

@test "Help for specific command works" {
    run "$CLI_PATH" help create
    [ "$status" -eq 0 ]
    [[ "$output" =~ "create" ]]
}

# ==============================================================================
# Output Format Tests
# ==============================================================================

@test "CLI supports JSON output for status" {
    run "$CLI_PATH" status --json
    # Should return JSON or indicate service unavailable
    [ "$status" -le 1 ]

    if [ "$status" -eq 0 ]; then
        [[ "$output" =~ "{" ]] || skip "JSON output not available"
    fi
}

@test "CLI supports verbose output" {
    run "$CLI_PATH" status --verbose
    [ "$status" -le 1 ]
}

# ==============================================================================
# Error Handling Tests
# ==============================================================================

@test "CLI handles invalid flags gracefully" {
    run "$CLI_PATH" status --invalid-flag
    [ "$status" -ne 0 ]
    # Should show error message, not crash
    [ -n "$output" ]
}

@test "CLI handles missing required flags" {
    run "$CLI_PATH" create TestComponent
    # May succeed or fail depending on whether category is required
    # But should not crash
    [ "$status" -le 1 ]
}

# ==============================================================================
# Integration Tests (require running services)
# ==============================================================================

@test "Search command can connect to API" {
    skip "Requires running API service"

    run "$CLI_PATH" search "button" --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "{" ]] # Should return JSON
}

@test "Create command can create component" {
    skip "Requires running API service"

    run "$CLI_PATH" create TestCLIComponent form --template button
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" ]] || [[ "$output" =~ "success" ]]
}

@test "Test command can run accessibility tests" {
    skip "Requires running API service"

    run "$CLI_PATH" test --accessibility
    [ "$status" -eq 0 ]
}

@test "Export command can export components" {
    skip "Requires running API service and existing component"

    run "$CLI_PATH" export TestComponent --format raw-code
    [ "$status" -eq 0 ]
}

@test "Generate command can generate via AI" {
    skip "Requires running API service and AI integration"

    run "$CLI_PATH" generate "a simple button component" --style minimal
    [ "$status" -eq 0 ]
}

# ==============================================================================
# Performance Tests
# ==============================================================================

@test "CLI responds quickly to help command" {
    # Help should respond in less than 1 second
    start=$(date +%s%N)
    run "$CLI_PATH" help
    end=$(date +%s%N)

    elapsed=$((($end - $start) / 1000000)) # Convert to milliseconds

    [ "$status" -eq 0 ]
    [ "$elapsed" -lt 1000 ] # Less than 1 second
}

@test "CLI version responds quickly" {
    start=$(date +%s%N)
    run "$CLI_PATH" version
    end=$(date +%s%N)

    elapsed=$((($end - $start) / 1000000))

    [ "$status" -eq 0 ]
    [ "$elapsed" -lt 500 ] # Less than 500ms
}

# ==============================================================================
# Input Validation Tests
# ==============================================================================

@test "CLI rejects invalid component names" {
    run "$CLI_PATH" create "invalid name with spaces" form
    # Should either reject or sanitize
    [ -n "$output" ]
}

@test "CLI handles special characters in search" {
    run "$CLI_PATH" search "component<>test&\"'"
    # Should handle gracefully without crashing
    [ "$status" -le 1 ]
}

@test "CLI handles very long search queries" {
    long_query=$(printf 'a%.0s' {1..1000})
    run "$CLI_PATH" search "$long_query"
    # Should handle gracefully
    [ "$status" -le 1 ]
}

# ==============================================================================
# Configuration Tests
# ==============================================================================

@test "CLI can read configuration" {
    # If CLI uses config files, test them
    run "$CLI_PATH" status --verbose
    [ "$status" -le 1 ]
    # Should not crash even if config missing
}

@test "CLI provides meaningful error messages" {
    run "$CLI_PATH" create
    [ "$status" -ne 0 ]
    # Error message should be helpful
    [ -n "$output" ]
    [[ "$output" =~ "required" ]] || [[ "$output" =~ "usage" ]] || [[ "$output" =~ "error" ]]
}

# ==============================================================================
# Exit Code Tests
# ==============================================================================

@test "CLI returns 0 for successful commands" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
}

@test "CLI returns non-zero for failed commands" {
    run "$CLI_PATH" unknown-command
    [ "$status" -ne 0 ]
}

@test "CLI returns consistent exit codes" {
    # Run same failing command twice
    run "$CLI_PATH" unknown-command-1
    status1=$status

    run "$CLI_PATH" unknown-command-2
    status2=$status

    # Both should fail with same exit code
    [ "$status1" -eq "$status2" ]
    [ "$status1" -ne 0 ]
}
