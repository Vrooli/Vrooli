#!/usr/bin/env bats
# Tests for MinIO manage.sh script

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set MinIO-specific test environment
    export MINIO_CUSTOM_PORT="9999"
    export MINIO_CONTAINER_NAME="minio-test"
    export BUCKET=""
    export POLICY=""
    export MONITOR_INTERVAL="5"
    export FORCE="no"
    export YES="no"
    
    # Load the script without executing main
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/manage.sh" || true
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "minio script loads without errors" {
    # Script loading happens in setup, this verifies it worked
    declare -f minio::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "minio defines all required functions" {
    declare -f minio::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
    declare -f main > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "minio::parse_arguments sets default action to status" {
    minio::parse_arguments
    [ "$ACTION" = "status" ]
}

@test "minio::parse_arguments accepts install action" {
    minio::parse_arguments --action install
    [ "$ACTION" = "install" ]
}

@test "minio::parse_arguments accepts uninstall action" {
    minio::parse_arguments --action uninstall
    [ "$ACTION" = "uninstall" ]
}

@test "minio::parse_arguments accepts bucket operations" {
    minio::parse_arguments --action create-bucket --bucket test-bucket --policy download
    [ "$ACTION" = "create-bucket" ]
    [ "$BUCKET" = "test-bucket" ]
    [ "$POLICY" = "download" ]
}

@test "minio::parse_arguments handles monitor parameters" {
    minio::parse_arguments --action monitor --interval 10
    [ "$ACTION" = "monitor" ]
    [ "$MONITOR_INTERVAL" = "10" ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "minio::parse_arguments function is defined" {
    declare -f minio::parse_arguments > /dev/null
    [ "$?" -eq 0 ]
}

@test "minio::usage function is defined" {
    declare -f minio::usage > /dev/null
    [ "$?" -eq 0 ]
}

@test "main function is defined" {
    declare -f main > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "manage.sh --help displays usage information" {
    run "$SCRIPT_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MinIO S3-compatible object storage" ]]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "--action install" ]]
}

@test "manage.sh -h displays usage information" {
    run "$SCRIPT_PATH" -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MinIO S3-compatible object storage" ]]
}

# ============================================================================
# Main Function Structure Tests
# ============================================================================

@test "main function contains install case" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main | grep -q 'install)'"
    [ "$status" -eq 0 ]
}

@test "main function contains uninstall case" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main | grep -q 'uninstall)'"
    [ "$status" -eq 0 ]
}

@test "main function contains status case" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main | grep -q 'status)'"
    [ "$status" -eq 0 ]
}

@test "main function contains bucket operations" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main | grep -q 'create-bucket'"
    [ "$status" -eq 0 ]
}

@test "main function handles unknown actions" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main | grep -q 'Unknown action'"
    [ "$status" -eq 0 ]
}