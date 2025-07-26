#!/usr/bin/env bash
# Node-RED API and Flow Management Functions
# Functions for interacting with Node-RED admin API and managing flows

#######################################
# List all flows in Node-RED
#######################################
node_red::list_flows() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Fetching flow information..."
    
    # Get flows via admin API
    local response
    response=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$RESOURCE_PORT/flows" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        node_red::show_flow_fetch_error
        return 1
    fi
    
    # Parse and display flows
    echo "$response" | jq -r '.[] | select(.type == "tab") | "ID: \(.id)\nName: \(.label)\nDisabled: \(.disabled // false)\n"' 2>/dev/null || {
        log::warning "No flows found or invalid response"
        echo "$response"
    }
}

#######################################
# Export all flows to a file
#######################################
node_red::export_flows() {
    local output_file="${OUTPUT:-./node-red-flows-export-$(date +%Y%m%d-%H%M%S).json}"
    
    node_red::export_flows_to_file "$output_file"
}

#######################################
# Export flows to a specific file (internal function)
#######################################
node_red::export_flows_to_file() {
    local output_file="$1"
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    node_red::show_exporting_flows "$output_file"
    
    # Get flows via admin API
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$RESOURCE_PORT/flows" > "$output_file" 2>/dev/null; then
        # Pretty print the JSON
        if jq . "$output_file" > "$output_file.tmp" 2>/dev/null && mv "$output_file.tmp" "$output_file"; then
            # Show summary
            local flow_count
            local node_count
            flow_count=$(jq '[.[] | select(.type == "tab")] | length' "$output_file" 2>/dev/null || echo "0")
            node_count=$(jq 'length' "$output_file" 2>/dev/null || echo "0")
            
            node_red::show_flow_export_success "$output_file" "$flow_count" "$node_count"
            return 0
        else
            log::warning "Flows exported but JSON formatting failed"
            return 0
        fi
    else
        node_red::show_flow_export_error
        return 1
    fi
}

#######################################
# Import flows from a file
#######################################
node_red::import_flow() {
    local flow_file="${1:-$FLOW_FILE}"
    
    node_red::import_flow_file "$flow_file"
}

#######################################
# Import flows from a specific file (internal function)
#######################################
node_red::import_flow_file() {
    local flow_file="$1"
    
    if [[ -z "$flow_file" ]]; then
        log::error "Flow file is required"
        return 1
    fi
    
    if [[ ! -f "$flow_file" ]]; then
        log::error "Flow file not found: $flow_file"
        return 1
    fi
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    node_red::show_importing_flow "$flow_file"
    
    # Validate JSON
    if ! node_red::validate_json "$flow_file"; then
        log::error "Invalid JSON in flow file"
        return 1
    fi
    
    # Import via admin API
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" \
        -X POST -H "Content-Type: application/json" \
        -d "@$flow_file" \
        "http://localhost:$RESOURCE_PORT/flows" >/dev/null 2>&1; then
        
        log::success "Flows imported successfully"
        
        # Reload Node-RED to apply changes
        curl -s --max-time "$NODE_RED_API_TIMEOUT" \
            -X POST "http://localhost:$RESOURCE_PORT/flows" \
            -H "Content-Type: application/json" \
            -H "Node-RED-Deployment-Type: reload" \
            -d "@$flow_file" >/dev/null 2>&1
        
        return 0
    else
        node_red::show_flow_import_error
        return 1
    fi
}

#######################################
# Execute a flow via HTTP endpoint
#######################################
node_red::execute_flow() {
    local endpoint="${1:-$ENDPOINT}"
    local data="${2:-$DATA}"
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    if [[ -z "$endpoint" ]]; then
        log::error "Endpoint is required for flow execution"
        return 1
    fi
    
    local url="http://localhost:$RESOURCE_PORT$endpoint"
    local method="POST"
    local curl_opts="-s --max-time $NODE_RED_API_TIMEOUT"
    
    node_red::show_executing_flow "$url"
    
    # Build curl command
    local curl_cmd="curl $curl_opts -X $method"
    
    if [[ -n "$data" ]]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    curl_cmd="$curl_cmd '$url'"
    
    # Execute
    local response
    local exit_code
    response=$(eval "$curl_cmd" 2>&1)
    exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        node_red::show_flow_execution_success "$response"
        return 0
    else
        node_red::show_flow_execution_error
        [[ -n "$response" ]] && echo "$response"
        return 1
    fi
}

