#!/usr/bin/env bash
#
# Browserless Workflow Result Caching
#
# This module provides caching for workflow execution results to improve 
# performance for repeated operations. Cache is stored locally with TTL
# and automatic cleanup of expired entries.
#
# Cache key generation is based on workflow parameters to ensure
# consistency while avoiding collisions.
#

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }

# Cache configuration
CACHE_DIR="${BROWSERLESS_CACHE_DIR:-${BROWSERLESS_DATA_DIR}/cache}"
CACHE_TTL="${BROWSERLESS_CACHE_TTL:-3600}"  # Default 1 hour TTL
CACHE_MAX_SIZE="${BROWSERLESS_CACHE_MAX_SIZE:-100}"  # Max cache entries
CACHE_ENABLED="${BROWSERLESS_CACHE_ENABLED:-true}"

#######################################
# Initialize cache directory
#######################################
cache::init() {
    if [[ ! -d "$CACHE_DIR" ]]; then
        mkdir -p "$CACHE_DIR"
        log::debug "Cache directory initialized: $CACHE_DIR"
    fi
}

#######################################
# Generate cache key from workflow parameters
# Arguments:
#   $1 - Operation type (screenshot, extract, navigate, etc.)
#   $2 - URL or target
#   $3 - Additional parameters (JSON string)
# Returns:
#   Cache key (hash)
#######################################
cache::generate_key() {
    local operation="${1:?Operation required}"
    local target="${2:?Target required}"
    local params="${3:-{}}"
    
    # Create a deterministic string from parameters
    local cache_string="${operation}:${target}:${params}"
    
    # Generate MD5 hash for the cache key
    local cache_key
    cache_key=$(echo -n "$cache_string" | md5sum | cut -d' ' -f1)
    
    echo "$cache_key"
}

#######################################
# Store result in cache
# Arguments:
#   $1 - Cache key
#   $2 - Result data (JSON or string)
#   $3 - TTL in seconds (optional, defaults to CACHE_TTL)
# Returns:
#   0 on success, 1 on failure
#######################################
cache::set() {
    local cache_key="${1:?Cache key required}"
    local data="${2:?Data required}"
    local ttl="${3:-$CACHE_TTL}"
    
    if [[ "$CACHE_ENABLED" != "true" ]]; then
        return 0
    fi
    
    cache::init
    
    # Calculate expiry time
    local expiry=$(($(date +%s) + ttl))
    
    # Create cache entry
    local cache_file="${CACHE_DIR}/${cache_key}"
    local meta_file="${CACHE_DIR}/${cache_key}.meta"
    
    # Store data
    echo "$data" > "$cache_file"
    
    # Store metadata
    cat > "$meta_file" <<EOF
{
    "key": "$cache_key",
    "created": $(date +%s),
    "expiry": $expiry,
    "ttl": $ttl,
    "size": $(stat -c%s "$cache_file" 2>/dev/null || echo "0")
}
EOF
    
    log::debug "Cached result: key=$cache_key, ttl=${ttl}s"
    
    # Trigger cleanup if cache is getting full
    cache::cleanup_if_needed
    
    return 0
}

#######################################
# Get result from cache
# Arguments:
#   $1 - Cache key
# Returns:
#   Cached data if valid, empty string if not found or expired
#######################################
cache::get() {
    local cache_key="${1:?Cache key required}"
    
    if [[ "$CACHE_ENABLED" != "true" ]]; then
        return 1
    fi
    
    local cache_file="${CACHE_DIR}/${cache_key}"
    local meta_file="${CACHE_DIR}/${cache_key}.meta"
    
    # Check if cache files exist
    if [[ ! -f "$cache_file" ]] || [[ ! -f "$meta_file" ]]; then
        log::debug "Cache miss: key=$cache_key (not found)"
        return 1
    fi
    
    # Check expiry
    local expiry
    expiry=$(jq -r '.expiry' "$meta_file" 2>/dev/null || echo "0")
    local current_time=$(date +%s)
    
    if [[ $current_time -gt $expiry ]]; then
        log::debug "Cache miss: key=$cache_key (expired)"
        # Remove expired entry
        rm -f "$cache_file" "$meta_file"
        return 1
    fi
    
    log::debug "Cache hit: key=$cache_key"
    cat "$cache_file"
    return 0
}

#######################################
# Delete cache entry
# Arguments:
#   $1 - Cache key
#######################################
cache::delete() {
    local cache_key="${1:?Cache key required}"
    
    local cache_file="${CACHE_DIR}/${cache_key}"
    local meta_file="${CACHE_DIR}/${cache_key}.meta"
    
    rm -f "$cache_file" "$meta_file"
    log::debug "Cache entry deleted: key=$cache_key"
}

