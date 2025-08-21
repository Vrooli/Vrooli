#!/usr/bin/env bash

#######################################
# N8n Workflow Injection for Browserless
# Compiles N8n workflow YAML files and injects them as browserless functions
#######################################

set -euo pipefail

# Get script directory
N8N_INJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROWSERLESS_DIR="$(dirname "$N8N_INJECT_DIR")"

# Source required components
source "${BROWSERLESS_DIR}/lib/common.sh"
source "${N8N_INJECT_DIR}/workflow/parser.sh"
source "${N8N_INJECT_DIR}/workflow/actions.sh"
source "${N8N_INJECT_DIR}/workflow/compiler.sh"
source "${N8N_INJECT_DIR}/workflow/debug.sh"
source "${N8N_INJECT_DIR}/workflow/flow-parser.sh"
source "${N8N_INJECT_DIR}/workflow/flow-actions.sh"
source "${N8N_INJECT_DIR}/workflow/flow-compiler.sh"

#######################################
# Inject N8n workflow into Browserless
# Arguments:
#   $1 - N8n workflow YAML file path
#   $2 - N8n instance URL (optional, defaults to localhost:5678)
# Returns:
#   0 on success, 1 on failure
#######################################
browserless::inject_n8n_workflow() {
    local workflow_file="${1:?N8n workflow YAML file required}"
    local n8n_url="${2:-http://localhost:5678}"
    
    log::header "🚀 Injecting N8n Workflow into Browserless"
    
    # Validate file exists
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    # Verify browserless is running
    if ! is_running; then
        log::error "Browserless container not running - start it first"
        log::info "Run: resource-browserless start"
        return 1
    fi
    
    # Parse and compile with flow control
    log::info "📄 Parsing workflow: $workflow_file"
    local workflow_json
    workflow_json=$(workflow::parse_with_flow_control "$workflow_file")
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to parse workflow"
        return 1
    fi
    
    # Extract workflow metadata
    local workflow_name
    workflow_name=$(echo "$workflow_json" | jq -r '.workflow.name // "n8n-workflow"')
    
    local workflow_desc
    workflow_desc=$(echo "$workflow_json" | jq -r '.workflow.description // "N8n workflow execution"')
    
    log::info "📦 Workflow: $workflow_name"
    log::info "📝 Description: $workflow_desc"
    
    # Compile to JavaScript with flow control
    log::info "🔧 Compiling to JavaScript state machine..."
    local compiled_js
    compiled_js=$(workflow::compile_with_flow_control "$workflow_json")
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to compile workflow"
        return 1
    fi
    
    # Check if flow control is enabled
    local flow_enabled
    flow_enabled=$(echo "$workflow_json" | jq -r '.flow_control.enabled // false')
    
    local execution_model="linear"
    if [[ "$flow_enabled" == "true" ]]; then
        execution_model="state_machine"
        log::info "✨ Flow control enabled - using state machine execution"
    fi
    
    # Create injection directory
    local injection_dir="${BROWSERLESS_DATA_DIR}/n8n-workflows/${workflow_name}"
    mkdir -p "${injection_dir}/executions"
    
    # Store workflow files
    echo "$workflow_json" > "${injection_dir}/workflow.json"
    echo "$compiled_js" > "${injection_dir}/compiled.js"
    
    # Create execution wrapper that includes N8n connection
    cat > "${injection_dir}/wrapper.js" << EOF
// N8n Workflow Execution Wrapper
// Workflow: $workflow_name
// N8n URL: $n8n_url

module.exports = async ({ page, context }) => {
    // Set N8n connection parameters
    context.n8nUrl = '$n8n_url';
    context.workflowName = '$workflow_name';
    context.executionModel = '$execution_model';
    context.outputDir = '/workspace/n8n-workflows/$workflow_name/executions';
    
    // Initialize execution context
    const params = {
        workflow_url: context.n8nUrl + '/workflow',
        credentials: {
            email: process.env.N8N_EMAIL || '',
            password: process.env.N8N_PASSWORD || ''
        },
        execution_timeout: 60000
    };
    
    // Import and execute compiled workflow
    const workflow = $compiled_js
    
    try {
        const result = await workflow({ page, params, context });
        
        // Store execution results
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
        const resultFile = \`\${context.outputDir}/execution-\${timestamp}.json\`;
        require('fs').writeFileSync(resultFile, JSON.stringify(result, null, 2));
        
        return result;
    } catch (error) {
        console.error('Workflow execution failed:', error);
        return {
            success: false,
            workflow: '$workflow_name',
            error: error.message,
            timestamp: new Date().toISOString()
        };
    }
};
EOF
    
    # Create injection manifest
    local manifest
    manifest=$(jq -n \
        --arg name "$workflow_name" \
        --arg desc "$workflow_desc" \
        --arg model "$execution_model" \
        --arg n8n "$n8n_url" \
        '{
            metadata: {
                name: $name,
                description: $desc,
                type: "n8n_workflow",
                execution_model: $model,
                n8n_url: $n8n,
                created: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            },
            files: {
                workflow: "workflow.json",
                compiled: "compiled.js",
                wrapper: "wrapper.js"
            },
            execution: {
                entry: "wrapper.js",
                persistent_session: true,
                flow_control: ($model == "state_machine")
            }
        }')
    
    echo "$manifest" > "${injection_dir}/manifest.json"
    
    # Register with browserless
    local functions_index="${BROWSERLESS_DATA_DIR}/functions.json"
    
    # Initialize functions index if needed
    if [[ ! -f "$functions_index" ]]; then
        echo '{"functions": {}}' > "$functions_index"
    fi
    
    # Add workflow to functions index
    local updated_index
    updated_index=$(jq \
        --arg name "$workflow_name" \
        --arg path "${injection_dir}/wrapper.js" \
        --arg desc "$workflow_desc" \
        '.functions[$name] = {
            path: $path,
            description: $desc,
            type: "n8n_workflow"
        }' "$functions_index")
    
    echo "$updated_index" > "$functions_index"
    
    log::success "✅ N8n workflow injected successfully!"
    log::info ""
    log::info "📋 Workflow Details:"
    log::info "   Name: $workflow_name"
    log::info "   Model: $execution_model"
    log::info "   Location: $injection_dir"
    log::info ""
    log::info "🚀 Execute with:"
    log::info "   resource-browserless execute $workflow_name"
    log::info ""
    log::info "🔧 Set credentials (if needed):"
    log::info "   export N8N_EMAIL='your-email@example.com'"
    log::info "   export N8N_PASSWORD='your-password'"
    
    return 0
}

#######################################
# List injected N8n workflows
#######################################
browserless::list_n8n_workflows() {
    local workflows_dir="${BROWSERLESS_DATA_DIR}/n8n-workflows"
    
    log::header "📋 Injected N8n Workflows"
    
    if [[ ! -d "$workflows_dir" ]]; then
        log::info "No N8n workflows found"
        return 0
    fi
    
    local count=0
    for workflow_dir in "$workflows_dir"/*; do
        if [[ -d "$workflow_dir" ]]; then
            local workflow_name=$(basename "$workflow_dir")
            
            if [[ -f "$workflow_dir/manifest.json" ]]; then
                local desc=$(jq -r '.metadata.description' "$workflow_dir/manifest.json")
                local model=$(jq -r '.metadata.execution_model' "$workflow_dir/manifest.json")
                local created=$(jq -r '.metadata.created' "$workflow_dir/manifest.json")
                
                echo ""
                echo "  📦 $workflow_name"
                echo "     Description: $desc"
                echo "     Model: $model"
                echo "     Created: $created"
                count=$((count + 1))
            fi
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        log::info "No N8n workflows found"
    else
        echo ""
        log::info "Total workflows: $count"
    fi
}

#######################################
# Execute injected N8n workflow
# Arguments:
#   $1 - Workflow name
# Returns:
#   0 on success, 1 on failure
#######################################
browserless::execute_n8n_workflow() {
    local workflow_name="${1:?Workflow name required}"
    local workflow_dir="${BROWSERLESS_DATA_DIR}/n8n-workflows/${workflow_name}"
    
    if [[ ! -d "$workflow_dir" ]]; then
        log::error "Workflow not found: $workflow_name"
        log::info "Use 'resource-browserless n8n list' to see available workflows"
        return 1
    fi
    
    if [[ ! -f "$workflow_dir/wrapper.js" ]]; then
        log::error "Workflow wrapper not found"
        return 1
    fi
    
    log::header "🚀 Executing N8n Workflow: $workflow_name"
    
    # Execute via browserless API
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"code\": \"$(cat "$workflow_dir/wrapper.js" | sed 's/"/\\"/g' | tr '\n' ' ')\",
            \"context\": {}
        }" \
        "http://localhost:${BROWSERLESS_PORT}/function")
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to execute workflow"
        return 1
    fi
    
    # Parse response
    local success=$(echo "$response" | jq -r '.success // false')
    
    if [[ "$success" == "true" ]]; then
        log::success "✅ Workflow executed successfully"
        
        # Show execution results
        echo ""
        echo "$response" | jq '.'
    else
        log::error "❌ Workflow execution failed"
        
        local error=$(echo "$response" | jq -r '.error // "Unknown error"')
        log::error "Error: $error"
        
        # Show full response for debugging
        echo ""
        echo "$response" | jq '.'
        
        return 1
    fi
}

# Export functions
export -f browserless::inject_n8n_workflow
export -f browserless::list_n8n_workflows
export -f browserless::execute_n8n_workflow