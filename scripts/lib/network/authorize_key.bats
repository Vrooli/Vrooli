#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test helpers
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Load mocks
load "../../__test/fixtures/mocks/logs.sh"
load "../../__test/fixtures/mocks/filesystem.sh"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/authorize_key.sh"

# Source necessary dependencies that the script needs
# Note: These need to be sourced here so functions are available
source "${BATS_TEST_DIRNAME}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
source "${var_LOG_FILE}"

setup() {
    # Initialize mocks
    mock::filesystem::reset
    mock::cleanup_logs "" || true
    
    # Set up test environment
    export TEST_HOME="$(mktemp -d)"
    export HOME="$TEST_HOME"
    
    # Configure filesystem mocking
    export MOCK_FILESYSTEM_MODE="mock"
}

teardown() {
    # Clean up mocks
    mock::filesystem::reset || true
    mock::cleanup_logs "" || true
    
    # Clean up test environment
    if [[ -n "${TEST_HOME:-}" ]] && [[ -d "$TEST_HOME" ]]; then
        trash::safe_remove "$TEST_HOME" --test-cleanup
    fi
    unset TEST_HOME HOME
}

# Tests for authorize_key::main function
@test "authorize_key::main creates .ssh directory with correct permissions" {
    source "$SCRIPT_PATH"
    
    # Mock cat to simulate user input
    cat() {
        echo "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"
        return 0
    }
    export -f cat
    
    # Mock mkdir and chmod to track calls
    mkdir_calls=()
    chmod_calls=()
    
    mkdir() {
        mkdir_calls+=("$*")
        return 0
    }
    export -f mkdir
    
    chmod() {
        chmod_calls+=("$*")
        return 0
    }
    export -f chmod
    
    run authorize_key::main
    assert_success
    
    # Verify .ssh directory was created
    [[ "${mkdir_calls[*]}" =~ "-p ~/.ssh" ]]
    
    # Verify permissions were set correctly
    [[ "${chmod_calls[*]}" =~ "700 ~/.ssh" ]]
    [[ "${chmod_calls[*]}" =~ "600 ~/.ssh/authorized_keys" ]]
}

@test "authorize_key::main appends public key to authorized_keys" {
    source "$SCRIPT_PATH"
    
    local test_key="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"
    
    # Mock cat to simulate user input
    cat() {
        echo "$test_key"
        return 0
    }
    export -f cat
    
    # Mock mkdir and chmod to succeed
    mkdir() { return 0; }
    chmod() { return 0; }
    export -f mkdir chmod
    
    run authorize_key::main
    assert_success
    assert_output --partial "Public key successfully added"
}

@test "authorize_key::main shows success message" {
    source "$SCRIPT_PATH"
    
    # Mock all filesystem operations
    cat() { echo "test-key"; return 0; }
    mkdir() { return 0; }
    chmod() { return 0; }
    export -f cat mkdir chmod
    
    run authorize_key::main
    assert_success
    assert_output --partial "✅ Public key successfully added to ~/.ssh/authorized_keys"
}

# Test help functionality
@test "script shows help with -h flag" {
    run bash "$SCRIPT_PATH" -h
    assert_success
    assert_output --partial "Append SSH public key to authorized_keys"
    assert_output --partial "Usage:"
}

@test "script shows help with --help flag" {
    run bash "$SCRIPT_PATH" --help
    assert_success
    assert_output --partial "Append SSH public key to authorized_keys"
    assert_output --partial "Usage:"
}

@test "script runs main function when executed directly" {
    # Mock all required functions
    cat() { echo "test-key"; return 0; }
    mkdir() { return 0; }
    chmod() { return 0; }
    export -f cat mkdir chmod
    
    run bash "$SCRIPT_PATH"
    assert_success
    assert_output --partial "Authorize SSH Public Key"
}