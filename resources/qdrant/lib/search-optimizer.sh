#!/usr/bin/env bash
# Optimized vector similarity search for Qdrant
# Addresses performance degradation issues documented in PROBLEMS.md

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/api-client.sh"

# Performance optimization constants
readonly OPTIMAL_EF_CONSTRUCT=200  # Improved from default 128
readonly OPTIMAL_M=16              # Connections per node
readonly MAX_BATCH_SIZE=1000       # Prevent memory exhaustion
readonly SEARCH_TIMEOUT=2          # 2 second timeout for searches
readonly CACHE_TTL=300             # 5 minute cache for repeated queries

#######################################
# Optimize collection for large-scale search
# Arguments:
#   $1 - Collection name
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::optimize_collection() {
    local collection="${1:-}"
    
    if [[ -z "$collection" ]]; then
        log::error "Collection name required"
        return 1
    fi
    
    log::info "Optimizing collection '$collection' for performance..."
    
    # Get current collection info
    local collection_info
    if ! collection_info=$(qdrant::api::get "/collections/${collection}" 2>/dev/null); then
        log::error "Failed to get collection info"
        return 1
    fi
    
    local vector_count
    vector_count=$(echo "$collection_info" | jq -r '.result.vectors_count // 0')
    
    # Calculate optimal parameters based on collection size
    local optimal_ef optimal_segments
    if [[ $vector_count -gt 1000000 ]]; then
        optimal_ef=400
        optimal_segments=8
    elif [[ $vector_count -gt 100000 ]]; then
        optimal_ef=200
        optimal_segments=4
    else
        optimal_ef=128
        optimal_segments=2
    fi
    
    # Update collection optimizer config
    local optimizer_config='{
        "indexing_threshold": 20000,
        "deleted_threshold": 0.2,
        "vacuum_min_vector_number": 1000,
        "default_segment_number": '$optimal_segments',
        "max_segment_size": 200000,
        "memmap_threshold": 100000,
        "max_optimization_threads": 4
    }'
    
    if ! qdrant::api::patch "/collections/${collection}" \
        "{\"optimizers_config\": $optimizer_config}" 2>/dev/null; then
        log::warn "Failed to update optimizer config"
    fi
    
    # Update HNSW config for better search performance
    local hnsw_config='{
        "m": '$OPTIMAL_M',
        "ef_construct": '$optimal_ef',
        "full_scan_threshold": 10000,
        "max_indexing_threads": 0,
        "on_disk": false,
        "payload_m": null
    }'
    
    if ! qdrant::api::patch "/collections/${collection}" \
        "{\"hnsw_config\": $hnsw_config}" 2>/dev/null; then
        log::warn "Failed to update HNSW config"
    fi
    
    log::success "Collection optimization complete"
    return 0
}

#######################################
# Perform optimized vector search with caching
# Arguments:
#   $1 - Collection name
#   $2 - Vector (JSON array)
#   $3 - Limit (default: 10)
#   $4 - With payload (true/false, default: false)
# Returns: Search results as JSON
#######################################
qdrant::search::optimized_search() {
    local collection="${1:-}"
    local vector="${2:-}"
    local limit="${3:-10}"
    local with_payload="${4:-false}"
    
    if [[ -z "$collection" ]] || [[ -z "$vector" ]]; then
        log::error "Collection and vector required"
        return 1
    fi
    
    # Generate cache key from search parameters
    local cache_key
    cache_key=$(echo "${collection}:${vector}:${limit}:${with_payload}" | sha256sum | cut -d' ' -f1)
    local cache_file="/tmp/qdrant_search_cache_${cache_key}"
    
    # Check cache
    if [[ -f "$cache_file" ]]; then
        local cache_age
        cache_age=$(($(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || echo 0)))
        if [[ $cache_age -lt $CACHE_TTL ]]; then
            log::debug "Returning cached search result"
            cat "$cache_file"
            return 0
        fi
    fi
    
    # Prepare search request with optimizations
    local search_params='{
        "params": {
            "hnsw_ef": 128,
            "exact": false,
            "indexed_only": true
        }
    }'
    
    # For very small result sets, use exact search
    if [[ $limit -le 5 ]]; then
        search_params='{"params": {"exact": true}}'
    fi
    
    local search_request
    search_request=$(jq -n \
        --argjson vector "$vector" \
        --argjson limit "$limit" \
        --argjson with_payload "$with_payload" \
        --argjson params "$search_params" \
        '{
            vector: $vector,
            limit: $limit,
            with_payload: $with_payload,
            params: $params.params
        }')
    
    # Perform search with timeout
    local result
    if ! result=$(timeout "$SEARCH_TIMEOUT" \
        qdrant::api::post "/collections/${collection}/points/search" \
        "$search_request" 2>/dev/null); then
        log::error "Search timeout or failed"
        return 1
    fi
    
    # Cache successful result
    echo "$result" > "$cache_file"
    
    echo "$result"
    return 0
}

