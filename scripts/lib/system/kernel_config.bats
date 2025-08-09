#!/usr/bin/env bats
# Tests for kernel_config.sh - Kernel parameter configuration

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

# Load mocks
load "${BATS_TEST_DIRNAME}/../../__test/fixtures/mocks/system"

setup() {
    vrooli_setup_unit_test
    
    # Source the kernel config utilities
    source "${BATS_TEST_DIRNAME}/kernel_config.sh"
    
    # Reset mocks
    mock::system::reset
    
    # Mock sysctl command behavior
    mock::system::set_command_available "sysctl" true
}

teardown() {
    vrooli_cleanup_test
}

@test "kernel_config::check_parameter checks kernel parameters" {
    # Mock sysctl to return expected value
    mock::system::set_sysctl_value "kernel.apparmor_restrict_unprivileged_userns" "0"
    
    run kernel_config::check_parameter "kernel.apparmor_restrict_unprivileged_userns" "0"
    assert_success
    
    # Test mismatch
    run kernel_config::check_parameter "kernel.apparmor_restrict_unprivileged_userns" "1"
    assert_failure
}

@test "kernel_config::configure_judge0 configures judge0 parameters" {
    # Mock that parameter is not already set correctly
    mock::system::set_sysctl_value "kernel.apparmor_restrict_unprivileged_userns" "1"
    mock::system::set_command_available "sudo" true
    
    run kernel_config::configure_judge0
    assert_success
    assert_output --partial "Configuring kernel parameters for Judge0"
}

@test "kernel_config::configure_judge0 skips when already configured" {
    # Mock that parameter is already set correctly
    mock::system::set_sysctl_value "kernel.apparmor_restrict_unprivileged_userns" "0"
    
    run kernel_config::configure_judge0
    assert_success
    assert_output --partial "already configured"
}

@test "kernel_config::configure_for_resources processes enabled resources" {
    # Create mock service.json
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "judge0": {
    "enabled": true
  }
}
EOF
    
    # Override var_SERVICE_JSON_FILE for test
    export var_SERVICE_JSON_FILE="$service_json"
    
    run kernel_config::configure_for_resources
    assert_success
    assert_output --partial "Configuring kernel parameters"
}

@test "kernel_config::configure_for_resources handles no enabled resources" {
    # Create mock service.json with no enabled judge0
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "judge0": {
    "enabled": false
  }
}
EOF
    
    # Override var_SERVICE_JSON_FILE for test
    export var_SERVICE_JSON_FILE="$service_json"
    
    run kernel_config::configure_for_resources
    assert_success
    assert_output --partial "No kernel parameter changes required"
}

@test "kernel_config::make_persistent creates persistent configuration" {
    # Mock successful file operations
    mock::system::set_command_available "sudo" true
    
    run kernel_config::make_persistent "test.param" "1" "Test parameter"
    assert_success
}

@test "kernel_config::cleanup removes configuration files" {
    # Mock that file exists and can be removed
    mock::system::set_command_available "sudo" true
    
    run kernel_config::cleanup
    assert_success
    assert_output --partial "Cleaning up Vrooli kernel configurations"
}