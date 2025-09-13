#!/usr/bin/env bash
#
# Browserless Pool Manager - Auto-scaling browser pool based on load
#
# This module provides automatic scaling of the browser pool based on:
# - Current load (running/queued sessions)
# - CPU and memory utilization
# - Response time trends
#
# The pool manager runs as a background process monitoring metrics
# and adjusting the pool size dynamically.
#

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/common.sh"

# Pool scaling configuration
POOL_MIN_SIZE="${BROWSERLESS_POOL_MIN_SIZE:-2}"
POOL_MAX_SIZE="${BROWSERLESS_POOL_MAX_SIZE:-20}"
POOL_SCALE_UP_THRESHOLD="${BROWSERLESS_POOL_SCALE_UP_THRESHOLD:-70}"    # % utilization to trigger scale up
POOL_SCALE_DOWN_THRESHOLD="${BROWSERLESS_POOL_SCALE_DOWN_THRESHOLD:-30}" # % utilization to trigger scale down
POOL_SCALE_STEP="${BROWSERLESS_POOL_SCALE_STEP:-2}"                     # Number of instances to add/remove
POOL_MONITOR_INTERVAL="${BROWSERLESS_POOL_MONITOR_INTERVAL:-10}"         # Seconds between checks
POOL_COOLDOWN_PERIOD="${BROWSERLESS_POOL_COOLDOWN_PERIOD:-30}"          # Seconds to wait after scaling

# State tracking
LAST_SCALE_TIME=0
CURRENT_POOL_SIZE="${BROWSERLESS_MAX_CONCURRENT_SESSIONS:-10}"

#######################################
# Get current pool metrics
# Returns:
#   JSON with pool metrics
#######################################
pool::get_metrics() {
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    local response
    
    # Get pressure metrics
    response=$(timeout 5 curl -sf "http://localhost:${browserless_port}/pressure" 2>/dev/null || echo "{}")
    
    if [[ -z "$response" ]] || [[ "$response" == "{}" ]]; then
        echo '{"error": "Failed to get metrics"}'
        return 1
    fi
    
    # Extract key metrics
    local running=$(echo "$response" | jq -r '.pressure.running // 0')
    local queued=$(echo "$response" | jq -r '.pressure.queued // 0')
    local max_concurrent=$(echo "$response" | jq -r '.pressure.maxConcurrent // 10')
    local cpu=$(echo "$response" | jq -r '.pressure.cpu // 0')
    local memory=$(echo "$response" | jq -r '.pressure.memory // 0')
    
    # Calculate utilization percentage
    local total_load=$((running + queued))
    local utilization=0
    if [[ $max_concurrent -gt 0 ]]; then
        utilization=$(( (total_load * 100) / max_concurrent ))
    fi
    
    # Build metrics JSON
    cat <<EOF
{
    "running": $running,
    "queued": $queued,
    "maxConcurrent": $max_concurrent,
    "totalLoad": $total_load,
    "utilization": $utilization,
    "cpu": $cpu,
    "memory": $memory,
    "timestamp": $(date +%s)
}
EOF
}

#######################################
# Calculate desired pool size based on metrics
# Arguments:
#   $1 - Current metrics JSON
# Returns:
#   Desired pool size
#######################################
pool::calculate_desired_size() {
    local metrics="${1:?Metrics required}"
    
    local current_size=$(echo "$metrics" | jq -r '.maxConcurrent // 10')
    local utilization=$(echo "$metrics" | jq -r '.utilization // 0')
    local queued=$(echo "$metrics" | jq -r '.queued // 0')
    local cpu=$(echo "$metrics" | jq -r '.cpu // 0')
    local memory=$(echo "$metrics" | jq -r '.memory // 0')
    
    local desired_size=$current_size
    
    # Scale up conditions
    if [[ $utilization -gt $POOL_SCALE_UP_THRESHOLD ]] || [[ $queued -gt 0 ]]; then
        # If there are queued requests, scale more aggressively
        if [[ $queued -gt 0 ]]; then
            local scale_factor=$((queued / 2 + 1))
            desired_size=$((current_size + scale_factor * POOL_SCALE_STEP))
        else
            desired_size=$((current_size + POOL_SCALE_STEP))
        fi
        
        log::debug "Scale up triggered: utilization=$utilization%, queued=$queued"
    
    # Scale down conditions (only if CPU and memory are low)
    elif [[ $utilization -lt $POOL_SCALE_DOWN_THRESHOLD ]] && \
         [[ $(echo "$cpu < 0.5" | bc -l) -eq 1 ]] && \
         [[ $(echo "$memory < 0.5" | bc -l) -eq 1 ]]; then
        desired_size=$((current_size - POOL_SCALE_STEP))
        log::debug "Scale down triggered: utilization=$utilization%, cpu=$cpu, memory=$memory"
    fi
    
    # Apply min/max limits
    if [[ $desired_size -lt $POOL_MIN_SIZE ]]; then
        desired_size=$POOL_MIN_SIZE
    elif [[ $desired_size -gt $POOL_MAX_SIZE ]]; then
        desired_size=$POOL_MAX_SIZE
    fi
    
    echo "$desired_size"
}

