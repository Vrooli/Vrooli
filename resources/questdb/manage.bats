#!/usr/bin/env bats
# Tests for QuestDB manage.sh script

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "questdb"
    
    # Load QuestDB specific configuration once per file
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    QUESTDB_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${QUESTDB_DIR}/config/defaults.sh"
    source "${QUESTDB_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export QUESTDB_CONTAINER_NAME="questdb-test"
    export QUESTDB_HTTP_PORT="19009"
    export QUESTDB_PG_PORT="18812"
    export QUESTDB_ILP_PORT="19003"
    export QUESTDB_DATA_DIR="${BATS_TEST_TMPDIR}/questdb/data"
    export QUESTDB_CONFIG_DIR="${BATS_TEST_TMPDIR}/questdb/config"
    export QUESTDB_LOG_DIR="${BATS_TEST_TMPDIR}/questdb/logs"
    export ACTION="status"
    export QUERY=""
    export LOG_LINES="50"
    export MONITOR_INTERVAL="5"
    
    # Export config functions
    questdb::export_config
    questdb::export_messages
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

@test "manage.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "manage.sh defines required functions" {
    # Functions should be available from setup_file
    declare -f questdb::parse_arguments >/dev/null
    declare -f questdb::main >/dev/null
    declare -f questdb::install >/dev/null
    declare -f questdb::status >/dev/null
}

@test "questdb::usage displays help text" {
    run questdb::usage
    [ "$status" -eq 0 ]
    [[ "$output" == *"Install and manage QuestDB"* ]]
    [[ "$output" == *"--action"* ]]
    [[ "$output" == *"--query"* ]]
}

@test "questdb::parse_arguments sets defaults correctly" {
    questdb::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$QUERY" = "" ]
    [ "$LOG_LINES" = "50" ]
    [ "$MONITOR_INTERVAL" = "5" ]
}

@test "questdb::parse_arguments handles custom values" {
    questdb::parse_arguments \
        --action query \
        --query "SELECT * FROM trades" \
        --lines 100 \
        --interval 10
    
    [ "$ACTION" = "query" ]
    [ "$QUERY" = "SELECT * FROM trades" ]
    [ "$LOG_LINES" = "100" ]
    [ "$MONITOR_INTERVAL" = "10" ]
}

@test "questdb::parse_arguments handles all valid actions" {
    local actions=(
        "install" "uninstall" "start" "stop" "restart" 
        "status" "logs" "diagnose" "monitor" 
        "query" "health" "upgrade"
    )
    
    for action in "${actions[@]}"; do
        questdb::parse_arguments --action "$action"
        [ "$ACTION" = "$action" ]
    done
}

@test "questdb::main calls correct function for install action" {
    # Mock questdb::install function
    questdb::install() { echo 'questdb::install called'; return 0; }
    export -f questdb::install
    
    export ACTION='install'
    run questdb::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "questdb::install called" ]]
}

@test "questdb::main calls correct function for status action" {
    # Mock questdb::status function
    questdb::status() { echo 'questdb::status called'; return 0; }
    export -f questdb::status
    
    export ACTION='status'
    run questdb::main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "questdb::status called" ]]
}

@test "questdb::main validates query arguments" {
    export ACTION='query'
    export QUERY=''
    
    run questdb::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No query specified" ]]
}

@test "questdb::main handles unknown action" {
    export ACTION='unknown-action'
    
    run questdb::main
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action: unknown-action" ]]
}

@test "configuration is exported correctly" {
    questdb::export_config
    
    [ -n "$QUESTDB_HTTP_PORT" ]
    [ -n "$QUESTDB_CONTAINER_NAME" ]
    [ -n "$QUESTDB_IMAGE" ]
}

@test "messages are initialized correctly" {
    questdb::export_messages
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_ALREADY_INSTALLED" ]
    [ -n "$MSG_HEALTHY" ]
}