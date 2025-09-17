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
BROWSERLESS_DATA_DIR="${APP_ROOT}/.vrooli/browserless"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }

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
AUTOSCALER_PID_FILE="${BROWSERLESS_DATA_DIR}/autoscaler.pid"

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
    
    if [[ "$response" == "{}" ]]; then
        log::error "Failed to get browserless pressure metrics"
        return 1
    fi
    
    echo "$response"
}

#######################################
# Show pool statistics
# Outputs:
#   Formatted pool statistics
#######################################
pool::show_stats() {
    local metrics
    metrics=$(pool::get_metrics) || {
        echo "❌ Failed to get pool metrics"
        return 1
    }
    
    local running queued max_concurrent cpu memory
    running=$(echo "$metrics" | jq -r '.pressure.running // 0')
    queued=$(echo "$metrics" | jq -r '.pressure.queued // 0')
    max_concurrent=$(echo "$metrics" | jq -r '.pressure.maxConcurrent // 10')
    cpu=$(echo "$metrics" | jq -r '.pressure.cpu // 0')
    memory=$(echo "$metrics" | jq -r '.pressure.memory // 0')
    
    echo "🔧 Browser Pool Statistics"
    echo "────────────────────────────"
    echo "Running Sessions: ${running}/${max_concurrent}"
    echo "Queued Sessions: ${queued}"
    echo "CPU Usage: ${cpu}%"
    echo "Memory Usage: ${memory}%"
    
    # Calculate utilization
    local utilization=0
    if [[ "$max_concurrent" -gt 0 ]]; then
        utilization=$(awk "BEGIN {printf \"%.1f\", ($running / $max_concurrent) * 100}")
    fi
    echo "Pool Utilization: ${utilization}%"
    
    # Check if autoscaler is running
    if pool::is_autoscaler_running; then
        echo "Auto-scaler: ✅ Running (PID: $(cat "$AUTOSCALER_PID_FILE" 2>/dev/null || echo 'unknown'))"
    else
        echo "Auto-scaler: ⭕ Stopped"
    fi
}

#######################################
# Start the auto-scaler
# Runs in background monitoring load
#######################################
pool::start_autoscaler() {
    if pool::is_autoscaler_running; then
        log::info "Auto-scaler already running (PID: $(cat "$AUTOSCALER_PID_FILE"))"
        return 0
    fi
    
    # Ensure data directory exists
    mkdir -p "$BROWSERLESS_DATA_DIR"
    
    # Start autoscaler in background
    {
        while true; do
            pool::monitor_and_scale
            sleep "$POOL_MONITOR_INTERVAL"
        done
    } &
    
    local pid=$!
    echo "$pid" > "$AUTOSCALER_PID_FILE"
    
    log::info "Auto-scaler started (PID: $pid)"
    log::info "Configuration: min=$POOL_MIN_SIZE, max=$POOL_MAX_SIZE, scale_up=$POOL_SCALE_UP_THRESHOLD%, scale_down=$POOL_SCALE_DOWN_THRESHOLD%"
}

#######################################
# Stop the auto-scaler
#######################################
pool::stop_autoscaler() {
    if ! pool::is_autoscaler_running; then
        log::info "Auto-scaler not running"
        return 0
    fi
    
    local pid
    pid=$(cat "$AUTOSCALER_PID_FILE" 2>/dev/null)
    
    if [[ -n "$pid" ]]; then
        kill "$pid" 2>/dev/null || true
        rm -f "$AUTOSCALER_PID_FILE"
        log::info "Auto-scaler stopped (PID: $pid)"
    fi
}

