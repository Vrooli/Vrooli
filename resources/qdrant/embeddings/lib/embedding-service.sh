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

# Source unified configuration first
source "${EMBEDDINGS_DIR}/config/unified.sh"

# Source embedding and collection libraries
source "${APP_ROOT}/resources/qdrant/lib/embeddings.sh"
source "${APP_ROOT}/resources/qdrant/lib/collections.sh"
source "${APP_ROOT}/resources/qdrant/lib/http-client-improved.sh"
source "${APP_ROOT}/resources/qdrant/lib/error-handler.sh"

# Configuration loaded from unified.sh - backward compatible aliases available

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
    
    # Generate embedding with error handling
    local embedding
    if ! embedding=$(error_handler::execute_with_retry "embedding" "generate embedding for $content_type" \
        "qdrant::embeddings::generate '$content' '$DEFAULT_MODEL' 2>/dev/null | jq -c ." \
        '\[.*\]'); then
        error_handler::log_error "embedding" "Failed to generate embedding" "$content_type" "{\"content_length\": ${#content}}"
        return 1
    fi
    
    # Create deterministic ID
    local item_id
    item_id=$(qdrant::embedding::generate_id "$content" "$content_type")
    
    # Build payload with standard fields + extras
    local payload
    payload=$(qdrant::embedding::build_payload "$content" "$content_type" "$app_id" "$extra_metadata")
    
    # Store in collection using direct API with proper JSON construction
    local points_data
    points_data=$(jq -n \
        --arg id "$item_id" \
        --argjson vector "$embedding" \
        --argjson payload "$payload" \
        '{
            points: [
                {
                    id: $id,
                    vector: $vector,
                    payload: $payload
                }
            ]
        }')
    
    # Store in Qdrant with retry logic
    local api_response
    local status
    local max_retries=3
    local retry_delay=1
    local attempt=1
    
    while [[ $attempt -le $max_retries ]]; do
        log::debug "Qdrant storage attempt $attempt/$max_retries for $content_type"
        
        # Use improved HTTP client with retry logic and connection pooling
        api_response=$(http_client::put "/collections/${collection}/points" "$points_data" "store embedding")
        
        if [[ -n "$api_response" ]]; then
            status=$(echo "$api_response" | jq -r '.status // "unknown"' 2>/dev/null)
            
            if [[ "$status" == "ok" ]]; then
                # Verify the point was actually stored
                local verify_response=$(curl -s -X POST "http://localhost:6333/collections/${collection}/points/count" \
                    -H "Content-Type: application/json" -d '{"exact": false}' 2>/dev/null)
                local current_count=$(echo "$verify_response" | jq -r '.result.count // 0' 2>/dev/null)
                
                if [[ "$current_count" -gt 0 ]]; then
                    log::debug "Successfully stored $content_type item (ID: ${item_id:0:8}...) - Collection now has $current_count vectors"
                    return 0
                else
                    log::warn "Qdrant reported success but collection still empty - possible ID format issue"
                fi
            else
                # Extract and log the actual error message
                local error_msg=$(echo "$api_response" | jq -r '.status.error // "No error message"' 2>/dev/null)
                log::warn "Qdrant storage failed: $error_msg"
                
                # Check if it's an ID format error and regenerate if needed
                if echo "$error_msg" | grep -q "not a valid point ID"; then
                    log::error "Invalid ID format detected - UUID generation may have failed"
                    return 1
                fi
            fi
        else
            log::warn "No response from Qdrant API - service may be down"
        fi
        
        if [[ $attempt -eq $max_retries ]]; then
            log::error "Failed to store $content_type embedding in collection $collection after $max_retries attempts"
            log::error "Last response: ${api_response:-No response}"
            return 1
        fi
        
        log::warn "Qdrant storage attempt $attempt failed, retrying in ${retry_delay}s..."
        sleep $retry_delay
        retry_delay=$((retry_delay * 2))  # Exponential backoff
        ((attempt++))
    done
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
    local hash=$(echo -n "${content_type}:${content}" | sha256sum | cut -d' ' -f1)
    
    # Generate a proper UUID v4 that's deterministic based on content
    # We use Python to ensure valid UUID format that Qdrant will accept
    # The UUID is seeded from the hash for determinism
    local uuid=$(python3 -c "
import uuid
import hashlib

# Use the first 16 bytes of the hash as seed for UUID
hash_bytes = bytes.fromhex('${hash}'[:32])
# Create a UUID from the hash bytes (namespace UUID method)
# Set version and variant bits to make it a valid UUID v4
uuid_int = int.from_bytes(hash_bytes, 'big')
# Ensure it fits in UUID range and set proper version/variant
uuid_obj = uuid.UUID(int=(uuid_int & 0xffffffffffff0fff3fffffffffffffff) | 0x0000000000004000c000000000000000)
print(str(uuid_obj))
" 2>/dev/null)
    
    # Fallback to numeric ID if Python fails
    if [[ -z "$uuid" ]]; then
        # Use first 16 hex chars as a large positive integer
        # This ensures uniqueness and Qdrant compatibility
        local numeric_id=$(echo "ibase=16; ${hash:0:16}" | bc 2>/dev/null || echo "0")
        echo "$numeric_id"
    else
        echo "$uuid"
    fi
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
    local extra_metadata
    if [[ -n "${4:-}" ]]; then
        extra_metadata="$4"
    else
        extra_metadata="{}"
    fi
    
    # Sanitize content to prevent JSON issues
    local safe_content
    safe_content=$(echo "$content" | tr '\n\r\t' ' ' | sed 's/"/\\"/g' | head -c 8000)
    
    # Base payload with standard fields
    local base_payload
    base_payload=$(jq -n \
        --arg content "$safe_content" \
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
    
    # Merge with extra metadata if provided - SIMPLIFIED & SAFE
    if [[ "$extra_metadata" != "{}" ]] && [[ -n "$extra_metadata" ]]; then
        # Only process if metadata is reasonably sized and valid
        local metadata_size=${#extra_metadata}
        if [[ $metadata_size -lt 2000 ]] && echo "$extra_metadata" | jq -e . >/dev/null 2>&1; then
            # Safe to merge - using jq -e for strict validation  
            local merged_payload
            if merged_payload=$(echo "$base_payload" | jq --argjson extra "$extra_metadata" '. + $extra' 2>/dev/null); then
                echo "$merged_payload"
            else
                log::debug "JSON merge failed, using base payload only"
                echo "$base_payload"
            fi
        else
            log::debug "Metadata invalid or too large (${metadata_size} chars), using base payload only"
            echo "$base_payload"
        fi
    else
        echo "$base_payload"
    fi
}

#######################################
# Batch processing with REAL batch embeddings
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
    
    local item_count
    item_count=$(echo "$content_array" | jq 'length')
    log::info "Processing batch of $item_count items for $content_type"
    
    local batch_size="${DEFAULT_BATCH_SIZE:-20}"
    local processed_count=0
    
    # Accumulate items for batch processing
    local text_batch="[]"
    local metadata_batch="[]"
    local batch_points="[]"
    
    # Process in chunks
    for ((i=0; i<item_count; i+=batch_size)); do
        local batch_end=$((i + batch_size))
        if [[ $batch_end -gt $item_count ]]; then
            batch_end=$item_count
        fi
        
        # Extract batch of items
        local batch_items
        batch_items=$(echo "$content_array" | jq ".[$i:$batch_end]")
        local current_batch_size
        current_batch_size=$(echo "$batch_items" | jq 'length')
        
        # Extract texts and metadata for this batch
        text_batch="[]"
        metadata_batch="[]"
        
        echo "$batch_items" | jq -c '.[]' | while IFS= read -r item; do
            local content
            content=$(echo "$item" | jq -r '.content // empty')
            if [[ -n "$content" ]]; then
                text_batch=$(echo "$text_batch" | jq --arg t "$content" '. + [$t]')
                local meta
                meta=$(echo "$item" | jq '.metadata // {}')
                metadata_batch=$(echo "$metadata_batch" | jq --argjson m "$meta" '. + [$m]')
            fi
        done
        
        # Generate embeddings for entire batch in ONE API call
        log::debug "Generating embeddings for batch of $current_batch_size items..."
        local embeddings
        embeddings=$(qdrant::embeddings::generate_batch "$text_batch" "$DEFAULT_MODEL")
        
        if [[ -n "$embeddings" ]] && [[ "$embeddings" != "[]" ]]; then
            # Build points for Qdrant
            batch_points="[]"
            
            for ((j=0; j<current_batch_size; j++)); do
                local text
                text=$(echo "$text_batch" | jq -r ".[$j]")
                local embedding
                embedding=$(echo "$embeddings" | jq ".[$j]")
                local extra_metadata
                extra_metadata=$(echo "$metadata_batch" | jq ".[$j]")
                
                if [[ -n "$embedding" ]] && [[ "$embedding" != "null" ]]; then
                    local item_id
                    item_id=$(qdrant::embedding::generate_id "$text" "$content_type")
                    
                    local payload
                    payload=$(qdrant::embedding::build_payload "$text" "$content_type" "$app_id" "$extra_metadata")
                    
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
                fi
            done
            
            # Upsert entire batch to Qdrant
            if [[ $(echo "$batch_points" | jq 'length') -gt 0 ]]; then
                if qdrant::collections::batch_upsert "$collection" "$batch_points"; then
                    log::debug "Upserted batch of $(echo "$batch_points" | jq 'length') items to $collection"
                else
                    log::error "Failed to batch upsert to $collection"
                fi
            fi
        else
            log::error "Failed to generate embeddings for batch"
        fi
    done
    
    log::success "Processed $processed_count items using real batch embeddings"
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

#######################################
# Process content from file using real batch embeddings
# Optimized for extractors to use directly
# Arguments:
#   $1 - JSONL file path with extracted content
#   $2 - Content type
#   $3 - Collection name
#   $4 - App ID
#   $5 - Batch size (optional, default: 50)
# Returns: Number of items processed
#######################################
qdrant::embedding::process_jsonl_file() {
    local jsonl_file="$1"
    local content_type="$2"
    local collection="$3"
    local app_id="$4"
    local batch_size="${5:-20}"  # Optimal batch size for performance
    
    if [[ ! -f "$jsonl_file" ]] || [[ ! -s "$jsonl_file" ]]; then
        log::debug "No content in file: $jsonl_file"
        echo "0"
        return 0
    fi
    
    local total_lines=$(wc -l < "$jsonl_file")
    log::info "Processing $total_lines items from $jsonl_file using optimized batch embeddings (batch_size=$batch_size)"
    
    local processed_count=0
    local batch_count=0
    
    # Use temporary files to avoid O(n²) JSON array building
    local temp_texts=$(mktemp)
    local temp_metadata=$(mktemp)
    trap "rm -f $temp_texts $temp_metadata" RETURN
    
    # Process file line by line, accumulating batches
    while IFS= read -r json_line; do
        if [[ -z "$json_line" ]]; then
            continue
        fi
        
        # Extract content and metadata
        local content
        content=$(echo "$json_line" | jq -r '.content // empty' 2>/dev/null)
        
        if [[ -n "$content" ]]; then
            # Store in temp files instead of building JSON arrays incrementally
            echo "$content" >> "$temp_texts"
            echo "$json_line" | jq -c '.metadata // {}' >> "$temp_metadata"
            ((batch_count++))
            
            # Process batch when it reaches the size limit
            if [[ $batch_count -ge $batch_size ]]; then
                local batch_num=$((processed_count / batch_size + 1))
                log::info "Processing batch $batch_num (items $((processed_count+1))-$((processed_count+batch_count)) of $total_lines)..."
                
                # Build text array from temp file
                local text_batch
                text_batch=$(jq -Rs 'split("\n") | map(select(. != ""))' "$temp_texts")
                
                # Generate embeddings for entire batch in ONE call
                local embeddings
                embeddings=$(qdrant::embeddings::generate_batch "$text_batch" "$DEFAULT_MODEL")
                
                if [[ -n "$embeddings" ]] && [[ "$embeddings" != "[]" ]]; then
                    # Build metadata array from temp file
                    local metadata_batch
                    metadata_batch=$(jq -s '.' "$temp_metadata")
                    
                    # Build points array efficiently using jq streaming
                    local points_json=$(mktemp)
                    echo "[]" > "$points_json"
                    
                    for ((j=0; j<batch_count; j++)); do
                        local text
                        text=$(sed -n "$((j+1))p" "$temp_texts")
                        local embedding
                        embedding=$(echo "$embeddings" | jq ".[$j]")
                        local meta
                        meta=$(echo "$metadata_batch" | jq ".[$j]")
                        
                        if [[ "$embedding" != "null" ]] && [[ -n "$text" ]]; then
                            local item_id
                            item_id=$(qdrant::embedding::generate_id "$text" "$content_type")
                            local payload
                            payload=$(qdrant::embedding::build_payload "$text" "$content_type" "$app_id" "$meta")
                            
                            # Append to points file efficiently
                            jq --arg id "$item_id" \
                                --argjson vector "$embedding" \
                                --argjson payload "$payload" \
                                '. += [{id: $id, vector: $vector, payload: $payload}]' \
                                "$points_json" > "${points_json}.tmp" && mv "${points_json}.tmp" "$points_json"
                            ((processed_count++))
                        fi
                    done
                    
                    local points=$(cat "$points_json")
                    rm -f "$points_json"
                    
                    # Store batch in Qdrant
                    if [[ $(echo "$points" | jq 'length') -gt 0 ]]; then
                        qdrant::collections::batch_upsert "$collection" "$points" || \
                            log::error "Failed to upsert batch to $collection"
                    fi
                fi
                
                # Reset for next batch
                > "$temp_texts"
                > "$temp_metadata"
                batch_count=0
            fi
        fi
    done < "$jsonl_file"
    
    # Process remaining items
    if [[ $batch_count -gt 0 ]]; then
        log::debug "Processing final batch of $batch_count items..."
        
        # Build text array from temp file
        local text_batch
        text_batch=$(jq -Rs 'split("\n") | map(select(. != ""))' "$temp_texts")
        
        local embeddings
        embeddings=$(qdrant::embeddings::generate_batch "$text_batch" "$DEFAULT_MODEL")
        
        if [[ -n "$embeddings" ]] && [[ "$embeddings" != "[]" ]]; then
            # Build metadata array from temp file
            local metadata_batch
            metadata_batch=$(jq -s '.' "$temp_metadata")
            
            # Build points array efficiently
            local points_json=$(mktemp)
            echo "[]" > "$points_json"
            
            for ((j=0; j<batch_count; j++)); do
                local text
                text=$(sed -n "$((j+1))p" "$temp_texts")
                local embedding
                embedding=$(echo "$embeddings" | jq ".[$j]")
                local meta
                meta=$(echo "$metadata_batch" | jq ".[$j]")
                
                if [[ "$embedding" != "null" ]] && [[ -n "$text" ]]; then
                    local item_id
                    item_id=$(qdrant::embedding::generate_id "$text" "$content_type")
                    local payload
                    payload=$(qdrant::embedding::build_payload "$text" "$content_type" "$app_id" "$meta")
                    
                    # Append to points file efficiently
                    jq --arg id "$item_id" \
                        --argjson vector "$embedding" \
                        --argjson payload "$payload" \
                        '. += [{id: $id, vector: $vector, payload: $payload}]' \
                        "$points_json" > "${points_json}.tmp" && mv "${points_json}.tmp" "$points_json"
                    ((processed_count++))
                fi
            done
            
            local points=$(cat "$points_json")
            rm -f "$points_json"
            
            if [[ $(echo "$points" | jq 'length') -gt 0 ]]; then
                qdrant::collections::batch_upsert "$collection" "$points" || \
                    log::error "Failed to upsert final batch to $collection"
            fi
        fi
    fi
    
    log::success "Processed $processed_count/$total_lines items using real batch embeddings"
    echo "$processed_count"
}

# Export functions for use by extractors
export -f qdrant::embedding::process_item
export -f qdrant::embedding::process_item_with_retry
export -f qdrant::embedding::process_batch
export -f qdrant::embedding::process_jsonl_file
export -f qdrant::embedding::generate_id
export -f qdrant::embedding::build_payload
export -f qdrant::embedding::health_check