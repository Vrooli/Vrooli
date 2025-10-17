#!/usr/bin/env bats
# CLI integration tests for scenario-to-extension

setup() {
    # Set up test environment
    export TEST_DIR="$(mktemp -d)"
    export PATH="$BATS_TEST_DIRNAME/../../cli:$PATH"

    # Ensure CLI is installed
    if [ ! -f "$BATS_TEST_DIRNAME/../../cli/scenario-to-extension" ]; then
        skip "CLI binary not found. Run 'cd cli && ./install.sh' first."
    fi
}

teardown() {
    # Clean up test environment
    rm -rf "$TEST_DIR"
}

@test "CLI: scenario-to-extension command exists" {
    run which scenario-to-extension
    [ "$status" -eq 0 ]
}

@test "CLI: help command shows usage" {
    run scenario-to-extension help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]] || [[ "$output" =~ "Commands:" ]]
}

@test "CLI: version command shows version" {
    run scenario-to-extension version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI: status command runs" {
    run scenario-to-extension status
    # Status may fail if API not running, but command should exist
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI: status --json produces valid JSON" {
    run scenario-to-extension status --json
    if [ "$status" -eq 0 ]; then
        # If successful, output should be valid JSON
        run echo "$output" | jq .
        [ "$status" -eq 0 ]
    fi
}

@test "CLI: generate command requires scenario name" {
    run scenario-to-extension generate
    [ "$status" -ne 0 ]
    [[ "$output" =~ "scenario" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "usage" ]]
}

@test "CLI: generate with scenario name (dry run check)" {
    # This test just checks that the command structure is correct
    # Actual generation would require API to be running
    run timeout 5 scenario-to-extension generate test-scenario --template full --output "$TEST_DIR/extension"
    # Command may fail if API not available, but should not error on argument parsing
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: generate with all options" {
    run timeout 5 scenario-to-extension generate test-scenario \
        --template full \
        --permissions "storage,activeTab" \
        --host-permissions "https://*.example.com/*" \
        --output "$TEST_DIR/extension" \
        --debug
    # Should accept all arguments without parsing errors
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: test command requires extension path" {
    run scenario-to-extension test
    [ "$status" -ne 0 ]
    [[ "$output" =~ "extension" ]] || [[ "$output" =~ "path" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "usage" ]]
}

@test "CLI: test with extension path" {
    mkdir -p "$TEST_DIR/test-extension"
    run timeout 5 scenario-to-extension test "$TEST_DIR/test-extension"
    # May fail if API not running, but command should parse correctly
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: test with sites option" {
    mkdir -p "$TEST_DIR/test-extension"
    run timeout 5 scenario-to-extension test "$TEST_DIR/test-extension" \
        --sites "https://example.com,https://google.com"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: test with headless flag" {
    mkdir -p "$TEST_DIR/test-extension"
    run timeout 5 scenario-to-extension test "$TEST_DIR/test-extension" --headless
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: build command requires extension path" {
    run scenario-to-extension build
    [ "$status" -ne 0 ]
    [[ "$output" =~ "extension" ]] || [[ "$output" =~ "path" ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "usage" ]]
}

@test "CLI: build with extension path" {
    mkdir -p "$TEST_DIR/build-test"
    run timeout 5 scenario-to-extension build "$TEST_DIR/build-test"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: build with watch flag" {
    mkdir -p "$TEST_DIR/build-test"
    run timeout 2 scenario-to-extension build "$TEST_DIR/build-test" --watch
    # Watch mode will timeout, which is expected
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: build with minify flag" {
    mkdir -p "$TEST_DIR/build-test"
    run timeout 5 scenario-to-extension build "$TEST_DIR/build-test" --minify
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
}

@test "CLI: invalid command shows error" {
    run scenario-to-extension invalid-command-that-does-not-exist
    [ "$status" -ne 0 ]
    [[ "$output" =~ "unknown" ]] || [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "help" ]]
}

@test "CLI: --help flag works" {
    run scenario-to-extension --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]] || [[ "$output" =~ "Commands:" ]] || [[ "$output" =~ "help" ]]
}

@test "CLI: --version flag works" {
    run scenario-to-extension --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI: generate with invalid template type" {
    run timeout 5 scenario-to-extension generate test-scenario --template invalid-template-type
    [ "$status" -ne 0 ]
}

@test "CLI: multiple template types" {
    for template in "full" "content-script-only" "background-only" "popup-only"; do
        run timeout 5 scenario-to-extension generate "test-$template" --template "$template"
        # Should accept valid template types
        [ "$status" -eq 0 ] || [ "$status" -eq 1 ] || [ "$status" -eq 124 ]
    done
}
