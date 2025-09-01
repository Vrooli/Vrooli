#!/usr/bin/env bash
################################################################################
# Qdrant Collections Optimizer
# 
# Optimizes existing collections with improved HNSW parameters for better
# search performance and handles large-scale vector operations
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
source "${APP_ROOT}/resources/qdrant/lib/http-client-improved.sh"

# Configuration
QDRANT_URL="http://localhost:6333"

#######################################
# Get collection info
# Arguments:
#   $1 - Collection name
#######################################
get_collection_info() {
    local collection="$1"
    http_client::get "/collections/$collection" "get collection info"
}

#######################################
# Update collection HNSW configuration
# Arguments:
#   $1 - Collection name
#   $2 - Points count (for parameter optimization)
#######################################
optimize_collection_hnsw() {
    local collection="$1"
    local points_count="$2"
    
    log::info "🔧 Optimizing HNSW parameters for $collection ($points_count points)"
    
    # Calculate optimal parameters based on collection size
    local m=16
    local ef_construct=100
    local ef=128
    
    if [[ $points_count -gt 100000 ]]; then
        # Large collections: optimize for accuracy
        m=32
        ef_construct=200
        ef=150
    elif [[ $points_count -gt 10000 ]]; then
        # Medium collections: balance speed/accuracy
        m=24
        ef_construct=150
        ef=140
    elif [[ $points_count -gt 1000 ]]; then
        # Small collections: optimize for speed
        m=16
        ef_construct=100
        ef=128
    fi
    
    local hnsw_config=$(jq -n \
        --argjson m "$m" \
        --argjson ef_construct "$ef_construct" \
        --argjson ef "$ef" \
        '{
            hnsw_config: {
                m: $m,
                ef_construct: $ef_construct,
                full_scan_threshold: 10000,
                max_indexing_threads: 0,
                on_disk: false
            }
        }')
    
    log::debug "HNSW config: m=$m, ef_construct=$ef_construct, ef=$ef"
    
    # Apply configuration
    if http_client::patch "/collections/$collection" "$hnsw_config" "optimize HNSW config" >/dev/null 2>&1; then
        log::success "✅ Optimized $collection HNSW parameters"
        
        # Trigger index rebuild
        trigger_optimization "$collection"
    else
        log::error "❌ Failed to optimize $collection"
        return 1
    fi
}

#######################################
# Update collection optimizer configuration
# Arguments:
#   $1 - Collection name
#   $2 - Points count
#######################################
optimize_collection_optimizer() {
    local collection="$1" 
    local points_count="$2"
    
    log::info "⚙️ Optimizing storage parameters for $collection"
    
    # Calculate optimal storage parameters
    local memmap_threshold=20000
    local indexing_threshold=1
    local max_segment_size=""
    
    if [[ $points_count -gt 100000 ]]; then
        memmap_threshold=50000
        indexing_threshold=10000
        max_segment_size="100000"
    elif [[ $points_count -gt 10000 ]]; then
        memmap_threshold=30000
        indexing_threshold=1000
    fi
    
    local optimizer_config=$(jq -n \
        --argjson memmap_threshold "$memmap_threshold" \
        --argjson indexing_threshold "$indexing_threshold" \
        --arg max_segment_size "$max_segment_size" \
        '{
            optimizers_config: {
                deleted_threshold: 0.2,
                vacuum_min_vector_number: 1000,
                default_segment_number: 0,
                max_segment_size: (if $max_segment_size == "" then null else ($max_segment_size | tonumber) end),
                memmap_threshold: $memmap_threshold,
                indexing_threshold: $indexing_threshold,
                flush_interval_sec: 5,
                max_optimization_threads: null
            }
        }')
    
    if http_client::patch "/collections/$collection" "$optimizer_config" "optimize storage config" >/dev/null 2>&1; then
        log::success "✅ Optimized $collection storage parameters"
    else
        log::error "❌ Failed to optimize $collection storage"
        return 1
    fi
}

#######################################
# Trigger collection optimization
# Arguments:
#   $1 - Collection name
#######################################
trigger_optimization() {
    local collection="$1"
    
    log::info "🔄 Triggering optimization for $collection"
    
    local optimize_request='{}'
    
    if http_client::post "/collections/$collection/index" "$optimize_request" "trigger optimization" >/dev/null 2>&1; then
        log::success "✅ Optimization triggered for $collection"
    else
        log::warn "⚠️ Could not trigger optimization for $collection (may not be needed)"
    fi
}

