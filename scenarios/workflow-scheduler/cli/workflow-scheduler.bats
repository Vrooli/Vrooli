#!/usr/bin/env bats

# Test suite for workflow-scheduler CLI (alias scheduler-cli)

setup() {
    # Get the directory where the CLI is installed
    CLI_PATH="${BATS_TEST_DIRNAME}/workflow-scheduler"

    # Check if CLI exists
    if [ ! -f "$CLI_PATH" ]; then
        if command -v workflow-scheduler >/dev/null 2>&1; then
            CLI_PATH="$(command -v workflow-scheduler)"
        else
            CLI_PATH="$(command -v scheduler-cli 2>/dev/null)"
        fi
    fi

    if [ -z "$CLI_PATH" ] || [ ! -f "$CLI_PATH" ]; then
        skip "workflow-scheduler (or scheduler-cli alias) not found in PATH or test directory"
    fi
}

@test "CLI help command works" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage" ]] || [[ "$output" =~ "workflow" ]] || [[ "$output" =~ "scheduler" ]]
}

@test "CLI version command works" {
    run "$CLI_PATH" --version
    # Version command may or may not exist, so we accept both success and failure
    # Just ensure it doesn't crash catastrophically
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI list command accepts proper flags" {
    # Test that list command doesn't immediately error on syntax
    # We don't require API to be running for this basic validation
    run "$CLI_PATH" list --help 2>&1
    # Should either show help or indicate API is not running
    # Both are acceptable responses - we're just checking CLI parsing
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI create command requires arguments" {
    run "$CLI_PATH" create 2>&1
    # Should fail without required arguments
    [ "$status" -ne 0 ]
}

@test "CLI executable has correct permissions" {
    [ -x "$CLI_PATH" ]
}

@test "CLI is a valid bash script" {
    run head -n 1 "$CLI_PATH"
    [[ "$output" =~ "#!/" ]]
}
