#!/bin/bash
# Qdrant setup for Smart File Photo Manager
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VROOLI_ROOT="$(cd "$SCENARIO_ROOT/../../.." && pwd)"

# Load environment variables
source "$VROOLI_ROOT/scripts/resources/lib/resource-helper.sh"

# Configuration
QDRANT_HOST="localhost"
QDRANT_PORT="6335"
VECTOR_SIZE=1536

# Get default qdrant port if available
if command -v "resources::get_default_port" &> /dev/null; then
    QDRANT_PORT=$(resources::get_default_port "qdrant" || echo "6335")
fi

QDRANT_URL="http://$QDRANT_HOST:$QDRANT_PORT"

log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_success() {
    echo "[SUCCESS] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Wait for Qdrant to be ready
wait_for_qdrant() {
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for Qdrant on port $QDRANT_PORT..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$QDRANT_URL/health" | grep -q "ok" 2>/dev/null; then
            log_success "Qdrant is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "Qdrant failed to become ready"
    return 1
}

# Create vector collection
create_collection() {
    local collection_name="$1"
    local description="$2"
    
    log_info "Creating collection: $collection_name"
    
    # Check if collection exists
    if curl -s "$QDRANT_URL/collections/$collection_name" | grep -q "\"status\":\"green\"" 2>/dev/null; then
        log_info "Collection '$collection_name' already exists"
        return 0
    fi
    
    # Create collection
    local payload='{
        "vectors": {
            "size": '$VECTOR_SIZE',
            "distance": "Cosine"
        },
        "optimizers_config": {
            "default_segment_number": 2
        },
        "replication_factor": 1
    }'
    
    if curl -s -X PUT "$QDRANT_URL/collections/$collection_name" \
        -H "Content-Type: application/json" \
        -d "$payload" | grep -q "\"status\":\"ok\""; then
        log_success "Created collection: $collection_name"
    else
        log_error "Failed to create collection: $collection_name"
        return 1
    fi
}

# Create index for better performance
create_index() {
    local collection_name="$1"
    local field_name="$2"
    
    log_info "Creating index on $collection_name.$field_name"
    
    local payload='{
        "field_name": "'$field_name'",
        "field_schema": "keyword"
    }'
    
    if curl -s -X PUT "$QDRANT_URL/collections/$collection_name/index" \
        -H "Content-Type: application/json" \
        -d "$payload" | grep -q "\"status\":\"ok\""; then
        log_success "Created index on $collection_name.$field_name"
    else
        log_info "Index creation failed or already exists for $collection_name.$field_name"
    fi
}

# Setup all collections
setup_collections() {
    log_info "Setting up Qdrant collections..."
    
    # File embeddings collection - for general file content
    create_collection "file_embeddings" "General file content embeddings for semantic search"
    create_index "file_embeddings" "file_id"
    create_index "file_embeddings" "file_type"
    create_index "file_embeddings" "folder_path"
    
    # Image embeddings collection - for visual content
    create_collection "image_embeddings" "Visual embeddings for image similarity and search"
    create_index "image_embeddings" "file_id"
    create_index "image_embeddings" "detected_objects"
    
    # Content chunks collection - for document chunks
    create_collection "content_chunks" "Document chunk embeddings for detailed search"
    create_index "content_chunks" "file_id"
    create_index "content_chunks" "chunk_type"
    create_index "content_chunks" "page_number"
    
    log_success "All collections created successfully"
}

# Verify collections
verify_collections() {
    log_info "Verifying Qdrant collections..."
    
    local collections=("file_embeddings" "image_embeddings" "content_chunks")
    local all_good=true
    
    for collection in "${collections[@]}"; do
        if curl -s "$QDRANT_URL/collections/$collection" | grep -q "\"status\":\"green\""; then
            log_success "Collection '$collection' is healthy"
        else
            log_error "Collection '$collection' is not healthy"
            all_good=false
        fi
    done
    
    if [ "$all_good" = true ]; then
        log_success "All collections verified successfully"
    else
        log_error "Some collections failed verification"
        return 1
    fi
}

# Load initial collection configuration if exists
load_collection_config() {
    local config_file="$SCENARIO_ROOT/initialization/storage/qdrant/collections.json"
    
    if [ -f "$config_file" ]; then
        log_info "Loading collection configuration from $config_file"
        # This would process custom collection configs if needed
        # For now, we use the standard setup above
        log_info "Using standard collection configuration"
    fi
}

# Test basic operations
test_operations() {
    log_info "Testing basic Qdrant operations..."
    
    # Test adding a point
    local test_payload='{
        "points": [{
            "id": 1,
            "vector": [0.1, 0.2, 0.3, 0.4],
            "payload": {"test": "data"}
        }]
    }'
    
    # We'll use a 4-dimensional vector for testing, then delete it
    local test_collection="test_collection"
    
    # Create test collection
    curl -s -X PUT "$QDRANT_URL/collections/$test_collection" \
        -H "Content-Type: application/json" \
        -d '{"vectors": {"size": 4, "distance": "Cosine"}}' > /dev/null
    
    # Add test point
    if curl -s -X PUT "$QDRANT_URL/collections/$test_collection/points" \
        -H "Content-Type: application/json" \
        -d "$test_payload" | grep -q "\"status\":\"ok\""; then
        log_success "Basic operations test passed"
    else
        log_error "Basic operations test failed"
        return 1
    fi
    
    # Clean up test collection
    curl -s -X DELETE "$QDRANT_URL/collections/$test_collection" > /dev/null
}

# Main execution
main() {
    log_info "Setting up Qdrant vector database for File Manager..."
    
    # Wait for Qdrant to be ready
    wait_for_qdrant
    
    # Load collection configuration
    load_collection_config
    
    # Setup collections
    setup_collections
    
    # Verify collections
    verify_collections
    
    # Test basic operations
    test_operations
    
    log_success "Qdrant setup completed successfully"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi