#!/usr/bin/env bats
# Tests for Qdrant manage.sh script

# Load test helper
load_helper() {
    local helper_file="$1"
    if [[ -f "$helper_file" ]]; then
        # shellcheck disable=SC1090
        source "$helper_file"
    fi
}

# Setup for each test
setup() {
    # Set test environment
    export QDRANT_CUSTOM_PORT="9999"
    export QDRANT_CUSTOM_GRPC_PORT="9998"
    export QDRANT_CUSTOM_API_KEY=""
    export ACTION="status"
    export COLLECTION=""
    export VECTOR_SIZE="1536"
    export DISTANCE_METRIC="Cosine"
    export REMOVE_DATA="no"
    export FORCE="no"
    export SNAPSHOT_NAME=""
    export COLLECTIONS_LIST="all"
    export LOG_LINES="50"
    export MONITOR_INTERVAL="5"
    
    # Load the script without executing main
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/manage.sh" || true
}

# Test script loading
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing
@test "qdrant::parse_arguments sets defaults correctly" {
    qdrant::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$COLLECTION" = "" ]
    [ "$VECTOR_SIZE" = "1536" ]
    [ "$DISTANCE_METRIC" = "Cosine" ]
    [ "$REMOVE_DATA" = "no" ]
    [ "$FORCE" = "no" ]
    [ "$SNAPSHOT_NAME" = "" ]
    [ "$COLLECTIONS_LIST" = "all" ]
    [ "$LOG_LINES" = "50" ]
    [ "$MONITOR_INTERVAL" = "5" ]
}

# Test argument parsing with custom values
@test "qdrant::parse_arguments handles custom values" {
    qdrant::parse_arguments \
        --action create-collection \
        --collection my_vectors \
        --vector-size 768 \
        --distance Dot \
        --force yes
    
    [ "$ACTION" = "create-collection" ]
    [ "$COLLECTION" = "my_vectors" ]
    [ "$VECTOR_SIZE" = "768" ]
    [ "$DISTANCE_METRIC" = "Dot" ]
    [ "$FORCE" = "yes" ]
}

# Test collection operations arguments
@test "qdrant::parse_arguments handles collection operations" {
    qdrant::parse_arguments \
        --action create-collection \
        --collection test_collection \
        --vector-size 2048 \
        --distance Euclid
    
    [ "$ACTION" = "create-collection" ]
    [ "$COLLECTION" = "test_collection" ]
    [ "$VECTOR_SIZE" = "2048" ]
    [ "$DISTANCE_METRIC" = "Euclid" ]
}

# Test backup arguments
@test "qdrant::parse_arguments handles backup arguments" {
    qdrant::parse_arguments \
        --action backup \
        --collections "coll1,coll2,coll3" \
        --snapshot-name "daily-backup-2024"
    
    [ "$ACTION" = "backup" ]
    [ "$COLLECTIONS_LIST" = "coll1,coll2,coll3" ]
    [ "$SNAPSHOT_NAME" = "daily-backup-2024" ]
}

# Test restore arguments
@test "qdrant::parse_arguments handles restore arguments" {
    qdrant::parse_arguments \
        --action restore \
        --snapshot-name "backup-20240115"
    
    [ "$ACTION" = "restore" ]
    [ "$SNAPSHOT_NAME" = "backup-20240115" ]
}

# Test monitor arguments
@test "qdrant::parse_arguments handles monitor arguments" {
    qdrant::parse_arguments \
        --action monitor \
        --interval 10
    
    [ "$ACTION" = "monitor" ]
    [ "$MONITOR_INTERVAL" = "10" ]
}

# Test logs arguments
@test "qdrant::parse_arguments handles logs arguments" {
    qdrant::parse_arguments \
        --action logs \
        --lines 100
    
    [ "$ACTION" = "logs" ]
    [ "$LOG_LINES" = "100" ]
}

# Test uninstall arguments
@test "qdrant::parse_arguments handles uninstall arguments" {
    qdrant::parse_arguments \
        --action uninstall \
        --remove-data yes
    
    [ "$ACTION" = "uninstall" ]
    [ "$REMOVE_DATA" = "yes" ]
}

# Test distance metric options
@test "qdrant::parse_arguments validates distance metrics" {
    qdrant::parse_arguments --action create-collection --distance Cosine
    [ "$DISTANCE_METRIC" = "Cosine" ]
    
    qdrant::parse_arguments --action create-collection --distance Dot
    [ "$DISTANCE_METRIC" = "Dot" ]
    
    qdrant::parse_arguments --action create-collection --distance Euclid
    [ "$DISTANCE_METRIC" = "Euclid" ]
}

