#!/usr/bin/env bats

# QuestDB Messages Configuration Tests

setup() {
    load '../../../../testing/test_helper'
    source "${BATS_TEST_DIRNAME}/defaults.sh"
    source "${BATS_TEST_DIRNAME}/messages.sh"
    
    # Initialize for tests
    questdb::export_config
    questdb::messages::init
}

@test "questdb::messages::init creates status messages" {
    questdb::messages::init
    
    [[ -n "${QUESTDB_STATUS_MESSAGES[checking]}" ]]
    [[ -n "${QUESTDB_STATUS_MESSAGES[running]}" ]]
    [[ -n "${QUESTDB_STATUS_MESSAGES[not_running]}" ]]
    [[ -n "${QUESTDB_STATUS_MESSAGES[ready]}" ]]
}

@test "questdb::messages::init creates install messages" {
    questdb::messages::init
    
    [[ -n "${QUESTDB_INSTALL_MESSAGES[checking_docker]}" ]]
    [[ -n "${QUESTDB_INSTALL_MESSAGES[creating_directories]}" ]]
    [[ -n "${QUESTDB_INSTALL_MESSAGES[success]}" ]]
    [[ -n "${QUESTDB_INSTALL_MESSAGES[failed]}" ]]
}

@test "questdb::messages::init creates API messages" {
    questdb::messages::init
    
    [[ -n "${QUESTDB_API_MESSAGES[executing_query]}" ]]
    [[ -n "${QUESTDB_API_MESSAGES[query_success]}" ]]
    [[ -n "${QUESTDB_API_MESSAGES[query_failed]}" ]]
    [[ -n "${QUESTDB_API_MESSAGES[connection_failed]}" ]]
}

@test "questdb::messages::init creates error messages" {
    questdb::messages::init
    
    [[ -n "${QUESTDB_ERROR_MESSAGES[docker_not_found]}" ]]
    [[ -n "${QUESTDB_ERROR_MESSAGES[port_conflict]}" ]]
    [[ -n "${QUESTDB_ERROR_MESSAGES[timeout]}" ]]
}

@test "questdb::messages::init creates info messages" {
    questdb::messages::init
    
    [[ -n "${QUESTDB_INFO_MESSAGES[web_console]}" ]]
    [[ -n "${QUESTDB_INFO_MESSAGES[pg_connection]}" ]]
    [[ -n "${QUESTDB_INFO_MESSAGES[performance]}" ]]
}

@test "questdb::messages::init includes port numbers in messages" {
    questdb::messages::init
    
    assert_contains "${QUESTDB_STATUS_MESSAGES[running]}" "${QUESTDB_HTTP_PORT}"
    assert_contains "${QUESTDB_STATUS_MESSAGES[running]}" "${QUESTDB_PG_PORT}"
}

@test "questdb::messages::init includes URLs in info messages" {
    questdb::messages::init
    
    assert_contains "${QUESTDB_INFO_MESSAGES[web_console]}" "${QUESTDB_BASE_URL}"
    assert_contains "${QUESTDB_INFO_MESSAGES[pg_connection]}" "${QUESTDB_PG_URL}"
}

@test "questdb::messages::init is idempotent" {
    questdb::messages::init
    local first_msg="${QUESTDB_STATUS_MESSAGES[checking]}"
    
    questdb::messages::init
    [[ "${QUESTDB_STATUS_MESSAGES[checking]}" == "${first_msg}" ]]
}
