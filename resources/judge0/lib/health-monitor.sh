#!/usr/bin/env bash
################################################################################
# Judge0 Optimized Health Monitoring System
# 
# Provides ultra-fast health checks with intelligent caching and
# background metrics collection for minimal latency
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"

# Source dependencies
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true
source "${SCRIPT_DIR}/health-cache.sh" 2>/dev/null || true

# Response time targets (ms)
readonly TARGET_QUICK_RESPONSE=5
readonly TARGET_DETAILED_RESPONSE=15
readonly TARGET_METRICS_RESPONSE=50

# Health check modes
readonly MODE_QUICK="quick"
readonly MODE_DETAILED="detailed"
readonly MODE_FULL="full"
readonly MODE_METRICS="metrics"

# Format output as JSON
format_json_response() {
    local mode="$1"
    local data="$2"
    local response_time="$3"
    
    cat <<EOF
{
    "mode": "$mode",
    "response_time_ms": $response_time,
    "target_ms": $(get_target_time "$mode"),
    "data": $data
}
EOF
}

# Get target response time for mode
get_target_time() {
    local mode="$1"
    
    case "$mode" in
        quick)
            echo $TARGET_QUICK_RESPONSE
            ;;
        detailed)
            echo $TARGET_DETAILED_RESPONSE
            ;;
        metrics)
            echo $TARGET_METRICS_RESPONSE
            ;;
        *)
            echo 100
            ;;
    esac
}

# Ultra-fast health check (<5ms target)
judge0::health::quick() {
    local start=$(date +%s%N)
    
    # Use cached quick health check
    local health_data=$(quick_health_check)
    
    local end=$(date +%s%N)
    local response_time=$(( (end - start) / 1000000 ))
    
    # Extract healthy status
    local healthy=$(echo "$health_data" | jq -r '.healthy')
    
    # Format response
    if [[ "${OUTPUT_FORMAT:-}" == "json" ]]; then
        format_json_response "$MODE_QUICK" "$health_data" "$response_time"
    else
        if [[ "$healthy" == "true" ]]; then
            echo "✅ Judge0 healthy (${response_time}ms)"
        else
            echo "❌ Judge0 unhealthy (${response_time}ms)"
        fi
    fi
    
    # Return appropriate exit code
    [[ "$healthy" == "true" ]] && return 0 || return 1
}

# Detailed health check (<15ms target) 
judge0::health::detailed() {
    local start=$(date +%s%N)
    
    # Use cached detailed health check
    local health_data=$(detailed_health_check)
    
    local end=$(date +%s%N)
    local response_time=$(( (end - start) / 1000000 ))
    
    # Format response
    if [[ "${OUTPUT_FORMAT:-}" == "json" ]]; then
        format_json_response "$MODE_DETAILED" "$health_data" "$response_time"
    else
        echo "📊 Judge0 Detailed Health Report"
        echo "=================================="
        echo "Response time: ${response_time}ms (target: <${TARGET_DETAILED_RESPONSE}ms)"
        echo ""
        
        # Parse and display details
        local api_version=$(echo "$health_data" | jq -r '.api.version')
        local languages=$(echo "$health_data" | jq -r '.api.languages_available')
        local server_status=$(echo "$health_data" | jq -r '.containers.server')
        local worker_count=$(echo "$health_data" | jq -r '.containers.workers')
        
        echo "API:"
        echo "  Version: $api_version"
        echo "  Languages: $languages"
        echo ""
        echo "Containers:"
        echo "  Server: $server_status"
        echo "  Workers: $worker_count"
        
        # Check execution methods
        local judge0_works=$(echo "$health_data" | jq -r '.execution_methods.judge0')
        local direct_works=$(echo "$health_data" | jq -r '.execution_methods.direct')
        local external_works=$(echo "$health_data" | jq -r '.execution_methods.external')
        
        echo ""
        echo "Execution Methods:"
        [[ "$judge0_works" == "true" ]] && echo "  ✅ Judge0 native" || echo "  ❌ Judge0 native"
        [[ "$direct_works" == "true" ]] && echo "  ✅ Direct executor" || echo "  ❌ Direct executor"
        [[ "$external_works" == "true" ]] && echo "  ✅ External API" || echo "  ⚠️  External API (not configured)"
    fi
}

# Full health check with performance metrics
judge0::health::full() {
    local start=$(date +%s%N)
    
    # Gather all health data
    local quick_data=$(quick_health_check)
    local detailed_data=$(detailed_health_check)
    local metrics_data=$(performance_metrics)
    
    local end=$(date +%s%N)
    local response_time=$(( (end - start) / 1000000 ))
    
    # Combine all data
    local full_data=$(cat <<EOF
{
    "quick": $quick_data,
    "detailed": $detailed_data,
    "metrics": $metrics_data
}
EOF
    )
    
    if [[ "${OUTPUT_FORMAT:-}" == "json" ]]; then
        format_json_response "$MODE_FULL" "$full_data" "$response_time"
    else
        echo "🏥 Judge0 Complete Health Report"
        echo "=================================="
        echo "Response time: ${response_time}ms"
        echo ""
        
        # Quick status
        local healthy=$(echo "$quick_data" | jq -r '.healthy')
        local api_response=$(echo "$quick_data" | jq -r '.response_time_ms')
        
        echo "Quick Status:"
        if [[ "$healthy" == "true" ]]; then
            echo "  ✅ Service healthy"
        else
            echo "  ❌ Service unhealthy"
        fi
        echo "  API response: ${api_response}ms"
        
        # Detailed info
        echo ""
        echo "Service Details:"
        local api_version=$(echo "$detailed_data" | jq -r '.api.version')
        local languages=$(echo "$detailed_data" | jq -r '.api.languages_available')
        echo "  Version: $api_version"
        echo "  Languages: $languages"
        
        # Performance metrics
        echo ""
        echo "Performance Metrics:"
        local cpu=$(echo "$metrics_data" | jq -r '.resource_usage.cpu_percent')
        local mem=$(echo "$metrics_data" | jq -r '.resource_usage.memory_percent')
        local python_ms=$(echo "$metrics_data" | jq -r '.execution_performance.python_ms')
        local js_ms=$(echo "$metrics_data" | jq -r '.execution_performance.javascript_ms')
        
        echo "  CPU: ${cpu}%"
        echo "  Memory: ${mem}%"
        echo "  Python execution: ${python_ms}ms"
        echo "  JavaScript execution: ${js_ms}ms"
        
        # Cache stats
        echo ""
        echo "Cache Statistics:"
        local cache_files=$(echo "$metrics_data" | jq -r '.cache.files')
        local cache_size=$(echo "$metrics_data" | jq -r '.cache.size_bytes')
        echo "  Files: $cache_files"
        echo "  Size: $((cache_size / 1024))KB"
    fi
}

