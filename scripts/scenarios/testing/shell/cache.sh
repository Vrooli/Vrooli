#!/usr/bin/env bash
# Test result caching utilities for faster test iterations
set -euo pipefail

# Global cache configuration
TESTING_CACHE_DIR=""
TESTING_CACHE_ENABLED=true
TESTING_CACHE_DEFAULT_TTL=3600  # 1 hour default
TESTING_CACHE_CHECK_GIT=true
TESTING_CACHE_VERBOSE=false

# Phase-specific cache settings
declare -A TESTING_CACHE_PHASE_TTL=()
declare -A TESTING_CACHE_PHASE_KEYS=()
declare -A TESTING_CACHE_PHASE_ENABLED=()

# Configure cache settings
# Usage: testing::cache::configure [options]
# Options:
#   --dir PATH              Cache directory (required)
#   --enabled BOOL          Enable caching globally (default: true)
#   --default-ttl SECONDS   Default TTL for cache entries (default: 3600)
#   --check-git BOOL        Include git hash in cache keys (default: true)
#   --verbose BOOL          Verbose cache operations (default: false)
testing::cache::configure() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                TESTING_CACHE_DIR="$2"
                shift 2
                ;;
            --enabled)
                TESTING_CACHE_ENABLED="$2"
                shift 2
                ;;
            --default-ttl)
                TESTING_CACHE_DEFAULT_TTL="$2"
                shift 2
                ;;
            --check-git)
                TESTING_CACHE_CHECK_GIT="$2"
                shift 2
                ;;
            --verbose)
                TESTING_CACHE_VERBOSE="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::cache::configure: $1" >&2
                return 1
                ;;
        esac
    done
    
    if [ -z "$TESTING_CACHE_DIR" ]; then
        echo "ERROR: --dir is required for cache configuration" >&2
        return 1
    fi
    
    mkdir -p "$TESTING_CACHE_DIR/.test-cache"
    TESTING_CACHE_DIR="$TESTING_CACHE_DIR/.test-cache"
}

# Configure caching for a specific phase
# Usage: testing::cache::configure_phase PHASE [options]
# Options:
#   --ttl SECONDS          TTL for this phase's cache (0 = no cache)
#   --key-from FILES       Comma-separated list of files to include in cache key
#   --enabled BOOL         Enable caching for this phase
testing::cache::configure_phase() {
    local phase="$1"
    shift
    
    local ttl="$TESTING_CACHE_DEFAULT_TTL"
    local key_from=""
    local enabled=true
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --ttl)
                ttl="$2"
                shift 2
                ;;
            --key-from)
                key_from="$2"
                shift 2
                ;;
            --enabled)
                enabled="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::cache::configure_phase: $1" >&2
                return 1
                ;;
        esac
    done
    
    TESTING_CACHE_PHASE_TTL["$phase"]="$ttl"
    TESTING_CACHE_PHASE_KEYS["$phase"]="$key_from"
    TESTING_CACHE_PHASE_ENABLED["$phase"]="$enabled"
}

# Generate cache key for a phase
# Usage: testing::cache::generate_key PHASE [EXTRA_FILES...]
testing::cache::generate_key() {
    local phase="$1"
    shift
    local extra_files=("$@")
    
    local key_parts=("phase:$phase")
    
    # Add git commit hash if available and enabled
    if [ "$TESTING_CACHE_CHECK_GIT" = "true" ] && command -v git >/dev/null 2>&1; then
        local git_hash=$(git rev-parse HEAD 2>/dev/null || echo "no-git")
        key_parts+=("git:$git_hash")
    fi
    
    # Add configured files for this phase
    local phase_files="${TESTING_CACHE_PHASE_KEYS[$phase]:-}"
    if [ -n "$phase_files" ]; then
        IFS=',' read -ra files <<< "$phase_files"
        for file in "${files[@]}"; do
            file=$(echo "$file" | xargs)  # Trim whitespace
            if [ -f "$file" ]; then
                local checksum=$(testing::cache::checksum_file "$file")
                key_parts+=("file:$(basename "$file"):$checksum")
            fi
        done
    fi
    
    # Add extra files
    for file in "${extra_files[@]}"; do
        if [ -f "$file" ]; then
            local checksum=$(testing::cache::checksum_file "$file")
            key_parts+=("file:$(basename "$file"):$checksum")
        fi
    done
    
    # Generate final key hash
    local key_string="${key_parts[*]}"
    echo "$key_string" | sha256sum | cut -d' ' -f1
}

