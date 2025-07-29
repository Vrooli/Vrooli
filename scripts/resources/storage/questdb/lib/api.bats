#!/usr/bin/env bats

# QuestDB API Functions Tests

setup() {
    load '../../../../testing/test_helper'
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    source "${BATS_TEST_DIRNAME}/../config/messages.sh"
    source "${BATS_TEST_DIRNAME}/common.sh"
    source "${BATS_TEST_DIRNAME}/docker.sh"
    source "${BATS_TEST_DIRNAME}/api.sh"
    
    # Use test configuration
    export QUESTDB_CONTAINER_NAME="questdb-test-api"
    export QUESTDB_HTTP_PORT="39009"
    export QUESTDB_PG_PORT="38812"
    export QUESTDB_ILP_PORT="39003"
    
    questdb::export_config
    questdb::messages::init
}

teardown() {
    docker stop "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    docker rm "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
}

@test "questdb::api::query fails when QuestDB not running" {
    run questdb::api::query "SELECT 1"
    assert_failure
    assert_output --partial "QuestDB is not running"
}

@test "questdb::api::query validates empty query" {
    # Mock docker::is_running to return true
    questdb::docker::is_running() { return 0; }
    export -f questdb::docker::is_running
    
    run questdb::api::query ""
    assert_failure
}

@test "questdb::api::status fails when not running" {
    run questdb::api::status
    assert_failure
    assert_output --partial "QuestDB is not running"
}

@test "questdb::api::list_tables fails when not running" {
    run questdb::api::list_tables
    assert_failure
}

@test "questdb::api::table_schema requires table name" {
    run questdb::api::table_schema ""
    assert_failure
    assert_output --partial "Table name required"
}

@test "questdb::ilp::send requires data" {
    run questdb::ilp::send ""
    assert_failure
    assert_output --partial "No data provided"
}

@test "questdb::ilp::send fails when not running" {
    run questdb::ilp::send "test,tag=value field=1"
    assert_failure
    assert_output --partial "QuestDB is not running"
}

@test "questdb::api::bulk_insert_csv requires existing file" {
    run questdb::api::bulk_insert_csv "test_table" "/nonexistent/file.csv"
    assert_failure
    assert_output --partial "CSV file not found"
}

@test "questdb::api::table_count requires table name" {
    run questdb::api::table_count ""
    assert_failure
    assert_output --partial "Table name required"
}

@test "questdb::api::health_check returns false when not accessible" {
    run questdb::api::health_check
    assert_failure
}