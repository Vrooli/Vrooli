#!/usr/bin/env bash
# n8n Memory Monitoring and Alerting System
# Provides predictive memory monitoring with configurable thresholds

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_MEMORY_MONITOR_SOURCED:-}" ]] && return 0
export _N8N_MEMORY_MONITOR_SOURCED=1

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_LIB_DIR="${APP_ROOT}/resources/n8n/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"

# Memory monitoring configuration
readonly N8N_MEM_WARNING_THRESHOLD="${N8N_MEM_WARNING_THRESHOLD:-70}"
readonly N8N_MEM_CRITICAL_THRESHOLD="${N8N_MEM_CRITICAL_THRESHOLD:-80}"
readonly N8N_MEM_EMERGENCY_THRESHOLD="${N8N_MEM_EMERGENCY_THRESHOLD:-90}"
readonly N8N_MEM_CHECK_INTERVAL="${N8N_MEM_CHECK_INTERVAL:-30}"
readonly N8N_MEM_HISTORY_SIZE="${N8N_MEM_HISTORY_SIZE:-10}"
readonly N8N_MEM_PREDICTION_ENABLED="${N8N_MEM_PREDICTION_ENABLED:-true}"

# Memory history for trend analysis
declare -ga N8N_MEM_HISTORY=()
declare -g N8N_MEM_LAST_ALERT_TIME=0
declare -g N8N_MEM_ALERT_COOLDOWN=300  # 5 minutes between alerts

#######################################
# Get current memory usage for n8n container
# Returns: Memory usage percentage (0-100)
#######################################
n8n::memory::get_usage() {
    local container="${1:-$N8N_CONTAINER_NAME}"
    
    if ! docker::is_running "$container"; then
        echo "0"
        return 1
    fi
    
    # Get memory stats from docker
    local mem_stats
    mem_stats=$(docker stats "$container" --no-stream --format "{{.MemPerc}}" 2>/dev/null | sed 's/%//')
    
    if [[ -z "$mem_stats" ]]; then
        echo "0"
        return 1
    fi
    
    echo "$mem_stats"
}

#######################################
# Get detailed memory statistics
# Returns: JSON object with memory details
#######################################
n8n::memory::get_detailed_stats() {
    local container="${1:-$N8N_CONTAINER_NAME}"
    
    if ! docker::is_running "$container"; then
        echo '{"error": "Container not running"}'
        return 1
    fi
    
    # Get detailed stats from docker
    local stats
    stats=$(docker stats "$container" --no-stream --format '{"cpu":"{{.CPUPerc}}","memory":"{{.MemUsage}}","percent":"{{.MemPerc}}"}' 2>/dev/null)
    
    if [[ -z "$stats" ]]; then
        echo '{"error": "Failed to get stats"}'
        return 1
    fi
    
    # Parse memory usage
    local mem_usage mem_limit mem_percent
    mem_usage=$(echo "$stats" | jq -r '.memory' | cut -d'/' -f1 | sed 's/[^0-9.]//g')
    mem_limit=$(echo "$stats" | jq -r '.memory' | cut -d'/' -f2 | sed 's/[^0-9.]//g')
    mem_percent=$(echo "$stats" | jq -r '.percent' | sed 's/%//')
    
    # Convert to MB if needed
    if echo "$stats" | grep -q "GiB"; then
        mem_usage=$(echo "$mem_usage * 1024" | bc)
        mem_limit=$(echo "$mem_limit * 1024" | bc)
    fi
    
    # Build response
    jq -n \
        --arg usage "$mem_usage" \
        --arg limit "$mem_limit" \
        --arg percent "$mem_percent" \
        --arg threshold_warning "$N8N_MEM_WARNING_THRESHOLD" \
        --arg threshold_critical "$N8N_MEM_CRITICAL_THRESHOLD" \
        --arg threshold_emergency "$N8N_MEM_EMERGENCY_THRESHOLD" \
        '{
            usage_mb: ($usage | tonumber),
            limit_mb: ($limit | tonumber),
            percent: ($percent | tonumber),
            thresholds: {
                warning: ($threshold_warning | tonumber),
                critical: ($threshold_critical | tonumber),
                emergency: ($threshold_emergency | tonumber)
            }
        }'
}