# Calculate checksum of a file
testing::cache::checksum_file() {
    local file="$1"
    
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" 2>/dev/null | cut -d' ' -f1 || echo "no-checksum"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" 2>/dev/null | cut -d' ' -f1 || echo "no-checksum"
    else
        # Fallback to modification time
        stat -c %Y "$file" 2>/dev/null || stat -f %m "$file" 2>/dev/null || echo "0"
    fi
}

# Check if a cached result exists and is valid
# Usage: testing::cache::is_valid PHASE CACHE_KEY
testing::cache::is_valid() {
    local phase="$1"
    local cache_key="$2"
    
    if [ "$TESTING_CACHE_ENABLED" != "true" ]; then
        return 1
    fi
    
    local phase_enabled="${TESTING_CACHE_PHASE_ENABLED[$phase]:-true}"
    if [ "$phase_enabled" != "true" ]; then
        return 1
    fi
    
    local cache_file="$TESTING_CACHE_DIR/${phase}/${cache_key}.cache"
    if [ ! -f "$cache_file" ]; then
        [ "$TESTING_CACHE_VERBOSE" = "true" ] && echo "  Cache miss: no cache file" >&2
        return 1
    fi
    
    # Check TTL
    local ttl="${TESTING_CACHE_PHASE_TTL[$phase]:-$TESTING_CACHE_DEFAULT_TTL}"
    if [ "$ttl" -eq 0 ]; then
        [ "$TESTING_CACHE_VERBOSE" = "true" ] && echo "  Cache disabled: TTL is 0" >&2
        return 1
    fi
    
    local cache_age=$(($(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || stat -f %m "$cache_file" 2>/dev/null || echo 0)))
    if [ $cache_age -gt $ttl ]; then
        [ "$TESTING_CACHE_VERBOSE" = "true" ] && echo "  Cache expired: ${cache_age}s > ${ttl}s" >&2
        return 1
    fi
    
    [ "$TESTING_CACHE_VERBOSE" = "true" ] && echo "  Cache hit: valid for $((ttl - cache_age))s more" >&2
    return 0
}

# Store test result in cache
# Usage: testing::cache::store PHASE CACHE_KEY EXIT_CODE DURATION LOG_FILE
testing::cache::store() {
    local phase="$1"
    local cache_key="$2"
    local exit_code="$3"
    local duration="$4"
    local log_file="$5"
    
    if [ "$TESTING_CACHE_ENABLED" != "true" ]; then
        return 0
    fi
    
    local phase_enabled="${TESTING_CACHE_PHASE_ENABLED[$phase]:-true}"
    if [ "$phase_enabled" != "true" ]; then
        return 0
    fi
    
    local cache_dir="$TESTING_CACHE_DIR/${phase}"
    mkdir -p "$cache_dir"
    
    local cache_file="$cache_dir/${cache_key}.cache"
    local meta_file="$cache_dir/${cache_key}.meta"
    
    # Store metadata
    cat > "$meta_file" <<EOF
{
  "phase": "$phase",
  "exit_code": $exit_code,
  "duration": $duration,
  "timestamp": $(date +%s),
  "date": "$(date -Iseconds)",
  "cache_key": "$cache_key"
}
EOF
    
    # Store log output
    if [ -f "$log_file" ]; then
        cp "$log_file" "$cache_file"
    else
        echo "No log file available" > "$cache_file"
    fi
    
    [ "$TESTING_CACHE_VERBOSE" = "true" ] && echo "  Cached result: exit=$exit_code, duration=${duration}s" >&2
}

