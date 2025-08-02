#!/usr/bin/env bash

# Simplified caching implementation without Redis dependency
# Uses filesystem-based caching for simplicity

# Cache configuration
readonly UNSTRUCTURED_IO_CACHE_DIR="${UNSTRUCTURED_IO_CACHE_DIR:-$HOME/.vrooli/cache/unstructured-io}"
readonly UNSTRUCTURED_IO_CACHE_TTL="${UNSTRUCTURED_IO_CACHE_TTL:-3600}"  # 1 hour default

#######################################
# Initialize cache directory
#######################################
unstructured_io::cache_init() {
    mkdir -p "$UNSTRUCTURED_IO_CACHE_DIR"
}

#######################################
# Check if cache is available
#######################################
unstructured_io::cache_available() {
    # Always available with filesystem cache
    return 0
}

#######################################
# Generate cache key for a file
#######################################
unstructured_io::get_cache_key() {
    local file="$1"
    local strategy="${2:-hi_res}"
    local output_format="${3:-json}"
    local languages="${4:-eng}"
    
    # Get file hash
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
    
    # Create safe filename
    echo "${file_hash}_${strategy}_${output_format}_${languages}"
}

#######################################
# Get cached result
#######################################
unstructured_io::get_cached() {
    local cache_key="$1"
    local cache_file="$UNSTRUCTURED_IO_CACHE_DIR/$cache_key"
    
    # Check if cache file exists
    if [[ ! -f "$cache_file" ]]; then
        return 1
    fi
    
    # Check if cache is expired
    local current_time=$(date +%s)
    local file_mtime=$(stat -f%m "$cache_file" 2>/dev/null || stat -c%Y "$cache_file" 2>/dev/null || echo "0")
    local age=$((current_time - file_mtime))
    
    if [[ $age -gt $UNSTRUCTURED_IO_CACHE_TTL ]]; then
        # Cache expired, remove it
        rm -f "$cache_file"
        return 1
    fi
    
    # Return cached content
    cat "$cache_file"
    return 0
}

#######################################
# Store result in cache
#######################################
unstructured_io::cache_result() {
    local cache_key="$1"
    local content="$2"
    
    # Initialize cache directory if needed
    unstructured_io::cache_init
    
    # Store content in cache file
    echo "$content" > "$UNSTRUCTURED_IO_CACHE_DIR/$cache_key"
    return $?
}

#######################################
# Clear cache for a specific file
#######################################
unstructured_io::clear_cache() {
    local file="$1"
    
    # Generate all possible cache keys for this file
    for strategy in fast hi_res auto; do
        for format in json markdown text elements; do
            for lang in eng fra deu; do
                local cache_key=$(unstructured_io::get_cache_key "$file" "$strategy" "$format" "$lang")
                rm -f "$UNSTRUCTURED_IO_CACHE_DIR/$cache_key" 2>/dev/null
            done
        done
    done
    
    echo "Cleared cache entries for: $file"
}

#######################################
# Clear all cache
#######################################
unstructured_io::clear_all_cache() {
    local count=0
    
    if [[ -d "$UNSTRUCTURED_IO_CACHE_DIR" ]]; then
        count=$(find "$UNSTRUCTURED_IO_CACHE_DIR" -type f | wc -l)
        rm -rf "$UNSTRUCTURED_IO_CACHE_DIR"/*
    fi
    
    echo "Cleared $count cache entries"
}

#######################################
# Get cache statistics
#######################################
unstructured_io::cache_stats() {
    if [[ ! -d "$UNSTRUCTURED_IO_CACHE_DIR" ]]; then
        echo "No cache directory found"
        return 0
    fi
    
    local count=0
    local total_size=0
    local expired_count=0
    local current_time=$(date +%s)
    
    # Analyze cache files
    while IFS= read -r -d '' cache_file; do
        ((count++))
        
        # Get file size
        local size=$(stat -f%z "$cache_file" 2>/dev/null || stat -c%s "$cache_file" 2>/dev/null || echo "0")
        ((total_size += size))
        
        # Check if expired
        local file_mtime=$(stat -f%m "$cache_file" 2>/dev/null || stat -c%Y "$cache_file" 2>/dev/null || echo "0")
        local age=$((current_time - file_mtime))
        
        if [[ $age -gt $UNSTRUCTURED_IO_CACHE_TTL ]]; then
            ((expired_count++))
        fi
    done < <(find "$UNSTRUCTURED_IO_CACHE_DIR" -type f -print0 2>/dev/null)
    
    # Format size
    local formatted_size=""
    if [[ $total_size -gt 1048576 ]]; then
        formatted_size="$((total_size / 1048576))MB"
    elif [[ $total_size -gt 1024 ]]; then
        formatted_size="$((total_size / 1024))KB"
    else
        formatted_size="${total_size}B"
    fi
    
    echo "Cache Statistics:"
    echo "================="
    echo "Cache directory: $UNSTRUCTURED_IO_CACHE_DIR"
    echo "Cached documents: $count"
    echo "Expired entries: $expired_count"
    echo "Total cache size: $formatted_size"
    echo "Cache TTL: ${UNSTRUCTURED_IO_CACHE_TTL}s"
}

# Export functions
export -f unstructured_io::cache_init
export -f unstructured_io::cache_available
export -f unstructured_io::get_cache_key
export -f unstructured_io::get_cached
export -f unstructured_io::cache_result
export -f unstructured_io::clear_cache
export -f unstructured_io::clear_all_cache
export -f unstructured_io::cache_stats