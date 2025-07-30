#!/usr/bin/env bats

# Expensive setup operations run once per file
setup_file() {
    # Minimal setup_file - most operations moved to lightweight setup()
    true
}

# Tests for Qdrant api.sh functions

# Setup for each test
# Lightweight per-test setup
setup() {
    # Basic mock functions (lightweight)
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
    
    # Original setup content follows...
    # Load shared test infrastructure
    # Lightweight setup instead of heavy common_setup.bash
    setup_standard_mocks() {
        export FORCE="${FORCE:-no}"
        export YES="${YES:-no}"
        export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
        export QUIET="${QUIET:-no}"
        mock::network::set_online() { return 0; }
        export -f mock::network::set_online
    }
    
    # Mock system functions (lightweight)
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    system::is_command() { command -v "$1" >/dev/null 2>&1; }
    
    # Mock basic curl function
    curl() {
        case "$*" in
            *"health"*) echo '{"status":"healthy"}';;
            *) echo '{"success":true}';;
        esac
        return 0
    }
    export -f curl
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export QDRANT_PORT="6333"
    export QDRANT_GRPC_PORT="6334"
    export QDRANT_CONTAINER_NAME="qdrant-test"
    export QDRANT_BASE_URL="http://localhost:6333"
    export QDRANT_GRPC_URL="grpc://localhost:6334"
    export QDRANT_IMAGE="qdrant/qdrant:latest"
    export QDRANT_DATA_DIR="/tmp/qdrant-test/data"
    export QDRANT_CONFIG_DIR="/tmp/qdrant-test/config"
    export QDRANT_SNAPSHOTS_DIR="/tmp/qdrant-test/snapshots"
    export QDRANT_API_KEY="test_qdrant_api_key_123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    QDRANT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$QDRANT_DATA_DIR"
    mkdir -p "$QDRANT_CONFIG_DIR"
    mkdir -p "$QDRANT_SNAPSHOTS_DIR"
    
    # Mock system functions
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".status"*) echo "ok" ;;
            *".version"*) echo "1.7.4" ;;
            *".result.collections"*) echo '[{"name":"test_collection","vectors_count":1000}]' ;;
            *".result.operation_id"*) echo "123" ;;
            *".result[0].score"*) echo "0.95" ;;
            *".result.points[0].id"*) echo "point1" ;;
            *".result.name"*) echo "snapshot_2024-01-15_14-30-00" ;;
            *".result.collections_total"*) echo "5" ;;
            *".result.points_total"*) echo "10000" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock grpcurl for gRPC calls
    grpcurl() {
        case "$*" in
            *"qdrant.Collections/List"*)
                echo '{"collections":[{"name":"test_collection"}]}'
                ;;
            *"qdrant.Points/Search"*)
                echo '{"result":[{"id":"point1","score":0.95}]}'
                ;;
            *) echo "GRPCURL: $*" ;;
        esac
        return 0
    }
    
    # Mock log functions
    
    # Load configuration and messages
    source "${QDRANT_DIR}/config/defaults.sh"
    source "${QDRANT_DIR}/config/messages.sh"
    qdrant::export_config
    qdrant::messages::init
    
    # Load the functions to test
    source "${QDRANT_DIR}/lib/api.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "/tmp/qdrant-test"
}

# Test API health check
@test "qdrant::api::health_check checks Qdrant API health" {
    result=$(qdrant::api::health_check)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "ok" ]]
}

# Test API health check with failure
@test "qdrant::api::health_check handles API failure" {
    # Override curl to simulate failure
    curl() {
        return 1
    }
    
    run qdrant::api::health_check
    [ "$status" -eq 1 ]
}

# Test API version check
@test "qdrant::api::get_version retrieves Qdrant version" {
    result=$(qdrant::api::get_version)
    
    [[ "$result" =~ "1.7.4" ]] || [[ "$result" =~ "version" ]]
}

# Test cluster information
@test "qdrant::api::get_cluster_info retrieves cluster information" {
    result=$(qdrant::api::get_cluster_info)
    
    [[ "$result" =~ "cluster" ]] || [[ "$result" =~ "peer" ]]
}

# Test collection listing
@test "qdrant::api::list_collections lists all collections" {
    result=$(qdrant::api::list_collections)
    
    [[ "$result" =~ "collections" ]] || [[ "$result" =~ "test_collection" ]]
}

