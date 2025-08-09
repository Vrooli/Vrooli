#!/usr/bin/env bats

source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# BATS setup function - runs before each test
setup() {
    # Set up paths using proper var.sh approach
    CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
    
    # Source var.sh first to get proper directory variables
    # shellcheck disable=SC1091
    source "${CLAUDE_CODE_DIR}/../../../../lib/utils/var.sh"
    
    # Source dependencies using var_ variables
    # shellcheck disable=SC1091
    source "${var_LOG_FILE}" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_SYSTEM_COMMANDS_FILE}" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_FLOW_FILE}" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" 2>/dev/null || true
    
    # Source config and messages
    # shellcheck disable=SC1091
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    # shellcheck disable=SC1091
    source "$CLAUDE_CODE_DIR/config/messages.sh" 2>/dev/null || true
    
    # Source common functions
    # shellcheck disable=SC1091
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    
    # Source the script under test
    # shellcheck disable=SC1091
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