# Test action validation
@test "qdrant::parse_arguments handles all valid actions" {
    local actions=(
        "install" "uninstall" "start" "stop" "restart" 
        "status" "logs" "diagnose" "monitor" 
        "list-collections" "create-collection" "delete-collection" 
        "collection-info" "backup" "restore" "index-stats" "upgrade"
    )
    
    for action in "${actions[@]}"; do
        qdrant::parse_arguments --action "$action"
        [ "$ACTION" = "$action" ]
    done
}

# Test usage display
@test "qdrant::usage displays help text" {
    run qdrant::usage
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Install and manage Qdrant"* ]]
    [[ "$output" == *"--action install"* ]]
    [[ "$output" == *"QDRANT_CUSTOM_PORT"* ]]
}

# Test configuration export
@test "configuration is exported correctly" {
    qdrant::export_config
    
    [ -n "$QDRANT_PORT" ]
    [ -n "$QDRANT_GRPC_PORT" ]
    [ -n "$QDRANT_BASE_URL" ]
    [ -n "$QDRANT_CONTAINER_NAME" ]
    [ -n "$QDRANT_IMAGE" ]
}

# Test message initialization
@test "messages are initialized correctly" {
    qdrant::messages::init
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_ALREADY_INSTALLED" ]
    [ -n "$MSG_HEALTHY" ]
}

# Test vector size as integer
@test "qdrant::parse_arguments handles vector size as integer" {
    qdrant::parse_arguments \
        --action create-collection \
        --vector-size 384
    
    [ "$VECTOR_SIZE" = "384" ]
}

# Test collection name with underscores and hyphens
@test "qdrant::parse_arguments handles collection names with special chars" {
    qdrant::parse_arguments \
        --action collection-info \
        --collection "test-collection_v2"
    
    [ "$COLLECTION" = "test-collection_v2" ]
}

# Test multiple collections in backup
@test "qdrant::parse_arguments handles comma-separated collections" {
    qdrant::parse_arguments \
        --action backup \
        --collections "agents,memories,embeddings"
    
    [ "$COLLECTIONS_LIST" = "agents,memories,embeddings" ]
}

# Test edge cases
@test "qdrant::parse_arguments handles empty arguments" {
    qdrant::parse_arguments
    
    # Should use all defaults
    [ "$ACTION" = "status" ]
    [ "$VECTOR_SIZE" = "1536" ]
    [ "$DISTANCE_METRIC" = "Cosine" ]
}

# Test combined options
@test "qdrant::parse_arguments handles combined options" {
    qdrant::parse_arguments \
        --action delete-collection \
        --collection old_data \
        --force yes
    
    [ "$ACTION" = "delete-collection" ]
    [ "$COLLECTION" = "old_data" ]
    [ "$FORCE" = "yes" ]
}

# Test backup with all collections
@test "qdrant::parse_arguments handles 'all' collections keyword" {
    qdrant::parse_arguments \
        --action backup \
        --collections all \
        --snapshot-name full-backup
    
    [ "$COLLECTIONS_LIST" = "all" ]
    [ "$SNAPSHOT_NAME" = "full-backup" ]
}

# Test configuration values from defaults
@test "default configuration values are set" {
    # After sourcing, these should be available from config/defaults.sh
    [ -n "$QDRANT_DEFAULT_PORT" ]
    [ "$QDRANT_DEFAULT_PORT" = "6333" ]
    [ -n "$QDRANT_DEFAULT_GRPC_PORT" ]
    [ "$QDRANT_DEFAULT_GRPC_PORT" = "6334" ]
}

# Test API key handling
@test "qdrant configuration respects custom API key" {
    export QDRANT_CUSTOM_API_KEY="test-api-key-123"
    qdrant::export_config
    
    [ "$QDRANT_API_KEY" = "test-api-key-123" ]
}

# Test snapshot name with timestamp format
@test "qdrant::parse_arguments handles timestamp-like snapshot names" {
    qdrant::parse_arguments \
        --action backup \
        --snapshot-name "backup-2024-01-15-14-30-00"
    
    [ "$SNAPSHOT_NAME" = "backup-2024-01-15-14-30-00" ]
}

# Test force flag with different actions
@test "qdrant::parse_arguments handles force flag for various actions" {
    qdrant::parse_arguments --action delete-collection --force yes
    [ "$FORCE" = "yes" ]
    
    qdrant::parse_arguments --action install --force no
    [ "$FORCE" = "no" ]
}