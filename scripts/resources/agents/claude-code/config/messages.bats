#!/usr/bin/env bats
# Tests for claude-code config/messages.sh message system


# Setup for each test
setup() {
    # Setup mock framework
    export BATS_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d)}"
    export MOCK_RESPONSES_DIR="$BATS_TEST_TMPDIR/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    # Load mock framework
    MOCK_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests/bats-fixtures/mocks"
    source "$MOCK_DIR/system_mocks.bash"
    source "$MOCK_DIR/mock_helpers.bash"
    source "$MOCK_DIR/resource_mocks.bash"
    
    # Set test environment
    export HOME="/tmp/test-home"
    export MIN_NODE_VERSION="18"
    export CLAUDE_PACKAGE="@anthropic-ai/claude-code"
    export DESCRIPTION="Claude Code CLI management"
    mkdir -p "$HOME"
    
    # Get resource directory path
    CLAUDE_CODE_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    
    # Mock logging functions
    log::info() { echo "INFO: $*"; }
    log::header() { echo "HEADER: $*"; }
    export -f log::info log::header
    
    # Mock args::usage function
    args::usage() { echo "USAGE: $*"; }
    export -f args::usage
    
    # Load configuration and messages
    source "${CLAUDE_CODE_DIR}/config/defaults.sh"
    source "${CLAUDE_CODE_DIR}/config/messages.sh"
}

teardown() {
    # Clean up test environment
    rm -rf "/tmp/test-home"
    [[ -d "$MOCK_RESPONSES_DIR" ]] && rm -rf "$MOCK_RESPONSES_DIR"
}

# Test message system functions

@test "claude_code::usage should display comprehensive usage information" {
    # Test that usage function exists and shows content
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Actions:" ]]
    [[ "$output" =~ "install" ]]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "run" ]]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "Requirements:" ]]
}

@test "claude_code::usage should include all standard actions" {
    # Test that all expected actions are listed
    run claude_code::usage
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
    [[ "$output" =~ "help" ]]
}

@test "claude_code::usage should include all MCP actions" {
    # Test that all MCP actions are listed
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MCP Actions:" ]]
    [[ "$output" =~ "register-mcp" ]]
    [[ "$output" =~ "unregister-mcp" ]]
    [[ "$output" =~ "mcp-status" ]]
    [[ "$output" =~ "mcp-test" ]]
}

@test "claude_code::usage should provide practical examples" {
    # Test that usage includes examples
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "--action install" ]]
    [[ "$output" =~ "--action run" ]]
    [[ "$output" =~ "--prompt" ]]
    [[ "$output" =~ "MCP Examples:" ]]
}

@test "claude_code::usage should include requirements" {
    # Test that requirements are specified
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Requirements:" ]]
    [[ "$output" =~ "Node.js" ]]
    [[ "$output" =~ "$MIN_NODE_VERSION" ]]
    [[ "$output" =~ "npm" ]]
    [[ "$output" =~ "Claude Pro" ]]
}

@test "claude_code::display_info should show comprehensive information" {
    # Test that info function exists and shows content
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Claude Code is Anthropic" ]]
    [[ "$output" =~ "Key Features:" ]]
    [[ "$output" =~ "Requirements:" ]]
    [[ "$output" =~ "Subscription Plans:" ]]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Documentation:" ]]
}

@test "claude_code::display_info should include key features" {
    # Test that key features are listed
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Deep codebase awareness" ]]
    [[ "$output" =~ "Direct file editing" ]]
    [[ "$output" =~ "Claude Opus 4" ]]
    [[ "$output" =~ "Composable" ]]
    [[ "$output" =~ "Automatic updates" ]]
}

@test "claude_code::display_info should include subscription information" {
    # Test that subscription plans are detailed
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Subscription Plans:" ]]
    [[ "$output" =~ "Pro" ]]
    [[ "$output" =~ "\$20/month" ]]
    [[ "$output" =~ "Max" ]]
    [[ "$output" =~ "\$100/month" ]]
}

@test "claude_code::display_info should provide installation guidance" {
    # Test that installation steps are provided
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "npm install -g" ]]
    [[ "$output" =~ "$CLAUDE_PACKAGE" ]]
    [[ "$output" =~ "cd /your/project" ]]
    [[ "$output" =~ "claude" ]]
}

@test "claude_code::display_info should include support links" {
    # Test that documentation and support links are provided
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Documentation:" ]]
    [[ "$output" =~ "docs.anthropic.com" ]]
    [[ "$output" =~ "Support:" ]]
    [[ "$output" =~ "support.anthropic.com" ]]
}

@test "claude_code::mcp_next_steps should provide clear guidance" {
    # Test that MCP next steps function works
    run claude_code::mcp_next_steps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER:" ]]
    [[ "$output" =~ "Next Steps" ]]
    [[ "$output" =~ "INFO:" ]]
    [[ "$output" =~ "claude" ]]
    [[ "$output" =~ "@vrooli" ]]
}

