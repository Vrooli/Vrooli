#!/usr/bin/env bash
################################################################################
# Agent Health Monitoring
# 
# Periodic health checking and heartbeat updates for all agent-capable resources
# Lightweight background monitoring with configurable intervals
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Start background health monitor
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
#   $3 - Check interval (seconds)
#   $4 - PID file for monitor process
# Returns:
#   0 on success, 1 on error
#######################################
agents::health::start_monitor() {
    local registry_file="$1"
    local resource_name="$2"
    local check_interval="${3:-30}"
    local pidfile="$4"
    
    # Check if monitor already running
    if [[ -f "$pidfile" ]]; then
        local monitor_pid
        monitor_pid=$(cat "$pidfile" 2>/dev/null)
        if [[ -n "$monitor_pid" ]] && kill -0 "$monitor_pid" 2>/dev/null; then
            log::debug "Health monitor already running for $resource_name (PID: $monitor_pid)"
            return 0
        fi
    fi
    
    # Start monitor in background
    (
        echo $$ > "$pidfile"
        trap "rm -f '$pidfile'" EXIT
        
        agents::health::monitor_loop "$registry_file" "$resource_name" "$check_interval"
    ) &
    
    local monitor_pid=$!
    echo "$monitor_pid" > "$pidfile"
    log::info "Started health monitor for $resource_name (PID: $monitor_pid, interval: ${check_interval}s)"
    
    return 0
}

#######################################
# Main monitoring loop
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
#   $3 - Check interval (seconds)
# Note: Runs until killed
#######################################
agents::health::monitor_loop() {
    local registry_file="$1"
    local resource_name="$2"
    local check_interval="$3"
    
    log::debug "Health monitor loop started for $resource_name"
    
    while true; do
        # Get list of agents to check
        if [[ -f "$registry_file" ]]; then
            local agent_ids
            agent_ids=$(jq -r '.agents | keys[]' "$registry_file" 2>/dev/null || echo "")
            
            if [[ -n "$agent_ids" ]]; then
                while IFS= read -r agent_id; do
                    [[ -n "$agent_id" ]] || continue
                    agents::health::check_agent "$registry_file" "$resource_name" "$agent_id"
                done <<< "$agent_ids"
            fi
        fi
        
        sleep "$check_interval"
    done
}

#######################################
# Health check a single agent
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
#   $3 - Agent ID
# Returns:
#   0 if healthy, 1 if unhealthy
#######################################
agents::health::check_agent() {
    local registry_file="$1"
    local resource_name="$2"
    local agent_id="$3"
    
    # Get agent data
    local agent_data
    agent_data=$(jq --arg id "$agent_id" '.agents[$id] // empty' "$registry_file" 2>/dev/null)
    
    if [[ -z "$agent_data" || "$agent_data" == "null" ]]; then
        return 1
    fi
    
    local pid
    pid=$(echo "$agent_data" | jq -r '.pid')
    
    if [[ -z "$pid" || "$pid" == "null" ]]; then
        return 1
    fi
    
    # Basic health check: Is process running?
    if kill -0 "$pid" 2>/dev/null; then
        # Process is alive - update heartbeat
        agents::health::update_heartbeat "$registry_file" "$agent_id" "healthy"
        
        # Collect basic metrics while we're here
        agents::health::collect_process_metrics "$registry_file" "$agent_id" "$pid"
        
        return 0
    else
        # Process is dead - mark as unhealthy
        agents::health::update_heartbeat "$registry_file" "$agent_id" "unhealthy"
        log::warn "Agent $agent_id (PID: $pid) is not running"
        
        return 1
    fi
}

