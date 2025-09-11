#!/bin/bash
# Gemini Redis caching functionality

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GEMINI_RESOURCE_DIR="${APP_ROOT}/resources/gemini"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${GEMINI_RESOURCE_DIR}/config/defaults.sh"

# Redis configuration
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
GEMINI_CACHE_TTL="${GEMINI_CACHE_TTL:-3600}"  # Default 1 hour TTL
GEMINI_CACHE_PREFIX="${GEMINI_CACHE_PREFIX:-gemini:cache}"
GEMINI_CACHE_ENABLED="${GEMINI_CACHE_ENABLED:-false}"

# Check if Redis is available
gemini::cache::is_available() {
    # Check if Redis resource is running
    if command -v redis-cli >/dev/null 2>&1; then
        timeout 2 redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping >/dev/null 2>&1
        return $?
    fi
    return 1
}

# Generate cache key from prompt and model
gemini::cache::generate_key() {
    local prompt="$1"
    local model="${2:-$GEMINI_DEFAULT_MODEL}"
    local temperature="${3:-0.7}"
    
    # Create a unique hash for the cache key
    local hash=$(echo -n "${model}:${temperature}:${prompt}" | sha256sum | cut -d' ' -f1)
    echo "${GEMINI_CACHE_PREFIX}:${model}:${hash:0:16}"
}

# Get cached response
gemini::cache::get() {
    local key="$1"
    
    if ! gemini::cache::is_available; then
        return 1
    fi
    
    local cached_value
    cached_value=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" GET "$key" 2>/dev/null)
    
    if [[ -n "$cached_value" && "$cached_value" != "(nil)" ]]; then
        echo "$cached_value"
        return 0
    fi
    
    return 1
}

# Set cached response with TTL
gemini::cache::set() {
    local key="$1"
    local value="$2"
    local ttl="${3:-$GEMINI_CACHE_TTL}"
    
    if ! gemini::cache::is_available; then
        return 1
    fi
    
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" SETEX "$key" "$ttl" "$value" >/dev/null 2>&1
    return $?
}

# Delete cached response
gemini::cache::delete() {
    local key="$1"
    
    if ! gemini::cache::is_available; then
        return 1
    fi
    
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" DEL "$key" >/dev/null 2>&1
    return $?
}

# Clear all Gemini cache entries
gemini::cache::clear_all() {
    if ! gemini::cache::is_available; then
        log::warn "Redis not available for cache clearing"
        return 1
    fi
    
    # Find and delete all keys with Gemini cache prefix
    local keys
    keys=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" KEYS "${GEMINI_CACHE_PREFIX}:*" 2>/dev/null)
    
    if [[ -n "$keys" ]]; then
        echo "$keys" | xargs -r redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" DEL >/dev/null 2>&1
        log::info "Cleared Gemini cache entries"
    else
        log::info "No cache entries to clear"
    fi
    
    return 0
}

# Get cache statistics
gemini::cache::stats() {
    if ! gemini::cache::is_available; then
        echo "Cache Status: Disabled (Redis not available)"
        return 1
    fi
    
    local count
    count=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" --scan --pattern "${GEMINI_CACHE_PREFIX}:*" 2>/dev/null | wc -l)
    
    echo "Cache Status: Enabled"
    echo "Redis Host: ${REDIS_HOST}:${REDIS_PORT}"
    echo "Cache Prefix: ${GEMINI_CACHE_PREFIX}"
    echo "Default TTL: ${GEMINI_CACHE_TTL} seconds"
    echo "Cached Entries: ${count}"
    
    return 0
}

# Export functions
export -f gemini::cache::is_available
export -f gemini::cache::generate_key
export -f gemini::cache::get
export -f gemini::cache::set
export -f gemini::cache::delete
export -f gemini::cache::clear_all
export -f gemini::cache::stats