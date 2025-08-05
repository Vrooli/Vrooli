#!/usr/bin/env bats
# Tests for Claude Code manage.sh script

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code"
    
    # Load Claude Code specific configuration once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    CLAUDE_CODE_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${CLAUDE_CODE_DIR}/config/defaults.sh"
    source "${CLAUDE_CODE_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export CLAUDE_CODE_CUSTOM_PORT="8080"
    export CLAUDE_CODE_CONTAINER_NAME="claude-code-test"
    export CLAUDE_CODE_BASE_URL="http://localhost:8080"
    export FORCE="no"
    export YES="no"
    export OUTPUT_FORMAT="text"
    export QUIET="no"
    
    # Export config functions
    claude_code::export_config
    claude_code::export_messages
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "manage.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "manage.sh defines required functions" {
    # Functions should be available from setup_file
    declare -f claude_code::parse_arguments >/dev/null
    declare -f claude_code::main >/dev/null
    declare -f claude_code::install >/dev/null
    declare -f claude_code::status >/dev/null
}

@test "manage.sh loads configuration correctly" {
    # Config should be loaded from setup_file
    [ "$RESOURCE_NAME" = "claude-code" ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "claude_code::parse_arguments sets default action to help" {
    claude_code::parse_arguments
    [ "$ACTION" = "help" ]
}

@test "claude_code::parse_arguments accepts install action" {
    claude_code::parse_arguments --action install
    [ "$ACTION" = "install" ]
}

@test "claude_code::parse_arguments accepts status action" {
    claude_code::parse_arguments --action status
    [ "$ACTION" = "status" ]
}

@test "claude_code::parse_arguments accepts MCP actions" {
    for action in register-mcp unregister-mcp mcp-status mcp-test; do
        claude_code::parse_arguments --action "$action"
        [ "$ACTION" = "$action" ]
    done
}

@test "claude_code::parse_arguments handles prompt parameter" {
    claude_code::parse_arguments --prompt "Test prompt"
    [ "$PROMPT" = "Test prompt" ]
}

@test "claude_code::parse_arguments sets default max-turns" {
    claude_code::parse_arguments
    [ "$MAX_TURNS" = "5" ]
}

@test "claude_code::parse_arguments accepts custom max-turns" {
    claude_code::parse_arguments --max-turns 10
    [ "$MAX_TURNS" = "10" ]
}

@test "claude_code::parse_arguments handles MCP scope parameter" {
    claude_code::parse_arguments --scope user
    [ "$MCP_SCOPE" = "user" ]
}

@test "claude_code::parse_arguments sets default timeout" {
    claude_code::parse_arguments
    [ "$TIMEOUT" = "600" ]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "claude_code::main shows help with --help flag" {
    run claude_code::main --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Actions:" ]]
    [[ "$output" =~ "MCP Actions:" ]]
}

@test "manage.sh shows help with -h flag" {
    run bash "$SCRIPT_PATH" -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "manage.sh shows help with no arguments" {
    run bash "$SCRIPT_PATH"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "manage.sh help includes all actions" {
    run bash "$SCRIPT_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
    [[ "$output" =~ "uninstall" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "info" ]]
    [[ "$output" =~ "run" ]]
    [[ "$output" =~ "batch" ]]
    [[ "$output" =~ "session" ]]
    [[ "$output" =~ "settings" ]]
    [[ "$output" =~ "logs" ]]
    [[ "$output" =~ "register-mcp" ]]
    [[ "$output" =~ "unregister-mcp" ]]
    [[ "$output" =~ "mcp-status" ]]
    [[ "$output" =~ "mcp-test" ]]
}

# ============================================================================
# Action Routing Tests
# ============================================================================

@test "claude_code::main routes to correct function for each action" {
    # Mock the action functions to verify they're called
    run bash -c "
        source '$SCRIPT_PATH'
        claude_code::install() { echo 'install called'; }
        claude_code::status() { echo 'status called'; }
        claude_code::main --action install
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install called" ]]
}

@test "claude_code::main handles unknown action" {
    run bash "$SCRIPT_PATH" --action unknown-action
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

# ============================================================================
# Environment Variable Tests
# ============================================================================

@test "claude_code::export_config exports required variables" {
    run bash -c "
        source '$SCRIPT_PATH'
        claude_code::export_config
        echo \"RESOURCE_NAME=\$RESOURCE_NAME\"
        echo \"MIN_NODE_VERSION=\$MIN_NODE_VERSION\"
        echo \"CLAUDE_PACKAGE=\$CLAUDE_PACKAGE\"
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RESOURCE_NAME=claude-code" ]]
    [[ "$output" =~ "MIN_NODE_VERSION=18" ]]
    [[ "$output" =~ "CLAUDE_PACKAGE=@anthropic-ai/claude-code" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "manage.sh can handle multiple arguments correctly" {
    run bash "$SCRIPT_PATH" --action run --prompt "test" --max-turns 3 --timeout 300
    [ "$status" -eq 1 ]  # Should fail because Claude isn't installed in test env
    # But should parse args correctly (not show help)
    [[ ! "$output" =~ "Usage:" ]]
}

@test "manage.sh respects --yes flag" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --yes yes; echo \"\$YES\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

@test "manage.sh respects --force flag" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --force yes; echo \"\$FORCE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}