#!/usr/bin/env bash
# ====================================================================
# Cache Manager for Layer 1 Syntax Validation
# ====================================================================
#
# Provides high-performance caching for validation results based on
# file content hashes. Reduces validation time by 90%+ for unchanged
# resources.
#
# Usage:
#   source cache-manager.sh
#   cache_manager_init
#   cache_get "resource_name" "file_hash" || cache_set "resource_name" "file_hash" "validation_result"
#
# Cache Format:
#   - Key: syntax-{resource_name}-{sha256_hash}.json
#   - TTL: 1 hour (3600 seconds)
#   - Location: tests/framework/cache/
#
# ====================================================================

set -euo pipefail

_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${_HERE}/../../../../../lib/utils/var.sh"
# shellcheck disable=SC1091,SC2154,SC1090
source "${var_LOG_FILE}" 2>/dev/null || true

# Cache configuration
CACHE_DIR=""
CACHE_TTL_SECONDS=3600  # 1 hour
CACHE_PREFIX="syntax"
CACHE_EXTENSION="json"

# Performance metrics
CACHE_HITS=0
CACHE_MISSES=0
CACHE_WRITES=0

#######################################
# Initialize cache manager
# Sets up cache directory and validates environment
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
cache_manager::init() {
    # Determine cache directory
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    CACHE_DIR="$script_dir"
    
    # Create cache directory if it doesn't exist
    if ! mkdir -p "$CACHE_DIR"; then
        echo "ERROR: Failed to create cache directory: $CACHE_DIR" >&2
        return 1
    fi
    
    # Validate required tools
    if ! command -v sha256sum &> /dev/null; then
        echo "ERROR: sha256sum not found - required for cache key generation" >&2
        return 1
    fi
    
    # Initialize performance counters
    CACHE_HITS=0
    CACHE_MISSES=0
    CACHE_WRITES=0
    
    return 0
}

#######################################
# Generate cache key from file path
# Arguments: $1 - resource name, $2 - file path
# Returns: 0 on success, outputs cache key
#######################################
cache_manager::generate_key() {
    local resource_name="$1"
    local file_path="$2"
    
    if [[ ! -f "$file_path" ]]; then
        echo "ERROR: File not found for cache key generation: $file_path" >&2
        return 1
    fi
    
    # Generate SHA256 hash of file content
    local file_hash
    file_hash=$(sha256sum "$file_path" | cut -d' ' -f1)
    
    # Generate cache key
    echo "${CACHE_PREFIX}-${resource_name}-${file_hash}"
}

#######################################
# Get cache file path
# Arguments: $1 - cache key
# Returns: 0 on success, outputs file path
#######################################
cache_manager::get_file_path() {
    local cache_key="$1"
    echo "${CACHE_DIR}/${cache_key}.${CACHE_EXTENSION}"
}

#######################################
# Check if cache entry is valid (not expired)
# Arguments: $1 - cache file path
# Returns: 0 if valid, 1 if expired or missing
#######################################
cache_manager::is_valid() {
    local cache_file="$1"
    
    # Check if file exists
    if [[ ! -f "$cache_file" ]]; then
        return 1
    fi
    
    # Check if file is within TTL
    if [[ $(find "$cache_file" -mtime -$((CACHE_TTL_SECONDS/86400)) -print 2>/dev/null | wc -l) -eq 0 ]]; then
        # Use more precise check for files less than a day old
        local file_age
        file_age=$(stat -c %Y "$cache_file" 2>/dev/null || echo 0)
        local current_time
        current_time=$(date +%s)
        local age_seconds=$((current_time - file_age))
        
        if [[ $age_seconds -gt $CACHE_TTL_SECONDS ]]; then
            return 1  # Expired
        fi
    fi
    
    return 0  # Valid
}

#######################################
# Get cached validation result
# Arguments: $1 - resource name, $2 - file path
# Returns: 0 if found, 1 if miss, outputs cached result
#######################################
cache_manager::get() {
    local resource_name="$1"
    local file_path="$2"
    
    # Generate cache key
    local cache_key
    cache_key=$(cache_manager::generate_key "$resource_name" "$file_path") || return 1
    
    # Get cache file path
    local cache_file
    cache_file=$(cache_manager::get_file_path "$cache_key")
    
    # Check if cache is valid
    if cache_manager::is_valid "$cache_file"; then
        # Cache hit - return cached result
        cat "$cache_file"
        CACHE_HITS=$((CACHE_HITS + 1))
        return 0
    else
        # Cache miss or expired
        CACHE_MISSES=$((CACHE_MISSES + 1))
        return 1
    fi
}

#######################################
# Set cached validation result
# Arguments: $1 - resource name, $2 - file path, $3 - validation result (JSON)
# Returns: 0 on success, 1 on failure
#######################################
cache_manager::set() {
    local resource_name="$1"
    local file_path="$2"
    local validation_result="$3"
    
    # Generate cache key
    local cache_key
    cache_key=$(cache_manager::generate_key "$resource_name" "$file_path") || return 1
    
    # Get cache file path
    local cache_file
    cache_file=$(cache_manager::get_file_path "$cache_key")
    
    # Write validation result to cache
    if echo "$validation_result" > "$cache_file"; then
        CACHE_WRITES=$((CACHE_WRITES + 1))
        return 0
    else
        echo "ERROR: Failed to write cache file: $cache_file" >&2
        return 1
    fi
}

