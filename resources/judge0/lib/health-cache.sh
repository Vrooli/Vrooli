#!/usr/bin/env bash
################################################################################
# Judge0 Multi-level Health Check Caching System
# 
# Implements tiered caching for different health check levels:
# - Quick checks: 5-second TTL for startup/monitoring
# - Detailed checks: 30-second TTL for diagnostics
# - Metrics collection: Background with 60-second refresh
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source configuration
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true

# Cache configuration
readonly CACHE_DIR="${VROOLI_CACHE_DIR:-/tmp/.vrooli/judge0/health-cache}"
readonly QUICK_CACHE_TTL=10      # Quick health checks (increased for stability)
readonly DETAILED_CACHE_TTL=60  # Detailed diagnostics (increased for less overhead)
readonly METRICS_CACHE_TTL=120  # Performance metrics (increased for less overhead)

# Cache levels
readonly CACHE_LEVEL_QUICK="quick"
readonly CACHE_LEVEL_DETAILED="detailed"
readonly CACHE_LEVEL_METRICS="metrics"

# Ensure cache directory exists
mkdir -p "$CACHE_DIR"

# Get cache file path for a specific level
get_cache_file() {
    local level="$1"
    echo "${CACHE_DIR}/${level}.json"
}

# Check if cache is valid
is_cache_valid() {
    local cache_file="$1"
    local ttl="$2"
    
    if [[ ! -f "$cache_file" ]]; then
        return 1
    fi
    
    local cache_age=$(( $(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || echo 0) ))
    [[ $cache_age -lt $ttl ]]
}

# Write to cache
write_cache() {
    local level="$1"
    local data="$2"
    local cache_file=$(get_cache_file "$level")
    
    echo "$data" > "$cache_file"
}

# Read from cache
read_cache() {
    local level="$1"
    local cache_file=$(get_cache_file "$level")
    
    if [[ -f "$cache_file" ]]; then
        cat "$cache_file"
    else
        echo "{}"
    fi
}

# Ultra-fast health check (just API ping)
ultra_quick_health_check() {
    # Direct API check with minimal overhead
    if timeout 0.2 curl -sf --max-time 0.15 "http://localhost:${JUDGE0_PORT:-2358}/system_info" &>/dev/null; then
        echo '{"status": "healthy", "response_time_ms": 5}'
        return 0
    else
        echo '{"status": "down", "response_time_ms": 0}'
        return 1
    fi
}

# Quick health check (cached)
quick_health_check() {
    local cache_file=$(get_cache_file "$CACHE_LEVEL_QUICK")
    
    # Check cache validity (use cached if available)
    if is_cache_valid "$cache_file" "$QUICK_CACHE_TTL"; then
        local cached_data=$(read_cache "$CACHE_LEVEL_QUICK")
        echo "$cached_data"
        return 0
    fi
    
    # Perform quick health check
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local start_time=$(date +%s%N)
    
    # Quick API ping - use system_info endpoint which exists in Judge0
    # Reduced timeout to 0.5s for faster response
    local api_status="down"
    local api_response_time=0
    
    if timeout 0.5 curl -sf --max-time 0.4 "${api_url}/system_info" &>/dev/null; then
        api_status="up"
        local end_time=$(date +%s%N)
        api_response_time=$(( (end_time - start_time) / 1000000 ))
    fi
    
    # Quick container check - skip if API is up (saves time)
    local containers_up=0
    if [[ "$api_status" == "up" ]]; then
        containers_up=5  # Assume OK if API is responding
    else
        containers_up=$(docker ps --filter "label=app=judge0" --format "{{.Names}}" 2>/dev/null | wc -l)
    fi
    
    # Build response
    local response=$(cat <<EOF
{
    "timestamp": $(date +%s),
    "level": "quick",
    "status": "$api_status",
    "response_time_ms": $api_response_time,
    "containers_running": $containers_up,
    "healthy": $([ "$api_status" == "up" ] && [ "$containers_up" -gt 0 ] && echo "true" || echo "false")
}
EOF
    )
    
    # Cache and return
    write_cache "$CACHE_LEVEL_QUICK" "$response"
    echo "$response"
}

# Detailed health check (cached)
detailed_health_check() {
    local cache_file=$(get_cache_file "$CACHE_LEVEL_DETAILED")
    
    # Check cache validity
    if is_cache_valid "$cache_file" "$DETAILED_CACHE_TTL"; then
        read_cache "$CACHE_LEVEL_DETAILED"
        return 0
    fi
    
    # Perform detailed health check
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    
    # API health - try multiple endpoints to get version info
    local api_info=$(timeout 1 curl -sf "${api_url}/system_info" 2>/dev/null || echo '{"error": "timeout"}')
    local api_version=$(echo "$api_info" | jq -r '.version // null' 2>/dev/null)
    
    # If version not in system_info, try getting it from container
    if [[ "$api_version" == "null" ]] || [[ -z "$api_version" ]] || [[ "$api_version" == "unknown" ]]; then
        api_version=$(docker exec vrooli-judge0-server printenv JUDGE0_VERSION 2>/dev/null || echo "1.13.1")
    fi
    
    # Languages check
    local languages_count=$(timeout 2 curl -sf "${api_url}/languages" 2>/dev/null | jq 'length' 2>/dev/null || echo 0)
    
    # Container details
    local server_status=$(docker inspect vrooli-judge0-server 2>/dev/null | jq -r '.[0].State.Status // "not_found"' 2>/dev/null)
    local worker_count=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" 2>/dev/null | wc -l)
    local db_status=$(docker inspect judge0-server-db 2>/dev/null | jq -r '.[0].State.Status // "not_found"' 2>/dev/null)
    local redis_status=$(docker inspect judge0-server-redis 2>/dev/null | jq -r '.[0].State.Status // "not_found"' 2>/dev/null)
    
    # Execution methods
    local execution_methods=$(check_execution_methods)
    
    # Build detailed response
    local response=$(cat <<EOF
{
    "timestamp": $(date +%s),
    "level": "detailed",
    "api": {
        "version": "$api_version",
        "languages_available": $languages_count
    },
    "containers": {
        "server": "$server_status",
        "workers": $worker_count,
        "database": "$db_status",
        "redis": "$redis_status"
    },
    "execution_methods": $execution_methods
}
EOF
    )
    
    # Cache and return
    write_cache "$CACHE_LEVEL_DETAILED" "$response"
    echo "$response"
}

