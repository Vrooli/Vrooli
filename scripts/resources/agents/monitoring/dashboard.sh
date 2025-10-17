#!/usr/bin/env bash
################################################################################
# Agent Monitoring Dashboard
# 
# Real-time terminal dashboard for agent monitoring
# Displays health, metrics, and events in a clean interface
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

#######################################
# Display monitoring dashboard
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
#   $3 - Refresh interval (seconds)
#   $4 - Output format (terminal|json)
# Returns:
#   0 on success, 1 on error
#######################################
agents::dashboard::display() {
    local registry_file="$1"
    local resource_name="$2"
    local refresh_interval="${3:-5}"
    local output_format="${4:-terminal}"
    
    if [[ "$output_format" == "json" ]]; then
        agents::dashboard::json_output "$registry_file" "$resource_name"
        return $?
    fi
    
    # Terminal dashboard with refresh loop
    local quit=false
    trap 'quit=true' INT TERM
    
    while [[ "$quit" == "false" ]]; do
        clear
        agents::dashboard::render "$registry_file" "$resource_name"
        
        # Show footer with controls
        echo ""
        echo "═══════════════════════════════════════════════════════════════════════"
        echo "Refresh: ${refresh_interval}s | Press Ctrl+C to quit"
        echo "═══════════════════════════════════════════════════════════════════════"
        
        # Wait for refresh interval or quit signal
        local count=0
        while [[ $count -lt $refresh_interval && "$quit" == "false" ]]; do
            sleep 1
            ((count++))
        done
    done
    
    clear
    return 0
}