#######################################
# Add quantization to reduce memory usage
# Arguments:
#   $1 - Collection name
#   $2 - Points count
#######################################
add_quantization() {
    local collection="$1"
    local points_count="$2"
    
    # Only add quantization for large collections
    if [[ $points_count -lt 10000 ]]; then
        log::debug "Skipping quantization for small collection $collection"
        return 0
    fi
    
    log::info "🗜️ Adding scalar quantization to $collection"
    
    local quantization_config='{
        quantization_config: {
            scalar: {
                type: "int8",
                quantile: 0.99,
                always_ram: false
            }
        }
    }'
    
    if http_client::patch "/collections/$collection" "$quantization_config" "add quantization" >/dev/null 2>&1; then
        log::success "✅ Added quantization to $collection"
    else
        log::warn "⚠️ Could not add quantization to $collection"
    fi
}

#######################################
# Get all collections and their stats
#######################################
get_all_collections() {
    http_client::get "/collections" "list collections" | jq -r '.result.collections[].name'
}

#######################################
# Optimize single collection
# Arguments:
#   $1 - Collection name
#######################################
optimize_single_collection() {
    local collection="$1"
    
    log::info "🎯 Analyzing collection: $collection"
    
    # Get collection info
    local collection_info
    if ! collection_info=$(get_collection_info "$collection"); then
        log::error "❌ Cannot get info for collection $collection"
        return 1
    fi
    
    # Extract points count and current config
    local points_count
    points_count=$(echo "$collection_info" | jq -r '.result.points_count // 0')
    
    local status
    status=$(echo "$collection_info" | jq -r '.result.status')
    
    log::info "📊 Collection $collection: $points_count points, status: $status"
    
    # Skip if collection is empty
    if [[ $points_count -eq 0 ]]; then
        log::info "⏭️ Skipping empty collection $collection"
        return 0
    fi
    
    # Get current HNSW settings
    local current_m
    current_m=$(echo "$collection_info" | jq -r '.result.config.hnsw_config.m // 16')
    local current_ef_construct
    current_ef_construct=$(echo "$collection_info" | jq -r '.result.config.hnsw_config.ef_construct // 100')
    
    log::debug "Current HNSW: m=$current_m, ef_construct=$current_ef_construct"
    
    # Optimize configurations
    optimize_collection_hnsw "$collection" "$points_count"
    optimize_collection_optimizer "$collection" "$points_count"
    add_quantization "$collection" "$points_count"
    
    log::success "✅ Completed optimization of $collection"
    echo
}

#######################################
# Create HTTP PATCH method (missing from http-client-improved.sh)
#######################################
http_client::patch() {
    http_client::request "PATCH" "$1" "$2" "${3:-PATCH $1}"
}

#######################################
# Main optimization function
#######################################
main() {
    local target_collection="${1:-}"
    
    echo "🚀 Starting Qdrant Collections Optimization"
    echo "============================================="
    echo
    
    # Initialize HTTP client
    http_client::init
    
    if [[ -n "$target_collection" ]]; then
        # Optimize single collection
        optimize_single_collection "$target_collection"
    else
        # Optimize all collections
        log::info "🔍 Finding all collections..."
        
        local collections
        if ! collections=$(get_all_collections); then
            log::error "❌ Cannot list collections"
            exit 1
        fi
        
        local collection_count
        collection_count=$(echo "$collections" | wc -l)
        
        log::info "📋 Found $collection_count collections to optimize"
        echo
        
        # Optimize each collection
        local optimized=0
        local failed=0
        
        echo "$collections" | while read -r collection; do
            [[ -n "$collection" && "$collection" != "null" ]] || continue
            
            if optimize_single_collection "$collection"; then
                ((optimized++))
            else
                ((failed++))
            fi
        done
        
        echo "============================================="
        log::success "🎉 Optimization complete: $optimized successful, $failed failed"
        
        # Show HTTP client stats
        echo
        http_client::stats
    fi
}

# Run optimization
main "$@"