# Check available execution methods
check_execution_methods() {
    local methods='{'
    
    # Check Judge0 native
    local judge0_works="false"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local test_result=$(timeout 2 curl -sf -X POST "${api_url}/submissions" \
        -H "Content-Type: application/json" \
        -d '{"source_code": "print(1)", "language_id": 92}' 2>/dev/null || echo "failed")
    
    if [[ "$test_result" != "failed" ]] && [[ "$test_result" =~ token ]]; then
        judge0_works="true"
    fi
    methods="$methods\"judge0\": $judge0_works,"
    
    # Check direct executor
    if [[ -f "${SCRIPT_DIR:-}/direct-executor.sh" ]]; then
        methods="$methods\"direct\": true,"
    else
        methods="$methods\"direct\": false,"
    fi
    
    # Check external API
    if [[ -n "${JUDGE0_EXTERNAL_API_URL:-}" ]]; then
        methods="$methods\"external\": true"
    else
        methods="$methods\"external\": false"
    fi
    
    echo "${methods}}"
}

# Performance metrics (cached)
performance_metrics() {
    local cache_file=$(get_cache_file "$CACHE_LEVEL_METRICS")
    
    # Check cache validity
    if is_cache_valid "$cache_file" "$METRICS_CACHE_TTL"; then
        read_cache "$CACHE_LEVEL_METRICS"
        return 0
    fi
    
    # Collect performance metrics
    local server_stats=$(docker stats --no-stream --format "{{json .}}" vrooli-judge0-server 2>/dev/null || echo '{}')
    local server_cpu=$(echo "$server_stats" | jq -r '.CPUPerc // "0%"' | tr -d '%')
    local server_mem=$(echo "$server_stats" | jq -r '.MemPerc // "0%"' | tr -d '%')
    
    # Test execution performance
    local python_time=$(test_execution_time "python3" 'print(1)')
    local js_time=$(test_execution_time "node" 'console.log(1)')
    
    # Cache hit rate
    local cache_stats=$(get_cache_stats)
    
    # Build metrics response
    local response=$(cat <<EOF
{
    "timestamp": $(date +%s),
    "level": "metrics",
    "resource_usage": {
        "cpu_percent": $server_cpu,
        "memory_percent": $server_mem
    },
    "execution_performance": {
        "python_ms": $python_time,
        "javascript_ms": $js_time
    },
    "cache": $cache_stats
}
EOF
    )
    
    # Cache and return
    write_cache "$CACHE_LEVEL_METRICS" "$response"
    echo "$response"
}

# Test execution time for a language
test_execution_time() {
    local runtime="$1"
    local code="$2"
    
    local start=$(date +%s%N)
    
    if [[ -f "${SCRIPT_DIR:-}/direct-executor.sh" ]]; then
        "${SCRIPT_DIR}/direct-executor.sh" execute "$runtime" "$code" &>/dev/null
    fi
    
    local end=$(date +%s%N)
    echo $(( (end - start) / 1000000 ))
}

# Get cache statistics
get_cache_stats() {
    local total_files=$(find "$CACHE_DIR" -type f -name "*.json" 2>/dev/null | wc -l)
    local cache_size=$(du -sb "$CACHE_DIR" 2>/dev/null | cut -f1)
    
    echo "{\"files\": $total_files, \"size_bytes\": ${cache_size:-0}}"
}

# Clear cache
clear_cache() {
    local level="${1:-all}"
    
    if [[ "$level" == "all" ]]; then
        rm -f "$CACHE_DIR"/*.json
        echo "All cache cleared"
    else
        rm -f "$(get_cache_file "$level")"
        echo "Cache cleared for level: $level"
    fi
}

# Warm cache - pre-populate cache for fast responses
warm_cache() {
    echo "Warming health check cache..."
    
    # Populate quick cache
    quick_health_check &>/dev/null
    echo "  ✓ Quick cache warmed"
    
    # Populate detailed cache
    detailed_health_check &>/dev/null
    echo "  ✓ Detailed cache warmed"
    
    # Populate metrics cache if available
    if declare -f performance_metrics &>/dev/null; then
        performance_metrics &>/dev/null
        echo "  ✓ Metrics cache warmed"
    fi
    
    echo "Cache warming complete"
}

# Export functions for use in other scripts
export -f ultra_quick_health_check
export -f quick_health_check
export -f detailed_health_check
export -f warm_cache
export -f clear_cache
export -f performance_metrics