#######################################
# Get flow by ID
#######################################
node_red::get_flow() {
    local flow_id="$1"
    
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID is required"
        return 1
    fi
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Fetching flow: $flow_id"
    
    local response
    response=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$RESOURCE_PORT/flow/$flow_id" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq . 2>/dev/null || echo "$response"
        return 0
    else
        log::error "Failed to fetch flow: $flow_id"
        return 1
    fi
}

#######################################
# Enable/disable a flow
#######################################
node_red::toggle_flow() {
    local flow_id="$1"
    local action="${2:-enable}"  # enable or disable
    
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID is required"
        return 1
    fi
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "${action^}ing flow: $flow_id"
    
    # Get current flow configuration
    local current_flow
    current_flow=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$RESOURCE_PORT/flow/$flow_id" 2>/dev/null)
    
    if [[ -z "$current_flow" ]]; then
        log::error "Flow not found: $flow_id"
        return 1
    fi
    
    # Update the disabled property
    local updated_flow
    if [[ "$action" == "enable" ]]; then
        updated_flow=$(echo "$current_flow" | jq 'del(.disabled)')
    else
        updated_flow=$(echo "$current_flow" | jq '.disabled = true')
    fi
    
    # Update the flow
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" \
        -X PUT -H "Content-Type: application/json" \
        -d "$updated_flow" \
        "http://localhost:$RESOURCE_PORT/flow/$flow_id" >/dev/null 2>&1; then
        
        log::success "Flow ${action}d successfully"
        return 0
    else
        log::error "Failed to ${action} flow"
        return 1
    fi
}

#######################################
# Enable a flow
#######################################
node_red::enable_flow() {
    local flow_id="$1"
    node_red::toggle_flow "$flow_id" "enable"
}

#######################################
# Disable a flow
#######################################
node_red::disable_flow() {
    local flow_id="$1"
    node_red::toggle_flow "$flow_id" "disable"
}

#######################################
# Deploy flows (restart Node-RED runtime)
#######################################
node_red::deploy_flows() {
    local deploy_type="${1:-full}"  # full, nodes, flows
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Deploying flows (type: $deploy_type)..."
    
    if curl -s --max-time "$NODE_RED_API_TIMEOUT" \
        -X POST -H "Content-Type: application/json" \
        -H "Node-RED-Deployment-Type: $deploy_type" \
        "http://localhost:$RESOURCE_PORT/flows" >/dev/null 2>&1; then
        
        log::success "Flows deployed successfully"
        return 0
    else
        log::error "Failed to deploy flows"
        return 1
    fi
}

#######################################
# Get Node-RED runtime information
#######################################
node_red::get_runtime_info() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Fetching Node-RED runtime information..."
    
    local response
    response=$(curl -s --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$RESOURCE_PORT/settings" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "Node-RED Runtime Information:"
        echo "============================"
        echo "$response" | jq -r '
            "Version: \(.version // "unknown")",
            "Editor Theme: \(.editorTheme.theme // "default")",
            "Flow File: \(.flowFile // "flows.json")",
            "User Directory: \(.userDir // "/data")",
            "Logging Level: \(.logging.console.level // "info")",
            "HTTP Timeout: \((.httpRequestTimeout // 120000) / 1000)s"
        ' 2>/dev/null || echo "$response"
        return 0
    else
        log::error "Failed to fetch runtime information"
        return 1
    fi
}

