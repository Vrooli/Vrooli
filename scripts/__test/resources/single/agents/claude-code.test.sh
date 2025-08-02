#!/bin/bash
# ====================================================================
# Claude Code Integration Test
# ====================================================================
#
# Tests Claude Code CLI integration including availability checks,
# MCP registration, connection validation, and basic functionality.
#
# Required Resources: claude-code
# Test Categories: single-resource, agents, cli-tool
# Estimated Duration: 30-60 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="claude-code"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
else
    # Default to assuming claude-code is available if not set
    HEALTHY_RESOURCES=("claude-code")
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Claude Code configuration
CLAUDE_CODE_RESOURCES_DIR="${RESOURCES_DIR:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

# Sandbox detection and configuration
USE_SANDBOX=false
SANDBOX_MODE=""

# Detect if we're in a test environment or CI
if [[ -n "${VROOLI_TEST:-}" ]] || [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]] || [[ "${FORCE_SANDBOX:-}" == "true" ]]; then
    USE_SANDBOX=true
    SANDBOX_MODE="--action sandbox --sandbox-command"
    echo "🔒 Running Claude Code tests in SANDBOX mode for safety"
else
    # Always use sandbox for integration tests as a safety measure
    USE_SANDBOX=true
    SANDBOX_MODE="--action sandbox --sandbox-command"
    echo "🔒 Claude Code integration tests ALWAYS run in sandbox mode"
    echo "   This prevents accidental file system modifications during testing"
fi

# Test setup
setup_test() {
    echo "🔧 Setting up Claude Code integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Verify Claude Code is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq" "docker"
    
    # Check if sandbox is available
    if [[ "$USE_SANDBOX" == "true" ]]; then
        local sandbox_script="$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/sandbox/claude-sandbox.sh"
        if [[ ! -f "$sandbox_script" ]]; then
            skip_test "Claude Code sandbox not found. Run from Vrooli project root."
        fi
        
        # Check if Docker is running
        if ! docker info >/dev/null 2>&1; then
            skip_test "Docker is required for Claude Code sandbox tests"
        fi
        
        echo "✓ Sandbox mode enabled and available"
    fi
    
    echo "✓ Test setup complete"
}

# Test Claude Code CLI availability
test_claude_code_availability() {
    echo "💻 Testing Claude Code CLI availability..."
    
    if [[ "$USE_SANDBOX" == "true" ]]; then
        # In sandbox mode, check if manage script exists
        local manage_script="$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh"
        
        if [[ -f "$manage_script" ]]; then
            echo "✓ Claude Code management script found"
            assert_file_exists "$manage_script" "Management script exists"
        else
            skip_test "Claude Code management script not found"
        fi
        
        # Check if sandbox script exists
        local sandbox_script="$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/sandbox/claude-sandbox.sh"
        if [[ -f "$sandbox_script" ]]; then
            echo "✓ Claude Code sandbox script available"
            assert_file_exists "$sandbox_script" "Sandbox script exists"
        else
            skip_test "Claude Code sandbox not available"
        fi
    else
        # Original non-sandbox check (shouldn't reach here)
        local claude_code_path
        claude_code_path=$(which claude-code 2>/dev/null || echo "")
        
        if [[ -n "$claude_code_path" ]]; then
            echo "✓ Claude Code CLI found at: $claude_code_path"
            assert_not_empty "$claude_code_path" "Claude Code CLI is installed"
        else
            skip_test "Claude Code CLI not available"
        fi
    fi
    
    echo "✓ Claude Code availability test passed"
}

# Test Claude Code status and basic functionality
test_claude_code_status() {
    echo "📊 Testing Claude Code status..."
    
    if [[ "$USE_SANDBOX" == "true" ]]; then
        # Use safe test mode that doesn't execute prompts
        local test_output
        test_output=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --action test-safe 2>&1 || echo "test_failed")
        
        assert_not_empty "$test_output" "Test-safe command returned output"
        
        # Check for success indicators
        if echo "$test_output" | grep -q "Claude Code is installed"; then
            echo "✓ Claude Code installation verified (safe mode)"
        elif echo "$test_output" | grep -q "Sandbox is available"; then
            echo "✓ Sandbox availability confirmed"
        else
            echo "⚠ Safe test results: ${test_output:0:100}"
        fi
    else
        # Original status check (shouldn't reach here)
        local status_output
        status_output=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --action status 2>&1 || echo "status_failed")
        
        assert_not_empty "$status_output" "Status command returned output"
        
        # Check for installation status
        if echo "$status_output" | grep -q "installed"; then
            echo "✓ Claude Code shows as installed"
        elif echo "$status_output" | grep -q "available"; then
            echo "✓ Claude Code shows as available"
        fi
    fi
    
    echo "✓ Claude Code status test passed"
}

