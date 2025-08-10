#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test helpers
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Load mocks
load "../../__test/fixtures/mocks/logs.sh"
load "../../__test/fixtures/mocks/system.sh"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/enableSsh.sh"

# Source necessary dependencies that the script needs
# Note: These need to be sourced here so functions are available
source "${BATS_TEST_DIRNAME}/../utils/var.sh"
source "${var_LOG_FILE}"
source "${var_EXIT_CODES_FILE}"
source "${var_FLOW_FILE}"

setup() {
    # Initialize mocks
    mock::system::reset
    mock::cleanup_logs "" || true
    
    # Set up test environment
    export TEST_HOME="$(mktemp -d)"
    export HOME="$TEST_HOME"
    export SUDO_MODE="auto"  # Allow sudo by default for tests
    
    # Create .ssh directory for tests
    mkdir -p "$TEST_HOME/.ssh"
    
    # Configure system mocks
    export MOCK_SUDO_AVAILABLE="true"
    export MOCK_SUDO_PASSWORDLESS="true"
    export MOCK_SYSTEMCTL_SUCCESS="true"
    
    # Mock functions need to be defined before sourcing
    setup_mocks
}

teardown() {
    # Clean up mocks
    mock::system::reset || true
    mock::cleanup_logs "" || true
    
    # Clean up test environment
    if [[ -n "${TEST_HOME:-}" ]] && [[ -d "$TEST_HOME" ]]; then
        trash::safe_remove "$TEST_HOME" --test-cleanup
    fi
    unset TEST_HOME HOME SUDO_MODE
    unset MOCK_SUDO_AVAILABLE MOCK_SUDO_PASSWORDLESS MOCK_SYSTEMCTL_SUCCESS
}

