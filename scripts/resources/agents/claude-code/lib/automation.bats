#!/usr/bin/env bats
# Tests for Claude Code automation.sh functions
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code-automation"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}/.."
    
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" || true
    
    # Load configuration if available
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
    fi
    
    # Load the automation functions
    source "${BATS_TEST_DIRNAME}/automation.sh"
    
    # Export functions for subshells
    export -f claude_code::parse_result
    export -f claude_code::extract
    export -f claude_code::session_manage
    export -f claude_code::health_check
    export -f claude_code::run_automation
}

setup() {
    # Reset mocks for each test
    mock::claude::reset >/dev/null 2>&1 || true
    
    # Create temporary test files
    export TEST_OUTPUT_DIR="${BATS_TMPDIR}/automation-test"
    mkdir -p "$TEST_OUTPUT_DIR"
}

teardown() {
    # Clean up test files
    trash::safe_remove "$TEST_OUTPUT_DIR" --test-cleanup
}

#######################################
# Parse Result Tests
#######################################

@test "claude_code::parse_result - should handle missing output file" {
    run claude_code::parse_result "/nonexistent/file" "json"
    
    assert_failure
    assert_output --partial "No output file or empty output"
}

@test "claude_code::parse_result - should handle empty output file" {
    local empty_file="${TEST_OUTPUT_DIR}/empty.json"
    touch "$empty_file"
    
    run claude_code::parse_result "$empty_file" "json"
    
    assert_failure
    assert_output --partial "No output file or empty output"
}

@test "claude_code::parse_result - should parse valid stream-json output" {
    local test_file="${TEST_OUTPUT_DIR}/test_output.json"
    cat > "$test_file" << 'EOF'
{"type": "message", "role": "assistant", "content": [{"type": "text", "text": "Hello, World!"}]}
EOF
    
    run claude_code::parse_result "$test_file" "json"
    
    assert_success
    assert_output --partial "Hello, World!"
}

#######################################
# Extract Tests
#######################################

@test "claude_code::extract - should handle missing input file" {
    run claude_code::extract "/nonexistent/file" "status"
    
    assert_failure
    assert_output --partial "Input file not found"
}

@test "claude_code::extract - should extract status from valid file" {
    local test_file="${TEST_OUTPUT_DIR}/test_status.json"
    cat > "$test_file" << 'EOF'
{"status": "success", "data": {"result": "completed"}}
EOF
    
    run claude_code::extract "$test_file" "status"
    
    assert_success
    assert_output --partial "success"
}

#######################################
# Health Check Tests
#######################################

@test "claude_code::health_check - should perform basic health check" {
    # Mock successful claude command
    mock::claude::scenario::setup_healthy
    
    run claude_code::health_check "basic" "text"
    
    assert_success
    assert_output --partial "Claude Code is running and healthy"
}

@test "claude_code::health_check - should handle authentication errors" {
    # Mock authentication error
    mock::claude::scenario::setup_auth_required
    
    run claude_code::health_check "basic" "text"
    
    assert_failure
    assert_output --partial "Authentication required"
}

#######################################
# Session Management Tests
#######################################

@test "claude_code::session_manage - should list sessions in text format" {
    # Mock session listing
    mock::claude::set_response "session list" "Session 1: active
Session 2: completed"
    
    run claude_code::session_manage "list" "" "text"
    
    assert_success
    assert_output --partial "Session"
}

@test "claude_code::session_manage - should manage specific session" {
    local test_session_id="test-session-123"
    
    # Mock session management
    mock::claude::set_response "session manage $test_session_id" "Session $test_session_id managed successfully"
    
    run claude_code::session_manage "manage" "$test_session_id" "text"
    
    assert_success
    assert_output --partial "managed successfully"
}

#######################################
# Run Automation Tests
#######################################

@test "claude_code::run_automation - should execute automation with prompt" {
    local test_prompt="Create a simple function"
    local test_tools="bash,edit"
    
    # Mock automation execution
    mock::claude::scenario::setup_development
    
    run claude_code::run_automation "$test_prompt" "$test_tools" "5" ""
    
    assert_success
    # Should output a temporary file path
    assert_output --regexp ".*\.json$"
}

@test "claude_code::run_automation - should handle authentication errors" {
    local test_prompt="Test prompt"
    
    # Mock authentication error
    mock::claude::scenario::setup_auth_required
    
    run claude_code::run_automation "$test_prompt" "" "5" ""
    
    assert_failure
}

@test "claude_code::run_automation - should handle empty prompt" {
    run claude_code::run_automation "" "" "5" ""
    
    assert_failure
    assert_output --partial "Prompt is required"
}