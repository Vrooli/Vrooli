#!/usr/bin/env bash
# Judge0 Status Module
# Handles status checking and health monitoring

# Load dependencies if not already loaded
if [[ -z "${JUDGE0_WORKERS_COUNT:-}" ]]; then
    # Try to load config from relative path
    JUDGE0_PARENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
    if [[ -f "${JUDGE0_PARENT_DIR}/config/defaults.sh" ]]; then
        # shellcheck disable=SC1091
        source "${JUDGE0_PARENT_DIR}/config/defaults.sh"
        judge0::export_config
    fi
fi

# Define fallback functions if not loaded
if ! declare -f judge0::is_running >/dev/null 2>&1; then
    judge0::is_running() {
        if declare -f docker::is_running >/dev/null 2>&1; then
            docker::is_running "${JUDGE0_CONTAINER_NAME:-vrooli-judge0-server}"
        else
            # Fallback check
            docker ps --format "{{.Names}}" | grep -q "^${JUDGE0_CONTAINER_NAME:-vrooli-judge0-server}$"
        fi
    }
fi

if ! declare -f judge0::api::health_check >/dev/null 2>&1; then
    judge0::api::health_check() {
        local port="${JUDGE0_PORT:-2358}"
        curl -f -s --max-time 5 "http://localhost:${port}/version" >/dev/null 2>&1
    }
fi

#######################################
# Show comprehensive Judge0 status
#######################################
judge0::status::show() {
    log::header "$JUDGE0_MSG_STATUS_CHECKING"
    
    # Basic status
    if ! judge0::is_installed; then
        log::error "$JUDGE0_MSG_STATUS_NOT_INSTALLED"
        echo
        echo "Run '$0 --action install' to install Judge0"
        return 1
    fi
    
    if ! judge0::is_running; then
        log::warning "$JUDGE0_MSG_STATUS_STOPPED"
        echo
        echo "Run '$0 --action start' to start Judge0"
        return 1
    fi
    
    log::success "$JUDGE0_MSG_STATUS_RUNNING"
    echo
    
    # System info
    judge0::status::show_system_info
    
    # Worker status
    judge0::status::show_workers
    
    # Queue status
    judge0::status::show_queue
    
    # Resource usage
    judge0::status::show_resources
    
    # Recent submissions
    judge0::status::show_recent_submissions
    
    # Functional test
    judge0::status::functional_test
    
    return 0
}

#######################################
# Show system information
#######################################
judge0::status::show_system_info() {
    log::info "📊 System Information:"
    
    local system_info=$(judge0::api::system_info 2>/dev/null || echo "{}")
    
    if [[ "$system_info" == "{}" ]]; then
        log::error "  Failed to retrieve system info"
        return 1
    fi
    
    # Parse and display system info
    local version=$(echo "$system_info" | jq -r '.version // "unknown"')
    local languages_count=$(echo "$system_info" | jq -r '.languages | length // 0')
    local cpu_time_limit=$(echo "$system_info" | jq -r '.cpu_time_limit // "N/A"')
    local memory_limit=$(echo "$system_info" | jq -r '.memory_limit // "N/A"')
    
    echo "  Version: $version"
    echo "  Languages: $languages_count available"
    echo "  CPU Limit: ${cpu_time_limit}s per submission"
    echo "  Memory Limit: $((memory_limit / 1024))MB per submission"
    echo
}

#######################################
# Show worker status
#######################################
judge0::status::show_workers() {
    log::info "👷 Worker Status:"
    
    local active_workers=0
    for i in $(seq 1 "$JUDGE0_WORKERS_COUNT"); do
        local worker_name="${JUDGE0_WORKERS_NAME}-${i}"
        if docker::is_running "$worker_name"; then
            ((active_workers++))
        fi
    done
    
    printf "  $JUDGE0_MSG_STATUS_WORKERS\n" "$active_workers"
    echo "  Configured: $JUDGE0_WORKERS_COUNT"
    
    if [[ $active_workers -lt $JUDGE0_WORKERS_COUNT ]]; then
        log::warning "  Some workers are not running!"
    fi
    echo
}

#######################################
# Show queue status
#######################################
judge0::status::show_queue() {
    log::info "📊 Queue Status:"
    
    # Get queue info from API
    local queue_info=$(judge0::api::get_queue_status 2>/dev/null || echo "{}")
    
    if [[ "$queue_info" == "{}" ]]; then
        echo "  Unable to retrieve queue status"
    else
        local pending=$(echo "$queue_info" | jq -r '.pending // 0')
        local processing=$(echo "$queue_info" | jq -r '.processing // 0')
        
        printf "  $JUDGE0_MSG_STATUS_QUEUE\n" "$pending"
        echo "  Processing: $processing"
    fi
    echo
}

