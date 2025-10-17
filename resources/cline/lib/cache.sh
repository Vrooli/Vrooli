#!/usr/bin/env bash
################################################################################
# Cline Cache Management - Performance optimization with response caching
# 
# Provides caching functionality for frequently used queries and responses
#
################################################################################

set -euo pipefail

# Cache directory
export CLINE_CACHE_DIR="${CLINE_DATA_DIR:-${HOME}/.cline/data}/cache"

# Cache configuration
export CLINE_CACHE_MAX_SIZE="${CLINE_CACHE_MAX_SIZE:-100}"  # MB
export CLINE_CACHE_TTL="${CLINE_CACHE_TTL:-3600}"  # seconds (1 hour)
export CLINE_CACHE_ENABLED="${CLINE_CACHE_ENABLED:-true}"

# Cache stats file
export CLINE_CACHE_STATS="${CLINE_CACHE_DIR}/stats.json"

#######################################
# Initialize cache directory
#######################################
cline::cache::init() {
    if [[ ! -d "$CLINE_CACHE_DIR" ]]; then
        mkdir -p "$CLINE_CACHE_DIR"
        
        # Initialize stats
        cat > "$CLINE_CACHE_STATS" << EOF
{
    "hits": 0,
    "misses": 0,
    "size_bytes": 0,
    "entries": 0,
    "created": "$(date -Iseconds)",
    "last_cleanup": "$(date -Iseconds)"
}
EOF
    fi
}

#######################################
# Show cache statistics
#######################################
cline::cache::stats() {
    cline::cache::init
    
    log::header "📊 Cline Cache Statistics"
    
    if [[ -f "$CLINE_CACHE_STATS" ]]; then
        local hits=$(jq -r '.hits // 0' "$CLINE_CACHE_STATS")
        local misses=$(jq -r '.misses // 0' "$CLINE_CACHE_STATS")
        local size_bytes=$(jq -r '.size_bytes // 0' "$CLINE_CACHE_STATS")
        local entries=$(jq -r '.entries // 0' "$CLINE_CACHE_STATS")
        local created=$(jq -r '.created // "unknown"' "$CLINE_CACHE_STATS")
        local last_cleanup=$(jq -r '.last_cleanup // "never"' "$CLINE_CACHE_STATS")
        
        # Calculate hit rate
        local total=$((hits + misses))
        local hit_rate=0
        if [[ $total -gt 0 ]]; then
            hit_rate=$((hits * 100 / total))
        fi
        
        # Convert size to human readable
        local size_mb=$((size_bytes / 1024 / 1024))
        
        echo "  Cache Status: $(cline::cache::is_enabled && echo "✅ Enabled" || echo "❌ Disabled")"
        echo "  Cache Directory: $CLINE_CACHE_DIR"
        echo ""
        echo "  Statistics:"
        echo "    • Hits: $hits"
        echo "    • Misses: $misses"
        echo "    • Hit Rate: ${hit_rate}%"
        echo "    • Entries: $entries"
        echo "    • Size: ${size_mb}MB / ${CLINE_CACHE_MAX_SIZE}MB"
        echo ""
        echo "  Maintenance:"
        echo "    • Created: $created"
        echo "    • Last Cleanup: $last_cleanup"
        echo "    • TTL: ${CLINE_CACHE_TTL}s"
    else
        log::warn "No cache statistics available"
    fi
    
    # Show recent cache entries
    echo ""
    log::info "Recent cache entries:"
    find "$CLINE_CACHE_DIR" -name "*.cache" -type f -printf '%T@ %p\n' 2>/dev/null | \
        sort -rn | head -5 | while read -r timestamp file; do
        local name=$(basename "$file" .cache)
        local age=$(( $(date +%s) - ${timestamp%.*} ))
        echo "    • $name ($(cline::cache::format_age $age) ago)"
    done
}

#######################################
# Format age in human readable form
#######################################
cline::cache::format_age() {
    local seconds="${1:-0}"
    
    if [[ $seconds -lt 60 ]]; then
        echo "${seconds}s"
    elif [[ $seconds -lt 3600 ]]; then
        echo "$((seconds / 60))m"
    elif [[ $seconds -lt 86400 ]]; then
        echo "$((seconds / 3600))h"
    else
        echo "$((seconds / 86400))d"
    fi
}

