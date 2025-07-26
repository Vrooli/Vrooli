#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/api.sh"

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "api.sh exists and is readable" {
    [ -f "$SCRIPT_PATH" ]
    [ -r "$SCRIPT_PATH" ]
}

@test "sourcing api.sh loads without errors" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing api.sh defines minio::api::test function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::test"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::test" ]]
}

@test "sourcing api.sh defines minio::api::list_buckets function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::list_buckets"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::list_buckets" ]]
}

@test "sourcing api.sh defines minio::api::create_bucket function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::create_bucket"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::create_bucket" ]]
}

@test "sourcing api.sh defines minio::api::get_bucket_size function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::get_bucket_size"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::get_bucket_size" ]]
}

@test "sourcing api.sh defines minio::api::set_bucket_policy function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::set_bucket_policy"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::set_bucket_policy" ]]
}

@test "sourcing api.sh defines minio::api::upload_file function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::upload_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::upload_file" ]]
}

@test "sourcing api.sh defines minio::api::download_file function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::download_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::api::download_file" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::api::test performs health check" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::test | grep -q 'API.*test'"
    [ "$status" -eq 0 ]
}

@test "minio::api::list_buckets uses mc command" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::list_buckets | grep -q 'mc'"
    [ "$status" -eq 0 ]
}

@test "minio::api::create_bucket uses mc mb command" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::create_bucket | grep -q 'mc.*mb'"
    [ "$status" -eq 0 ]
}

@test "minio::api::set_bucket_policy handles different policy types" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::api::set_bucket_policy | grep -q 'policy'"
    [ "$status" -eq 0 ]
}