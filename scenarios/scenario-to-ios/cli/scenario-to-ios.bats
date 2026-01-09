#!/usr/bin/env bats
# CLI tests for scenario-to-ios

setup() {
    # Store the directory containing the CLI script
    CLI_DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" && pwd )"
    CLI_SCRIPT="${CLI_DIR}/scenario-to-ios"

    # Verify CLI script exists and is executable
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI shows help with no arguments" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scenario-to-ios - Convert Vrooli scenarios to native iOS applications" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows help with --help flag" {
    run "$CLI_SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scenario-to-ios - Convert Vrooli scenarios to native iOS applications" ]]
}

@test "CLI shows help with help command" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE:" ]]
}

@test "CLI shows version with version command" {
    run "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scenario-to-ios version" ]]
}

@test "CLI rejects invalid command" {
    run "$CLI_SCRIPT" invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error: Unknown command" ]]
}

@test "CLI build command requires scenario argument" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" build
    # Should fail without scenario name
    [ "$status" -ne 0 ]
}

@test "CLI status command shows system status" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Checking prerequisites" ]]
}

@test "CLI testflight command requires IPA path" {
    run "$CLI_SCRIPT" testflight
    # Should fail without IPA path
    [ "$status" -ne 0 ]
}

@test "CLI validate command requires IPA path" {
    run "$CLI_SCRIPT" validate
    # Should fail without IPA path
    [ "$status" -ne 0 ]
}

@test "CLI validate command rejects non-existent IPA" {
    run "$CLI_SCRIPT" validate /nonexistent/path.ipa
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error: IPA file not found" ]]
}

@test "CLI simulator command requires scenario argument" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" simulator
    # Should fail without scenario name
    [ "$status" -ne 0 ]
}

@test "CLI build command accepts --output flag" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    # This will fail at prerequisite check or scenario not found, but should parse arguments
    run "$CLI_SCRIPT" build test-scenario --output /tmp/test-output
    # We're just checking that the flag is recognized, not that it succeeds
    [[ ! "$output" =~ "Unknown option" ]]
}

@test "CLI build command accepts --device flag" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" build test-scenario --device iphone
    # We're just checking that the flag is recognized
    [[ ! "$output" =~ "Unknown option" ]]
}

@test "CLI build command accepts --sign flag" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" build test-scenario --sign "iPhone Developer"
    # We're just checking that the flag is recognized
    [[ ! "$output" =~ "Unknown option" ]]
}

@test "CLI testflight command accepts --notes flag" {
    run "$CLI_SCRIPT" testflight /tmp/test.ipa --notes "Test release"
    # Will fail on missing file, but flag should be recognized
    [[ ! "$output" =~ "Unknown option" ]]
}

@test "CLI testflight command accepts --groups flag" {
    run "$CLI_SCRIPT" testflight /tmp/test.ipa --groups "internal,beta"
    # Will fail on missing file, but flag should be recognized
    [[ ! "$output" =~ "Unknown option" ]]
}

@test "CLI simulator command accepts --device flag" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" simulator test-scenario --device "iPhone 14"
    # We're just checking that the flag is recognized
    [[ ! "$output" =~ "Unknown option" ]]
}

@test "CLI simulator command accepts --ios-version flag" {
    # Only run on macOS where Xcode is available
    if [[ "$OSTYPE" != "darwin"* ]]; then
        skip "Test requires macOS with Xcode"
    fi

    run "$CLI_SCRIPT" simulator test-scenario --ios-version "16.0"
    # We're just checking that the flag is recognized
    [[ ! "$output" =~ "Unknown option" ]]
}
