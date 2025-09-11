#!/usr/bin/env bash
################################################################################
# Judge0 Health Monitoring Library - Enhanced Diagnostics
# 
# Comprehensive health checks with detailed diagnostics
################################################################################

set -euo pipefail

# Health status codes
readonly HEALTH_OK=0
readonly HEALTH_WARNING=1
readonly HEALTH_CRITICAL=2

# Enhanced health check with diagnostics
judge0::health::check() {
    local verbose="${1:-false}"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local health_status=$HEALTH_OK
    local health_messages=()
    
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
        local exec_test=$(timeout 10 curl -sf -X POST "${api_url}/submissions?wait=true" \
            -H "Content-Type: application/json" \
            -d '{"source_code": "print(1)", "language_id": 92}' 2>/dev/null || echo "FAILED")
        
        if [[ "$exec_test" != "FAILED" ]]; then
            local status_id=$(echo "$exec_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
            if [[ $status_id -eq 3 ]]; then
                health_messages+=("✅ Code execution: working")
            else
                health_messages+=("⚠️  Code execution: status $status_id")
                [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
            fi
        else
            health_messages+=("❌ Code execution: failed")
            health_status=$HEALTH_CRITICAL
        fi
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
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    timeout 5 curl -sf --max-time 3 "${api_url}/system_info" &>/dev/null
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

# Get detailed health metrics
judge0::health::metrics() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local metrics=()
    
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
    
    # Output metrics
    for metric in "${metrics[@]}"; do
        echo "$metric"
    done
}