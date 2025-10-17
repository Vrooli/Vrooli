#!/usr/bin/env bash
################################################################################
# Judge0 Health Monitoring Library - Enhanced Diagnostics
# 
# Comprehensive health checks with detailed diagnostics and performance optimizations
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source config
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true

# Health status codes
readonly HEALTH_OK=0
readonly HEALTH_WARNING=1
readonly HEALTH_CRITICAL=2

# Cache settings
HEALTH_CACHE_FILE="/tmp/judge0_health_cache.json"
HEALTH_CACHE_TTL=5  # Cache valid for 5 seconds
HEALTH_PERF_FILE="/tmp/judge0_health_perf.json"  # Performance metrics history

# Check if cached health result is valid
judge0::health::check_cache() {
    if [[ -f "$HEALTH_CACHE_FILE" ]]; then
        local cache_age=$(( $(date +%s) - $(stat -c %Y "$HEALTH_CACHE_FILE" 2>/dev/null || echo 0) ))
        if [[ $cache_age -lt $HEALTH_CACHE_TTL ]]; then
            cat "$HEALTH_CACHE_FILE"
            return 0
        fi
    fi
    return 1
}

# Enhanced health check with diagnostics
judge0::health::check() {
    local verbose="${1:-false}"
    local use_cache="${2:-true}"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local health_status=$HEALTH_OK
    local health_messages=()
    
    # Check cache if enabled
    if [[ "$use_cache" == "true" ]] && [[ "$verbose" == "false" ]]; then
        if judge0::health::check_cache; then
            return 0
        fi
    fi
    
    # Check API responsiveness
    local api_start=$(date +%s%N)
    local api_response=$(timeout 5 curl -sf --max-time 3 "${api_url}/system_info" 2>/dev/null || echo "FAILED")
    local api_end=$(date +%s%N)
    local api_time=$(( (api_end - api_start) / 1000000 ))
    
    if [[ "$api_response" == "FAILED" ]]; then
        health_messages+=("❌ API not responding")
        health_status=$HEALTH_CRITICAL
    else
        if [[ $api_time -lt 500 ]]; then
            health_messages+=("✅ API response: ${api_time}ms")
        elif [[ $api_time -lt 1000 ]]; then
            health_messages+=("⚠️  API response: ${api_time}ms (slow)")
            [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
        else
            health_messages+=("❌ API response: ${api_time}ms (very slow)")
            health_status=$HEALTH_CRITICAL
        fi
    fi
    
    # Check containers
    local server_running=$(docker ps --filter "name=vrooli-judge0-server" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$server_running" ]]; then
        if echo "$server_running" | grep -q "healthy\|Up"; then
            health_messages+=("✅ Server container: running")
        else
            health_messages+=("⚠️  Server container: $server_running")
            [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
        fi
    else
        health_messages+=("❌ Server container: not found")
        health_status=$HEALTH_CRITICAL
    fi
    
    # Check workers
    local worker_count=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" 2>/dev/null | wc -l)
    if [[ $worker_count -gt 0 ]]; then
        health_messages+=("✅ Workers: $worker_count active")
    else
        health_messages+=("❌ Workers: none active")
        health_status=$HEALTH_CRITICAL
    fi
    
    # Check database
    local db_running=$(docker ps --filter "name=judge0-server-db" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$db_running" ]]; then
        health_messages+=("✅ Database: running")
    else
        health_messages+=("❌ Database: not running")
        health_status=$HEALTH_CRITICAL
    fi
    
    # Check Redis
    local redis_running=$(docker ps --filter "name=judge0-server-redis" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$redis_running" ]]; then
        health_messages+=("✅ Redis: running")
    else
        health_messages+=("❌ Redis: not running")
        health_status=$HEALTH_CRITICAL
    fi
    
    # Check memory usage
    local server_mem=$(docker stats --no-stream --format "{{.MemPerc}}" vrooli-judge0-server 2>/dev/null | head -1 | tr -d '%')
    if [[ -n "$server_mem" ]]; then
        local mem_int=${server_mem%%.*}
        if [[ $mem_int -lt 80 ]]; then
            health_messages+=("✅ Memory usage: ${server_mem}%")
        elif [[ $mem_int -lt 90 ]]; then
            health_messages+=("⚠️  Memory usage: ${server_mem}% (high)")
            [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
        else
            health_messages+=("❌ Memory usage: ${server_mem}% (critical)")
            health_status=$HEALTH_CRITICAL
        fi
    fi
    
    # Test code execution capability
    if [[ "$health_status" -ne $HEALTH_CRITICAL ]]; then
        # First try direct executor
        local direct_exec_test=$("${SCRIPT_DIR}/direct-executor.sh" execute \
            "python3" \
            'print("health_ok")' \
            "" \
            2 \
            128 2>/dev/null || echo "FAILED")
        
        if [[ "$direct_exec_test" != "FAILED" ]] && echo "$direct_exec_test" | grep -q "health_ok"; then
            health_messages+=("✅ Code execution: working (direct executor)")
        else
            # Try Judge0 API
            local exec_test=$(timeout 10 curl -sf -X POST "${api_url}/submissions?wait=true" \
                -H "Content-Type: application/json" \
                -H "X-Auth-Token: ${JUDGE0_API_KEY:-}" \
                -d '{"source_code": "print(1)", "language_id": 71}' 2>/dev/null || echo "FAILED")
            
            if [[ "$exec_test" != "FAILED" ]]; then
                local status_id=$(echo "$exec_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
                if [[ $status_id -eq 3 ]]; then
                    health_messages+=("✅ Code execution: working (Judge0 API)")
                elif [[ $status_id -eq 13 ]]; then
                    health_messages+=("⚠️  Code execution: internal error (isolate issue)")
                    [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
                else
                    health_messages+=("⚠️  Code execution: status $status_id")
                    [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
                fi
            else
                health_messages+=("❌ Code execution: failed")
                health_status=$HEALTH_CRITICAL
            fi
        fi
    fi
    
    # Save results to cache
    if [[ "$use_cache" == "true" ]]; then
        echo "{\"status\":$health_status,\"timestamp\":$(date +%s),\"messages\":$(printf '%s\n' "${health_messages[@]}" | jq -R . | jq -s .)}" > "$HEALTH_CACHE_FILE"
    fi
    
    # Output results
    if [[ "$verbose" == "true" ]]; then
        echo "Judge0 Health Diagnostics"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        for msg in "${health_messages[@]}"; do
            echo "  $msg"
        done
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        case $health_status in
            $HEALTH_OK)
                echo "Overall Status: ✅ HEALTHY"
                ;;
            $HEALTH_WARNING)
                echo "Overall Status: ⚠️  WARNING"
                ;;
            $HEALTH_CRITICAL)
                echo "Overall Status: ❌ CRITICAL"
                ;;
        esac
    fi
    
    return $health_status
}

# Quick health check for startup
judge0::health::quick() {
    # First try cache
    if judge0::health::check_cache &>/dev/null; then
        return 0
    fi
    
    # Otherwise do quick API check
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    timeout 2 curl -sf --max-time 1 "${api_url}/system_info" &>/dev/null
}

# Wait for service to become healthy
judge0::health::wait() {
    local max_wait="${1:-60}"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local elapsed=0
    
    log "Waiting for Judge0 to become healthy..."
    
    while [[ $elapsed -lt $max_wait ]]; do
        if judge0::health::quick; then
            log "✅ Judge0 is healthy!"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo
    log "❌ Judge0 failed to become healthy after ${max_wait}s"
    return 1
}

# Get detailed health metrics with performance tracking
judge0::health::metrics() {
    local verbose="${1:-false}"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local metrics=()
    local timestamp=$(date +%s)
    
    # API response time
    local start_time=$(date +%s%N)
    timeout 1 curl -sf "${api_url}/system_info" &>/dev/null
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    metrics+=("api_response_time_ms:$response_time")
    
    # Worker count
    local worker_count=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" 2>/dev/null | wc -l)
    metrics+=("worker_count:$worker_count")
    
    # Memory usage
    local mem_usage=$(docker stats --no-stream --format "{{.MemPerc}}" vrooli-judge0-server 2>/dev/null | head -1 | tr -d '%')
    [[ -n "$mem_usage" ]] && metrics+=("memory_usage_percent:$mem_usage")
    
    # CPU usage
    local cpu_usage=$(docker stats --no-stream --format "{{.CPUPerc}}" vrooli-judge0-server 2>/dev/null | head -1 | tr -d '%')
    [[ -n "$cpu_usage" ]] && metrics+=("cpu_usage_percent:$cpu_usage")
    
    # Language count
    local languages=$(timeout 5 curl -sf "${api_url}/languages" 2>/dev/null || echo "[]")
    local lang_count=$(echo "$languages" | python3 -c "import sys, json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
    metrics+=("available_languages:$lang_count")
    
    # Cache hit rate (if cache exists)
    if [[ -d "/tmp/judge0_exec_cache" ]]; then
        local cache_files=$(ls -1 /tmp/judge0_exec_cache 2>/dev/null | wc -l)
        metrics+=("cache_entries:$cache_files")
    fi
    
    # Save performance history
    if [[ -f "$HEALTH_PERF_FILE" ]]; then
        local perf_data=$(cat "$HEALTH_PERF_FILE" 2>/dev/null || echo "[]")
    else
        local perf_data="[]"
    fi
    
    # Add current metrics to history (keep last 100 entries)
    echo "$perf_data" | python3 -c "
import sys, json
data = json.load(sys.stdin)
data.append({'timestamp': $timestamp, 'response_time': $response_time})
if len(data) > 100:
    data = data[-100:]
print(json.dumps(data))
" > "$HEALTH_PERF_FILE" 2>/dev/null || true
    
    # Output metrics
    for metric in "${metrics[@]}"; do
        echo "$metric"
    done
}

# Get performance trend analysis
judge0::health::performance_trend() {
    if [[ ! -f "$HEALTH_PERF_FILE" ]]; then
        echo "No performance history available"
        return 1
    fi
    
    python3 -c "
import json
with open('$HEALTH_PERF_FILE', 'r') as f:
    data = json.load(f)
    if len(data) < 2:
        print('Insufficient data for trend analysis')
    else:
        times = [d['response_time'] for d in data[-10:]]
        avg = sum(times) / len(times)
        print(f'Average response time (last 10): {avg:.1f}ms')
        if times[-1] > avg * 1.5:
            print('⚠️  Performance degradation detected')
        elif times[-1] < avg * 0.7:
            print('✅ Performance improvement detected')
        else:
            print('📊 Performance stable')
" 2>/dev/null || echo "Error analyzing performance trend"
}