#######################################
# Get Node-RED admin authentication status
#######################################
node_red::get_auth_status() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Checking authentication status..."
    
    # Try to access the admin API without authentication
    local response
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time "$NODE_RED_API_TIMEOUT" "http://localhost:$RESOURCE_PORT/auth/login" 2>/dev/null)
    
    case "$http_code" in
        "200")
            echo "Authentication: Disabled (no authentication required)"
            ;;
        "404")
            echo "Authentication: Disabled (login endpoint not found)"
            ;;
        "401")
            echo "Authentication: Enabled (authentication required)"
            ;;
        *)
            echo "Authentication: Unknown (HTTP $http_code)"
            ;;
    esac
}

#######################################
# Backup flows to a timestamped file
#######################################
node_red::backup_flows() {
    local backup_dir="${1:-$HOME/.node-red-backups}"
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_file="$backup_dir/flows-backup-$timestamp.json"
    
    # Create backup directory
    mkdir -p "$backup_dir"
    
    log::info "Creating flows backup..."
    
    if node_red::export_flows_to_file "$backup_file"; then
        log::success "Flows backed up to: $backup_file"
        return 0
    else
        log::error "Failed to create flows backup"
        return 1
    fi
}

#######################################
# Restore flows from a backup
#######################################
node_red::restore_flows() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        log::error "Backup file is required"
        return 1
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    log::info "Restoring flows from backup: $backup_file"
    
    if node_red::import_flow_file "$backup_file"; then
        # Deploy the restored flows
        node_red::deploy_flows "full"
        log::success "Flows restored from backup"
        return 0
    else
        log::error "Failed to restore flows from backup"
        return 1
    fi
}

#######################################
# Validate all flows for syntax errors
#######################################
node_red::validate_flows() {
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Validating flows..."
    
    # Export flows to temporary file for validation
    local temp_file=$(mktemp)
    trap "rm -f $temp_file" EXIT
    
    if node_red::export_flows_to_file "$temp_file"; then
        # Validate JSON structure
        if node_red::validate_json "$temp_file"; then
            # Check for common issues
            local issues=0
            
            # Check for missing node types (basic validation)
            local missing_types
            missing_types=$(jq -r '.[] | select(.type != "tab" and .type != "subflow") | .type' "$temp_file" 2>/dev/null | sort -u | while read -r node_type; do
                if [[ -z "$node_type" || "$node_type" == "null" ]]; then
                    echo "unknown-type"
                    ((issues++))
                fi
            done)
            
            if [[ -n "$missing_types" ]]; then
                log::warning "Potential issues found in flows:"
                echo "$missing_types" | while read -r issue; do
                    echo "- $issue"
                done
                ((issues++))
            fi
            
            if [[ $issues -eq 0 ]]; then
                log::success "Flow validation passed"
                return 0
            else
                log::warning "Flow validation completed with warnings"
                return 1
            fi
        else
            log::error "Flow validation failed: Invalid JSON"
            return 1
        fi
    else
        log::error "Failed to export flows for validation"
        return 1
    fi
}

#######################################
# Search flows for specific content
#######################################
node_red::search_flows() {
    local search_term="$1"
    
    if [[ -z "$search_term" ]]; then
        log::error "Search term is required"
        return 1
    fi
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    log::info "Searching flows for: $search_term"
    
    # Export flows to temporary file for searching
    local temp_file=$(mktemp)
    trap "rm -f $temp_file" EXIT
    
    if node_red::export_flows_to_file "$temp_file"; then
        # Search in the JSON content
        local results
        results=$(jq -r --arg term "$search_term" '
            .[] | select(
                (.name // .label // "") | test($term; "i") or 
                (.info // "") | test($term; "i") or
                (if .type == "function" then .func else "" end) | test($term; "i")
            ) | 
            "Flow: \(.name // .label // "Unnamed") (ID: \(.id), Type: \(.type))"
        ' "$temp_file" 2>/dev/null)
        
        if [[ -n "$results" ]]; then
            echo "Search Results:"
            echo "==============="
            echo "$results"
            return 0
        else
            log::info "No matches found for: $search_term"
            return 1
        fi
    else
        log::error "Failed to export flows for searching"
        return 1
    fi
}