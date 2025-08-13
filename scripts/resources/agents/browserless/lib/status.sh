#!/usr/bin/env bash
# Browserless Status Display Functions
# Uses status-engine framework for consistent display

# Source required libraries
BROWSERLESS_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"

#######################################
# Get status configuration
# Returns: JSON configuration for status engine
#######################################
browserless::get_status_config() {
    local config='{
        "resource": {
            "name": "Browserless",
            "category": "agents",
            "description": "High-performance headless Chrome automation service",
            "container_name": "'$BROWSERLESS_CONTAINER_NAME'"
        },
        "health": {
            "check_func": "browserless::is_healthy",
            "endpoints": {
                "main": "http://localhost:'$BROWSERLESS_PORT'/pressure",
                "metrics": "http://localhost:'$BROWSERLESS_PORT'/metrics",
                "config": "http://localhost:'$BROWSERLESS_PORT'/config"
            }
        },
        "service": {
            "ports": {
                "api": '$BROWSERLESS_PORT'
            },
            "endpoints": {
                "Dashboard": "http://localhost:'$BROWSERLESS_PORT'",
                "Pressure": "http://localhost:'$BROWSERLESS_PORT'/pressure",
                "Metrics": "http://localhost:'$BROWSERLESS_PORT'/metrics",
                "Config": "http://localhost:'$BROWSERLESS_PORT'/config"
            }
        },
        "paths": {
            "data": "'$BROWSERLESS_DATA_DIR'",
            "screenshots": "'$BROWSERLESS_DATA_DIR'/screenshots",
            "downloads": "'$BROWSERLESS_DATA_DIR'/downloads"
        }
    }'
    
    echo "$config"
}

#######################################
# Display status using status engine
#######################################
browserless::status() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    local config
    config=$(browserless::get_status_config)
    status::display_unified_status "$config" "browserless::display_additional_info"
}

#######################################
# Display additional Browserless-specific information
#######################################
browserless::display_additional_info() {
    local container_running=false
    
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        container_running=true
    fi
    
    # Display browser pressure information
    if [[ "$container_running" == "true" ]]; then
        echo
        echo "Browser Pressure:"
        echo "----------------"
        
        local pressure_data
        pressure_data=$(http::get "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null || echo "{}")
        
        if [[ -n "$pressure_data" ]] && [[ "$pressure_data" != "{}" ]]; then
            local running queued max_concurrent cpu memory
            running=$(echo "$pressure_data" | jq -r '.pressure.running // 0' 2>/dev/null || echo "0")
            queued=$(echo "$pressure_data" | jq -r '.pressure.queued // 0' 2>/dev/null || echo "0")
            max_concurrent=$(echo "$pressure_data" | jq -r '.pressure.maxConcurrent // 5' 2>/dev/null || echo "5")
            cpu=$(echo "$pressure_data" | jq -r '.pressure.cpu // "N/A"' 2>/dev/null || echo "N/A")
            memory=$(echo "$pressure_data" | jq -r '.pressure.memory // "N/A"' 2>/dev/null || echo "N/A")
            
            echo "  Running Browsers: $running / $max_concurrent"
            echo "  Queued Requests: $queued"
            
            if [[ "$cpu" != "N/A" ]]; then
                echo "  CPU Usage: ${cpu}%"
            fi
            
            if [[ "$memory" != "N/A" ]]; then
                echo "  Memory Usage: ${memory}%"
            fi
            
            # Show warning if at capacity
            if [[ "$running" -ge "$max_concurrent" ]]; then
                echo
                log::warn "Browserless is at maximum capacity!"
                echo "  Consider increasing MAX_BROWSERS if performance degrades"
            fi
        else
            echo "  Unable to retrieve pressure data"
        fi
        
        # Display workspace statistics
        echo
        echo "Workspace Statistics:"
        echo "--------------------"
        
        if [[ -d "$BROWSERLESS_DATA_DIR" ]]; then
            local screenshot_count download_count upload_count workspace_size
            screenshot_count=$(find "$BROWSERLESS_DATA_DIR/screenshots" -type f 2>/dev/null | wc -l)
            download_count=$(find "$BROWSERLESS_DATA_DIR/downloads" -type f 2>/dev/null | wc -l)
            upload_count=$(find "$BROWSERLESS_DATA_DIR/uploads" -type f 2>/dev/null | wc -l)
            workspace_size=$(du -sh "$BROWSERLESS_DATA_DIR" 2>/dev/null | cut -f1)
            
            echo "  Screenshots: $screenshot_count files"
            echo "  Downloads: $download_count files"
            echo "  Uploads: $upload_count files"
            echo "  Total Size: $workspace_size"
        else
            echo "  Workspace directory not found"
        fi
        
        # Display recent activity (if logs available)
        echo
        echo "Recent Activity:"
        echo "---------------"
        
        local recent_logs
        recent_logs=$(docker logs --tail 5 "$BROWSERLESS_CONTAINER_NAME" 2>&1 | grep -E "Chrome|Browser|Session" | tail -3)
        
        if [[ -n "$recent_logs" ]]; then
            echo "$recent_logs" | while IFS= read -r line; do
                echo "  $line"
            done
        else
            echo "  No recent browser activity"
        fi
    fi
    
    # Display configuration
    echo
    echo "Configuration:"
    echo "--------------"
    echo "  Max Browsers: ${MAX_BROWSERS:-5}"
    echo "  Timeout: ${TIMEOUT:-30000}ms"
    echo "  Headless Mode: ${HEADLESS:-yes}"
    echo "  Shared Memory: ${BROWSERLESS_DOCKER_SHM_SIZE:-2gb}"
    
    # Display available actions
    echo
    echo "Available Actions:"
    echo "-----------------"
    
    if [[ "$container_running" == "true" ]]; then
        echo "  ./manage.sh --action screenshot --url <URL>  # Take a screenshot"
        echo "  ./manage.sh --action pdf --url <URL>         # Generate PDF"
        echo "  ./manage.sh --action scrape --url <URL>      # Scrape content"
        echo "  ./manage.sh --action test                    # Run tests"
        echo "  ./manage.sh --action logs                    # View logs"
        echo "  ./manage.sh --action restart                 # Restart service"
    else
        echo "  ./manage.sh --action install                 # Install Browserless"
        echo "  ./manage.sh --action start                   # Start container"
    fi
}

#######################################
# Get compact status for integration
#######################################
browserless::get_compact_status() {
    local status="Stopped"
    local details=""
    
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        status="Running"
        
        # Get pressure info
        local pressure_data
        pressure_data=$(http::get "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null || echo "{}")
        
        if [[ -n "$pressure_data" ]] && [[ "$pressure_data" != "{}" ]]; then
            local running max_concurrent
            running=$(echo "$pressure_data" | jq -r '.pressure.running // 0' 2>/dev/null || echo "0")
            max_concurrent=$(echo "$pressure_data" | jq -r '.pressure.maxConcurrent // 5' 2>/dev/null || echo "5")
            details="($running/$max_concurrent browsers)"
        fi
    fi
    
    echo "Browserless: $status $details"
}

#######################################
# Legacy function compatibility
#######################################
browserless::show_status() {
    browserless::status
}

browserless::show_info() {
    browserless::info
}