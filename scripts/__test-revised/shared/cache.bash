#!/usr/bin/env bash
# JSON-based cache utilities for Vrooli testing infrastructure
# Provides efficient caching with a single JSON file per test phase
#
# Cache structure:
# - One JSON file per phase (static.json, resources.json, etc.)
# - File paths as keys, test results as values
# - Automatic invalidation based on file mtime and TTL

# shellcheck disable=SC1091
[[ -z "${SCRIPT_DIR:-}" ]] && SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$SCRIPT_DIR/shared/logging.bash"

# Cache configuration
CACHE_DIR="${SCRIPT_DIR}/cache"
CACHE_VERSION="3.0"
CACHE_TTL="${TEST_CACHE_TTL:-3600}"  # 1 hour default
CACHE_MAX_AGE="${TEST_CACHE_MAX_AGE:-86400}"  # 24 hours max

# Global cache data - loaded once per phase
declare -gA CACHE_DATA=()
declare -g CACHE_PHASE=""
declare -g CACHE_MODIFIED=false
declare -g CACHE_STATS_HITS=0
declare -g CACHE_STATS_MISSES=0
declare -g CACHE_STATS_EXPIRED=0
declare -g CACHE_STATS_INVALIDATED=0

# Ensure cache directory exists
init_cache() {
    if ! mkdir -p "$CACHE_DIR" 2>/dev/null; then
        log_debug "Failed to create cache directory: $CACHE_DIR"
        return 1
    fi
    
    # Clean up old cache files older than max age
    if command -v find >/dev/null 2>&1; then
        find "$CACHE_DIR" -type f -name "*.json" -mtime +1 -delete 2>/dev/null || true
        # Remove any leftover temp files
        find "$CACHE_DIR" -type f -name "*.json.tmp.*" -delete 2>/dev/null || true
    fi
    
    return 0
}