# Define all mocks in a function that can be called from setup
setup_mocks() {
    # Mock sed command
    sed() {
        # Track sed calls for verification
        if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
            echo "sed $*" >> "${MOCK_RESPONSES_DIR}/sed_calls.log"
        fi
        
        # Check for version flag
        if [[ "$1" == "--version" ]]; then
            echo "sed (GNU sed) 4.7"
            return 0
        fi
        
        # Handle inline editing
        if [[ "$1" == "-i" ]] || [[ "$1" == "-i ''" ]]; then
            # Just succeed for test purposes
            return 0
        fi
        
        return 0
    }
    export -f sed
    
    # Mock sudo command
    sudo() {
        # Handle sudo -n true test (for passwordless check)
        if [[ "$1" == "-n" ]] && [[ "$2" == "true" ]]; then
            [[ "${MOCK_SUDO_PASSWORDLESS:-true}" == "true" ]]
            return $?
        fi
        
        # Pass through to the actual command being run (remove 'sudo' prefix)
        "$@"
    }
    export -f sudo
    
    # Mock systemctl command
    systemctl() {
        # Track systemctl calls
        if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
            echo "systemctl $*" >> "${MOCK_RESPONSES_DIR}/systemctl_calls.log"
        fi
        
        case "$1" in
            "restart")
                if [[ "${MOCK_SYSTEMCTL_SUCCESS:-true}" == "true" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f systemctl
    
    # Mock mkdir and touch commands
    mkdir() {
        if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
            echo "mkdir $*" >> "${MOCK_RESPONSES_DIR}/mkdir_calls.log"
        fi
        return 0
    }
    export -f mkdir
    
    touch() {
        if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
            echo "touch $*" >> "${MOCK_RESPONSES_DIR}/touch_calls.log"
        fi
        return 0
    }
    export -f touch
    
    # Mock chmod command
    chmod() {
        if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
            echo "chmod $*" >> "${MOCK_RESPONSES_DIR}/chmod_calls.log"
        fi
        return 0
    }
    export -f chmod
}

# Tests for enableSsh::ensure_ssh_files function
@test "enableSsh::ensure_ssh_files creates .ssh directory and authorized_keys file" {
    source "$SCRIPT_PATH"
    
    # Set up mock response directory for tracking
    export MOCK_RESPONSES_DIR="$(mktemp -d)"
    
    run enableSsh::ensure_ssh_files
    assert_success
    
    # Verify mkdir was called for .ssh
    [[ -f "${MOCK_RESPONSES_DIR}/mkdir_calls.log" ]]
    grep -q "mkdir -p ~/.ssh" "${MOCK_RESPONSES_DIR}/mkdir_calls.log"
    
    # Verify touch was called for authorized_keys
    [[ -f "${MOCK_RESPONSES_DIR}/touch_calls.log" ]]
    grep -q "touch ~/.ssh/authorized_keys" "${MOCK_RESPONSES_DIR}/touch_calls.log"
    
    # Verify correct permissions were set
    [[ -f "${MOCK_RESPONSES_DIR}/chmod_calls.log" ]]
    grep -q "chmod 700 ~/.ssh" "${MOCK_RESPONSES_DIR}/chmod_calls.log"
    grep -q "chmod 600 ~/.ssh/authorized_keys" "${MOCK_RESPONSES_DIR}/chmod_calls.log"
    
    # Clean up
    trash::safe_remove "${MOCK_RESPONSES_DIR}" --test-cleanup
}

# Tests for enableSsh::enable_password_authentication function
@test "enableSsh::enable_password_authentication modifies sshd_config when sudo available" {
    export SUDO_MODE="auto"
    source "$SCRIPT_PATH"
    
    # Set up mock response directory for tracking
    export MOCK_RESPONSES_DIR="$(mktemp -d)"
    
    run enableSsh::enable_password_authentication
    assert_success
    
    # Verify sed was called to modify sshd_config
    [[ -f "${MOCK_RESPONSES_DIR}/sed_calls.log" ]]
    grep -q "PasswordAuthentication yes" "${MOCK_RESPONSES_DIR}/sed_calls.log"
    grep -q "PubkeyAuthentication yes" "${MOCK_RESPONSES_DIR}/sed_calls.log"
    grep -q "AuthorizedKeysFile" "${MOCK_RESPONSES_DIR}/sed_calls.log"
    
    # Clean up
    trash::safe_remove "${MOCK_RESPONSES_DIR}" --test-cleanup
}

@test "enableSsh::enable_password_authentication skips when sudo mode is skip" {
    export SUDO_MODE="skip"
    source "$SCRIPT_PATH"
    
    run enableSsh::enable_password_authentication
    assert_success
    assert_output --partial "Skipping PasswordAuthentication setup due to sudo mode"
}

# Tests for enableSsh::restart_ssh function
@test "enableSsh::restart_ssh attempts to restart sshd service first" {
    export SUDO_MODE="auto"
    source "$SCRIPT_PATH"
    
    # Set up mock response directory for tracking
    export MOCK_RESPONSES_DIR="$(mktemp -d)"
    
    run enableSsh::restart_ssh
    assert_success
    
    # Verify systemctl restart sshd was called
    [[ -f "${MOCK_RESPONSES_DIR}/systemctl_calls.log" ]]
    grep -q "systemctl restart sshd" "${MOCK_RESPONSES_DIR}/systemctl_calls.log"
    
    # Clean up
    trash::safe_remove "${MOCK_RESPONSES_DIR}" --test-cleanup
}

@test "enableSsh::restart_ssh falls back to ssh service when sshd fails" {
    export SUDO_MODE="auto"
    export MOCK_SYSTEMCTL_SUCCESS="false"
    source "$SCRIPT_PATH"
    setup_mocks  # Re-setup mocks with new configuration
    
    # Custom systemctl mock that fails for sshd but succeeds for ssh
    systemctl() {
        case "$*" in
            "restart sshd")
                return 1  # Fail for sshd
                ;;
            "restart ssh")
                return 0  # Succeed for ssh
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f systemctl
    
    run enableSsh::restart_ssh
    assert_success
}

@test "enableSsh::restart_ssh exits with error when both services fail" {
    export SUDO_MODE="auto"
    source "$SCRIPT_PATH"
    
    # Custom systemctl mock that always fails
    systemctl() {
        return 1
    }
    export -f systemctl
    
    run enableSsh::restart_ssh
    assert_failure
    assert_output --partial "Failed to restart ssh"
}

@test "enableSsh::restart_ssh skips when sudo mode is skip" {
    export SUDO_MODE="skip"
    source "$SCRIPT_PATH"
    
    run enableSsh::restart_ssh
    assert_success
    assert_output --partial "Skipping SSH restart due to sudo mode"
}

# Tests for enableSsh::setup_ssh function
@test "enableSsh::setup_ssh runs all SSH setup steps" {
    export SUDO_MODE="auto"
    source "$SCRIPT_PATH"
    
    run enableSsh::setup_ssh
    assert_success
    assert_output --partial "Setting up SSH"
}

# Integration test
@test "script runs setup_ssh when executed directly" {
    export SUDO_MODE="auto"
    
    run bash "$SCRIPT_PATH"
    assert_success
    assert_output --partial "Setting up SSH"
}

# Test help functionality (this script doesn't have help, but we test direct execution)
@test "script executes without arguments" {
    export SUDO_MODE="auto"
    
    run bash "$SCRIPT_PATH"
    assert_success
}

# Test sed version detection
@test "enableSsh::enable_password_authentication handles different sed versions" {
    export SUDO_MODE="auto"
    source "$SCRIPT_PATH"
    
    # Mock sed to not support --version (like BSD sed)
    sed() {
        if [[ "$1" == "--version" ]]; then
            return 1  # BSD sed doesn't support --version
        fi
        return 0
    }
    export -f sed
    
    run enableSsh::enable_password_authentication
    assert_success
}