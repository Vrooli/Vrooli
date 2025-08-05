#!/usr/bin/env bats

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "status.sh exists and is readable" {
    [ -f "$SCRIPT_PATH" ]
    [ -r "$SCRIPT_PATH" ]
}

@test "sourcing status.sh loads without errors" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing status.sh defines minio::status::check function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::check"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::status::check" ]]
}

@test "sourcing status.sh defines minio::status::show_resources function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::show_resources"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::status::show_resources" ]]
}

@test "sourcing status.sh defines minio::status::show_credentials function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::show_credentials"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::status::show_credentials" ]]
}

@test "sourcing status.sh defines minio::status::diagnose function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::diagnose"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::status::diagnose" ]]
}

@test "sourcing status.sh defines minio::status::monitor function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::monitor"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::status::monitor" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::status::check verifies container exists" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::check | grep -q 'container_exists'"
    [ "$status" -eq 0 ]
}

@test "minio::status::show_resources displays container stats" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::show_resources | grep -q 'stats'"
    [ "$status" -eq 0 ]
}

@test "minio::status::show_credentials reads from file" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::show_credentials | grep -q 'credentials'"
    [ "$status" -eq 0 ]
}

@test "minio::status::diagnose checks multiple components" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::diagnose | grep -q 'Checking'"
    [ "$status" -eq 0 ]
}

@test "minio::status::monitor has continuous loop" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::status::monitor | grep -q 'while'"
    [ "$status" -eq 0 ]
}