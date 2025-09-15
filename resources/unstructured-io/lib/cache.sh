#!/usr/bin/env bash

# Unstructured.io Caching Functions
# This file provides Redis-based caching for document processing results

# Cache configuration
readonly UNSTRUCTURED_IO_CACHE_TTL="${UNSTRUCTURED_IO_CACHE_TTL:-3600}"  # 1 hour default
readonly UNSTRUCTURED_IO_REDIS_HOST="${UNSTRUCTURED_IO_REDIS_HOST:-localhost}"
readonly UNSTRUCTURED_IO_REDIS_PORT="${UNSTRUCTURED_IO_REDIS_PORT:-6380}"
readonly UNSTRUCTURED_IO_CACHE_PREFIX="unstructured:cache:"

#######################################
# Check if Redis is available
#######################################
unstructured_io::cache_available() {
    # Try to connect using nc (netcat) as a fallback
    if command -v nc &> /dev/null; then
        if echo "PING" | nc -w 1 "$UNSTRUCTURED_IO_REDIS_HOST" "$UNSTRUCTURED_IO_REDIS_PORT" 2>/dev/null | grep -q "+PONG"; then
            return 0
        fi
    fi
    
    # Try using Docker if available
    if command -v docker &> /dev/null; then
        if docker exec vrooli-redis-resource redis-cli ping 2>/dev/null | grep -q "PONG"; then
            return 0
        fi
    fi
    
    # If redis-cli is available, use it
    if command -v redis-cli &> /dev/null; then
        if redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" ping 2>/dev/null | grep -q "PONG"; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Generate cache key for a file
#######################################
unstructured_io::get_cache_key() {
    local file="$1"
    local strategy="${2:-hi_res}"
    local output_format="${3:-json}"
    local languages="${4:-eng}"
    
    # Get file hash (use sha256sum if available, otherwise md5sum)
    local file_hash=""
    if command -v sha256sum &> /dev/null; then
        file_hash=$(sha256sum "$file" 2>/dev/null | cut -d' ' -f1)
    elif command -v md5sum &> /dev/null; then
        file_hash=$(md5sum "$file" 2>/dev/null | cut -d' ' -f1)
    else
        # Fallback to file size and modification time
        local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
        local file_mtime=$(stat -f%m "$file" 2>/dev/null || stat -c%Y "$file" 2>/dev/null || echo "0")
        file_hash="${file_size}_${file_mtime}"
    fi
    
    # Include processing parameters in key
    echo "${UNSTRUCTURED_IO_CACHE_PREFIX}${file_hash}:${strategy}:${output_format}:${languages}"
}

#######################################
# Get cached result
#######################################
unstructured_io::get_cached() {
    local cache_key="$1"
    
    if ! unstructured_io::cache_available; then
        return 1
    fi
    
    # Get from Redis
    local cached_result
    if command -v docker &> /dev/null; then
        cached_result=$(docker exec vrooli-redis-resource redis-cli GET "$cache_key" 2>/dev/null)
    elif command -v redis-cli &> /dev/null; then
        cached_result=$(redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" GET "$cache_key" 2>/dev/null)
    else
        return 1
    fi
    
    if [[ -n "$cached_result" && "$cached_result" != "(nil)" ]]; then
        # Update TTL on access
        if command -v docker &> /dev/null; then
            docker exec vrooli-redis-resource redis-cli EXPIRE "$cache_key" "$UNSTRUCTURED_IO_CACHE_TTL" >/dev/null 2>&1
        elif command -v redis-cli &> /dev/null; then
            redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" EXPIRE "$cache_key" "$UNSTRUCTURED_IO_CACHE_TTL" >/dev/null 2>&1
        fi
        
        # Decode from base64
        echo "$cached_result" | base64 -d 2>/dev/null || echo "$cached_result"
        return 0
    else
        return 1
    fi
}

#######################################
# Store result in cache
#######################################
unstructured_io::cache_result() {
    local cache_key="$1"
    local content="$2"
    local ttl="${3:-$UNSTRUCTURED_IO_CACHE_TTL}"
    
    if ! unstructured_io::cache_available; then
        return 1
    fi
    
    # Encode to base64 to handle special characters
    local encoded_content=$(echo "$content" | base64 -w 0 2>/dev/null || echo "$content" | base64 2>/dev/null)
    
    # Store in Redis with TTL
    if command -v docker &> /dev/null; then
        docker exec vrooli-redis-resource redis-cli SETEX "$cache_key" "$ttl" "$encoded_content" >/dev/null 2>&1
    elif command -v redis-cli &> /dev/null; then
        redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" \
            SETEX "$cache_key" "$ttl" "$encoded_content" >/dev/null 2>&1
    else
        return 1
    fi
    
    return $?
}

#######################################
# Clear cache for a specific file
#######################################
unstructured_io::clear_cache() {
    local file="$1"
    
    if ! unstructured_io::cache_available; then
        return 1
    fi
    
    # Clear all cache entries for this file (all strategies/formats)
    local file_hash=""
    if command -v sha256sum &> /dev/null; then
        file_hash=$(sha256sum "$file" 2>/dev/null | cut -d' ' -f1)
    elif command -v md5sum &> /dev/null; then
        file_hash=$(md5sum "$file" 2>/dev/null | cut -d' ' -f1)
    else
        local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
        local file_mtime=$(stat -f%m "$file" 2>/dev/null || stat -c%Y "$file" 2>/dev/null || echo "0")
        file_hash="${file_size}_${file_mtime}"
    fi
    
    # Delete all keys matching this file
    local pattern="${UNSTRUCTURED_IO_CACHE_PREFIX}${file_hash}:*"
    redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" --scan --pattern "$pattern" | \
        xargs -r redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" DEL >/dev/null 2>&1
}

#######################################
# Clear all Unstructured.io cache
#######################################
unstructured_io::clear_all_cache() {
    if ! unstructured_io::cache_available; then
        return 1
    fi
    
    # Delete all Unstructured.io cache keys
    local pattern="${UNSTRUCTURED_IO_CACHE_PREFIX}*"
    local count=0
    
    # Count keys first
    count=$(redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" --scan --pattern "$pattern" | wc -l)
    
    # Delete keys
    redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" --scan --pattern "$pattern" | \
        xargs -r redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" DEL >/dev/null 2>&1
    
    echo "Cleared $count cache entries"
}

#######################################
# Get cache statistics
#######################################
unstructured_io::cache_stats() {
    if ! unstructured_io::cache_available; then
        echo "Cache not available (Redis not running or not accessible)"
        return 1
    fi
    
    local pattern="${UNSTRUCTURED_IO_CACHE_PREFIX}*"
    local count=0
    local total_size=0
    local total_ttl=0
    
    # Get all cache keys
    local keys=$(redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" --scan --pattern "$pattern")
    
    if [[ -z "$keys" ]]; then
        echo "No cached documents found"
        return 0
    fi
    
    # Analyze each key
    while IFS= read -r key; do
        [[ -z "$key" ]] && continue
        ((count++))
        
        # Get memory usage for this key
        local mem=$(redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" MEMORY USAGE "$key" 2>/dev/null || echo "0")
        ((total_size += mem))
        
        # Get TTL
        local ttl=$(redis-cli -h "$UNSTRUCTURED_IO_REDIS_HOST" -p "$UNSTRUCTURED_IO_REDIS_PORT" TTL "$key" 2>/dev/null || echo "0")
        [[ $ttl -gt 0 ]] && ((total_ttl += ttl))
    done <<< "$keys"
    
    # Format size
    local formatted_size=""
    if [[ $total_size -gt 1048576 ]]; then
        formatted_size="$((total_size / 1048576))MB"
    elif [[ $total_size -gt 1024 ]]; then
        formatted_size="$((total_size / 1024))KB"
    else
        formatted_size="${total_size}B"
    fi
    
    # Calculate average TTL
    local avg_ttl=0
    [[ $count -gt 0 ]] && avg_ttl=$((total_ttl / count))
    
    echo "Cache Statistics:"
    echo "================="
    echo "Cached documents: $count"
    echo "Total cache size: $formatted_size"
    echo "Average TTL: ${avg_ttl}s"
    echo "Redis endpoint: ${UNSTRUCTURED_IO_REDIS_HOST}:${UNSTRUCTURED_IO_REDIS_PORT}"
}

# Export functions for subshell availability
export -f unstructured_io::cache_available
export -f unstructured_io::get_cache_key
export -f unstructured_io::get_cached
export -f unstructured_io::cache_result
export -f unstructured_io::clear_cache
export -f unstructured_io::clear_all_cache
export -f unstructured_io::cache_stats