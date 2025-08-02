#!/usr/bin/env bash
# Intelligent Test Cache Manager - Phase 3 Optimization
# Provides persistent caching for test setup operations, dependency checks, and results

set -euo pipefail

TESTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPTS_DIR=$(dirname "${TESTS_DIR}")

# Persistent cache directory (survives across test runs)
PERSISTENT_CACHE_DIR="${HOME}/.cache/vrooli-tests"
DEPENDENCY_CACHE="$PERSISTENT_CACHE_DIR/dependencies.cache"
SETUP_CACHE_DIR="$PERSISTENT_CACHE_DIR/setup"
RESULTS_CACHE_DIR="$PERSISTENT_CACHE_DIR/results"
METADATA_CACHE="$PERSISTENT_CACHE_DIR/metadata.cache"

# Create cache directories
mkdir -p "$PERSISTENT_CACHE_DIR" "$SETUP_CACHE_DIR" "$RESULTS_CACHE_DIR"

# Source utilities
source "${TESTS_DIR}/../helpers/utils/log.sh"

#######################################
# Cache Configuration
#######################################

# Cache TTL settings (in seconds)
readonly DEPENDENCY_TTL=3600      # 1 hour - dependencies don't change often
readonly SETUP_TTL=1800          # 30 minutes - setup operations are expensive
readonly RESULTS_TTL=600         # 10 minutes - test results should be relatively fresh
readonly METADATA_TTL=86400      # 24 hours - metadata (file hashes, etc.)

#######################################
# Cache Utility Functions
#######################################

# Generate cache key for a file
generate_cache_key() {
    local file_path="$1"
    local key_type="${2:-file}"
    
    # Create hash based on file path and modification time
    local file_mtime=$(stat -c %Y "$file_path" 2>/dev/null || echo 0)
    local file_size=$(stat -c %s "$file_path" 2>/dev/null || echo 0)
    
    # Generate stable key
    echo "${key_type}_$(echo "${file_path}_${file_mtime}_${file_size}" | sha256sum | cut -d' ' -f1)"
}

# Check if cache entry is valid
is_cache_valid() {
    local cache_file="$1"
    local ttl="$2"
    
    [[ -f "$cache_file" ]] || return 1
    
    local cache_age=$(($(date +%s) - $(stat -c %Y "$cache_file")))
    [[ $cache_age -lt $ttl ]]
}

# Store data in cache with metadata
cache_store() {
    local cache_key="$1"
    local data="$2"
    local cache_type="${3:-generic}"
    
    local cache_file="$PERSISTENT_CACHE_DIR/${cache_key}.cache"
    
    # Store data with metadata
    {
        echo "# Cache metadata"
        echo "CACHE_TYPE=$cache_type"
        echo "CREATED=$(date +%s)"
        echo "CREATED_READABLE=$(date -Iseconds)"
        echo "# Cache data begins"
        echo "$data"
    } > "$cache_file"
    
    log::debug "📦 Cached data for key: $cache_key"
}

# Retrieve data from cache
cache_retrieve() {
    local cache_key="$1"
    local ttl="${2:-$SETUP_TTL}"
    
    local cache_file="$PERSISTENT_CACHE_DIR/${cache_key}.cache"
    
    if is_cache_valid "$cache_file" "$ttl"; then
        # Skip metadata lines and return data
        sed -n '/# Cache data begins/,$p' "$cache_file" | tail -n +2
        log::debug "🎯 Cache hit for key: $cache_key"
        return 0
    else
        log::debug "❌ Cache miss for key: $cache_key"
        return 1
    fi
}

#######################################
# Dependency Caching System
#######################################

# Cache service availability
cache_service_status() {
    local service_name="$1"
    local port="$2"
    local is_available="$3"
    
    local status_line="SERVICE:$service_name:$port:$is_available:$(date +%s)"
    echo "$status_line" >> "$DEPENDENCY_CACHE"
    
    # Keep cache file manageable (last 1000 entries)
    if [[ -f "$DEPENDENCY_CACHE" ]]; then
        tail -1000 "$DEPENDENCY_CACHE" > "$DEPENDENCY_CACHE.tmp"
        mv "$DEPENDENCY_CACHE.tmp" "$DEPENDENCY_CACHE"
    fi
    
    log::debug "💾 Cached service status: $service_name:$port = $is_available"
}

