#!/usr/bin/env bash
# n8n API Management Functions
# Workflow execution, API key management, and REST API interactions

# Source shared libraries
N8N_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"

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
        echo "  docker::exec $N8N_CONTAINER_NAME n8n list:workflow"
        return 1
    fi
    # Check for API key using unified resolution
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::warn "No API key found"
        echo ""
        n8n::show_api_setup_instructions
        return 1
    fi
    # First, get workflow details to check if it has a webhook
    log::info "Fetching workflow details: $workflow_id"
    local workflow_details
    # Use standardized HTTP utility
    workflow_details=$(http::request "GET" "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" "" "X-N8N-API-KEY: $api_key")
    # Check if workflow exists
    if echo "$workflow_details" | grep -q '"message".*not found'; then
        log::error "Workflow not found"
        echo ""
        echo "Available workflows:"
        docker::exec "$N8N_CONTAINER_NAME" n8n list:workflow
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
            # Use standardized HTTP utility
            activate_response=$(http::request "POST" "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}/activate" '{}' "X-N8N-API-KEY: $api_key")
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
            # Use standardized HTTP utility
            response=$(http::request "GET" "${N8N_BASE_URL}/webhook/${webhook_path}${query_string}")
            http_code=$?
        else
            # For POST/PUT/etc, send data in body
            # Use standardized HTTP utility
            response=$(http::request "$webhook_method" "${N8N_BASE_URL}/webhook/${webhook_path}" "$data")
            http_code=$?
        fi
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
    # Save API key to project secrets using shared library
    if secrets::save_key "N8N_API_KEY" "$api_key"; then
        local secrets_file
        secrets_file="$(secrets::get_secrets_file)"
        log::success "✅ API key saved to $secrets_file"
        echo ""
        echo "You can now execute workflows without setting N8N_API_KEY:"
        echo "  $0 --action execute --workflow-id YOUR_WORKFLOW_ID"
        echo ""
        echo "The API key will be loaded automatically using 3-layer resolution:"
        echo "  1. Environment variable N8N_API_KEY"
        echo "  2. Project secrets file: $secrets_file"
        echo "  3. HashiCorp Vault (if configured)"
    else
        log::error "Failed to save API key"
        return 1
    fi
}

#######################################
# List workflows via API
# Returns: List of workflows with IDs
#######################################
n8n::list_workflows() {
    local api_key="${N8N_API_KEY:-}"
    # Get API key using unified resolution
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        # Fallback to CLI if no API key
        if docker::container_exists "$N8N_CONTAINER_NAME"; then
            docker::exec "$N8N_CONTAINER_NAME" n8n list:workflow
        else
            log::error "n8n container not found and no API key available"
            return 1
        fi
    else
        # Use API to list workflows
        local response
        # Use standardized HTTP utility
        response=$(http::request "GET" "${N8N_BASE_URL}/api/v1/workflows" "" "X-N8N-API-KEY: $api_key")
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
    # Get API key using unified resolution
    api_key=$(n8n::resolve_api_key)
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
    # Use standardized HTTP utility
    response=$(http::request "GET" "$url" "" "X-N8N-API-KEY: $api_key")
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
    # Get API key using unified resolution
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        # Test without API key (basic connectivity)
        # Use standardized HTTP utility
        if http::request "GET" "$N8N_BASE_URL/healthz" >/dev/null 2>&1; then
            log::success "n8n API is accessible (no auth)"
            return 0
        else
            log::error "n8n API is not accessible"
            return 1
        fi
    else
        # Test with API key
        local response
        # Use standardized HTTP utility
        response=$(http::request "GET" "${N8N_BASE_URL}/api/v1/workflows?limit=1" "" "X-N8N-API-KEY: $api_key")
        if echo "$response" | jq empty 2>/dev/null; then
            log::success "n8n API is accessible and authenticated"
            return 0
        else
            log::error "n8n API authentication failed"
            return 1
        fi
    fi
}

#######################################
# Get n8n service URLs with health status
# Outputs JSON with service URLs and health information
#######################################
n8n::get_urls() {
    local health_status="unknown"
    local health_path="/api/v1/info"
    local primary_url="$N8N_BASE_URL"
    local health_url="${N8N_BASE_URL}${health_path}"
    # Check if n8n is running and healthy
    if n8n::is_healthy >/dev/null 2>&1; then
        health_status="healthy"
    elif n8n::is_installed >/dev/null 2>&1; then
        # Installed but not running
        health_status="unhealthy"
    else
        # Not installed
        health_status="unavailable"
    fi
    # Output URL information as JSON
    cat << EOF
{
  "primary": "${primary_url}",
  "health": "${health_url}",
  "name": "n8n Workflow Automation",
  "type": "http",
  "status": "${health_status}",
  "port": ${N8N_PORT},
  "endpoints": {
    "api": "${N8N_BASE_URL}/api/v1",
    "workflows": "${N8N_BASE_URL}/api/v1/workflows",
    "executions": "${N8N_BASE_URL}/api/v1/executions",
    "webhooks": "${N8N_BASE_URL}/webhook"
  }
}
EOF
}