#######################################
# Update agent heartbeat and health status
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Health status (healthy|unhealthy|degraded)
# Returns:
#   0 on success, 1 on error
#######################################
agents::health::update_heartbeat() {
    local registry_file="$1"
    local agent_id="$2"
    local health_status="${3:-healthy}"
    
    [[ -f "$registry_file" ]] || return 1
    
    local current_time
    current_time=$(date -Iseconds)
    
    # Create temporary file for atomic update
    local temp_file="${registry_file}.tmp.$$"
    
    # Update last_seen and health status
    if ! jq --arg id "$agent_id" \
            --arg time "$current_time" \
            --arg health "$health_status" \
            '
            if .agents[$id] then
                .agents[$id].last_seen = $time |
                .agents[$id].health = $health |
                if .agents[$id].health_checks == null then
                    .agents[$id].health_checks = {}
                else . end |
                .agents[$id].health_checks.last_check = $time |
                if $health == "healthy" then
                    .agents[$id].health_checks.consecutive_failures = 0 |
                    .agents[$id].health_checks.last_success = $time
                else
                    .agents[$id].health_checks.consecutive_failures = ((.agents[$id].health_checks.consecutive_failures // 0) + 1)
                end
            else . end
            ' "$registry_file" > "$temp_file"; then
        rm -f "$temp_file"
        return 1
    fi
    
    # Atomically replace registry file
    if ! mv "$temp_file" "$registry_file"; then
        rm -f "$temp_file"
        return 1
    fi
    
    return 0
}

#######################################
# Collect basic process metrics
# Arguments:
#   $1 - Registry file path
#   $2 - Agent ID
#   $3 - Process ID
# Returns:
#   0 on success, 1 on error
#######################################
agents::health::collect_process_metrics() {
    local registry_file="$1"
    local agent_id="$2"
    local pid="$3"
    
    # Skip if process info not available
    [[ -f "/proc/$pid/stat" ]] || return 1
    
    # Get CPU usage (simplified - just user + system time)
    local stat_data
    stat_data=$(cat "/proc/$pid/stat" 2>/dev/null) || return 1
    
    # Fields 14 (utime) and 15 (stime) in clock ticks
    local utime stime
    utime=$(echo "$stat_data" | awk '{print $14}')
    stime=$(echo "$stat_data" | awk '{print $15}')
    local total_time=$((utime + stime))
    
    # Get memory usage from /proc/pid/status
    local memory_kb=0
    if [[ -f "/proc/$pid/status" ]]; then
        memory_kb=$(grep "VmRSS:" "/proc/$pid/status" 2>/dev/null | awk '{print $2}')
    fi
    local memory_mb=$((memory_kb / 1024))
    
    # Update metrics in registry
    local temp_file="${registry_file}.tmp.$$"
    local current_time
    current_time=$(date -Iseconds)
    
    if ! jq --arg id "$agent_id" \
            --argjson cpu_ticks "$total_time" \
            --argjson memory_mb "$memory_mb" \
            --arg time "$current_time" \
            '
            if .agents[$id] then
                if .agents[$id].metrics == null then
                    .agents[$id].metrics = {}
                else . end |
                if .agents[$id].metrics.gauges == null then
                    .agents[$id].metrics.gauges = {}
                else . end |
                .agents[$id].metrics.gauges.cpu_ticks = $cpu_ticks |
                .agents[$id].metrics.gauges.memory_mb = $memory_mb |
                .agents[$id].metrics.last_collected = $time
            else . end
            ' "$registry_file" > "$temp_file"; then
        rm -f "$temp_file"
        return 1
    fi
    
    mv "$temp_file" "$registry_file" || rm -f "$temp_file"
    return 0
}

#######################################
# Stop health monitor
# Arguments:
#   $1 - PID file for monitor process
# Returns:
#   0 on success, 1 on error
#######################################
agents::health::stop_monitor() {
    local pidfile="$1"
    
    if [[ ! -f "$pidfile" ]]; then
        return 0
    fi
    
    local monitor_pid
    monitor_pid=$(cat "$pidfile" 2>/dev/null)
    
    if [[ -n "$monitor_pid" ]] && kill -0 "$monitor_pid" 2>/dev/null; then
        kill "$monitor_pid" 2>/dev/null || true
        rm -f "$pidfile"
        log::info "Stopped health monitor (PID: $monitor_pid)"
    else
        rm -f "$pidfile"
    fi
    
    return 0
}

# Export functions for use by agent manager
export -f agents::health::start_monitor
export -f agents::health::monitor_loop
export -f agents::health::check_agent
export -f agents::health::update_heartbeat
export -f agents::health::collect_process_metrics
export -f agents::health::stop_monitor