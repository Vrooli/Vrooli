#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/lib/network"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load test helpers  
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/ports.sh"

setup() {
    # Setup will load the required mocks via setup.bash
    vrooli_setup_unit_test
    
    # Reset mocks if functions are available (Tier 2 mock naming)
    if declare -F logs_mock_reset >/dev/null 2>&1; then
        logs_mock_reset
    fi
    if declare -F system_mock_reset >/dev/null 2>&1; then
        system_mock_reset
    fi
    
    # Set up test environment variables
    export YES=""
    export SUDO_MODE="test"  # Avoid the "skip" mode that outputs messages
    export MOCK_LSOF_AVAILABLE="true"  # Default to lsof being available
    export MOCK_KILL_SUCCESS="true"    # Default to kill succeeding
    export MOCK_CONFIRM_RESPONSE="no"  # Default to no confirmation
}

teardown() {
    vrooli_cleanup_test
    
    # Clean up environment variables
    unset YES SUDO_MODE MOCK_KILL_SUCCESS MOCK_CONFIRM_RESPONSE || true
    unset MOCK_LSOF_OUTPUT MOCK_LSOF_EXIT_CODE MOCK_LSOF_AVAILABLE || true
}

# Helper functions for mocking

# Mock system::is_command to control lsof availability
mock_system_is_command() {
    system::is_command() {
        case "$1" in
            lsof)
                [[ "${MOCK_LSOF_AVAILABLE:-true}" == "true" ]]
                ;;
            *)
                command -v "$1" >/dev/null 2>&1
                ;;
        esac
    }
    export -f system::is_command
}

# Mock lsof command behavior
mock_lsof() {
    lsof() {
        local args="$*"
        
        # Check if we're being called with the expected arguments
        if [[ "$args" == *"-tiTCP:"* ]] && [[ "$args" == *"-sTCP:LISTEN"* ]]; then
            local exit_code="${MOCK_LSOF_EXIT_CODE:-0}"
            
            if [[ -n "${MOCK_LSOF_OUTPUT:-}" ]]; then
                echo "${MOCK_LSOF_OUTPUT}"
            fi
            
            return "$exit_code"
        fi
        
        return 1
    }
    export -f lsof
}

# Mock flow functions for testing
mock_flow_functions() {
    # Mock flow::can_run_sudo to control sudo behavior
    flow::can_run_sudo() {
        return 0  # Always allow sudo in tests
    }
    export -f flow::can_run_sudo
    
    # Mock sudo for kill commands
    sudo() {
        if [[ "$1" == "kill" ]]; then
            [[ "${MOCK_KILL_SUCCESS:-true}" == "true" ]]
        else
            "$@"  # Run the actual command for other cases
        fi
    }
    export -f sudo
    
    # Mock flow::confirm for interactive tests
    flow::confirm() {
        [[ "${MOCK_CONFIRM_RESPONSE:-no}" == "yes" ]]
    }
    export -f flow::confirm
    
    # Mock flow::is_yes for auto-yes tests
    flow::is_yes() {
        [[ "$1" == "yes" ]] || [[ "$1" == "YES" ]] || [[ "$1" == "y" ]] || [[ "$1" == "Y" ]]
    }
    export -f flow::is_yes
}

# =============================================================================
# Tests for ports::validate_port
# =============================================================================

@test "ports::validate_port accepts valid port 80" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port 80
    assert_success
}

@test "ports::validate_port accepts valid port 65535" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port 65535
    assert_success
}

@test "ports::validate_port accepts valid port 1" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port 1
    assert_success
}

@test "ports::validate_port rejects empty port" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port ""
    assert_failure
    assert_output --partial "Invalid port: ''"
}

@test "ports::validate_port rejects non-numeric port" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port "abc"
    assert_failure
    assert_output --partial "Invalid port: 'abc'"
}

@test "ports::validate_port rejects port 0" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port 0
    assert_failure
    assert_output --partial "Port out of range: 0"
}

@test "ports::validate_port rejects port 65536" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port 65536
    assert_failure
    assert_output --partial "Port out of range: 65536"
}

@test "ports::validate_port rejects negative port numbers" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::validate_port -1
    assert_failure
    assert_output --partial "Invalid port: '-1'"
}

@test "ports::validate_port handles leading zeros correctly" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    # Should interpret as decimal 80, not octal
    run ports::validate_port 0080
    assert_success
}

# =============================================================================
# Tests for ports::preflight
# =============================================================================

@test "ports::preflight succeeds when lsof is available" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::preflight
    assert_success
}

@test "ports::preflight fails when lsof is not available" {
    export MOCK_LSOF_AVAILABLE="false"
    source "$SCRIPT_PATH"
    mock_system_is_command
    
    run ports::preflight
    assert_failure
    assert_output --partial "Required command 'lsof' not found"
}

