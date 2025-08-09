#!/bin/bash

# Task Planner - Qdrant Vector Database Setup Script
# Creates collections and configures vector search for semantic similarity

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"; }
success() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $*${NC}"; }
warn() { echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $*${NC}"; }
error() { echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $*${NC}" >&2; }

# Qdrant configuration
QDRANT_HOST="${QDRANT_HOST:-localhost}"
QDRANT_PORT="${QDRANT_PORT:-6334}"
QDRANT_API_KEY="${QDRANT_API_KEY:-}"
QDRANT_BASE_URL="http://${QDRANT_HOST}:${QDRANT_PORT}"

# Configuration file
CONFIG_FILE="${SCENARIO_DIR}/initialization/storage/qdrant/collections.json"

# Helper function to make API calls to Qdrant
qdrant_api() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    
    local curl_args=("-s" "-X" "$method")
    
    # Add API key header if provided
    if [[ -n "${QDRANT_API_KEY}" ]]; then
        curl_args+=("-H" "api-key: ${QDRANT_API_KEY}")
    fi
    
    curl_args+=("-H" "Content-Type: application/json")
    
    if [[ -n "$data" ]]; then
        curl_args+=("-d" "$data")
    fi
    
    curl_args+=("${QDRANT_BASE_URL}${endpoint}")
    
    curl "${curl_args[@]}"
}

# Wait for Qdrant to be ready
wait_for_qdrant() {
    log "Waiting for Qdrant to be ready..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if qdrant_api "GET" "/health" > /dev/null 2>&1; then
            success "Qdrant is ready"
            return 0
        fi
        
        warn "Waiting for Qdrant... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    error "Qdrant did not become ready within expected time"
    return 1
}

# Check if collection exists
collection_exists() {
    local collection_name="$1"
    
    local response
    response=$(qdrant_api "GET" "/collections/${collection_name}" 2>/dev/null) || return 1
    
    # Check if response contains error
    if echo "$response" | jq -e '.status.error' > /dev/null 2>&1; then
        return 1
    fi
    
    return 0
}

# Create a collection
create_collection() {
    local collection_name="$1"
    local vector_size="$2"
    local distance="${3:-Cosine}"
    
    log "Creating collection: $collection_name"
    
    if collection_exists "$collection_name"; then
        warn "Collection $collection_name already exists, skipping creation"
        return 0
    fi
    
    local collection_config
    collection_config=$(cat << EOF
{
  "vectors": {
    "size": $vector_size,
    "distance": "$distance",
    "on_disk": true
  },
  "shard_number": 1,
  "replication_factor": 1,
  "write_consistency_factor": 1,
  "on_disk_payload": true,
  "hnsw_config": {
    "m": 16,
    "ef_construct": 100,
    "full_scan_threshold": 10000,
    "max_indexing_threads": 4,
    "on_disk": false,
    "payload_m": 16
  },
  "optimizer_config": {
    "deleted_threshold": 0.2,
    "vacuum_min_vector_number": 1000,
    "default_segment_number": 2,
    "max_segment_size": 200000,
    "memmap_threshold": 50000,
    "indexing_threshold": 20000,
    "flush_interval_sec": 30,
    "max_optimization_threads": 2
  },
  "wal_config": {
    "wal_capacity_mb": 32,
    "wal_segments_ahead": 2
  }
}
EOF
    )
    
    local response
    response=$(qdrant_api "PUT" "/collections/${collection_name}" "$collection_config")
    
    if echo "$response" | jq -e '.status.error' > /dev/null 2>&1; then
        error "Failed to create collection $collection_name: $(echo "$response" | jq -r '.status.error')"
        return 1
    fi
    
    success "Collection $collection_name created successfully"
}

# Create indexes for a collection
create_indexes() {
    local collection_name="$1"
    shift
    local fields=("$@")
    
    log "Creating payload indexes for $collection_name"
    
    for field in "${fields[@]}"; do
        # Split field definition (format: "field_name:field_type")
        IFS=':' read -r field_name field_type <<< "$field"
        
        local index_config
        case "$field_type" in
            "keyword")
                index_config='{"field_name": "'$field_name'", "field_schema": "keyword"}'
                ;;
            "integer")
                index_config='{"field_name": "'$field_name'", "field_schema": "integer"}'
                ;;
            "float")
                index_config='{"field_name": "'$field_name'", "field_schema": "float"}'
                ;;
            "datetime")
                index_config='{"field_name": "'$field_name'", "field_schema": "datetime"}'
                ;;
            "text")
                index_config='{"field_name": "'$field_name'", "field_schema": "text", "field_params": {"tokenizer": "word", "min_token_len": 2, "max_token_len": 20}}'
                ;;
            *)
                warn "Unknown field type $field_type for field $field_name, skipping"
                continue
                ;;
        esac
        
        local response
        response=$(qdrant_api "PUT" "/collections/${collection_name}/index" "$index_config")
        
        if echo "$response" | jq -e '.status.error' > /dev/null 2>&1; then
            warn "Failed to create index for $field_name: $(echo "$response" | jq -r '.status.error')"
        else
            log "Created index for field: $field_name ($field_type)"
        fi
    done
}