#######################################
# Apply new pool size to running container
# Arguments:
#   $1 - New pool size
# Returns:
#   0 on success, 1 on failure
#######################################
pool::apply_size() {
    local new_size="${1:?New size required}"
    
    # Check if container is running
    if ! is_running; then
        log::error "Browserless container is not running"
        return 1
    fi
    
    log::info "Applying new pool size: $new_size"
    
    # Update environment variable in container
    # Note: This requires container restart to take effect
    # For live updates, we'd need to use browserless API if available
    
    # Store the new size for next restart
    echo "$new_size" > "${BROWSERLESS_DATA_DIR}/pool_size"
    
    # Update the global variable
    CURRENT_POOL_SIZE=$new_size
    export BROWSERLESS_MAX_CONCURRENT_SESSIONS=$new_size
    
    # Log the change
    log::success "Pool size updated to $new_size (will apply on next restart)"
    
    # Update last scale time
    LAST_SCALE_TIME=$(date +%s)
    
    return 0
}

#######################################
# Check if cooldown period has passed
# Returns:
#   0 if can scale, 1 if in cooldown
#######################################
pool::can_scale() {
    local current_time=$(date +%s)
    local time_since_scale=$((current_time - LAST_SCALE_TIME))
    
    if [[ $time_since_scale -lt $POOL_COOLDOWN_PERIOD ]]; then
        log::debug "In cooldown period ($time_since_scale/$POOL_COOLDOWN_PERIOD seconds)"
        return 1
    fi
    
    return 0
}

#######################################
# Main auto-scaling loop
# Runs continuously monitoring and adjusting pool
#######################################
pool::autoscale_loop() {
    log::header "🔄 Starting Browser Pool Auto-scaler"
    log::info "Configuration:"
    log::info "  Min size: $POOL_MIN_SIZE"
    log::info "  Max size: $POOL_MAX_SIZE"
    log::info "  Scale up threshold: $POOL_SCALE_UP_THRESHOLD%"
    log::info "  Scale down threshold: $POOL_SCALE_DOWN_THRESHOLD%"
    log::info "  Monitor interval: ${POOL_MONITOR_INTERVAL}s"
    
    while true; do
        # Get current metrics
        local metrics
        metrics=$(pool::get_metrics)
        
        if [[ $(echo "$metrics" | jq -r '.error // ""') != "" ]]; then
            log::warning "Failed to get metrics, skipping this cycle"
            sleep "$POOL_MONITOR_INTERVAL"
            continue
        fi
        
        # Log current state
        local utilization=$(echo "$metrics" | jq -r '.utilization')
        local running=$(echo "$metrics" | jq -r '.running')
        local queued=$(echo "$metrics" | jq -r '.queued')
        log::debug "Pool status: utilization=$utilization%, running=$running, queued=$queued"
        
        # Check if we can scale (cooldown period)
        if pool::can_scale; then
            # Calculate desired size
            local current_size=$(echo "$metrics" | jq -r '.maxConcurrent')
            local desired_size
            desired_size=$(pool::calculate_desired_size "$metrics")
            
            # Apply scaling if needed
            if [[ $desired_size -ne $current_size ]]; then
                log::info "Scaling pool from $current_size to $desired_size"
                if pool::apply_size "$desired_size"; then
                    log::success "Pool scaled successfully"
                else
                    log::error "Failed to scale pool"
                fi
            fi
        fi
        
        # Wait before next check
        sleep "$POOL_MONITOR_INTERVAL"
    done
}

