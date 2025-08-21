#!/usr/bin/env bash

#######################################
# N8N Workflow Integration using Flow Control
# 
# This module provides N8N workflow operations using
# the new YAML-based flow control system instead of
# manual JavaScript generation.
#######################################

# Get script directory
WORKFLOWS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
N8N_ADAPTER_DIR="$WORKFLOWS_DIR"
BROWSERLESS_DIR="$(dirname "$(dirname "$N8N_ADAPTER_DIR")")"

# Source required libraries
source "${BROWSERLESS_DIR}/lib/common.sh"

# Source atomic operations libraries
if [[ -f "${BROWSERLESS_DIR}/lib/browser-ops.sh" ]]; then
    source "${BROWSERLESS_DIR}/lib/browser-ops.sh"
else
    log::warn "Browser operations library not found - using fallback mode"
fi

if [[ -f "${BROWSERLESS_DIR}/lib/session-manager.sh" ]]; then
    source "${BROWSERLESS_DIR}/lib/session-manager.sh"
else
    log::warn "Session manager library not found - using fallback mode"
fi

# Source secrets system for N8n authentication
SECRETS_LIB="$(dirname "$(dirname "$(dirname "$(dirname "$BROWSERLESS_DIR")")")")/scripts/lib/service/secrets.sh"
if [[ -f "$SECRETS_LIB" ]]; then
    source "$SECRETS_LIB"
fi

# Ensure workflow YAML directory exists
N8N_WORKFLOWS_DIR="${N8N_ADAPTER_DIR}/workflows"
mkdir -p "$N8N_WORKFLOWS_DIR"

#######################################
# Initialize N8n workflow system
# Compiles YAML workflows on first use
#######################################
n8n::init_workflows() {
    local workflows_compiled="${N8N_WORKFLOWS_DIR}/.compiled"
    
    # Skip if already compiled
    if [[ -f "$workflows_compiled" ]]; then
        return 0
    fi
    
    log::info "Initializing N8n workflow definitions..."
    
    # YAML workflows are used directly at runtime - no pre-compilation needed
    # The flow control system compiles YAML to JavaScript on-demand for better flexibility
    
    touch "$workflows_compiled"
    log::info "N8n workflows initialized"
}

#######################################
# Execute N8n workflow using flow control
# Arguments:
#   $1 - Workflow ID or URL
#   $2 - N8n base URL (optional)
#   $3 - Timeout (optional)
#   $4 - Input data (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
browserless::execute_n8n_workflow() {
    local workflow_id="${1:?Workflow ID required}"
    local n8n_url="${2:-http://localhost:5678}"
    local timeout="${3:-60000}"
    local input_data="${4:-}"
    
    log::header "🚀 Executing N8n Workflow via Atomic Operations"
    
    # Initialize workflows if needed
    n8n::init_workflows
    
    # Check browserless health
    if ! docker ps --format "{{.Names}}" | grep -q "^browserless$"; then
        log::error "Browserless container is not running"
        return 1
    fi
    
    # Check if browserless API is responding
    if ! curl -s -f "http://localhost:${BROWSERLESS_PORT}/pressure" > /dev/null 2>&1; then
        log::error "Browserless API is not responding"
        return 1
    fi
    
    log::info "Workflow ID: $workflow_id"
    log::info "N8n URL: $n8n_url"
    log::info "Timeout: ${timeout}ms"
    
    # Get N8n credentials from environment or secrets
    local n8n_email n8n_password
    if command -v secrets::get_key >/dev/null 2>&1; then
        n8n_email=$(secrets::get_key "N8N_EMAIL" 2>/dev/null || echo "")
        n8n_password=$(secrets::get_key "N8N_PASSWORD" 2>/dev/null || echo "")
    fi
    n8n_email="${n8n_email:-${N8N_EMAIL:-}}"
    n8n_password="${n8n_password:-${N8N_PASSWORD:-}}"
    
    # Execute using atomic operations
    local execute_script="${N8N_ADAPTER_DIR}/execute-atomic.sh"
    local result
    
    if [[ -f "$execute_script" ]]; then
        log::info "Using atomic operations for workflow execution"
        
        # Build command arguments  
        local cmd_args=("--id" "$workflow_id" "--url" "$n8n_url" "--timeout" "$((timeout/1000))")
        
        if [[ -n "$n8n_email" ]]; then
            cmd_args+=("--email" "$n8n_email")
        fi
        
        if [[ -n "$n8n_password" ]]; then
            cmd_args+=("--password" "$n8n_password")
        fi
        
        # Execute the workflow
        if "$execute_script" "${cmd_args[@]}"; then
            result='{"success": true, "message": "Workflow executed successfully"}'
            log::success "✅ Workflow executed successfully"
            echo "$result" | jq '.'
            return 0
        else
            result='{"success": false, "message": "Workflow execution failed"}'
            log::error "❌ Workflow execution failed"
            echo "$result" | jq '.'
            return 1
        fi
    else
        log::error "Atomic execution script not found: $execute_script"
        log::info "Please ensure the browserless resource is properly installed"
        return 1
    fi
}

