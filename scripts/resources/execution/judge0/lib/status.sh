#!/usr/bin/env bash
# Judge0 Status Module
# Handles status checking and health monitoring

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