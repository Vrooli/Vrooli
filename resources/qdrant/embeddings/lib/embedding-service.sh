#!/usr/bin/env bash
# Unified Embedding Service - Eliminates duplication across extractors
# Provides standardized embedding pipeline for all content types

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EMBEDDING_SERVICE_DIR="${EMBEDDINGS_DIR}/lib"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source embedding and collection libraries
source "${EMBEDDINGS_DIR}/../lib/embeddings.sh"
source "${EMBEDDINGS_DIR}/../lib/collections.sh"

# Default configuration
DEFAULT_MODEL="${QDRANT_DEFAULT_MODEL:-mxbai-embed-large}"
DEFAULT_BATCH_SIZE="${QDRANT_EMBEDDING_BATCH_SIZE:-50}"

#######################################
# Process content item through complete embedding pipeline
# Arguments:
#   $1 - Content text
#   $2 - Content type (workflow, scenario, documentation, code, resource)
#   $3 - Collection name
#   $4 - App ID  
#   $5 - Additional metadata (JSON, optional)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embedding::process_item() {
    local content="$1"
    local content_type="$2"
    local collection="$3"
    local app_id="$4"
    local extra_metadata="${5:-{}}"
    
    # Validation
    if [[ -z "$content" || -z "$content_type" || -z "$collection" || -z "$app_id" ]]; then
        log::error "Missing required parameters for embedding processing"
        return 1
    fi
    
    # Generate embedding
    local embedding
    embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL" 2>/dev/null)
    if [[ -z "$embedding" ]]; then
        log::warn "Failed to generate embedding for $content_type content"
        return 1
    fi
    
    # Create deterministic ID
    local item_id
    item_id=$(qdrant::embedding::generate_id "$content" "$content_type")
    
    # Build payload with standard fields + extras
    local payload
    payload=$(qdrant::embedding::build_payload "$content" "$content_type" "$app_id" "$extra_metadata")
    
    # Store in collection
    if qdrant::collections::upsert_point "$collection" "$item_id" "$embedding" "$payload"; then
        log::debug "Successfully processed $content_type item (ID: ${item_id:0:8}...)"
        return 0
    else
        log::error "Failed to store $content_type embedding in collection $collection"
        return 1
    fi
}

#######################################
# Generate deterministic ID for content
# Arguments:
#   $1 - Content text
#   $2 - Content type
# Returns: SHA256 hash as unique identifier
#######################################
qdrant::embedding::generate_id() {
    local content="$1"
    local content_type="$2"
    
    # Include content type in hash for uniqueness across types
    echo -n "${content_type}:${content}" | sha256sum | cut -d' ' -f1
}

#######################################
# Build standardized payload JSON
# Arguments:
#   $1 - Content text
#   $2 - Content type
#   $3 - App ID
#   $4 - Extra metadata (JSON, optional)
# Returns: Complete payload JSON
#######################################
qdrant::embedding::build_payload() {
    local content="$1"
    local content_type="$2" 
    local app_id="$3"
    local extra_metadata="${4:-{}}"
    
    # Base payload with standard fields
    local base_payload
    base_payload=$(jq -n \
        --arg content "$content" \
        --arg type "$content_type" \
        --arg app_id "$app_id" \
        --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg length "$(echo -n "$content" | wc -c)" \
        '{
            content: $content,
            type: $type,
            app_id: $app_id,
            indexed_at: $timestamp,
            content_length: ($length | tonumber)
        }')
    
    # Merge with extra metadata if provided
    if [[ "$extra_metadata" != "{}" ]] && [[ -n "$extra_metadata" ]]; then
        # Validate that extra_metadata is valid JSON before using it
        if echo "$extra_metadata" | jq . >/dev/null 2>&1; then
            echo "$base_payload" | jq --argjson extra "$extra_metadata" '. + $extra'
        else
            log::warn "Invalid JSON in extra_metadata, using base payload only"
            echo "$base_payload"
        fi
    else
        echo "$base_payload"
    fi
}