# Performance trend analysis
judge0::health::trends() {
    local history_file="${CACHE_DIR}/performance_history.jsonl"
    local max_entries=100
    
    # Collect current metrics
    local metrics=$(performance_metrics)
    local timestamp=$(date +%s)
    
    # Add to history
    echo "{\"timestamp\": $timestamp, \"metrics\": $metrics}" >> "$history_file"
    
    # Keep only last N entries
    if [[ -f "$history_file" ]]; then
        tail -n "$max_entries" "$history_file" > "${history_file}.tmp"
        mv "${history_file}.tmp" "$history_file"
    fi
    
    # Calculate trends
    if [[ $(wc -l < "$history_file" 2>/dev/null || echo 0) -ge 10 ]]; then
        echo "📈 Performance Trends"
        echo "===================="
        
        # Calculate average response times
        local avg_python=$(tail -10 "$history_file" | jq -s 'map(.metrics.execution_performance.python_ms) | add/length' 2>/dev/null || echo 0)
        local avg_js=$(tail -10 "$history_file" | jq -s 'map(.metrics.execution_performance.javascript_ms) | add/length' 2>/dev/null || echo 0)
        
        echo "Last 10 executions average:"
        echo "  Python: ${avg_python}ms"
        echo "  JavaScript: ${avg_js}ms"
        
        # Check for performance degradation
        local first_python=$(head -1 "$history_file" | jq -r '.metrics.execution_performance.python_ms' 2>/dev/null || echo 0)
        local last_python=$(tail -1 "$history_file" | jq -r '.metrics.execution_performance.python_ms' 2>/dev/null || echo 0)
        
        if [[ $last_python -gt $((first_python * 2)) ]]; then
            echo "  ⚠️  Warning: Python execution time has degraded significantly"
        fi
    else
        echo "Not enough data for trend analysis (need 10+ data points)"
    fi
}

# Background health monitoring daemon
judge0::health::monitor_daemon() {
    local interval="${1:-30}"
    local pid_file="/tmp/judge0_health_monitor.pid"
    
    # Check if already running
    if [[ -f "$pid_file" ]] && kill -0 $(cat "$pid_file") 2>/dev/null; then
        echo "Health monitor already running (PID: $(cat $pid_file))"
        return 1
    fi
    
    # Start background monitoring
    echo "Starting health monitor daemon (interval: ${interval}s)"
    
    (
        echo $$ > "$pid_file"
        
        while true; do
            # Collect all health data
            quick_health_check > /dev/null 2>&1
            detailed_health_check > /dev/null 2>&1
            performance_metrics > /dev/null 2>&1
            
            # Sleep for interval
            sleep "$interval"
        done
    ) &
    
    echo "Health monitor started (PID: $!)"
}

# Stop health monitoring daemon
judge0::health::stop_monitor() {
    local pid_file="/tmp/judge0_health_monitor.pid"
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f "$pid_file"
            echo "Health monitor stopped"
        else
            echo "Health monitor not running"
            rm -f "$pid_file"
        fi
    else
        echo "Health monitor not running"
    fi
}

# Main execution - only if run directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-quick}" in
        quick)
            judge0::health::quick
            ;;
        detailed)
            judge0::health::detailed
            ;;
        full)
            judge0::health::full
            ;;
        trends)
            judge0::health::trends
            ;;
        monitor)
            judge0::health::monitor_daemon "${2:-30}"
            ;;
        stop-monitor)
            judge0::health::stop_monitor
            ;;
        clear-cache)
            clear_cache "${2:-all}"
            ;;
        *)
            echo "Usage: $0 {quick|detailed|full|trends|monitor [interval]|stop-monitor|clear-cache [level]}"
            echo ""
            echo "Modes:"
            echo "  quick        - Ultra-fast health check (<5ms target)"
            echo "  detailed     - Detailed health status (<15ms target)"
            echo "  full         - Complete health report with metrics"
            echo "  trends       - Performance trend analysis"
            echo "  monitor      - Start background monitoring daemon"
            echo "  stop-monitor - Stop monitoring daemon"
            echo "  clear-cache  - Clear health cache"
            echo ""
            echo "Environment variables:"
            echo "  OUTPUT_FORMAT=json - Output in JSON format"
            exit 1
            ;;
    esac
fi