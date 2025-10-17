#!/usr/bin/env bash
################################################################################
# Judge0 Enhanced Health Monitoring with Execution Testing
# 
# Comprehensive health checks including execution method validation
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"

# Source config
source "${SCRIPT_DIR}/../config/defaults.sh" 2>/dev/null || true

# Don't auto-execute execution-manager, just reference it
EXECUTION_MANAGER="$SCRIPT_DIR/execution-manager.sh"

# Health status codes
readonly HEALTH_OK=0
readonly HEALTH_WARNING=1
readonly HEALTH_CRITICAL=2

# Simple logging
log() {
    local level="$1"
    shift
    echo "[$(date +'%H:%M:%S')] [$level] $*" >&2
}

# Enhanced health check with execution testing
judge0::health::check_enhanced() {
    local verbose="${1:-false}"
    local api_url="http://localhost:${JUDGE0_PORT:-2358}"
    local health_status=$HEALTH_OK
    local health_messages=()
    
    echo "🏥 Judge0 Health Check Report"
    echo "=============================="
    
    # 1. Check API responsiveness
    echo ""
    echo "📡 API Health:"
    local api_start=$(date +%s%N)
    local api_response=$(timeout 5 curl -sf --max-time 3 "${api_url}/system_info" 2>/dev/null || echo "FAILED")
    local api_end=$(date +%s%N)
    local api_time=$(( (api_end - api_start) / 1000000 ))
    
    if [[ "$api_response" == "FAILED" ]]; then
        echo "  ❌ API not responding at ${api_url}"
        health_status=$HEALTH_CRITICAL
    else
        if [[ $api_time -lt 500 ]]; then
            echo "  ✅ API response: ${api_time}ms (excellent)"
        elif [[ $api_time -lt 1000 ]]; then
            echo "  ⚠️  API response: ${api_time}ms (acceptable)"
            [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
        else
            echo "  ❌ API response: ${api_time}ms (too slow)"
            health_status=$HEALTH_CRITICAL
        fi
        
        # Show version info if available
        local version=$(echo "$api_response" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")
        echo "  📌 Version: $version"
    fi
    
    # 2. Check containers
    echo ""
    echo "🐳 Container Status:"
    
    # Server container
    local server_running=$(docker ps --filter "name=vrooli-judge0-server" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$server_running" ]]; then
        if echo "$server_running" | grep -q "healthy\|Up"; then
            echo "  ✅ Server: running"
        else
            echo "  ⚠️  Server: $server_running"
            [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
        fi
    else
        echo "  ❌ Server: not running"
        health_status=$HEALTH_CRITICAL
    fi
    
    # Worker containers
    local worker_count=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" 2>/dev/null | wc -l)
    if [[ $worker_count -gt 0 ]]; then
        echo "  ✅ Workers: $worker_count active"
    else
        echo "  ❌ Workers: none active"
        health_status=$HEALTH_CRITICAL
    fi
    
    # Database
    local db_running=$(docker ps --filter "name=judge0-server-db" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$db_running" ]]; then
        echo "  ✅ Database: running"
    else
        echo "  ❌ Database: not running"
        health_status=$HEALTH_CRITICAL
    fi
    
    # Redis
    local redis_running=$(docker ps --filter "name=judge0-server-redis" --format "{{.Status}}" 2>/dev/null | head -1)
    if [[ -n "$redis_running" ]]; then
        echo "  ✅ Redis: running"
    else
        echo "  ❌ Redis: not running"
        health_status=$HEALTH_CRITICAL
    fi
    
    # 3. Check execution methods
    echo ""
    echo "🚀 Execution Methods:"
    
    # Test Judge0 native execution
    local judge0_works="no"
    local test_result=$(curl -sf -X POST "${api_url}/submissions" \
        -H "Content-Type: application/json" \
        -d '{"source_code": "print(1)", "language_id": 92}' 2>/dev/null || echo "failed")
    
    if [[ "$test_result" != "failed" ]] && [[ "$test_result" =~ token ]]; then
        # Try to get the result
        local token=$(echo "$test_result" | jq -r '.token' 2>/dev/null || echo "")
        if [[ -n "$token" ]]; then
            sleep 1
            local exec_result=$(curl -sf "${api_url}/submissions/${token}" 2>/dev/null || echo "{}")
            local status_id=$(echo "$exec_result" | jq -r '.status.id // 0' 2>/dev/null || echo "0")
            if [[ "$status_id" == "3" ]]; then
                echo "  ✅ Judge0 API: Working"
                judge0_works="yes"
            else
                echo "  ❌ Judge0 API: Execution fails (isolate issue)"
            fi
        else
            echo "  ❌ Judge0 API: Token generation failed"
        fi
    else
        echo "  ❌ Judge0 API: Submission failed"
    fi
    
    # Test direct executor
    if [[ -f "$SCRIPT_DIR/direct-executor.sh" ]]; then
        local direct_result=$("$SCRIPT_DIR/direct-executor.sh" execute python3 'print(1)' 2>/dev/null || echo '{"status": "error"}')
        if [[ "$direct_result" =~ \"status\":\ *\"accepted\" ]]; then
            echo "  ✅ Direct Executor: Working"
        else
            echo "  ⚠️  Direct Executor: Available but failed test"
        fi
    else
        echo "  ❌ Direct Executor: Not installed"
    fi
    
    # Test simple executor
    if [[ -f "$SCRIPT_DIR/simple-exec.sh" ]]; then
        local simple_result=$("$SCRIPT_DIR/simple-exec.sh" 'print(1)' 2>/dev/null || echo '{"status": "error"}')
        if [[ "$simple_result" =~ \"status\":\ *\"accepted\" ]]; then
            echo "  ✅ Simple Executor: Working (Python only)"
        else
            echo "  ⚠️  Simple Executor: Available but failed test"
        fi
    else
        echo "  ❌ Simple Executor: Not installed"
    fi
    
    # Check external API configuration
    if [[ -n "${JUDGE0_EXTERNAL_API_URL:-}" ]] && [[ -n "${JUDGE0_EXTERNAL_API_KEY:-}" ]]; then
        echo "  ✅ External API: Configured"
    else
        echo "  ⚠️  External API: Not configured"
    fi
    
    # 4. Check execution manager
    echo ""
    echo "🎯 Execution Manager:"
    if [[ -f "$EXECUTION_MANAGER" ]]; then
        local best_method=$("$EXECUTION_MANAGER" methods 2>/dev/null || echo "none")
        if [[ "$best_method" != "none" ]]; then
            echo "  ✅ Best method: $best_method"
            
            # Test actual execution
            local exec_test=$("$EXECUTION_MANAGER" execute python3 'print("test")' 2>/dev/null || echo '{"status": "error"}')
            if [[ "$exec_test" =~ \"status\":\ *\"accepted\" ]]; then
                echo "  ✅ Execution test: Passed"
            else
                echo "  ❌ Execution test: Failed"
            fi
        else
            echo "  ❌ No execution methods available"
            health_status=$HEALTH_CRITICAL
        fi
    else
        echo "  ⚠️  Execution manager not installed"
    fi
    
    # 5. Performance metrics
    echo ""
    echo "📊 Performance Metrics:"
    
    # Memory usage
    local server_mem=$(docker stats --no-stream --format "{{.MemPerc}}" vrooli-judge0-server 2>/dev/null | head -1 | tr -d '%')
    if [[ -n "$server_mem" ]]; then
        local mem_int=${server_mem%%.*}
        if [[ $mem_int -lt 80 ]]; then
            echo "  ✅ Memory: ${server_mem}%"
        elif [[ $mem_int -lt 90 ]]; then
            echo "  ⚠️  Memory: ${server_mem}% (high)"
            [[ $health_status -lt $HEALTH_WARNING ]] && health_status=$HEALTH_WARNING
        else
            echo "  ❌ Memory: ${server_mem}% (critical)"
            health_status=$HEALTH_CRITICAL
        fi
    fi
    
    # CPU usage
    local server_cpu=$(docker stats --no-stream --format "{{.CPUPerc}}" vrooli-judge0-server 2>/dev/null | head -1 | tr -d '%')
    if [[ -n "$server_cpu" ]]; then
        echo "  📈 CPU: ${server_cpu}%"
    fi
    
    # 6. Language support
    if [[ "$verbose" == "true" ]]; then
        echo ""
        echo "🌐 Language Support:"
        local languages=$(curl -sf "${api_url}/languages" 2>/dev/null || echo "[]")
        local lang_count=$(echo "$languages" | jq 'length' 2>/dev/null || echo "0")
        echo "  📚 Total languages: $lang_count"
        
        if [[ "$lang_count" -gt 0 ]]; then
            echo "  🔤 Sample languages:"
            echo "$languages" | jq -r '.[:5] | .[] | "    • \(.name) v\(.version)"' 2>/dev/null || echo "    Unable to parse"
        fi
    fi
    
    # 7. Overall status
    echo ""
    echo "=============================="
    case $health_status in
        $HEALTH_OK)
            echo "✅ Overall Status: HEALTHY"
            if [[ "$judge0_works" != "yes" ]]; then
                echo "⚠️  Note: Using workaround execution methods"
            fi
            ;;
        $HEALTH_WARNING)
            echo "⚠️  Overall Status: WARNING"
            echo "Some issues detected, but service is operational"
            ;;
        $HEALTH_CRITICAL)
            echo "❌ Overall Status: CRITICAL"
            echo "Service has critical issues and may not be functional"
            ;;
    esac
    echo "=============================="
    
    return $health_status
}

# Quick health check (non-verbose)
judge0::health::quick_check() {
    local api_url="http://localhost:${JUDGE0_PORT:-2358}/system_info"
    
    if timeout 2 curl -sf "$api_url" &>/dev/null; then
        echo "✅ Judge0 is healthy"
        return 0
    else
        echo "❌ Judge0 health check failed"
        return 1
    fi
}

# Performance diagnostics
judge0::health::performance_check() {
    echo "📊 Judge0 Performance Analysis"
    echo "=============================="
    
    # Test execution speed for different languages
    local languages=("python3" "javascript" "ruby")
    
    for lang in "${languages[@]}"; do
        echo ""
        echo "Testing $lang performance:"
        
        local start=$(date +%s%N)
        local result=$("$EXECUTION_MANAGER" execute "$lang" 'print("test")' 2>/dev/null || echo "failed")
        local end=$(date +%s%N)
        local time=$(( (end - start) / 1000000 ))
        
        if [[ "$result" != "failed" ]] && [[ "$result" =~ \"status\":\ *\"accepted\" ]]; then
            echo "  ✅ Execution time: ${time}ms"
        else
            echo "  ❌ Execution failed"
        fi
    done
    
    echo ""
    echo "=============================="
}

# Main execution
case "${1:-check}" in
    check)
        judge0::health::check_enhanced "${2:-false}"
        ;;
    quick)
        judge0::health::quick_check
        ;;
    performance)
        judge0::health::performance_check
        ;;
    *)
        echo "Usage: $0 {check|quick|performance} [verbose]"
        exit 1
        ;;
esac