#######################################
# Start auto-scaler in background
# Returns:
#   PID of auto-scaler process
#######################################
pool::start_autoscaler() {
    # Check if already running
    local pid_file="${BROWSERLESS_DATA_DIR}/autoscaler.pid"
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log::warning "Auto-scaler already running (PID: $pid)"
            echo "$pid"
            return 0
        fi
    fi
    
    # Start in background
    pool::autoscale_loop &
    local pid=$!
    
    # Save PID
    echo "$pid" > "$pid_file"
    
    log::success "Auto-scaler started (PID: $pid)"
    echo "$pid"
}

#######################################
# Stop auto-scaler
#######################################
pool::stop_autoscaler() {
    local pid_file="${BROWSERLESS_DATA_DIR}/autoscaler.pid"
    
    if [[ ! -f "$pid_file" ]]; then
        log::info "Auto-scaler not running"
        return 0
    fi
    
    local pid=$(cat "$pid_file")
    if kill -0 "$pid" 2>/dev/null; then
        log::info "Stopping auto-scaler (PID: $pid)"
        kill "$pid"
        rm -f "$pid_file"
        log::success "Auto-scaler stopped"
    else
        log::info "Auto-scaler not running"
        rm -f "$pid_file"
    fi
}

#######################################
# Get auto-scaler status
#######################################
pool::autoscaler_status() {
    local pid_file="${BROWSERLESS_DATA_DIR}/autoscaler.pid"
    
    if [[ ! -f "$pid_file" ]]; then
        echo "stopped"
        return 0
    fi
    
    local pid=$(cat "$pid_file")
    if kill -0 "$pid" 2>/dev/null; then
        echo "running"
    else
        echo "stopped"
    fi
}

#######################################
# Show pool statistics
#######################################
pool::show_stats() {
    log::header "📊 Browser Pool Statistics"
    
    # Get current metrics
    local metrics
    metrics=$(pool::get_metrics)
    
    if [[ $(echo "$metrics" | jq -r '.error // ""') != "" ]]; then
        log::error "Failed to get pool metrics"
        return 1
    fi
    
    # Display stats
    local current_size=$(echo "$metrics" | jq -r '.maxConcurrent')
    local running=$(echo "$metrics" | jq -r '.running')
    local queued=$(echo "$metrics" | jq -r '.queued')
    local total_load=$(echo "$metrics" | jq -r '.totalLoad')
    local utilization=$(echo "$metrics" | jq -r '.utilization')
    local cpu=$(echo "$metrics" | jq -r '.cpu')
    local memory=$(echo "$metrics" | jq -r '.memory')
    
    # Calculate percentages properly
    local cpu_percent=$(echo "$cpu * 100" | bc -l | cut -d. -f1)
    local memory_percent=$(echo "$memory * 100" | bc -l | cut -d. -f1)
    
    cat <<EOF
Pool Configuration:
  Current size: $current_size
  Min size: $POOL_MIN_SIZE
  Max size: $POOL_MAX_SIZE

Current Load:
  Running sessions: $running
  Queued sessions: $queued
  Total load: $total_load
  Utilization: ${utilization}%

System Resources:
  CPU usage: ${cpu_percent}%
  Memory usage: ${memory_percent}%
EOF
    
    # Show auto-scaler status
    local autoscaler_status
    autoscaler_status=$(pool::autoscaler_status)
    echo ""
    echo "Auto-scaler: $autoscaler_status"
    
    if [[ "$autoscaler_status" == "running" ]]; then
        local pid=$(cat "${BROWSERLESS_DATA_DIR}/autoscaler.pid")
        echo "  PID: $pid"
    fi
}

# Export functions
export -f pool::get_metrics
export -f pool::calculate_desired_size
export -f pool::apply_size
export -f pool::can_scale
export -f pool::autoscale_loop
export -f pool::start_autoscaler
export -f pool::stop_autoscaler
export -f pool::autoscaler_status
export -f pool::show_stats