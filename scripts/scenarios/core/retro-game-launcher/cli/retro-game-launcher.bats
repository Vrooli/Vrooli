#!/usr/bin/env bats
# Tests for Retro Game Launcher CLI

CLI_SCRIPT="$BATS_TEST_DIRNAME/retro-game-launcher"

@test "CLI script exists and is executable" {
    [ -f "$CLI_SCRIPT" ]
    [ -x "$CLI_SCRIPT" ]
}

@test "CLI shows help by default" {
    run "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Retro Game Launcher" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI shows help with help command" {
    run "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Commands:" ]]
    [[ "$output" =~ "Examples:" ]]
}

@test "CLI shows version" {
    run "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]]
}

@test "CLI handles unknown commands gracefully" {
    run "$CLI_SCRIPT" invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI shows templates without API connection" {
    # This should work even without API running
    run "$CLI_SCRIPT" templates
    [ "$status" -eq 0 ]
    [[ "$output" =~ "templates" ]]
}

@test "CLI requires arguments for certain commands" {
    run "$CLI_SCRIPT" search
    [ "$status" -eq 1 ]
    [[ "$output" =~ "required" ]]
    
    run "$CLI_SCRIPT" show
    [ "$status" -eq 1 ]
    [[ "$output" =~ "required" ]]
    
    run "$CLI_SCRIPT" generate
    [ "$status" -eq 1 ]
    [[ "$output" =~ "required" ]]
}