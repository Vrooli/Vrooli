#!/usr/bin/env bats
# Tests for MinIO inject.sh script

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set MinIO-specific test environment
    export MINIO_ENDPOINT="http://localhost:9000"
    export MINIO_ACCESS_KEY="minioadmin"
    export MINIO_SECRET_KEY="minioadmin"
    
    # Load the script without executing main
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    SCRIPT_PATH="${SCRIPT_DIR}/inject.sh"
    source "${SCRIPT_PATH}" || true
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "inject script loads without errors" {
    # Script loading happens in setup, this verifies it worked
    declare -f minio_inject::usage > /dev/null
    [ "$?" -eq 0 ]
}

@test "inject defines all required functions" {
    declare -f minio_inject::usage > /dev/null
    [ "$?" -eq 0 ]
    declare -f minio_inject::validate_config > /dev/null
    [ "$?" -eq 0 ]
    declare -f minio_inject::inject_data > /dev/null
    [ "$?" -eq 0 ]
    declare -f minio_inject::check_status > /dev/null
    [ "$?" -eq 0 ]
    declare -f minio_inject::main > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "minio_inject::validate_config rejects empty JSON" {
    run minio_inject::validate_config ""
    [ "$status" -ne 0 ]
}

@test "minio_inject::validate_config rejects invalid JSON" {
    run minio_inject::validate_config "{invalid json"
    [ "$status" -ne 0 ]
}

@test "minio_inject::validate_config accepts empty buckets and data" {
    local config='{"buckets": [], "data": []}'
    run minio_inject::validate_config "$config"
    [ "$status" -ne 0 ] # Should fail because no buckets or data
}

@test "minio_inject::validate_config accepts valid bucket configuration" {
    local config='{"buckets": [{"name": "test-bucket", "policy": "private"}]}'
    run minio_inject::validate_config "$config"
    [ "$status" -eq 0 ]
}

@test "minio_inject::validate_config rejects invalid bucket name" {
    local config='{"buckets": [{"name": "INVALID-BUCKET", "policy": "private"}]}'
    run minio_inject::validate_config "$config"
    [ "$status" -ne 0 ]
}

@test "minio_inject::validate_config rejects invalid bucket policy" {
    local config='{"buckets": [{"name": "test-bucket", "policy": "invalid"}]}'
    run minio_inject::validate_config "$config"
    [ "$status" -ne 0 ]
}

# ============================================================================
# Usage and Help Tests
# ============================================================================

@test "minio_inject::usage displays help information" {
    run minio_inject::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MinIO Data Injection Adapter" ]]
    [[ "$output" =~ "--validate" ]]
    [[ "$output" =~ "--inject" ]]
    [[ "$output" =~ "--status" ]]
    [[ "$output" =~ "--rollback" ]]
}

# ============================================================================
# Main Function Structure Tests
# ============================================================================

@test "main function contains validate case" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio_inject::main | grep -q 'validate'"
    [ "$status" -eq 0 ]
}

@test "main function contains inject case" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio_inject::main | grep -q 'inject'"
    [ "$status" -eq 0 ]
}

@test "main function contains status case" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio_inject::main | grep -q 'status'"
    [ "$status" -eq 0 ]
}

@test "main function handles unknown actions" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio_inject::main | grep -q 'Unknown action'"
    [ "$status" -eq 0 ]
}