#!/usr/bin/env bats

# QuestDB Configuration Defaults Tests

setup() {
    load '../../../../testing/test_helper'
    source "${BATS_TEST_DIRNAME}/defaults.sh"
}

@test "questdb::export_config sets default ports" {
    questdb::export_config
    
    [[ -n "${QUESTDB_HTTP_PORT}" ]]
    [[ -n "${QUESTDB_PG_PORT}" ]]
    [[ -n "${QUESTDB_ILP_PORT}" ]]
    [[ "${QUESTDB_PG_PORT}" == "8812" ]]
}

@test "questdb::export_config sets container configuration" {
    questdb::export_config
    
    [[ "${QUESTDB_CONTAINER_NAME}" == "questdb" ]]
    [[ "${QUESTDB_IMAGE}" == "questdb/questdb:8.1.2" ]]
    [[ -n "${QUESTDB_BASE_URL}" ]]
    [[ -n "${QUESTDB_PG_URL}" ]]
}

@test "questdb::export_config sets data directories" {
    questdb::export_config
    
    [[ "${QUESTDB_DATA_DIR}" == "${HOME}/.questdb/data" ]]
    [[ "${QUESTDB_CONFIG_DIR}" == "${HOME}/.questdb/config" ]]
    [[ "${QUESTDB_LOG_DIR}" == "${HOME}/.questdb/logs" ]]
}

@test "questdb::export_config sets performance parameters" {
    questdb::export_config
    
    [[ "${QUESTDB_SHARED_WORKER_COUNT}" == "2" ]]
    [[ "${QUESTDB_HTTP_WORKER_COUNT}" == "2" ]]
    [[ "${QUESTDB_WAL_ENABLED}" == "true" ]]
    [[ -n "${QUESTDB_COMMIT_LAG}" ]]
}

@test "questdb::export_config sets security configuration" {
    questdb::export_config
    
    [[ "${QUESTDB_HTTP_SECURITY_READONLY}" == "false" ]]
    [[ "${QUESTDB_PG_USER}" == "admin" ]]
    [[ -n "${QUESTDB_PG_PASSWORD}" ]]
}

@test "questdb::export_config sets health check parameters" {
    questdb::export_config
    
    [[ "${QUESTDB_HEALTH_CHECK_INTERVAL}" -gt 0 ]]
    [[ "${QUESTDB_HEALTH_CHECK_MAX_ATTEMPTS}" -gt 0 ]]
    [[ "${QUESTDB_API_TIMEOUT}" -gt 0 ]]
}

@test "questdb::export_config exports all variables" {
    questdb::export_config
    
    # Check key exports
    [[ -n "${QUESTDB_HTTP_PORT}" ]]
    [[ -n "${QUESTDB_BASE_URL}" ]]
    [[ -n "${QUESTDB_CONTAINER_NAME}" ]]
    [[ -n "${QUESTDB_DATA_DIR}" ]]
}

@test "questdb::export_config respects custom ports" {
    export QUESTDB_CUSTOM_HTTP_PORT="29009"
    export QUESTDB_CUSTOM_PG_PORT="28812"
    export QUESTDB_CUSTOM_ILP_PORT="29003"
    
    questdb::export_config
    
    [[ "${QUESTDB_HTTP_PORT}" == "29009" ]]
    [[ "${QUESTDB_PG_PORT}" == "28812" ]]
    [[ "${QUESTDB_ILP_PORT}" == "29003" ]]
}

@test "questdb::export_config is idempotent" {
    questdb::export_config
    local first_port="${QUESTDB_HTTP_PORT}"
    
    questdb::export_config
    [[ "${QUESTDB_HTTP_PORT}" == "${first_port}" ]]
}