#######################################
# Update memory history for trend analysis
# Args: $1 - Current memory percentage
#######################################
n8n::memory::update_history() {
    local current_mem="$1"
    
    # Add to history
    N8N_MEM_HISTORY+=("$current_mem")
    
    # Keep only last N entries
    if [[ ${#N8N_MEM_HISTORY[@]} -gt $N8N_MEM_HISTORY_SIZE ]]; then
        N8N_MEM_HISTORY=("${N8N_MEM_HISTORY[@]:1}")
    fi
}

#######################################
# Predict memory exhaustion time
# Returns: Estimated minutes until OOM (or -1 if stable)
#######################################
n8n::memory::predict_exhaustion() {
    if [[ ${#N8N_MEM_HISTORY[@]} -lt 3 ]]; then
        echo "-1"
        return
    fi
    
    # Calculate average growth rate
    local total_growth=0
    local samples=0
    
    for ((i=1; i<${#N8N_MEM_HISTORY[@]}; i++)); do
        local prev="${N8N_MEM_HISTORY[$((i-1))]}"
        local curr="${N8N_MEM_HISTORY[$i]}"
        local growth=$(echo "$curr - $prev" | bc)
        total_growth=$(echo "$total_growth + $growth" | bc)
        ((samples++))
    done
    
    if [[ $samples -eq 0 ]]; then
        echo "-1"
        return
    fi
    
    local avg_growth=$(echo "scale=2; $total_growth / $samples" | bc)
    
    # If memory is decreasing or stable
    if (( $(echo "$avg_growth <= 0" | bc -l) )); then
        echo "-1"
        return
    fi
    
    # Calculate time to 100%
    local current="${N8N_MEM_HISTORY[-1]}"
    local remaining=$(echo "100 - $current" | bc)
    local intervals=$(echo "scale=0; $remaining / $avg_growth" | bc)
    local minutes=$(echo "$intervals * $N8N_MEM_CHECK_INTERVAL / 60" | bc)
    
    echo "$minutes"
}

#######################################
# Generate alert message based on severity
# Args: $1 - severity level, $2 - current memory %, $3 - prediction
#######################################
n8n::memory::generate_alert() {
    local severity="$1"
    local current_mem="$2"
    local prediction="${3:--1}"
    
    local message=""
    local emoji=""
    
    case "$severity" in
        warning)
            emoji="⚠️"
            message="Memory usage at ${current_mem}% (threshold: ${N8N_MEM_WARNING_THRESHOLD}%)"
            ;;
        critical)
            emoji="🔴"
            message="CRITICAL: Memory usage at ${current_mem}% (threshold: ${N8N_MEM_CRITICAL_THRESHOLD}%)"
            ;;
        emergency)
            emoji="🚨"
            message="EMERGENCY: Memory usage at ${current_mem}% - Container may crash soon!"
            ;;
    esac
    
    # Add prediction if available
    if [[ "$prediction" != "-1" ]] && [[ -n "$prediction" ]]; then
        message="$message - Estimated OOM in ${prediction} minutes"
    fi
    
    echo "$emoji n8n Memory Alert: $message"
}

#######################################
# Check memory and trigger alerts
# Returns: 0 if ok, 1 if warning, 2 if critical, 3 if emergency
#######################################
n8n::memory::check_and_alert() {
    local current_mem
    current_mem=$(n8n::memory::get_usage)
    
    if [[ -z "$current_mem" ]] || [[ "$current_mem" == "0" ]]; then
        return 0
    fi
    
    # Update history
    n8n::memory::update_history "$current_mem"
    
    # Get prediction if enabled
    local prediction=-1
    if [[ "$N8N_MEM_PREDICTION_ENABLED" == "true" ]]; then
        prediction=$(n8n::memory::predict_exhaustion)
    fi
    
    # Check thresholds
    local severity=""
    local return_code=0
    
    if (( $(echo "$current_mem >= $N8N_MEM_EMERGENCY_THRESHOLD" | bc -l) )); then
        severity="emergency"
        return_code=3
    elif (( $(echo "$current_mem >= $N8N_MEM_CRITICAL_THRESHOLD" | bc -l) )); then
        severity="critical"
        return_code=2
    elif (( $(echo "$current_mem >= $N8N_MEM_WARNING_THRESHOLD" | bc -l) )); then
        severity="warning"
        return_code=1
    else
        return 0
    fi
    
    # Check alert cooldown
    local current_time
    current_time=$(date +%s)
    local time_since_last=$((current_time - N8N_MEM_LAST_ALERT_TIME))
    
    if [[ $time_since_last -lt $N8N_MEM_ALERT_COOLDOWN ]] && [[ "$severity" != "emergency" ]]; then
        return $return_code
    fi
    
    # Generate and log alert
    local alert_message
    alert_message=$(n8n::memory::generate_alert "$severity" "$current_mem" "$prediction")
    
    case "$severity" in
        warning)
            log::warn "$alert_message"
            ;;
        critical)
            log::error "$alert_message"
            ;;
        emergency)
            log::error "$alert_message"
            log::error "Consider restarting n8n or increasing memory allocation immediately!"
            ;;
    esac
    
    # Update last alert time
    N8N_MEM_LAST_ALERT_TIME=$current_time
    
    # Write to monitoring file for external tools
    if [[ -n "${N8N_MONITORING_FILE:-}" ]]; then
        echo "$(date -Iseconds) | $severity | $current_mem% | $alert_message" >> "$N8N_MONITORING_FILE"
    fi
    
    return $return_code
}

#######################################
# Start continuous memory monitoring
# Args: $1 - Run as daemon (true/false)
#######################################
n8n::memory::monitor_loop() {
    local daemon="${1:-false}"
    
    log::info "Starting n8n memory monitoring (interval: ${N8N_MEM_CHECK_INTERVAL}s)"
    log::info "Thresholds - Warning: ${N8N_MEM_WARNING_THRESHOLD}%, Critical: ${N8N_MEM_CRITICAL_THRESHOLD}%, Emergency: ${N8N_MEM_EMERGENCY_THRESHOLD}%"
    
    if [[ "$daemon" == "true" ]]; then
        # Run in background
        (
            while true; do
                n8n::memory::check_and_alert
                sleep "$N8N_MEM_CHECK_INTERVAL"
            done
        ) &
        local pid=$!
        echo "$pid" > /tmp/n8n-memory-monitor.pid
        log::info "Memory monitor started in background (PID: $pid)"
    else
        # Run in foreground
        while true; do
            n8n::memory::check_and_alert
            local status=$?
            
            # Exit on emergency
            if [[ $status -eq 3 ]]; then
                log::error "Emergency memory threshold reached. Consider immediate action!"
            fi
            
            sleep "$N8N_MEM_CHECK_INTERVAL"
        done
    fi
}

#######################################
# Stop memory monitoring daemon
#######################################
n8n::memory::stop_monitor() {
    if [[ -f /tmp/n8n-memory-monitor.pid ]]; then
        local pid
        pid=$(cat /tmp/n8n-memory-monitor.pid)
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            rm -f /tmp/n8n-memory-monitor.pid
            log::info "Memory monitor stopped (PID: $pid)"
        fi
    fi
}

#######################################
# Get memory monitoring status
# Returns: JSON status object
#######################################
n8n::memory::get_monitor_status() {
    local running="false"
    local pid=""
    
    if [[ -f /tmp/n8n-memory-monitor.pid ]]; then
        pid=$(cat /tmp/n8n-memory-monitor.pid)
        if kill -0 "$pid" 2>/dev/null; then
            running="true"
        fi
    fi
    
    local current_mem
    current_mem=$(n8n::memory::get_usage)
    
    jq -n \
        --arg running "$running" \
        --arg pid "$pid" \
        --arg current "$current_mem" \
        --argjson history "$(printf '%s\n' "${N8N_MEM_HISTORY[@]}" | jq -R . | jq -s .)" \
        '{
            monitor_running: ($running == "true"),
            monitor_pid: $pid,
            current_usage: ($current | tonumber),
            history: $history
        }'
}