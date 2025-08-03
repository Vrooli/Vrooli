#!/usr/bin/env bash
set -euo pipefail

# Node-RED Data Injection Adapter
# This script handles injection of flows and nodes into Node-RED
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject flows and configurations into Node-RED flow programming platform"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Source Node-RED configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
    
    # Export configuration if function exists
    if declare -f nodered::export_config >/dev/null 2>&1; then
        nodered::export_config 2>/dev/null || true
    fi
fi

# Default Node-RED API settings
if [[ -z "${NODE_RED_BASE_URL:-}" ]]; then
    NODE_RED_BASE_URL="http://localhost:1880"
fi

# Operation tracking
declare -a NODERED_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
nodered_inject::usage() {
    cat << EOF
Node-RED Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects flows and configurations into Node-RED based on scenario
    configuration. Supports validation, injection, status checks, and rollback.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "flows": [
        {
          "name": "flow-name",
          "file": "path/to/flow.json",
          "deploy": true
        }
      ],
      "nodes": [
        {
          "name": "custom-node",
          "package": "@scope/package-name"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"flows": [{"name": "test-flow", "file": "test-flow.json"}]}'
    
    # Inject flows
    $0 --inject '{"flows": [{"name": "monitor", "file": "monitor.json", "deploy": true}]}'

EOF
}

#######################################
# Check if Node-RED is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
nodered_inject::check_accessibility() {
    if ! system::is_command "curl"; then
        log::error "curl command not available"
        return 1
    fi
    
    # Check if Node-RED is running
    if curl -s --max-time 5 "$NODE_RED_BASE_URL/flows" >/dev/null 2>&1; then
        log::debug "Node-RED is accessible at $NODE_RED_BASE_URL"
        return 0
    fi
    
    log::error "Node-RED is not accessible at $NODE_RED_BASE_URL"
    log::info "Ensure Node-RED is running: ./scripts/resources/automation/node-red/manage.sh --action start"
    return 1
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
nodered_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    NODERED_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Node-RED rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
nodered_inject::execute_rollback() {
    if [[ ${#NODERED_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Node-RED rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Node-RED rollback actions..."
    
    local success_count=0
    local total_count=${#NODERED_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#NODERED_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${NODERED_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Node-RED rollback completed: $success_count/$total_count actions successful"
    NODERED_ROLLBACK_ACTIONS=()
}

#######################################
# Validate flow configuration
# Arguments:
#   $1 - flows configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
nodered_inject::validate_flows() {
    local flows_config="$1"
    
    log::debug "Validating flow configurations..."
    
    # Check if flows is an array
    local flows_type
    flows_type=$(echo "$flows_config" | jq -r 'type')
    
    if [[ "$flows_type" != "array" ]]; then
        log::error "Flows configuration must be an array, got: $flows_type"
        return 1
    fi
    
    # Validate each flow
    local flow_count
    flow_count=$(echo "$flows_config" | jq 'length')
    
    for ((i=0; i<flow_count; i++)); do
        local flow
        flow=$(echo "$flows_config" | jq -c ".[$i]")
        
        # Check required fields
        local name file
        name=$(echo "$flow" | jq -r '.name // empty')
        file=$(echo "$flow" | jq -r '.file // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Flow at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Flow '$name' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists
        local flow_file="$VROOLI_PROJECT_ROOT/$file"
        if [[ ! -f "$flow_file" ]]; then
            log::error "Flow file not found: $flow_file"
            return 1
        fi
        
        # Validate flow file is valid JSON
        if ! jq . "$flow_file" >/dev/null 2>&1; then
            log::error "Flow file contains invalid JSON: $flow_file"
            return 1
        fi
        
        # Check if it's a valid Node-RED flow format (should be an array)
        local flow_type
        flow_type=$(jq -r 'type' "$flow_file")
        if [[ "$flow_type" != "array" ]]; then
            log::error "Flow file must contain a JSON array: $flow_file"
            return 1
        fi
        
        log::debug "Flow '$name' configuration is valid"
    done
    
    log::success "All flow configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
nodered_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Node-RED injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Node-RED injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_flows has_nodes
    has_flows=$(echo "$config" | jq -e '.flows' >/dev/null 2>&1 && echo "true" || echo "false")
    has_nodes=$(echo "$config" | jq -e '.nodes' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_flows" == "false" && "$has_nodes" == "false" ]]; then
        log::error "Node-RED injection configuration must have 'flows' or 'nodes'"
        return 1
    fi
    
    # Validate flows if present
    if [[ "$has_flows" == "true" ]]; then
        local flows
        flows=$(echo "$config" | jq -c '.flows')
        
        if ! nodered_inject::validate_flows "$flows"; then
            return 1
        fi
    fi
    
    # Note: Node installation validation would require npm/package checking
    if [[ "$has_nodes" == "true" ]]; then
        log::warn "Node package injection validation not yet implemented"
    fi
    
    log::success "Node-RED injection configuration is valid"
    return 0
}

#######################################
# Get current flows from Node-RED
# Returns:
#   JSON array of current flows
#######################################
nodered_inject::get_current_flows() {
    local response
    response=$(curl -s "$NODE_RED_BASE_URL/flows" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response"
    else
        echo "[]"
    fi
}

#######################################
# Deploy flows to Node-RED
# Arguments:
#   $1 - flows JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
nodered_inject::deploy_flows() {
    local flows="$1"
    
    log::info "Deploying flows to Node-RED..."
    
    # Deploy flows via API
    local response
    if response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Node-RED-Deployment-Type: full" \
        -d "$flows" \
        "$NODE_RED_BASE_URL/flows" 2>/dev/null); then
        
        log::success "Flows deployed successfully"
        return 0
    else
        log::error "Failed to deploy flows"
        return 1
    fi
}

#######################################
# Import flow into Node-RED
# Arguments:
#   $1 - flow configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
nodered_inject::import_flow() {
    local flow_config="$1"
    
    local name file deploy
    name=$(echo "$flow_config" | jq -r '.name')
    file=$(echo "$flow_config" | jq -r '.file')
    deploy=$(echo "$flow_config" | jq -r '.deploy // true')
    
    log::info "Importing flow: $name"
    
    # Resolve file path
    local flow_file="$VROOLI_PROJECT_ROOT/$file"
    
    # Read flow content
    local new_flow
    new_flow=$(cat "$flow_file")
    
    # Get current flows
    local current_flows
    current_flows=$(nodered_inject::get_current_flows)
    
    # Generate unique IDs for new flow nodes
    # This is important to avoid ID conflicts
    local timestamp
    timestamp=$(date +%s%N)
    
    # Add unique prefix to all node IDs in the new flow
    new_flow=$(echo "$new_flow" | jq --arg prefix "${name}_${timestamp}_" '
        map(
            if .id then
                .id = ($prefix + .id)
            else . end |
            if .z then
                .z = ($prefix + .z)
            else . end |
            if .wires then
                .wires = (.wires | map(map(if . then ($prefix + .) else . end)))
            else . end
        )
    ')
    
    # Merge with existing flows
    local merged_flows
    merged_flows=$(echo "$current_flows" | jq --argjson new "$new_flow" '. + $new')
    
    # Save original flows for rollback
    local original_flows_file="/tmp/nodered_original_flows_$(date +%s).json"
    echo "$current_flows" > "$original_flows_file"
    
    # Add rollback action
    nodered_inject::add_rollback_action \
        "Restore original flows for: $name" \
        "curl -s -X POST -H 'Content-Type: application/json' -d @'$original_flows_file' '$NODE_RED_BASE_URL/flows' >/dev/null 2>&1 && rm -f '$original_flows_file'"
    
    # Deploy merged flows if requested
    if [[ "$deploy" == "true" ]]; then
        if nodered_inject::deploy_flows "$merged_flows"; then
            log::success "Imported and deployed flow: $name"
            return 0
        else
            log::error "Failed to deploy flow: $name"
            return 1
        fi
    else
        # Save flows without deploying
        if response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -H "Node-RED-Deployment-Type: nodes" \
            -d "$merged_flows" \
            "$NODE_RED_BASE_URL/flows" 2>/dev/null); then
            
            log::success "Imported flow (not deployed): $name"
            return 0
        else
            log::error "Failed to import flow: $name"
            return 1
        fi
    fi
}

#######################################
# Inject flows into Node-RED
# Arguments:
#   $1 - flows configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
nodered_inject::inject_flows() {
    local flows_config="$1"
    
    log::info "Injecting flows into Node-RED..."
    
    local flow_count
    flow_count=$(echo "$flows_config" | jq 'length')
    
    if [[ "$flow_count" -eq 0 ]]; then
        log::info "No flows to inject"
        return 0
    fi
    
    local failed_flows=()
    
    for ((i=0; i<flow_count; i++)); do
        local flow
        flow=$(echo "$flows_config" | jq -c ".[$i]")
        
        local flow_name
        flow_name=$(echo "$flow" | jq -r '.name')
        
        if ! nodered_inject::import_flow "$flow"; then
            failed_flows+=("$flow_name")
        fi
    done
    
    if [[ ${#failed_flows[@]} -eq 0 ]]; then
        log::success "All flows injected successfully"
        return 0
    else
        log::error "Failed to inject flows: ${failed_flows[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
nodered_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Node-RED"
    
    # Check Node-RED accessibility
    if ! nodered_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    NODERED_ROLLBACK_ACTIONS=()
    
    # Inject flows if present
    local has_flows
    has_flows=$(echo "$config" | jq -e '.flows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_flows" == "true" ]]; then
        local flows
        flows=$(echo "$config" | jq -c '.flows')
        
        if ! nodered_inject::inject_flows "$flows"; then
            log::error "Failed to inject flows"
            nodered_inject::execute_rollback
            return 1
        fi
    fi
    
    # Note: Node package installation not yet implemented
    local has_nodes
    has_nodes=$(echo "$config" | jq -e '.nodes' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_nodes" == "true" ]]; then
        log::warn "Node package injection not yet implemented for Node-RED"
        log::info "Install nodes manually via: npm install <package-name>"
    fi
    
    log::success "✅ Node-RED data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
nodered_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Node-RED injection status"
    
    # Check Node-RED accessibility
    if ! nodered_inject::check_accessibility; then
        return 1
    fi
    
    # Get current flows
    local current_flows
    current_flows=$(nodered_inject::get_current_flows)
    
    # Check flow status
    local has_flows
    has_flows=$(echo "$config" | jq -e '.flows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_flows" == "true" ]]; then
        local flows
        flows=$(echo "$config" | jq -c '.flows')
        
        log::info "Checking flow status..."
        
        local flow_count
        flow_count=$(echo "$flows" | jq 'length')
        
        for ((i=0; i<flow_count; i++)); do
            local flow
            flow=$(echo "$flows" | jq -c ".[$i]")
            
            local name
            name=$(echo "$flow" | jq -r '.name')
            
            # Check if flow exists in current flows (by looking for nodes with matching prefix)
            if echo "$current_flows" | jq -e --arg name "$name" 'any(.id; contains($name))' >/dev/null 2>&1; then
                log::success "✅ Flow '$name' found"
            else
                log::error "❌ Flow '$name' not found"
            fi
        done
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
nodered_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        nodered_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            nodered_inject::validate_config "$config"
            ;;
        "--inject")
            nodered_inject::inject_data "$config"
            ;;
        "--status")
            nodered_inject::check_status "$config"
            ;;
        "--rollback")
            nodered_inject::execute_rollback
            ;;
        "--help")
            nodered_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            nodered_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        nodered_inject::usage
        exit 1
    fi
    
    nodered_inject::main "$@"
fi