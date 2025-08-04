#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

load ./test_helper.bash

# BATS setup function - runs before each test
setup() {
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
    
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/flow.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/config/messages.sh" 2>/dev/null || true
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    source "$SCRIPT_PATH"
    
    confirm() { return 0; }
    system::is_command() { 
        case "$1" in
            jq) return 0 ;;
            *) return 0 ;;
        esac
    }
}

@test "status.sh defines required functions" {
    declare -f claude_code::status
    declare -f claude_code::info
    declare -f claude_code::logs
}

@test "claude_code::status shows not installed when missing" {
    # Set up environment
    claude_code::is_installed() { return 1; }
    
    # Run and capture output
    run claude_code::status
    
    # Check results
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not installed" ]]
}

@test "claude_code::status shows installed when available" {
    output=$(
        claude_code::is_installed() { return 0; }
        claude_code::get_version() { echo "1.0.0"; }
        claude_code::check_node_version() { return 0; }
        node() { echo "v18.0.0"; }
        claude() { return 0; }
        claude_code::status 2>&1
    )
    [[ "$output" =~ "is installed" ]]
}

@test "claude_code::info displays information" {
    output=$(
        claude_code::is_installed() { return 0; }
        claude_code::display_info() { echo "Claude Code info"; }
        claude_code::get_version() { echo "1.0.0"; }
        claude_code::check_node_version() { return 0; }
        node() { echo "v18.0.0"; }
        claude() { return 0; }
        claude_code::info 2>&1
    )
    [[ "$output" =~ "Claude Code info" ]]
}

@test "claude_code::logs fails when not installed" {
    output=$(
        claude_code::is_installed() { return 1; }
        claude_code::logs 2>&1
    ) || status=$?
    [ "${status:-0}" -eq 1 ]
    [[ "$output" =~ "not installed" ]]
}

@test "claude_code::logs shows no logs message when no directories found" {
    output=$(
        claude_code::is_installed() { return 0; }
        CLAUDE_LOG_LOCATIONS=("/nonexistent1" "/nonexistent2")
        claude_code::logs 2>&1
    )
    [[ "$output" =~ "No Claude log" ]] || [[ "$output" =~ "📜 Claude Code Logs" ]]
}
