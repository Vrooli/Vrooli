#!/usr/bin/env bats

# Wiki.js Integration Tests

setup() {
    # Set test directory
    export TEST_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
    export WIKIJS_CLI="$TEST_DIR/../cli.sh"
    
    # Skip tests if Wiki.js is not installed
    if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Installed: Yes"; then
        skip "Wiki.js not installed"
    fi
}

@test "Wiki.js CLI exists and is executable" {
    [ -f "$WIKIJS_CLI" ]
    [ -x "$WIKIJS_CLI" ]
}

@test "Wiki.js shows help" {
    run $WIKIJS_CLI help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Wiki.js Resource Manager" ]]
}

@test "Wiki.js status command works" {
    run $WIKIJS_CLI status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Wiki.js Status" ]]
}

@test "Wiki.js status JSON mode works" {
    run $WIKIJS_CLI status --json
    [ "$status" -eq 0 ]
    # Check for valid JSON
    echo "$output" | jq . >/dev/null
}

@test "Wiki.js health check" {
    # Skip if not running
    if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Running: Yes"; then
        skip "Wiki.js not running"
    fi
    
    run $WIKIJS_CLI status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Health:" ]]
}

@test "Wiki.js version check" {
    # Skip if not running
    if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Running: Yes"; then
        skip "Wiki.js not running"
    fi
    
    run $WIKIJS_CLI status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Version:" ]]
}

@test "Wiki.js API endpoint accessible" {
    # Skip if not running
    if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Running: Yes"; then
        skip "Wiki.js not running"
    fi
    
    run $WIKIJS_CLI status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "API Endpoint:" ]]
}

@test "Wiki.js test command runs" {
    # Skip if not running
    if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Running: Yes"; then
        skip "Wiki.js not running"
    fi
    
    run $WIKIJS_CLI test
    # Test command may fail some tests, but should run
    [[ "$output" =~ "Test Results:" ]]
}

@test "Wiki.js logs command works" {
    # Skip if not running
    if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Running: Yes"; then
        skip "Wiki.js not running"
    fi
    
    run timeout 5 $WIKIJS_CLI logs --lines 5
    [ "$status" -eq 0 ] || [ "$status" -eq 124 ]  # Success or timeout
}

@test "Wiki.js dry-run mode works" {
    run $WIKIJS_CLI start --dry-run
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    [[ "$output" =~ "DRY-RUN" ]] || [[ "$output" =~ "already running" ]]
}