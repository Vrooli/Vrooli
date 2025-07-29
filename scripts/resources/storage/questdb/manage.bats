#!/usr/bin/env bats

# QuestDB Management Script Tests

# Set up test environment
setup() {
    load '../../../testing/test_helper'
    load 'config/defaults'
    load 'config/messages'
    
    # Export test configuration
    export QUESTDB_CONTAINER_NAME="questdb-test"
    export QUESTDB_HTTP_PORT="19009"
    export QUESTDB_PG_PORT="18812"
    export QUESTDB_ILP_PORT="19003"
    export QUESTDB_DATA_DIR="${BATS_TEST_TMPDIR}/questdb/data"
    export QUESTDB_CONFIG_DIR="${BATS_TEST_TMPDIR}/questdb/config"
    export QUESTDB_LOG_DIR="${BATS_TEST_TMPDIR}/questdb/logs"
    
    # Export config for tests
    questdb::export_config
    questdb::messages::init
}

# Clean up after tests
teardown() {
    # Stop and remove test container if exists
    docker stop "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    docker rm "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    
    # Clean up test directories
    rm -rf "${BATS_TEST_TMPDIR}/questdb"
}

@test "manage.sh exists and is executable" {
    [[ -f "${BATS_TEST_DIRNAME}/manage.sh" ]]
    [[ -x "${BATS_TEST_DIRNAME}/manage.sh" ]]
}

@test "manage.sh shows help" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --help
    assert_success
    assert_output --partial "Install and manage QuestDB time-series database"
    assert_output --partial "--action"
    assert_output --partial "--query"
}

@test "manage.sh requires action" {
    run "${BATS_TEST_DIRNAME}/manage.sh"
    assert_failure
    assert_output --partial "No action specified"
}

@test "manage.sh handles invalid action" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --action invalid
    assert_failure
    assert_output --partial "Unknown action: invalid"
}

@test "manage.sh status when not running" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --action status
    assert_failure
    assert_output --partial "QuestDB is not running"
}

@test "manage.sh query requires running instance" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --action query --query "SELECT 1"
    assert_failure
    assert_output --partial "QuestDB is not running"
}

@test "manage.sh query requires query parameter" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --action query
    assert_failure
    assert_output --partial "No query specified"
}

@test "manage.sh logs handles non-existent container" {
    run "${BATS_TEST_DIRNAME}/manage.sh" --action logs
    assert_failure
    assert_output --partial "QuestDB container does not exist"
}