#######################################
# Batch upsert with memory management
# Arguments:
#   $1 - Collection name
#   $2 - Points data (JSON)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::search::batch_upsert_optimized() {
    local collection="${1:-}"
    local points_data="${2:-}"
    
    if [[ -z "$collection" ]] || [[ -z "$points_data" ]]; then
        log::error "Collection and points data required"
        return 1
    fi
    
    # Parse points array
    local points_count
    points_count=$(echo "$points_data" | jq -r '.points | length')
    
    if [[ $points_count -eq 0 ]]; then
        log::warn "No points to upsert"
        return 0
    fi
    
    log::info "Upserting $points_count points in optimized batches..."
    
    # Process in batches to prevent memory exhaustion
    local batch_start=0
    local success_count=0
    
    while [[ $batch_start -lt $points_count ]]; do
        local batch_end=$((batch_start + MAX_BATCH_SIZE))
        if [[ $batch_end -gt $points_count ]]; then
            batch_end=$points_count
        fi
        
        # Extract batch
        local batch
        batch=$(echo "$points_data" | jq \
            --argjson start "$batch_start" \
            --argjson end "$batch_end" \
            '{points: .points[$start:$end]}')
        
        # Upsert batch with wait for indexing
        local upsert_params='?wait=true&ordering=weak'
        
        if qdrant::api::put "/collections/${collection}/points${upsert_params}" \
            "$batch" 2>/dev/null; then
            success_count=$((success_count + (batch_end - batch_start)))
            log::debug "Batch $batch_start-$batch_end upserted successfully"
        else
            log::warn "Failed to upsert batch $batch_start-$batch_end"
        fi
        
        batch_start=$batch_end
        
        # Brief pause between batches to prevent overload
        sleep 0.1
    done
    
    log::success "Upserted $success_count/$points_count points"
    
    if [[ $success_count -eq $points_count ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Monitor search performance metrics
# Returns: Performance metrics as JSON
#######################################
qdrant::search::performance_metrics() {
    local collections_json
    if ! collections_json=$(qdrant::api::get "/collections" 2>/dev/null); then
        log::error "Failed to get collections"
        return 1
    fi
    
    local metrics='{"collections": []}'
    
    # Gather metrics for each collection
    local collections
    collections=$(echo "$collections_json" | jq -r '.result.collections[].name')
    
    while IFS= read -r collection; do
        [[ -z "$collection" ]] && continue
        
        # Get collection metrics
        local collection_info
        if collection_info=$(qdrant::api::get "/collections/${collection}" 2>/dev/null); then
            local vector_count segment_count indexed_vectors
            vector_count=$(echo "$collection_info" | jq -r '.result.vectors_count // 0')
            segment_count=$(echo "$collection_info" | jq -r '.result.segments_count // 0')
            indexed_vectors=$(echo "$collection_info" | jq -r '.result.indexed_vectors_count // 0')
            
            # Calculate indexing percentage
            local index_percentage=0
            if [[ $vector_count -gt 0 ]]; then
                index_percentage=$((indexed_vectors * 100 / vector_count))
            fi
            
            # Test search latency with small random vector
            local vector_size
            vector_size=$(echo "$collection_info" | jq -r '.result.config.params.vectors.size // 0')
            
            if [[ $vector_size -gt 0 ]]; then
                local test_vector
                test_vector=$(python3 -c "import random, json; print(json.dumps([random.random() for _ in range($vector_size)]))" 2>/dev/null || echo "[]")
                
                local search_start search_end latency_ms
                search_start=$(date +%s%N)
                
                if timeout 1 qdrant::api::post "/collections/${collection}/points/search" \
                    "{\"vector\": $test_vector, \"limit\": 1}" &>/dev/null; then
                    search_end=$(date +%s%N)
                    latency_ms=$(((search_end - search_start) / 1000000))
                else
                    latency_ms=-1
                fi
            else
                latency_ms=-1
            fi
            
            # Add to metrics
            metrics=$(echo "$metrics" | jq \
                --arg name "$collection" \
                --argjson vectors "$vector_count" \
                --argjson segments "$segment_count" \
                --argjson indexed "$index_percentage" \
                --argjson latency "$latency_ms" \
                '.collections += [{
                    name: $name,
                    vector_count: $vectors,
                    segment_count: $segments,
                    indexed_percentage: $indexed,
                    search_latency_ms: $latency
                }]')
        fi
    done <<< "$collections"
    
    # Add overall health status
    local total_vectors total_collections avg_latency
    total_vectors=$(echo "$metrics" | jq '[.collections[].vector_count] | add // 0')
    total_collections=$(echo "$metrics" | jq '.collections | length')
    avg_latency=$(echo "$metrics" | jq '[.collections[] | select(.search_latency_ms > 0) | .search_latency_ms] | if length > 0 then add/length else 0 end')
    
    metrics=$(echo "$metrics" | jq \
        --argjson total_vectors "$total_vectors" \
        --argjson total_collections "$total_collections" \
        --argjson avg_latency "$avg_latency" \
        '. + {
            summary: {
                total_vectors: $total_vectors,
                total_collections: $total_collections,
                average_search_latency_ms: ($avg_latency | floor),
                performance_status: (if $avg_latency < 50 then "excellent" elif $avg_latency < 200 then "good" elif $avg_latency < 1000 then "degraded" else "critical" end)
            }
        }')
    
    echo "$metrics"
    return 0
}

#######################################
# Clean search cache
#######################################
qdrant::search::clean_cache() {
    log::info "Cleaning search cache..."
    find /tmp -name "qdrant_search_cache_*" -mmin +5 -delete 2>/dev/null || true
    log::success "Cache cleaned"
    return 0
}