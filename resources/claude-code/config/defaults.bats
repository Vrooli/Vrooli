#!/usr/bin/env bats
# Tests for claude-code config/defaults.sh configuration management
bats_require_minimum_version 1.5.0

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment (this will trigger claude_code::is_test_environment)
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME}"
    export TEST_MODE="true"
    export HOME="${BATS_TEST_TMPDIR}/home"
    mkdir -p "$HOME"
    mkdir -p "$HOME/.claude"
    
    # Clear any existing configuration to test defaults
    unset RESOURCE_NAME MIN_NODE_VERSION CLAUDE_PACKAGE
    unset VROOLI_MCP_SERVER_NAME VROOLI_DEFAULT_PORT VROOLI_MCP_ENDPOINT
    unset VROOLI_HEALTH_ENDPOINT DEFAULT_MAX_TURNS DEFAULT_TIMEOUT
    unset DEFAULT_OUTPUT_FORMAT DEFAULT_MCP_SCOPE
    unset CLAUDE_CONFIG_DIR CLAUDE_SESSIONS_DIR CLAUDE_SETTINGS_FILE
    unset CLAUDE_PROJECT_SETTINGS CLAUDE_LOG_LOCATIONS
    
    # Get resource directory path
    CLAUDE_CODE_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
    
    # Load configuration
    source "${CLAUDE_CODE_DIR}/config/defaults.sh"
}

teardown() {
    # Clean up test environment
    if [[ -n "${HOME:-}" && -d "$HOME" && "$HOME" == *"/tmp"* ]]; then
        trash::safe_remove "$HOME" --test-cleanup
    fi
    
    vrooli_cleanup_test
}

# Test configuration loading and initialization

@test "claude_code::is_test_environment should detect test environment correctly" {
    # Should detect BATS_TEST_DIRNAME
    run claude_code::is_test_environment
    [ "$status" -eq 0 ]
    
    # Test with TEST_MODE
    unset BATS_TEST_DIRNAME
    export TEST_MODE="true"
    run claude_code::is_test_environment
    [ "$status" -eq 0 ]
    
    # Test without test markers
    unset TEST_MODE
    run claude_code::is_test_environment
    [ "$status" -ne 0 ]
}

@test "resource configuration should be set correctly" {
    # Test that resource constants are set
    [ "$RESOURCE_NAME" = "claude-code" ]
    [ "$MIN_NODE_VERSION" = "18" ]
    [ "$CLAUDE_PACKAGE" = "@anthropic-ai/claude-code" ]
}

@test "MCP configuration should be set correctly" {
    # Test that MCP constants are set
    [ "$VROOLI_MCP_SERVER_NAME" = "vrooli-local" ]
    [ "$VROOLI_DEFAULT_PORT" = "3000" ]
    [ "$VROOLI_MCP_ENDPOINT" = "/mcp/sse" ]
    [ "$VROOLI_HEALTH_ENDPOINT" = "/mcp/health" ]
}

@test "default CLI arguments should be set correctly" {
    # Test that CLI defaults are set
    [ "$DEFAULT_MAX_TURNS" = "5" ]
    [ "$DEFAULT_TIMEOUT" = "600" ]
    [ "$DEFAULT_OUTPUT_FORMAT" = "text" ]
    [ "$DEFAULT_MCP_SCOPE" = "auto" ]
}

@test "directory paths should be set correctly" {
    # Test that paths are constructed correctly
    [ "$CLAUDE_CONFIG_DIR" = "$HOME/.claude" ]
    [ "$CLAUDE_SESSIONS_DIR" = "$HOME/.claude/sessions" ]
    [ "$CLAUDE_SETTINGS_FILE" = "$HOME/.claude/settings.json" ]
    [[ "$CLAUDE_PROJECT_SETTINGS" =~ \.claude/settings\.json$ ]]
}

@test "log locations should include multiple paths" {
    # Test that log locations array is set
    [ -n "$CLAUDE_LOG_LOCATIONS" ]
    
    # Test that log locations logic exists in function
    grep -q "CLAUDE_LOG_LOCATIONS" "$CLAUDE_CODE_DIR/config/defaults.sh"
    grep -q "Library/Logs/claude" "$CLAUDE_CODE_DIR/config/defaults.sh"
}

@test "claude_code::export_config should export all required variables" {
    # Call export function
    claude_code::export_config
    
    # Test that variables are exported
    [ -n "${RESOURCE_NAME:-}" ]
    [ -n "${MIN_NODE_VERSION:-}" ]
    [ -n "${CLAUDE_PACKAGE:-}" ]
    [ -n "${VROOLI_MCP_SERVER_NAME:-}" ]
    [ -n "${VROOLI_DEFAULT_PORT:-}" ]
    [ -n "${VROOLI_MCP_ENDPOINT:-}" ]
    [ -n "${VROOLI_HEALTH_ENDPOINT:-}" ]
}

@test "test environment should use non-readonly variables" {
    # In test environment, variables should be modifiable
    original_value="$RESOURCE_NAME"
    RESOURCE_NAME="modified-name"
    [ "$RESOURCE_NAME" = "modified-name" ]
    
    # Restore original value
    RESOURCE_NAME="$original_value"
}