# Check cached service availability
get_cached_service_status() {
    local service_name="$1"
    local port="$2"
    
    [[ -f "$DEPENDENCY_CACHE" ]] || return 1
    
    # Find most recent entry for this service
    local cache_entry=$(grep "^SERVICE:$service_name:$port:" "$DEPENDENCY_CACHE" | tail -1)
    [[ -n "$cache_entry" ]] || return 1
    
    # Parse cache entry
    local timestamp=$(echo "$cache_entry" | cut -d':' -f5)
    local is_available=$(echo "$cache_entry" | cut -d':' -f4)
    local age=$(($(date +%s) - timestamp))
    
    # Check if cache is still valid
    if [[ $age -lt $DEPENDENCY_TTL ]]; then
        log::debug "🎯 Using cached service status: $service_name = $is_available"
        [[ "$is_available" == "true" ]]
        return $?
    else
        log::debug "⏰ Service cache expired for: $service_name"
        return 1
    fi
}

#######################################
# Test Setup Caching System
#######################################

# Cache expensive test setup operations
cache_test_setup() {
    local test_file="$1"
    local setup_data="$2"
    
    local cache_key=$(generate_cache_key "$test_file" "setup")
    local cache_file="$SETUP_CACHE_DIR/${cache_key}.setup"
    
    # Store setup data with rich metadata
    {
        echo "# Test setup cache"
        echo "TEST_FILE=$test_file"
        echo "CACHED_AT=$(date +%s)"
        echo "CACHED_AT_READABLE=$(date -Iseconds)"
        echo "FILE_MTIME=$(stat -c %Y "$test_file" 2>/dev/null || echo 0)"
        echo "# Setup data"
        echo "$setup_data"
    } > "$cache_file"
    
    log::debug "🔧 Cached setup for: $(basename "$test_file")"
}

# Retrieve cached test setup
get_cached_test_setup() {
    local test_file="$1"
    
    local cache_key=$(generate_cache_key "$test_file" "setup")
    local cache_file="$SETUP_CACHE_DIR/${cache_key}.setup"
    
    # Check if cache exists and is valid
    if is_cache_valid "$cache_file" "$SETUP_TTL"; then
        # Verify file hasn't changed since caching
        local cached_mtime=$(grep "^FILE_MTIME=" "$cache_file" | cut -d'=' -f2)
        local current_mtime=$(stat -c %Y "$test_file" 2>/dev/null || echo 0)
        
        if [[ "$cached_mtime" == "$current_mtime" ]]; then
            # Return setup data
            sed -n '/# Setup data/,$p' "$cache_file" | tail -n +2
            log::debug "⚡ Using cached setup for: $(basename "$test_file")"
            return 0
        else
            log::debug "📝 Test file modified, cache invalid: $(basename "$test_file")"
        fi
    fi
    
    return 1
}

#######################################
# Test Results Caching System
#######################################

# Cache test results for quick re-runs
cache_test_results() {
    local test_file="$1"
    local exit_code="$2"
    local test_count="$3"
    local failure_count="$4"
    local duration="$5"
    local output="$6"
    
    local cache_key=$(generate_cache_key "$test_file" "results")
    local cache_file="$RESULTS_CACHE_DIR/${cache_key}.results"
    
    # Store comprehensive test results
    {
        echo "# Test results cache"
        echo "TEST_FILE=$test_file"
        echo "EXIT_CODE=$exit_code"
        echo "TEST_COUNT=$test_count"
        echo "FAILURE_COUNT=$failure_count"
        echo "DURATION=$duration"
        echo "CACHED_AT=$(date +%s)"
        echo "CACHED_AT_READABLE=$(date -Iseconds)"
        echo "FILE_MTIME=$(stat -c %Y "$test_file" 2>/dev/null || echo 0)"
        echo "# Test output"
        echo "$output"
    } > "$cache_file"
    
    log::debug "📊 Cached results for: $(basename "$test_file")"
}

