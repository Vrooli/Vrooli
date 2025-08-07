#!/usr/bin/env bash
# Node-RED Status and Monitoring Functions
# Functions for displaying status, metrics, and monitoring Node-RED

# Source shared secrets management library
# Use the same project root detection method as the secrets library
_node_red_status_detect_project_root() {
    local current_dir
    current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    # Walk up directory tree looking for .vrooli directory
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/.vrooli" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    # Fallback: assume we're in scripts and go up to project root
    echo "/home/matthalloran8/Vrooli"
}

PROJECT_ROOT="$(_node_red_status_detect_project_root)"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/helpers/utils/secrets.sh"

#######################################
# Show comprehensive Node-RED status
#######################################
node_red::show_status() {
    node_red::show_status_header
    
    if ! node_red::is_installed; then
        node_red::show_not_installed
        return 0
    fi
    
    if node_red::is_running; then
        node_red::show_running_status
        
        # Container information
        local container_info
        container_info=$(node_red::get_container_info)
        if [[ -n "$container_info" ]]; then
            node_red::show_container_info "$container_info"
        fi
        
        # Resource usage
        node_red::show_resource_usage
        
        # Health status
        node_red::show_health_status
        
        # Flow information
        node_red::show_flow_info
    else
        node_red::show_stopped_status
    fi
}

#######################################
# Show detailed system metrics
#######################################
node_red::show_metrics() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    node_red::show_metrics_header
    
    # Container stats
    node_red::get_docker_stats
    
    # Flow metrics
    node_red::show_flow_metrics
    
    # Node.js process metrics
    node_red::show_nodejs_metrics
}

#######################################
# View Node-RED logs
#######################################
node_red::view_logs() {
    local follow="${FOLLOW:-no}"
    local lines="${LOG_LINES:-$NODE_RED_LOG_LINES}"
    
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    if [[ "$follow" == "yes" ]]; then
        docker logs -f "$CONTAINER_NAME"
    else
        docker logs --tail "$lines" "$CONTAINER_NAME"
    fi
}

#######################################
# Show service information
#######################################
node_red::show_info() {
    echo "Node-RED Service Information"
    echo "============================"
    echo
    
    # Basic information
    echo "Configuration:"
    echo "- Resource Name: $RESOURCE_NAME"
    echo "- Category: $RESOURCE_CATEGORY"
    echo "- Port: $RESOURCE_PORT"
    echo "- Container: $CONTAINER_NAME"
    echo "- Network: $NETWORK_NAME" 
    echo "- Volume: $VOLUME_NAME"
    echo
    
    # Image information
    echo "Images:"
    echo "- Official: $OFFICIAL_IMAGE"
    echo "- Custom: $IMAGE_NAME"
    echo "- Custom Enabled: $NODE_RED_ENABLE_CUSTOM_IMAGE"
    echo
    
    # Feature flags
    echo "Features:"
    echo "- Host Access: $NODE_RED_ENABLE_HOST_ACCESS"
    echo "- Docker Socket: $NODE_RED_ENABLE_DOCKER_SOCKET"
    echo
    
    # Directories
    echo "Directories:"
    echo "- Script Directory: $SCRIPT_DIR"
    echo "- Flows: $SCRIPT_DIR/flows"
    echo "- Nodes: $SCRIPT_DIR/nodes"
    echo "- Docker Files: $SCRIPT_DIR/docker"
    echo
    
    # Health check settings
    echo "Health Check:"
    echo "- Interval: ${NODE_RED_HEALTH_CHECK_INTERVAL}s"
    echo "- Max Attempts: $NODE_RED_HEALTH_CHECK_MAX_ATTEMPTS"
    echo "- Timeout: ${NODE_RED_HEALTH_CHECK_TIMEOUT}s"
    echo
    
    # Configuration files
    echo "Configuration:"
    local config_file
    config_file="$(secrets::get_project_config_file)"
    if [[ -f "$config_file" ]] && jq -e '.services.automation."node-red"' "$config_file" >/dev/null 2>&1; then
        echo "- Resource Config: ✓ Configured"
    else
        echo "- Resource Config: ✗ Missing"
    fi
    
    if [[ -f "$SCRIPT_DIR/settings.js" ]]; then
        echo "- Settings File: ✓ Present"
    else
        echo "- Settings File: ✗ Missing"
    fi
}