#######################################
# Show resource usage
#######################################
judge0::status::show_resources() {
    log::info "💻 Resource Usage:"
    
    # Server container stats
    local server_stats=$(judge0::get_container_stats)
    if [[ "$server_stats" != "{}" ]]; then
        local cpu_usage=$(echo "$server_stats" | jq -r '.CPUPerc // "0%"' | tr -d '%')
        local mem_usage=$(echo "$server_stats" | jq -r '.MemUsage // "0MiB / 0MiB"')
        
        echo "  Server:"
        printf "    $JUDGE0_MSG_USAGE_CPU\n" "$cpu_usage"
        printf "    $JUDGE0_MSG_USAGE_MEMORY\n" "$mem_usage"
    fi
    
    # Worker stats summary
    local worker_stats=$(judge0::get_worker_stats)
    if [[ "$worker_stats" != "[]" ]]; then
        local total_cpu=0
        local worker_count=0
        
        # Calculate average CPU usage
        while IFS= read -r line; do
            local cpu=$(echo "$line" | jq -r '.CPUPerc // "0%"' | tr -d '%')
            total_cpu=$(echo "$total_cpu + $cpu" | bc)
            ((worker_count++))
        done < <(echo "$worker_stats" | jq -c '.[]')
        
        if [[ $worker_count -gt 0 ]]; then
            local avg_cpu=$(echo "scale=2; $total_cpu / $worker_count" | bc)
            echo "  Workers (average):"
            printf "    $JUDGE0_MSG_USAGE_CPU\n" "$avg_cpu"
        fi
    fi
    echo
}

#######################################
# Show recent submissions
#######################################
judge0::status::show_recent_submissions() {
    log::info "📈 Recent Activity:"
    
    # Check submissions directory
    if [[ -d "$JUDGE0_SUBMISSIONS_DIR" ]]; then
        local total_submissions=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" 2>/dev/null | wc -l)
        printf "  $JUDGE0_MSG_USAGE_SUBMISSIONS\n" "$total_submissions"
        
        # Calculate success rate (simplified - in real implementation would query API)
        if [[ $total_submissions -gt 0 ]]; then
            local successful=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" -exec grep -l '"status":{"id":3}' {} \; 2>/dev/null | wc -l)
            local success_rate=$(echo "scale=2; $successful * 100 / $total_submissions" | bc)
            printf "  $JUDGE0_MSG_USAGE_SUCCESS_RATE\n" "$success_rate"
        fi
    else
        echo "  No submission data available"
    fi
    echo
}

#######################################
# Check if Judge0 is healthy
# Returns:
#   0 if healthy, 1 if not
#######################################
judge0::status::is_healthy() {
    if ! judge0::is_running; then
        return 1
    fi
    
    # Check API health
    if ! judge0::api::health_check >/dev/null 2>&1; then
        return 1
    fi
    
    # Check workers
    local active_workers=0
    for i in $(seq 1 "$JUDGE0_WORKERS_COUNT"); do
        local worker_name="${JUDGE0_WORKERS_NAME}-${i}"
        if docker::is_running "$worker_name"; then
            ((active_workers++))
        fi
    done
    
    if [[ $active_workers -eq 0 ]]; then
        return 1
    fi
    
    return 0
}

