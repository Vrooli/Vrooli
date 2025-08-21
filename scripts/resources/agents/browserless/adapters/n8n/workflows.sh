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

# Source flow control libraries with proper error checking
if [[ -f "${BROWSERLESS_DIR}/lib/workflow/flow-compiler.sh" ]]; then
    source "${BROWSERLESS_DIR}/lib/workflow/flow-compiler.sh"
else
    log::warn "Flow compiler not found - YAML workflows will use fallback mode"
fi

if [[ -f "${BROWSERLESS_DIR}/lib/workflow/flow-parser.sh" ]]; then
    source "${BROWSERLESS_DIR}/lib/workflow/flow-parser.sh"
else
    log::warn "Flow parser not found - YAML workflows will use fallback mode"
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
    
    log::header "🚀 Executing N8n Workflow via Flow Control"
    
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
    
    # Prepare parameters for YAML workflow
    local params_json
    
    # Handle input_data - it might be a string or empty
    local input_json="{}"
    if [[ -n "$input_data" ]]; then
        # Validate it's valid JSON
        if echo "$input_data" | jq '.' >/dev/null 2>&1; then
            input_json="$input_data"
        else
            log::warn "Invalid JSON in input_data, using empty object"
        fi
    fi
    
    params_json=$(jq -n \
        --arg id "$workflow_id" \
        --arg url "$n8n_url" \
        --arg timeout "$timeout" \
        --argjson input "$input_json" \
        '{
            workflow_id: $id,
            n8n_url: $url,
            timeout: ($timeout | tonumber),
            input_data: $input
        }')
    
    log::info "Workflow ID: $workflow_id"
    log::info "N8n URL: $n8n_url"
    log::info "Timeout: ${timeout}ms"
    
    # Execute using YAML workflow with runtime compilation
    local workflow_yaml="${N8N_WORKFLOWS_DIR}/execute-workflow.yaml"
    local result
    
    if [[ -f "$workflow_yaml" ]]; then
        log::info "Using flow control workflow: execute-workflow.yaml"
        
        # Execute via browserless with runtime-compiled workflow
        result=$(browserless::execute_flow_workflow "$workflow_yaml" "$params_json")
        local exit_code=$?
        
        # Check if curl/execution failed completely
        if [[ $exit_code -ne 0 ]]; then
            log::error "❌ Failed to connect to browserless service"
            echo "$result" | jq '.' 2>/dev/null || echo "$result"
            return 1
        fi
        
        # Parse the result to determine actual success
        local is_json=false
        local error_msg=""
        
        # Check if result is valid JSON
        if echo "$result" | jq '.' >/dev/null 2>&1; then
            is_json=true
            # Check for error indicators in JSON response
            error_msg=$(echo "$result" | jq -r '.error // .message // ""' 2>/dev/null)
            local success_flag=$(echo "$result" | jq -r '.success // ""' 2>/dev/null)
            
            if [[ "$success_flag" == "false" ]] || [[ -n "$error_msg" ]]; then
                log::error "❌ Workflow execution failed"
                if [[ -n "$error_msg" ]]; then
                    log::error "Error: $error_msg"
                fi
                echo "$result" | jq '.'
                return 1
            elif [[ "$success_flag" == "true" ]]; then
                log::success "✅ Workflow executed successfully"
                echo "$result" | jq '.'
                return 0
            fi
        fi
        
        # Handle non-JSON responses (likely error pages or "Not Found")
        if [[ "$result" == "Not Found" ]] || [[ "$result" == *"404"* ]] || [[ "$result" == *"not found"* ]]; then
            log::error "❌ Workflow not found: $workflow_id"
            log::info "Possible causes:"
            log::info "  • Workflow ID '$workflow_id' doesn't exist in N8n"
            log::info "  • N8n is not running at $n8n_url"
            log::info "  • Browserless couldn't connect to N8n"
            log::info ""
            log::info "To debug:"
            log::info "  1. Check if N8n is running: curl $n8n_url"
            log::info "  2. List available workflows: $0 list"
            log::info "  3. Verify workflow ID in N8n UI"
            return 1
        elif [[ "$result" == *"<html"* ]] || [[ "$result" == *"<!DOCTYPE"* ]]; then
            log::error "❌ Received HTML response instead of workflow result"
            log::info "This usually means:"
            log::info "  • N8n returned an error page"
            log::info "  • Authentication is required"
            log::info "  • The workflow URL is incorrect"
            log::debug "HTML response (first 200 chars): ${result:0:200}..."
            return 1
        elif [[ -z "$result" ]]; then
            log::error "❌ Empty response from browserless"
            log::info "Possible causes:"
            log::info "  • Browserless container crashed"
            log::info "  • Workflow compilation failed"
            log::info "  • Network timeout"
            return 1
        else
            # Unknown response format
            log::warn "⚠️ Workflow execution completed with unknown result format"
            log::info "Response: $result"
            return 1
        fi
    else
        log::error "Workflow definition not found: $workflow_yaml"
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
    
    # Execute via browserless API using direct JavaScript approach
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    local js_file="${N8N_WORKFLOWS_DIR}/list-workflows-direct.json"
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"code\": $(cat "$js_file" | jq -r .code | jq -Rs .),
            \"context\": {
                \"outputDir\": \"/tmp\",
                \"params\": {},
                \"env\": $env_context
            }
        }" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    # Extract and return workflows
    if [[ -n "$response" ]]; then
        # Check if response is valid JSON and extract workflows
        if echo "$response" | jq '.' >/dev/null 2>&1; then
            local workflow_data=$(echo "$response" | jq '.workflowData' 2>/dev/null)
            if [[ "$workflow_data" != "null" ]]; then
                local workflows_array=$(echo "$workflow_data" | jq '.workflows // []' 2>/dev/null)
                
                # Filter inactive workflows if requested
                if [[ "$include_inactive" == "false" ]]; then
                    workflows_array=$(echo "$workflows_array" | jq '[.[] | select(.active == true)]' 2>/dev/null)
                fi
                echo "$workflows_array"
            else
                echo "[]"
            fi
        else
            echo "[]"
        fi
    else
        echo "[]"
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
    
    # Execute export workflow
    local workflow_yaml="${N8N_WORKFLOWS_DIR}/export-workflow.yaml"
    
    if [[ -f "$workflow_yaml" ]]; then
        local result
        result=$(browserless::execute_flow_workflow "$workflow_yaml" "$params_json")
        
        if [[ $? -eq 0 ]]; then
            log::success "✅ Workflow exported to $output_file"
            return 0
        else
            log::error "❌ Failed to export workflow"
            return 1
        fi
    else
        log::error "Export workflow definition not found"
        return 1
    fi
}

