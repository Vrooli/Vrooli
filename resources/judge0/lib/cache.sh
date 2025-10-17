#!/usr/bin/env bash
# Judge0 Result Caching Module
# Implements caching for identical code submissions to improve performance

# Source shared utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/judge0/lib/common.sh"

# Cache configuration
export JUDGE0_CACHE_DIR="${JUDGE0_DATA_DIR}/cache"
export JUDGE0_CACHE_TTL="${JUDGE0_CACHE_TTL:-3600}"  # 1 hour default TTL
export JUDGE0_CACHE_MAX_SIZE="${JUDGE0_CACHE_MAX_SIZE:-1000}"  # Max cached entries

#######################################
# Initialize cache directory
#######################################
judge0::cache::init() {
    if [[ ! -d "$JUDGE0_CACHE_DIR" ]]; then
        mkdir -p "$JUDGE0_CACHE_DIR"
    fi
    
    # Create cache index if it doesn't exist
    local index_file="${JUDGE0_CACHE_DIR}/index.json"
    if [[ ! -f "$index_file" ]]; then
        echo '{"entries": {}, "stats": {"hits": 0, "misses": 0, "evictions": 0}}' > "$index_file"
    fi
}

#######################################
# Generate cache key for submission
# Arguments:
#   $1 - Source code
#   $2 - Language ID
#   $3 - Input data
#   $4 - Expected output (optional)
# Outputs:
#   Cache key (hash)
#######################################
judge0::cache::generate_key() {
    local code="$1"
    local lang_id="$2"
    local input="$3"
    local expected="${4:-}"
    
    # Create deterministic key from submission parameters
    local key_data="${code}|${lang_id}|${input}|${expected}"
    local cache_key=$(echo -n "$key_data" | sha256sum | cut -d' ' -f1)
    
    echo "$cache_key"
}

#######################################
# Check if result exists in cache
# Arguments:
#   $1 - Cache key
# Outputs:
#   Cached result if found, empty otherwise
#######################################
judge0::cache::get() {
    local cache_key="$1"
    
    judge0::cache::init
    
    local index_file="${JUDGE0_CACHE_DIR}/index.json"
    local cache_file="${JUDGE0_CACHE_DIR}/${cache_key}.json"
    
    # Check if cache file exists
    if [[ ! -f "$cache_file" ]]; then
        # Update miss counter
        local index=$(cat "$index_file")
        index=$(echo "$index" | jq '.stats.misses += 1')
        echo "$index" > "$index_file"
        return 1
    fi
    
    # Check if cache entry is expired
    local entry=$(cat "$cache_file")
    local created=$(echo "$entry" | jq -r '.created')
    local now=$(date +%s)
    local age=$((now - created))
    
    if [[ $age -gt $JUDGE0_CACHE_TTL ]]; then
        # Cache expired, remove entry
        judge0::cache::evict "$cache_key"
        return 1
    fi
    
    # Update hit counter and last accessed time
    local index=$(cat "$index_file")
    index=$(echo "$index" | jq \
        --arg key "$cache_key" \
        --arg now "$now" \
        '.stats.hits += 1 | .entries[$key].last_accessed = ($now | tonumber)')
    echo "$index" > "$index_file"
    
    # Return cached result
    echo "$entry" | jq -r '.result'
    return 0
}

#######################################
# Store result in cache
# Arguments:
#   $1 - Cache key
#   $2 - Result data (JSON)
# Outputs:
#   Storage status
#######################################
judge0::cache::set() {
    local cache_key="$1"
    local result="$2"
    
    judge0::cache::init
    
    # Check cache size limit
    judge0::cache::check_size_limit
    
    local index_file="${JUDGE0_CACHE_DIR}/index.json"
    local cache_file="${JUDGE0_CACHE_DIR}/${cache_key}.json"
    local now=$(date +%s)
    
    # Create cache entry
    local entry=$(jq -n \
        --arg created "$now" \
        --argjson result "$result" \
        '{
            created: ($created | tonumber),
            result: $result
        }')
    
    echo "$entry" > "$cache_file"
    
    # Update index
    local index=$(cat "$index_file")
    index=$(echo "$index" | jq \
        --arg key "$cache_key" \
        --arg now "$now" \
        '.entries[$key] = {
            created: ($now | tonumber),
            last_accessed: ($now | tonumber),
            size: 1
        }')
    echo "$index" > "$index_file"
    
    return 0
}

