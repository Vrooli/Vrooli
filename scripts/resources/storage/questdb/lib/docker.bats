#!/usr/bin/env bats

# QuestDB Docker Functions Tests

setup() {
    load '../../../../testing/test_helper'
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    source "${BATS_TEST_DIRNAME}/docker.sh"
    
    # Use test container name to avoid conflicts
    export QUESTDB_CONTAINER_NAME="questdb-test-${BATS_TEST_NAME//[^a-zA-Z0-9]/-}"
    export QUESTDB_NETWORK_NAME="questdb-test-network"
    export QUESTDB_HTTP_PORT="39009"
    export QUESTDB_PG_PORT="38812"
    export QUESTDB_ILP_PORT="39003"
    
    questdb::export_config
}

teardown() {
    # Clean up test containers and networks
    docker stop "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    docker rm "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    docker network rm "${QUESTDB_NETWORK_NAME}" &>/dev/null || true
}

@test "questdb::docker::is_running returns false when container not running" {
    run questdb::docker::is_running
    assert_failure
}

@test "questdb::docker::container_exists returns false when container doesn't exist" {
    run questdb::docker::container_exists
    assert_failure
}

@test "questdb::docker::create_network creates network if not exists" {
    docker network rm "${QUESTDB_NETWORK_NAME}" &>/dev/null || true
    
    run questdb::docker::create_network
    assert_success
    
    # Verify network was created
    docker network ls | grep -q "${QUESTDB_NETWORK_NAME}"
}

@test "questdb::docker::create_network is idempotent" {
    questdb::docker::create_network
    
    run questdb::docker::create_network
    assert_success
}

@test "questdb::docker::health_check returns false when not running" {
    run questdb::docker::health_check
    assert_failure
}

@test "questdb::docker::logs fails when container doesn't exist" {
    run questdb::docker::logs
    assert_failure
    assert_output --partial "QuestDB container does not exist"
}

@test "questdb::docker::exec fails when container not running" {
    run questdb::docker::exec echo "test"
    assert_failure
    assert_output --partial "QuestDB is not running"
}

@test "questdb::docker::stats returns empty JSON when not running" {
    run questdb::docker::stats
    assert_failure
    [[ "$output" == "{}" ]]
}

# Integration test - requires Docker
@test "questdb::docker::stop succeeds when already stopped" {
    run questdb::docker::stop
    assert_success
    assert_output --partial "QuestDB is not running"
}