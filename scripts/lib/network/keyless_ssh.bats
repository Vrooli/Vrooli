#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test helpers
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Load mocks
load "../../__test/fixtures/mocks/logs.sh"
load "../../__test/fixtures/mocks/commands.bash"
load "../../__test/fixtures/mocks/system.sh"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/keyless_ssh.sh"

# Source necessary dependencies that the script needs
source "${BATS_TEST_DIRNAME}/../utils/var.sh"
source "${var_LOG_FILE}"
source "${var_EXIT_CODES_FILE}"
source "${var_FLOW_FILE}"

setup() {
    # Initialize mocks
    mock::cleanup_logs "" || true
    mock::commands::reset || true
    mock::system::reset || true
    
    # Set up test environment
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH=""
    export HOME="/tmp/test_home"
    export TEST_NAMESPACE="test_$$"
    
    # Create temporary directories for testing
    mkdir -p "$HOME/.ssh"
    
    # Mock responses directory for tracking calls
    export MOCK_RESPONSES_DIR="/tmp/mock_responses_$$"
    mkdir -p "$MOCK_RESPONSES_DIR"
    : > "${MOCK_RESPONSES_DIR}/command_calls.log"
}

teardown() {
    # Clean up mocks
    mock::cleanup_logs "" || true
    mock::commands::reset || true
    mock::system::reset || true
    
    # Clean up temporary directories and files
    rm -rf "$HOME" || true
    rm -rf "$MOCK_RESPONSES_DIR" || true
    
    # Unset environment variables
    unset SITE_IP SSH_KEY_PATH HOME TEST_NAMESPACE MOCK_RESPONSES_DIR
}

# Tests for keyless_ssh::get_key_path function
@test "keyless_ssh::get_key_path returns SSH_KEY_PATH when set" {
    source "$SCRIPT_PATH"
    
    export SSH_KEY_PATH="/custom/path/id_rsa"
    
    run keyless_ssh::get_key_path
    assert_success
    assert_output "/custom/path/id_rsa"
}

@test "keyless_ssh::get_key_path returns default path based on SITE_IP" {
    source "$SCRIPT_PATH"
    
    export SSH_KEY_PATH=""
    export SITE_IP="192.168.1.100"
    
    run keyless_ssh::get_key_path
    assert_success
    assert_output "/tmp/test_home/.ssh/id_rsa_192.168.1.100"
}

@test "keyless_ssh::get_key_path fails when neither SSH_KEY_PATH nor SITE_IP is set" {
    source "$SCRIPT_PATH"
    
    export SSH_KEY_PATH=""
    unset SITE_IP
    
    run keyless_ssh::get_key_path
    assert_failure
    assert_output "No SSH key path or SITE_IP set"
}

# Tests for keyless_ssh::get_remote_server function
@test "keyless_ssh::get_remote_server returns root@SITE_IP when SITE_IP is set" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    
    run keyless_ssh::get_remote_server
    assert_success
    assert_output "root@192.168.1.100"
}

@test "keyless_ssh::get_remote_server fails when SITE_IP is not set" {
    source "$SCRIPT_PATH"
    
    unset SITE_IP
    
    run keyless_ssh::get_remote_server
    assert_failure
}

# Tests for keyless_ssh::check_key function
@test "keyless_ssh::check_key generates key when it doesn't exist" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Mock ssh-keygen to create the key files
    ssh-keygen() {
        echo "ssh-keygen $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        # Create mock key files
        touch "$4" "$4.pub"
        echo "mock_public_key" > "$4.pub"
        return 0
    }
    export -f ssh-keygen
    
    # Mock cat to simulate reading public key
    cat() {
        if [[ "$1" =~ \.pub$ ]]; then
            echo "mock_public_key"
        fi
    }
    export -f cat
    
    # Mock ssh to simulate successful key copy
    ssh() {
        echo "ssh $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        return 0
    }
    export -f ssh
    
    run keyless_ssh::check_key
    assert_success
    
    # Verify ssh-keygen was called
    assert grep -q "ssh-keygen" "${MOCK_RESPONSES_DIR}/command_calls.log"
    # Verify ssh was called for key copy
    assert grep -q "ssh.*root@192.168.1.100" "${MOCK_RESPONSES_DIR}/command_calls.log"
}

@test "keyless_ssh::check_key skips generation when key exists" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Create existing key file
    touch "$SSH_KEY_PATH"
    
    # Mock ssh-keygen (should not be called)
    ssh-keygen() {
        echo "ssh-keygen $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        return 0
    }
    export -f ssh-keygen
    
    run keyless_ssh::check_key
    assert_success
    
    # Verify ssh-keygen was NOT called
    refute grep -q "ssh-keygen" "${MOCK_RESPONSES_DIR}/command_calls.log"
}

