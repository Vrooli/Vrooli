#!/usr/bin/env bash
# n8n API Management Functions
# Workflow execution, API key management, and REST API interactions

#######################################
# Execute workflow via API
#######################################
n8n::execute() {
    local workflow_id="${WORKFLOW_ID:-}"
    
    log::header "🚀 n8n Workflow Execution"
    
    # Check if workflow ID is provided
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo ""
        echo "Usage: $0 --action execute --workflow-id YOUR_WORKFLOW_ID"
        echo ""
        echo "To find workflow IDs:"
        echo "  docker exec $N8N_CONTAINER_NAME n8n list:workflow"
        return 1
    fi
    
    # Check for API key - first from env, then from config
    local api_key="${N8N_API_KEY:-}"
    
    # If not in env, try to load from resources config
    if [[ -z "$api_key" ]]; then
        local config_file="${HOME}/.vrooli/service.json"
        if [[ -f "$config_file" ]]; then
            api_key=$(jq -r '.services.automation.n8n.apiKey // empty' "$config_file" 2>/dev/null)
        fi
    fi
    
    if [[ -z "$api_key" ]]; then
        log::warn "No API key found"
        echo ""
        n8n::show_api_setup_instructions
        return 1
    fi
    
    # First, get workflow details to check if it has a webhook
    log::info "Fetching workflow details: $workflow_id"
    
    local workflow_details
    workflow_details=$(curl -s -H "X-N8N-API-KEY: $api_key" \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" 2>&1)
    
    # Check if workflow exists
    if echo "$workflow_details" | grep -q '"message".*not found'; then
        log::error "Workflow not found"
        echo ""
        echo "Available workflows:"
        docker exec "$N8N_CONTAINER_NAME" n8n list:workflow
        return 1
    fi
    
    # Check if workflow has a webhook node
    local webhook_path
    webhook_path=$(echo "$workflow_details" | jq -r '.nodes[]? | select(.type == "n8n-nodes-base.webhook") | .parameters.path // empty' 2>/dev/null | head -1)
    
    if [[ -n "$webhook_path" ]]; then
        # Workflow has webhook - check if it's active
        local is_active
        is_active=$(echo "$workflow_details" | jq -r '.active' 2>/dev/null)
        
        if [[ "$is_active" != "true" ]]; then
            log::warn "Webhook workflow is not active. Activating it..."
            
            # Activate the workflow
            local activate_response
            activate_response=$(curl -s -X POST \
                -H "X-N8N-API-KEY: $api_key" \
                -H "Content-Type: application/json" \
                -d '{}' \
                "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}/activate" 2>&1)
            
            if echo "$activate_response" | grep -q '"active":true'; then
                log::success "✅ Workflow activated"
            else
                log::error "Failed to activate workflow"
                echo "Response: $activate_response"
                return 1
            fi
        fi
        
        # Execute via webhook
        log::info "Executing webhook workflow at path: /$webhook_path"
        
        # Get webhook method from node parameters
        local webhook_method
        webhook_method=$(echo "$workflow_details" | jq -r '.nodes[]? | select(.type == "n8n-nodes-base.webhook") | .parameters.method // empty' 2>/dev/null | head -1)
        webhook_method="${webhook_method:-GET}"
        
        local data="${WORKFLOW_DATA:-{}}"
        local response
        local http_code
        
        if [[ "$webhook_method" == "GET" ]] || [[ -z "$webhook_method" ]]; then
            # For GET requests, append data as query parameters if provided
            local query_string=""
            if [[ "$data" != "{}" ]]; then
                query_string="?data=$(echo "$data" | jq -c . | sed 's/ /%20/g')"
            fi
            response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" -X GET \
                "${N8N_BASE_URL}/webhook/${webhook_path}${query_string}" 2>&1)
        else
            # For POST/PUT/etc, send data in body
            response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" -X "$webhook_method" \
                -H "Content-Type: application/json" \
                -d "$data" \
                "${N8N_BASE_URL}/webhook/${webhook_path}" 2>&1)
        fi
        
        # Extract HTTP code
        http_code=$(echo "$response" | grep "__HTTP_CODE__:" | cut -d':' -f2)
        response=$(echo "$response" | grep -v "__HTTP_CODE__:")
        
        if [[ "$http_code" == "200" ]] || [[ "$http_code" == "201" ]]; then
            log::success "✅ Webhook workflow executed successfully"
            if [[ -n "$response" ]] && [[ "$response" != "{}" ]] && [[ "$response" != "null" ]]; then
                echo ""
                echo "Response:"
                # Try to parse as JSON, fallback to raw output
                if echo "$response" | jq empty 2>/dev/null; then
                    echo "$response" | jq '.' 2>/dev/null
                else
                    echo "$response"
                fi
            fi
        else
            log::error "Webhook execution failed (HTTP $http_code)"
            echo "Response: $response"
            return 1
        fi
    else
        # No webhook - workflow has manual trigger only
        log::error "This workflow uses a Manual Trigger and cannot be executed via API"
        echo ""
        echo "n8n's public API does not support executing manual trigger workflows."
        echo ""
        echo "Options:"
        echo "  1. Replace the Manual Trigger with a Webhook node in your workflow"
        echo "  2. Use the n8n web interface to test manual workflows"
        echo "  3. Use the n8n CLI (currently broken in v1.93.0+)"
        echo ""
        echo "To create an API-executable workflow:"
        echo "  - Add a Webhook node as the trigger"
        echo "  - Set a unique path (e.g., 'my-workflow')"
        echo "  - In Webhook settings, set:"
        echo "    • Respond: 'When Last Node Finishes'"
        echo "    • Response Data: 'Last Node' (to get workflow output)"
        echo "  - Save and activate the workflow"
        echo "  - Execute with: $0 --action execute --workflow-id $workflow_id"
        return 1
    fi
}

#######################################
# API setup helper
#######################################
n8n::api_setup() {
    log::header "🔑 n8n API Setup Guide"
    n8n::show_api_setup_instructions
}

#######################################
# Save API key to configuration
#######################################
n8n::save_api_key() {
    local api_key="${API_KEY:-}"
    
    log::header "💾 Save n8n API Key"
    
    # Check if API key is provided
    if [[ -z "$api_key" ]]; then
        log::error "API key is required"
        echo ""
        echo "Usage: $0 --action save-api-key --api-key YOUR_API_KEY"
        echo ""
        echo "To create an API key:"
        echo "  1. Access n8n at $N8N_BASE_URL"
        echo "  2. Go to Settings → n8n API"
        echo "  3. Create and copy your API key"
        return 1
    fi
    
    # Ensure config directory exists
    local config_dir="${HOME}/.vrooli"
    mkdir -p "$config_dir"
    
    # Load existing config or create new
    local config_file="${config_dir}/service.json"
    local config
    
    if [[ -f "$config_file" ]]; then
        # Backup existing config
        cp "$config_file" "${config_file}.backup" 2>/dev/null || true
        config=$(cat "$config_file")
    else
        # Create default config structure
        config='{
  "version": "1.0.0",
  "enabled": true,
  "services": {
    "ai": {},
    "automation": {},
    "storage": {},
    "agents": {}
  }
}'
    fi
    
    # Update config with API key
    local updated_config
    updated_config=$(echo "$config" | jq --arg key "$api_key" '
        .services.automation.n8n = (.services.automation.n8n // {}) |
        .services.automation.n8n.apiKey = $key |
        .services.automation.n8n.enabled = true |
        .services.automation.n8n.baseUrl = "http://localhost:5678" |
        .services.automation.n8n.healthCheck = {
            "intervalMs": 60000,
            "timeoutMs": 5000
        }
    ')
    
    # Write updated config
    echo "$updated_config" | jq '.' > "$config_file"
    
    # Set secure permissions
    chmod 600 "$config_file"
    
    log::success "✅ API key saved to $config_file"
    echo ""
    echo "You can now execute workflows without setting N8N_API_KEY:"
    echo "  $0 --action execute --workflow-id YOUR_WORKFLOW_ID"
    echo ""
    echo "The API key will be loaded automatically from the configuration."
}

#######################################
# List workflows via API
# Returns: List of workflows with IDs
#######################################
n8n::list_workflows() {
    local api_key="${N8N_API_KEY:-}"
    
    # Try to load API key from config if not in env
    if [[ -z "$api_key" ]]; then
        local config_file="${HOME}/.vrooli/service.json"
        if [[ -f "$config_file" ]]; then
            api_key=$(jq -r '.services.automation.n8n.apiKey // empty' "$config_file" 2>/dev/null)
        fi
    fi
    
    if [[ -z "$api_key" ]]; then
        # Fallback to CLI if no API key
        if n8n::container_exists; then
            docker exec "$N8N_CONTAINER_NAME" n8n list:workflow
        else
            log::error "n8n container not found and no API key available"
            return 1
        fi
    else
        # Use API to list workflows
        local response
        response=$(curl -s -H "X-N8N-API-KEY: $api_key" \
            "${N8N_BASE_URL}/api/v1/workflows" 2>&1)
        
        if echo "$response" | jq empty 2>/dev/null; then
            echo "$response" | jq -r '.data[] | "\(.id)\t\(.name)\t\(.active)"' | \
                column -t -s $'\t' -N "ID,Name,Active"
        else
            log::error "Failed to list workflows via API"
            return 1
        fi
    fi
}

#######################################
# Get workflow execution history
# Args: workflow_id (optional)
# Returns: Execution history
#######################################
n8n::get_executions() {
    local workflow_id="${1:-}"
    local api_key="${N8N_API_KEY:-}"
    
    # Try to load API key from config if not in env
    if [[ -z "$api_key" ]]; then
        local config_file="${HOME}/.vrooli/service.json"
        if [[ -f "$config_file" ]]; then
            api_key=$(jq -r '.services.automation.n8n.apiKey // empty' "$config_file" 2>/dev/null)
        fi
    fi
    
    if [[ -z "$api_key" ]]; then
        log::error "API key required for viewing executions"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    local url="${N8N_BASE_URL}/api/v1/executions"
    if [[ -n "$workflow_id" ]]; then
        url="${url}?workflowId=${workflow_id}"
    fi
    
    local response
    response=$(curl -s -H "X-N8N-API-KEY: $api_key" "$url" 2>&1)
    
    if echo "$response" | jq empty 2>/dev/null; then
        echo "$response" | jq -r '.data[] | "\(.id)\t\(.workflowData.name)\t\(.finished)\t\(.stoppedAt)"' | \
            column -t -s $'\t' -N "ID,Workflow,Success,Finished"
    else
        log::error "Failed to get executions"
        return 1
    fi
}

#######################################
# Test API connectivity
# Returns: 0 if API is accessible, 1 otherwise
#######################################
n8n::test_api() {
    local api_key="${N8N_API_KEY:-}"
    
    # Try to load API key from config if not in env
    if [[ -z "$api_key" ]]; then
        local config_file="${HOME}/.vrooli/service.json"
        if [[ -f "$config_file" ]]; then
            api_key=$(jq -r '.services.automation.n8n.apiKey // empty' "$config_file" 2>/dev/null)
        fi
    fi
    
    if [[ -z "$api_key" ]]; then
        # Test without API key (basic connectivity)
        if curl -s -f "$N8N_BASE_URL/healthz" >/dev/null 2>&1; then
            log::success "n8n API is accessible (no auth)"
            return 0
        else
            log::error "n8n API is not accessible"
            return 1
        fi
    else
        # Test with API key
        local response
        response=$(curl -s -H "X-N8N-API-KEY: $api_key" \
            "${N8N_BASE_URL}/api/v1/workflows?limit=1" 2>&1)
        
        if echo "$response" | jq empty 2>/dev/null; then
            log::success "n8n API is accessible and authenticated"
            return 0
        else
            log::error "n8n API authentication failed"
            return 1
        fi
    fi
}