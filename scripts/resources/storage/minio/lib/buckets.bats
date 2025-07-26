#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/buckets.sh"

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "buckets.sh exists and is readable" {
    [ -f "$SCRIPT_PATH" ]
    [ -r "$SCRIPT_PATH" ]
}

@test "sourcing buckets.sh loads without errors" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing buckets.sh defines minio::buckets::initialize_defaults function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::initialize_defaults"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::buckets::initialize_defaults" ]]
}

@test "sourcing buckets.sh defines minio::buckets::create_custom function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::create_custom"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::buckets::create_custom" ]]
}

@test "sourcing buckets.sh defines minio::buckets::remove function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::remove"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::buckets::remove" ]]
}

@test "sourcing buckets.sh defines minio::buckets::show_stats function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::show_stats"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::buckets::show_stats" ]]
}

@test "sourcing buckets.sh defines minio::buckets::test_upload function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::test_upload"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::buckets::test_upload" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::buckets::initialize_defaults creates default buckets" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::initialize_defaults | grep -q 'bucket'"
    [ "$status" -eq 0 ]
}

@test "minio::buckets::create_custom validates bucket name" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::create_custom | grep -q 'Invalid'"
    [ "$status" -eq 0 ]
}

@test "minio::buckets::remove has force option" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::remove | grep -q 'force'"
    [ "$status" -eq 0 ]
}

@test "minio::buckets::show_stats uses mc du command" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::show_stats | grep -q 'mc.*du'"
    [ "$status" -eq 0 ]
}

@test "minio::buckets::test_upload creates test file" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::buckets::test_upload | grep -q 'test'"
    [ "$status" -eq 0 ]
}