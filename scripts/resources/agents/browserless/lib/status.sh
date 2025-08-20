#!/usr/bin/env bash
# Browserless Status Display Functions
# Uses status-engine framework for consistent display

# Source required libraries
BROWSERLESS_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/format.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/../../../lib/status-args.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/health.sh"

#######################################
# Get status configuration
# Returns: JSON configuration for status engine
#######################################
browserless::get_status_config() {
    local config='{
        "resource": {
            "name": "browserless",
            "category": "agents",
            "description": "High-performance headless Chrome automation service",
            "port": '$BROWSERLESS_PORT',
            "container_name": "'$BROWSERLESS_CONTAINER_NAME'",
            "data_dir": "'$BROWSERLESS_DATA_DIR'"
        },
        "endpoints": {
            "ui": "http://localhost:'$BROWSERLESS_PORT'",
            "api": "http://localhost:'$BROWSERLESS_PORT'/",
            "health": "http://localhost:'$BROWSERLESS_PORT'/pressure"
        },
        "health_tiers": {
            "healthy": "All systems operational - browsers ready",
            "degraded": "Service degraded - check browser pressure and memory usage",
            "unhealthy": "Service failed - restart container or check logs"
        },
        "auth": {
            "type": "none"
        }
    }'
    
    echo "$config"
}

#######################################
# Collect Browserless status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
browserless::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if docker::container_exists "$BROWSERLESS_CONTAINER_NAME"; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
            running="true"
            
            if browserless::is_healthy; then
                healthy="true"
                health_message="Healthy - All systems operational - browsers ready"
            else
                health_message="Unhealthy - Service not responding or degraded"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container not found"
    fi
    
    # Basic resource information
    status_data+=("name" "browserless")
    status_data+=("category" "agents")
    status_data+=("description" "High-performance headless Chrome automation service")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$BROWSERLESS_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$BROWSERLESS_PORT")
    
    # Service endpoints
    status_data+=("ui_url" "http://localhost:$BROWSERLESS_PORT")
    status_data+=("api_url" "http://localhost:$BROWSERLESS_PORT/")
    status_data+=("health_url" "http://localhost:$BROWSERLESS_PORT/pressure")
    
    if [[ "$running" == "true" ]]; then
        # Image information
        local image
        image=$(docker inspect --format='{{.Config.Image}}' "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        status_data+=("image" "$image")
        
        # Configuration
        status_data+=("max_browsers" "${MAX_BROWSERS:-5}")
        status_data+=("timeout" "${TIMEOUT:-30000}")
        status_data+=("headless_mode" "${HEADLESS:-yes}")
        status_data+=("shm_size" "${BROWSERLESS_DOCKER_SHM_SIZE:-2gb}")
        
        # Browser pressure data (skip if fast mode)
        if [[ "$fast_mode" == "false" ]]; then
            local pressure_data
            pressure_data=$(timeout 2s http::get "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null || echo "{}")
            
            if [[ -n "$pressure_data" ]] && [[ "$pressure_data" != "{}" ]]; then
                local running_browsers queued max_concurrent cpu memory
                running_browsers=$(echo "$pressure_data" | jq -r '.pressure.running // 0' 2>/dev/null || echo "0")
                queued=$(echo "$pressure_data" | jq -r '.pressure.queued // 0' 2>/dev/null || echo "0")
                max_concurrent=$(echo "$pressure_data" | jq -r '.pressure.maxConcurrent // 5' 2>/dev/null || echo "5")
                cpu=$(echo "$pressure_data" | jq -r '.pressure.cpu // "N/A"' 2>/dev/null || echo "N/A")
                memory=$(echo "$pressure_data" | jq -r '.pressure.memory // "N/A"' 2>/dev/null || echo "N/A")
                
                status_data+=("running_browsers" "$running_browsers")
                status_data+=("queued_requests" "$queued")
                status_data+=("max_concurrent" "$max_concurrent")
                if [[ "$cpu" != "N/A" ]]; then
                    status_data+=("cpu_usage" "${cpu}%")
                fi
                if [[ "$memory" != "N/A" ]]; then
                    status_data+=("memory_usage" "${memory}%")
                fi
            fi
        else
            status_data+=("running_browsers" "N/A")
            status_data+=("queued_requests" "N/A")
            status_data+=("max_concurrent" "N/A")
        fi
        
        # Workspace statistics (skip if fast mode)
        if [[ "$fast_mode" == "false" && -d "$BROWSERLESS_DATA_DIR" ]]; then
            local screenshot_count download_count upload_count workspace_size
            screenshot_count=$(timeout 3s find "$BROWSERLESS_DATA_DIR/screenshots" -type f 2>/dev/null | wc -l)
            download_count=$(timeout 3s find "$BROWSERLESS_DATA_DIR/downloads" -type f 2>/dev/null | wc -l)
            upload_count=$(timeout 3s find "$BROWSERLESS_DATA_DIR/uploads" -type f 2>/dev/null | wc -l)
            workspace_size=$(timeout 2s du -sh "$BROWSERLESS_DATA_DIR" 2>/dev/null | cut -f1)
            
            status_data+=("screenshot_files" "$screenshot_count")
            status_data+=("download_files" "$download_count")
            status_data+=("upload_files" "$upload_count")
            status_data+=("workspace_size" "$workspace_size")
        elif [[ "$fast_mode" == "true" ]]; then
            status_data+=("screenshot_files" "N/A")
            status_data+=("download_files" "N/A")
            status_data+=("upload_files" "N/A")
            status_data+=("workspace_size" "N/A")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display status using text format with status engine
#######################################
browserless::status::display_text() {
    local data_array=("$@")
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    local config
    config=$(browserless::get_status_config)
    status::display_unified_status "$config" "browserless::display_additional_info"
}

#######################################
# Display status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
browserless::status() {
    status::run_standard "browserless" "browserless::status::collect_data" "browserless::status::display_text" "$@"
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
        pressure_data=$(http::request "GET" "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null || echo "{}")
        
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
            screenshot_count=$(timeout 3s find "$BROWSERLESS_DATA_DIR/screenshots" -type f 2>/dev/null | wc -l)
            download_count=$(timeout 3s find "$BROWSERLESS_DATA_DIR/downloads" -type f 2>/dev/null | wc -l)
            upload_count=$(timeout 3s find "$BROWSERLESS_DATA_DIR/uploads" -type f 2>/dev/null | wc -l)
            workspace_size=$(timeout 2s du -sh "$BROWSERLESS_DATA_DIR" 2>/dev/null | cut -f1)
            
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
        pressure_data=$(http::request "GET" "http://localhost:$BROWSERLESS_PORT/pressure" 2>/dev/null || echo "{}")
        
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