#######################################
# Batch processing with automatic optimization
# Arguments:
#   $1 - Content array (JSON array of content items)
#   $2 - Content type
#   $3 - Collection name
#   $4 - App ID
# Returns: Number of items processed
#######################################
qdrant::embedding::process_batch() {
    local content_array="$1"
    local content_type="$2"
    local collection="$3"
    local app_id="$4"
    
    if [[ -z "$content_array" || "$content_array" == "[]" ]]; then
        echo "0"
        return 0
    fi
    
    local processed_count=0
    local batch_points="[]"
    local batch_size="$DEFAULT_BATCH_SIZE"
    
    # Process each item
    echo "$content_array" | jq -c '.[]' | while read -r content_item; do
        local content
        content=$(echo "$content_item" | jq -r '.content // empty')
        if [[ -z "$content" ]]; then
            continue
        fi
        
        local extra_metadata
        extra_metadata=$(echo "$content_item" | jq -r '.metadata // {}')
        
        # Generate embedding and build point
        local embedding
        embedding=$(qdrant::embeddings::generate "$content" "$DEFAULT_MODEL" 2>/dev/null)
        
        if [[ -n "$embedding" ]]; then
            local item_id
            item_id=$(qdrant::embedding::generate_id "$content" "$content_type")
            
            local payload
            payload=$(qdrant::embedding::build_payload "$content" "$content_type" "$app_id" "$extra_metadata")
            
            # Add to batch
            local point
            point=$(jq -n \
                --arg id "$item_id" \
                --argjson vector "$embedding" \
                --argjson payload "$payload" \
                '{
                    id: $id,
                    vector: $vector,
                    payload: $payload
                }')
            
            batch_points=$(echo "$batch_points" | jq ". += [$point]")
            ((processed_count++))
            
            # Flush batch when threshold reached
            if [[ $((processed_count % batch_size)) -eq 0 ]]; then
                if qdrant::collections::batch_upsert "$collection" "$batch_points"; then
                    log::debug "Batch upserted $batch_size items to $collection"
                    batch_points="[]"
                else
                    log::error "Failed to batch upsert to $collection"
                fi
            fi
        fi
    done
    
    # Flush remaining items
    local remaining_count
    remaining_count=$(echo "$batch_points" | jq 'length')
    if [[ "$remaining_count" -gt 0 ]]; then
        if qdrant::collections::batch_upsert "$collection" "$batch_points"; then
            log::debug "Final batch upserted $remaining_count items to $collection"
        else
            log::error "Failed to upsert final batch to $collection"
        fi
    fi
    
    echo "$processed_count"
}

#######################################
# Process content with retry logic for robustness
# Arguments:
#   $1 - Content text
#   $2 - Content type
#   $3 - Collection name
#   $4 - App ID
#   $5 - Extra metadata (JSON, optional)
#   $6 - Max retries (optional, default: 2)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::embedding::process_item_with_retry() {
    local content="$1"
    local content_type="$2"
    local collection="$3"
    local app_id="$4"
    local extra_metadata="${5:-{}}"
    local max_retries="${6:-2}"
    
    local attempt=0
    while [[ $attempt -le $max_retries ]]; do
        if qdrant::embedding::process_item "$content" "$content_type" "$collection" "$app_id" "$extra_metadata"; then
            return 0
        fi
        
        ((attempt++))
        if [[ $attempt -le $max_retries ]]; then
            log::debug "Retry attempt $attempt for $content_type embedding"
            sleep 1
        fi
    done
    
    log::error "Failed to process $content_type item after $max_retries retries"
    return 1
}

#######################################
# Validate embedding service health
# Returns: 0 if healthy, 1 if issues detected
#######################################
qdrant::embedding::health_check() {
    log::info "Performing embedding service health check..."
    
    # Check if embedding generation works
    local test_embedding
    test_embedding=$(qdrant::embeddings::generate "test content" "$DEFAULT_MODEL" 2>/dev/null)
    if [[ -z "$test_embedding" ]]; then
        log::error "Embedding generation test failed"
        return 1
    fi
    
    # Check if ID generation works
    local test_id
    test_id=$(qdrant::embedding::generate_id "test content" "test")
    if [[ -z "$test_id" ]]; then
        log::error "ID generation test failed"
        return 1
    fi
    
    # Check if payload building works
    local test_payload
    test_payload=$(qdrant::embedding::build_payload "test content" "test" "test-app" "{}")
    if [[ -z "$test_payload" ]]; then
        log::error "Payload building test failed"
        return 1
    fi
    
    log::success "Embedding service health check passed"
    return 0
}

# Export functions for use by extractors
export -f qdrant::embedding::process_item
export -f qdrant::embedding::process_item_with_retry
export -f qdrant::embedding::process_batch
export -f qdrant::embedding::generate_id
export -f qdrant::embedding::build_payload
export -f qdrant::embedding::health_check