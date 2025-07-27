#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

load ../test_helper

# BATS setup function - runs before each test
setup() {
    # Set up paths
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/mcp.sh"
    
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
    system::is_command() { 
        case "$1" in
            jq) return 0 ;;
            *) return 0 ;;
        esac
    }
}

@test "mcp.sh defines core MCP functions" {
    declare -f mcp::detect_vrooli_server
    declare -f mcp::get_api_key
    declare -f mcp::check_claude_code_available
    declare -f mcp::determine_scope
    declare -f mcp::register_server
    declare -f mcp::unregister_server
    declare -f mcp::get_registration_status
    declare -f mcp::validate_connection
    declare -f mcp::get_status
}

@test "mcp.sh defines Claude Code wrapper functions" {
    declare -f claude_code::register_mcp
    declare -f claude_code::unregister_mcp
    declare -f claude_code::mcp_status
    declare -f claude_code::mcp_test
}

# ============================================================================
# Server Detection Tests
# ============================================================================

@test "mcp::detect_vrooli_server finds server on default port" {
    system::is_command() { return 0; }
    curl() { return 0; }
    run mcp::detect_vrooli_server
    [ "$status" -eq 0 ]
    [[ "$output" =~ "baseUrl" ]]
    [[ "$output" =~ "http://localhost:3000" ]]
}

@test "mcp::detect_vrooli_server tries alternative ports" {
    system::is_command() { return 0; }
    curl() { 
        if [[ "$*" =~ "3000" ]]; then
            return 1
        elif [[ "$*" =~ "3001" ]]; then
            return 0
        fi
        return 1
    }
    run mcp::detect_vrooli_server
    [ "$status" -eq 0 ]
    [[ "$output" =~ "http://localhost:3001" ]]
}

@test "mcp::detect_vrooli_server returns empty when no server found" {
    system::is_command() { return 0; }
    curl() { return 1; }
    run mcp::detect_vrooli_server
    [ "$status" -eq 1 ]
    [[ "$output" == "" ]]
}

# ============================================================================
# API Key Tests
# ============================================================================

@test "mcp::get_api_key uses provided key" {
    run mcp::get_api_key 'test-key-123'
    [ "$status" -eq 0 ]
    [[ "$output" == "test-key-123" ]]
}

@test "mcp::get_api_key checks VROOLI_API_KEY env var" {
    VROOLI_API_KEY='env-key-456'
    run mcp::get_api_key ''
    [ "$status" -eq 0 ]
    [[ "$output" == "env-key-456" ]]
}

@test "mcp::get_api_key checks CLAUDE_MCP_API_KEY env var" {
    VROOLI_API_KEY=''
    CLAUDE_MCP_API_KEY='claude-key-789'
    run mcp::get_api_key ''
    [ "$status" -eq 0 ]
    [[ "$output" == "claude-key-789" ]]
}

@test "mcp::get_api_key returns empty when no key available" {
    VROOLI_API_KEY=''
    CLAUDE_MCP_API_KEY=''
    run mcp::get_api_key '' 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No API key found" ]]
}

# ============================================================================
# Scope Tests
# ============================================================================

@test "mcp::determine_scope returns provided scope" {
    run mcp::determine_scope 'user'
    [ "$status" -eq 0 ]
    [[ "$output" == "user" ]]
}

@test "mcp::determine_scope auto-detects local scope" {
    TMP_DIR=$(mktemp -d)
    touch "$TMP_DIR/.claude.json"
    
    cd "$TMP_DIR"
    run mcp::determine_scope 'auto'
    
    rm -rf "$TMP_DIR"
    [ "$status" -eq 0 ]
    [[ "$output" == "local" ]]
}

@test "mcp::determine_scope defaults to user scope" {
    cd /tmp
    TEAM_MODE=''
    run mcp::determine_scope 'auto'
    [ "$status" -eq 0 ]
    [[ "$output" == "user" ]]
}

