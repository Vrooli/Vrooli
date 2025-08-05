#!/usr/bin/env bats

# BATS setup function - runs before each test
setup() {
    # Set up paths
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"
    
    # Source dependencies in order
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    
    # Source config
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "common.sh defines all required functions" {
    # Functions are already loaded by setup()
    declare -f claude_code::check_node_version
    declare -f claude_code::is_installed
    declare -f claude_code::get_version
    declare -f claude_code::set_timeouts
    declare -f claude_code::build_allowed_tools
}
# ============================================================================
# Node Version Tests
# ============================================================================
@test "claude_code::check_node_version returns 1 if node not installed" {
    # Override system::is_command to simulate node not installed
    system::is_command() { return 1; }
    run claude_code::check_node_version
    [ "$status" -eq 1 ]
}
@test "claude_code::check_node_version returns 0 for valid version" {
    # Override to simulate node installed with valid version
    system::is_command() { return 0; }
    node() { echo 'v20.0.0'; }
    run claude_code::check_node_version
    [ "$status" -eq 0 ]
}

@test "claude_code::check_node_version returns 1 for old version" {
    # Override to simulate node installed with old version
    system::is_command() { return 0; }
    node() { echo 'v16.0.0'; }
    
    run claude_code::check_node_version
    [ "$status" -eq 1 ]
}

# ============================================================================
# Installation Check Tests
# ============================================================================

@test "claude_code::is_installed returns 0 if claude command exists" {
    # Override to simulate claude command exists
    system::is_command() { [[ "$1" == "claude" ]] && return 0 || return 1; }
    
    run claude_code::is_installed
    [ "$status" -eq 0 ]
}

@test "claude_code::is_installed returns 1 if claude command missing" {
    # Override to simulate claude command missing
    system::is_command() { return 1; }
    
    run claude_code::is_installed
    [ "$status" -eq 1 ]
}

# ============================================================================
# Version Tests
# ============================================================================

@test "claude_code::get_version returns version when installed" {
    # Override to simulate claude installed
    claude_code::is_installed() { return 0; }
    claude() { echo '1.0.0'; }
    
    run claude_code::get_version
    [ "$status" -eq 0 ]
    [[ "$output" == "1.0.0" ]]
}

@test "claude_code::get_version returns 'not installed' when not installed" {
    # Override to simulate claude not installed
    claude_code::is_installed() { return 1; }
    
    run claude_code::get_version
    [ "$status" -eq 0 ]
    [[ "$output" == "not installed" ]]
}

# ============================================================================
# Timeout Tests
# ============================================================================

@test "claude_code::set_timeouts sets environment variables correctly" {
    # Call the function
    claude_code::set_timeouts 300
    
    # Check environment variables were set
    [[ "$BASH_DEFAULT_TIMEOUT_MS" == "300000" ]]
    [[ "$BASH_MAX_TIMEOUT_MS" == "300000" ]]
    [[ "$MCP_TOOL_TIMEOUT" == "300000" ]]
}

@test "claude_code::set_timeouts uses default when no argument provided" {
    # DEFAULT_TIMEOUT should be set by config/defaults.sh
    claude_code::set_timeouts
    
    # Check it used the default (600 seconds = 600000 ms)
    [[ "$BASH_DEFAULT_TIMEOUT_MS" == "600000" ]]
}

# ============================================================================
# Allowed Tools Tests
# ============================================================================

@test "claude_code::build_allowed_tools handles empty input" {
    run claude_code::build_allowed_tools ''
    [ "$status" -eq 0 ]
    [[ "$output" == "" ]]
}

@test "claude_code::build_allowed_tools handles single tool" {
    run claude_code::build_allowed_tools 'tool1'
    [ "$status" -eq 0 ]
    [[ "$output" == " --allowedTools \"tool1\"" ]]
}

@test "claude_code::build_allowed_tools handles multiple tools" {
    run claude_code::build_allowed_tools 'tool1,tool2,tool3'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--allowedTools \"tool1\"" ]]
    [[ "$output" =~ "--allowedTools \"tool2\"" ]]
    [[ "$output" =~ "--allowedTools \"tool3\"" ]]
}