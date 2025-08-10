#!/usr/bin/env bats

# Claude Code CLI Mock Tests
# Tests the comprehensive claude command mocking functionality

bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    export CLAUDE_MOCK_MODE="normal"
    
    # Load the mock utilities first (required by claude mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the claude mock
    source "${BATS_TEST_DIRNAME}/claude-code.sh"
    
    # Initialize clean state for each test
    mock::claude::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    export MOCK_LOG_DIR="$TEST_LOG_DIR"
    
    # Initialize mock logging if available
    if command -v mock::init_logging &>/dev/null; then
        mock::init_logging "$TEST_LOG_DIR"
    fi
}

# Wrapper for run command that reloads claude state afterward
run_claude_command() {
    run "$@"
    # Reload state from file after claude commands that might modify state
    if [[ -n "${CLAUDE_MOCK_STATE_FILE}" && -f "$CLAUDE_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$CLAUDE_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Test cleanup
teardown() {
    # Clean up test logs
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        trash::safe_remove "$TEST_LOG_DIR" --test-cleanup
    fi
    
    # Clean up mock state files
    if [[ -n "${CLAUDE_MOCK_STATE_FILE:-}" && -f "$CLAUDE_MOCK_STATE_FILE" ]]; then
        trash::safe_remove "$CLAUDE_MOCK_STATE_FILE" --test-cleanup
    fi
    
    # Clean up environment
    unset CLAUDE_MOCK_MODE
    unset MOCK_LOG_DIR
}

# Assertion helpers for cleaner tests
assert_success() {
    [[ "$status" -eq 0 ]]
}

assert_failure() {
    [[ "$status" -ne 0 ]]
}

assert_output() {
    local expected="$1"
    [[ "$output" == "$expected" ]]
}

assert_output_contains() {
    local expected="$1"
    [[ "$output" == *"$expected"* ]]
}

assert_line_contains() {
    local expected="$1"
    local found=false
    while IFS= read -r line; do
        if [[ "$line" == *"$expected"* ]]; then
            found=true
            break
        fi
    done <<< "$output"
    [[ "$found" == true ]]
}

assert_exit_code() {
    local expected="$1"
    [[ "$status" -eq "$expected" ]]
}

# Test basic functionality
@test "claude mock loads successfully" {
    [[ "$CLAUDE_MOCKS_LOADED" == "true" ]]
}

@test "claude --version shows version information" {
    run_claude_command claude --version
    assert_success
    assert_output "claude-code v3.5.0"
}

@test "claude --help shows help message" {
    run_claude_command claude --help
    assert_success
    assert_line_contains "Claude Code - AI assistant for software development"
    assert_line_contains "Usage: claude [options] [prompt]"
    assert_line_contains "  -p, --print           Print response without interactive mode"
}

@test "claude with no arguments starts interactive mode" {
    run_claude_command claude
    assert_success
    assert_line_contains "Starting Claude interactive session..."
    assert_line_contains "Type your message and press Enter. Type 'exit' to quit."
}

# Test prompt execution
@test "claude with simple prompt generates response" {
    run_claude_command claude -p "Hello"
    assert_success
    assert_line_contains "I understand your request: \"Hello\""
}

@test "claude generates context-aware responses for status prompts" {
    run_claude_command claude -p "health check"
    assert_success
    assert_line_contains "Claude Code is running and healthy"
}

@test "claude generates context-aware responses for debug prompts" {
    run_claude_command claude -p "debug this error"
    assert_success
    assert_line_contains "I've analyzed the issue"  
    assert_line_contains "npm install missing-package"
}

@test "claude generates context-aware responses for create prompts" {
    run_claude_command claude -p "create a function"
    assert_success
    assert_line_contains "I'll help you create that feature"
    assert_line_contains "function newFeature()"
}

@test "claude generates context-aware responses for test prompts" {
    run_claude_command claude -p "write unit tests"
    assert_success
    assert_line_contains "I've created comprehensive tests"
    assert_line_contains "describe('Feature Tests'"
}

# Test output formats
@test "claude supports JSON output format" {
    run_claude_command claude -p "test prompt" --output-format json
    assert_success
    assert_line_contains "\"type\": \"message\""
    assert_line_contains "\"role\": \"assistant\""
    assert_line_contains "\"model\": \"claude-3-5-sonnet\""
}

@test "claude supports stream-json output format" {
    run_claude_command claude -p "test prompt" --output-format stream-json
    assert_success
    assert_line_contains "\"type\": \"message\""
    assert_line_contains "\"usage\":"
}

# Test model selection
@test "claude accepts model parameter" {
    run_claude_command claude -p "test prompt" --model haiku --verbose
    assert_success
    assert_line_contains "Processing prompt with model: haiku"
}

@test "claude accepts max-turns parameter" {
    run_claude_command claude -p "test prompt" --max-turns 5 --verbose
    assert_success
    assert_line_contains "Max turns: 5"
}

# Test authentication commands
@test "claude auth login works" {
    run_claude_command claude auth login
    assert_success
    assert_output "Successfully logged in to Claude"
}

@test "claude auth logout works" {
    run_claude_command claude auth logout
    assert_success
    assert_output "Successfully logged out from Claude"
}

@test "claude auth status shows status" {
    run_claude_command claude auth status
    assert_success
    assert_line_contains "Logged in as: user@example.com"
    assert_line_contains "API Key: cl-***-***"
    assert_line_contains "Model Access:"
}

@test "claude auth with invalid command shows usage" {
    run_claude_command claude auth invalid
    assert_failure
    assert_output "Usage: claude auth [login|logout|status]"
}

# Test MCP commands
@test "claude mcp list shows configured servers" {
    run_claude_command claude mcp list
    assert_success
    assert_line_contains "Configured MCP servers:"
    assert_line_contains "filesystem: @modelcontextprotocol/server-filesystem"
    assert_line_contains "browser: @modelcontextprotocol/server-brave-search"
}

@test "claude mcp add works" {
    run_claude_command claude mcp add test-server
    assert_success
    assert_output "Added MCP server: test-server"
}

@test "claude mcp remove works" {
    run_claude_command claude mcp remove test-server
    assert_success
    assert_output "Removed MCP server: test-server"
}

@test "claude mcp with no subcommand shows configuration" {
    run_claude_command claude mcp
    assert_success
    assert_output "MCP server configuration..."
}

# Test update command
@test "claude update works" {
    run_claude_command claude update
    assert_success
    assert_line_contains "Updating Claude Code..."
    assert_line_contains "Claude Code updated to latest version"
}

# Test continue mode
@test "claude --continue works" {
    run_claude_command claude --continue
    assert_success
    assert_line_contains "Continuing previous conversation..."
    assert_line_contains "I'm continuing from where we left off"
}

@test "claude --continue with JSON format" {
    run_claude_command claude --continue --output-format json
    assert_success
    assert_line_contains "\"type\": \"message\""
}

# Test session resumption
@test "claude --resume creates new session if not exists" {
    run_claude_command claude --resume test-session-123
    assert_success
    assert_line_contains "Resuming session: test-session-123"
    assert_line_contains "\"session_id\": \"test-session-123\""
}

@test "claude --resume uses existing session data" {
    # Set up existing session
    mock::claude::set_session "existing-session" '{"session_id":"existing-session","status":"active","custom":"data"}'
    
    run_claude_command claude --resume existing-session
    assert_success
    assert_line_contains "Resuming session: existing-session"
    assert_line_contains "\"custom\":\"data\""
}

# Test mock state management
@test "mock::claude::reset clears all state" {
    # Set some state
    mock::claude::set_response "test prompt" "custom response"
    
    # Verify state is working
    run_claude_command claude -p "test prompt"
    assert_success
    assert_output "custom response"
    
    # Reset state
    local reset_output
    reset_output="$(mock::claude::reset)"
    [[ "$reset_output" == "[MOCK] claude state reset" ]]
    
    # Verify state is cleared - should get default response now
    run_claude_command claude -p "test prompt"
    assert_success
    assert_line_contains "I understand your request: \"test prompt\""
}

@test "mock::claude::set_response works" {
    mock::claude::set_response "custom prompt" "custom response"
    
    run_claude_command claude -p "custom prompt"
    assert_success
    assert_output "custom response"
}

@test "mock::claude::set_session works" {
    mock::claude::set_session "test-session" '{"custom":"session_data"}'
    
    run_claude_command claude --resume test-session
    assert_success
    assert_line_contains "\"custom\":\"session_data\""
}

@test "mock::claude::set_config works" {
    mock::claude::set_config "test_key" "test_value"
    
    # Config is internal state - verify it's set
    [[ "${MOCK_CLAUDE_CONFIGS[test_key]}" == "test_value" ]]
}

# Test error injection
@test "mock::claude::inject_error for prompts works" {
    mock::claude::inject_error "prompt:test error" "auth_error"
    
    run_claude_command claude -p "test error"
    assert_failure
    assert_line_contains "Authentication required"
}

@test "mock::claude::inject_error for continue works" {
    mock::claude::inject_error "continue" "rate_limit"
    
    run_claude_command claude --continue
    assert_failure
    assert_line_contains "Rate limit exceeded"
}

@test "mock::claude::inject_error for session works" {
    mock::claude::inject_error "resume:bad-session" "session_error"
    
    run_claude_command claude --resume bad-session
    assert_failure
    assert_line_contains "Session 'bad-session' not found"
}

# Test different error types
@test "auth_error returns proper message and exit code" {
    mock::claude::inject_error "prompt:test" "auth_error"
    
    run_claude_command claude -p "test"
    assert_failure
    assert_line_contains "Authentication required"
}

@test "model_error returns proper message" {
    mock::claude::inject_error "prompt:test" "model_error"
    
    run_claude_command claude -p "test"
    assert_failure
    assert_line_contains "Model 'claude-3-5-sonnet' is not available"
}

@test "timeout error returns proper exit code" {
    mock::claude::inject_error "prompt:test" "timeout"
    
    run_claude_command claude -p "test"
    assert_failure
    assert_exit_code 124
    assert_line_contains "Request timed out"
}

# Test global mock modes
@test "CLAUDE_MOCK_MODE=error causes all commands to fail" {
    export CLAUDE_MOCK_MODE="error"
    
    run_claude_command claude -p "any prompt"
    assert_failure
    assert_line_contains "An unexpected error occurred"
}

@test "CLAUDE_MOCK_MODE=not_installed simulates command not found" {
    export CLAUDE_MOCK_MODE="not_installed"
    
    run -127 claude -p "any prompt"
    assert_exit_code 127
    assert_line_contains "claude: command not found"
}

@test "CLAUDE_MOCK_MODE=auth_error requires authentication" {
    export CLAUDE_MOCK_MODE="auth_error"
    
    run_claude_command claude -p "any prompt"
    assert_failure
    assert_line_contains "Authentication failed"
}

@test "CLAUDE_MOCK_MODE=rate_limit shows rate limiting" {
    export CLAUDE_MOCK_MODE="rate_limit"
    
    run_claude_command claude -p "any prompt"
    assert_failure
    assert_line_contains "Rate limit exceeded"
}

# Test custom exit codes
@test "mock::claude::set_exit_code works" {
    mock::claude::set_exit_code "failing prompt" 42
    
    run_claude_command claude -p "failing prompt"
    assert_exit_code 42
}

# Test call tracking
@test "claude calls are tracked in history" {
    run_claude_command claude -p "first call"
    run_claude_command claude -p "second call"
    
    local history
    history="$(mock::claude::get::call_history)"
    [[ "$history" == *"1: -p first call"* ]]
    [[ "$history" == *"2: -p second call"* ]]
}

@test "mock::claude::get::last_call returns most recent call" {
    run_claude_command claude -p "test call"
    
    local last_call
    last_call="$(mock::claude::get::last_call)"
    [[ "$last_call" == "-p test call" ]]
}

# Test scenario helpers
@test "mock::claude::scenario::setup_healthy works" {
    run_claude_command mock::claude::scenario::setup_healthy
    assert_success
    assert_output "[MOCK] Set up healthy Claude scenario"
    
    # Verify health response is set
    run_claude_command claude -p "health check"
    assert_success
    assert_line_contains "Claude Code is running and healthy"
}

@test "mock::claude::scenario::setup_auth_required works" {
    local setup_output
    setup_output="$(mock::claude::scenario::setup_auth_required)"
    [[ "$setup_output" == "[MOCK] Set up auth required scenario" ]]
    
    # Verify auth errors are injected
    run_claude_command claude -p "any prompt"
    assert_failure
    assert_line_contains "Authentication required"
}

@test "mock::claude::scenario::setup_development works" {
    run_claude_command mock::claude::scenario::setup_development
    assert_success
    assert_output "[MOCK] Set up development scenario"
    
    # Verify development responses are set
    run_claude_command claude -p "create function"
    assert_success
    assert_line_contains "function myFunction()"
}

# Test assertion helpers
@test "mock::claude::assert::response_contains works" {
    mock::claude::set_response "test prompt" "expected content here"
    
    run_claude_command mock::claude::assert::response_contains "expected content" "test prompt"
    assert_success
}

@test "mock::claude::assert::response_contains fails correctly" {
    mock::claude::set_response "test prompt" "different content"
    
    run_claude_command mock::claude::assert::response_contains "missing content" "test prompt"
    assert_failure
    assert_line_contains "ASSERTION FAILED: Response doesn't contain 'missing content'"
}

@test "mock::claude::assert::command_called works" {
    run_claude_command claude -p "test command"
    
    local result
    result="$(mock::claude::assert::command_called "test command" 2>&1)"
    [[ $? -eq 0 ]]
}

@test "mock::claude::assert::command_called fails correctly" {
    run_claude_command mock::claude::assert::command_called "never called"
    assert_failure
    assert_line_contains "ASSERTION FAILED: Command matching 'never called' was not called"
}

@test "mock::claude::assert::call_count works" {
    run_claude_command claude -p "call 1"
    run_claude_command claude -p "call 2"
    
    local result
    result="$(mock::claude::assert::call_count 2 2>&1)"
    [[ $? -eq 0 ]]
}

@test "mock::claude::assert::call_count fails correctly" {
    # Test that the assertion function correctly identifies mismatches
    # Start with 0 calls, so expecting 5 should fail
    
    # Use run to capture both success/failure and output
    run mock::claude::assert::call_count 5
    
    # Should fail since we expect 5 but have 0 calls
    assert_failure
    
    # Check that it reports the failure correctly  
    assert_line_contains "ASSERTION FAILED"
    assert_line_contains "Expected 5"
    assert_line_contains "got 0"
}

@test "mock::claude::assert::exit_code works" {
    mock::claude::set_exit_code "failing prompt" 42
    
    run_claude_command mock::claude::assert::exit_code 42 "failing prompt"
    assert_success
}

@test "mock::claude::assert::exit_code fails correctly" {
    run_claude_command mock::claude::assert::exit_code 42 "successful prompt"
    assert_failure
    assert_line_contains "ASSERTION FAILED: Expected exit code 42, got 0"
}

# Test debug functionality
@test "mock::claude::debug::dump_state shows comprehensive state" {
    # Set up some state
    mock::claude::set_response "test" "response"
    mock::claude::set_session "session-1" "session-data"
    mock::claude::set_config "key" "value"
    mock::claude::inject_error "prompt:error" "timeout"
    run_claude_command claude -p "test call"
    
    local dump_output
    dump_output="$(mock::claude::debug::dump_state)"
    [[ "$dump_output" == *"=== Claude Mock State Dump ==="* ]]
    [[ "$dump_output" == *"Call Counter: 1"* ]]
    [[ "$dump_output" == *"Custom Responses:"* ]]
    [[ "$dump_output" == *"Sessions:"* ]]
    [[ "$dump_output" == *"Configurations:"* ]]
    [[ "$dump_output" == *"Injected Errors:"* ]]
    [[ "$dump_output" == *"Call History:"* ]]
}

# Test edge cases and robustness
@test "claude handles empty prompt gracefully" {
    run_claude_command claude -p ""
    assert_success
    assert_line_contains "I understand your request"
}

@test "claude handles special characters in prompts" {
    run_claude_command claude -p "prompt with \$special &characters |and pipes"
    assert_success
    assert_line_contains "I understand your request"
}

@test "claude handles multiple flags correctly" {
    run_claude_command claude -p "test" --model haiku --output-format json --max-turns 3 --verbose
    assert_success
    assert_line_contains "\"model\": \"claude-3-5-sonnet\""
}

@test "claude ignores unknown flags gracefully" {
    run_claude_command claude -p "test" --unknown-flag --another-unknown
    assert_success
    assert_line_contains "I understand your request"
}

# Integration tests with realistic scenarios
@test "realistic development workflow" {
    # Set up development environment
    mock::claude::scenario::setup_development
    
    # Create a function
    run_claude_command claude -p "create function"
    assert_success
    assert_line_contains "function myFunction()"
    
    # Debug an error
    run_claude_command claude -p "debug error"
    assert_success
    assert_line_contains "I found the issue"
    
    # Verify call tracking works
    [[ "$CLAUDE_MOCK_CALL_COUNTER" -eq 2 ]]
}

@test "realistic authentication workflow" {
    # Start with auth required
    local setup_output
    setup_output="$(mock::claude::scenario::setup_auth_required)"
    [[ "$setup_output" == "[MOCK] Set up auth required scenario" ]]
    
    # Try to use claude - should fail
    run_claude_command claude -p "test"
    assert_failure
    assert_line_contains "Authentication required"
    
    # Log in (auth commands bypass the error injection)
    run_claude_command claude auth login
    assert_success
    
    # Check status
    run_claude_command claude auth status
    assert_success
    assert_line_contains "Logged in as:"
}

@test "realistic session management workflow" {
    # Start interactive session
    run_claude_command claude
    assert_success
    assert_line_contains "Starting Claude interactive session"
    
    # Resume a session
    run_claude_command claude --resume work-session-1
    assert_success
    assert_line_contains "Resuming session: work-session-1"
    
    # Continue conversation
    run_claude_command claude --continue
    assert_success
    assert_line_contains "Continuing previous conversation"
}

# Final validation
@test "all mock functions are exported correctly" {
    # Test that key functions are available
    command -v claude &>/dev/null
    command -v mock::claude::reset &>/dev/null
}