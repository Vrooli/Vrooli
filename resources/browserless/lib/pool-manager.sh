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
    # Use awk instead of bc for better portability
    elif [[ $utilization -lt $POOL_SCALE_DOWN_THRESHOLD ]] && \
         [[ $(awk -v cpu="$cpu" 'BEGIN {if (cpu < 0.5) print 1; else print 0}') -eq 1 ]] && \
         [[ $(awk -v mem="$memory" 'BEGIN {if (mem < 0.5) print 1; else print 0}') -eq 1 ]]; then
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
        # Check pool health and recover if needed
        if ! pool::health_check_and_recover; then
            log::error "Pool recovery failed - waiting before retry"
            sleep "$POOL_MONITOR_INTERVAL"
            continue
        fi
        
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
        
        # Check if pre-warming is needed (when system is idle)
        pool::smart_prewarm
        
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
# Pre-warm browser instances to reduce cold start latency
# Arguments:
#   $1 - Number of instances to pre-warm (optional, defaults to min pool size)
# Returns:
#   0 on success, 1 on failure
#######################################
pool::prewarm() {
    local instances="${1:-$POOL_MIN_SIZE}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::header "🔥 Pre-warming Browser Pool"
    log::info "Pre-warming $instances browser instances..."
    
    # Check if browserless is running
    if ! is_running; then
        log::error "Browserless container is not running"
        return 1
    fi
    
    # Track successful pre-warm attempts
    local success_count=0
    local failed_count=0
    
    # Pre-warm instances by triggering lightweight operations
    for ((i=1; i<=instances; i++)); do
        log::debug "Pre-warming instance $i/$instances..."
        
        # Use a simple JavaScript evaluation to trigger browser creation
        # This is lightweight but ensures a browser instance is created
        local prewarm_js='JSON.stringify({status: "ready", timestamp: Date.now()})'
        
        # Execute the pre-warm request in background to parallelize
        (
            timeout 10 curl -sf \
                -X POST "http://localhost:${browserless_port}/function" \
                -H "Content-Type: application/json" \
                -d "{
                    \"code\": \"async () => { return ${prewarm_js}; }\"
                }" &>/dev/null && echo "success" || echo "failed"
        ) &
        
        # Add small delay to avoid overwhelming the service
        sleep 0.2
    done
    
    # Wait for all background jobs to complete
    wait
    
    # Check pressure metrics to verify pre-warming effect
    sleep 2
    local metrics
    metrics=$(pool::get_metrics)
    
    if [[ $(echo "$metrics" | jq -r '.error // ""') == "" ]]; then
        local max_concurrent=$(echo "$metrics" | jq -r '.maxConcurrent // 0')
        log::success "Pre-warming complete. Pool capacity: $max_concurrent"
        
        # Store pre-warm timestamp for tracking
        echo "$(date +%s)" > "${BROWSERLESS_DATA_DIR}/last_prewarm"
        return 0
    else
        log::warning "Pre-warming completed but could not verify pool status"
        return 1
    fi
}

#######################################
# Get time since last pre-warm
# Returns:
#   Seconds since last pre-warm, or -1 if never pre-warmed
#######################################
pool::time_since_prewarm() {
    local prewarm_file="${BROWSERLESS_DATA_DIR}/last_prewarm"
    
    if [[ ! -f "$prewarm_file" ]]; then
        echo "-1"
        return 0
    fi
    
    local last_prewarm=$(cat "$prewarm_file")
    local current_time=$(date +%s)
    local elapsed=$((current_time - last_prewarm))
    
    echo "$elapsed"
}

#######################################
# Intelligent pre-warming based on system state
# Pre-warms only when:
#   - System is idle (low utilization)
#   - Haven't pre-warmed recently
#   - Browserless is running
#######################################
pool::smart_prewarm() {
    local prewarm_interval="${BROWSERLESS_PREWARM_INTERVAL:-300}"  # Default 5 minutes
    local idle_threshold="${BROWSERLESS_IDLE_THRESHOLD:-20}"       # % utilization to consider idle
    
    # Check if pre-warming is needed
    local time_since_last
    time_since_last=$(pool::time_since_prewarm)
    
    if [[ $time_since_last -ne -1 ]] && [[ $time_since_last -lt $prewarm_interval ]]; then
        log::debug "Skipping pre-warm (last pre-warm ${time_since_last}s ago)"
        return 0
    fi
    
    # Check current utilization
    local metrics
    metrics=$(pool::get_metrics)
    
    if [[ $(echo "$metrics" | jq -r '.error // ""') != "" ]]; then
        log::debug "Cannot pre-warm: metrics unavailable"
        return 1
    fi
    
    local utilization=$(echo "$metrics" | jq -r '.utilization // 100')
    
    if [[ $utilization -gt $idle_threshold ]]; then
        log::debug "Skipping pre-warm (utilization ${utilization}% > ${idle_threshold}%)"
        return 0
    fi
    
    # System is idle and pre-warm is due - proceed with pre-warming
    log::info "System idle (${utilization}% utilization) - initiating pre-warm"
    pool::prewarm "$POOL_MIN_SIZE"
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
    # Use awk for portable floating point math
    local cpu_percent=$(awk -v cpu="$cpu" 'BEGIN {printf "%.0f", cpu * 100}')
    local memory_percent=$(awk -v mem="$memory" 'BEGIN {printf "%.0f", mem * 100}')
    
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

#######################################
# Check browser pool health and recover if needed
# Detects and recovers from browser crashes
# Returns:
#   0 if healthy or recovered, 1 if recovery failed
#######################################
pool::health_check_and_recover() {
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    local max_recovery_attempts=3
    local attempt=0
    
    # First check if container is running
    if ! is_running; then
        log::error "Browserless container is not running - cannot recover pool"
        return 1
    fi
    
    # Check health endpoint
    local health_response
    health_response=$(timeout 5 curl -sf "http://localhost:${browserless_port}/pressure" 2>/dev/null || echo "")
    
    if [[ -z "$health_response" ]]; then
        log::warning "Browser pool not responding - attempting recovery"
        
        while [[ $attempt -lt $max_recovery_attempts ]]; do
            attempt=$((attempt + 1))
            log::info "Recovery attempt $attempt/$max_recovery_attempts"
            
            # Try to restart the container
            log::info "Restarting browserless container..."
            docker restart "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1
            
            # Wait for container to come up
            sleep 10
            
            # Check if it's healthy now
            health_response=$(timeout 5 curl -sf "http://localhost:${browserless_port}/pressure" 2>/dev/null || echo "")
            
            if [[ -n "$health_response" ]]; then
                log::success "Browser pool recovered successfully"
                
                # Pre-warm after recovery to ensure readiness
                pool::prewarm "$POOL_MIN_SIZE"
                return 0
            fi
            
            sleep 5
        done
        
        log::error "Failed to recover browser pool after $max_recovery_attempts attempts"
        return 1
    fi
    
    # Check for high error rate or rejected requests
    local recently_rejected=$(echo "$health_response" | jq -r '.pressure.recentlyRejected // 0')
    
    if [[ $recently_rejected -gt 5 ]]; then
        log::warning "High rejection rate detected ($recently_rejected) - clearing pool"
        
        # Clear any stuck sessions by restarting container
        docker restart "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1
        sleep 10
        
        # Pre-warm to restore capacity
        pool::prewarm "$POOL_MIN_SIZE"
        
        log::success "Pool cleared and restored"
    fi
    
    return 0
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
export -f pool::prewarm
export -f pool::time_since_prewarm
export -f pool::smart_prewarm
export -f pool::health_check_and_recover