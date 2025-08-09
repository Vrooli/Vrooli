#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Personal Digital Twin - Qdrant Collections Setup
# Initializes Qdrant vector collections for semantic search
################################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
QDRANT_HOST="${QDRANT_HOST:-localhost}"
QDRANT_PORT="${QDRANT_PORT:-6333}"
QDRANT_API_URL="http://${QDRANT_HOST}:${QDRANT_PORT}"

log::info() { echo "[INFO] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::success() { echo "[SUCCESS] $*"; }

# Wait for Qdrant to be ready
log::info "Waiting for Qdrant to be ready..."
until curl -sf "${QDRANT_API_URL}/health" >/dev/null 2>&1; do
    echo "Waiting for Qdrant..."
    sleep 2
done

log::success "Qdrant is ready!"

# Function to create collection if it doesn't exist
create_collection() {
    local collection_name="$1"
    local vector_size="$2"
    local distance="$3"
    
    log::info "Creating collection: $collection_name"
    
    # Check if collection exists
    if curl -sf "${QDRANT_API_URL}/collections/${collection_name}" >/dev/null 2>&1; then
        log::info "Collection $collection_name already exists, skipping..."
        return 0
    fi
    
    # Create collection
    curl -X PUT "${QDRANT_API_URL}/collections/${collection_name}" \
        -H "Content-Type: application/json" \
        -d "{
            \"vectors\": {
                \"size\": ${vector_size},
                \"distance\": \"${distance}\"
            },
            \"optimizers_config\": {
                \"memmap_threshold\": 20000,
                \"indexing_threshold\": 10000
            }
        }"
    
    if [[ $? -eq 0 ]]; then
        log::success "Collection $collection_name created successfully"
    else
        log::error "Failed to create collection $collection_name"
        return 1
    fi
}

# Create collections based on configuration
log::info "Setting up Qdrant collections..."

# Memories collection (semantic memories with importance scoring)
create_collection "memories" 768 "Cosine"

# Documents collection (chunked document content)
create_collection "documents" 768 "Cosine"

# Conversations collection (chat history for context)
create_collection "conversations" 768 "Cosine"

# Set up payload schemas for better indexing
log::info "Configuring payload schemas..."

# Memories collection payload schema
curl -X PUT "${QDRANT_API_URL}/collections/memories" \
    -H "Content-Type: application/json" \
    -d '{
        "vectors": {
            "size": 768,
            "distance": "Cosine"
        },
        "payload_schema": {
            "persona_id": {"type": "keyword", "indexed": true},
            "source_type": {"type": "keyword", "indexed": true},
            "timestamp": {"type": "datetime", "indexed": true},
            "importance_score": {"type": "float", "indexed": true},
            "text": {"type": "text"},
            "metadata": {"type": "json"}
        }
    }'

# Documents collection payload schema  
curl -X PUT "${QDRANT_API_URL}/collections/documents" \
    -H "Content-Type: application/json" \
    -d '{
        "vectors": {
            "size": 768,
            "distance": "Cosine"
        },
        "payload_schema": {
            "persona_id": {"type": "keyword", "indexed": true},
            "document_id": {"type": "keyword", "indexed": true},
            "chunk_index": {"type": "integer", "indexed": true},
            "content": {"type": "text"},
            "file_type": {"type": "keyword", "indexed": true}
        }
    }'

# Conversations collection payload schema
curl -X PUT "${QDRANT_API_URL}/collections/conversations" \
    -H "Content-Type: application/json" \
    -d '{
        "vectors": {
            "size": 768,
            "distance": "Cosine"
        },
        "payload_schema": {
            "persona_id": {"type": "keyword", "indexed": true},
            "conversation_id": {"type": "keyword", "indexed": true},
            "role": {"type": "keyword", "indexed": true},
            "message": {"type": "text"},
            "timestamp": {"type": "datetime", "indexed": true}
        }
    }'

log::success "Qdrant collections setup complete!"