#######################################
# List N8n workflows using flow control
# Arguments:
#   $1 - N8n URL (optional)
#   $2 - Include inactive (optional)
# Returns:
#   JSON array of workflows
#######################################
n8n::list_workflows() {
    local n8n_url="${1:-http://localhost:5678}"
    local include_inactive="${2:-true}"
    
    log::header "📋 Listing N8n Workflows"
    log::info "Fetching workflows from $n8n_url..."
    
    # Get N8n credentials from secrets system
    local n8n_email n8n_password
    if command -v secrets::get_key >/dev/null 2>&1; then
        n8n_email=$(secrets::get_key "N8N_EMAIL" 2>/dev/null || echo "")
        n8n_password=$(secrets::get_key "N8N_PASSWORD" 2>/dev/null || echo "")
    fi
    
    # Create environment context
    local env_context
    env_context=$(jq -n \
        --arg email "$n8n_email" \
        --arg password "$n8n_password" \
        '{
            N8N_EMAIL: $email,
            N8N_PASSWORD: $password
        }')
    
    # Create screenshots directory
    local screenshots_dir="${HOME}/.vrooli/browserless/screenshots/n8n/list-workflows/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$screenshots_dir"
    
    # Start browser session
    local session_id="n8n-list-$(date +%s)"
    
    # Use a single combined operation for consistent results
    log::info "Taking screenshot of workflows page..."
    local nav_screenshot="${screenshots_dir}/01-navigate.png"
    log::debug "About to call browser::navigate_and_screenshot"
    
    local nav_result
    nav_result=$(browser::navigate_and_screenshot "$n8n_url/workflows" "$nav_screenshot")
    log::debug "Got navigation result"
    
    # Extract JSON from mixed output using awk
    local nav_json success current_url
    nav_json=$(echo "$nav_result" | awk '/^{/,/^}/')
    success=$(echo "$nav_json" | jq -r '.success // false' 2>/dev/null)
    
    if [[ "$success" != "true" ]]; then
        log::error "Failed to navigate to workflows page" 
        log::error "Result was: $nav_result"
        return 1
    fi
    
    log::info "Navigation screenshot: $nav_screenshot"
    
    # Check if login is needed from the URL in the result
    current_url=$(echo "$nav_json" | jq -r '.url // ""' 2>/dev/null)
    log::debug "Current URL: $current_url"
    
    if [[ "$current_url" =~ signin ]]; then
        log::info "Login required - showing login page in screenshot"
        log::info "To implement full login: set N8N_EMAIL and N8N_PASSWORD environment variables"
        
        # Return empty array since we can't login without credentials
        echo '[]'
        return 0
    else
        log::info "Already authenticated - would extract workflows from page"
        # For now, return empty array - full parsing could be implemented here
        echo '[]'
        return 0
    fi
}

#######################################
# Export N8n workflow using flow control
# Arguments:
#   $1 - Workflow ID
#   $2 - N8n URL (optional)
#   $3 - Output file (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
n8n::export_workflow() {
    local workflow_id="${1:?Workflow ID required}"
    local n8n_url="${2:-http://localhost:5678}"
    local output_file="${3:-${BROWSERLESS_DATA_DIR}/exports/workflow-${workflow_id}.json}"
    
    log::header "📥 Exporting N8n Workflow"
    
    # Initialize workflows
    n8n::init_workflows
    
    # Create export directory
    mkdir -p "$(dirname "$output_file")"
    
    # Prepare parameters
    local params_json
    params_json=$(jq -n \
        --arg id "$workflow_id" \
        --arg url "$n8n_url" \
        --arg output "$output_file" \
        '{
            workflow_id: $id,
            n8n_url: $url,
            output_file: $output
        }')
    
    log::info "Exporting workflow: $workflow_id"
    log::info "Output: $output_file"
    
    # TODO: Implement export using atomic operations
    log::error "Export functionality is being migrated to atomic operations"
    log::info "This feature will be available soon"
    return 1
}

# Export functions for use by adapter
export -f browserless::execute_n8n_workflow
export -f n8n::list_workflows
export -f n8n::export_workflow
export -f n8n::init_workflows