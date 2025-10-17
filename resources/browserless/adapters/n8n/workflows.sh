#!/usr/bin/env bash

#######################################
# N8N Workflow Integration using Flow Control
# 
# This module provides N8N workflow operations using
# the new YAML-based flow control system instead of
# manual JavaScript generation.
#######################################

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
WORKFLOWS_DIR="${APP_ROOT}/resources/browserless/adapters/n8n"
N8N_ADAPTER_DIR="$WORKFLOWS_DIR"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

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
SECRETS_LIB="${BROWSERLESS_DIR%/*/*/*}/scripts/lib/service/secrets.sh"
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
    
    # Execute using YAML workflow
    local workflow_file="${N8N_ADAPTER_DIR}/workflows/execute-workflow.yaml"
    local result
    
    if [[ -f "$workflow_file" ]]; then
        log::info "Using YAML workflow for execution"
        
        # Set environment variables for the workflow
        export N8N_EMAIL="$n8n_email"
        export N8N_PASSWORD="$n8n_password"
        
        # Execute the YAML workflow using the enhanced interpreter
        local workflow_result
        if workflow_result=$("${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" \
            "$workflow_file" \
            --param "workflow_id=$workflow_id" \
            --param "n8n_url=$n8n_url" \
            --param "timeout=$timeout" \
            --param "input_data=${input_data:-{}}" \
            --session "n8n_exec_$(date +%s)" 2>&1); then
            
            result='{"success": true, "message": "Workflow executed successfully"}'
            log::success "✅ Workflow executed successfully"
            echo "$result" | jq '.'
            return 0
        else
            result='{"success": false, "message": "Workflow execution failed"}'
            log::error "❌ Workflow execution failed"
            log::debug "Workflow output: $workflow_result"
            echo "$result" | jq '.'
            return 1
        fi
    else
        log::error "YAML workflow not found: $workflow_file"
        log::info "Please ensure the browserless resource is properly installed"
        return 1
    fi
}

#######################################
# List N8n workflows using YAML workflow
# Arguments:
#   $1 - N8n URL (optional)
#   $2 - Include inactive (optional)
# Returns:
#   JSON array of workflows
#######################################
n8n::list_workflows() {
    local n8n_url="${1:-http://localhost:5678}"
    local include_inactive="${2:-true}"
    
    log::header "📋 Listing N8n Workflows via YAML Workflow"
    log::info "Executing list-workflows.yaml with interpreter..."
    
    # Execute the working YAML workflow with proper screenshot handling
    local workflow_result
    if workflow_result=$("${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" \
        "${N8N_ADAPTER_DIR}/workflows/list-workflows-working.yaml" \
        --param "n8n_url=$n8n_url" \
        --param "include_inactive=$include_inactive" \
        --session "n8n_list_$(date +%s)" 2>&1); then
        
        log::success "YAML workflow completed successfully"
        
        # Extract debug output directory from workflow result
        local debug_dir
        debug_dir=$(echo "$workflow_result" | grep "Debug output directory:" | sed 's/.*Debug output directory: //')
        
        if [[ -n "$debug_dir" ]] && [[ -d "$debug_dir" ]]; then
            log::info "🔍 DEBUG: Debug files available in $debug_dir"
            
            # List debug files with full paths
            local debug_files
            debug_files=$(find "$debug_dir" -type f 2>/dev/null | sort || true)
            if [[ -n "$debug_files" ]]; then
                log::info "📁 DEBUG FILES (click to open):"
                echo "$debug_files" | while read -r file; do
                    local filesize=$(stat -c%s "$file" 2>/dev/null || echo "0")
                    local filetype=""
                    case "$file" in
                        *.png) filetype="🖼️" ;;
                        *.json) filetype="📄" ;;
                        *.html) filetype="🌐" ;;
                        *) filetype="📄" ;;
                    esac
                    log::info "  $filetype $file (${filesize} bytes)"
                done
            fi
            
            # Try to read final results from debug output
            local final_results_file="$debug_dir/final-results.json"
            if [[ -f "$final_results_file" ]]; then
                local json_content
                json_content=$(cat "$final_results_file" 2>/dev/null || echo '{}')
                log::info "📄 Reading results from: $final_results_file"
                
                # Show workflow count if available
                local workflow_count
                workflow_count=$(echo "$json_content" | jq -r '.workflows | length' 2>/dev/null || echo "unknown")
                log::info "🔢 Found $workflow_count workflows"
                
                # Extract just the workflows array
                local workflows_json
                workflows_json=$(echo "$json_content" | jq '.workflows // []' 2>/dev/null || echo '[]')
                echo "$workflows_json"
                return 0
            fi
        fi
        
        # Fallback: try to extract JSON result from workflow output
        local json_result
        json_result=$(echo "$workflow_result" | grep -o '{.*}' | tail -1 2>/dev/null || echo '[]')
        
        # Validate the JSON result
        if echo "$json_result" | jq . >/dev/null 2>&1; then
            echo "$json_result"
        else
            log::warn "Could not extract valid JSON from workflow result, returning empty array"
            log::info "📝 Full workflow output:"
            echo "$workflow_result" | head -20
            echo '[]'
        fi
        
        return 0
    else
        log::error "YAML workflow execution failed"
        log::debug "Workflow output: $workflow_result"
        echo '[]'
        return 1
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
    mkdir -p "${output_file%/*}"
    
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