#######################################
# Submit with caching
# Arguments:
#   $1 - Source code
#   $2 - Language
#   $3 - Input data (optional)
#   $4 - Expected output (optional)
# Outputs:
#   Submission result (from cache or new execution)
#######################################
judge0::cache::submit_with_cache() {
    local code="$1"
    local language="$2"
    local input="${3:-}"
    local expected="${4:-}"
    
    # Get language ID
    local lang_id=$(judge0::get_language_id "$language")
    if [[ -z "$lang_id" ]]; then
        log::error "Unsupported language: $language"
        return 1
    fi
    
    # Generate cache key
    local cache_key=$(judge0::cache::generate_key "$code" "$lang_id" "$input" "$expected")
    
    # Check cache first
    local cached_result=$(judge0::cache::get "$cache_key")
    if [[ -n "$cached_result" ]]; then
        log::info "Cache hit: Using cached result"
        echo "$cached_result"
        return 0
    fi
    
    log::info "Cache miss: Executing code"
    
    # Submit to Judge0
    local result=$(judge0::api::submit "$code" "$language" "$input" "$expected")
    if [[ -z "$result" ]]; then
        log::error "Failed to execute code"
        return 1
    fi
    
    # Store in cache if successful
    local status=$(echo "$result" | jq -r '.status.id')
    if [[ "$status" == "3" ]]; then  # Status 3 = Accepted
        judge0::cache::set "$cache_key" "$result"
        log::info "Result cached for future use"
    fi
    
    echo "$result"
    return 0
}

#######################################
# Evict cache entry
# Arguments:
#   $1 - Cache key
#######################################
judge0::cache::evict() {
    local cache_key="$1"
    
    local index_file="${JUDGE0_CACHE_DIR}/index.json"
    local cache_file="${JUDGE0_CACHE_DIR}/${cache_key}.json"
    
    # Remove cache file
    rm -f "$cache_file"
    
    # Update index
    local index=$(cat "$index_file")
    index=$(echo "$index" | jq \
        --arg key "$cache_key" \
        'del(.entries[$key]) | .stats.evictions += 1')
    echo "$index" > "$index_file"
}

#######################################
# Clear entire cache
#######################################
judge0::cache::clear() {
    log::info "Clearing cache..."
    
    judge0::cache::init
    
    # Remove all cache files
    find "$JUDGE0_CACHE_DIR" -name "*.json" -type f ! -name "index.json" -delete
    
    # Reset index
    echo '{"entries": {}, "stats": {"hits": 0, "misses": 0, "evictions": 0}}' > "${JUDGE0_CACHE_DIR}/index.json"
    
    log::success "Cache cleared"
}

#######################################
# Show cache statistics
#######################################
judge0::cache::stats() {
    judge0::cache::init
    
    local index_file="${JUDGE0_CACHE_DIR}/index.json"
    local index=$(cat "$index_file")
    
    log::header "Cache Statistics"
    
    # Extract stats
    local hits=$(echo "$index" | jq -r '.stats.hits')
    local misses=$(echo "$index" | jq -r '.stats.misses')
    local evictions=$(echo "$index" | jq -r '.stats.evictions')
    local entries=$(echo "$index" | jq -r '.entries | length')
    
    # Calculate hit ratio
    local total=$((hits + misses))
    local hit_ratio="0.00"
    if [[ $total -gt 0 ]]; then
        hit_ratio=$(echo "scale=2; $hits * 100 / $total" | bc)
    fi
    
    echo "📊 Cache Performance:"
    echo "  • Hits: $hits"
    echo "  • Misses: $misses"
    echo "  • Hit Ratio: ${hit_ratio}%"
    echo "  • Evictions: $evictions"
    echo "  • Current Entries: $entries / $JUDGE0_CACHE_MAX_SIZE"
    echo "  • TTL: ${JUDGE0_CACHE_TTL}s"
    
    # Show top cached entries by access
    if [[ $entries -gt 0 ]]; then
        echo ""
        echo "📈 Most Accessed Entries:"
        echo "$index" | jq -r '.entries | to_entries | sort_by(.value.last_accessed) | reverse | limit(5;.[]) | "  • \(.key[0:8])... - Last: \(.value.last_accessed | todate)"'
    fi
}