# Create collection aliases
create_aliases() {
    local collection_name="$1"
    shift
    local aliases=("$@")
    
    for alias in "${aliases[@]}"; do
        local alias_config='{"actions": [{"create_alias": {"collection_name": "'$collection_name'", "alias_name": "'$alias'"}}]}'
        
        local response
        response=$(qdrant_api "POST" "/collections/aliases" "$alias_config")
        
        if echo "$response" | jq -e '.status.error' > /dev/null 2>&1; then
            warn "Failed to create alias $alias for $collection_name: $(echo "$response" | jq -r '.status.error')"
        else
            log "Created alias: $alias -> $collection_name"
        fi
    done
}

# Setup task embeddings collection
setup_task_embeddings() {
    log "Setting up task_embeddings collection..."
    
    create_collection "task_embeddings" 1536 "Cosine"
    
    # Create indexes for task fields
    create_indexes "task_embeddings" \
        "task_id:keyword" \
        "app_id:keyword" \
        "status:keyword" \
        "priority:keyword" \
        "tags:keyword" \
        "created_at:datetime" \
        "updated_at:datetime" \
        "embedding_model:keyword"
    
    # Create aliases
    create_aliases "task_embeddings" "tasks" "task_vectors"
    
    success "task_embeddings collection setup complete"
}

# Setup research embeddings collection  
setup_research_embeddings() {
    log "Setting up research_embeddings collection..."
    
    create_collection "research_embeddings" 1536 "Cosine"
    
    # Create indexes for research artifact fields
    create_indexes "research_embeddings" \
        "artifact_id:keyword" \
        "task_id:keyword" \
        "app_id:keyword" \
        "type:keyword" \
        "relevance_score:float" \
        "quality_score:float" \
        "domain:keyword" \
        "language:keyword" \
        "created_at:datetime"
    
    # Create aliases
    create_aliases "research_embeddings" "research" "artifacts"
    
    success "research_embeddings collection setup complete"
}

# Verify collections are working
verify_collections() {
    log "Verifying collections setup..."
    
    local collections=("task_embeddings" "research_embeddings")
    
    for collection in "${collections[@]}"; do
        local response
        response=$(qdrant_api "GET" "/collections/${collection}")
        
        if echo "$response" | jq -e '.status.error' > /dev/null 2>&1; then
            error "Collection $collection verification failed: $(echo "$response" | jq -r '.status.error')"
            return 1
        fi
        
        local vector_count
        vector_count=$(echo "$response" | jq -r '.result.vectors_count // 0')
        
        local points_count
        points_count=$(echo "$response" | jq -r '.result.points_count // 0')
        
        success "Collection $collection: $points_count points, $vector_count vectors"
    done
}

