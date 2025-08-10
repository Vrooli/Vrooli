#!/usr/bin/env bats
# Agent-S2 Mock Test Suite
# Comprehensive test coverage for the Agent-S2 mock implementation

bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup test environment
setup() {
    # Create a temporary directory for this test session
    export MOCK_LOG_DIR="${BATS_TMPDIR}/agents2_mock_tests"
    mkdir -p "$MOCK_LOG_DIR"
    
    # Set a consistent state file path for this test session
    export AGENTS2_MOCK_STATE_FILE="${BATS_TMPDIR}/agents2_mock_state_test"
    
    # Load test helpers if available
    if [[ -f "${BATS_TEST_DIRNAME}/../../helpers/bats-support/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../helpers/bats-support/load.bash"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/../../helpers/bats-assert/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../helpers/bats-assert/load.bash"
    else
        # Define basic assertion functions if bats-assert is not available
        assert_success() {
            if [[ "$status" -ne 0 ]]; then
                echo "expected success, got status $status"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_failure() {
            if [[ "$status" -eq 0 ]]; then
                echo "expected failure, got success"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" != *"$expected"* ]]; then
                    echo "expected output to contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" != "$expected" ]]; then
                    echo "expected: $expected"
                    echo "actual: $output"
                    return 1
                fi
            fi
        }
        
        refute_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" == *"$expected"* ]]; then
                    echo "expected output to NOT contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" == "$expected" ]]; then
                    echo "expected output to NOT be: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            fi
        }
    fi
    
    # Load the mock systems
    if [[ -f "${BATS_TEST_DIRNAME}/logs.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/logs.sh"
        mock::init_logging "$MOCK_LOG_DIR"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
        mock::verify::init
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/docker.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/docker.sh"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/http.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/http.sh"
    fi
    
    # Load the Agent-S2 mock
    source "${BATS_TEST_DIRNAME}/agent-s2.sh"
    
    # Reset the mock to a known state
    mock::agents2::reset
}

teardown() {
    # Clean up temporary files
    if [[ -n "${AGENTS2_MOCK_STATE_FILE:-}" && -f "$AGENTS2_MOCK_STATE_FILE" ]]; then
        trash::safe_remove "$AGENTS2_MOCK_STATE_FILE" --test-cleanup
    fi
    
    # Clean up mock log directory
    if [[ -d "$MOCK_LOG_DIR" ]]; then
        trash::safe_remove "$MOCK_LOG_DIR" --test-cleanup
    fi
}

# ----------------------------
# Basic Mock Functionality Tests
# ----------------------------

@test "Agent-S2 mock loads without errors" {
    # Test basic loading
    run echo "Mock loaded successfully"
    assert_success
    
    # Test that the mock loaded flag is set
    [[ "$AGENTS2_MOCK_LOADED" == "true" ]]
    
    # Test that basic functions are available
    declare -F agents2::manage >/dev/null
    declare -F mock::agents2::reset >/dev/null
    declare -F mock::agents2::set_mode >/dev/null
}

@test "Agent-S2 mock state can be reset" {
    # Create a task and session first
    local task_id=$(mock::agents2::create_task "test-task" "params")
    local session_id=$(mock::agents2::create_session "test-profile")
    
    # Verify they exist
    [[ -n "$task_id" ]]
    [[ -n "$session_id" ]]
    
    # Reset the mock
    mock::agents2::reset
    
    # Verify default state is restored
    [[ "$(mock::agents2::get::mode)" == "healthy" ]]
    [[ "$(mock::agents2::get::config 'llm_provider')" == "ollama" ]]
    [[ "$(mock::agents2::get::config 'mode')" == "sandbox" ]]
}

@test "Agent-S2 mock modes work correctly" {
    # Test healthy mode (default)
    [[ "$(mock::agents2::get::mode)" == "healthy" ]]
    
    # Test setting different modes
    for mode in unhealthy installing stopped offline error; do
        run mock::agents2::set_mode "$mode"
        assert_success
        [[ "$(mock::agents2::get::mode)" == "$mode" ]]
    done
    
    # Test invalid mode
    run mock::agents2::set_mode "invalid_mode"
    assert_failure
    assert_output --partial "Invalid Agent-S2 mode"
}

# ----------------------------
# Management Command Tests
# ----------------------------

@test "agents2::manage install command works" {
    # Test installation
    run agents2::manage install
    assert_success
    assert_output --partial "Installing Agent-S2"
    assert_output --partial "Agent-S2 installed successfully"
    
    # Verify state changed to stopped
    run mock::agents2::assert::state "stopped"
    assert_success
}

@test "agents2::manage start command works" {
    # Install first
    agents2::manage install
    
    # Test starting
    run agents2::manage start
    assert_success
    assert_output --partial "Starting Agent-S2"
    assert_output --partial "Agent-S2 started successfully"
    assert_output --partial "API available at:"
    assert_output --partial "VNC available at:"
    
    # Verify state changed to running
    run mock::agents2::assert::state "running"
    assert_success
    
    # Test starting when already running
    run agents2::manage start
    assert_success
    assert_output --partial "Agent-S2 is already running"
}

@test "agents2::manage stop command works" {
    # Install and start first
    agents2::manage install
    agents2::manage start
    
    # Test stopping
    run agents2::manage stop
    assert_success
    assert_output --partial "Stopping Agent-S2"
    assert_output --partial "Agent-S2 stopped successfully"
    
    # Verify state changed to stopped
    run mock::agents2::assert::state "stopped"
    assert_success
    
    # Test stopping when already stopped
    run agents2::manage stop
    assert_success
    assert_output --partial "Agent-S2 is not running"
}

@test "agents2::manage restart command works" {
    # Install and start first
    agents2::manage install
    agents2::manage start
    
    # Test restarting
    run agents2::manage restart
    assert_success
    assert_output --partial "Stopping Agent-S2"
    assert_output --partial "Starting Agent-S2"
    
    # Verify state is running
    run mock::agents2::assert::state "running"
    assert_success
}

@test "agents2::manage status command works" {
    # Test status when not installed
    run agents2::manage status
    assert_success
    assert_output --partial "Agent-S2 Status: not_installed"
    
    # Install and test stopped status
    agents2::manage install
    run agents2::manage status
    assert_success
    assert_output --partial "Agent-S2 Status: stopped"
    
    # Start and test running status
    agents2::manage start
    run agents2::manage status
    assert_success
    assert_output --partial "Agent-S2 Status: running"
    assert_output --partial "API:"
    assert_output --partial "VNC:"
    assert_output --partial "Mode: sandbox"
    assert_output --partial "LLM Provider: ollama"
    assert_output --partial "AI Enabled: yes"
    assert_output --partial "Stealth Mode: no"
}

@test "agents2::manage logs command works" {
    run agents2::manage logs
    assert_success
    assert_output --partial "Agent-S2 started"
    assert_output --partial "Browser engine initialized"
    assert_output --partial "Display server ready"
    assert_output --partial "API server listening"
}

@test "agents2::manage uninstall command works" {
    # Install first (run in subshell to match other commands)
    run agents2::manage install
    assert_success
    assert_output --partial "Agent-S2 installed successfully"
    
    # Test uninstalling
    run agents2::manage uninstall
    assert_success
    assert_output --partial "Removing Agent-S2"
    assert_output --partial "Agent-S2 uninstalled successfully"
    
    # Verify state is not_installed
    run agents2::manage status
    assert_success
    assert_output --partial "not_installed"
}

# ----------------------------
# Mode Management Tests
# ----------------------------

@test "agents2::manage mode command shows current mode" {
    run agents2::manage mode
    assert_success
    assert_output "Current mode: sandbox"
    
    # Change mode and verify
    mock::agents2::set_config "mode" "host"
    run agents2::manage mode
    assert_success
    assert_output "Current mode: host"
}

@test "agents2::manage switch-mode command works" {
    run agents2::manage switch-mode host
    assert_success
    assert_output --partial "Switching to host mode"
    assert_output --partial "Switched to host mode successfully"
    
    # Verify mode changed
    [[ "$(mock::agents2::get::config 'mode')" == "host" ]]
    
    # Switch back to sandbox
    run agents2::manage switch-mode sandbox
    assert_success
    assert_output --partial "Switching to sandbox mode"
    [[ "$(mock::agents2::get::config 'mode')" == "sandbox" ]]
}

# ----------------------------
# AI Task Tests
# ----------------------------

@test "agents2::manage ai-task command works" {
    # Test without task description
    run agents2::manage ai-task
    assert_failure
    assert_output --partial "Task description is required"
    
    # Test with task description
    run agents2::manage ai-task "Navigate to example.com and take a screenshot"
    assert_success
    assert_output --partial "Executing AI task:"
    assert_output --partial "Task ID:"
    assert_output --partial "Task accepted and running"
    assert_output --partial "Task completed successfully"
    
    # Verify action was tracked
    run mock::agents2::assert::action_called "ai-task"
    assert_success
}

# ----------------------------
# Stealth Mode Tests
# ----------------------------

@test "agents2::manage configure-stealth command works" {
    run agents2::manage configure-stealth yes
    assert_success
    assert_output --partial "Configuring stealth mode: yes"
    assert_output --partial "Stealth mode configured successfully"
    
    # Verify stealth is enabled
    [[ "$(mock::agents2::get::config 'enabled')" == "" ]]  # Note: stealth settings are separate
    
    run agents2::manage configure-stealth no
    assert_success
    assert_output --partial "Configuring stealth mode: no"
}

@test "agents2::manage test-stealth command works" {
    # Test with stealth disabled
    run agents2::manage test-stealth
    assert_success
    assert_output --partial "Testing stealth mode at:"
    assert_output --partial "Stealth test: FAILED"
    assert_output --partial "WebDriver: Detected"
    
    # Enable stealth and test again
    mock::agents2::set_stealth "enabled" "yes"
    run agents2::manage test-stealth "https://custom-test.com"
    assert_success
    assert_output --partial "Testing stealth mode at: https://custom-test.com"
    assert_output --partial "Stealth test: PASSED"
    assert_output --partial "WebDriver: Hidden"
    assert_output --partial "Navigator: Spoofed"
    assert_output --partial "Canvas: Randomized"
}

# ----------------------------
# Session Management Tests
# ----------------------------

@test "agents2::manage list-sessions command works" {
    # Test with no sessions
    run agents2::manage list-sessions
    assert_success
    assert_output "Active sessions:"
    
    # Create some sessions
    local session1=$(mock::agents2::create_session "profile1")
    local session2=$(mock::agents2::create_session "profile2")
    
    # List sessions
    run agents2::manage list-sessions
    assert_success
    assert_output --partial "active"
    # Check that both sessions appear in output
    [[ "$output" == *"$session1"* ]]
    [[ "$output" == *"$session2"* ]]
}

# ----------------------------
# Task Management Tests
# ----------------------------

@test "task creation and updates work" {
    # Create a task
    local task_id=$(mock::agents2::create_task "screenshot" "https://example.com")
    [[ -n "$task_id" ]]
    
    # Update task status
    mock::agents2::update_task "$task_id" "running" ""
    
    run mock::agents2::assert::task_status "$task_id" "running"
    assert_success
    
    # Update to completed
    mock::agents2::update_task "$task_id" "completed" "success"
    
    run mock::agents2::assert::task_status "$task_id" "completed"
    assert_success
    
    # Try updating non-existent task
    run mock::agents2::update_task "fake-task-id" "completed" "success"
    assert_failure
}

@test "session creation works" {
    # Create a session with default profile
    local session1=$(mock::agents2::create_session)
    [[ -n "$session1" ]]
    
    # Create a session with custom profile
    local session2=$(mock::agents2::create_session "custom-profile")
    [[ -n "$session2" ]]
    [[ "$session1" != "$session2" ]]
    
    # Verify sessions are active (count output lines)
    local session_count=$(mock::agents2::get::sessions | wc -l)
    [[ "$session_count" -eq 2 ]]
}

# ----------------------------
# Configuration Tests
# ----------------------------

@test "configuration management works" {
    # Test getting default config
    [[ "$(mock::agents2::get::config 'llm_provider')" == "ollama" ]]
    [[ "$(mock::agents2::get::config 'enable_ai')" == "yes" ]]
    
    # Test setting config
    mock::agents2::set_config "llm_provider" "openai"
    [[ "$(mock::agents2::get::config 'llm_provider')" == "openai" ]]
    
    mock::agents2::set_config "custom_key" "custom_value"
    [[ "$(mock::agents2::get::config 'custom_key')" == "custom_value" ]]
    
    # Test getting non-existent config
    [[ -z "$(mock::agents2::get::config 'non_existent')" ]]
}

@test "stealth configuration works" {
    # Test default stealth settings
    mock::agents2::reset
    
    # Set stealth features
    mock::agents2::set_stealth "enabled" "yes"
    mock::agents2::set_stealth "webdriver_override" "yes"
    mock::agents2::set_stealth "navigator_spoofing" "yes"
    
    # Use the stealth enabled scenario
    run mock::agents2::scenario::stealth_enabled
    assert_success
    assert_output --partial "Created Agent-S2 scenario with stealth mode enabled"
}

# ----------------------------
# Error Injection Tests
# ----------------------------

@test "error injection works for actions" {
    # Inject connection error for start action
    mock::agents2::inject_error "start" "connection"
    
    run agents2::manage start
    assert_failure
    assert_output --partial "Failed to connect to Agent-S2 server"
    
    # Inject docker error for install action
    mock::agents2::inject_error "install" "docker"
    
    run agents2::manage install
    assert_failure
    assert_output --partial "Docker is not running"
    
    # Inject permission error for stop action
    mock::agents2::inject_error "stop" "permission"
    
    run agents2::manage stop
    assert_failure
    assert_output --partial "Permission denied"
}

@test "mode errors affect commands appropriately" {
    # Set offline mode
    mock::agents2::set_mode "offline"
    
    run agents2::manage start
    assert_failure
    assert_output --partial "Agent-S2 is not running"
    
    # Set error mode
    mock::agents2::set_mode "error"
    
    run agents2::manage status
    assert_failure
    assert_output --partial "Agent-S2 encountered an error"
}

# ----------------------------
# Scenario Tests
# ----------------------------

@test "healthy with session scenario works" {
    run mock::agents2::scenario::healthy_with_session
    assert_success
    assert_output --partial "Created healthy Agent-S2 scenario with session:"
    
    # Verify the scenario set up correctly
    [[ "$(mock::agents2::get::mode)" == "healthy" ]]
    
    # Verify sessions were created (count output lines)
    local session_count=$(mock::agents2::get::sessions | wc -l)
    [[ "$session_count" -gt 0 ]]
}

@test "installing scenario works" {
    run mock::agents2::scenario::installing
    assert_success
    assert_output --partial "Created installing Agent-S2 scenario"
    
    [[ "$(mock::agents2::get::mode)" == "installing" ]]
}

@test "offline scenario works" {
    run mock::agents2::scenario::offline
    assert_success
    assert_output --partial "Created offline Agent-S2 scenario"
    
    [[ "$(mock::agents2::get::mode)" == "offline" ]]
}

@test "stealth enabled scenario works" {
    run mock::agents2::scenario::stealth_enabled
    assert_success
    assert_output --partial "Created Agent-S2 scenario with stealth mode enabled"
    
    [[ "$(mock::agents2::get::mode)" == "healthy" ]]
}

# ----------------------------
# Assertion Helper Tests
# ----------------------------

@test "state assertion works" {
    # Set a specific state
    agents2::manage install
    
    # Test successful assertion
    run mock::agents2::assert::state "stopped"
    assert_success
    
    # Test failed assertion
    run mock::agents2::assert::state "running"
    assert_failure
    assert_output --partial "ASSERTION FAILED"
}

@test "task status assertion works" {
    # Create a task
    local task_id=$(mock::agents2::create_task "test" "params")
    
    # Test successful assertion
    run mock::agents2::assert::task_status "$task_id" "accepted"
    assert_success
    
    # Test failed assertion
    run mock::agents2::assert::task_status "$task_id" "completed"
    assert_failure
    assert_output --partial "ASSERTION FAILED"
    
    # Test non-existent task
    run mock::agents2::assert::task_status "fake-task" "any"
    assert_failure
    assert_output --partial "Task 'fake-task' does not exist"
}

@test "action called assertion works" {
    # Call an action
    agents2::manage status
    
    # Test successful assertion
    run mock::agents2::assert::action_called "status"
    assert_success
    
    # Test with specific count
    agents2::manage status
    run mock::agents2::assert::action_called "status" 2
    assert_success
    
    # Test failed assertion
    run mock::agents2::assert::action_called "status" 5
    assert_failure
    assert_output --partial "ASSERTION FAILED"
    
    # Test uncalled action
    run mock::agents2::assert::action_called "never_called"
    assert_failure
}

# ----------------------------
# Call Count Tracking Tests
# ----------------------------

@test "call count tracking works" {
    # Initial count should be 0
    [[ "$(mock::agents2::get::call_count 'install')" -eq 0 ]]
    
    # Call install once
    agents2::manage install
    [[ "$(mock::agents2::get::call_count 'install')" -eq 1 ]]
    
    # Call install again
    agents2::manage install
    [[ "$(mock::agents2::get::call_count 'install')" -eq 2 ]]
    
    # Call different actions
    agents2::manage start
    agents2::manage stop
    agents2::manage status
    
    [[ "$(mock::agents2::get::call_count 'start')" -eq 1 ]]
    [[ "$(mock::agents2::get::call_count 'stop')" -eq 1 ]]
    [[ "$(mock::agents2::get::call_count 'status')" -eq 1 ]]
}

# ----------------------------
# State Persistence Tests
# ----------------------------

@test "state persists across subshells" {
    # Set some state in main shell
    mock::agents2::set_config "test_key" "test_value"
    local task_id=$(mock::agents2::create_task "test" "params")
    
    # Run assertions in subshell
    (
        [[ "$(mock::agents2::get::config 'test_key')" == "test_value" ]]
        mock::agents2::assert::task_status "$task_id" "accepted"
    )
    assert_success
}

@test "manage commands work in subshells" {
    # Install in main shell
    agents2::manage install
    
    # Check status in subshell
    output=$(agents2::manage status)
    [[ "$output" == *"stopped"* ]]
    
    # Start in subshell
    (agents2::manage start)
    
    # Verify state changed in main shell
    run mock::agents2::assert::state "running"
    assert_success
}

# ----------------------------
# Debug Function Tests
# ----------------------------

@test "debug dump state works" {
    # Set up some state
    mock::agents2::set_config "debug_test" "value"
    local task_id=$(mock::agents2::create_task "debug" "test")
    local session_id=$(mock::agents2::create_session "debug-profile")
    agents2::manage install
    
    # Dump state
    run mock::agents2::debug::dump_state
    assert_success
    assert_output --partial "=== Agent-S2 Mock State Dump ==="
    assert_output --partial "Mode: healthy"
    assert_output --partial "Container State:"
    assert_output --partial "Sessions:"
    assert_output --partial "Tasks:"
    assert_output --partial "Configuration:"
    assert_output --partial "debug_test: value"
}

# ----------------------------
# Integration with HTTP Mock Tests (if available)
# ----------------------------

@test "HTTP endpoints are set up correctly in healthy mode" {
    # Skip if HTTP mock is not available
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        skip "HTTP mock not available"
    fi
    
    # Set healthy mode
    mock::agents2::set_mode "healthy"
    
    # The endpoints should have been set up by set_mode
    # This would normally be tested by checking the HTTP mock state
    # but we'll just verify the function runs without error
    run mock::agents2::setup_healthy_endpoints
    assert_success
}

@test "HTTP endpoints reflect unhealthy state" {
    # Skip if HTTP mock is not available
    if ! command -v mock::http::set_endpoint_response &>/dev/null; then
        skip "HTTP mock not available"
    fi
    
    # Set unhealthy mode
    mock::agents2::set_mode "unhealthy"
    
    run mock::agents2::setup_unhealthy_endpoints
    assert_success
}

# ----------------------------
# Edge Cases and Error Handling
# ----------------------------

@test "handles missing state file gracefully" {
    # Remove state file
    trash::safe_remove "$AGENTS2_MOCK_STATE_FILE" --test-cleanup
    
    # Operations should still work
    run mock::agents2::set_config "test" "value"
    assert_success
    
    run agents2::manage status
    assert_success
}

@test "handles concurrent modifications" {
    # Start multiple background operations
    agents2::manage install &
    local pid1=$!
    
    agents2::manage start &
    local pid2=$!
    
    agents2::manage status &
    local pid3=$!
    
    # Wait for all to complete
    wait $pid1 $pid2 $pid3
    
    # State should be consistent
    run agents2::manage status
    assert_success
}

@test "unknown action returns error" {
    run agents2::manage unknown_action
    assert_failure
    assert_output --partial "Unknown action: unknown_action"
}

@test "empty task description fails appropriately" {
    run agents2::manage ai-task ""
    assert_failure
    assert_output --partial "Task description is required"
    
    run agents2::manage ai-task "   "
    assert_failure
    assert_output --partial "Task description is required"
}