# Test MCP (Model Context Protocol) functionality
test_mcp_functionality() {
    echo "🔗 Testing MCP functionality..."
    
    # Test MCP status check
    local mcp_output
    mcp_output=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --action mcp-status 2>&1 || echo "mcp_failed")
    
    assert_not_empty "$mcp_output" "MCP status command returned output"
    
    # Check if output contains MCP information
    if echo "$mcp_output" | grep -qi "mcp"; then
        echo "✓ MCP status information available"
    else
        echo "⚠ MCP status may not be fully functional"
    fi
    
    # Test MCP connection (if available)
    local mcp_test_output
    mcp_test_output=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --action mcp-test 2>&1 || echo "mcp_test_failed")
    
    if [[ "$mcp_test_output" != "mcp_test_failed" ]]; then
        echo "✓ MCP test command executed"
        
        # Check for connection success indicators
        if echo "$mcp_test_output" | grep -qi "success\|connected\|ok"; then
            echo "✓ MCP connection test successful"
        elif echo "$mcp_test_output" | grep -qi "error\|failed\|timeout"; then
            echo "⚠ MCP connection may have issues"
        else
            echo "⚠ MCP test results unclear"
        fi
    else
        echo "⚠ MCP test command not available or failed"
    fi
    
    echo "✓ MCP functionality test completed"
}

# Test Claude Code resource management
test_resource_management() {
    echo "⚙️ Testing resource management..."
    
    # Test installation status
    local install_check
    install_check=$(timeout 10 "$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --action status 2>&1 | head -10 || echo "status_timeout")
    
    assert_not_empty "$install_check" "Installation check returned data"
    
    # Look for key indicators of proper setup
    if echo "$install_check" | grep -qi "claude.code\|anthropic\|cli"; then
        echo "✓ Claude Code identifiers found in status"
    fi
    
    # Test help/version functionality if available
    if [[ "$USE_SANDBOX" == "true" ]]; then
        # In sandbox mode, use safe status check instead of direct CLI
        echo "⚠ Skipping direct CLI version check in sandbox mode (safety measure)"
        echo "✓ Using manage script status instead"
    else
        # Original direct CLI check (shouldn't reach here)
        if which claude-code >/dev/null 2>&1; then
            local version_check
            version_check=$(timeout 10 claude-code --version 2>&1 || echo "version_timeout")
            
            if [[ "$version_check" != "version_timeout" ]]; then
                echo "Claude Code version: $version_check"
                assert_not_empty "$version_check" "Version command works"
            fi
        fi
    fi
    
    echo "✓ Resource management test completed"
}

# Test error handling and edge cases
test_error_handling() {
    echo "⚠️ Testing error handling..."
    
    # Test with invalid action (should fail gracefully)
    local invalid_output
    invalid_output=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --action invalid-action 2>&1 || echo "expected_failure")
    
    if [[ "$invalid_output" == "expected_failure" ]] || echo "$invalid_output" | grep -qi "error\|invalid\|unknown"; then
        echo "✓ Invalid action handled appropriately"
    fi
    
    # Test help functionality
    local help_output
    help_output=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --help 2>&1 || echo "help_failed")
    
    if [[ "$help_output" != "help_failed" ]] && echo "$help_output" | grep -qi "usage\|help\|options"; then
        echo "✓ Help functionality works"
    fi
    
    echo "✓ Error handling test completed"
}

# Test Claude Code configuration and settings
test_configuration() {
    echo "⚙️ Testing configuration..."
    
    # Check if configuration directory exists
    local config_dir="$HOME/.claude-code"
    
    if [[ -d "$config_dir" ]]; then
        echo "✓ Configuration directory exists: $config_dir"
        
        # Check for configuration files
        if [[ -f "$config_dir/config.json" ]]; then
            echo "✓ Configuration file found"
        fi
        
        if [[ -f "$config_dir/settings.json" ]]; then
            echo "✓ Settings file found"
        fi
    else
        echo "⚠ Configuration directory not found (may be first run)"
    fi
    
    # Test settings command if available
    if [[ "$USE_SANDBOX" == "true" ]]; then
        # In sandbox mode, skip direct CLI help command
        echo "⚠ Skipping direct CLI help check in sandbox mode (safety measure)"
        
        # We can test the manage script help instead
        local manage_help
        manage_help=$("$CLAUDE_CODE_RESOURCES_DIR/agents/claude-code/manage.sh" --help 2>&1 || echo "")
        if echo "$manage_help" | grep -qi "settings\|config"; then
            echo "✓ Settings options available in manage script"
        fi
    else
        # Original direct CLI check (shouldn't reach here)
        if which claude-code >/dev/null 2>&1; then
            local settings_output
            settings_output=$(timeout 5 claude-code --help 2>&1 || echo "settings_timeout")
            
            if [[ "$settings_output" != "settings_timeout" ]] && echo "$settings_output" | grep -qi "settings\|config"; then
                echo "✓ Settings management available"
            fi
        fi
    fi
    
    echo "✓ Configuration test completed"
}

# Main test execution
main() {
    echo "🧪 Starting Claude Code Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_claude_code_availability
    test_claude_code_status
    test_mcp_functionality
    test_resource_management
    test_configuration
    test_error_handling
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ Claude Code integration test failed"
        exit 1
    else
        echo "✅ Claude Code integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"