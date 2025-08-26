#!/usr/bin/env bash
# n8n API Management Functions
# Workflow execution, API key management, and REST API interactions

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_API_SOURCED:-}" ]] && return 0
export _N8N_API_SOURCED=1

# Source shared libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
N8N_LIB_DIR="${APP_ROOT}/resources/n8n/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
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
# Check if workflow exists by name
# Arguments:
#   $1 - workflow name
# Returns:
#   0 if exists (outputs workflow ID to stdout)
#   1 if not found
#######################################
n8n::workflow_exists_by_name() {
    local workflow_name="${1:-}"
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name is required"
        return 1
    fi
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::debug "No API key available for workflow existence check"
        return 1
    fi
    
    # Get all workflows
    local workflows_response
    workflows_response=$(curl -s -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows" 2>/dev/null || echo '{"data":[]}')
    
    # Find workflow by exact name match
    local workflow_id
    workflow_id=$(echo "$workflows_response" | jq -r --arg name "$workflow_name" \
        '.data[]? | select(.name == $name) | .id // empty' | head -1)
    
    if [[ -n "$workflow_id" ]]; then
        echo "$workflow_id"
        return 0
    else
        return 1
    fi
}

#######################################
# Get full workflow data by name
# Arguments:
#   $1 - workflow name
# Returns:
#   0 if found (outputs workflow JSON to stdout)
#   1 if not found
#######################################
n8n::get_workflow_by_name() {
    local workflow_name="${1:-}"
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name is required"
        return 1
    fi
    
    # First check if it exists and get ID
    local workflow_id
    if ! workflow_id=$(n8n::workflow_exists_by_name "$workflow_name"); then
        return 1
    fi
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        return 1
    fi
    
    # Get full workflow details
    local workflow_details
    workflow_details=$(curl -s -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" 2>/dev/null)
    
    if [[ -n "$workflow_details" ]] && echo "$workflow_details" | jq empty 2>/dev/null; then
        echo "$workflow_details"
        return 0
    else
        return 1
    fi
}

#######################################
# Calculate hash of workflow structure
# Arguments:
#   $1 - workflow JSON
# Returns:
#   Hash string to stdout
#######################################
n8n::calculate_workflow_hash() {
    local workflow_json="${1:-}"
    
    if [[ -z "$workflow_json" ]]; then
        echo ""
        return 1
    fi
    
    # Extract only structural elements (not state/runtime data)
    # This creates a deterministic representation of the workflow structure
    local structural_json
    structural_json=$(echo "$workflow_json" | jq -S '{
        nodes: .nodes | map({
            type: .type,
            typeVersion: .typeVersion,
            parameters: .parameters,
            position: .position
        }) | sort_by(.type, .typeVersion),
        connections: .connections,
        settings: {
            executionOrder: .settings.executionOrder,
            errorWorkflow: .settings.errorWorkflow,
            timezone: .settings.timezone,
            executionTimeout: .settings.executionTimeout,
            maxExecutionTimeout: .settings.maxExecutionTimeout
        }
    }' 2>/dev/null || echo '{}')
    
    # Calculate SHA256 hash
    echo -n "$structural_json" | sha256sum | cut -d' ' -f1
}

#######################################
# Detect if workflow has changed
# Arguments:
#   $1 - existing workflow JSON
#   $2 - new workflow JSON
# Returns:
#   0 if changed, 1 if identical
#######################################
n8n::detect_workflow_changes() {
    local existing_workflow="${1:-}"
    local new_workflow="${2:-}"
    
    if [[ -z "$existing_workflow" ]] || [[ -z "$new_workflow" ]]; then
        return 0  # Assume changed if can't compare
    fi
    
    local existing_hash
    existing_hash=$(n8n::calculate_workflow_hash "$existing_workflow")
    
    local new_hash
    new_hash=$(n8n::calculate_workflow_hash "$new_workflow")
    
    log::debug "Existing workflow hash: $existing_hash"
    log::debug "New workflow hash: $new_hash"
    
    if [[ "$existing_hash" == "$new_hash" ]]; then
        return 1  # No changes
    else
        return 0  # Has changes
    fi
}

#######################################
# Upgrade existing workflow
# Arguments:
#   $1 - workflow ID
#   $2 - new workflow data JSON
#   $3 - preserve fields (comma-separated, optional)
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n::upgrade_workflow() {
    local workflow_id="${1:-}"
    local new_workflow_data="${2:-}"
    local preserve_fields="${3:-staticData,credentials,active,webhookId,versionId}"
    
    if [[ -z "$workflow_id" ]] || [[ -z "$new_workflow_data" ]]; then
        log::error "Workflow ID and new data are required for upgrade"
        return 1
    fi
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow upgrade"
        return 1
    fi
    
    # Get existing workflow to preserve fields
    local existing_workflow
    existing_workflow=$(curl -s -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" 2>/dev/null)
    
    if [[ -z "$existing_workflow" ]]; then
        log::error "Failed to fetch existing workflow for upgrade"
        return 1
    fi
    
    # Build upgrade payload preserving specified fields
    # n8n PUT endpoint only accepts specific fields: name, nodes, connections, settings, staticData
    # Note: 'active' field is read-only and cannot be set via PUT
    local upgrade_payload
    upgrade_payload=$(echo "$new_workflow_data" | jq --arg preserve "$preserve_fields" --argjson existing "$existing_workflow" '
        . as $new |
        ($preserve | split(",")) as $preserve_list |
        # Start with base fields from new workflow
        {
            name: $new.name,
            nodes: $new.nodes,
            connections: $new.connections,
            settings: $new.settings
        } |
        # Add preserved staticData if requested
        if (($preserve_list | index("staticData")) and ($existing | has("staticData"))) then
            .staticData = $existing.staticData
        else . end
    ')
    
    # PUT the workflow (n8n uses PUT for updates, not PATCH)
    local response
    response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -X PUT \
        -H "X-N8N-API-KEY: $api_key" \
        -H "Content-Type: application/json" \
        -d "$upgrade_payload" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    local http_code
    http_code=$(echo "$response" | grep "__HTTP_CODE__:" | sed 's/.*__HTTP_CODE__://')
    response=$(echo "$response" | grep -v "__HTTP_CODE__:")
    
    if [[ "$http_code" == "200" ]]; then
        log::success "Successfully upgraded workflow: $workflow_id"
        return 0
    else
        log::error "Failed to upgrade workflow (HTTP $http_code)"
        if [[ -n "$response" ]]; then
            local error_message
            error_message=$(echo "$response" | jq -r '.message // .error // .' 2>/dev/null || echo "$response")
            log::error "API Error: $error_message"
        fi
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
# Validate API key is still working
# Returns: 0 if valid, 1 if invalid/missing
#######################################
n8n::validate_api_key() {
    local api_key
    api_key=$(n8n::resolve_api_key)
    
    if [[ -z "$api_key" ]]; then
        return 1
    fi
    
    # Test API key by making a simple request
    local response
    response=$(http::request "GET" "${N8N_BASE_URL}/api/v1/workflows?limit=1" "" "X-N8N-API-KEY: $api_key" 2>/dev/null || echo "")
    
    # Check if response is valid JSON with data field
    if echo "$response" | jq -e '.data' >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Warn about API key invalidation risk
# Args: operation_name
#######################################
n8n::warn_api_key_risk() {
    local operation="${1:-operation}"
    
    # Check if we have an API key stored
    local stored_key
    stored_key=$(n8n::resolve_api_key)
    
    if [[ -n "$stored_key" ]]; then
        log::warn "⚠️  WARNING: API Key Invalidation Risk"
        echo ""
        echo "This $operation may invalidate your stored API key!"
        echo ""
        echo "API keys are tied to the n8n database and user accounts."
        echo "If the database is reset or user accounts are recreated,"
        echo "your stored API key will become invalid."
        echo ""
        echo "Current API key status:"
        if n8n::validate_api_key; then
            log::success "  ✅ API key is currently valid"
        else
            log::error "  ❌ API key is already invalid or n8n is not accessible"
        fi
        echo ""
        echo "If the API key becomes invalid after this operation:"
        echo "  1. Access n8n at $N8N_BASE_URL"
        echo "  2. Login with your credentials"
        echo "  3. Go to Settings → n8n API"
        echo "  4. Create a new API key"
        echo "  5. Save it with: $0 --action save-api-key --api-key YOUR_NEW_KEY"
        echo ""
        
        if [[ "${YES:-no}" != "yes" ]]; then
            read -p "Do you want to continue? (y/N): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log::info "Operation cancelled"
                return 1
            fi
        fi
    fi
    
    return 0
}

#######################################
# Check and report API key status after operation
# Args: operation_name
#######################################
n8n::check_api_key_after_operation() {
    local operation="${1:-operation}"
    
    # Only check if we have a stored key
    local stored_key
    stored_key=$(n8n::resolve_api_key)
    
    if [[ -n "$stored_key" ]]; then
        log::info "Checking API key status after $operation..."
        
        # Wait a moment for services to stabilize
        sleep 2
        
        if n8n::validate_api_key; then
            log::success "✅ API key is still valid"
        else
            log::error "❌ API key has been invalidated!"
            echo ""
            echo "Your stored API key is no longer valid."
            echo "This typically happens when:"
            echo "  • The n8n database was reset"
            echo "  • User accounts were recreated"
            echo "  • The container was recreated with different settings"
            echo ""
            echo "To fix this:"
            echo "  1. Access n8n at $N8N_BASE_URL"
            echo "  2. Login with your credentials"
            echo "  3. Go to Settings → n8n API"
            echo "  4. Create a new API key"
            echo "  5. Save it with: $0 --action save-api-key --api-key YOUR_NEW_KEY"
        fi
    fi
}

#######################################
# Test API key command for users
# Provides detailed feedback about API key status
#######################################
n8n::test_api_key() {
    log::header "🔑 Testing n8n API Key"
    
    # Check if n8n is running
    if ! docker::is_running "$N8N_CONTAINER_NAME"; then
        log::error "n8n is not running"
        echo "Start n8n with: $0 --action start"
        return 1
    fi
    
    # Check if we have an API key
    local api_key
    api_key=$(n8n::resolve_api_key)
    
    if [[ -z "$api_key" ]]; then
        log::error "No API key found"
        echo ""
        echo "API key resolution order:"
        echo "  1. Environment variable: N8N_API_KEY"
        echo "  2. Project secrets: .vrooli/secrets.json"
        echo "  3. Container environment (if running)"
        echo ""
        echo "To set up an API key:"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    log::info "Found API key in: $(
        if [[ -n "${N8N_API_KEY:-}" ]]; then
            echo "environment variable"
        elif secrets::get_key "N8N_API_KEY" >/dev/null 2>&1; then
            echo "project secrets (.vrooli/secrets.json)"
        else
            echo "container environment"
        fi
    )"
    
    log::info "Testing API key validity..."
    
    if n8n::validate_api_key; then
        log::success "✅ API key is valid and working!"
        
        # Try to get some basic info to prove it works
        local workflows_response
        workflows_response=$(http::request "GET" "${N8N_BASE_URL}/api/v1/workflows?limit=5" "" "X-N8N-API-KEY: $api_key" 2>/dev/null || echo "{}")
        
        if echo "$workflows_response" | jq -e '.data' >/dev/null 2>&1; then
            local workflow_count
            workflow_count=$(echo "$workflows_response" | jq '.data | length')
            echo ""
            echo "API key has access to $workflow_count workflow(s)"
        fi
    else
        log::error "❌ API key is invalid or not working"
        echo ""
        echo "This can happen when:"
        echo "  • The n8n database was reset or corrupted"
        echo "  • User accounts were recreated"
        echo "  • The API key was revoked in n8n"
        echo ""
        echo "To fix this:"
        echo "  1. Access n8n at $N8N_BASE_URL"
        echo "  2. Login with your credentials"
        echo "  3. Go to Settings → n8n API"
        echo "  4. Create a new API key"
        echo "  5. Save it with: $0 --action save-api-key --api-key YOUR_NEW_KEY"
        echo ""
        echo "For detailed recovery instructions, see:"
        echo "  docs/API_KEY_RECOVERY.md"
        return 1
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
    if n8n::check_basic_health >/dev/null 2>&1; then
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

#######################################
# Activate a single workflow by ID
# Args: $1 - workflow_id
# Returns: 0 if successful, 1 if failed
#######################################
n8n::activate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: n8n::activate_workflow <workflow_id>"
        return 1
    fi
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow activation"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    log::info "Activating workflow: $workflow_id"
    log::debug "Making HTTP request to: ${N8N_BASE_URL}/api/v1/workflows/${workflow_id}/activate"
    
    # Use direct curl instead of http::request to avoid hanging issues
    local response http_code exit_code
    
    response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -X POST \
        -H "X-N8N-API-KEY: $api_key" \
        -H "Content-Type: application/json" \
        -d '{}' \
        --max-time 3 \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}/activate" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    # Extract HTTP code and body
    http_code=$(echo "$response" | grep "__HTTP_CODE__:" | cut -d':' -f2 | tr -d '\n' | sed 's/[^0-9]//g')
    response=$(echo "$response" | grep -v "__HTTP_CODE__:")
    
    # Set exit code based on HTTP status
    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        exit_code=0
    else
        exit_code=1
    fi
    
    log::debug "HTTP request completed with code: $http_code, exit code: $exit_code"
    log::debug "Response length: ${#response} characters"
    
    # Handle timeout specifically
    if [[ $exit_code -eq 124 ]]; then
        log::error "❌ Activation request timed out after 10 seconds for workflow: $workflow_id"
        return 1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        # Check if response indicates success
        if echo "$response" | jq -e '.active == true' >/dev/null 2>&1; then
            local workflow_name
            workflow_name=$(echo "$response" | jq -r '.name // "Unknown"')
            log::success "✅ Activated workflow: $workflow_name ($workflow_id)"
            return 0
        else
            log::error "❌ Activation request succeeded but workflow may not be active"
            log::debug "Response: $response"
            return 1
        fi
    else
        log::error "❌ Failed to activate workflow: $workflow_id (exit code: $exit_code)"
        
        # Try to extract error message from response
        if [[ -n "$response" ]]; then
            if echo "$response" | jq empty 2>/dev/null; then
                local error_msg
                error_msg=$(echo "$response" | jq -r '.message // .error // "Unknown error"')
                log::error "API Error: $error_msg"
            else
                log::error "Response: $response"
            fi
        fi
        return 1
    fi
}

#######################################
# Deactivate a single workflow by ID
# Args: $1 - workflow_id
# Returns: 0 if successful, 1 if failed
#######################################
n8n::deactivate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: n8n::deactivate_workflow <workflow_id>"
        return 1
    fi
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow deactivation"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    log::info "Deactivating workflow: $workflow_id"
    
    # Make deactivation request
    local response
    response=$(http::request "POST" "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}/deactivate" '{}' "X-N8N-API-KEY: $api_key")
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        # Check if response indicates success
        if echo "$response" | jq -e '.active == false' >/dev/null 2>&1; then
            local workflow_name
            workflow_name=$(echo "$response" | jq -r '.name // "Unknown"')
            log::success "✅ Deactivated workflow: $workflow_name ($workflow_id)"
            return 0
        else
            log::error "❌ Deactivation request succeeded but workflow may still be active"
            return 1
        fi
    else
        log::error "❌ Failed to deactivate workflow: $workflow_id"
        if echo "$response" | jq empty 2>/dev/null; then
            local error_msg
            error_msg=$(echo "$response" | jq -r '.message // .error // "Unknown error"')
            log::error "Error: $error_msg"
        fi
        return 1
    fi
}

#######################################
# Activate all workflows
# Returns: 0 if all successful, 1 if any failed
#######################################
n8n::activate_all_workflows() {
    log::header "🚀 Activating All Workflows"
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow activation"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    # Get all workflows - use direct curl to avoid hanging
    local workflows_response http_code
    workflows_response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    # Extract HTTP code and body
    http_code=$(echo "$workflows_response" | grep "__HTTP_CODE__:" | cut -d':' -f2 | tr -d '\n' | sed 's/[^0-9]//g')
    workflows_response=$(echo "$workflows_response" | grep -v "__HTTP_CODE__:")
    
    if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]] || ! echo "$workflows_response" | jq empty 2>/dev/null; then
        log::error "Failed to get workflows list (HTTP $http_code)"
        return 1
    fi
    
    local total_count
    total_count=$(echo "$workflows_response" | jq '.data | length')
    
    if [[ "$total_count" -eq 0 ]]; then
        log::info "No workflows found to activate"
        return 0
    fi
    
    log::info "Found $total_count workflows to process"
    echo ""
    
    local success_count=0
    local skip_count=0
    local fail_count=0
    
    # Process each workflow using array approach instead of process substitution
    local workflow_count
    workflow_count=$(echo "$workflows_response" | jq '.data | length')
    
    # Pre-declare variables outside the loop to avoid local declaration issues
    local workflow_data workflow_id workflow_name is_active
    
    for ((i=0; i<workflow_count; i++)); do
        # Extract workflow data safely
        workflow_data=$(echo "$workflows_response" | jq -c ".data[$i]" 2>/dev/null || echo "{}")
        [[ "$workflow_data" == "{}" ]] && continue
        
        workflow_id=$(echo "$workflow_data" | jq -r '.id // ""' 2>/dev/null || echo "")
        workflow_name=$(echo "$workflow_data" | jq -r '.name // ""' 2>/dev/null || echo "")
        is_active=$(echo "$workflow_data" | jq -r '.active // false' 2>/dev/null || echo "false")
        
        [[ -z "$workflow_id" ]] && continue
        [[ -z "$workflow_name" ]] && continue
        
        if [[ "$is_active" == "true" ]]; then
            log::info "⏭️  Skipping already active: $workflow_name ($workflow_id)"
            skip_count=$((skip_count + 1))
            continue
        fi
        
        # Activate the workflow
        log::info "Activating: $workflow_name ($workflow_id)"
        if n8n::activate_workflow "$workflow_id"; then
            success_count=$((success_count + 1))
        else
            log::error "❌ Failed to activate: $workflow_name ($workflow_id)"
            fail_count=$((fail_count + 1))
        fi
    done
    
    # Summary
    echo ""
    log::info "📊 Activation Summary:"
    log::info "   Activated: $success_count"
    log::info "   Already Active: $skip_count"
    log::info "   Failed: $fail_count"
    log::info "   Total: $total_count"
    
    if [[ $fail_count -eq 0 ]]; then
        log::success "✅ All workflows processed successfully"
        return 0
    else
        log::error "❌ $fail_count workflows failed to activate"
        return 1
    fi
}

#######################################
# Activate workflows matching a pattern
# Args: $1 - pattern (supports wildcards and regex)
# Returns: 0 if any activated, 1 if none found or all failed
#######################################
n8n::activate_workflows_by_pattern() {
    local pattern="${1:-}"
    
    if [[ -z "$pattern" ]]; then
        log::error "Pattern is required"
        echo "Usage: n8n::activate_workflows_by_pattern <pattern>"
        echo ""
        echo "Examples:"
        echo "  n8n::activate_workflows_by_pattern \"embedding*\""
        echo "  n8n::activate_workflows_by_pattern \"*generator*\""
        echo "  n8n::activate_workflows_by_pattern \"reasoning-chain\""
        return 1
    fi
    
    log::header "🎯 Activating Workflows Matching Pattern: $pattern"
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow activation"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    # Get all workflows - use direct curl to avoid hanging
    local workflows_response http_code
    workflows_response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    # Extract HTTP code and body
    http_code=$(echo "$workflows_response" | grep "__HTTP_CODE__:" | cut -d':' -f2 | tr -d '\n' | sed 's/[^0-9]//g')
    workflows_response=$(echo "$workflows_response" | grep -v "__HTTP_CODE__:")
    
    if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]] || ! echo "$workflows_response" | jq empty 2>/dev/null; then
        log::error "Failed to get workflows list (HTTP $http_code)"
        return 1
    fi
    
    local total_count
    total_count=$(echo "$workflows_response" | jq '.data | length')
    
    if [[ "$total_count" -eq 0 ]]; then
        log::info "No workflows found"
        return 1
    fi
    
    log::info "Searching through $total_count workflows for pattern: $pattern"
    echo ""
    
    local match_count=0
    local success_count=0
    local skip_count=0
    local fail_count=0
    
    # Convert wildcard pattern to regex if needed
    local regex_pattern="$pattern"
    if [[ "$pattern" == *"*"* ]] || [[ "$pattern" == *"?"* ]]; then
        # Convert shell wildcards to regex (use .* instead of .*? for greedy matching)
        regex_pattern=$(echo "$pattern" | sed 's/\./\\./g; s/\*/.*/g; s/\?/./g')
        regex_pattern="^${regex_pattern}$"
    fi
    
    # OPTIMIZED: Extract all workflow data at once to avoid multiple jq calls in loop
    local workflows_extract
    workflows_extract=$(echo "$workflows_response" | jq -r '.data[] | "\(.id)|\(.name)|\(.active)"' 2>/dev/null)
    
    if [[ -z "$workflows_extract" ]]; then
        log::error "Failed to extract workflow data"
        return 1
    fi
    
    # Use index-based iteration for reliability
    local workflow_count
    workflow_count=$(echo "$workflows_response" | jq '.data | length')
    
    # Pre-declare loop variables  
    local matches workflow_id workflow_name is_active workflow_data
    
    # Process each workflow using index-based iteration (most reliable)
    for ((i=0; i<workflow_count; i++)); do
        # Extract workflow data directly from JSON
        workflow_data=$(echo "$workflows_response" | jq -c ".data[$i]" 2>/dev/null || echo "{}")
        [[ "$workflow_data" == "{}" ]] && continue
        
        workflow_id=$(echo "$workflow_data" | jq -r '.id // ""' 2>/dev/null || echo "")
        workflow_name=$(echo "$workflow_data" | jq -r '.name // ""' 2>/dev/null || echo "")
        is_active=$(echo "$workflow_data" | jq -r '.active // false' 2>/dev/null || echo "false")
        
        [[ -z "$workflow_id" || -z "$workflow_name" ]] && continue
        
        # Check if workflow name matches pattern
        matches=false
        if [[ "$pattern" == *"*"* ]] || [[ "$pattern" == *"?"* ]]; then
            # Use regex matching for wildcards
            if echo "$workflow_name" | grep -qE "$regex_pattern"; then
                matches=true
            fi
        else
            # Simple substring matching
            if [[ "$workflow_name" == *"$pattern"* ]]; then
                matches=true
            fi
        fi
        
        if [[ "$matches" == "true" ]]; then
            match_count=$((match_count + 1))
            log::info "🎯 Found match: $workflow_name"
            
            if [[ "$is_active" == "true" ]]; then
                log::info "⏭️  Already active: $workflow_name ($workflow_id)"
                skip_count=$((skip_count + 1))
                continue
            fi
            
            # Activate the workflow
            log::info "Activating: $workflow_name ($workflow_id)"
            if n8n::activate_workflow "$workflow_id"; then
                success_count=$((success_count + 1))
                log::info "✅ Successfully activated: $workflow_name"
            else
                log::error "❌ Failed to activate: $workflow_name ($workflow_id)"
                fail_count=$((fail_count + 1))
            fi
        fi
    done
    
    log::debug "Completed workflow iteration. Found $match_count matches."
    
    # Summary
    echo ""
    if [[ $match_count -eq 0 ]]; then
        log::warn "No workflows matched pattern: $pattern"
        echo ""
        echo "Available workflows:"
        echo "$workflows_response" | jq -r '.data[] | "  • \(.name)"' | head -10
        if [[ $total_count -gt 10 ]]; then
            echo "  ... and $((total_count - 10)) more"
        fi
        return 1
    fi
    
    log::info "📊 Pattern Activation Summary:"
    log::info "   Pattern: $pattern"
    log::info "   Matched: $match_count"
    log::info "   Activated: $success_count"
    log::info "   Already Active: $skip_count"
    log::info "   Failed: $fail_count"
    
    if [[ $success_count -gt 0 ]]; then
        log::success "✅ Successfully activated $success_count workflow(s)"
        return 0
    elif [[ $skip_count -gt 0 ]]; then
        log::info "ℹ️  All matching workflows were already active"
        return 0
    else
        log::error "❌ No workflows were activated"
        return 1
    fi
}

