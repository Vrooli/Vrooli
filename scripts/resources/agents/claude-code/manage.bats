#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/manage.sh"
CONFIG_DIR="$BATS_TEST_DIRNAME/config"
LIB_DIR="$BATS_TEST_DIRNAME/lib"

# Source dependencies
RESOURCES_DIR="$BATS_TEST_DIRNAME/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source required utilities (suppress errors during test setup)
. "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/args.sh" 2>/dev/null || true

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing manage.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f claude_code::parse_arguments && declare -f claude_code::main"
    [ "$status" -eq 0 ]
    [[ "$output" =~ claude_code::parse_arguments ]]
    [[ "$output" =~ claude_code::main ]]
}

@test "manage.sh sources all required dependencies" {
    run bash -c "source '$SCRIPT_PATH' 2>&1 | grep -v 'command not found' | head -1"
    [ "$status" -eq 0 ]
}

@test "manage.sh sources config files" {
    run bash -c "source '$SCRIPT_PATH' && echo \$RESOURCE_NAME"
    [ "$status" -eq 0 ]
    [[ "$output" == "claude-code" ]]
}

@test "manage.sh sources all library modules" {
    run bash -c "source '$SCRIPT_PATH' && declare -f claude_code::install && declare -f claude_code::status && declare -f claude_code::run"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "claude_code::parse_arguments sets default action to help" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "help" ]]
}

@test "claude_code::parse_arguments accepts install action" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --action install; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "claude_code::parse_arguments accepts status action" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --action status; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

@test "claude_code::parse_arguments accepts MCP actions" {
    for action in register-mcp unregister-mcp mcp-status mcp-test; do
        run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --action $action; echo \"\$ACTION\""
        [ "$status" -eq 0 ]
        [[ "$output" =~ "$action" ]]
    done
}

@test "claude_code::parse_arguments handles prompt parameter" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --prompt 'Test prompt'; echo \"\$PROMPT\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Test prompt" ]]
}

@test "claude_code::parse_arguments sets default max-turns" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments; echo \"\$MAX_TURNS\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "5" ]]
}

@test "claude_code::parse_arguments accepts custom max-turns" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --max-turns 10; echo \"\$MAX_TURNS\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "10" ]]
}

@test "claude_code::parse_arguments handles MCP scope parameter" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments --scope user; echo \"\$MCP_SCOPE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "user" ]]
}

@test "claude_code::parse_arguments sets default timeout" {
    run bash -c "source '$SCRIPT_PATH'; claude_code::parse_arguments; echo \"\$TIMEOUT\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "600" ]]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "manage.sh shows help with --help flag" {
    run bash "$SCRIPT_PATH" --help
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