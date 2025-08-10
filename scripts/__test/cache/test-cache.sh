#!/usr/bin/env bash
# Shell test result caching system
# Tracks test results to skip recently passed tests and speed up development

set -euo pipefail

# Source hash utilities
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/hash.sh" 2>/dev/null || true

# Cache configuration
CACHE_DIR="${HOME}/.cache/vrooli/test-results"
CACHE_FILE="${CACHE_DIR}/shell-tests.json"
CACHE_TTL_MINUTES="${VROOLI_TEST_CACHE_TTL:-120}"  # Default: 120 minutes (2 hours)
CACHE_ENABLED="${VROOLI_TEST_CACHE_ENABLED:-true}"

# Ensure cache directory exists
mkdir -p "$CACHE_DIR"

# Initialize cache file if it doesn't exist
if [[ ! -f "$CACHE_FILE" ]]; then
    echo '{}' > "$CACHE_FILE"
fi

# Calculate file checksum using hash utility (fallback to modification time if needed)
cache::calculate_checksum() {
    local file="$1"
    
    # Try to use hash utility first
    if command -v hash::compute_file_hash &>/dev/null; then
        hash::compute_file_hash "$file" 2>/dev/null || {
            # Fallback to modification time if hash utility fails
            stat -c %Y "$file" 2>/dev/null || stat -f %m "$file" 2>/dev/null || echo "0"
        }
    else
        # Fallback to modification time if hash utility not available
        stat -c %Y "$file" 2>/dev/null || stat -f %m "$file" 2>/dev/null || echo "0"
    fi
}

# Derive source file path from test file path
cache::get_source_file() {
    local test_file="$1"
    
    # Replace .bats extension with .sh in the same directory
    local source_file="${test_file%.bats}.sh"
    
    # Only return if the source file actually exists
    if [[ -f "$source_file" ]]; then
        echo "$source_file"
    fi
    # Return empty string if source file doesn't exist
}

# Get current timestamp in seconds
cache::timestamp() {
    date +%s
}