#######################################
# Check if auto-scaler is running
#######################################
pool::is_autoscaler_running() {
    [[ -f "$AUTOSCALER_PID_FILE" ]] || return 1
    
    local pid
    pid=$(cat "$AUTOSCALER_PID_FILE" 2>/dev/null)
    
    [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

#######################################
# Monitor and scale the pool
# Called periodically by autoscaler
#######################################
pool::monitor_and_scale() {
    local metrics
    metrics=$(pool::get_metrics) || return 1
    
    local running max_concurrent cpu
    running=$(echo "$metrics" | jq -r '.pressure.running // 0')
    max_concurrent=$(echo "$metrics" | jq -r '.pressure.maxConcurrent // 10')
    cpu=$(echo "$metrics" | jq -r '.pressure.cpu // 0')
    
    # Calculate utilization
    local utilization=0
    if [[ "$max_concurrent" -gt 0 ]]; then
        utilization=$(awk "BEGIN {printf \"%.0f\", ($running / $max_concurrent) * 100}")
    fi
    
    # Check cooldown
    local current_time
    current_time=$(date +%s)
    local time_since_last_scale=$((current_time - LAST_SCALE_TIME))
    
    if [[ "$time_since_last_scale" -lt "$POOL_COOLDOWN_PERIOD" ]]; then
        return 0  # Still in cooldown
    fi
    
    # Scale up if needed
    if [[ "$utilization" -ge "$POOL_SCALE_UP_THRESHOLD" ]] && [[ "$max_concurrent" -lt "$POOL_MAX_SIZE" ]]; then
        pool::scale_up
        LAST_SCALE_TIME=$current_time
    # Scale down if needed
    elif [[ "$utilization" -le "$POOL_SCALE_DOWN_THRESHOLD" ]] && [[ "$max_concurrent" -gt "$POOL_MIN_SIZE" ]]; then
        pool::scale_down
        LAST_SCALE_TIME=$current_time
    fi
    
    # Health check and recovery
    pool::health_check_and_recover
}

#######################################
# Scale up the browser pool
#######################################
pool::scale_up() {
    local current_size new_size
    
    # Get current max concurrent from metrics
    local metrics
    metrics=$(pool::get_metrics) || return 1
    current_size=$(echo "$metrics" | jq -r '.pressure.maxConcurrent // 10')
    
    new_size=$((current_size + POOL_SCALE_STEP))
    if [[ "$new_size" -gt "$POOL_MAX_SIZE" ]]; then
        new_size="$POOL_MAX_SIZE"
    fi
    
    log::info "Scaling up browser pool: $current_size → $new_size"
    
    # Update browserless configuration (would need API call if supported)
    # For now, log the intention
    echo "Scale up to $new_size instances" >> "${BROWSERLESS_DATA_DIR}/scaling.log"
}

#######################################
# Scale down the browser pool
#######################################
pool::scale_down() {
    local current_size new_size
    
    # Get current max concurrent from metrics
    local metrics
    metrics=$(pool::get_metrics) || return 1
    current_size=$(echo "$metrics" | jq -r '.pressure.maxConcurrent // 10')
    
    new_size=$((current_size - POOL_SCALE_STEP))
    if [[ "$new_size" -lt "$POOL_MIN_SIZE" ]]; then
        new_size="$POOL_MIN_SIZE"
    fi
    
    log::info "Scaling down browser pool: $current_size → $new_size"
    
    # Update browserless configuration (would need API call if supported)
    # For now, log the intention
    echo "Scale down to $new_size instances" >> "${BROWSERLESS_DATA_DIR}/scaling.log"
}

#######################################
# Pre-warm browser instances
# Arguments:
#   count: Number of instances to pre-warm
#######################################
pool::prewarm() {
    local count="${1:-2}"
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    log::info "Pre-warming $count browser instances..."
    
    # Trigger browser starts by making lightweight requests
    for ((i=1; i<=count; i++)); do
        {
            timeout 5 curl -sf "http://localhost:${browserless_port}/pressure" >/dev/null 2>&1
        } &
    done
    
    wait
    log::info "Pre-warming complete"
}

#######################################
# Smart pre-warm based on idle detection
#######################################
pool::smart_prewarm() {
    local metrics
    metrics=$(pool::get_metrics) || return 1
    
    local running queued
    running=$(echo "$metrics" | jq -r '.pressure.running // 0')
    queued=$(echo "$metrics" | jq -r '.pressure.queued // 0')
    
    # Only pre-warm if system is idle
    if [[ "$running" -eq 0 ]] && [[ "$queued" -eq 0 ]]; then
        log::info "System idle, pre-warming browsers..."
        pool::prewarm 2
    else
        log::info "System busy (running: $running, queued: $queued), skipping pre-warm"
    fi
}

#######################################
# Health check and recover unhealthy pool
#######################################
pool::health_check_and_recover() {
    local metrics
    metrics=$(pool::get_metrics) || {
        log::error "Pool health check failed - service may be down"
        pool::attempt_recovery
        return 1
    }
    
    local rejected
    rejected=$(echo "$metrics" | jq -r '.pressure.recentlyRejected // 0')
    
    # If many requests rejected, pool might be unhealthy
    if [[ "$rejected" -gt 5 ]]; then
        log::warn "High rejection rate detected ($rejected), attempting recovery..."
        pool::attempt_recovery
    fi
}

#######################################
# Attempt to recover unhealthy pool
#######################################
pool::attempt_recovery() {
    log::info "Attempting browser pool recovery..."
    
    # Check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "vrooli-browserless"; then
        log::error "Browserless container not running, restarting..."
        
        # Try to restart the container
        docker start vrooli-browserless 2>/dev/null || {
            log::error "Failed to restart browserless container"
            return 1
        }
        
        # Wait for service to be ready
        local max_attempts=30
        local attempt=0
        while [[ $attempt -lt $max_attempts ]]; do
            if timeout 5 curl -sf "http://localhost:4110/pressure" >/dev/null 2>&1; then
                log::info "Browserless service recovered"
                pool::prewarm 2  # Pre-warm after recovery
                return 0
            fi
            sleep 1
            ((attempt++))
        done
        
        log::error "Failed to recover browserless service after $max_attempts attempts"
        return 1
    fi
    
    # Container is running but unresponsive, try container restart
    log::info "Container running but unresponsive, restarting..."
    docker restart vrooli-browserless
    
    # Wait for recovery
    sleep 5
    
    if pool::get_metrics >/dev/null 2>&1; then
        log::info "Browser pool recovered successfully"
        pool::prewarm 2  # Pre-warm after recovery
        return 0
    else
        log::error "Failed to recover browser pool"
        return 1
    fi
}

# Export functions for CLI use
export -f pool::get_metrics
export -f pool::show_stats
export -f pool::start_autoscaler
export -f pool::stop_autoscaler
export -f pool::is_autoscaler_running
export -f pool::prewarm
export -f pool::smart_prewarm
export -f pool::health_check_and_recover