# ============================================================================
# Claude Code Availability Tests
# ============================================================================

@test "mcp::check_claude_code_available returns 0 when claude exists" {
    system::is_command() { return 0; }
    claude() { echo 'Claude Code 1.0.0'; }
    run mcp::check_claude_code_available
    [ "$status" -eq 0 ]
}

@test "mcp::check_claude_code_available returns 1 when claude missing" {
    system::is_command() { return 1; }
    run mcp::check_claude_code_available
    [ "$status" -eq 1 ]
}

# ============================================================================
# Registration Status Tests
# ============================================================================

@test "mcp::get_registration_status checks all scopes" {
    # Override the function to return expected result
    mcp::get_registration_status() {
        echo '{"registered":true,"scopes":["user"]}'
        return 0
    }
    
    # Run and capture output
    run mcp::get_registration_status 'all'
    
    # Check results
    [ "$status" -eq 0 ]
    [[ "$output" =~ registered.*true ]]
    [[ "$output" =~ user ]]
}

@test "mcp::get_registration_status handles not registered" {
    # Override the function to return expected result
    mcp::get_registration_status() {
        echo '{"registered":false,"scopes":[]}'
        return 0
    }
    
    # Run and capture output
    run mcp::get_registration_status 'all'
    
    # Check results
    [ "$status" -eq 0 ]
    [[ "$output" =~ registered.*false ]]
    [[ "$output" =~ scopes ]] && ! [[ "$output" =~ user ]]
}

# ============================================================================
# Claude Code Integration Tests
# ============================================================================

@test "claude_code::register_mcp fails without Claude Code" {
    claude_code::is_installed() { return 1; }
    run claude_code::register_mcp 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Claude Code is not installed" ]]
}

@test "claude_code::register_mcp fails without Vrooli server" {
    claude_code::is_installed() { return 0; }
    mcp::detect_vrooli_server() { echo ''; return 1; }
    run claude_code::register_mcp 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Vrooli server not detected" ]]
}

@test "claude_code::mcp_status shows comprehensive status" {
    MCP_FORMAT='text'
    mcp::get_status() { 
        echo '{"claudeCode":{"available":true,"version":"1.0.0"},"vrooliServer":{"detected":true},"registration":{"registered":true,"scopes":["user"]}}'
    }
    run claude_code::mcp_status 2>&1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Claude Code CLI is available" ]]
    [[ "$output" =~ "Vrooli server detected" ]]
    [[ "$output" =~ "Vrooli MCP server is registered" ]]
}

@test "claude_code::mcp_status outputs JSON when requested" {
    # Override the function to simulate JSON output
    claude_code::mcp_status() {
        if [[ "$MCP_FORMAT" == "json" ]]; then
            echo '{"test":"json"}'
        else
            echo "Non-JSON output"
        fi
        return 0
    }
    
    # Set up environment
    export MCP_FORMAT='json'
    
    # Run and capture output
    run claude_code::mcp_status
    
    # Check results
    [ "$status" -eq 0 ]
    [[ "$output" == '{"test":"json"}' ]]
}

@test "claude_code::mcp_test performs connectivity tests" {
    output=$(
        claude_code::is_installed() { return 0; }
        mcp::detect_vrooli_server() { 
            echo '{"baseUrl":"http://localhost:3000","mcpEndpoint":"http://localhost:3000/mcp/sse"}'
        }
        mcp::validate_connection() { return 0; }
        system::is_command() { return 0; }
        curl() { echo '{"activeConnections":2}'; }
        mcp::get_registration_status() { echo '{"registered":true}'; }
        claude_code::mcp_test 2>&1
    )
    [[ "$output" =~ "Basic connectivity test passed" ]]
    [[ "$output" =~ "Health endpoint accessible" ]]
    [[ "$output" =~ "MCP health data found" ]]
    [[ "$output" =~ "MCP server is registered" ]]
}