#######################################
# Render dashboard content
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
# Returns:
#   0 on success, 1 on error
#######################################
agents::dashboard::render() {
    local registry_file="$1"
    local resource_name="$2"
    
    if [[ ! -f "$registry_file" ]]; then
        echo "No agents registry found for $resource_name"
        return 1
    fi
    
    # Header
    echo "═══════════════════════════════════════════════════════════════════════"
    printf "                    %s AGENTS MONITOR\n" "$(echo "$resource_name" | tr '[:lower:]' '[:upper:]')"
    echo "═══════════════════════════════════════════════════════════════════════"
    
    # Timestamp and summary
    local current_time
    current_time=$(date '+%Y-%m-%d %H:%M:%S')
    local total_agents
    total_agents=$(jq -r '.agents | length' "$registry_file" 2>/dev/null || echo "0")
    local healthy_agents
    healthy_agents=$(jq -r '.agents | to_entries | map(select(.value.health == "healthy")) | length' "$registry_file" 2>/dev/null || echo "0")
    
    echo "Time: $current_time | Total Agents: $total_agents | Healthy: $healthy_agents"
    echo ""
    
    # Agent table header
    printf "%-36s %-10s %-10s %6s %7s %10s\n" \
           "AGENT ID" "STATUS" "HEALTH" "CPU" "MEM(MB)" "UPTIME"
    echo "───────────────────────────────────────────────────────────────────────"
    
    # Agent rows
    local agents_data
    agents_data=$(jq -r '.agents | to_entries | .[] | 
        [
            .key,
            .value.status // "unknown",
            .value.health // "unknown",
            (.value.metrics.gauges.cpu_ticks // 0),
            (.value.metrics.gauges.memory_mb // 0),
            .value.start_time // ""
        ] | @tsv' "$registry_file" 2>/dev/null || echo "")
    
    if [[ -n "$agents_data" ]]; then
        while IFS=$'\t' read -r agent_id status health cpu_ticks memory_mb start_time; do
            # Calculate uptime
            local uptime="N/A"
            if [[ -n "$start_time" && "$start_time" != "null" ]]; then
                local start_epoch
                start_epoch=$(date -d "$start_time" +%s 2>/dev/null || echo "0")
                local now_epoch
                now_epoch=$(date +%s)
                local uptime_seconds=$((now_epoch - start_epoch))
                uptime=$(agents::dashboard::format_duration "$uptime_seconds")
            fi
            
            # Format CPU (convert ticks to rough percentage)
            local cpu_display="N/A"
            if [[ "$cpu_ticks" != "0" ]]; then
                # Very rough estimate - ticks/100 for display
                cpu_display="~$(( cpu_ticks / 100 ))%"
            fi
            
            # Color code health status
            local health_display="$health"
            case "$health" in
                healthy)   health_display="${health}" ;;
                unhealthy) health_display="${health}" ;;
                degraded)  health_display="${health}" ;;
                *)         health_display="unknown" ;;
            esac
            
            # Truncate agent ID if too long
            local display_id="$agent_id"
            if [[ ${#agent_id} -gt 36 ]]; then
                display_id="${agent_id:0:33}..."
            fi
            
            printf "%-36s %-10s %-10s %6s %7s %10s\n" \
                   "$display_id" "$status" "$health_display" "$cpu_display" "$memory_mb" "$uptime"
        done <<< "$agents_data"
    else
        echo "No agents running"
    fi
    
    echo ""
    
    # Aggregate metrics section
    agents::dashboard::show_aggregate_metrics "$registry_file" "$resource_name"
    
    # Recent events section (simplified - just show last health check times)
    agents::dashboard::show_recent_events "$registry_file" "$resource_name"
}

#######################################
# Show aggregate metrics
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
# Returns:
#   0 on success
#######################################
agents::dashboard::show_aggregate_metrics() {
    local registry_file="$1"
    local resource_name="$2"
    
    echo "AGGREGATE METRICS"
    echo "───────────────────────────────────────────────────────────────────────"
    
    # Calculate totals
    local total_requests
    total_requests=$(jq -r '[.agents[].metrics.counters.requests // 0] | add' "$registry_file" 2>/dev/null || echo "0")
    local total_errors
    total_errors=$(jq -r '[.agents[].metrics.counters.errors // 0] | add' "$registry_file" 2>/dev/null || echo "0")
    local total_memory
    total_memory=$(jq -r '[.agents[].metrics.gauges.memory_mb // 0] | add' "$registry_file" 2>/dev/null || echo "0")
    
    # Calculate error rate
    local error_rate="0.0"
    if [[ "$total_requests" -gt 0 ]]; then
        error_rate=$(awk "BEGIN {printf \"%.2f\", ($total_errors / $total_requests) * 100}")
    fi
    
    echo "Requests: $total_requests | Errors: $total_errors (${error_rate}%) | Total Memory: ${total_memory}MB"
    echo ""
}

#######################################
# Show recent events
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
# Returns:
#   0 on success
#######################################
agents::dashboard::show_recent_events() {
    local registry_file="$1"
    local resource_name="$2"
    
    echo "RECENT ACTIVITY"
    echo "───────────────────────────────────────────────────────────────────────"
    
    # Show last 5 agents by last_seen time
    local recent_activity
    recent_activity=$(jq -r '.agents | to_entries | 
        map(select(.value.last_seen != null)) |
        sort_by(.value.last_seen) | reverse | .[0:5] |
        .[] | 
        [
            (.value.last_seen // ""),
            .key,
            (.value.health // "unknown")
        ] | @tsv' "$registry_file" 2>/dev/null || echo "")
    
    if [[ -n "$recent_activity" ]]; then
        while IFS=$'\t' read -r last_seen agent_id health; do
            # Format timestamp
            local display_time="N/A"
            if [[ -n "$last_seen" && "$last_seen" != "null" ]]; then
                display_time=$(date -d "$last_seen" '+%H:%M:%S' 2>/dev/null || echo "$last_seen")
            fi
            
            # Truncate agent ID
            local display_id="$agent_id"
            if [[ ${#agent_id} -gt 30 ]]; then
                display_id="${agent_id:0:27}..."
            fi
            
            printf "[%s] %-30s health: %s\n" "$display_time" "$display_id" "$health"
        done <<< "$recent_activity"
    else
        echo "No recent activity"
    fi
}

#######################################
# Format duration in human-readable form
# Arguments:
#   $1 - Duration in seconds
# Returns:
#   Formatted string (e.g., "2h 15m")
#######################################
agents::dashboard::format_duration() {
    local seconds="$1"
    
    if [[ "$seconds" -lt 60 ]]; then
        echo "${seconds}s"
    elif [[ "$seconds" -lt 3600 ]]; then
        local minutes=$((seconds / 60))
        echo "${minutes}m"
    elif [[ "$seconds" -lt 86400 ]]; then
        local hours=$((seconds / 3600))
        local minutes=$(((seconds % 3600) / 60))
        echo "${hours}h ${minutes}m"
    else
        local days=$((seconds / 86400))
        local hours=$(((seconds % 86400) / 3600))
        echo "${days}d ${hours}h"
    fi
}

#######################################
# Output dashboard data as JSON
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
# Returns:
#   0 on success, 1 on error
# Outputs:
#   JSON data to stdout
#######################################
agents::dashboard::json_output() {
    local registry_file="$1"
    local resource_name="$2"
    
    if [[ ! -f "$registry_file" ]]; then
        echo '{"error": "No registry found"}'
        return 1
    fi
    
    # Build comprehensive JSON output
    local current_time
    current_time=$(date -Iseconds)
    
    jq --arg resource "$resource_name" \
       --arg time "$current_time" \
       '{
           resource: $resource,
           timestamp: $time,
           summary: {
               total_agents: (.agents | length),
               healthy: (.agents | to_entries | map(select(.value.health == "healthy")) | length),
               unhealthy: (.agents | to_entries | map(select(.value.health == "unhealthy")) | length),
               degraded: (.agents | to_entries | map(select(.value.health == "degraded")) | length)
           },
           agents: .agents,
           aggregate_metrics: {
               total_requests: ([.agents[].metrics.counters.requests // 0] | add),
               total_errors: ([.agents[].metrics.counters.errors // 0] | add),
               total_memory_mb: ([.agents[].metrics.gauges.memory_mb // 0] | add),
               error_rate: (
                   if ([.agents[].metrics.counters.requests // 0] | add) > 0 then
                       (([.agents[].metrics.counters.errors // 0] | add) / ([.agents[].metrics.counters.requests // 0] | add))
                   else 0 end
               )
           }
       }' "$registry_file"
    
    return 0
}

# Export functions for use by agent manager
export -f agents::dashboard::display
export -f agents::dashboard::render
export -f agents::dashboard::show_aggregate_metrics
export -f agents::dashboard::show_recent_events
export -f agents::dashboard::format_duration
export -f agents::dashboard::json_output