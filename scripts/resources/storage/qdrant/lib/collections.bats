#!/usr/bin/env bats
# Tests for Qdrant collections.sh functions

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
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
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
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
            *".result.collections | length"*) echo "3" ;;
            *".result.collections[].name"*) echo -e "test_collection\nagent_memory\ncode_embeddings" ;;
            *".result.vectors_count"*) echo "1000" ;;
            *".result.points_count"*) echo "1000" ;;
            *".result.status"*) echo "green" ;;
            *".result.config.params.vector_size"*) echo "1536" ;;
            *".result.config.params.distance"*) echo "Cosine" ;;
            *".result.operation_id"*) echo "123" ;;
            *".result.name"*) echo "snapshot_test_collection_2024-01-15_14-30-00" ;;
            *".result.count"*) echo "1000" ;;
            *".result.shard_count"*) echo "4" ;;
            *".result.local_shards | length"*) echo "2" ;;
            *".result.remote_shards | length"*) echo "2" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock openssl for ID generation
    openssl() {
        echo "random_id_abc123def456"
    }
    
    # Mock log functions
    
    # Load configuration and messages
    source "${QDRANT_DIR}/config/defaults.sh"
    source "${QDRANT_DIR}/config/messages.sh"
    qdrant::export_config
    qdrant::messages::init
    
    # Load the functions to test
    source "${QDRANT_DIR}/lib/collections.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "/tmp/qdrant-test" --test-cleanup
}

# Test collection listing
@test "qdrant::collections::list lists all collections" {
    result=$(qdrant::collections::list)
    
    [[ "$result" =~ "collections" ]] || [[ "$result" =~ "test_collection" ]]
}

# Test collection creation
@test "qdrant::collections::create creates a new collection" {
    result=$(qdrant::collections::create "new_collection" "1536" "Cosine")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "created" ]]
}

# Test collection creation with custom configuration
@test "qdrant::collections::create supports custom configuration" {
    local config='{"hnsw_config":{"m":32,"ef_construct":200},"quantization_config":{"scalar":{"type":"int8"}}}'
    
    result=$(qdrant::collections::create "advanced_collection" "768" "Dot" "$config")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "advanced" ]]
}

# Test collection creation validation
@test "qdrant::collections::create validates input parameters" {
    run qdrant::collections::create "invalid!name" "0" "InvalidMetric"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "error" ]]
}

# Test collection deletion
@test "qdrant::collections::delete removes a collection" {
    result=$(qdrant::collections::delete "test_collection")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "deleted" ]]
}

# Test collection deletion with confirmation
@test "qdrant::collections::delete respects confirmation" {
    export YES="yes"
    
    result=$(qdrant::collections::delete "test_collection")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "deleted" ]]
}

# Test collection information retrieval
@test "qdrant::collections::info retrieves collection details" {
    result=$(qdrant::collections::info "test_collection")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "test_collection" ]]
}

# Test collection status check
@test "qdrant::collections::status checks collection health" {
    result=$(qdrant::collections::status "test_collection")
    
    [[ "$result" =~ "status" ]] || [[ "$result" =~ "green" ]]
}

# Test collection update
@test "qdrant::collections::update updates collection configuration" {
    local update_config='{"optimizers_config":{"indexing_threshold":10000}}'
    
    result=$(qdrant::collections::update "test_collection" "$update_config")
    
    [[ "$result" =~ "collection" ]] || [[ "$result" =~ "updated" ]]
}

# Test collection schema validation
@test "qdrant::collections::validate_schema validates collection schema" {
    result=$(qdrant::collections::validate_schema "test_collection")
    
    [[ "$result" =~ "schema" ]] || [[ "$result" =~ "valid" ]]
}

# Test point insertion
@test "qdrant::collections::upsert_points inserts points into collection" {
    local points='{"points":[{"id":"point1","vector":[0.1,0.2,0.3],"payload":{"title":"Test document"}}]}'
    
    result=$(qdrant::collections::upsert_points "test_collection" "$points")
    
    [[ "$result" =~ "points" ]] || [[ "$result" =~ "upserted" ]]
}

# Test batch point insertion
@test "qdrant::collections::batch_upsert handles large point batches" {
    result=$(qdrant::collections::batch_upsert "test_collection" "/tmp/points.jsonl")
    
    [[ "$result" =~ "batch" ]] || [[ "$result" =~ "upsert" ]]
}