@test "keyless_ssh::check_key removes key files when copy fails" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Mock ssh-keygen to create the key files
    ssh-keygen() {
        touch "$4" "$4.pub"
        echo "mock_public_key" > "$4.pub"
        return 0
    }
    export -f ssh-keygen
    
    # Mock cat to simulate reading public key
    cat() {
        if [[ "$1" =~ \.pub$ ]]; then
            echo "mock_public_key"
        fi
    }
    export -f cat
    
    # Mock ssh to simulate failed key copy
    ssh() {
        return 1
    }
    export -f ssh
    
    # Mock rm to track file removal
    rm() {
        echo "rm $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        command rm "$@"
    }
    export -f rm
    
    run keyless_ssh::check_key
    assert_failure
    
    # Verify rm was called to clean up key files
    assert grep -q "rm.*id_rsa_test" "${MOCK_RESPONSES_DIR}/command_calls.log"
}

# Tests for keyless_ssh::connect function
@test "keyless_ssh::connect succeeds with valid SSH connection" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Create mock key file
    touch "$SSH_KEY_PATH"
    
    # Mock ssh to simulate successful connection
    ssh() {
        echo "ssh $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        return 0
    }
    export -f ssh
    
    run keyless_ssh::connect
    assert_success
    
    # Verify ssh was called with correct parameters
    assert grep -q "ssh.*-i.*id_rsa_test.*root@192.168.1.100" "${MOCK_RESPONSES_DIR}/command_calls.log"
}

@test "keyless_ssh::connect retries after removing host key on first failure" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Create mock key file
    touch "$SSH_KEY_PATH"
    
    # Track SSH call count
    SSH_CALL_COUNT=0
    
    # Mock ssh to fail first time, succeed second time
    ssh() {
        echo "ssh $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        ((SSH_CALL_COUNT++))
        if [[ $SSH_CALL_COUNT -eq 1 ]]; then
            return 1  # First call fails
        else
            return 0  # Second call succeeds
        fi
    }
    export -f ssh
    export SSH_CALL_COUNT
    
    # Mock ssh-keygen for host key removal
    ssh-keygen() {
        echo "ssh-keygen $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        return 0
    }
    export -f ssh-keygen
    
    run keyless_ssh::connect
    assert_success
    
    # Verify ssh-keygen was called for host key removal
    assert grep -q "ssh-keygen.*-R.*192.168.1.100" "${MOCK_RESPONSES_DIR}/command_calls.log"
    # Verify ssh was called twice
    assert [ $(grep -c "ssh.*-i.*id_rsa_test" "${MOCK_RESPONSES_DIR}/command_calls.log") -eq 2 ]
}

@test "keyless_ssh::connect fails and removes key file after retry failure" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Create mock key file
    touch "$SSH_KEY_PATH"
    
    # Mock ssh to always fail
    ssh() {
        return 1
    }
    export -f ssh
    
    # Mock ssh-keygen for host key removal
    ssh-keygen() {
        return 0
    }
    export -f ssh-keygen
    
    # Mock keyless_ssh::remove_key_file
    keyless_ssh::remove_key_file() {
        echo "remove_key_file called" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        return 0
    }
    export -f keyless_ssh::remove_key_file
    
    run keyless_ssh::connect
    assert_failure
    
    # Verify remove_key_file was called
    assert grep -q "remove_key_file called" "${MOCK_RESPONSES_DIR}/command_calls.log"
}

# Tests for keyless_ssh::remove_key_file function
@test "keyless_ssh::remove_key_file removes recent key files" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Create mock key files
    touch "$SSH_KEY_PATH" "$SSH_KEY_PATH.pub"
    
    # Mock find to return the key file (simulate recent file)
    find() {
        echo "find $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        if [[ "$*" =~ -mmin ]]; then
            echo "$SSH_KEY_PATH"  # Return the key file as if it's recent
        fi
    }
    export -f find
    
    # Mock wc to return count > 0
    wc() {
        if [[ "$*" =~ -l ]]; then
            echo "1"
        fi
    }
    export -f wc
    
    # Mock rm to track file removal
    rm() {
        echo "rm $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        command rm "$@" 2>/dev/null || true
    }
    export -f rm
    
    run keyless_ssh::remove_key_file
    assert_success
    
    # Verify find was called with mmin parameter
    assert grep -q "find.*-mmin" "${MOCK_RESPONSES_DIR}/command_calls.log"
    # Verify rm was called with key files
    assert grep -q "rm.*id_rsa_test" "${MOCK_RESPONSES_DIR}/command_calls.log"
}

@test "keyless_ssh::remove_key_file skips removal when key is not recent" {
    source "$SCRIPT_PATH"
    
    export SITE_IP="192.168.1.100"
    export SSH_KEY_PATH="/tmp/test_home/.ssh/id_rsa_test"
    
    # Mock find to return empty (simulate old file)
    find() {
        echo "find $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        # Return nothing for recent files check
        return 0
    }
    export -f find
    
    # Mock wc to return count 0
    wc() {
        if [[ "$*" =~ -l ]]; then
            echo "0"
        fi
    }
    export -f wc
    
    # Mock rm to track file removal (should not be called)
    rm() {
        echo "rm $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
        return 0
    }
    export -f rm
    
    run keyless_ssh::remove_key_file
    assert_success
    
    # Verify rm was NOT called
    refute grep -q "rm.*id_rsa_test" "${MOCK_RESPONSES_DIR}/command_calls.log"
}