# Insert sample vectors for testing (optional)
insert_sample_data() {
    if [[ "${LOAD_SAMPLE_DATA:-false}" != "true" ]]; then
        log "Skipping sample data insertion (set LOAD_SAMPLE_DATA=true to enable)"
        return 0
    fi
    
    log "Inserting sample vectors for testing..."
    
    # Sample task vector
    local sample_task_vector='[0.1, -0.2, 0.3, 0.15, -0.05'$(printf ", 0.0%.0s" {1..1531})']'
    local task_payload='{
        "task_id": "sample-task-1",
        "app_id": "sample-app",
        "title": "Sample Task",
        "description": "This is a sample task for testing",
        "status": "backlog",
        "priority": "medium",
        "tags": ["sample", "test"],
        "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
        "embedding_model": "nomic-embed-text",
        "embedding_version": 1
    }'
    
    local task_point='{
        "points": [{
            "id": 1,
            "vector": '$sample_task_vector',
            "payload": '$task_payload'
        }]
    }'
    
    qdrant_api "PUT" "/collections/task_embeddings/points" "$task_point" > /dev/null
    
    # Sample research vector
    local sample_research_vector='[-0.1, 0.2, -0.3, 0.25, 0.05'$(printf ", 0.0%.0s" {1..1531})']'
    local research_payload='{
        "artifact_id": "sample-artifact-1",
        "task_id": "sample-task-1", 
        "app_id": "sample-app",
        "type": "documentation",
        "title": "Sample Documentation",
        "content": "This is sample documentation for testing",
        "relevance_score": 0.85,
        "quality_score": 0.90,
        "domain": "example.com",
        "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
        "embedding_model": "nomic-embed-text"
    }'
    
    local research_point='{
        "points": [{
            "id": 1,
            "vector": '$sample_research_vector',
            "payload": '$research_payload'
        }]
    }'
    
    qdrant_api "PUT" "/collections/research_embeddings/points" "$research_point" > /dev/null
    
    success "Sample vectors inserted for testing"
}

# Display connection information
show_connection_info() {
    log "Qdrant Vector Database Information:"
    echo ""
    echo "🚀 API Endpoint: $QDRANT_BASE_URL"
    echo "🔍 Health Check: $QDRANT_BASE_URL/health"
    echo "📊 Dashboard: $QDRANT_BASE_URL/dashboard"
    echo ""
    echo "📚 Collections Created:"
    echo "  • task_embeddings (aliases: tasks, task_vectors)"
    echo "  • research_embeddings (aliases: research, artifacts)"
    echo ""
    echo "🔧 Management Commands:"
    echo "  curl $QDRANT_BASE_URL/collections"
    echo "  curl $QDRANT_BASE_URL/collections/task_embeddings"
    echo ""
    echo "🔍 Search Examples:"
    echo "  # Find similar tasks"
    echo "  curl -X POST $QDRANT_BASE_URL/collections/task_embeddings/points/search \\"
    echo "    -H 'Content-Type: application/json' \\"
    echo "    -d '{\"vector\": [0.1, 0.2, ...], \"limit\": 5}'"
    echo ""
}

# Main execution
main() {
    log "Setting up Qdrant for Task Planner..."
    
    # Check if jq is available for JSON processing
    if ! command -v jq &> /dev/null; then
        error "jq is required for Qdrant setup. Please install it first."
        return 1
    fi
    
    # Setup steps
    wait_for_qdrant
    setup_task_embeddings
    setup_research_embeddings
    verify_collections
    insert_sample_data
    show_connection_info
    
    success "Qdrant setup completed successfully!"
    
    # Export connection details for other scripts
    cat > "${SCRIPT_DIR}/.qdrant_connection" << EOF
export QDRANT_HOST="$QDRANT_HOST"
export QDRANT_PORT="$QDRANT_PORT"
export QDRANT_BASE_URL="$QDRANT_BASE_URL"
export QDRANT_API_KEY="$QDRANT_API_KEY"
EOF
    
    success "Connection details saved to ${SCRIPT_DIR}/.qdrant_connection"
}

# Execute main function
main "$@"