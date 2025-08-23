#!/usr/bin/env bash
# Browserless Health Check Functions
# Tiered health checking using health framework

#######################################
# Get health check configuration
# Returns: JSON configuration for health framework
#######################################
browserless::get_health_config() {
    echo '{
        "container_name": "'$BROWSERLESS_CONTAINER_NAME'",
        "service_name": "Browserless",
        "checks": {
            "basic": "browserless::check_basic_health",
            "api": "browserless::check_api_health",
            "functional": "browserless::check_functional_health"
        },
        "timeout": 30
    }'
}

#######################################
# Perform health check
#######################################
browserless::health() {
    local config
    config=$(browserless::get_health_config)
    health::tiered_check "$config"
}

#######################################
# Basic health check (container running)
#######################################
browserless::check_basic_health() {
    # Check if container is running
    if ! docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        return 1
    fi
    
    # Check if port is responding
    if ! http::check_endpoint "http://localhost:$BROWSERLESS_PORT/pressure"; then
        return 1
    fi
    
    return 0
}

#######################################
# API health check
#######################################
browserless::check_api_health() {
    local endpoints=(
        "/pressure"
        "/metrics"
        "/config"
    )
    
    for endpoint in "${endpoints[@]}"; do
        if ! http::check_endpoint "http://localhost:$BROWSERLESS_PORT$endpoint"; then
            log::debug "API endpoint $endpoint is not responding"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Functional health check
#######################################
browserless::check_functional_health() {
    local pressure_data
    pressure_data=$(http::get "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null)
    
    if [[ -z "$pressure_data" ]]; then
        log::debug "Unable to get pressure data"
        return 1
    fi
    
    # Check if we can parse the pressure data (now under .pressure key)
    local running queued
    running=$(echo "$pressure_data" | jq -r '.pressure.running // 0' 2>/dev/null || echo "0")
    queued=$(echo "$pressure_data" | jq -r '.pressure.queued // 0' 2>/dev/null || echo "0")
    
    # Verify we got valid numbers
    if ! [[ "$running" =~ ^[0-9]+$ ]] || ! [[ "$queued" =~ ^[0-9]+$ ]]; then
        log::debug "Invalid pressure data format"
        return 1
    fi
    
    # Check browser capacity
    local max_concurrent
    max_concurrent=$(echo "$pressure_data" | jq -r '.pressure.maxConcurrent // 5' 2>/dev/null || echo "5")
    
    if [[ "$running" -ge "$max_concurrent" ]]; then
        log::warn "Browserless is at maximum capacity ($running/$max_concurrent)"
        # Not a failure, just a warning
    fi
    
    return 0
}

#######################################
# Get detailed health status
#######################################
browserless::get_health_details() {
    local health_status="Unknown"
    local details=""
    
    if browserless::check_basic_health; then
        health_status="Running"
        
        if browserless::check_api_health; then
            health_status="Healthy"
            
            if browserless::check_functional_health; then
                health_status="Fully Operational"
                
                # Get pressure details
                local pressure_data
                pressure_data=$(http::get "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null)
                
                if [[ -n "$pressure_data" ]]; then
                    local running queued max_concurrent
                    running=$(echo "$pressure_data" | jq -r '.pressure.running // 0' 2>/dev/null || echo "0")
                    queued=$(echo "$pressure_data" | jq -r '.pressure.queued // 0' 2>/dev/null || echo "0")
                    max_concurrent=$(echo "$pressure_data" | jq -r '.pressure.maxConcurrent // 5' 2>/dev/null || echo "5")
                    
                    details="Browsers: $running running, $queued queued (max: $max_concurrent)"
                fi
            else
                details="API endpoints responding but functional checks failing"
            fi
        else
            details="Container running but API not responding"
        fi
    else
        health_status="Not Running"
        details="Container is not running or port is not accessible"
    fi
    
    echo "{\"status\": \"$health_status\", \"details\": \"$details\"}"
}

#######################################
# Monitor health continuously
#######################################
browserless::monitor_health() {
    local interval="${1:-10}"
    local duration="${2:-0}"  # 0 means infinite
    local start_time=$(date +%s)
    
    log::info "Monitoring Browserless health (interval: ${interval}s)..."
    
    while true; do
        clear
        echo "======================================"
        echo "Browserless Health Monitor"
        echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "======================================"
        echo
        
        local health_details
        health_details=$(browserless::get_health_details)
        
        echo "Status: $(echo "$health_details" | jq -r '.status')"
        echo "Details: $(echo "$health_details" | jq -r '.details')"
        echo
        
        # Show container stats if running
        if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
            echo "Container Statistics:"
            docker stats --no-stream "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || true
            echo
            
            # Show metrics
            local metrics
            metrics=$(http::get "http://localhost:$BROWSERLESS_PORT/metrics" 2>/dev/null || echo "")
            if [[ -n "$metrics" ]]; then
                echo "Metrics Summary:"
                echo "$metrics" | grep -E "^browserless_" | head -10
            fi
        fi
        
        # Check if we should stop
        if [[ $duration -gt 0 ]]; then
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))
            if [[ $elapsed -ge $duration ]]; then
                break
            fi
        fi
        
        echo
        echo "Press Ctrl+C to stop monitoring..."
        sleep "$interval"
    done
}

#######################################
# Quick health status for scripts
#######################################
browserless::is_healthy() {
    browserless::check_basic_health && browserless::check_api_health
}