#######################################
# Clear all cache entries
#######################################
cache::clear() {
    if [[ -d "$CACHE_DIR" ]]; then
        log::info "Clearing cache directory..."
        rm -rf "${CACHE_DIR:?}"/*
        log::success "Cache cleared"
    fi
}

#######################################
# Remove expired cache entries
#######################################
cache::cleanup_expired() {
    local current_time=$(date +%s)
    local cleaned_count=0
    
    if [[ ! -d "$CACHE_DIR" ]]; then
        return 0
    fi
    
    for meta_file in "$CACHE_DIR"/*.meta; do
        [[ -f "$meta_file" ]] || continue
        
        local expiry
        expiry=$(jq -r '.expiry' "$meta_file" 2>/dev/null || echo "0")
        
        if [[ $current_time -gt $expiry ]]; then
            local cache_key
            cache_key=$(basename "$meta_file" .meta)
            cache::delete "$cache_key"
            ((cleaned_count++))
        fi
    done
    
    if [[ $cleaned_count -gt 0 ]]; then
        log::debug "Cleaned $cleaned_count expired cache entries"
    fi
}

#######################################
# Cleanup if cache size exceeds limit
#######################################
cache::cleanup_if_needed() {
    local entry_count
    entry_count=$(find "$CACHE_DIR" -name "*.meta" 2>/dev/null | wc -l)
    
    if [[ $entry_count -gt $CACHE_MAX_SIZE ]]; then
        log::debug "Cache size exceeded ($entry_count > $CACHE_MAX_SIZE), cleaning up..."
        
        # Remove oldest entries first
        local to_remove=$((entry_count - CACHE_MAX_SIZE + 10))  # Remove 10 extra for buffer
        
        # Sort by creation time and remove oldest
        find "$CACHE_DIR" -name "*.meta" -type f -printf '%T+ %p\n' 2>/dev/null | \
            sort | head -n "$to_remove" | cut -d' ' -f2 | \
            while read -r meta_file; do
                local cache_key
                cache_key=$(basename "$meta_file" .meta)
                cache::delete "$cache_key"
            done
        
        log::debug "Removed $to_remove old cache entries"
    fi
}

#######################################
# Get cache statistics
#######################################
cache::stats() {
    if [[ ! -d "$CACHE_DIR" ]]; then
        echo "Cache not initialized"
        return 0
    fi
    
    local total_entries
    local total_size
    local expired_count=0
    local current_time=$(date +%s)
    
    total_entries=$(find "$CACHE_DIR" -name "*.meta" 2>/dev/null | wc -l)
    total_size=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)
    
    # Count expired entries
    for meta_file in "$CACHE_DIR"/*.meta; do
        [[ -f "$meta_file" ]] || continue
        
        local expiry
        expiry=$(jq -r '.expiry' "$meta_file" 2>/dev/null || echo "0")
        
        if [[ $current_time -gt $expiry ]]; then
            ((expired_count++))
        fi
    done
    
    cat <<EOF
Cache Statistics:
  Directory: $CACHE_DIR
  Total entries: $total_entries
  Expired entries: $expired_count
  Active entries: $((total_entries - expired_count))
  Total size: ${total_size:-0}
  Max entries: $CACHE_MAX_SIZE
  Default TTL: ${CACHE_TTL}s
  Cache enabled: $CACHE_ENABLED
EOF
}

#######################################
# Cached workflow execution wrapper
# Arguments:
#   $1 - Operation type
#   $2 - Target URL
#   $3 - Parameters (JSON)
#   $4 - Execution function
# Returns:
#   Cached or fresh result
#######################################
cache::workflow() {
    local operation="${1:?Operation required}"
    local target="${2:?Target required}"
    local params="${3:-{}}"
    local exec_function="${4:?Execution function required}"
    
    # Generate cache key
    local cache_key
    cache_key=$(cache::generate_key "$operation" "$target" "$params")
    
    # Try to get from cache
    local cached_result
    if cached_result=$(cache::get "$cache_key"); then
        echo "$cached_result"
        return 0
    fi
    
    # Execute the workflow
    log::debug "Executing workflow: $operation on $target"
    local result
    if result=$($exec_function "$target" "$params"); then
        # Cache the successful result
        cache::set "$cache_key" "$result"
        echo "$result"
        return 0
    else
        log::error "Workflow execution failed"
        return 1
    fi
}

# Export functions
export -f cache::init
export -f cache::generate_key
export -f cache::set
export -f cache::get
export -f cache::delete
export -f cache::clear
export -f cache::cleanup_expired
export -f cache::cleanup_if_needed
export -f cache::stats
export -f cache::workflow