# =============================================================================
# Tests for ports::get_listening_pids
# =============================================================================

@test "ports::get_listening_pids returns PIDs when port has listeners" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT="1234
5678"
    export MOCK_LSOF_EXIT_CODE=0
    
    run ports::get_listening_pids 8080
    assert_success
    assert_output "1234
5678"
}

@test "ports::get_listening_pids returns empty when no listeners" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT=""
    export MOCK_LSOF_EXIT_CODE=1
    
    run ports::get_listening_pids 8080
    assert_success
    assert_output ""
}

@test "ports::get_listening_pids fails on lsof error" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT="Permission denied"
    export MOCK_LSOF_EXIT_CODE=2
    
    run ports::get_listening_pids 8080
    assert_failure
    assert_output --partial "Error checking port 8080"
}

@test "ports::get_listening_pids validates port before checking" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    run ports::get_listening_pids "invalid"
    assert_failure
    assert_output --partial "Invalid port"
}

# =============================================================================
# Tests for ports::is_port_in_use
# =============================================================================

@test "ports::is_port_in_use returns true when port has listeners" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    
    run ports::is_port_in_use 8080
    assert_success
}

@test "ports::is_port_in_use returns false when port has no listeners" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT=""
    export MOCK_LSOF_EXIT_CODE=1
    
    run ports::is_port_in_use 8080
    assert_failure
}

# =============================================================================
# Tests for ports::kill
# =============================================================================

@test "ports::kill kills processes on port successfully" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234 5678"
    export MOCK_LSOF_EXIT_CODE=0
    export MOCK_KILL_SUCCESS=true
    
    run ports::kill 8080
    assert_success
    assert_output --partial "Killed processes on port 8080"
}

@test "ports::kill does nothing when no processes on port" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT=""
    export MOCK_LSOF_EXIT_CODE=1
    
    run ports::kill 8080
    assert_success
    refute_output --partial "Killed processes"
}

@test "ports::kill fails when kill command fails" {
    export MOCK_KILL_SUCCESS="false"
    
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    
    run ports::kill 8080
    assert_failure
    assert_output --partial "Failed to kill processes on port 8080"
}

@test "ports::kill handles multiple PIDs on separate lines" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    # Test with multiple PIDs on separate lines  
    export MOCK_LSOF_OUTPUT="1234
5678
9012"
    export MOCK_LSOF_EXIT_CODE=0
    
    run ports::kill 8080
    assert_success
    assert_output --partial "Killed processes on port 8080"
}

# =============================================================================
# Tests for ports::check_and_free
# =============================================================================

@test "ports::check_and_free does nothing when port is free" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    
    export MOCK_LSOF_OUTPUT=""
    export MOCK_LSOF_EXIT_CODE=1
    
    run ports::check_and_free 8080
    assert_success
    refute_output --partial "Port 8080 is in use"
}

@test "ports::check_and_free kills processes with YES=yes" {
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    export MOCK_KILL_SUCCESS=true
    
    run ports::check_and_free 8080 "yes"
    assert_success
    assert_output --partial "Port 8080 is in use"
    assert_output --partial "Killed processes on port 8080"
}

@test "ports::check_and_free prompts and kills when user confirms" {
    export MOCK_CONFIRM_RESPONSE="yes"
    
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    export MOCK_KILL_SUCCESS=true
    
    run ports::check_and_free 8080 ""
    assert_success
    assert_output --partial "Port 8080 is in use"
    assert_output --partial "Killed processes on port 8080"
}

@test "ports::check_and_free exits when user declines" {
    export MOCK_CONFIRM_RESPONSE="no"
    
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    
    run ports::check_and_free 8080 ""
    assert_failure
    assert_output --partial "Port 8080 is in use"
    assert_output --partial "Please free port 8080 and retry"
}

@test "ports::check_and_free handles YES environment variable" {
    export YES="yes"
    
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    
    # Call without second parameter, should use YES env var
    run ports::check_and_free 8080
    assert_success
    assert_output --partial "Killed processes on port 8080"
}

@test "ports::check_and_free parameter overrides YES environment variable" {
    export YES="yes"
    export MOCK_CONFIRM_RESPONSE="no"
    
    source "$SCRIPT_PATH"
    mock_system_is_command
    mock_lsof
    mock_flow_functions
    
    export MOCK_LSOF_OUTPUT="1234"
    export MOCK_LSOF_EXIT_CODE=0
    
    # Pass "no" as second parameter to override YES=yes
    # This should NOT auto-kill, but instead prompt (and we decline)
    run ports::check_and_free 8080 "no"
    assert_failure
    assert_output --partial "Please free port 8080 and retry"
}