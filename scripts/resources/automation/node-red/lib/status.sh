#!/usr/bin/env bash
# Node-RED Status Display Functions
# Provides unified status display using status engine

NODE_RED_STATUS_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${NODE_RED_STATUS_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"

#######################################
# Display Node-RED status using status engine
# Returns: 0 on success, 1 on failure
#######################################
node_red::status() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    local config
    config=$(node_red::get_status_config)
    status::display_unified_status "$config" "node_red::display_additional_info"
}

#######################################
# Display additional Node-RED-specific status information
# Called by status engine for custom information
#######################################
node_red::display_additional_info() {
    echo
    echo "🔧 Node-RED Specific Information:"
    
    # Show flows information
    if [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        local flow_count
        flow_count=$(jq length "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" 2>/dev/null || echo "0")
        echo "   📋 Flows: $flow_count loaded"
    else
        echo "   📋 Flows: No flows file found"
    fi
    
    # Show credentials information
    if [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_CREDENTIALS_FILE" ]]; then
        echo "   🔐 Credentials: File exists"
    else
        echo "   🔐 Credentials: No credentials file"
    fi
    
    # Show authentication status
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        echo "   🛡️  Authentication: Enabled (username: $AUTH_USERNAME)"
    else
        echo "   🛡️  Authentication: Disabled"
    fi
    
    # Show Node-RED version if container is running
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        local version
        version=$(docker exec "$NODE_RED_CONTAINER_NAME" node -e "console.log(require('/usr/src/node-red/package.json').version)" 2>/dev/null || echo "unknown")
        echo "   📦 Version: $version"
    fi
    
    # Show recent activity
    if docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        echo
        echo "📊 Recent Activity:"
        docker logs --tail 5 "$NODE_RED_CONTAINER_NAME" 2>/dev/null | while IFS= read -r line; do
            echo "   $line"
        done
    fi
    
    # Show resource usage if available
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        echo
        echo "💾 Resource Usage:"
        local stats
        stats=$(docker stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}" "$NODE_RED_CONTAINER_NAME" 2>/dev/null | tail -n +2)
        if [[ -n "$stats" ]]; then
            local cpu mem
            cpu=$(echo "$stats" | awk '{print $1}')
            mem=$(echo "$stats" | awk '{print $2}')
            echo "   🔥 CPU: $cpu"
            echo "   🧠 Memory: $mem"
        fi
    fi
}

#######################################
# Get Node-RED status as JSON
# Returns: JSON status information
#######################################
node_red::get_status_json() {
    local container_status="not_installed"
    local service_status="stopped"
    local port_accessible=false
    local flows_count=0
    local has_credentials=false
    local version="unknown"
    
    if docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
            container_status="running"
            
            # Check if service is responding
            if http::check_endpoint "$NODE_RED_BASE_URL"; then
                service_status="running"
                port_accessible=true
            else
                service_status="starting"
            fi
            
            # Get version
            version=$(docker exec "$NODE_RED_CONTAINER_NAME" node -e "console.log(require('/usr/src/node-red/package.json').version)" 2>/dev/null || echo "unknown")
        else
            container_status="stopped"
        fi
    fi
    
    # Check flows
    if [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        flows_count=$(jq length "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" 2>/dev/null || echo "0")
    fi
    
    # Check credentials
    if [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_CREDENTIALS_FILE" ]]; then
        has_credentials=true
    fi
    
    echo '{
        "service": "Node-RED",
        "container_status": "'$container_status'",
        "service_status": "'$service_status'",
        "port_accessible": '$port_accessible',
        "version": "'$version'",
        "configuration": {
            "port": '$NODE_RED_PORT',
            "data_dir": "'$NODE_RED_DATA_DIR'",
            "authentication_enabled": '$([[ "$BASIC_AUTH" == "yes" ]] && echo "true" || echo "false")',
            "flows_count": '$flows_count',
            "has_credentials": '$has_credentials'
        },
        "urls": {
            "editor": "http://localhost:'$NODE_RED_PORT'",
            "settings": "http://localhost:'$NODE_RED_PORT'/settings",
            "flows": "http://localhost:'$NODE_RED_PORT'/flows"
        },
        "timestamp": "'$(date -Iseconds)'"
    }'
}