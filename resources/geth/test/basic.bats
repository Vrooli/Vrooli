#!/usr/bin/env bats
# Basic Geth Resource Tests

setup() {
    # Set test environment
    export GETH_NETWORK="dev"
    export GETH_CONTAINER_NAME="vrooli-geth-test"
    export GETH_PORT="18545"
    export GETH_WS_PORT="18546"
    export GETH_P2P_PORT="30313"
    export GETH_DATA_DIR="/tmp/geth-test-$$"
    
    # Source the CLI
    source "${BATS_TEST_DIRNAME}/../cli.sh"
}

teardown() {
    # Clean up test container
    docker stop "${GETH_CONTAINER_NAME}" >/dev/null 2>&1 || true
    docker rm -f "${GETH_CONTAINER_NAME}" >/dev/null 2>&1 || true
    
    # Clean up test data
    rm -rf "${GETH_DATA_DIR}"
}

@test "Geth CLI exists and is executable" {
    [ -f "${BATS_TEST_DIRNAME}/../cli.sh" ]
    [ -x "${BATS_TEST_DIRNAME}/../cli.sh" ]
}

@test "Geth can show help" {
    run bash "${BATS_TEST_DIRNAME}/../cli.sh" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Geth Resource CLI" ]]
}

@test "Geth status works when not installed" {
    run bash "${BATS_TEST_DIRNAME}/../cli.sh" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "not installed" ]] || [[ "$output" =~ "not running" ]]
}

@test "Geth can check if installed" {
    run geth::is_installed
    # Should return non-zero when not installed
    [ "$status" -ne 0 ]
}

@test "Geth directories are created correctly" {
    geth::init_directories
    [ -d "${GETH_DATA_DIR}/data" ]
    [ -d "${GETH_DATA_DIR}/contracts" ]
    [ -d "${GETH_DATA_DIR}/scripts" ]
    [ -d "${GETH_DATA_DIR}/logs" ]
}