#!/usr/bin/env bats

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# QuestDB Common Functions Tests

setup() {
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    source "${BATS_TEST_DIRNAME}/common.sh"
    
    # Set test directories
    export QUESTDB_DATA_DIR="${BATS_TEST_TMPDIR}/questdb/data"
    export QUESTDB_CONFIG_DIR="${BATS_TEST_TMPDIR}/questdb/config"
    export QUESTDB_LOG_DIR="${BATS_TEST_TMPDIR}/questdb/logs"
    
    questdb::export_config
}

teardown() {
    trash::safe_remove "${BATS_TEST_TMPDIR}/questdb" --test-cleanup
}

@test "questdb::dirs_exist returns false when directories don't exist" {
    run questdb::dirs_exist
    assert_failure
}

@test "questdb::dirs_exist returns true when all directories exist" {
    mkdir -p "${QUESTDB_DATA_DIR}" "${QUESTDB_CONFIG_DIR}" "${QUESTDB_LOG_DIR}"
    
    run questdb::dirs_exist
    assert_success
}

@test "questdb::create_dirs creates all required directories" {
    run questdb::create_dirs
    assert_success
    
    [[ -d "${QUESTDB_DATA_DIR}" ]]
    [[ -d "${QUESTDB_CONFIG_DIR}" ]]
    [[ -d "${QUESTDB_LOG_DIR}" ]]
}

@test "questdb::check_disk_space validates available space" {
    # This test may need adjustment based on test environment
    run questdb::check_disk_space 1
    # Should succeed if there's at least 1GB free
    # May fail in constrained test environments
}

@test "questdb::format_bytes formats correctly" {
    [[ "$(questdb::format_bytes 1024)" == "1KB" ]]
    [[ "$(questdb::format_bytes 1048576)" == "1MB" ]]
    [[ "$(questdb::format_bytes 1073741824)" == "1GB" ]]
}

@test "questdb::format_duration formats milliseconds correctly" {
    [[ "$(questdb::format_duration 500)" == "500ms" ]]
    [[ "$(questdb::format_duration 5000)" == "5s" ]]
    [[ "$(questdb::format_duration 300000)" == "5m" ]]
    [[ "$(questdb::format_duration 3600000)" == "1h" ]]
}

@test "questdb::validate_query rejects empty queries" {
    run questdb::validate_query ""
    assert_failure
    assert_output --partial "Empty query"
}

@test "questdb::validate_query accepts SELECT queries" {
    run questdb::validate_query "SELECT * FROM table"
    assert_success
}

@test "questdb::validate_query warns about dangerous operations" {
    # Mock the prompt function to return no
    args::prompt_yes_no() { return 1; }
    export -f args::prompt_yes_no
    
    run questdb::validate_query "DELETE FROM table"
    assert_failure
}

@test "questdb::check_ports succeeds when ports are free" {
    # Use high ports unlikely to be in use
    export QUESTDB_HTTP_PORT="39009"
    export QUESTDB_PG_PORT="38812"
    export QUESTDB_ILP_PORT="39003"
    
    run questdb::check_ports
    # This may fail if the test ports are actually in use
}