#!/usr/bin/env bats

# QuestDB API Functions Tests

# Expensive setup operations run once per file
setup_file() {
    # Source all dependencies once
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    source "${BATS_TEST_DIRNAME}/../config/messages.sh"
    source "${BATS_TEST_DIRNAME}/common.sh"
    source "${BATS_TEST_DIRNAME}/docker.sh"
    source "${BATS_TEST_DIRNAME}/api.sh"
    
    # Export configuration once
    questdb::export_config
    questdb::messages::init
}

# Lightweight per-test setup
setup() {
    # Use test configuration
    # Mock resources functions to avoid hang
    declare -A DEFAULT_PORTS=(
        ["ollama"]="11434"
        ["agent-s2"]="4113"
        ["browserless"]="3000"
        ["unstructured-io"]="8000"
        ["n8n"]="5678"
        ["node-red"]="1880"
        ["huginn"]="3000"
        ["windmill"]="8000"
        ["judge0"]="2358"
        ["searxng"]="8080"
        ["qdrant"]="6333"
        ["questdb"]="9000"
        ["vault"]="8200"
    )
    resources::get_default_port() { echo "${DEFAULT_PORTS[$1]:-8080}"; }
    export -f resources::get_default_port
    
    export QUESTDB_CONTAINER_NAME="questdb-test-api"
    export QUESTDB_HTTP_PORT="39009"  
    export QUESTDB_PG_PORT="38812"
    export QUESTDB_ILP_PORT="39003"
    
    # Basic mock functions (lightweight)
    mock::network::set_online() { return 0; }
    setup_standard_mocks() { 
        export FORCE="${FORCE:-no}"
        export YES="${YES:-no}"
        export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
        export QUIET="${QUIET:-no}"
        mock::network::set_online
    }
    
    # Setup mocks
    setup_standard_mocks
    
    # Re-source config to ensure export functions are available
    source "${BATS_TEST_DIRNAME}/../config/defaults.sh"
    source "${BATS_TEST_DIRNAME}/../config/messages.sh"
    
    # Re-source only essential API functions
    source "${BATS_TEST_DIRNAME}/api.sh"
    
    # Export configuration
    questdb::export_config
    questdb::messages::init
    
    # Mock Docker operations to prevent real calls and timeouts
    docker() {
        case "$1" in
            "stop"|"rm") 
                echo "Mocked docker $1 $2"
                return 0
                ;;
            *)
                echo "Mocked docker call: $*"
                return 0
                ;;
        esac
    }
    export -f docker
}

teardown() {
    # Remove real Docker calls - these cause timeouts
    # docker stop "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    # docker rm "${QUESTDB_CONTAINER_NAME}" &>/dev/null || true
    echo "# Test cleanup completed (mocked)"
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