# Load cache for a specific phase
load_cache() {
    local phase="${1:-generic}"
    CACHE_PHASE="$phase"
    CACHE_MODIFIED=false
    CACHE_DATA=()
    
    local cache_file="$CACHE_DIR/${phase}.json"
    
    if [[ ! -f "$cache_file" ]]; then
        log_debug "No cache file found for phase: $phase"
        return 0
    fi
    
    # Check if jq is available for JSON parsing
    if ! command -v jq >/dev/null 2>&1; then
        log_warning "jq not available - cache system disabled"
        return 1
    fi
    
    # Validate and load cache file
    if ! jq empty < "$cache_file" >/dev/null 2>&1; then
        log_warning "Cache file corrupted, removing: $cache_file"
        rm -f "$cache_file"
        return 0
    fi
    
    # Check cache version
    local cache_version
    cache_version=$(jq -r '.version // "unknown"' "$cache_file" 2>/dev/null)
    if [[ "$cache_version" != "$CACHE_VERSION" ]]; then
        log_debug "Cache version mismatch (expected: $CACHE_VERSION, got: $cache_version), clearing cache"
        rm -f "$cache_file"
        return 0
    fi
    
    # Load cache entries into memory
    # Format: CACHE_DATA["file:test_type"]="status:timestamp:duration:message:exit_code:mtime"
    local entries
    entries=$(jq -r '.entries | to_entries[] | 
        .key as $file | 
        .value.file.mtime as $mtime |
        .value.results | to_entries[] | 
        "\($file):\(.key)=\(.value.status):\(.value.timestamp):\(.value.duration_ms):\(.value.message // ""):\(.value.exit_code):\($mtime)"' \
        "$cache_file" 2>/dev/null) || true
    
    while IFS='=' read -r key value; do
        [[ -n "$key" ]] && CACHE_DATA["$key"]="$value"
    done <<< "$entries"
    
    local num_entries=${#CACHE_DATA[@]}
    log_debug "Loaded $num_entries cache entries for phase: $phase"
    
    return 0
}

# Save cache for current phase
save_cache() {
    local phase="${CACHE_PHASE:-generic}"
    
    if [[ "$CACHE_MODIFIED" != "true" ]]; then
        log_debug "Cache not modified, skipping save for phase: $phase"
        return 0
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        return 1
    fi
    
    local cache_file="$CACHE_DIR/${phase}.json"
    local temp_file="${cache_file}.tmp.$$"
    
    # Build JSON structure
    local json_entries="{}"
    local current_time
    current_time=$(date +%s)
    
    # Group cache entries by file
    declare -A file_results=()
    
    for key in "${!CACHE_DATA[@]}"; do
        IFS=':' read -r file test_type <<< "$key"
        IFS=':' read -r status timestamp duration message exit_code mtime <<< "${CACHE_DATA[$key]}"
        
        # Skip expired entries
        local age=$((current_time - timestamp))
        if [[ $age -gt $CACHE_MAX_AGE ]]; then
            continue
        fi
        
        # Build JSON for this result - use --arg for all values to avoid JSON encoding issues
        local result_json
        local clean_timestamp="${timestamp:-0}"
        local clean_duration="${duration:-0}"
        local clean_exit_code="${exit_code:-0}"
        local clean_message="${message:-}"
        
        # Ensure numeric values are valid integers
        [[ ! "$clean_timestamp" =~ ^[0-9]+$ ]] && clean_timestamp="0"
        [[ ! "$clean_duration" =~ ^[0-9]+$ ]] && clean_duration="0"
        [[ ! "$clean_exit_code" =~ ^[0-9]+$ ]] && clean_exit_code="0"
        
        # Clean message of problematic characters that could break JSON
        clean_message="$(echo "$clean_message" | tr '"' "'" | tr '\n' ' ' | tr '\r' ' ')"
        
        result_json=$(jq -n \
            --arg status "$status" \
            --arg timestamp "$clean_timestamp" \
            --arg duration "$clean_duration" \
            --arg message "$clean_message" \
            --arg exit_code "$clean_exit_code" \
            '{status: $status, timestamp: ($timestamp | tonumber), duration_ms: ($duration | tonumber), message: (if $message == "" then null else $message end), exit_code: ($exit_code | tonumber)}')
        
        # Add to file's results
        if [[ -z "${file_results[$file]:-}" ]]; then
            # Ensure mtime is never empty
            local safe_mtime="${mtime:-0}"
            [[ -z "$safe_mtime" ]] && safe_mtime="0"
            
            file_results["$file"]=$(jq -n \
                --arg mtime "$safe_mtime" \
                --argjson result "$result_json" \
                --arg test_type "$test_type" \
                '{file: {mtime: ($mtime | tonumber)}, results: {($test_type): $result}}')
        else
            file_results["$file"]=$(echo "${file_results[$file]}" | jq \
                --argjson result "$result_json" \
                --arg test_type "$test_type" \
                '.results[$test_type] = $result')
        fi
    done
    
    # Build final JSON using file-based approach to avoid "Argument list too long" errors
    local entries_file="${temp_file}.entries"
    local metadata_file="${temp_file}.metadata"
    
    # Write entries using individual JSON files to avoid memory/argument limits
    local entries_dir="${temp_file}.entries_dir"
    mkdir -p "$entries_dir"
    
    # Write each file's data to a separate JSON file to avoid large string concatenation
    local file_index=0
    for file in "${!file_results[@]}"; do
        # Create individual entry file with proper JSON structure
        jq -n \
            --arg filepath "$file" \
            --argjson data "${file_results[$file]}" \
            '{($filepath): $data}' \
            > "${entries_dir}/${file_index}.json"
        ((file_index++))
    done
    
    # Combine all entry files using jq's efficient file processing
    if [[ $file_index -gt 0 ]]; then
        # Use jq to merge all JSON files efficiently
        jq -s 'add' "${entries_dir}"/*.json > "$entries_file"
    else
        echo "{}" > "$entries_file"
    fi
    
    # Clean up individual entry files
    rm -rf "$entries_dir"
    
    # Create metadata in separate file
    jq -n \
        --arg version "$CACHE_VERSION" \
        --arg phase "$phase" \
        --arg updated "$(date -Iseconds)" \
        --arg ttl "$CACHE_TTL" \
        '{version: $version, phase: $phase, updated: $updated, ttl_seconds: ($ttl | tonumber)}' \
        > "$metadata_file"
    
    # Combine metadata and entries using file-based approach
    if jq --slurpfile entries "$entries_file" \
          '. + {entries: $entries[0]}' \
          "$metadata_file" > "$temp_file" 2>/dev/null; then
        mv "$temp_file" "$cache_file"
        log_debug "Saved cache for phase: $phase"
        CACHE_MODIFIED=false
        
        # Clean up temporary files
        rm -f "$entries_file" "$metadata_file"
        return 0
    else
        log_debug "Failed to save cache for phase: $phase"
        rm -f "$temp_file" "$entries_file" "$metadata_file" "${entries_file}.tmp"
        rm -rf "${entries_dir}" 2>/dev/null || true
        return 1
    fi
}

# Check if a test result is cached and valid
check_cache() {
    local file_path="$1"
    local test_type="$2"
    
    # Skip caching if NO_CACHE is set
    if [[ -n "${NO_CACHE:-}" ]]; then
        ((CACHE_STATS_MISSES++))
        return 1
    fi
    
    local cache_key="${file_path}:${test_type}"
    
    # Check if we have this entry
    if [[ -z "${CACHE_DATA[$cache_key]:-}" ]]; then
        ((CACHE_STATS_MISSES++))
        log_cache_miss "$cache_key"
        return 1
    fi
    
    # Parse cache entry
    IFS=':' read -r status timestamp duration message exit_code cached_mtime <<< "${CACHE_DATA[$cache_key]}"
    
    # Check file modification time
    local current_mtime
    if [[ -f "$file_path" ]]; then
        if command -v stat >/dev/null 2>&1; then
            current_mtime=$(stat -c %Y "$file_path" 2>/dev/null || echo "0")
        else
            current_mtime="0"
        fi
        
        if [[ "$current_mtime" != "$cached_mtime" ]]; then
            ((CACHE_STATS_INVALIDATED++))
            log_debug "Cache invalidated due to file change: $file_path"
            return 1
        fi
    fi
    
    # Check TTL
    local current_time
    current_time=$(date +%s)
    local age=$((current_time - timestamp))
    
    if [[ $age -gt $CACHE_TTL ]]; then
        ((CACHE_STATS_EXPIRED++))
        log_debug "Cache expired for: $cache_key (age: ${age}s)"
        return 1
    fi
    
    # Re-run failed tests (don't cache failures long-term)
    if [[ "$status" == "failed" ]] && [[ $age -gt 300 ]]; then  # 5 minutes for failed tests
        ((CACHE_STATS_EXPIRED++))
        log_debug "Re-running previously failed test: $cache_key"
        return 1
    fi
    
    ((CACHE_STATS_HITS++))
    log_cache_hit "$cache_key"
    
    # Return the cached status
    echo "${status}:${message}"
    return 0
}

# Update cache with test result
update_cache() {
    local file_path="$1"
    local test_type="$2"
    local status="$3"
    local duration_ms="${4:-0}"
    local message="${5:-}"
    local exit_code="${6:-0}"
    
    local cache_key="${file_path}:${test_type}"
    
    # Get file mtime
    local mtime="0"
    if [[ -f "$file_path" ]]; then
        if command -v stat >/dev/null 2>&1; then
            mtime=$(stat -c %Y "$file_path" 2>/dev/null || echo "0")
        fi
    fi
    
    local timestamp
    timestamp=$(date +%s)
    
    # Store in cache
    CACHE_DATA["$cache_key"]="${status}:${timestamp}:${duration_ms}:${message}:${exit_code}:${mtime}"
    CACHE_MODIFIED=true
    
    log_debug "Updated cache: $cache_key = $status"
}

# Clear cache for specific phase or all
clear_cache() {
    local phase="${1:-}"
    
    if [[ -n "$phase" ]]; then
        log_info "Clearing cache for phase: $phase"
        rm -f "$CACHE_DIR/${phase}.json"
        rm -f "$CACHE_DIR/${phase}.json.tmp."*
        if [[ "$phase" == "$CACHE_PHASE" ]]; then
            CACHE_DATA=()
            CACHE_MODIFIED=false
        fi
    else
        log_info "Clearing all cache"
        rm -f "$CACHE_DIR"/*.json
        rm -f "$CACHE_DIR"/*.json.tmp.*
        CACHE_DATA=()
        CACHE_MODIFIED=false
    fi
}

# Show cache statistics
show_cache_stats() {
    log_info "Cache Statistics:"
    log_info "  Phase: ${CACHE_PHASE:-none}"
    log_info "  Entries in memory: ${#CACHE_DATA[@]}"
    log_info "  Hits: $CACHE_STATS_HITS"
    log_info "  Misses: $CACHE_STATS_MISSES"
    log_info "  Expired: $CACHE_STATS_EXPIRED"
    log_info "  Invalidated: $CACHE_STATS_INVALIDATED"
    
    local hit_rate=0
    local total=$((CACHE_STATS_HITS + CACHE_STATS_MISSES))
    if [[ $total -gt 0 ]]; then
        hit_rate=$((CACHE_STATS_HITS * 100 / total))
    fi
    log_info "  Hit rate: ${hit_rate}%"
    
    # Show cache files
    if [[ -d "$CACHE_DIR" ]]; then
        log_info ""
        log_info "Cache files:"
        for cache_file in "$CACHE_DIR"/*.json; do
            if [[ -f "$cache_file" ]]; then
                local phase_name
                phase_name=$(basename "$cache_file" .json)
                local file_size
                file_size=$(du -h "$cache_file" 2>/dev/null | cut -f1)
                local num_entries=0
                if command -v jq >/dev/null 2>&1; then
                    num_entries=$(jq '.entries | length' "$cache_file" 2>/dev/null || echo "0")
                fi
                log_info "    $phase_name: $num_entries entries ($file_size)"
            fi
        done
    fi
}

# Reset cache statistics
reset_cache_stats() {
    CACHE_STATS_HITS=0
    CACHE_STATS_MISSES=0
    CACHE_STATS_EXPIRED=0
    CACHE_STATS_INVALIDATED=0
}

# Get cached test result (backward compatibility wrapper)
get_cached_test_result() {
    local file_path="$1"
    local test_type="$2"
    
    check_cache "$file_path" "$test_type"
}

# Cache test result (backward compatibility wrapper)
cache_test_result() {
    local file_path="$1"
    local test_type="$2"
    local test_result="$3"
    local error_message="${4:-}"
    
    update_cache "$file_path" "$test_type" "$test_result" 0 "$error_message" 0
}

# Check if test should be skipped due to caching
should_skip_test() {
    local file_path="$1"
    local test_type="$2"
    
    local cached_result
    if cached_result=$(check_cache "$file_path" "$test_type"); then
        local result_status="${cached_result%%:*}"
        local error_message="${cached_result#*:}"
        
        case "$result_status" in
            "passed")
                log_test_pass "$(relative_path "$file_path")" "(cached)"
                increment_test_counter "passed"
                return 0
                ;;
            "skipped")
                log_test_skip "$(relative_path "$file_path")" "${error_message:-cached skip}"
                increment_test_counter "skipped"
                return 0
                ;;
            "failed")
                # Re-run failed tests
                return 1
                ;;
            *)
                return 1
                ;;
        esac
    else
        return 1
    fi
}

# Run test with caching support
run_cached_test() {
    local file_path="$1"
    local test_type="$2"
    local test_command="$3"
    local test_name="${4:-$(relative_path "$file_path")}"
    
    # Check if we can skip this test due to caching
    if should_skip_test "$file_path" "$test_type"; then
        return 0
    fi
    
    # Run the actual test
    local test_result="failed"
    local error_message=""
    local start_time
    local end_time
    local duration_ms=0
    
    log_test_start "$test_name"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would run test: $test_command"
        test_result="skipped"
        error_message="dry run"
    else
        start_time=$(date +%s%N)
        
        # Use eval carefully - ensure test_command is trusted
        if (eval "$test_command") >/dev/null 2>&1; then
            test_result="passed"
            log_test_pass "$test_name"
            increment_test_counter "passed"
        else
            local exit_code=$?
            test_result="failed"
            error_message="exit code: $exit_code"
            log_test_fail "$test_name" "$error_message"
            increment_test_counter "failed"
        fi
        
        end_time=$(date +%s%N)
        duration_ms=$(( (end_time - start_time) / 1000000 ))
    fi
    
    # Cache the result
    update_cache "$file_path" "$test_type" "$test_result" "$duration_ms" "$error_message" "${exit_code:-0}"
    
    # Return appropriate exit code
    case "$test_result" in
        "passed"|"skipped")
            return 0
            ;;
        "failed")
            return 1
            ;;
    esac
}

# Initialize cache on script load
init_cache

# Export functions for use in subshells
export -f init_cache load_cache save_cache check_cache update_cache
export -f clear_cache show_cache_stats reset_cache_stats
export -f get_cached_test_result cache_test_result should_skip_test run_cached_test