# Retrieve cached result
# Usage: testing::cache::get PHASE CACHE_KEY
# Returns: Sets CACHE_EXIT_CODE, CACHE_DURATION, CACHE_LOG_FILE variables
testing::cache::get() {
    local phase="$1"
    local cache_key="$2"
    
    local cache_file="$TESTING_CACHE_DIR/${phase}/${cache_key}.cache"
    local meta_file="$TESTING_CACHE_DIR/${phase}/${cache_key}.meta"
    
    if [ ! -f "$cache_file" ] || [ ! -f "$meta_file" ]; then
        return 1
    fi
    
    # Export results for caller
    CACHE_EXIT_CODE=$(jq -r '.exit_code' "$meta_file" 2>/dev/null || echo 1)
    CACHE_DURATION=$(jq -r '.duration' "$meta_file" 2>/dev/null || echo 0)
    CACHE_LOG_FILE="$cache_file"
    CACHE_TIMESTAMP=$(jq -r '.timestamp' "$meta_file" 2>/dev/null || echo 0)
    
    return 0
}

# Show cache statistics
# Usage: testing::cache::stats
testing::cache::stats() {
    if [ -z "$TESTING_CACHE_DIR" ] || [ ! -d "$TESTING_CACHE_DIR" ]; then
        echo "No cache directory configured"
        return 0
    fi
    
    echo "📊 Test Cache Statistics"
    echo "════════════════════════════════════════"
    echo "Directory: $TESTING_CACHE_DIR"
    echo "Enabled: $TESTING_CACHE_ENABLED"
    echo "Default TTL: ${TESTING_CACHE_DEFAULT_TTL}s"
    echo ""
    
    local total_entries=0
    local total_size=0
    
    for phase_dir in "$TESTING_CACHE_DIR"/*/; do
        if [ -d "$phase_dir" ]; then
            local phase=$(basename "$phase_dir")
            local count=$(find "$phase_dir" -name "*.cache" 2>/dev/null | wc -l || echo 0)
            local size=$(du -sh "$phase_dir" 2>/dev/null | cut -f1 || echo "0")
            
            if [ $count -gt 0 ]; then
                local ttl="${TESTING_CACHE_PHASE_TTL[$phase]:-$TESTING_CACHE_DEFAULT_TTL}"
                echo "  $phase: $count entries, $size, TTL=${ttl}s"
                total_entries=$((total_entries + count))
            fi
        fi
    done
    
    echo ""
    echo "Total entries: $total_entries"
    
    local total_size=$(du -sh "$TESTING_CACHE_DIR" 2>/dev/null | cut -f1 || echo "0")
    echo "Total size: $total_size"
}

# Clean expired cache entries
# Usage: testing::cache::cleanup
testing::cache::cleanup() {
    if [ -z "$TESTING_CACHE_DIR" ] || [ ! -d "$TESTING_CACHE_DIR" ]; then
        return 0
    fi
    
    local cleaned=0
    
    for phase_dir in "$TESTING_CACHE_DIR"/*/; do
        if [ -d "$phase_dir" ]; then
            local phase=$(basename "$phase_dir")
            local ttl="${TESTING_CACHE_PHASE_TTL[$phase]:-$TESTING_CACHE_DEFAULT_TTL}"
            
            if [ "$ttl" -gt 0 ]; then
                # Find and remove expired cache files
                find "$phase_dir" -name "*.cache" -type f -mmin +$((ttl / 60)) -delete 2>/dev/null || true
                find "$phase_dir" -name "*.meta" -type f -mmin +$((ttl / 60)) -delete 2>/dev/null || true
                cleaned=$((cleaned + 1))
            fi
        fi
    done
    
    [ "$TESTING_CACHE_VERBOSE" = "true" ] && echo "Cleaned $cleaned phase caches" >&2
}