#######################################
# Delete a single workflow by ID
# Args: $1 - workflow_id
# Returns: 0 if successful, 1 if failed
#######################################
n8n::delete_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: n8n::delete_workflow <workflow_id>"
        return 1
    fi
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow deletion"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    log::info "Deleting workflow: $workflow_id"
    
    # Use direct curl instead of http::request to avoid hanging issues
    local response http_code
    response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -X DELETE \
        -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows/${workflow_id}" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    # Extract HTTP code and body
    http_code=$(echo "$response" | grep "__HTTP_CODE__:" | cut -d':' -f2 | tr -d '\n' | sed 's/[^0-9]//g')
    response=$(echo "$response" | grep -v "__HTTP_CODE__:")
    
    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        log::success "✅ Deleted workflow: $workflow_id"
        return 0
    else
        log::error "❌ Failed to delete workflow: $workflow_id (HTTP $http_code)"
        if [[ -n "$response" ]]; then
            if echo "$response" | jq empty 2>/dev/null; then
                local error_msg
                error_msg=$(echo "$response" | jq -r '.message // .error // "Unknown error"')
                log::error "API Error: $error_msg"
            else
                log::error "Response: $response"
            fi
        fi
        return 1
    fi
}

#######################################
# Delete all workflows
# Returns: 0 if all successful, 1 if any failed
#######################################
n8n::delete_all_workflows() {
    log::header "🗑️  Deleting All Workflows"
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "API key required for workflow deletion"
        n8n::show_api_setup_instructions
        return 1
    fi
    
    # Get all workflows - use direct curl to avoid hanging
    local workflows_response http_code
    workflows_response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -H "X-N8N-API-KEY: $api_key" \
        --max-time 10 \
        "${N8N_BASE_URL}/api/v1/workflows" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    # Extract HTTP code and body
    http_code=$(echo "$workflows_response" | grep "__HTTP_CODE__:" | cut -d':' -f2 | tr -d '\n' | sed 's/[^0-9]//g')
    workflows_response=$(echo "$workflows_response" | grep -v "__HTTP_CODE__:")
    
    if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]] || ! echo "$workflows_response" | jq empty 2>/dev/null; then
        log::error "Failed to get workflows list (HTTP $http_code)"
        return 1
    fi
    
    local total_count
    total_count=$(echo "$workflows_response" | jq '.data | length')
    
    if [[ "$total_count" -eq 0 ]]; then
        log::info "No workflows found to delete"
        return 0
    fi
    
    log::info "Found $total_count workflows to delete"
    echo ""
    
    # Confirm deletion unless YES flag is set
    if [[ "${YES:-no}" != "yes" ]]; then
        log::warn "⚠️  This will permanently delete ALL $total_count workflows!"
        echo ""
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Operation cancelled"
            return 1
        fi
        echo ""
    fi
    
    local success_count=0
    local fail_count=0
    
    # Pre-declare variables outside the loop
    local workflow_data workflow_id workflow_name
    
    for ((i=0; i<total_count; i++)); do
        # Extract workflow data safely
        workflow_data=$(echo "$workflows_response" | jq -c ".data[$i]" 2>/dev/null || echo "{}")
        [[ "$workflow_data" == "{}" ]] && continue
        
        workflow_id=$(echo "$workflow_data" | jq -r '.id // ""' 2>/dev/null || echo "")
        workflow_name=$(echo "$workflow_data" | jq -r '.name // ""' 2>/dev/null || echo "")
        
        [[ -z "$workflow_id" ]] && continue
        [[ -z "$workflow_name" ]] && continue
        
        # Delete the workflow
        log::info "Deleting: $workflow_name ($workflow_id)"
        if n8n::delete_workflow "$workflow_id" >/dev/null 2>&1; then
            success_count=$((success_count + 1))
        else
            log::error "❌ Failed to delete: $workflow_name ($workflow_id)"
            fail_count=$((fail_count + 1))
        fi
    done
    
    # Summary
    echo ""
    log::info "📊 Deletion Summary:"
    log::info "   Deleted: $success_count"
    log::info "   Failed: $fail_count"
    log::info "   Total: $total_count"
    
    if [[ $fail_count -eq 0 ]]; then
        log::success "✅ All workflows deleted successfully"
        return 0
    else
        log::error "❌ $fail_count workflows failed to delete"
        return 1
    fi
}
