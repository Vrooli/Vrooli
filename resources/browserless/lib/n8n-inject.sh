#!/usr/bin/env bash

#######################################
# N8n Workflow Injection for Browserless
# Compiles N8n workflow YAML files and injects them as browserless functions
#######################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_INJECT_DIR="${APP_ROOT}/resources/browserless/lib"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source required components
source "${BROWSERLESS_DIR}/lib/common.sh"
source "${BROWSERLESS_DIR}/lib/workflow/interpreter.sh"
source "${BROWSERLESS_DIR}/lib/browser-ops.sh"
source "${BROWSERLESS_DIR}/lib/session-manager.sh"

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
    
    # N8n workflow injection has been replaced by atomic operations
    log::error "N8n workflow injection is deprecated and replaced by atomic operations"
    log::info "Use direct workflow execution instead: resource-browserless for n8n execute-workflow <id>"
    log::info "Atomic operations provide better debugging and maintainability"
    return 1
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