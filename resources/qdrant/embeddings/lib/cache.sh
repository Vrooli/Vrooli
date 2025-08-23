#!/usr/bin/env bash
# Resource location caching for improved performance
# Optimized for flatter directory structure

set -euo pipefail

# Cache file location
CACHE_DIR="${HOME}/.cache/qdrant-embeddings"
RESOURCE_CACHE_FILE="${CACHE_DIR}/resource-locations.cache"
CACHE_TTL=3600  # 1 hour in seconds

# Ensure cache directory exists
mkdir -p "$CACHE_DIR"

#######################################
# Check if cache is valid
# Returns: 0 if valid, 1 if expired or missing
#######################################
qdrant::cache::is_valid() {
    local cache_file="$1"
    
    if [[ ! -f "$cache_file" ]]; then
        return 1
    fi
    
    local cache_age=$(($(date +%s) - $(stat -f%m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null)))
    
    if [[ $cache_age -gt $CACHE_TTL ]]; then
        return 1
    fi
    
    return 0
}

#######################################
# Cache resource locations
# Arguments:
#   $1 - Directory to scan
# Returns: 0 on success
#######################################
qdrant::cache::resources() {
    local dir="${1:-.}"
    
    # Clear old cache
    rm -f "$RESOURCE_CACHE_FILE"
    
    # Cache new structure resources
    if [[ -d "$dir/resources" ]]; then
        find "$dir/resources" -mindepth 1 -maxdepth 1 -type d -exec test -f {}/cli.sh \; -print > "$RESOURCE_CACHE_FILE"
    fi
    
    # Also check old structure if needed (backwards compatibility)
    if [[ ! -s "$RESOURCE_CACHE_FILE" ]] && [[ -d "$dir/scripts/resources" ]]; then
        find "$dir/scripts/resources" -mindepth 2 -maxdepth 3 -type d -exec test -f {}/cli.sh \; -print > "$RESOURCE_CACHE_FILE"
    fi
    
    local count=$(wc -l < "$RESOURCE_CACHE_FILE")
    echo "Cached $count resource locations"
    
    return 0
}

#######################################
# Get cached resource locations
# Returns: List of resource directories
#######################################
qdrant::cache::get_resources() {
    if qdrant::cache::is_valid "$RESOURCE_CACHE_FILE"; then
        cat "$RESOURCE_CACHE_FILE"
    else
        # Rebuild cache
        qdrant::cache::resources "." >&2
        cat "$RESOURCE_CACHE_FILE"
    fi
}

#######################################
# Clear all caches
# Returns: 0 on success
#######################################
qdrant::cache::clear() {
    rm -rf "$CACHE_DIR"
    mkdir -p "$CACHE_DIR"
    echo "Cache cleared"
    return 0
}

#######################################
# Get cache statistics
# Returns: Cache info
#######################################
qdrant::cache::stats() {
    echo "=== Cache Statistics ==="
    
    if [[ -f "$RESOURCE_CACHE_FILE" ]]; then
        local size=$(du -h "$RESOURCE_CACHE_FILE" | cut -f1)
        local age=$(($(date +%s) - $(stat -f%m "$RESOURCE_CACHE_FILE" 2>/dev/null || stat -c %Y "$RESOURCE_CACHE_FILE" 2>/dev/null)))
        local entries=$(wc -l < "$RESOURCE_CACHE_FILE")
        
        echo "Resource cache:"
        echo "  Size: $size"
        echo "  Age: ${age}s"
        echo "  Entries: $entries"
        echo "  Valid: $(qdrant::cache::is_valid "$RESOURCE_CACHE_FILE" && echo "yes" || echo "no")"
    else
        echo "No cache found"
    fi
    
    return 0
}