#######################################
# Get Judge0 health status as JSON
# Returns:
#   JSON object with health status
#######################################
judge0::status::get_health_json() {
    local health_status="unhealthy"
    local api_status="down"
    local workers_active=0
    local details=""
    
    if judge0::is_running; then
        # Check API
        if judge0::api::health_check >/dev/null 2>&1; then
            api_status="up"
        fi
        
        # Count active workers
        for i in $(seq 1 "$JUDGE0_WORKERS_COUNT"); do
            local worker_name="${JUDGE0_WORKERS_NAME}-${i}"
            if docker::is_running "$worker_name"; then
                ((workers_active++))
            fi
        done
        
        if [[ "$api_status" == "up" ]] && [[ $workers_active -gt 0 ]]; then
            health_status="healthy"
        else
            details="API: $api_status, Workers: $workers_active/$JUDGE0_WORKERS_COUNT"
        fi
    else
        details="Service not running"
    fi
    
    # Construct JSON
    cat <<EOF
{
  "status": "$health_status",
  "api": "$api_status",
  "workers": {
    "active": $workers_active,
    "configured": $JUDGE0_WORKERS_COUNT
  },
  "details": "$details",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
}

#######################################
# Functional test to detect common issues
#######################################
judge0::status::functional_test() {
    log::header "🧪 Functional Test"
    echo "Testing core Judge0 functionality..."
    echo
    
    # Get auth token from docker-compose config
    local auth_token=$(grep "JUDGE0_AUTHENTICATION_TOKEN=" /home/matthalloran8/.vrooli/resources/judge0/config/docker-compose.yml | head -1 | cut -d'=' -f2)
    
    # Test 1: Simple echo test (should always work)
    echo "1. Testing basic API connectivity..."
    local echo_token=$(curl -s -X POST "http://localhost:${JUDGE0_PORT}/submissions" \
        -H "X-Auth-Token: ${auth_token}" \
        -H "Content-Type: application/json" \
        -d '{"source_code": "echo \"Hello World\"", "language_id": 46}' | jq -r '.token // "error"')
    
    if [[ "$echo_token" == "error" ]] || [[ -z "$echo_token" ]]; then
        log::error "❌ API connectivity test failed"
        return 1
    fi
    
    sleep 3
    local echo_result=$(curl -s "http://localhost:${JUDGE0_PORT}/submissions/$echo_token" \
        -H "X-Auth-Token: ${auth_token}" | jq -r '.status.id // "error"')
    
    case "$echo_result" in
        "3")
            log::success "✅ Basic execution working (Bash echo)"
            ;;
        "1"|"2")
            log::warning "⏳ Submission still processing (this may indicate performance issues)"
            ;;
        "5")
            log::error "❌ Time limit exceeded (performance issue detected)"
            echo "   💡 Consider increasing time limits for containerized environments"
            ;;
        "13")
            log::error "❌ Internal error (likely isolate/namespace issue)"
            echo "   💡 Check kernel parameters and container privileges"
            ;;
        *)
            log::warning "⚠️  Unexpected status: $echo_result"
            ;;
    esac
    echo
    
    # Test 2: JavaScript/Node.js test (common bottleneck)
    echo "2. Testing Node.js performance..."
    local js_token=$(curl -s -X POST "http://localhost:${JUDGE0_PORT}/submissions" \
        -H "X-Auth-Token: ${auth_token}" \
        -H "Content-Type: application/json" \
        -d '{"source_code": "console.log(\"Hello Node\");", "language_id": 63, "cpu_time_limit": 5, "wall_time_limit": 10}' | jq -r '.token // "error"')
    
    if [[ "$js_token" != "error" ]] && [[ -n "$js_token" ]]; then
        sleep 5
        local js_result=$(curl -s "http://localhost:${JUDGE0_PORT}/submissions/$js_token" \
            -H "X-Auth-Token: ${auth_token}")
        
        local js_status=$(echo "$js_result" | jq -r '.status.id // "error"')
        local js_time=$(echo "$js_result" | jq -r '.time // "0"')
        local js_stderr=$(echo "$js_result" | jq -r '.stderr // ""')
        
        case "$js_status" in
            "3")
                log::success "✅ Node.js execution working (time: ${js_time}s)"
                if (( $(echo "$js_time > 3" | bc -l) )); then
                    log::warning "   ⚠️  Slow startup detected (${js_time}s) - consider optimizing"
                fi
                ;;
            "5")
                log::error "❌ Node.js timeout (containerization performance issue)"
                echo "   💡 Increase wall_time_limit or optimize Node.js environment"
                ;;
            "13")
                if echo "$js_stderr" | grep -q "clone failed"; then
                    log::error "❌ Clone/namespace permission denied"
                    echo "   💡 Enable unprivileged_userns_clone or use privileged containers"
                else
                    log::error "❌ Node.js internal error: $js_stderr"
                fi
                ;;
            "1"|"2")
                log::warning "⏳ Node.js submission still processing (worker issue?)"
                ;;
            *)
                log::warning "⚠️  Node.js unexpected status: $js_status"
                ;;
        esac
    else
        log::error "❌ Failed to submit Node.js test"
    fi
    echo
    
    # Test 3: Worker health
    echo "3. Testing worker health..."
    local worker_count=0
    local containers=$(docker ps --filter "name=judge0-workers" --format "{{.Names}}" 2>/dev/null)
    
    for container in $containers; do
        ((worker_count++))
        local processes=$(docker exec "$container" ps aux | grep -c "rake resque:work" 2>/dev/null || echo "0")
        if [[ "$processes" -gt 10 ]]; then
            log::warning "⚠️  Container $container has excessive worker processes ($processes)"
            echo "   💡 Workers may be crashing and restarting - check isolate functionality"
        elif [[ "$processes" -eq 0 ]]; then
            log::error "❌ Container $container has no active workers"
        else
            log::success "✅ Container $container has $processes workers"
        fi
    done
    
    if [[ "$worker_count" -eq 0 ]]; then
        log::error "❌ No worker containers found"
        return 1
    fi
    echo
    
    # Test 4: Isolate direct test
    echo "4. Testing isolate sandbox directly..."
    if docker exec "${containers%% *}" /usr/local/bin/isolate --init >/dev/null 2>&1; then
        if docker exec "${containers%% *}" timeout 5 /usr/local/bin/isolate --run -- /bin/echo "test" >/dev/null 2>&1; then
            log::success "✅ Isolate sandbox working"
        else
            log::error "❌ Isolate execution fails (namespace/clone issue)"
            echo "   💡 This is the root cause of Internal Error (13) submissions"
        fi
        docker exec "${containers%% *}" /usr/local/bin/isolate --cleanup >/dev/null 2>&1
    else
        log::error "❌ Isolate initialization fails"
    fi
    echo
    
    echo "Functional test complete. Check issues above for troubleshooting guidance."
}