# Retrieve cached test results
get_cached_test_results() {
    local test_file="$1"
    local force_refresh="${2:-false}"
    
    # Skip cache if force refresh requested
    $force_refresh && return 1
    
    local cache_key=$(generate_cache_key "$test_file" "results")
    local cache_file="$RESULTS_CACHE_DIR/${cache_key}.results"
    
    # Check if cache exists and is valid
    if is_cache_valid "$cache_file" "$RESULTS_TTL"; then
        # Verify file hasn't changed since caching
        local cached_mtime=$(grep "^FILE_MTIME=" "$cache_file" | cut -d'=' -f2)
        local current_mtime=$(stat -c %Y "$test_file" 2>/dev/null || echo 0)
        
        if [[ "$cached_mtime" == "$current_mtime" ]]; then
            # Parse and return cached results
            local exit_code=$(grep "^EXIT_CODE=" "$cache_file" | cut -d'=' -f2)
            local test_count=$(grep "^TEST_COUNT=" "$cache_file" | cut -d'=' -f2)
            local failure_count=$(grep "^FAILURE_COUNT=" "$cache_file" | cut -d'=' -f2)
            local duration=$(grep "^DURATION=" "$cache_file" | cut -d'=' -f2)
            
            # Export results for caller
            export CACHED_EXIT_CODE="$exit_code"
            export CACHED_TEST_COUNT="$test_count"
            export CACHED_FAILURE_COUNT="$failure_count"
            export CACHED_DURATION="$duration"
            
            # Return output
            sed -n '/# Test output/,$p' "$cache_file" | tail -n +2
            
            log::debug "⚡ Using cached results for: $(basename "$test_file") (${test_count} tests, ${duration}s)"
            return 0
        else
            log::debug "📝 Test file modified, results cache invalid: $(basename "$test_file")"
        fi
    fi
    
    return 1
}

#######################################
# Cache Management Functions
#######################################

# Clean expired cache entries
clean_expired_cache() {
    local cleaned_count=0
    
    log::info "🧹 Cleaning expired cache entries..."
    
    # Clean dependency cache
    if [[ -f "$DEPENDENCY_CACHE" ]]; then
        local current_time=$(date +%s)
        local temp_file=$(mktemp)
        
        while IFS= read -r line; do
            if [[ "$line" =~ SERVICE:.*:([0-9]+)$ ]]; then
                local timestamp="${BASH_REMATCH[1]}"
                local age=$((current_time - timestamp))
                
                if [[ $age -le $DEPENDENCY_TTL ]]; then
                    echo "$line" >> "$temp_file"
                else
                    ((cleaned_count++))
                fi
            fi
        done < "$DEPENDENCY_CACHE"
        
        mv "$temp_file" "$DEPENDENCY_CACHE"
    fi
    
    # Clean setup cache
    find "$SETUP_CACHE_DIR" -name "*.setup" -type f -mtime +$((SETUP_TTL / 86400)) -delete 2>/dev/null || true
    cleaned_count=$((cleaned_count + $(find "$SETUP_CACHE_DIR" -name "*.setup" -type f -mtime +1 | wc -l)))
    
    # Clean results cache
    find "$RESULTS_CACHE_DIR" -name "*.results" -type f -mtime +$((RESULTS_TTL / 86400)) -delete 2>/dev/null || true
    cleaned_count=$((cleaned_count + $(find "$RESULTS_CACHE_DIR" -name "*.results" -type f -mtime +1 | wc -l)))
    
    log::info "✅ Cleaned $cleaned_count expired cache entries"
}

