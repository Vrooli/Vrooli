#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/manage.sh"
MINIO_DIR="$BATS_TEST_DIRNAME"

# Source dependencies
RESOURCES_DIR="$MINIO_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source required utilities (suppress errors during test setup)
. "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
. "$HELPERS_DIR/utils/args.sh" 2>/dev/null || true

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "manage.sh exists and is executable" {
    [ -f "$SCRIPT_PATH" ]
    [ -x "$SCRIPT_PATH" ]
}

@test "sourcing manage.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main && declare -f minio::parse_arguments"
    [ "$status" -eq 0 ]
    [[ "$output" =~ main ]]
    [[ "$output" =~ minio::parse_arguments ]]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "minio::parse_arguments sets default action to status" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null; minio::parse_arguments; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

@test "minio::parse_arguments accepts install action" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null; minio::parse_arguments --action install; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "minio::parse_arguments accepts uninstall action" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null; minio::parse_arguments --action uninstall; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uninstall" ]]
}

@test "minio::parse_arguments accepts bucket operations" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null; minio::parse_arguments --action create-bucket --bucket test-bucket --policy download; echo \"ACTION=\$ACTION BUCKET=\$BUCKET POLICY=\$POLICY\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ACTION=create-bucket" ]]
    [[ "$output" =~ "BUCKET=test-bucket" ]]
    [[ "$output" =~ "POLICY=download" ]]
}

@test "minio::parse_arguments handles monitor parameters" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null; minio::parse_arguments --action monitor --interval 10; echo \"ACTION=\$ACTION INTERVAL=\$MONITOR_INTERVAL\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ACTION=monitor" ]]
    [[ "$output" =~ "INTERVAL=10" ]]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing manage.sh defines minio::parse_arguments function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::parse_arguments"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::parse_arguments" ]]
}

@test "sourcing manage.sh defines minio::usage function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::usage"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::usage" ]]
}

@test "sourcing manage.sh defines main function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f main"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "main" ]]
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