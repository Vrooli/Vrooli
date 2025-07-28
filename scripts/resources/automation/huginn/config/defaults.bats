#!/usr/bin/env bats
# Tests for Huginn config/defaults.sh

load ../test-fixtures/test_helper

setup() {
    setup_test_environment
    # Source only the defaults configuration
    source "$HUGINN_TEST_DIR/config/defaults.sh"
}

teardown() {
    teardown_test_environment
}

@test "defaults.sh: sets correct port number" {
    [[ "$HUGINN_PORT" == "4111" ]]
}

@test "defaults.sh: sets correct container names" {
    [[ "$CONTAINER_NAME" == "huginn" ]]
    [[ "$DB_CONTAINER_NAME" == "huginn-postgres" ]]
}

@test "defaults.sh: sets correct network name" {
    [[ "$NETWORK_NAME" == "huginn-network" ]]
}

@test "defaults.sh: sets correct data directories" {
    assert_output_contains "$HUGINN_DATA_DIR" "huginn_data"
    assert_output_contains "$HUGINN_DB_DIR" "huginn_db"
    assert_output_contains "$HUGINN_UPLOADS_DIR" "huginn_uploads"
}

@test "defaults.sh: sets correct timeout values" {
    [[ "$HUGINN_STARTUP_TIMEOUT" -eq 300 ]]
    [[ "$HUGINN_HEALTH_CHECK_TIMEOUT" -eq 10 ]]
    [[ "$HUGINN_HEALTH_CHECK_INTERVAL" -eq 2 ]]
    [[ "$RAILS_RUNNER_TIMEOUT" -eq 30 ]]
}

@test "defaults.sh: sets correct default credentials" {
    [[ "$HUGINN_ADMIN_USERNAME" == "admin" ]]
    [[ "$HUGINN_ADMIN_EMAIL" == "admin@huginn.local" ]]
    [[ "$HUGINN_ADMIN_PASSWORD" == "vrooli_huginn_secure_2025" ]]
}

@test "defaults.sh: sets database configuration" {
    [[ "$DATABASE_ADAPTER" == "postgresql" ]]
    [[ "$DATABASE_NAME" == "huginn" ]]
    [[ "$DATABASE_USERNAME" == "huginn" ]]
    [[ "$DATABASE_PASSWORD" == "huginn_secure_password" ]]
    [[ "$DATABASE_HOST" == "huginn-postgres" ]]
    [[ "$DATABASE_PORT" == "5432" ]]
}

@test "defaults.sh: sets correct image name" {
    [[ "$HUGINN_IMAGE" == "huginn/huginn:latest" ]]
}

@test "defaults.sh: export_config function exists" {
    declare -f huginn::export_config >/dev/null
}

@test "defaults.sh: export_config exports required variables" {
    # Call export config
    huginn::export_config
    
    # Check critical variables are exported
    [[ -n "${CONTAINER_NAME:-}" ]]
    [[ -n "${DB_CONTAINER_NAME:-}" ]]
    [[ -n "${HUGINN_PORT:-}" ]]
    [[ -n "${NETWORK_NAME:-}" ]]
}

@test "defaults.sh: respects custom port environment variable" {
    export HUGINN_CUSTOM_PORT="5555"
    source "$HUGINN_TEST_DIR/config/defaults.sh"
    [[ "$HUGINN_PORT" == "5555" ]]
}

@test "defaults.sh: base URL uses correct port" {
    [[ "$HUGINN_BASE_URL" == "http://localhost:$HUGINN_PORT" ]]
}

@test "defaults.sh: volume names are unique" {
    # Ensure volume names don't conflict with other resources
    assert_output_contains "$HUGINN_DATA_VOLUME" "huginn"
    assert_output_contains "$HUGINN_DB_VOLUME" "huginn"
    assert_output_not_contains "$HUGINN_DATA_VOLUME" "node-red"
    assert_output_not_contains "$HUGINN_DATA_VOLUME" "n8n"
}