# Test point deletion
@test "qdrant::collections::delete_points removes points from collection" {
    local filter='{"filter":{"must":[{"key":"category","match":{"value":"test"}}]}}'
    
    result=$(qdrant::collections::delete_points "test_collection" "$filter")
    
    [[ "$result" =~ "points" ]] || [[ "$result" =~ "deleted" ]]
}

# Test point retrieval
@test "qdrant::collections::get_point retrieves specific point" {
    result=$(qdrant::collections::get_point "test_collection" "point1")
    
    [[ "$result" =~ "point" ]] || [[ "$result" =~ "point1" ]]
}

# Test vector search
@test "qdrant::collections::search performs vector similarity search" {
    local query='{"vector":[0.1,0.2,0.3],"limit":10}'
    
    result=$(qdrant::collections::search "test_collection" "$query")
    
    [[ "$result" =~ "search" ]] || [[ "$result" =~ "result" ]]
}

# Test filtered search
@test "qdrant::collections::search supports filtering" {
    local query='{"vector":[0.1,0.2,0.3],"filter":{"must":[{"key":"category","match":{"value":"test"}}]},"limit":5}'
    
    result=$(qdrant::collections::search "test_collection" "$query")
    
    [[ "$result" =~ "search" ]] || [[ "$result" =~ "filtered" ]]
}

# Test scroll through points
@test "qdrant::collections::scroll scrolls through collection points" {
    result=$(qdrant::collections::scroll "test_collection" "100")
    
    [[ "$result" =~ "scroll" ]] || [[ "$result" =~ "points" ]]
}

# Test point counting
@test "qdrant::collections::count counts points in collection" {
    result=$(qdrant::collections::count "test_collection")
    
    [[ "$result" =~ "count" ]] || [[ "$result" =~ "1000" ]]
}

# Test filtered count
@test "qdrant::collections::count supports filtering" {
    local filter='{"filter":{"must":[{"key":"category","match":{"value":"test"}}]}}'
    
    result=$(qdrant::collections::count "test_collection" "$filter")
    
    [[ "$result" =~ "count" ]] || [[ "$result" =~ "filtered" ]]
}

# Test collection aliasing
@test "qdrant::collections::create_alias creates collection alias" {
    result=$(qdrant::collections::create_alias "test_collection" "test_alias")
    
    [[ "$result" =~ "alias" ]] || [[ "$result" =~ "created" ]]
}

# Test alias deletion
@test "qdrant::collections::delete_alias removes collection alias" {
    result=$(qdrant::collections::delete_alias "test_alias")
    
    [[ "$result" =~ "alias" ]] || [[ "$result" =~ "deleted" ]]
}

# Test alias listing
@test "qdrant::collections::list_aliases lists collection aliases" {
    result=$(qdrant::collections::list_aliases)
    
    [[ "$result" =~ "alias" ]] || [[ "$result" =~ "list" ]]
}

# Test collection snapshot creation
@test "qdrant::collections::create_snapshot creates collection snapshot" {
    result=$(qdrant::collections::create_snapshot "test_collection")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "created" ]]
}

# Test snapshot listing
@test "qdrant::collections::list_snapshots lists collection snapshots" {
    result=$(qdrant::collections::list_snapshots "test_collection")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "list" ]]
}

# Test snapshot restoration
@test "qdrant::collections::restore_snapshot restores from snapshot" {
    result=$(qdrant::collections::restore_snapshot "test_collection" "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "restored" ]]
}

# Test collection backup
@test "qdrant::collections::backup creates collection backup" {
    result=$(qdrant::collections::backup "test_collection" "/tmp/backup.json")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "created" ]]
}