#######################################
# Execute flow control workflow via browserless
# Internal helper function
# Arguments:
#   $1 - Workflow YAML file
#   $2 - Parameters JSON
# Returns:
#   Execution result JSON
#######################################
browserless::execute_flow_workflow() {
    local workflow_yaml="${1:?Workflow YAML required}"
    local params_json="${2:-{}}"
    
    # Ensure output directory exists
    local output_dir="${BROWSERLESS_DATA_DIR}/n8n-executions/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$output_dir"
    
    # Runtime compile YAML workflow to JavaScript
    local compiled_js
    log::debug "Checking if flow control functions are available..."
    
    if declare -f workflow::compile_with_flow_control >/dev/null 2>&1; then
        log::debug "✓ workflow::compile_with_flow_control function found"
        if declare -f workflow::parse_with_flow_control >/dev/null 2>&1; then
            log::debug "✓ workflow::parse_with_flow_control function found"
            
            # Parse and compile workflow at runtime
            local workflow_json
            log::debug "Parsing YAML workflow: $workflow_yaml"
            workflow_json=$(workflow::parse_with_flow_control "$workflow_yaml" 2>/dev/null)
            
            if [[ $? -eq 0 ]]; then
                log::debug "✓ YAML parsing successful ($(echo "$workflow_json" | wc -c) chars)"
                echo "[WORKFLOWS-DEBUG] First 200 chars of parsed JSON:" >&2
                echo "$workflow_json" | head -c 200 >&2
                echo "" >&2
                echo "[WORKFLOWS-DEBUG] Testing JSON validity with jq..." >&2
                if echo "$workflow_json" | jq '.' >/dev/null 2>&1; then
                    echo "[WORKFLOWS-DEBUG] ✓ JSON is valid" >&2
                else
                    echo "[WORKFLOWS-DEBUG] ✗ JSON is invalid!" >&2
                    echo "$workflow_json" | head -c 500 >&2
                    echo "" >&2
                fi
                
                log::debug "Compiling workflow to JavaScript..."
                compiled_js=$(workflow::compile_with_flow_control "$workflow_json" 2>&1)
                if [[ $? -eq 0 ]]; then
                    log::debug "✓ Workflow compilation successful"
                else
                    log::error "✗ Workflow compilation failed: $compiled_js"
                fi
            else
                log::error "✗ YAML parsing failed: $workflow_json"
            fi
        else
            log::error "✗ workflow::parse_with_flow_control function missing"
        fi
    else
        log::error "✗ workflow::compile_with_flow_control function missing"
    fi
    
    # Fallback to direct execution if compilation fails
    if [[ -z "$compiled_js" ]]; then
        log::debug "Flow control compiler not available, using fallback"
        compiled_js=$(cat <<'EOF'
export default async ({ page, context }) => {
    // Fallback execution - flow control compiler not available
    return {
        success: false,
        error: "Flow control compiler not available - YAML workflows require flow control system",
        timestamp: new Date().toISOString()
    };
};
EOF
        )
    fi
    
    # Create a wrapper that injects parameters from context
    # This avoids sed issues with special characters in JSON
    local wrapped_js="
export default async ({ page, context }) => {
    // Inject parameters and context into the workflow
    const params = context.params || {};
    const outputDir = context.outputDir || '/tmp';
    const env = context.env || {};
    
    // Runtime-compiled workflow function
    const workflowFn = $compiled_js;
    
    // Execute with injected context including environment
    return await workflowFn({ page, context: { ...context, params, outputDir, env } });
};
"
    
    # Get N8n credentials from secrets system
    local n8n_email n8n_password
    if command -v secrets::get_key >/dev/null 2>&1; then
        n8n_email=$(secrets::get_key "N8N_EMAIL" 2>/dev/null || echo "")
        n8n_password=$(secrets::get_key "N8N_PASSWORD" 2>/dev/null || echo "")
    fi
    
    # Fallback to environment variables if secrets not available
    n8n_email="${n8n_email:-${N8N_EMAIL:-}}"
    n8n_password="${n8n_password:-${N8N_PASSWORD:-}}"
    
    # Create environment context for YAML workflow
    local env_context
    env_context=$(jq -n \
        --arg email "$n8n_email" \
        --arg password "$n8n_password" \
        '{
            N8N_EMAIL: $email,
            N8N_PASSWORD: $password
        }')
    
    # Execute via browserless API
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"code\": $(echo "$wrapped_js" | jq -Rs .),
            \"context\": {
                \"outputDir\": \"$output_dir\",
                \"params\": $params_json,
                \"env\": $env_context
            }
        }" \
        "http://localhost:${BROWSERLESS_PORT}/function" 2>/dev/null)
    
    echo "$response"
}

# Export functions for use by adapter
export -f browserless::execute_n8n_workflow
export -f n8n::list_workflows
export -f n8n::export_workflow
export -f n8n::init_workflows
export -f browserless::execute_flow_workflow