@test "configuration should handle missing HOME directory" {
    # Test with missing HOME
    result=$(bash -c "
        unset HOME
        export BATS_TEST_DIRNAME=\"/tmp/test\"
        source \"$CLAUDE_CODE_DIR/config/defaults.sh\"
        echo \$CLAUDE_CONFIG_DIR
    ")
    
    # Should still set a config directory
    [ -n "$result" ]
}

@test "MCP endpoints should have correct format" {
    # Test endpoint format
    [[ "$VROOLI_MCP_ENDPOINT" =~ ^/mcp/ ]]
    [[ "$VROOLI_HEALTH_ENDPOINT" =~ ^/mcp/ ]]
    
    # Test that endpoints are paths, not full URLs
    [[ ! "$VROOLI_MCP_ENDPOINT" =~ ^http ]]
    [[ ! "$VROOLI_HEALTH_ENDPOINT" =~ ^http ]]
}

@test "default port should be valid" {
    # Test that port is numeric and in valid range
    [[ "$VROOLI_DEFAULT_PORT" =~ ^[0-9]+$ ]]
    [ "$VROOLI_DEFAULT_PORT" -ge 1024 ]
    [ "$VROOLI_DEFAULT_PORT" -le 65535 ]
}

@test "timeout values should be reasonable" {
    # Test that timeout is numeric and reasonable
    [[ "$DEFAULT_TIMEOUT" =~ ^[0-9]+$ ]]
    [ "$DEFAULT_TIMEOUT" -ge 60 ]  # At least 1 minute
    [ "$DEFAULT_TIMEOUT" -le 3600 ]  # At most 1 hour
}

@test "max turns should be reasonable" {
    # Test that max turns is numeric and reasonable
    [[ "$DEFAULT_MAX_TURNS" =~ ^[0-9]+$ ]]
    [ "$DEFAULT_MAX_TURNS" -ge 1 ]
    [ "$DEFAULT_MAX_TURNS" -le 100 ]
}

@test "output format should be valid" {
    # Test that output format is one of expected values
    [[ "$DEFAULT_OUTPUT_FORMAT" =~ ^(text|json|markdown)$ ]]
}

@test "MCP scope should be valid" {
    # Test that MCP scope is one of expected values
    [[ "$DEFAULT_MCP_SCOPE" =~ ^(auto|user|global)$ ]]
}

@test "Node.js version should be reasonable" {
    # Test that Node version is numeric and reasonable
    [[ "$MIN_NODE_VERSION" =~ ^[0-9]+$ ]]
    [ "$MIN_NODE_VERSION" -ge 16 ]  # Minimum modern Node
    [ "$MIN_NODE_VERSION" -le 30 ]  # Maximum reasonable Node
}

@test "package name should be valid npm package" {
    # Test that package follows npm naming conventions
    [[ "$CLAUDE_PACKAGE" =~ ^@[a-z0-9-]+/[a-z0-9-]+$ ]]
    [[ "$CLAUDE_PACKAGE" =~ anthropic ]]
    [[ "$CLAUDE_PACKAGE" =~ claude ]]
}

@test "configuration should be consistent across reloads" {
    # Test that configuration is consistent when loaded multiple times
    local name1="$RESOURCE_NAME"
    local port1="$VROOLI_DEFAULT_PORT"
    
    # Reload configuration
    source "${CLAUDE_CODE_DIR}/config/defaults.sh"
    
    local name2="$RESOURCE_NAME"
    local port2="$VROOLI_DEFAULT_PORT"
    
    [ "$name1" = "$name2" ]
    [ "$port1" = "$port2" ]
}

@test "MCP server name should be descriptive" {
    # Test that MCP server name is descriptive
    [[ "$VROOLI_MCP_SERVER_NAME" =~ vrooli ]]
    [ ${#VROOLI_MCP_SERVER_NAME} -ge 5 ]
    [ ${#VROOLI_MCP_SERVER_NAME} -le 50 ]
}

@test "project settings path should be relative to current directory" {
    # Test that project settings uses current directory
    [[ "$CLAUDE_PROJECT_SETTINGS" =~ /\.claude/settings\.json$ ]]
    
    # Should not be in HOME directory
    [[ ! "$CLAUDE_PROJECT_SETTINGS" =~ ^$HOME ]]
}

@test "log locations should include platform-specific paths" {
    # Test that log locations include macOS path
    grep -q "Library/Logs/claude" "$CLAUDE_CODE_DIR/config/defaults.sh"
    
    # Test that log locations include standard paths
    grep -q "\.claude/logs" "$CLAUDE_CODE_DIR/config/defaults.sh"
    grep -q "\.cache/claude/logs" "$CLAUDE_CODE_DIR/config/defaults.sh"
}

@test "configuration paths should be absolute" {
    # Test that config directory paths are absolute
    [[ "$CLAUDE_CONFIG_DIR" =~ ^/ ]]
    [[ "$CLAUDE_SESSIONS_DIR" =~ ^/ ]]
    [[ "$CLAUDE_SETTINGS_FILE" =~ ^/ ]]
}

@test "export function should be idempotent" {
    # Test that calling export multiple times doesn't break
    claude_code::export_config
    local name1="$RESOURCE_NAME"
    
    claude_code::export_config
    local name2="$RESOURCE_NAME"
    
    [ "$name1" = "$name2" ]
}

@test "resource name should match directory structure" {
    # Test that resource name is consistent with file location
    [ "$RESOURCE_NAME" = "claude-code" ]
    [[ "$CLAUDE_CODE_DIR" =~ claude-code$ ]]
}

@test "defaults should work in both test and production environments" {
    # Test production environment behavior
    result=$(bash -c "
        unset BATS_TEST_DIRNAME TEST_MODE
        export HOME=\"/tmp/test-home\"
        source \"$CLAUDE_CODE_DIR/config/defaults.sh\" 2>/dev/null || true
        echo \$RESOURCE_NAME
    ")
    
    [ "$result" = "claude-code" ]
}