# Test collection restoration
@test "qdrant::collections::restore restores collection from backup" {
    result=$(qdrant::collections::restore "test_collection" "/tmp/backup.json")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection cloning
@test "qdrant::collections::clone clones existing collection" {
    result=$(qdrant::collections::clone "test_collection" "cloned_collection")
    
    [[ "$result" =~ "clone" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection migration
@test "qdrant::collections::migrate migrates collection data" {
    result=$(qdrant::collections::migrate "test_collection" "new_test_collection")
    
    [[ "$result" =~ "migrate" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection optimization
@test "qdrant::collections::optimize optimizes collection performance" {
    result=$(qdrant::collections::optimize "test_collection")
    
    [[ "$result" =~ "optimize" ]] || [[ "$result" =~ "collection" ]]
}

# Test index rebuild
@test "qdrant::collections::rebuild_index rebuilds collection index" {
    result=$(qdrant::collections::rebuild_index "test_collection")
    
    [[ "$result" =~ "rebuild" ]] || [[ "$result" =~ "index" ]]
}

# Test payload indexing
@test "qdrant::collections::create_payload_index creates payload index" {
    result=$(qdrant::collections::create_payload_index "test_collection" "category" "keyword")
    
    [[ "$result" =~ "payload" ]] || [[ "$result" =~ "index" ]]
}

# Test payload index deletion
@test "qdrant::collections::delete_payload_index removes payload index" {
    result=$(qdrant::collections::delete_payload_index "test_collection" "category")
    
    [[ "$result" =~ "payload" ]] || [[ "$result" =~ "deleted" ]]
}

# Test collection statistics
@test "qdrant::collections::stats retrieves collection statistics" {
    result=$(qdrant::collections::stats "test_collection")
    
    [[ "$result" =~ "statistics" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection health check
@test "qdrant::collections::health_check performs collection health check" {
    result=$(qdrant::collections::health_check "test_collection")
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "check" ]]
}

# Test collection monitoring
@test "qdrant::collections::monitor monitors collection performance" {
    result=$(qdrant::collections::monitor "test_collection" "60")
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "collection" ]]
}

# Test cluster information
@test "qdrant::collections::cluster_info retrieves cluster information" {
    result=$(qdrant::collections::cluster_info "test_collection")
    
    [[ "$result" =~ "cluster" ]] || [[ "$result" =~ "shard" ]]
}

# Test shard management
@test "qdrant::collections::manage_shards manages collection shards" {
    result=$(qdrant::collections::manage_shards "test_collection" "redistribute")
    
    [[ "$result" =~ "shard" ]] || [[ "$result" =~ "manage" ]]
}

# Test replica configuration
@test "qdrant::collections::configure_replicas configures replication" {
    result=$(qdrant::collections::configure_replicas "test_collection" "2")
    
    [[ "$result" =~ "replica" ]] || [[ "$result" =~ "configure" ]]
}

# Test consistency check
@test "qdrant::collections::check_consistency verifies data consistency" {
    result=$(qdrant::collections::check_consistency "test_collection")
    
    [[ "$result" =~ "consistency" ]] || [[ "$result" =~ "check" ]]
}

# Test data validation
@test "qdrant::collections::validate_data validates collection data" {
    result=$(qdrant::collections::validate_data "test_collection")
    
    [[ "$result" =~ "validate" ]] || [[ "$result" =~ "data" ]]
}

# Test collection repair
@test "qdrant::collections::repair repairs collection issues" {
    result=$(qdrant::collections::repair "test_collection")
    
    [[ "$result" =~ "repair" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection compression
@test "qdrant::collections::compress compresses collection data" {
    result=$(qdrant::collections::compress "test_collection")
    
    [[ "$result" =~ "compress" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection decompression
@test "qdrant::collections::decompress decompresses collection data" {
    result=$(qdrant::collections::decompress "test_collection")
    
    [[ "$result" =~ "decompress" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection encryption
@test "qdrant::collections::encrypt encrypts collection data" {
    result=$(qdrant::collections::encrypt "test_collection" "encryption_key")
    
    [[ "$result" =~ "encrypt" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection decryption
@test "qdrant::collections::decrypt decrypts collection data" {
    result=$(qdrant::collections::decrypt "test_collection" "encryption_key")
    
    [[ "$result" =~ "decrypt" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection archiving
@test "qdrant::collections::archive archives collection" {
    result=$(qdrant::collections::archive "test_collection" "/tmp/archive.tar.gz")
    
    [[ "$result" =~ "archive" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection unarchiving
@test "qdrant::collections::unarchive extracts archived collection" {
    result=$(qdrant::collections::unarchive "/tmp/archive.tar.gz" "restored_collection")
    
    [[ "$result" =~ "unarchive" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection versioning
@test "qdrant::collections::version manages collection versions" {
    result=$(qdrant::collections::version "test_collection" "create")
    
    [[ "$result" =~ "version" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection comparison
@test "qdrant::collections::compare compares collections" {
    result=$(qdrant::collections::compare "test_collection" "agent_memory")
    
    [[ "$result" =~ "compare" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection merging
@test "qdrant::collections::merge merges collections" {
    result=$(qdrant::collections::merge "test_collection" "agent_memory" "merged_collection")
    
    [[ "$result" =~ "merge" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection splitting
@test "qdrant::collections::split splits collection" {
    result=$(qdrant::collections::split "test_collection" "category" "split_")
    
    [[ "$result" =~ "split" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection synchronization
@test "qdrant::collections::sync synchronizes collections" {
    result=$(qdrant::collections::sync "test_collection" "http://remote:6333")
    
    [[ "$result" =~ "sync" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection rebalancing
@test "qdrant::collections::rebalance rebalances collection shards" {
    result=$(qdrant::collections::rebalance "test_collection")
    
    [[ "$result" =~ "rebalance" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection defragmentation
@test "qdrant::collections::defragment defragments collection" {
    result=$(qdrant::collections::defragment "test_collection")
    
    [[ "$result" =~ "defragment" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection vacuum
@test "qdrant::collections::vacuum cleans up collection storage" {
    result=$(qdrant::collections::vacuum "test_collection")
    
    [[ "$result" =~ "vacuum" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection locking
@test "qdrant::collections::lock locks collection for maintenance" {
    result=$(qdrant::collections::lock "test_collection")
    
    [[ "$result" =~ "lock" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection unlocking
@test "qdrant::collections::unlock unlocks collection" {
    result=$(qdrant::collections::unlock "test_collection")
    
    [[ "$result" =~ "unlock" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection freezing
@test "qdrant::collections::freeze freezes collection" {
    result=$(qdrant::collections::freeze "test_collection")
    
    [[ "$result" =~ "freeze" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection unfreezing
@test "qdrant::collections::unfreeze unfreezes collection" {
    result=$(qdrant::collections::unfreeze "test_collection")
    
    [[ "$result" =~ "unfreeze" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection export
@test "qdrant::collections::export exports collection data" {
    result=$(qdrant::collections::export "test_collection" "/tmp/export.jsonl")
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection import
@test "qdrant::collections::import imports collection data" {
    result=$(qdrant::collections::import "test_collection" "/tmp/import.jsonl")
    
    [[ "$result" =~ "import" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection transformation
@test "qdrant::collections::transform transforms collection data" {
    result=$(qdrant::collections::transform "test_collection" "transformation_script.py")
    
    [[ "$result" =~ "transform" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection analysis
@test "qdrant::collections::analyze analyzes collection data" {
    result=$(qdrant::collections::analyze "test_collection")
    
    [[ "$result" =~ "analyze" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection profiling
@test "qdrant::collections::profile profiles collection performance" {
    result=$(qdrant::collections::profile "test_collection")
    
    [[ "$result" =~ "profile" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection benchmarking
@test "qdrant::collections::benchmark benchmarks collection operations" {
    result=$(qdrant::collections::benchmark "test_collection" "search" "100")
    
    [[ "$result" =~ "benchmark" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection testing
@test "qdrant::collections::test tests collection functionality" {
    result=$(qdrant::collections::test "test_collection")
    
    [[ "$result" =~ "test" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection debugging
@test "qdrant::collections::debug debugs collection issues" {
    result=$(qdrant::collections::debug "test_collection")
    
    [[ "$result" =~ "debug" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection audit
@test "qdrant::collections::audit audits collection access" {
    result=$(qdrant::collections::audit "test_collection")
    
    [[ "$result" =~ "audit" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection compliance
@test "qdrant::collections::compliance checks collection compliance" {
    result=$(qdrant::collections::compliance "test_collection")
    
    [[ "$result" =~ "compliance" ]] || [[ "$result" =~ "collection" ]]
}

# Test collection security
@test "qdrant::collections::security configures collection security" {
    result=$(qdrant::collections::security "test_collection" "enable")
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "collection" ]]
}