@test "claude_code::mcp_next_steps should include Vrooli-specific instructions" {
    # Test that Vrooli MCP instructions are provided
    run claude_code::mcp_next_steps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "@vrooli send_message" ]]
    [[ "$output" =~ "@vrooli run_routine" ]]
    [[ "$output" =~ "@vrooli spawn_swarm" ]]
    [[ "$output" =~ "Type '@'" ]]
}

@test "claude_code::install_next_steps should provide installation guidance" {
    # Test that installation next steps function works
    run claude_code::install_next_steps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER:" ]]
    [[ "$output" =~ "Next Steps" ]]
    [[ "$output" =~ "INFO:" ]]
    [[ "$output" =~ "Navigate to your project" ]]
    [[ "$output" =~ "Start Claude Code" ]]
}

@test "claude_code::install_next_steps should include verification steps" {
    # Test that verification instructions are provided
    run claude_code::install_next_steps
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Login with your Claude credentials" ]]
    [[ "$output" =~ "claude doctor" ]]
    [[ "$output" =~ "verify your installation" ]]
}

@test "functions should handle missing dependencies gracefully" {
    # Test without log functions
    unset -f log::info log::header
    
    run claude_code::mcp_next_steps
    # Should not fail completely
    [ "$status" -eq 0 ]
}

@test "usage function should handle missing args::usage gracefully" {
    # Test without args::usage
    unset -f args::usage
    
    run claude_code::usage
    # Should still show basic information
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Actions:" ]]
}

@test "display_info should reference configuration variables" {
    # Test that display_info uses configuration variables
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$MIN_NODE_VERSION" ]]
    [[ "$output" =~ "$CLAUDE_PACKAGE" ]]
}

@test "usage should show meaningful action descriptions" {
    # Test that action descriptions are meaningful
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install Claude Code CLI globally" ]]
    [[ "$output" =~ "Check installation status" ]]
    [[ "$output" =~ "Run Claude with a prompt" ]]
    [[ "$output" =~ "Manage Claude sessions" ]]
    [[ "$output" =~ "Register Vrooli as MCP server" ]]
}

@test "examples should show proper command syntax" {
    # Test that examples use correct syntax
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--action install" ]]
    [[ "$output" =~ "--prompt" ]]
    [[ "$output" =~ "--max-turns" ]]
    [[ "$output" =~ "--session-id" ]]
    [[ "$output" =~ "--scope user" ]]
    [[ "$output" =~ "--format json" ]]
}

@test "MCP examples should be Vrooli-specific" {
    # Test that MCP examples are relevant to Vrooli
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "register-mcp" ]]
    [[ "$output" =~ "Auto-register Vrooli" ]]
    [[ "$output" =~ "mcp-status" ]]
    [[ "$output" =~ "mcp-test" ]]
    [[ "$output" =~ "unregister-mcp" ]]
}

@test "functions should provide actionable information" {
    # Test that information is actionable
    run claude_code::display_info
    [ "$status" -eq 0 ]
    
    # Should provide specific commands
    [[ "$output" =~ "npm install -g" ]]
    [[ "$output" =~ "cd /your/project" ]]
    
    # Should provide specific links
    [[ "$output" =~ "https://docs.anthropic.com" ]]
    [[ "$output" =~ "https://support.anthropic.com" ]]
}

@test "next steps should be logically ordered" {
    # Test that install next steps are in logical order
    run claude_code::install_next_steps
    [ "$status" -eq 0 ]
    
    lines=()
    while IFS= read -r line; do
        lines+=("$line")
    done <<< "$output"
    
    # Should mention navigation before starting Claude
    navigation_found=false
    start_found=false
    for line in "${lines[@]}"; do
        if [[ "$line" =~ "Navigate to your project" ]]; then
            navigation_found=true
        elif [[ "$line" =~ "Start Claude Code" ]] && [ "$navigation_found" = true ]; then
            start_found=true
            break
        fi
    done
    [ "$start_found" = true ]
}

@test "messages should be consistent with Claude Code branding" {
    # Test that messages use consistent terminology
    run claude_code::display_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Claude Code" ]]
    [[ "$output" =~ "Anthropic" ]]
    
    run claude_code::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Claude Code" || "$output" =~ "Claude" ]]
}

@test "functions should handle environment variables correctly" {
    # Test with missing environment variables
    unset MIN_NODE_VERSION CLAUDE_PACKAGE
    
    run claude_code::usage
    [ "$status" -eq 0 ]
    # Should still provide basic functionality
    [[ "$output" =~ "Actions:" ]]
    
    run claude_code::display_info
    [ "$status" -eq 0 ]
    # Should still provide basic information
    [[ "$output" =~ "Claude Code" ]]
}