# Test collection creation
@test "qdrant::api::create_collection creates a new collection" {
    result=$(qdrant::api::create_collection "test_collection" "1536" "Cosine")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "created" ]]
}

# Test collection creation with invalid parameters
@test "qdrant::api::create_collection validates parameters" {
    run qdrant::api::create_collection "invalid!name" "0" "InvalidMetric"
    [ "$status" -eq 1 ]
}

# Test collection deletion
@test "qdrant::api::delete_collection deletes a collection" {
    result=$(qdrant::api::delete_collection "test_collection")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "deleted" ]]
}

# Test collection information
@test "qdrant::api::get_collection_info retrieves collection details" {
    result=$(qdrant::api::get_collection_info "test_collection")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "test_collection" ]]
}

# Test collection update
@test "qdrant::api::update_collection updates collection configuration" {
    result=$(qdrant::api::update_collection "test_collection" '{"optimizers_config":{"indexing_threshold":10000}}')
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "updated" ]]
}

# Test point insertion
@test "qdrant::api::upsert_points inserts points into collection" {
    local points='{"points":[{"id":"point1","vector":[0.1,0.2,0.3],"payload":{"title":"Test"}}]}'
    
    result=$(qdrant::api::upsert_points "test_collection" "$points")
    
    [[ "$result" =~ "points" ]] || [[ "$result" =~ "upserted" ]]
}

# Test point retrieval
@test "qdrant::api::get_point retrieves a specific point" {
    result=$(qdrant::api::get_point "test_collection" "point1")
    
    [[ "$result" =~ "point" ]] || [[ "$result" =~ "point1" ]]
}

# Test point deletion
@test "qdrant::api::delete_points deletes points from collection" {
    result=$(qdrant::api::delete_points "test_collection" '{"points":["point1"]}')
    
    [[ "$result" =~ "points" ]] || [[ "$result" =~ "deleted" ]]
}

# Test vector search
@test "qdrant::api::search_points performs vector similarity search" {
    local query='{"vector":[0.1,0.2,0.3],"limit":10}'
    
    result=$(qdrant::api::search_points "test_collection" "$query")
    
    [[ "$result" =~ "search" ]] || [[ "$result" =~ "result" ]]
}

# Test vector search with filters
@test "qdrant::api::search_points supports filtering" {
    local query='{"vector":[0.1,0.2,0.3],"filter":{"must":[{"key":"category","match":{"value":"test"}}]},"limit":5}'
    
    result=$(qdrant::api::search_points "test_collection" "$query")
    
    [[ "$result" =~ "search" ]] || [[ "$result" =~ "filtered" ]]
}

# Test point scrolling
@test "qdrant::api::scroll_points scrolls through collection points" {
    result=$(qdrant::api::scroll_points "test_collection" "100")
    
    [[ "$result" =~ "scroll" ]] || [[ "$result" =~ "points" ]]
}

# Test point counting
@test "qdrant::api::count_points counts points in collection" {
    result=$(qdrant::api::count_points "test_collection")
    
    [[ "$result" =~ "count" ]] || [[ "$result" =~ "1000" ]]
}

# Test collection aliasing
@test "qdrant::api::create_alias creates collection alias" {
    result=$(qdrant::api::create_alias "test_collection" "test_alias")
    
    [[ "$result" =~ "alias" ]] || [[ "$result" =~ "created" ]]
}

# Test alias deletion
@test "qdrant::api::delete_alias removes collection alias" {
    result=$(qdrant::api::delete_alias "test_alias")
    
    [[ "$result" =~ "alias" ]] || [[ "$result" =~ "deleted" ]]
}

# Test snapshot creation
@test "qdrant::api::create_snapshot creates collection snapshot" {
    result=$(qdrant::api::create_snapshot "test_collection")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "created" ]]
}

# Test snapshot listing
@test "qdrant::api::list_snapshots lists available snapshots" {
    result=$(qdrant::api::list_snapshots "test_collection")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "list" ]]
}

# Test snapshot restoration
@test "qdrant::api::restore_snapshot restores from snapshot" {
    result=$(qdrant::api::restore_snapshot "test_collection" "snapshot_2024-01-15_14-30-00")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "restored" ]]
}

# Test metrics retrieval
@test "qdrant::api::get_metrics retrieves system metrics" {
    result=$(qdrant::api::get_metrics)
    
    [[ "$result" =~ "metrics" ]] || [[ "$result" =~ "collections_total" ]]
}