#######################################
# Check and enforce cache size limit
#######################################
judge0::cache::check_size_limit() {
    local index_file="${JUDGE0_CACHE_DIR}/index.json"
    local index=$(cat "$index_file")
    local entries=$(echo "$index" | jq -r '.entries | length')
    
    if [[ $entries -ge $JUDGE0_CACHE_MAX_SIZE ]]; then
        # Evict oldest entry (LRU)
        local oldest_key=$(echo "$index" | jq -r '.entries | to_entries | min_by(.value.last_accessed) | .key')
        if [[ -n "$oldest_key" ]]; then
            judge0::cache::evict "$oldest_key"
            log::info "Evicted oldest cache entry: ${oldest_key:0:8}..."
        fi
    fi
}

#######################################
# Warm cache with common submissions
# Arguments:
#   $1 - File with test cases (optional)
#######################################
judge0::cache::warm() {
    local test_file="${1:-${JUDGE0_CONFIG_DIR}/common_tests.json}"
    
    log::header "Warming Cache"
    
    if [[ ! -f "$test_file" ]]; then
        # Create default test cases
        cat > "$test_file" << 'EOF'
[
  {
    "name": "Hello World - Python",
    "code": "print('Hello, World!')",
    "language": "python",
    "input": "",
    "expected": "Hello, World!\n"
  },
  {
    "name": "Hello World - JavaScript",
    "code": "console.log('Hello, World!');",
    "language": "javascript",
    "input": "",
    "expected": "Hello, World!\n"
  },
  {
    "name": "Simple Math - Python",
    "code": "print(2 + 2)",
    "language": "python",
    "input": "",
    "expected": "4\n"
  },
  {
    "name": "Input Echo - Python",
    "code": "print(input())",
    "language": "python",
    "input": "test",
    "expected": "test\n"
  }
]
EOF
    fi
    
    # Process test cases
    local count=0
    while IFS= read -r test; do
        local name=$(echo "$test" | jq -r '.name')
        local code=$(echo "$test" | jq -r '.code')
        local language=$(echo "$test" | jq -r '.language')
        local input=$(echo "$test" | jq -r '.input')
        local expected=$(echo "$test" | jq -r '.expected')
        
        log::info "Warming: $name"
        judge0::cache::submit_with_cache "$code" "$language" "$input" "$expected" >/dev/null 2>&1
        ((count++))
    done < <(jq -c '.[]' "$test_file")
    
    log::success "Cache warmed with $count entries"
}

#######################################
# Export cache for backup/migration
# Arguments:
#   $1 - Export file path
#######################################
judge0::cache::export() {
    local export_file="${1:-${JUDGE0_DATA_DIR}/cache_export_$(date +%Y%m%d_%H%M%S).tar.gz}"
    
    judge0::cache::init
    
    log::info "Exporting cache to: $export_file"
    
    tar -czf "$export_file" -C "$JUDGE0_CACHE_DIR" .
    
    local size=$(du -h "$export_file" | cut -f1)
    log::success "Cache exported: $export_file ($size)"
}

#######################################
# Import cache from backup
# Arguments:
#   $1 - Import file path
#######################################
judge0::cache::import() {
    local import_file="$1"
    
    if [[ ! -f "$import_file" ]]; then
        log::error "Import file not found: $import_file"
        return 1
    fi
    
    judge0::cache::init
    
    log::info "Importing cache from: $import_file"
    
    # Backup current cache
    judge0::cache::export "${JUDGE0_DATA_DIR}/cache_backup_before_import.tar.gz"
    
    # Clear current cache
    judge0::cache::clear
    
    # Import new cache
    tar -xzf "$import_file" -C "$JUDGE0_CACHE_DIR"
    
    log::success "Cache imported successfully"
    judge0::cache::stats
}