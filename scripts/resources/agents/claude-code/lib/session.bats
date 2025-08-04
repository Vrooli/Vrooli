#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

load ./test_helper.bash

# BATS setup function - runs before each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Load the functions we are testing (required for bats isolation)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/session.sh"
    
    # Set up paths
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/session.sh"
    
    # Source dependencies in order
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/flow.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    
    # Source config and messages
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/config/messages.sh" 2>/dev/null || true
    
    # Source common functions
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Default mocks
    confirm() { return 0; }  # Always confirm
    
    # Mock log functions to prevent "command not found" errors
}

@test "session.sh defines required functions" {
    declare -f claude_code::session
    declare -f claude_code::session_list
    declare -f claude_code::session_resume
    declare -f claude_code::session_delete
    declare -f claude_code::session_view
}

# ============================================================================
# Main Session Function Tests
# ============================================================================

@test "claude_code::session fails when not installed" {
    claude_code::is_installed() { return 1; }
    run claude_code::session 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Claude Code is not installed" ]]
}

@test "claude_code::session lists sessions when no ID provided" {
    claude_code::is_installed() { return 0; }
    claude_code::session_list() { echo 'Sessions listed'; }
    SESSION_ID=''
    run claude_code::session 2>&1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Sessions listed" ]]
}

@test "claude_code::session resumes when ID provided" {
    claude_code::is_installed() { return 0; }
    claude_code::session_resume() { echo "Resuming session: $1"; }
    SESSION_ID='test-session-123'
    run claude_code::session 2>&1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Resuming session: test-session-123" ]]
}

# ============================================================================
# Session List Tests
# ============================================================================

@test "claude_code::session_list handles missing directory" {
    output=$(
        CLAUDE_SESSIONS_DIR='/nonexistent/sessions'
        claude_code::session_list 2>&1
    )
    [[ "$output" =~ "No session directory found" ]]
}

@test "claude_code::session_list handles empty directory" {
    TMP_DIR=$(mktemp -d)
    output=$(
        CLAUDE_SESSIONS_DIR="$TMP_DIR"
        claude_code::session_list 2>&1
    )
    rm -rf "$TMP_DIR"
    [[ "$output" =~ "No sessions found" ]]
}

@test "claude_code::session_list shows existing sessions" {
    TMP_DIR=$(mktemp -d)
    touch "$TMP_DIR/session1" "$TMP_DIR/session2" "$TMP_DIR/session3"
    
    output=$(
        CLAUDE_SESSIONS_DIR="$TMP_DIR"
        claude_code::session_list 2>&1
    )
    
    rm -rf "$TMP_DIR"
    [[ "$output" =~ "Recent sessions:" ]]
    [[ "$output" =~ "session" ]]
}

# ============================================================================
# Session Resume Tests
# ============================================================================

@test "claude_code::session_resume builds basic command" {
    # Set up environment with intelligent eval mock
    eval() {
        local cmd="$*"
        # Skip log color variables and BATS debug info
        if [[ "$cmd" =~ ^(RED|GREEN|YELLOW|BLUE|MAGENTA|CYAN|WHITE)= ]] || \
           [[ "$cmd" =~ ^color_value= ]] || \
           [[ "$cmd" =~ ^stack_trace= ]] || \
           [[ "$cmd" =~ BATS_DEBUG ]]; then
            # Execute the actual command for log functions
            builtin eval "$cmd" 2>/dev/null || true
            return 0
        fi
        # Only show commands that look like claude commands
        if [[ "$cmd" =~ ^claude ]]; then
            echo "Command: $cmd"
        fi
        return 0
    }
    
    # Set variables
    export DEFAULT_MAX_TURNS=5
    export MAX_TURNS=5
    
    # Run and capture output
    run claude_code::session_resume 'test-session'
    
    # Check results
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Command: claude --resume \"test-session\"" ]]
}

@test "claude_code::session_resume adds custom max turns" {
    # Set up environment with intelligent eval mock
    eval() {
        local cmd="$*"
        # Skip log color variables and BATS debug info
        if [[ "$cmd" =~ ^(RED|GREEN|YELLOW|BLUE|MAGENTA|CYAN|WHITE)= ]] || \
           [[ "$cmd" =~ ^color_value= ]] || \
           [[ "$cmd" =~ ^stack_trace= ]] || \
           [[ "$cmd" =~ BATS_DEBUG ]]; then
            # Execute the actual command for log functions
            builtin eval "$cmd" 2>/dev/null || true
            return 0
        fi
        # Only show commands that look like claude commands
        if [[ "$cmd" =~ ^claude ]]; then
            echo "Command: $cmd"
        fi
        return 0
    }
    
    # Set variables
    export DEFAULT_MAX_TURNS=5
    export MAX_TURNS=10
    
    # Run and capture output
    run claude_code::session_resume 'test-session'
    
    # Check results
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--max-turns 10" ]]
}

# ============================================================================
# Session Delete Tests
# ============================================================================

@test "claude_code::session_delete fails without session ID" {
    run claude_code::session_delete '' 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No session ID provided" ]]
}

@test "claude_code::session_delete handles missing session" {
    output=$(
        CLAUDE_SESSIONS_DIR='/tmp'
        claude_code::session_delete 'nonexistent' 2>&1
    )
    [[ "$output" =~ "Session not found" ]]
}

@test "claude_code::session_delete removes existing session" {
    TMP_DIR=$(mktemp -d)
    touch "$TMP_DIR/test-session"
    
    output=$(
        CLAUDE_SESSIONS_DIR="$TMP_DIR"
        confirm() { return 0; }
        claude_code::session_delete 'test-session' 2>&1
    )
    
    [ ! -f "$TMP_DIR/test-session" ]
    rm -rf "$TMP_DIR"
    [[ "$output" =~ "Session deleted" ]]
}

@test "claude_code::session_delete cancels on no confirmation" {
    TMP_DIR=$(mktemp -d)
    touch "$TMP_DIR/test-session"
    
    output=$(
        CLAUDE_SESSIONS_DIR="$TMP_DIR"
        confirm() { return 1; }
        claude_code::session_delete 'test-session' 2>&1
    )
    
    [ -f "$TMP_DIR/test-session" ]
    rm -rf "$TMP_DIR"
    [[ "$output" =~ "cancelled" ]]
}

# ============================================================================
# Session View Tests
# ============================================================================

@test "claude_code::session_view fails without session ID" {
    run claude_code::session_view '' 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No session ID provided" ]]
}

@test "claude_code::session_view handles missing session" {
    output=$(
        CLAUDE_SESSIONS_DIR='/tmp'
        claude_code::session_view 'nonexistent' 2>&1
    ) || status=$?
    [ "${status:-0}" -eq 1 ]
    [[ "$output" =~ "Session not found" ]]
}

@test "claude_code::session_view displays session content" {
    TMP_DIR=$(mktemp -d)
    echo '{"test": "data"}' > "$TMP_DIR/test-session"
    
    output=$(
        CLAUDE_SESSIONS_DIR="$TMP_DIR"
        system::is_command() { [[ "$1" == "jq" ]] && return 1; }
        claude_code::session_view 'test-session' 2>&1
    )
    
    rm -rf "$TMP_DIR"
    [[ "$output" =~ "Session details for: test-session" ]]
    [[ "$output" =~ "test" ]]
    [[ "$output" =~ "data" ]]
}