# Test telemetry data
@test "qdrant::api::get_telemetry retrieves telemetry information" {
    result=$(qdrant::api::get_telemetry)
    
    [[ "$result" =~ "telemetry" ]] || [[ "$result" =~ "system" ]]
}

# Test API authentication
@test "qdrant::api::authenticate tests API key authentication" {
    result=$(qdrant::api::authenticate "$QDRANT_API_KEY")
    
    [[ "$result" =~ "authenticated" ]] || [[ "$result" =~ "API key" ]]
}

# Test batch operations
@test "qdrant::api::batch_upsert performs batch point operations" {
    local batch_data='{"operations":[{"upsert":{"points":[{"id":"point1","vector":[0.1,0.2,0.3]}]}}]}'
    
    result=$(qdrant::api::batch_upsert "test_collection" "$batch_data")
    
    [[ "$result" =~ "batch" ]] || [[ "$result" =~ "operations" ]]
}

# Test collection optimization
@test "qdrant::api::optimize_collection optimizes collection indices" {
    result=$(qdrant::api::optimize_collection "test_collection")
    
    [[ "$result" =~ "optimize" ]] || [[ "$result" =~ "collection" ]]
}

# Test index rebuild
@test "qdrant::api::rebuild_index rebuilds collection index" {
    result=$(qdrant::api::rebuild_index "test_collection")
    
    [[ "$result" =~ "rebuild" ]] || [[ "$result" =~ "index" ]]
}

# Test payload indexing
@test "qdrant::api::create_payload_index creates payload index" {
    result=$(qdrant::api::create_payload_index "test_collection" "category" "keyword")
    
    [[ "$result" =~ "payload" ]] || [[ "$result" =~ "index" ]]
}

# Test payload index deletion
@test "qdrant::api::delete_payload_index removes payload index" {
    result=$(qdrant::api::delete_payload_index "test_collection" "category")
    
    [[ "$result" =~ "payload" ]] || [[ "$result" =~ "deleted" ]]
}

# Test collection backup
@test "qdrant::api::backup_collection backs up collection data" {
    result=$(qdrant::api::backup_collection "test_collection" "/tmp/backup.json")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection restore
@test "qdrant::api::restore_collection restores collection from backup" {
    result=$(qdrant::api::restore_collection "test_collection" "/tmp/backup.json")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "collection" ]]
}

# Test gRPC API calls
@test "qdrant::api::grpc_list_collections lists collections via gRPC" {
    result=$(qdrant::api::grpc_list_collections)
    
    [[ "$result" =~ "collections" ]] || [[ "$result" =~ "gRPC" ]]
}

# Test gRPC search
@test "qdrant::api::grpc_search performs search via gRPC" {
    result=$(qdrant::api::grpc_search "test_collection" "[0.1,0.2,0.3]" "10")
    
    [[ "$result" =~ "search" ]] || [[ "$result" =~ "gRPC" ]]
}

# Test API rate limiting
@test "qdrant::api::check_rate_limit checks API rate limits" {
    result=$(qdrant::api::check_rate_limit)
    
    [[ "$result" =~ "rate" ]] || [[ "$result" =~ "limit" ]]
}

# Test API error handling
@test "qdrant::api::handle_error processes API errors" {
    result=$(qdrant::api::handle_error "404" "Collection not found")
    
    [[ "$result" =~ "error" ]] || [[ "$result" =~ "404" ]]
}

# Test API timeout configuration
@test "qdrant::api::set_timeout configures API timeout" {
    result=$(qdrant::api::set_timeout "30")
    
    [[ "$result" =~ "timeout" ]] || [[ "$result" =~ "30" ]]
}

# Test API retry mechanism
@test "qdrant::api::retry_request retries failed API requests" {
    result=$(qdrant::api::retry_request "GET" "/health" "3")
    
    [[ "$result" =~ "retry" ]] || [[ "$result" =~ "request" ]]
}

# Test bulk import
@test "qdrant::api::bulk_import imports large datasets" {
    result=$(qdrant::api::bulk_import "test_collection" "/tmp/vectors.jsonl")
    
    [[ "$result" =~ "bulk" ]] || [[ "$result" =~ "import" ]]
}

# Test bulk export
@test "qdrant::api::bulk_export exports collection data" {
    result=$(qdrant::api::bulk_export "test_collection" "/tmp/export.jsonl")
    
    [[ "$result" =~ "bulk" ]] || [[ "$result" =~ "export" ]]
}

