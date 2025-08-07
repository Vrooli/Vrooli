#!/usr/bin/env bats
# Tests for Claude Code manage.sh script
bats_require_minimum_version 1.5.0

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code"
    
    # Load Claude Code specific configuration once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    
    # Load configuration and manage script once
    source "${SCRIPT_DIR}/config/defaults.sh"
    source "${SCRIPT_DIR}/config/messages.sh"
    
    # Export DEFAULT values that manage.sh uses
    export DEFAULT_MAX_TURNS="5"
    export DEFAULT_TIMEOUT="600"
    export DEFAULT_OUTPUT_FORMAT="text"
    export DEFAULT_MCP_SCOPE="auto"
    
    # Load all dependencies needed by manage.sh
    export RESOURCES_DIR="${SCRIPT_DIR}/../.."
    source "${RESOURCES_DIR}/common.sh" || true
    source "${RESOURCES_DIR}/../helpers/utils/args.sh" || true
    
    # Load lib files that manage.sh needs
    for lib_file in "${SCRIPT_DIR}"/lib/*.sh; do
        [[ -f "$lib_file" ]] && source "$lib_file"
    done
    
    # Now load manage.sh with all dependencies available
    source "${SCRIPT_DIR}/manage.sh"
    
    # Export key functions for BATS subshells
    export -f claude_code::main
    export -f claude_code::parse_arguments
    export -f claude_code::usage
    export -f claude_code::export_config
    export -f claude_code::install
    export -f claude_code::uninstall
    export -f claude_code::start
    export -f claude_code::stop
    export -f claude_code::restart
    export -f claude_code::status
    export -f claude_code::info
    export -f claude_code::test
    export -f claude_code::logs
    export -f claude_code::run
    export -f claude_code::batch
    export -f claude_code::session
    export -f claude_code::settings
    export -f claude_code::handle_mcp
    export -f claude_code::handle_batch
    export -f claude_code::register_mcp
    export -f claude_code::unregister_mcp
    export -f claude_code::mcp_status
    export -f claude_code::mcp_test
    
    # Export helper functions from common.sh that tests might need
    export -f claude_code::check_node_version 2>/dev/null || true
    export -f claude_code::is_installed 2>/dev/null || true
    export -f claude_code::get_version 2>/dev/null || true
    
    # Export args functions needed by claude_code::parse_arguments
    export -f args::reset 2>/dev/null || true
    export -f args::register 2>/dev/null || true
    export -f args::register_help 2>/dev/null || true
    export -f args::register_yes 2>/dev/null || true
    export -f args::parse 2>/dev/null || true
    export -f args::get 2>/dev/null || true
    export -f args::is_asking_for_help 2>/dev/null || true
    export -f args::usage 2>/dev/null || true
    
    # Export log functions once
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# Lightweight per-test setup
setup() {
    # Load claude-code mock in each test to ensure functions are available
    MOCK_DIR="$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/mocks"
    if [[ -f "$MOCK_DIR/claude-code.sh" ]]; then
        source "$MOCK_DIR/claude-code.sh"
    fi
    
    # Setup standard Vrooli mocks (this sets MOCK_LOG_DIR)
    vrooli_auto_setup
    
    # Ensure mock log directory exists after vrooli_auto_setup
    if [[ -n "${MOCK_LOG_DIR:-}" ]]; then
        mkdir -p "$MOCK_LOG_DIR"
    fi
    
    # Reset mock state to clean slate for each test
    if declare -f mock::claude::reset >/dev/null 2>&1; then
        mock::claude::reset
    fi
    
    # Set test environment variables (lightweight per-test)
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    export CLAUDE_CODE_CUSTOM_PORT="8080"
    export CLAUDE_CODE_CONTAINER_NAME="claude-code-test"
    export CLAUDE_CODE_BASE_URL="http://localhost:8080"
    export FORCE="no"
    export YES="no"
    export OUTPUT_FORMAT="text"
    export QUIET="no"
    
    # Export config functions (quick operation)
    claude_code::export_config 2>/dev/null || true
    
    # Configure claude mock with clean default state
    if declare -f mock::claude::scenario::setup_healthy >/dev/null 2>&1; then
        mock::claude::scenario::setup_healthy >/dev/null
    fi
}

# BATS teardown function - runs after each test
teardown() {
    # Clean up mock log directory
    if [[ -n "${MOCK_LOG_DIR:-}" && -d "$MOCK_LOG_DIR" ]]; then
        rm -rf "$MOCK_LOG_DIR"
    fi
    
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "manage.sh loads without errors" {
    # Should load successfully in setup_file
    run echo "Script loaded successfully"
    assert_success
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
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "Actions:"
    assert_output_contains "MCP Actions:"
}

@test "manage.sh shows help with -h flag" {
    run bash "${SCRIPT_DIR}/manage.sh" -h
    assert_success
    assert_output_contains "Usage:"
}

@test "manage.sh shows help with no arguments" {
    run bash "${SCRIPT_DIR}/manage.sh"
    assert_success
    assert_output_contains "Usage:"
}

@test "manage.sh help includes all actions" {
    run bash "${SCRIPT_DIR}/manage.sh" --help
    assert_success
    assert_output_contains "install"
    assert_output_contains "uninstall"
    assert_output_contains "status"
    assert_output_contains "info"
    assert_output_contains "run"
    assert_output_contains "batch"
    assert_output_contains "session"
    assert_output_contains "settings"
    assert_output_contains "logs"
    assert_output_contains "register-mcp"
    assert_output_contains "unregister-mcp"
    assert_output_contains "mcp-status"
    assert_output_contains "mcp-test"
}

# ============================================================================
# Action Routing Tests
# ============================================================================

@test "claude_code::main routes to correct function for each action" {
    # Mock the action functions to verify they're called
    run bash -c "
        source '${SCRIPT_DIR}/manage.sh'
        claude_code::install() { echo 'install called'; }
        claude_code::status() { echo 'status called'; }
        claude_code::main --action install
    "
    assert_success
    assert_output_contains "install called"
}

@test "claude_code::main handles unknown action" {
    run bash "${SCRIPT_DIR}/manage.sh" --action unknown-action
    assert_success
    assert_output_contains "Usage:"
}

# ============================================================================
# Environment Variable Tests
# ============================================================================

@test "claude_code::export_config exports required variables" {
    run bash -c "
        source '${SCRIPT_DIR}/manage.sh'
        claude_code::export_config
        echo \"RESOURCE_NAME=\$RESOURCE_NAME\"
        echo \"MIN_NODE_VERSION=\$MIN_NODE_VERSION\"
        echo \"CLAUDE_PACKAGE=\$CLAUDE_PACKAGE\"
    "
    assert_success
    assert_output_contains "RESOURCE_NAME=claude-code"
    assert_output_contains "MIN_NODE_VERSION=18"
    assert_output_contains "CLAUDE_PACKAGE=@anthropic-ai/claude-code"
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "manage.sh can handle multiple arguments correctly" {
    run bash "${SCRIPT_DIR}/manage.sh" --action run --prompt "test" --max-turns 3 --timeout 300
    # Now that claude mock is working, this should succeed
    assert_success
    # Should parse args correctly (not show help)
    assert_output_not_contains "Usage:"
    # Should show some output from the claude mock
    [[ "$output" != "" ]]
}

@test "manage.sh respects --yes flag" {
    run bash -c "source '${SCRIPT_DIR}/manage.sh'; claude_code::parse_arguments --yes yes; echo \"\$YES\""
    assert_success
    assert_output_contains "yes"
}

@test "manage.sh respects --force flag" {
    run bash -c "source '${SCRIPT_DIR}/manage.sh'; claude_code::parse_arguments --force yes; echo \"\$FORCE\""
    assert_success
    assert_output_contains "yes"
}

# ============================================================================
# Service Lifecycle Tests
# ============================================================================

@test "claude_code::install function exists" {
    declare -f claude_code::install >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::uninstall function exists" {
    declare -f claude_code::uninstall >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::start function exists" {
    declare -f claude_code::start >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::stop function exists" {
    declare -f claude_code::stop >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::restart function exists" {
    declare -f claude_code::restart >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::status function exists" {
    declare -f claude_code::status >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::info function exists" {
    declare -f claude_code::info >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude_code::test function exists" {
    declare -f claude_code::test >/dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Configuration Management Tests
# ============================================================================

@test "claude code config directory exists" {
    [ -d "$(dirname "${BATS_TEST_FILENAME}")/config" ]
}

@test "claude code lib directory exists" {
    [ -d "$(dirname "${BATS_TEST_FILENAME}")/lib" ]
}

@test "claude code defaults.sh config exists" {
    [ -f "$(dirname "${BATS_TEST_FILENAME}")/config/defaults.sh" ]
}

@test "claude code messages.sh config exists" {
    [ -f "$(dirname "${BATS_TEST_FILENAME}")/config/messages.sh" ]
}

@test "claude_code::parse_arguments handles session-id parameter" {
    claude_code::parse_arguments --action session --session-id "test-session-123"
    [ "$ACTION" = "session" ]
    [ "$SESSION_ID" = "test-session-123" ]
}

@test "claude_code::parse_arguments handles allowed-tools parameter" {
    claude_code::parse_arguments --action run --allowed-tools "bash,edit,read"
    [ "$ACTION" = "run" ]
    [ "$ALLOWED_TOOLS" = "bash,edit,read" ]
}

@test "claude_code::parse_arguments handles output-format parameter" {
    claude_code::parse_arguments --action run --output-format "stream-json"
    [ "$ACTION" = "run" ]
    [ "$OUTPUT_FORMAT" = "stream-json" ]
}

@test "claude_code::parse_arguments handles dangerously-skip-permissions parameter" {
    claude_code::parse_arguments --action run --dangerously-skip-permissions yes
    [ "$ACTION" = "run" ]
    [ "$SKIP_PERMISSIONS" = "yes" ]
}

# ============================================================================
# Mock Integration Tests
# ============================================================================

@test "claude command mock is available" {
    command -v claude >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude mock responds to --version" {
    run claude --version
    assert_success
    assert_output_contains "claude-code"
}

@test "claude mock responds to --help" {
    run claude --help
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "claude"
}

@test "claude mock handles auth commands" {
    run claude auth status
    assert_success
    assert_output_contains "Logged in"
}

@test "claude mock handles MCP commands" {
    run claude mcp list
    assert_success
    assert_output_contains "Configured MCP servers"
}

@test "claude mock generates appropriate responses for prompts" {
    run claude -p "Hello, can you help me?"
    [ "$status" -eq 0 ]
    [[ "$output" != "" ]]
}

# ============================================================================
# Error Scenario Tests (Basic)
# ============================================================================

@test "claude mock functions are available" {
    # Verify mock functions exist
    command -v mock::claude::reset >/dev/null
    [ "$?" -eq 0 ]
}

@test "claude mock supports basic error modes" {
    # Test global error mode setting
    export CLAUDE_MOCK_MODE="error"
    run claude -p "test"
    assert_failure
    
    # Reset to normal mode
    export CLAUDE_MOCK_MODE="normal"
}

@test "claude mock supports auth error mode" {
    # Test auth error mode
    export CLAUDE_MOCK_MODE="auth_error"
    run claude -p "test"
    assert_failure
    assert_output_contains "Authentication"
    
    # Reset to normal mode
    export CLAUDE_MOCK_MODE="normal"
}

@test "claude mock supports not_installed mode" {
    # Test not installed mode
    export CLAUDE_MOCK_MODE="not_installed"
    run -127 claude -p "test"
    assert_output_contains "command not found"
    
    # Reset to normal mode
    export CLAUDE_MOCK_MODE="normal"
}

@test "claude mock basic functionality verification" {
    # Verify basic mock functionality works
    run claude -p "basic test"
    assert_success
    [[ "$output" != "" ]]
}

@test "claude_code::main handles missing arguments gracefully" {
    run claude_code::main
    assert_success
    assert_output_contains "Usage:"
}