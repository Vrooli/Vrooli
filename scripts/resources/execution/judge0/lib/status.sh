#!/usr/bin/env bash
# Judge0 Status Module - Standardized Format
# Handles status checking and health monitoring with format-agnostic output

# Source format utilities
JUDGE0_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${JUDGE0_STATUS_DIR}/../../../../lib/utils/format.sh"

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
# Collect Judge0 status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
judge0::status::collect_data() {
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if judge0::is_installed; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${JUDGE0_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        if judge0::is_running; then
            running="true"
            
            if judge0::status::is_healthy; then
                healthy="true"
                health_message="Healthy - Code execution service operational with active workers"
            else
                health_message="Unhealthy - Service running but workers or API not responding"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Judge0 not found"
    fi
    
    # Basic resource information
    status_data+=("name" "judge0")
    status_data+=("category" "execution")
    status_data+=("description" "Judge0 code execution engine")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$JUDGE0_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$JUDGE0_PORT")
    
    # Service endpoints
    status_data+=("api_url" "http://localhost:$JUDGE0_PORT")
    status_data+=("version_endpoint" "http://localhost:$JUDGE0_PORT/version")
    status_data+=("submissions_endpoint" "http://localhost:$JUDGE0_PORT/submissions")
    
    # Configuration
    status_data+=("workers_configured" "$JUDGE0_WORKERS_COUNT")
    status_data+=("workers_name_prefix" "$JUDGE0_WORKERS_NAME")
    status_data+=("image" "${JUDGE0_SERVER_IMAGE:-unknown}")
    
    if [[ "$running" == "true" ]]; then
        # Count active workers
        local active_workers=0
        for i in $(seq 1 "$JUDGE0_WORKERS_COUNT"); do
            local worker_name="${JUDGE0_WORKERS_NAME}-${i}"
            if docker::is_running "$worker_name" 2>/dev/null || docker ps --format "{{.Names}}" | grep -q "^${worker_name}$"; then
                ((active_workers++))
            fi
        done
        status_data+=("workers_active" "$active_workers")
        
        # System information from API
        local system_info
        system_info=$(judge0::api::system_info 2>/dev/null || echo "{}")
        if [[ "$system_info" != "{}" ]]; then
            local version
            version=$(echo "$system_info" | jq -r '.version // "unknown"' 2>/dev/null)
            status_data+=("version" "$version")
            
            local languages_count
            languages_count=$(echo "$system_info" | jq -r '.languages | length // 0' 2>/dev/null)
            status_data+=("languages_available" "$languages_count")
            
            local cpu_limit
            cpu_limit=$(echo "$system_info" | jq -r '.cpu_time_limit // "N/A"' 2>/dev/null)
            status_data+=("cpu_time_limit" "${cpu_limit}s")
            
            local memory_limit
            memory_limit=$(echo "$system_info" | jq -r '.memory_limit // "N/A"' 2>/dev/null)
            if [[ "$memory_limit" != "N/A" ]]; then
                status_data+=("memory_limit" "$((memory_limit / 1024))MB")
            else
                status_data+=("memory_limit" "N/A")
            fi
        fi
        
        # Queue status
        local queue_info
        queue_info=$(judge0::api::get_queue_status 2>/dev/null || echo "{}")
        if [[ "$queue_info" != "{}" ]]; then
            local pending
            pending=$(echo "$queue_info" | jq -r '.pending // 0' 2>/dev/null)
            status_data+=("queue_pending" "$pending")
            
            local processing
            processing=$(echo "$queue_info" | jq -r '.processing // 0' 2>/dev/null)
            status_data+=("queue_processing" "$processing")
        fi
        
        # Submissions data
        if [[ -d "$JUDGE0_SUBMISSIONS_DIR" ]]; then
            local total_submissions
            total_submissions=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" 2>/dev/null | wc -l)
            status_data+=("total_submissions" "$total_submissions")
            
            if [[ $total_submissions -gt 0 ]]; then
                local successful
                successful=$(find "$JUDGE0_SUBMISSIONS_DIR" -type f -name "*.json" -exec grep -l '"status":{"id":3}' {} \; 2>/dev/null | wc -l)
                local success_rate
                success_rate=$(echo "scale=2; $successful * 100 / $total_submissions" | bc 2>/dev/null || echo "0")
                status_data+=("success_rate" "${success_rate}%")
            fi
        fi
        
        # Container resource stats
        local server_stats
        server_stats=$(judge0::get_container_stats 2>/dev/null || echo "{}")
        if [[ "$server_stats" != "{}" ]]; then
            local cpu_usage
            cpu_usage=$(echo "$server_stats" | jq -r '.CPUPerc // "0%"' 2>/dev/null | tr -d '%')
            status_data+=("cpu_usage" "${cpu_usage}%")
            
            local mem_usage
            mem_usage=$(echo "$server_stats" | jq -r '.MemUsage // "0MiB / 0MiB"' 2>/dev/null)
            status_data+=("memory_usage" "$mem_usage")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# CLI framework compatibility wrapper
# Args: All arguments passed to judge0::status::show
#######################################
judge0::status() {
    judge0::status::show "$@"
}

#######################################
# Show comprehensive Judge0 status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
judge0::status::show() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data
    local data_string
    data_string=$(judge0::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Judge0 status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        # Text format with standardized structure
        judge0::status::display_text "$verbose" "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}

#######################################
# Display status in text format
#######################################
judge0::status::display_text() {
    local verbose="$1"
    shift
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 Judge0 Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Judge0, run: ./manage.sh --action install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    log::info "   📋 Submissions: ${data[submissions_endpoint]:-unknown}"
    log::info "   ℹ️  Version: ${data[version_endpoint]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   🔢 Version: ${data[version]:-unknown}"
    log::info "   🗣️  Languages: ${data[languages_available]:-0} available"
    log::info "   ⏱️  CPU Limit: ${data[cpu_time_limit]:-unknown} per submission"
    log::info "   💾 Memory Limit: ${data[memory_limit]:-unknown} per submission"
    echo
    
    # Worker information (only if running)
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::info "👷 Worker Status:"
        log::info "   ✅ Active: ${data[workers_active]:-0}/${data[workers_configured]:-0}"
        log::info "   📛 Name Prefix: ${data[workers_name_prefix]:-unknown}"
        
        local active_workers="${data[workers_active]:-0}"
        local configured_workers="${data[workers_configured]:-0}"
        if [[ "$active_workers" -lt "$configured_workers" ]]; then
            log::warn "   ⚠️  Warning: Some workers are not running!"
        fi
        echo
        
        # Queue status
        log::info "📊 Queue Status:"
        log::info "   ⏳ Pending: ${data[queue_pending]:-0}"
        log::info "   ⚙️  Processing: ${data[queue_processing]:-0}"
        echo
        
        # Runtime information
        log::info "📈 Runtime Information:"
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "   🖥️  CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" ]]; then
            log::info "   💾 Memory Usage: ${data[memory_usage]}"
        fi
        log::info "   📊 Total Submissions: ${data[total_submissions]:-0}"
        if [[ -n "${data[success_rate]:-}" ]]; then
            log::info "   ✅ Success Rate: ${data[success_rate]}"
        fi
        echo
        
        # Show detailed tests if verbose
        if [[ "$verbose" == "true" ]]; then
            judge0::status::functional_test
        else
            log::info "💡 Run with --verbose to see functional tests"
        fi
    fi
}

#######################################
# Show system information (legacy function for compatibility)
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