#######################################
# Check Node-RED health status
#######################################
node_red::health_check() {
    local exit_code=0
    
    echo "Node-RED Health Check"
    echo "===================="
    echo
    
    # Container status
    echo -n "Container Status: "
    if node_red::is_installed; then
        if node_red::is_running; then
            echo "✓ Running"
        else
            echo "✗ Stopped"
            exit_code=1
        fi
    else
        echo "✗ Not Installed"
        exit_code=1
    fi
    
    # HTTP endpoint
    echo -n "HTTP Endpoint: "
    if node_red::is_healthy; then
        echo "✓ Responsive"
    else
        echo "✗ Not Responsive"
        exit_code=1
    fi
    
    # Docker health status
    echo -n "Docker Health: "
    local health_status
    health_status=$(node_red::get_health_status)
    case "$health_status" in
        "healthy")
            echo "✓ Healthy"
            ;;
        "unhealthy")
            echo "✗ Unhealthy"
            exit_code=1
            ;;
        "starting")
            echo "⚠ Starting"
            ;;
        "not-installed")
            echo "✗ Not Installed"
            exit_code=1
            ;;
        *)
            echo "? Unknown ($health_status)"
            exit_code=1
            ;;
    esac
    
    # Port availability
    echo -n "Port $RESOURCE_PORT: "
    if node_red::is_running; then
        echo "✓ In Use (Node-RED)"
    elif node_red::check_port "$RESOURCE_PORT"; then
        echo "✓ Available"
    else
        echo "✗ In Use (Other Service)"
        exit_code=1
    fi
    
    # Configuration files
    echo -n "Configuration: "
    local config_file
    config_file="$(secrets::get_project_config_file)"
    if [[ -f "$config_file" ]] && jq -e '.services.automation."node-red"' "$config_file" >/dev/null 2>&1; then
        echo "✓ Valid"
    else
        echo "✗ Missing/Invalid"
        exit_code=1
    fi
    
    # Settings file
    echo -n "Settings File: "
    if [[ -f "$SCRIPT_DIR/settings.js" ]]; then
        echo "✓ Present"
    else
        echo "✗ Missing"
        exit_code=1
    fi
    
    # Flow directory
    echo -n "Flows Directory: "
    if [[ -d "$SCRIPT_DIR/flows" ]]; then
        local flow_count=$(find "$SCRIPT_DIR/flows" -name "*.json" -type f 2>/dev/null | wc -l)
        echo "✓ Present ($flow_count flows)"
    else
        echo "✗ Missing"
        exit_code=1
    fi
    
    echo
    if [[ $exit_code -eq 0 ]]; then
        echo "Overall Status: ✓ Healthy"
    else
        echo "Overall Status: ✗ Issues Found"
    fi
    
    return $exit_code
}

#######################################
# Show resource usage over time
#######################################
node_red::monitor() {
    local interval="${1:-5}"
    local count="${2:-0}"  # 0 means infinite
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    echo "Node-RED Resource Monitor (interval: ${interval}s)"
    echo "Press Ctrl+C to stop"
    echo
    
    local iteration=0
    while [[ $count -eq 0 || $iteration -lt $count ]]; do
        clear
        echo "Node-RED Resource Monitor - $(date)"
        echo "=================================="
        echo
        
        # Basic status
        echo "Status: Running"
        echo "URL: http://localhost:$RESOURCE_PORT"
        echo "Uptime: $(docker ps -f name="$CONTAINER_NAME" --format "{{.Status}}" | tail -n 1)"
        echo
        
        # Resource usage
        echo "Resource Usage:"
        node_red::get_docker_stats
        echo
        
        # Health status
        local health_status
        health_status=$(node_red::get_health_status)
        echo "Health: $health_status"
        echo
        
        # Flow count
        local flow_count=$(docker exec "$CONTAINER_NAME" find /data -name "*.json" -type f 2>/dev/null | wc -l || echo "0")
        echo "Flow Files: $flow_count"
        echo
        
        # Disk usage
        local disk_usage=$(docker exec "$CONTAINER_NAME" du -sh /data 2>/dev/null | cut -f1 || echo "unknown")
        echo "Data Directory: $disk_usage"
        
        if [[ $count -ne 0 ]]; then
            ((iteration++))
            if [[ $iteration -ge $count ]]; then
                break
            fi
        fi
        
        sleep "$interval"
    done
}