# Check if a test result is still valid
cache::is_valid() {
    local test_file="$1"
    local max_age_minutes="${2:-$CACHE_TTL_MINUTES}"
    
    if [[ "$CACHE_ENABLED" != "true" ]]; then
        return 1  # Cache disabled
    fi
    
    if [[ ! -f "$CACHE_FILE" ]]; then
        return 1  # No cache file
    fi
    
    # Get cache entry using jq if available, otherwise use grep/sed
    local cache_entry
    if command -v jq &>/dev/null; then
        cache_entry=$(jq -r ".\"$test_file\" // empty" "$CACHE_FILE")
    else
        # Fallback to basic parsing
        cache_entry=$(grep -o "\"$test_file\":{[^}]*}" "$CACHE_FILE" 2>/dev/null | sed 's/.*://' || echo "")
    fi
    
    if [[ -z "$cache_entry" ]]; then
        return 1  # No cache entry
    fi
    
    # Parse cache entry (support both old and new format)
    local cached_test_checksum cached_source_checksum cached_timestamp cached_result
    if command -v jq &>/dev/null; then
        # Try new format first, fall back to old format
        cached_test_checksum=$(echo "$cache_entry" | jq -r '.test_checksum // .checksum // empty')
        cached_source_checksum=$(echo "$cache_entry" | jq -r '.source_checksum // empty')
        cached_timestamp=$(echo "$cache_entry" | jq -r '.timestamp // 0')
        cached_result=$(echo "$cache_entry" | jq -r '.result // empty')
    else
        # Basic parsing fallback
        cached_test_checksum=$(echo "$cache_entry" | grep -o '"test_checksum":"[^"]*"' | cut -d'"' -f4)
        if [[ -z "$cached_test_checksum" ]]; then
            # Fall back to old format
            cached_test_checksum=$(echo "$cache_entry" | grep -o '"checksum":"[^"]*"' | cut -d'"' -f4)
        fi
        cached_source_checksum=$(echo "$cache_entry" | grep -o '"source_checksum":"[^"]*"' | cut -d'"' -f4)
        cached_timestamp=$(echo "$cache_entry" | grep -o '"timestamp":[0-9]*' | cut -d':' -f2)
        cached_result=$(echo "$cache_entry" | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
    fi
    
    # Check if test passed
    if [[ "$cached_result" != "passed" ]]; then
        return 1  # Only skip passed tests
    fi
    
    # Check test file checksum
    local current_test_checksum
    current_test_checksum=$(cache::calculate_checksum "$test_file")
    if [[ "$current_test_checksum" != "$cached_test_checksum" ]]; then
        return 1  # Test file has changed
    fi
    
    # Check source file checksum (if source file exists)
    local source_file
    source_file=$(cache::get_source_file "$test_file")
    if [[ -n "$source_file" ]]; then
        local current_source_checksum
        current_source_checksum=$(cache::calculate_checksum "$source_file")
        
        # If we have a cached source checksum, compare it
        if [[ -n "$cached_source_checksum" ]]; then
            if [[ "$current_source_checksum" != "$cached_source_checksum" ]]; then
                return 1  # Source file has changed
            fi
        else
            # Old cache entry without source checksum - invalidate for safety
            return 1  # Cache entry is outdated format
        fi
    fi
    
    # Check age
    local current_time max_age_seconds age
    current_time=$(cache::timestamp)
    max_age_seconds=$((max_age_minutes * 60))
    age=$((current_time - cached_timestamp))
    
    if [[ $age -gt $max_age_seconds ]]; then
        return 1  # Cache expired
    fi
    
    return 0  # Cache is valid
}

# Store test result in cache
cache::store_result() {
    local test_file="$1"
    local result="$2"  # "passed" or "failed"
    local duration="${3:-0}"  # Test duration in seconds
    
    if [[ "$CACHE_ENABLED" != "true" ]]; then
        return 0  # Cache disabled
    fi
    
    local test_checksum source_checksum timestamp
    test_checksum=$(cache::calculate_checksum "$test_file")
    timestamp=$(cache::timestamp)
    
    # Get source file checksum if source file exists
    local source_file
    source_file=$(cache::get_source_file "$test_file")
    if [[ -n "$source_file" ]]; then
        source_checksum=$(cache::calculate_checksum "$source_file")
    else
        source_checksum=""
    fi
    
    # Create cache entry with new format (including both checksums)
    local entry
    if [[ -n "$source_checksum" ]]; then
        entry="{\"test_checksum\":\"$test_checksum\",\"source_checksum\":\"$source_checksum\",\"timestamp\":$timestamp,\"result\":\"$result\",\"duration\":$duration}"
    else
        # No source file found - store only test checksum (for tests without corresponding .sh files)
        entry="{\"test_checksum\":\"$test_checksum\",\"timestamp\":$timestamp,\"result\":\"$result\",\"duration\":$duration}"
    fi
    
    # Update cache file
    if command -v jq &>/dev/null; then
        # Use jq for proper JSON handling
        local temp_file="${CACHE_FILE}.tmp"
        jq ".\"$test_file\" = $entry" "$CACHE_FILE" > "$temp_file" && mv "$temp_file" "$CACHE_FILE"
    else
        # Fallback: recreate the file (less efficient but works)
        local temp_file="${CACHE_FILE}.tmp"
        
        # Remove old entry if exists
        if grep -q "\"$test_file\":" "$CACHE_FILE"; then
            grep -v "\"$test_file\":" "$CACHE_FILE" > "$temp_file" || true
        else
            cp "$CACHE_FILE" "$temp_file"
        fi
        
        # Add new entry
        if [[ $(wc -c < "$temp_file") -le 3 ]]; then
            # Empty or near-empty file
            echo "{\"$test_file\":$entry}" > "$temp_file"
        else
            # Add to existing entries
            sed -i 's/}$//' "$temp_file"
            echo ",\"$test_file\":$entry}" >> "$temp_file"
        fi
        
        mv "$temp_file" "$CACHE_FILE"
    fi
}

# Clear cache (all or specific pattern)
cache::clear() {
    local pattern="${1:-}"
    
    if [[ -z "$pattern" ]]; then
        # Clear all
        echo '{}' > "$CACHE_FILE"
        echo "Cleared all shell test cache"
    elif command -v jq &>/dev/null; then
        # Clear matching entries
        local temp_file="${CACHE_FILE}.tmp"
        jq "with_entries(select(.key | test(\"$pattern\") | not))" "$CACHE_FILE" > "$temp_file" && mv "$temp_file" "$CACHE_FILE"
        echo "Cleared cache entries matching: $pattern"
    else
        echo "Pattern-based clearing requires jq. Clearing all instead."
        echo '{}' > "$CACHE_FILE"
    fi
}

# Show cache statistics
cache::stats() {
    if [[ ! -f "$CACHE_FILE" ]]; then
        echo "No cache file found"
        return 0
    fi
    
    local total_entries valid_entries expired_entries total_time_saved
    
    if command -v jq &>/dev/null; then
        total_entries=$(jq 'length' "$CACHE_FILE")
        
        local current_time max_age_seconds
        current_time=$(cache::timestamp)
        max_age_seconds=$((CACHE_TTL_MINUTES * 60))
        
        valid_entries=$(jq "[to_entries[] | select((.value.timestamp + $max_age_seconds) > $current_time)] | length" "$CACHE_FILE")
        expired_entries=$((total_entries - valid_entries))
        
        # Calculate total time saved (sum of durations for valid entries)
        total_time_saved=$(jq "[to_entries[] | select((.value.timestamp + $max_age_seconds) > $current_time) | .value.duration // 0] | add // 0" "$CACHE_FILE")
    else
        # Basic stats without jq
        total_entries=$(grep -o '"scripts/[^"]*":' "$CACHE_FILE" | wc -l)
        valid_entries="?"
        expired_entries="?"
        total_time_saved="?"
    fi
    
    echo "=== Shell Test Cache Statistics ==="
    echo "Cache file: $CACHE_FILE"
    echo "Cache TTL: ${CACHE_TTL_MINUTES} minutes"
    echo "Total entries: $total_entries"
    echo "Valid entries: $valid_entries"
    echo "Expired entries: $expired_entries"
    echo "Time saved (estimated): ${total_time_saved}s"
    echo "==================================="
}

# List cached test results
cache::list() {
    local filter="${1:-}"  # Optional filter: "passed", "failed", "valid", "expired"
    
    if [[ ! -f "$CACHE_FILE" ]]; then
        echo "No cache file found"
        return 0
    fi
    
    echo "=== Cached Test Results ==="
    echo "Filter: ${filter:-all}"
    echo ""
    
    if command -v jq &>/dev/null; then
        local current_time max_age_seconds
        current_time=$(cache::timestamp)
        max_age_seconds=$((CACHE_TTL_MINUTES * 60))
        
        case "$filter" in
            passed)
                jq -r 'to_entries[] | select(.value.result == "passed") | "\(.key): \(.value.result) (age: \(('"$current_time"' - .value.timestamp) / 60 | floor)m)"' "$CACHE_FILE"
                ;;
            failed)
                jq -r 'to_entries[] | select(.value.result == "failed") | "\(.key): \(.value.result) (age: \(('"$current_time"' - .value.timestamp) / 60 | floor)m)"' "$CACHE_FILE"
                ;;
            valid)
                jq -r 'to_entries[] | select((.value.timestamp + '"$max_age_seconds"') > '"$current_time"') | "\(.key): \(.value.result) (age: \(('"$current_time"' - .value.timestamp) / 60 | floor)m)"' "$CACHE_FILE"
                ;;
            expired)
                jq -r 'to_entries[] | select((.value.timestamp + '"$max_age_seconds"') <= '"$current_time"') | "\(.key): \(.value.result) (age: \(('"$current_time"' - .value.timestamp) / 60 | floor)m)"' "$CACHE_FILE"
                ;;
            *)
                jq -r 'to_entries[] | "\(.key): \(.value.result) (age: \(('"$current_time"' - .value.timestamp) / 60 | floor)m)"' "$CACHE_FILE"
                ;;
        esac
    else
        # Basic listing without jq
        echo "Install jq for detailed cache listing"
        cat "$CACHE_FILE"
    fi
}

# CLI interface for direct usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        clear)
            cache::clear "${2:-}"
            ;;
        stats)
            cache::stats
            ;;
        list)
            cache::list "${2:-}"
            ;;
        *)
            echo "Usage: $0 {clear [pattern]|stats|list [filter]}"
            echo ""
            echo "Commands:"
            echo "  clear [pattern]  Clear cache (all or matching pattern)"
            echo "  stats           Show cache statistics"
            echo "  list [filter]   List cached results (filter: passed/failed/valid/expired)"
            echo ""
            echo "Environment variables:"
            echo "  VROOLI_TEST_CACHE_TTL      Cache TTL in minutes (default: 30)"
            echo "  VROOLI_TEST_CACHE_ENABLED  Enable/disable cache (default: true)"
            exit 1
            ;;
    esac
fi