#######################################
# Clear expired cache entries
# Arguments: None
# Returns: 0 on success, outputs number of cleared entries
#######################################
cache_manager::clear_expired() {
    local cleared_count=0
    
    # Find and remove expired cache files
    while IFS= read -r -d '' cache_file; do
        if [[ -f "$cache_file" ]] && ! cache_manager::is_valid "$cache_file"; then
            rm -f "$cache_file"
            cleared_count=$((cleared_count + 1))
        fi
    done < <(find "$CACHE_DIR" -name "${CACHE_PREFIX}-*.${CACHE_EXTENSION}" -print0 2>/dev/null)
    
    echo "Cleared $cleared_count expired cache entries"
    return 0
}

#######################################
# Clear all cache entries
# Arguments: None
# Returns: 0 on success, outputs number of cleared entries
#######################################
cache_manager::clear_all() {
    local cleared_count=0
    
    # Count and remove all cache files
    while IFS= read -r -d '' cache_file; do
        if [[ -f "$cache_file" ]]; then
            rm -f "$cache_file"
            cleared_count=$((cleared_count + 1))
        fi
    done < <(find "$CACHE_DIR" -name "${CACHE_PREFIX}-*.${CACHE_EXTENSION}" -print0 2>/dev/null)
    
    echo "Cleared $cleared_count cache entries"
    return 0
}

#######################################
# Get cache statistics
# Arguments: None
# Returns: 0 on success, outputs cache stats in JSON format
#######################################
cache_manager::get_stats() {
    local total_entries
    total_entries=$(find "$CACHE_DIR" -name "${CACHE_PREFIX}-*.${CACHE_EXTENSION}" 2>/dev/null | wc -l)
    
    local cache_size_kb
    cache_size_kb=$(du -sk "$CACHE_DIR" 2>/dev/null | cut -f1 || echo 0)
    
    local hit_rate=0
    if [[ $((CACHE_HITS + CACHE_MISSES)) -gt 0 ]]; then
        hit_rate=$(( (CACHE_HITS * 100) / (CACHE_HITS + CACHE_MISSES) ))
    fi
    
    cat << EOF
{
  "cache_hits": ${CACHE_HITS},
  "cache_misses": ${CACHE_MISSES},
  "cache_writes": ${CACHE_WRITES},
  "hit_rate_percent": ${hit_rate},
  "total_entries": ${total_entries},
  "cache_size_kb": ${cache_size_kb},
  "cache_directory": "${CACHE_DIR}",
  "ttl_seconds": ${CACHE_TTL_SECONDS}
}
EOF
}

#######################################
# Create validation result JSON
# Helper function to format validation results for caching
# Arguments: $1 - status (passed/failed), $2 - details, $3 - duration_ms
# Returns: 0 on success, outputs JSON
#######################################
cache_manager::create_result_json() {
    local status="$1"
    local details="$2"
    local duration_ms="$3"
    local timestamp
    timestamp=$(date -Iseconds)
    
    # Escape quotes in details
    local escaped_details
    escaped_details="${details//\"/\\\"}"
    
    cat << EOF
{
  "status": "$status",
  "details": "$escaped_details",
  "duration_ms": $duration_ms,
  "timestamp": "$timestamp",
  "cache_version": "1.0"
}
EOF
}

#######################################
# Parse cached result JSON
# Arguments: $1 - cached JSON result
# Returns: 0 if passed, 1 if failed, outputs details to stderr
#######################################
cache_manager::parse_result() {
    local cached_json="$1"
    
    # Extract status and details from JSON
    local status
    status=$(echo "$cached_json" | grep -o '"status"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "unknown")
    
    local details
    details=$(echo "$cached_json" | grep -o '"details"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "No details")
    
    # Output details to stderr for logging
    echo "$details" >&2
    
    # Return status code
    if [[ "$status" == "passed" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate cache integrity
# Check for corrupted or invalid cache files
# Arguments: None
# Returns: 0 on success, outputs validation report
#######################################
cache_manager::validate_integrity() {
    local total_files=0
    local valid_files=0
    local invalid_files=0
    local corrupted_files=()
    
    while IFS= read -r -d '' cache_file; do
        total_files=$((total_files + 1))
        
        # Check if file contains valid JSON
        if jq . "$cache_file" >/dev/null 2>&1; then
            # Check for required fields
            if jq -e '.status and .details and .timestamp' "$cache_file" >/dev/null 2>&1; then
                valid_files=$((valid_files + 1))
            else
                invalid_files=$((invalid_files + 1))
                corrupted_files+=("$cache_file")
            fi
        else
            invalid_files=$((invalid_files + 1))
            corrupted_files+=("$cache_file")
        fi
    done < <(find "$CACHE_DIR" -name "${CACHE_PREFIX}-*.${CACHE_EXTENSION}" -print0 2>/dev/null)
    
    # Report results
    cat << EOF
Cache Integrity Report:
  Total files: $total_files
  Valid files: $valid_files
  Invalid files: $invalid_files
EOF
    
    if [[ ${#corrupted_files[@]} -gt 0 ]]; then
        echo "  Corrupted files:"
        for file in "${corrupted_files[@]}"; do
            echo "    - $(basename "$file")"
        done
    fi
    
    return 0
}

# Export functions for use in other scripts
export -f cache_manager::init cache_manager::generate_key cache_manager::get_file_path cache_manager::is_valid
export -f cache_manager::get cache_manager::set cache_manager::clear_expired cache_manager::clear_all cache_manager::get_stats
export -f cache_manager::create_result_json cache_manager::parse_result cache_manager::validate_integrity