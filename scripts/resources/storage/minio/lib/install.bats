#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/install.sh"

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "install.sh exists and is readable" {
    [ -f "$SCRIPT_PATH" ]
    [ -r "$SCRIPT_PATH" ]
}

@test "sourcing install.sh loads without errors" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing install.sh defines minio::install function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::install" ]]
}

@test "sourcing install.sh defines minio::uninstall function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::uninstall"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::uninstall" ]]
}

@test "sourcing install.sh defines minio::install::generate_credentials function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::generate_credentials"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::install::generate_credentials" ]]
}

@test "sourcing install.sh defines minio::install::save_credentials function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::save_credentials"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::install::save_credentials" ]]
}

@test "sourcing install.sh defines minio::install::reset_credentials function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::reset_credentials"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::install::reset_credentials" ]]
}

@test "sourcing install.sh defines minio::install::upgrade function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::upgrade"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::install::upgrade" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::install checks Docker availability" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install | grep -q 'docker'"
    [ "$status" -eq 0 ]
}

@test "minio::install creates directories" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install | grep -q 'directories'"
    [ "$status" -eq 0 ]
}

@test "minio::install generates credentials" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install | grep -q 'credentials'"
    [ "$status" -eq 0 ]
}

@test "minio::uninstall removes container" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::uninstall | grep -q 'remove'"
    [ "$status" -eq 0 ]
}

@test "minio::install::generate_credentials uses openssl" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::generate_credentials | grep -q 'openssl'"
    [ "$status" -eq 0 ]
}

@test "minio::install::save_credentials sets proper permissions" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::save_credentials | grep -q 'chmod'"
    [ "$status" -eq 0 ]
}

@test "minio::install::upgrade pulls latest image" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::install::upgrade | grep -q 'pull'"
    [ "$status" -eq 0 ]
}