# Test collection schema validation
@test "qdrant::api::validate_schema validates collection schema" {
    result=$(qdrant::api::validate_schema "test_collection")
    
    [[ "$result" =~ "schema" ]] || [[ "$result" =~ "valid" ]]
}

# Test data integrity check
@test "qdrant::api::check_integrity verifies data integrity" {
    result=$(qdrant::api::check_integrity "test_collection")
    
    [[ "$result" =~ "integrity" ]] || [[ "$result" =~ "check" ]]
}

# Test performance benchmarking
@test "qdrant::api::benchmark_performance measures API performance" {
    result=$(qdrant::api::benchmark_performance "test_collection" "100")
    
    [[ "$result" =~ "benchmark" ]] || [[ "$result" =~ "performance" ]]
}

# Test API monitoring
@test "qdrant::api::monitor_performance monitors API performance" {
    result=$(qdrant::api::monitor_performance)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "performance" ]]
}

# Test collection statistics
@test "qdrant::api::get_collection_stats retrieves collection statistics" {
    result=$(qdrant::api::get_collection_stats "test_collection")
    
    [[ "$result" =~ "statistics" ]] || [[ "$result" =~ "collection" ]]
}

# Test query explanation
@test "qdrant::api::explain_query explains search queries" {
    local query='{"vector":[0.1,0.2,0.3],"limit":10}'
    
    result=$(qdrant::api::explain_query "test_collection" "$query")
    
    [[ "$result" =~ "explain" ]] || [[ "$result" =~ "query" ]]
}

# Test API documentation
@test "qdrant::api::get_documentation retrieves API documentation" {
    result=$(qdrant::api::get_documentation)
    
    [[ "$result" =~ "documentation" ]] || [[ "$result" =~ "API" ]]
}

# Test API status
@test "qdrant::api::get_status retrieves comprehensive API status" {
    result=$(qdrant::api::get_status)
    
    [[ "$result" =~ "status" ]] || [[ "$result" =~ "API" ]]
}

# Test connection testing
@test "qdrant::api::test_connection tests API connection" {
    result=$(qdrant::api::test_connection)
    
    [[ "$result" =~ "connection" ]] || [[ "$result" =~ "test" ]]
}

# Test API configuration
@test "qdrant::api::configure_api configures API settings" {
    result=$(qdrant::api::configure_api)
    
    [[ "$result" =~ "configure" ]] || [[ "$result" =~ "API" ]]
}

# Test debug information
@test "qdrant::api::get_debug_info retrieves debug information" {
    result=$(qdrant::api::get_debug_info)
    
    [[ "$result" =~ "debug" ]] || [[ "$result" =~ "information" ]]
}

# Test API versioning
@test "qdrant::api::check_api_version verifies API version compatibility" {
    result=$(qdrant::api::check_api_version)
    
    [[ "$result" =~ "API" ]] || [[ "$result" =~ "version" ]]
}

# Test collection migration
@test "qdrant::api::migrate_collection migrates collection between clusters" {
    result=$(qdrant::api::migrate_collection "test_collection" "http://target:6333")
    
    [[ "$result" =~ "migrate" ]] || [[ "$result" =~ "collection" ]]
}

# Test cluster synchronization
@test "qdrant::api::sync_cluster synchronizes cluster data" {
    result=$(qdrant::api::sync_cluster)
    
    [[ "$result" =~ "sync" ]] || [[ "$result" =~ "cluster" ]]
}

# Test shard management
@test "qdrant::api::manage_shards manages collection shards" {
    result=$(qdrant::api::manage_shards "test_collection" "redistribute")
    
    [[ "$result" =~ "shard" ]] || [[ "$result" =~ "manage" ]]
}

# Test replica configuration
@test "qdrant::api::configure_replicas configures data replication" {
    result=$(qdrant::api::configure_replicas "test_collection" "2")
    
    [[ "$result" =~ "replica" ]] || [[ "$result" =~ "configure" ]]
}

# Test consistency check
@test "qdrant::api::check_consistency verifies data consistency" {
    result=$(qdrant::api::check_consistency "test_collection")
    
    [[ "$result" =~ "consistency" ]] || [[ "$result" =~ "check" ]]
}

# Test disaster recovery
@test "qdrant::api::disaster_recovery initiates disaster recovery" {
    result=$(qdrant::api::disaster_recovery "test_collection")
    
    [[ "$result" =~ "disaster" ]] || [[ "$result" =~ "recovery" ]]
}
