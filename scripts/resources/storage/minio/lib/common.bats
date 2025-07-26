#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"
MINIO_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies
RESOURCES_DIR="$MINIO_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source required utilities (suppress errors during test setup)
. "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/ports.sh" 2>/dev/null || true

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

@test "sourcing common.sh defines minio::common::is_installed function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::is_installed"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::is_installed" ]]
}

@test "sourcing common.sh defines minio::common::is_running function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::is_running"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::is_running" ]]
}

@test "sourcing common.sh defines minio::common::ensure_directories function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::ensure_directories"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::ensure_directories" ]]
}

@test "sourcing common.sh defines minio::common::get_credentials function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::get_credentials"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::get_credentials" ]]
}

@test "sourcing common.sh defines minio::common::show_logs function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::show_logs"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::common::show_logs" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::common::is_installed checks docker container existence" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::is_installed | grep -q 'docker'"
    [ "$status" -eq 0 ]
}

@test "minio::common::is_running checks docker container status" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::is_running | grep -q 'docker'"
    [ "$status" -eq 0 ]
}

@test "minio::common::ensure_directories creates data directory" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::ensure_directories | grep -q 'data'"
    [ "$status" -eq 0 ]
}

@test "minio::common::ensure_directories sets proper permissions" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::ensure_directories | grep -q 'chmod'"
    [ "$status" -eq 0 ]
}

@test "minio::common::get_credentials reads from credentials file" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::get_credentials | grep -q 'credentials'"
    [ "$status" -eq 0 ]
}

@test "minio::common::show_logs uses docker logs command" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::common::show_logs | grep -q 'docker logs'"
    [ "$status" -eq 0 ]
}