#######################################
# Clear cache
#######################################
cline::cache::clear() {
    log::info "Clearing Cline cache..."
    
    if [[ -d "$CLINE_CACHE_DIR" ]]; then
        local count=$(find "$CLINE_CACHE_DIR" -name "*.cache" -type f | wc -l)
        rm -rf "${CLINE_CACHE_DIR:?}"/*.cache
        
        # Reset stats
        cat > "$CLINE_CACHE_STATS" << EOF
{
    "hits": 0,
    "misses": 0,
    "size_bytes": 0,
    "entries": 0,
    "created": "$(date -Iseconds)",
    "last_cleanup": "$(date -Iseconds)"
}
EOF
        
        log::success "✓ Cleared $count cache entries"
    else
        log::info "Cache directory does not exist"
    fi
}

#######################################
# Enable cache
#######################################
cline::cache::enable() {
    log::info "Enabling Cline cache..."
    echo "CLINE_CACHE_ENABLED=true" > "${CLINE_CONFIG_DIR}/cache.conf"
    export CLINE_CACHE_ENABLED=true
    cline::cache::init
    log::success "✓ Cache enabled"
}

#######################################
# Disable cache
#######################################
cline::cache::disable() {
    log::info "Disabling Cline cache..."
    echo "CLINE_CACHE_ENABLED=false" > "${CLINE_CONFIG_DIR}/cache.conf"
    export CLINE_CACHE_ENABLED=false
    log::success "✓ Cache disabled"
}

#######################################
# Check if cache is enabled
#######################################
cline::cache::is_enabled() {
    [[ "${CLINE_CACHE_ENABLED}" == "true" ]]
}

#######################################
# Get cache key from query
#######################################
cline::cache::get_key() {
    local query="${1:-}"
    local provider="${2:-$(cline::get_provider)}"
    local model="${3:-$(cline::get_model)}"
    
    # Create a unique key from query + provider + model
    echo -n "${provider}_${model}_${query}" | md5sum | cut -d' ' -f1
}

#######################################
# Store response in cache
#######################################
cline::cache::put() {
    local key="${1:-}"
    local response="${2:-}"
    
    if ! cline::cache::is_enabled; then
        return 0
    fi
    
    cline::cache::init
    
    # Clean old entries if needed
    cline::cache::cleanup
    
    local cache_file="${CLINE_CACHE_DIR}/${key}.cache"
    local meta_file="${CLINE_CACHE_DIR}/${key}.meta"
    
    # Store response
    echo "$response" > "$cache_file"
    
    # Store metadata
    cat > "$meta_file" << EOF
{
    "timestamp": $(date +%s),
    "expires": $(($(date +%s) + CLINE_CACHE_TTL)),
    "size": $(stat -c%s "$cache_file" 2>/dev/null || echo 0),
    "provider": "$(cline::get_provider)",
    "model": "$(cline::get_model)"
}
EOF
    
    # Update stats
    cline::cache::update_stats "put"
}

#######################################
# Get response from cache
#######################################
cline::cache::get() {
    local key="${1:-}"
    
    if ! cline::cache::is_enabled; then
        return 1
    fi
    
    local cache_file="${CLINE_CACHE_DIR}/${key}.cache"
    local meta_file="${CLINE_CACHE_DIR}/${key}.meta"
    
    if [[ ! -f "$cache_file" ]] || [[ ! -f "$meta_file" ]]; then
        cline::cache::update_stats "miss"
        return 1
    fi
    
    # Check if expired
    local expires=$(jq -r '.expires // 0' "$meta_file")
    local now=$(date +%s)
    
    if [[ $expires -lt $now ]]; then
        # Expired, remove it
        rm -f "$cache_file" "$meta_file"
        cline::cache::update_stats "miss"
        return 1
    fi
    
    # Return cached response
    cat "$cache_file"
    cline::cache::update_stats "hit"
    return 0
}

#######################################
# Update cache statistics
#######################################
cline::cache::update_stats() {
    local action="${1:-}"
    
    if [[ ! -f "$CLINE_CACHE_STATS" ]]; then
        cline::cache::init
    fi
    
    case "$action" in
        hit)
            jq '.hits += 1' "$CLINE_CACHE_STATS" > "${CLINE_CACHE_STATS}.tmp"
            ;;
        miss)
            jq '.misses += 1' "$CLINE_CACHE_STATS" > "${CLINE_CACHE_STATS}.tmp"
            ;;
        put)
            local entries=$(find "$CLINE_CACHE_DIR" -name "*.cache" | wc -l)
            local size=$(du -sb "$CLINE_CACHE_DIR" | cut -f1)
            jq ".entries = $entries | .size_bytes = $size" "$CLINE_CACHE_STATS" > "${CLINE_CACHE_STATS}.tmp"
            ;;
    esac
    
    if [[ -f "${CLINE_CACHE_STATS}.tmp" ]]; then
        mv "${CLINE_CACHE_STATS}.tmp" "$CLINE_CACHE_STATS"
    fi
}

#######################################
# Cleanup old cache entries
#######################################
cline::cache::cleanup() {
    local now=$(date +%s)
    local cleaned=0
    
    # Remove expired entries
    find "$CLINE_CACHE_DIR" -name "*.meta" -type f | while read -r meta_file; do
        local expires=$(jq -r '.expires // 0' "$meta_file" 2>/dev/null || echo 0)
        if [[ $expires -lt $now ]]; then
            local cache_file="${meta_file%.meta}.cache"
            rm -f "$cache_file" "$meta_file"
            ((cleaned++))
        fi
    done
    
    # Check size limit
    local size=$(du -sb "$CLINE_CACHE_DIR" | cut -f1)
    local max_size=$((CLINE_CACHE_MAX_SIZE * 1024 * 1024))
    
    if [[ $size -gt $max_size ]]; then
        # Remove oldest entries until under limit
        find "$CLINE_CACHE_DIR" -name "*.cache" -type f -printf '%T@ %p\n' | \
            sort -n | while read -r timestamp file; do
            if [[ $size -gt $max_size ]]; then
                local file_size=$(stat -c%s "$file" 2>/dev/null || echo 0)
                rm -f "$file" "${file%.cache}.meta"
                size=$((size - file_size))
                ((cleaned++))
            else
                break
            fi
        done
    fi
    
    if [[ $cleaned -gt 0 ]]; then
        # Update last cleanup time
        jq ".last_cleanup = \"$(date -Iseconds)\"" "$CLINE_CACHE_STATS" > "${CLINE_CACHE_STATS}.tmp"
        mv "${CLINE_CACHE_STATS}.tmp" "$CLINE_CACHE_STATS"
    fi
}

#######################################
# Get current model
#######################################
cline::get_model() {
    # This would normally get from config
    echo "${CLINE_DEFAULT_MODEL:-llama3.2:3b}"
}

# Export functions
export -f cline::cache::init
export -f cline::cache::stats
export -f cline::cache::clear
export -f cline::cache::enable
export -f cline::cache::disable
export -f cline::cache::is_enabled
export -f cline::cache::get_key
export -f cline::cache::put
export -f cline::cache::get
export -f cline::cache::update_stats
export -f cline::cache::cleanup
export -f cline::cache::format_age
export -f cline::get_model