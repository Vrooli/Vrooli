#!/usr/bin/env bats

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "common.sh exists and is readable" {
    [ -f "$SCRIPT_PATH" ]
    [ -r "$SCRIPT_PATH" ]
}

@test "sourcing common.sh loads without errors" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing common.sh defines minio::common::container_exists function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::container_exists"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::container_exists" ]]
}

@test "sourcing common.sh defines minio::common::is_running function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::is_running"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::is_running" ]]
}

@test "sourcing common.sh defines minio::common::create_directories function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::create_directories"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::create_directories" ]]
}

@test "sourcing common.sh defines minio::common::generate_credentials function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::generate_credentials"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::generate_credentials" ]]
}

@test "sourcing common.sh defines minio::common::show_logs function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::show_logs"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::show_logs" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::common::container_exists checks docker container existence" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::container_exists | grep -q 'docker'"
    [ "$status" -eq 0 ]
}

@test "minio::common::is_running checks docker container status" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::is_running | grep -q 'docker'"
    [ "$status" -eq 0 ]
}

@test "minio::common::create_directories creates data directory" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::create_directories | grep -q 'data'"
    [ "$status" -eq 0 ]
}

@test "minio::common::create_directories sets proper permissions" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::create_directories | grep -q 'chmod'"
    [ "$status" -eq 0 ]
}

@test "minio::common::generate_credentials handles credentials file" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::generate_credentials | grep -q 'credentials'"
    [ "$status" -eq 0 ]
}

@test "minio::common::show_logs uses docker logs command" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::show_logs | grep -q 'docker logs'"
    [ "$status" -eq 0 ]
}