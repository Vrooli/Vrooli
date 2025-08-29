#!/usr/bin/env bash
# Optimized version of process_jsonl_file to fix performance issues
# This file contains the optimized batch processing function

set -euo pipefail

#######################################
# Process JSONL file with optimized batch handling
# Fixes O(n²) performance issue in original implementation
# Arguments:
#   $1 - JSONL file path
#   $2 - Content type
#   $3 - Collection name
#   $4 - App ID
#   $5 - Batch size (default: 25, reduced from 50 for stability)
# Returns: Number of items processed
#######################################
qdrant::embedding::process_jsonl_file_optimized() {
    local jsonl_file="$1"
    local content_type="$2"
    local collection="$3"
    local app_id="$4"
    local batch_size="${5:-25}"  # Reduced default batch size
    
    if [[ ! -f "$jsonl_file" ]] || [[ ! -s "$jsonl_file" ]]; then
        log::debug "No content in file: $jsonl_file"
        echo "0"
        return 0
    fi
    
    local total_lines=$(wc -l < "$jsonl_file")
    log::info "Processing $total_lines items from $jsonl_file using optimized batch processing"
    
    local processed_count=0
    local batch_num=0
    
    # Use temporary files to avoid repeated JSON parsing
    local temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT
    
    # Split the JSONL file into batches
    local current_batch=0
    local batch_file="$temp_dir/batch_${batch_num}.jsonl"
    
    while IFS= read -r json_line; do
        if [[ -z "$json_line" ]]; then
            continue
        fi
        
        echo "$json_line" >> "$batch_file"
        ((current_batch++))
        
        # Process batch when it reaches the size limit
        if [[ $current_batch -ge $batch_size ]]; then
            # Process this batch
            local batch_processed=$(qdrant::embedding::process_single_batch \
                "$batch_file" "$content_type" "$collection" "$app_id" "$batch_num")
            ((processed_count += batch_processed))
            
            # Start new batch
            ((batch_num++))
            current_batch=0
            batch_file="$temp_dir/batch_${batch_num}.jsonl"
        fi
    done < "$jsonl_file"
    
    # Process remaining items
    if [[ $current_batch -gt 0 ]] && [[ -f "$batch_file" ]]; then
        local batch_processed=$(qdrant::embedding::process_single_batch \
            "$batch_file" "$content_type" "$collection" "$app_id" "$batch_num")
        ((processed_count += batch_processed))
    fi
    
    log::success "Processed $processed_count/$total_lines items using optimized batch processing"
    echo "$processed_count"
}

#######################################
# Process a single batch efficiently
# Arguments:
#   $1 - Batch JSONL file
#   $2 - Content type
#   $3 - Collection name
#   $4 - App ID
#   $5 - Batch number (for logging)
# Returns: Number of items processed
#######################################
qdrant::embedding::process_single_batch() {
    local batch_file="$1"
    local content_type="$2"
    local collection="$3"
    local app_id="$4"
    local batch_num="${5:-0}"
    
    if [[ ! -f "$batch_file" ]] || [[ ! -s "$batch_file" ]]; then
        echo "0"
        return 0
    fi
    
    local batch_count=$(wc -l < "$batch_file")
    log::debug "Processing batch #$batch_num with $batch_count items..."
    
    # Extract all texts at once using jq
    local texts_file=$(mktemp)
    local metadata_file=$(mktemp)
    trap "rm -f $texts_file $metadata_file" RETURN
    
    # Extract content and metadata in single passes
    jq -r '.content // empty' "$batch_file" > "$texts_file"
    jq -c '.metadata // {}' "$batch_file" > "$metadata_file"
    
    # Build texts array for embedding generation
    local text_array=$(jq -Rs 'split("\n") | map(select(. != ""))' "$texts_file")
    
    # Generate embeddings for entire batch
    local embeddings
    embeddings=$(qdrant::embeddings::generate_batch "$text_array" "$DEFAULT_MODEL")
    
    if [[ -z "$embeddings" ]] || [[ "$embeddings" == "[]" ]]; then
        log::warn "Failed to generate embeddings for batch #$batch_num"
        echo "0"
        return 0
    fi
    
    # Build points array efficiently
    local points_file=$(mktemp)
    trap "rm -f $texts_file $metadata_file $points_file" RETURN
    
    echo "[]" > "$points_file"
    
    # Process each item and build points
    local idx=0
    local processed=0
    
    while IFS= read -r text; do
        if [[ -z "$text" ]]; then
            ((idx++))
            continue
        fi
        
        # Get corresponding metadata
        local metadata=$(sed -n "$((idx+1))p" "$metadata_file")
        
        # Get embedding
        local embedding=$(echo "$embeddings" | jq ".[$idx]")
        
        if [[ "$embedding" != "null" ]] && [[ -n "$embedding" ]]; then
            # Generate ID and payload
            local item_id=$(qdrant::embedding::generate_id "$text" "$content_type")
            local payload=$(qdrant::embedding::build_payload "$text" "$content_type" "$app_id" "$metadata")
            
            # Add point to array (using file to avoid memory issues)
            jq --arg id "$item_id" \
               --argjson vector "$embedding" \
               --argjson payload "$payload" \
               '. += [{id: $id, vector: $vector, payload: $payload}]' \
               "$points_file" > "$points_file.tmp" && mv "$points_file.tmp" "$points_file"
            
            ((processed++))
        fi
        ((idx++))
    done < "$texts_file"
    
    # Store batch in Qdrant
    if [[ $processed -gt 0 ]]; then
        local points=$(cat "$points_file")
        if qdrant::collections::batch_upsert "$collection" "$points"; then
            log::debug "Successfully stored batch #$batch_num with $processed points"
        else
            log::error "Failed to upsert batch #$batch_num to $collection"
        fi
    fi
    
    echo "$processed"
}

# Export the optimized function
export -f qdrant::embedding::process_jsonl_file_optimized
export -f qdrant::embedding::process_single_batch