#######################################
# Show recent log entries with timestamps
#######################################
node_red::show_recent_logs() {
    local lines="${1:-20}"
    
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    echo "Recent Node-RED Logs (last $lines lines)"
    echo "========================================"
    echo
    
    docker logs --tail "$lines" --timestamps "$CONTAINER_NAME"
}

#######################################
# Show error logs only
#######################################
node_red::show_error_logs() {
    local lines="${1:-50}"
    
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    echo "Node-RED Error Logs (last $lines lines)"
    echo "======================================"
    echo
    
    docker logs --tail "$lines" "$CONTAINER_NAME" 2>&1 | grep -i "error\|warning\|fail\|exception" || echo "No errors found in recent logs"
}

#######################################
# Show container resource limits
#######################################
node_red::show_resource_limits() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    echo "Node-RED Resource Limits"
    echo "======================="
    echo
    
    # Get container resource configuration
    local container_info
    container_info=$(docker inspect "$CONTAINER_NAME" 2>/dev/null)
    
    if [[ -n "$container_info" ]]; then
        echo "Memory Limit: $(echo "$container_info" | jq -r '.[0].HostConfig.Memory // "unlimited"')"
        echo "CPU Limit: $(echo "$container_info" | jq -r '.[0].HostConfig.CpuQuota // "unlimited"')"
        echo "Restart Policy: $(echo "$container_info" | jq -r '.[0].HostConfig.RestartPolicy.Name')"
    else
        echo "Unable to retrieve container information"
    fi
}

#######################################
# Show network information
#######################################
node_red::show_network_info() {
    echo "Node-RED Network Information"
    echo "==========================="
    echo
    
    # Container network
    if node_red::is_running; then
        echo "Container Network:"
        docker inspect "$CONTAINER_NAME" --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null | head -1 | sed 's/^/- IP Address: /'
        echo "- Network: $NETWORK_NAME"
        echo "- Port Mapping: $RESOURCE_PORT:1880"
        echo
    fi
    
    # Network details
    if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
        echo "Network Details:"
        docker network inspect "$NETWORK_NAME" --format '{{.IPAM.Config}}' | sed 's/^/- Subnet: /'
        echo "- Driver: $(docker network inspect "$NETWORK_NAME" --format '{{.Driver}}')"
        
        # Connected containers
        local connected_containers
        connected_containers=$(docker network inspect "$NETWORK_NAME" --format '{{len .Containers}}' 2>/dev/null)
        echo "- Connected Containers: $connected_containers"
    else
        echo "Network not found: $NETWORK_NAME"
    fi
}

#######################################
# Export status report
#######################################
node_red::export_status_report() {
    local output_file="${1:-./node-red-status-$(date +%Y%m%d-%H%M%S).txt}"
    
    echo "Generating Node-RED status report..."
    
    {
        echo "Node-RED Status Report"
        echo "====================="
        echo "Generated: $(date)"
        echo "Host: $(hostname)"
        echo
        
        node_red::show_info
        echo
        node_red::health_check
        echo
        node_red::show_network_info
        echo
        node_red::show_resource_limits
        echo
        echo "Recent Logs:"
        echo "============"
        node_red::show_recent_logs 50
        
    } > "$output_file"
    
    log::success "Status report exported to: $output_file"
}