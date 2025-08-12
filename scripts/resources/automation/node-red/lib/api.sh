#!/usr/bin/env bash
# Node-RED API and Flow Management Functions
# Streamlined functions for Node-RED admin API operations

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh" 2>/dev/null || true

#######################################
# List all flows in Node-RED
#######################################
node_red::list_flows() {
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not running"
        return 1
    fi
    
    log::info "Fetching flow information..."
    
    local response
    response=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$NODE_RED_PORT/flows" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Failed to fetch flows from Node-RED API"
        return 1
    fi
    
    echo "$response" | jq -r '.[] | select(.type == "tab") | "ID: \(.id)\nName: \(.label)\nDisabled: \(.disabled // false)\n"' 2>/dev/null || {
        log::warning "No flows found or invalid response"
        echo "$response"
    }
}

#######################################
# Export flows to a file
#######################################
node_red::export_flows() {
    local output_file="${1:-${OUTPUT:-./node-red-flows-export-$(date +%Y%m%d-%H%M%S).json}}"
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not running"
        return 1
    fi
    
    log::info "Exporting flows to: $output_file"
    
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$NODE_RED_PORT/flows" > "$output_file" 2>/dev/null; then
        # Pretty print the JSON
        if jq . "$output_file" > "$output_file.tmp" 2>/dev/null && mv "$output_file.tmp" "$output_file"; then
            local flow_count
            flow_count=$(jq '[.[] | select(.type == "tab")] | length' "$output_file" 2>/dev/null || echo "0")
            log::success "Exported $flow_count flows to: $output_file"
        else
            log::warning "Flows exported but JSON formatting failed"
        fi
        return 0
    else
        log::error "Failed to export flows"
        return 1
    fi
}

#######################################
# Import flows from a file
#######################################
node_red::import_flows() {
    local flow_file="${1:-$FLOW_FILE}"
    
    if [[ -z "$flow_file" ]]; then
        log::error "Flow file is required"
        return 1
    fi
    
    if [[ ! -f "$flow_file" ]]; then
        log::error "Flow file not found: $flow_file"
        return 1
    fi
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not running"
        return 1
    fi
    
    # Validate JSON
    if ! jq empty "$flow_file" 2>/dev/null; then
        log::error "Invalid JSON in flow file"
        return 1
    fi
    
    log::info "Importing flows from: $flow_file"
    
    # Import via admin API
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" \
        -X POST -H "Content-Type: application/json" \
        -d "@$flow_file" \
        "http://localhost:$NODE_RED_PORT/flows" >/dev/null 2>&1; then
        
        log::success "Flows imported successfully"
        return 0
    else
        log::error "Failed to import flows"
        return 1
    fi
}

#######################################
# Execute a flow via HTTP endpoint
#######################################
node_red::execute_flow() {
    local endpoint="${1:-$ENDPOINT}"
    local data="${2:-$DATA}"
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not running"
        return 1
    fi
    
    if [[ -z "$endpoint" ]]; then
        log::error "Endpoint is required for flow execution"
        return 1
    fi
    
    local url="http://localhost:$NODE_RED_PORT$endpoint"
    log::info "Executing flow endpoint: $url"
    
    local response
    if [[ -n "$data" ]]; then
        response=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" -X POST -H "Content-Type: application/json" -d "$data" "$url" 2>&1)
    else
        response=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "$url" 2>&1)
    fi
    
    local exit_code=$?
    if [[ $exit_code -eq 0 ]]; then
        log::success "Flow executed successfully"
        [[ -n "$response" ]] && echo "Response: $response"
        return 0
    else
        log::error "Flow execution failed"
        [[ -n "$response" ]] && echo "Error: $response"
        return 1
    fi
}

#######################################
# Enable a flow
#######################################
node_red::enable_flow() {
    local flow_id="$1"
    
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID is required"
        return 1
    fi
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not running"
        return 1
    fi
    
    log::info "Enabling flow: $flow_id"
    
    # Get current flow and remove disabled property
    local current_flow
    current_flow=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$NODE_RED_PORT/flow/$flow_id" 2>/dev/null)
    
    if [[ -z "$current_flow" ]]; then
        log::error "Flow not found: $flow_id"
        return 1
    fi
    
    local updated_flow
    updated_flow=$(echo "$current_flow" | jq 'del(.disabled)')
    
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" \
        -X PUT -H "Content-Type: application/json" \
        -d "$updated_flow" \
        "http://localhost:$NODE_RED_PORT/flow/$flow_id" >/dev/null 2>&1; then
        log::success "Flow enabled successfully"
    else
        log::error "Failed to enable flow"
        return 1
    fi
}

#######################################
# Disable a flow
#######################################
node_red::disable_flow() {
    local flow_id="$1"
    
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID is required"
        return 1
    fi
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not running"
        return 1
    fi
    
    log::info "Disabling flow: $flow_id"
    
    # Get current flow and add disabled property
    local current_flow
    current_flow=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$NODE_RED_PORT/flow/$flow_id" 2>/dev/null)
    
    if [[ -z "$current_flow" ]]; then
        log::error "Flow not found: $flow_id"
        return 1
    fi
    
    local updated_flow
    updated_flow=$(echo "$current_flow" | jq '.disabled = true')
    
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" \
        -X PUT -H "Content-Type: application/json" \
        -d "$updated_flow" \
        "http://localhost:$NODE_RED_PORT/flow/$flow_id" >/dev/null 2>&1; then
        log::success "Flow disabled successfully"
    else
        log::error "Failed to disable flow"
        return 1
    fi
}


