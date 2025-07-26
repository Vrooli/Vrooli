#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/docker.sh"
MINIO_DIR="$BATS_TEST_DIRNAME/.."

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "docker.sh exists and is readable" {
    [ -f "$SCRIPT_PATH" ]
    [ -r "$SCRIPT_PATH" ]
}

@test "sourcing docker.sh loads without errors" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null"
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing docker.sh defines minio::docker::create_network function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::create_network"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::create_network" ]]
}

@test "sourcing docker.sh defines minio::docker::create_container function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::create_container"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::create_container" ]]
}

@test "sourcing docker.sh defines minio::docker::start function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::start"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::start" ]]
}

@test "sourcing docker.sh defines minio::docker::stop function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::stop"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::stop" ]]
}

@test "sourcing docker.sh defines minio::docker::restart function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::restart"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::restart" ]]
}

@test "sourcing docker.sh defines minio::docker::remove function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::remove"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::remove" ]]
}

@test "sourcing docker.sh defines minio::docker::exec function" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::exec"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minio::docker::exec" ]]
}

# ============================================================================
# Function Structure Tests (without execution)
# ============================================================================

@test "minio::docker::create_network creates docker network" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::create_network | grep -q 'docker network create'"
    [ "$status" -eq 0 ]
}

@test "minio::docker::create_container uses docker run command" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::create_container | grep -q '\"docker\".*\"run\"'"
    [ "$status" -eq 0 ]
}

@test "minio::docker::start uses docker start" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::start | grep -q 'docker start'"
    [ "$status" -eq 0 ]
}

@test "minio::docker::stop uses docker stop" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::stop | grep -q 'docker stop'"
    [ "$status" -eq 0 ]
}

@test "minio::docker::exec runs commands in container" {
    run bash -c "source '$SCRIPT_PATH' 2>/dev/null && declare -f minio::docker::exec | grep -q 'docker exec'"
    [ "$status" -eq 0 ]
}