# Show cache statistics
show_cache_stats() {
    log::header "📊 Test Cache Statistics"
    
    # Dependency cache stats
    local dep_entries=0
    [[ -f "$DEPENDENCY_CACHE" ]] && dep_entries=$(wc -l < "$DEPENDENCY_CACHE")
    log::info "Dependencies cached: $dep_entries entries"
    
    # Setup cache stats
    local setup_files=$(find "$SETUP_CACHE_DIR" -name "*.setup" -type f | wc -l)
    local setup_size=$(du -sh "$SETUP_CACHE_DIR" 2>/dev/null | cut -f1 || echo "0K")
    log::info "Setup operations cached: $setup_files files ($setup_size)"
    
    # Results cache stats
    local results_files=$(find "$RESULTS_CACHE_DIR" -name "*.results" -type f | wc -l)
    local results_size=$(du -sh "$RESULTS_CACHE_DIR" 2>/dev/null | cut -f1 || echo "0K")
    log::info "Test results cached: $results_files files ($results_size)"
    
    # Total cache size
    local total_size=$(du -sh "$PERSISTENT_CACHE_DIR" 2>/dev/null | cut -f1 || echo "0K")
    log::info "Total cache size: $total_size"
    
    # Cache hit rate (if available)
    if [[ -f "$METADATA_CACHE" ]]; then
        local cache_hits=$(grep -c "CACHE_HIT" "$METADATA_CACHE" 2>/dev/null || echo 0)
        local cache_misses=$(grep -c "CACHE_MISS" "$METADATA_CACHE" 2>/dev/null || echo 0)
        local total_requests=$((cache_hits + cache_misses))
        
        if [[ $total_requests -gt 0 ]]; then
            local hit_rate=$(echo "scale=1; $cache_hits * 100 / $total_requests" | bc -l)
            log::info "Cache hit rate: ${hit_rate}% ($cache_hits/$total_requests)"
        fi
    fi
}

# Clear all cached data
clear_cache() {
    log::warning "🗑️  Clearing all cached data..."
    
    rm -rf "$PERSISTENT_CACHE_DIR"
    mkdir -p "$PERSISTENT_CACHE_DIR" "$SETUP_CACHE_DIR" "$RESULTS_CACHE_DIR"
    
    log::success "✅ All cache data cleared"
}

# Warm up cache with common operations
warm_cache() {
    log::info "🔥 Warming up cache with common test operations..."
    
    # Pre-cache common service checks
    local common_services=("ollama:11434" "whisper:8090" "node-red:1880" "minio:9000")
    
    for service_port in "${common_services[@]}"; do
        local service=$(echo "$service_port" | cut -d':' -f1)
        local port=$(echo "$service_port" | cut -d':' -f2)
        
        # Check service and cache result
        if timeout 2 bash -c "echo >/dev/tcp/localhost/$port" 2>/dev/null; then
            cache_service_status "$service" "$port" "true"
        else
            cache_service_status "$service" "$port" "false"
        fi
    done
    
    log::success "✅ Cache warmed up with common services"
}

#######################################
# Main Cache Manager
#######################################

main() {
    local action="${1:-stats}"
    
    case "$action" in
        stats|status)
            show_cache_stats
            ;;
        clean)
            clean_expired_cache
            ;;
        clear)
            clear_cache
            ;;
        warm)
            warm_cache
            ;;
        help|--help|-h)
            echo "Test Cache Manager - Phase 3 Optimization"
            echo ""
            echo "Usage: $0 [ACTION]"
            echo ""
            echo "Actions:"
            echo "  stats                Show cache statistics (default)"
            echo "  clean                Clean expired cache entries"
            echo "  clear                Clear all cached data"
            echo "  warm                 Warm up cache with common operations"
            echo ""
            echo "Cache Locations:"
            echo "  Main cache: $PERSISTENT_CACHE_DIR"
            echo "  Dependencies: $DEPENDENCY_CACHE"
            echo "  Setup cache: $SETUP_CACHE_DIR"
            echo "  Results cache: $RESULTS_CACHE_DIR"
            echo ""
            echo "Cache TTL Settings:"
            echo "  Dependencies: ${DEPENDENCY_TTL}s ($(($DEPENDENCY_TTL / 60))min)"
            echo "  Setup operations: ${SETUP_TTL}s ($(($SETUP_TTL / 60))min)"
            echo "  Test results: ${RESULTS_TTL}s ($(($RESULTS_TTL / 60))min)"
            ;;
        *)
            log::error "Unknown action: $action"
            log::info "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Only run main if not being sourced by another script
if [[ -z "${CACHE_MANAGER_SOURCED:-}" ]]; then
    main "$@"
fi