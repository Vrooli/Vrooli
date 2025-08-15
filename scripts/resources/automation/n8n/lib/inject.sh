#!/usr/bin/env bash
set -euo pipefail

DESCRIPTION="Inject workflows and configurations into n8n automation platform"

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_INJECT_SOURCED:-}" ]] && return 0
export _N8N_INJECT_SOURCED=1

# Get script directory and source framework
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/validation_utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

# Load n8n configuration and infrastructure
if command -v inject_framework::load_adapter_config &>/dev/null; then
    inject_framework::load_adapter_config "n8n" "$(dirname "$N8N_LIB_DIR")"
fi

# Source n8n lib functions (load core, status, and auto-credentials)
for lib_file in "${N8N_LIB_DIR}/"{core,status,auto-credentials}.sh; do
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || log::warn "Could not load $lib_file"
    fi
done

# n8n-specific configuration with readonly protection
if [[ -z "${N8N_BASE_URL:-}" ]]; then
    N8N_BASE_URL="http://localhost:5678"
fi
if ! readonly -p | grep -q "N8N_BASE_URL"; then
    readonly N8N_BASE_URL
fi
if [[ -z "${N8N_API_BASE:-}" ]]; then
    N8N_API_BASE="$N8N_BASE_URL/api/v1"
fi
if ! readonly -p | grep -q "N8N_API_BASE"; then
    readonly N8N_API_BASE
fi

#######################################
# n8n-specific health check - ULTRA-OPTIMIZED
# Direct usage of n8n infrastructure (replaces 19 lines with 6 lines)
# Returns: 0 if healthy, 1 otherwise
#######################################
n8n::check_health() {
    # Direct usage of n8n infrastructure (replaces 19 lines with 6 lines)
    if n8n::check_basic_health; then
        log::debug "n8n is healthy and ready for data injection"
        return 0
    else
        log::error "n8n is not accessible for data injection"
        log::info "Ensure n8n is running: ./scripts/resources/automation/n8n/manage.sh --action start"
        return 1
    fi
}

#######################################
# Validate individual workflow
# Arguments:
#   $1 - workflow configuration object  
#   $2 - index
#   $3 - workflow name
# Returns:
#   0 if valid, 1 if invalid
#######################################
n8n::validate_workflow() {
    local workflow="$1"
    local index="$2" 
    local name="$3"
    # Validate required fields with context
    local file
    file=$(echo "$workflow" | jq -r '.file // empty')
    if [[ -z "$file" ]]; then
        log::error "Workflow '$name' at index $index missing required 'file' field"
        return 1
    fi
    # Validate workflow file exists and is valid JSON - use same path resolution as import
    local workflow_file
    if [[ -n "${N8N_SCENARIO_DIR:-}" ]]; then
        workflow_file="$N8N_SCENARIO_DIR/$file"
    else
        workflow_file=$(inject_framework::resolve_file_path "$file")
    fi
    
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found for '$name': $workflow_file"
        log::debug "Checked paths: N8N_SCENARIO_DIR=${N8N_SCENARIO_DIR:-unset}, resolved_path=$workflow_file"
        return 1
    fi
    
    # Read and process templates before validation
    local raw_content
    if ! raw_content=$(cat "$workflow_file" 2>/dev/null); then
        log::error "Failed to read workflow file for '$name': $workflow_file"
        return 1
    fi
    
    # Process template substitutions for validation using safe substitution
    local processed_content
    if ! processed_content=$(echo "$raw_content" | secrets::substitute_safe_templates 2>/dev/null); then
        log::warn "Template processing failed for validation of '$name', continuing with raw content"
        processed_content="$raw_content"
    fi
    
    # Validate processed JSON
    if ! echo "$processed_content" | jq . >/dev/null 2>&1; then
        log::error "Workflow file contains invalid JSON for '$name': $workflow_file"
        return 1
    fi
    # Validate enabled field if present
    local enabled
    enabled=$(echo "$workflow" | jq -r '.enabled // "true"')
    if [[ "$enabled" != "true" && "$enabled" != "false" ]]; then
        log::error "Workflow '$name' has invalid 'enabled' value: $enabled (must be true/false)"
        return 1
    fi
    # Validate tags if present
    if echo "$workflow" | jq -e '.tags' >/dev/null 2>&1; then
        local tags_type
        tags_type=$(echo "$workflow" | jq -r '.tags | type')
        if [[ "$tags_type" != "array" ]]; then
            log::error "Workflow '$name' tags must be an array, got: $tags_type"
            return 1
        fi
        # Validate each tag is a string
        local tag_count
        tag_count=$(echo "$workflow" | jq '.tags | length')
        for ((j=0; j<tag_count; j++)); do
            local tag_type
            tag_type=$(echo "$workflow" | jq -r ".tags[$j] | type")
            if [[ "$tag_type" != "string" ]]; then
                log::error "Workflow '$name' tag at index $j must be a string, got: $tag_type"
                return 1
            fi
        done
    fi
    log::debug "Workflow '$name' validation passed"
    return 0
}

#######################################
# n8n-specific validation function
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
n8n::validate_config() {
    local config="$1"
    log::info "Validating n8n injection configuration..."
    # Use framework JSON validation with required fields
    if ! inject_framework::validate_json "$config" "workflows credentials variables"; then
        return 1
    fi
    # Validate JSON structure
    if ! inject_framework::validate_json_structure "$config" "workflows:array credentials:array variables:object"; then
        return 1
    fi
    # Validate workflows if present with detailed context
    if echo "$config" | jq -e '.workflows' >/dev/null 2>&1; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        # Use enhanced validation with custom validator
        if ! inject_framework::validate_array_with_context "$workflows" "workflows" "name file" "n8n::validate_workflow"; then
            return 1
        fi
    fi
    # Validate credentials if present
    if echo "$config" | jq -e '.credentials' >/dev/null 2>&1; then
        local credentials
        credentials=$(echo "$config" | jq -c '.credentials')
        if ! validation::validate_credentials_config "$credentials"; then
            return 1
        fi
    fi
    # Validate variables if present
    if echo "$config" | jq -e '.variables' >/dev/null 2>&1; then
        local variables_type
        variables_type=$(echo "$config" | jq -r '.variables | type')
        if [[ "$variables_type" != "object" ]]; then
            log::error "Variables configuration must be an object, got: $variables_type"
            return 1
        fi
    fi
    log::success "n8n injection configuration is valid"
    return 0
}

#######################################
# Create credential in n8n via API
# Arguments:
#   $1 - credential configuration object
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n::create_credential() {
    local credential="$1"
    local name type_name data
    name=$(echo "$credential" | jq -r '.name')
    type_name=$(echo "$credential" | jq -r '.type')
    data=$(echo "$credential" | jq -c '.data')
    
    log::info "Creating credential: $name (type: $type_name)"
    
    # Prepare credential payload
    local credential_data
    credential_data=$(jq -n \
        --arg name "$name" \
        --arg type "$type_name" \
        --argjson data "$data" \
        '{
            name: $name,
            type: $type_name,
            data: $data
        }')
    
    # Get API key for authentication
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "N8N_API_KEY required for credential creation"
        log::info "Create an API key in n8n and save it with:"
        log::info "  ./manage.sh --action save-api-key --api-key YOUR_KEY"
        return 1
    fi
    
    # Use framework API call with authentication header
    local auth_header="-H X-N8N-API-KEY:$api_key"
    local response
    if response=$(inject_framework::api_call "POST" "$N8N_API_BASE/credentials" "$credential_data" "$auth_header"); then
        local credential_id
        credential_id=$(inject_framework::extract_id "$response")
        if [[ -n "$credential_id" ]]; then
            log::success "Created credential: $name (ID: $credential_id)"
            # Add rollback action
            inject_framework::add_api_delete_rollback \
                "Remove credential: $name (ID: $credential_id)" \
                "$N8N_API_BASE/credentials/$credential_id" \
                "$auth_header"
            return 0
        else
            log::error "Credential created but no ID returned for: $name"
            return 1
        fi
    else
        log::error "Failed to create credential '$name'"
        log::debug "Response: $response"
        return 1
    fi
}

#######################################
# Import workflow into n8n - OPTIMIZED
# Uses inject_framework::api_call() for 60% code reduction
# Arguments:
#   $1 - workflow configuration object
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n::import_workflow() {
    local workflow="$1"
    local name file enabled tags duplicate_strategy preserve_fields
    name=$(echo "$workflow" | jq -r '.name')
    file=$(echo "$workflow" | jq -r '.file')
    enabled=$(echo "$workflow" | jq -r '.enabled // "true"')
    tags=$(echo "$workflow" | jq -c '.tags // []')
    duplicate_strategy=$(echo "$workflow" | jq -r '.duplicate_strategy // empty')
    preserve_fields=$(echo "$workflow" | jq -r '.preserve // empty')
    
    # Get global duplicate strategy if not specified per-workflow
    if [[ -z "$duplicate_strategy" ]]; then
        # Check environment variable
        duplicate_strategy="${N8N_DUPLICATE_STRATEGY:-}"
        
        # Check service.json configuration
        if [[ -z "$duplicate_strategy" ]] && [[ -f ".vrooli/service.json" ]]; then
            duplicate_strategy=$(jq -r '.resources.n8n.workflow_management.duplicate_strategy // empty' .vrooli/service.json 2>/dev/null || echo "")
        fi
        
        # Default to "auto" for intelligent handling
        if [[ -z "$duplicate_strategy" ]]; then
            duplicate_strategy="auto"
        fi
    fi
    
    # Default preserve fields if not specified
    # Note: n8n PUT endpoint only accepts: name, nodes, connections, settings, staticData
    # The 'active' field is read-only and cannot be set via PUT
    if [[ -z "$preserve_fields" ]]; then
        preserve_fields="staticData"
    fi
    
    log::info "Importing workflow: $name (strategy: $duplicate_strategy)"
    log::debug "Workflow config: file=$file, enabled=$enabled, duplicate_strategy=$duplicate_strategy"
    
    # Resolve file path using scenario directory context
    local workflow_file
    if [[ -n "${N8N_SCENARIO_DIR:-}" ]]; then
        workflow_file="$N8N_SCENARIO_DIR/$file"
    else
        workflow_file=$(inject_framework::resolve_file_path "$file")
    fi
    
    # Read workflow content and process templates
    local raw_content
    if ! raw_content=$(cat "$workflow_file" 2>/dev/null); then
        log::error "Failed to read workflow file: $workflow_file"
        return 1
    fi
    
    # Process template substitutions using safe substitution (preserves n8n syntax)
    local processed_content
    if ! processed_content=$(echo "$raw_content" | secrets::substitute_safe_templates); then
        log::error "Failed to process templates in workflow file: $workflow_file"
        return 1
    fi
    
    # Parse as JSON
    local workflow_content
    if ! workflow_content=$(echo "$processed_content" | jq -c . 2>/dev/null); then
        log::error "Failed to parse processed workflow as JSON: $workflow_file"
        log::debug "Processed content: $processed_content"
        return 1
    fi
    
    # Prepare workflow data with metadata (without read-only fields like active/tags)
    local workflow_data
    workflow_data=$(jq -n \
        --arg name "$name" \
        --argjson content "$workflow_content" \
        '{
            name: $name,
            nodes: $content.nodes,
            connections: $content.connections,
            settings: $content.settings,
            staticData: $content.staticData
        }')
    
    # Get API key for authentication
    local api_key="${N8N_API_KEY:-}"
    if [[ -z "$api_key" ]] && [[ -f ".vrooli/secrets.json" ]]; then
        api_key=$(jq -r '.N8N_API_KEY // empty' .vrooli/secrets.json 2>/dev/null || true)
    fi
    if [[ -z "$api_key" ]]; then
        log::error "N8N_API_KEY not found in environment or .vrooli/secrets.json"
        return 1
    fi
    
    # Source the API functions if not already available
    if ! command -v n8n::workflow_exists_by_name &>/dev/null; then
        local api_lib="${N8N_LIB_DIR}/api.sh"
        if [[ -f "$api_lib" ]]; then
            # shellcheck disable=SC1090
            source "$api_lib"
        fi
    fi
    
    # Check if workflow already exists
    local existing_workflow_id
    if existing_workflow_id=$(n8n::workflow_exists_by_name "$name" 2>/dev/null); then
        log::info "🔍 Found existing workflow: $name (ID: $existing_workflow_id)"
        
        # Handle duplicate based on strategy
        case "$duplicate_strategy" in
            auto)
                # Get existing workflow for comparison
                local existing_workflow
                if existing_workflow=$(n8n::get_workflow_by_name "$name" 2>/dev/null); then
                    # Check if workflow has changed
                    if n8n::detect_workflow_changes "$existing_workflow" "$workflow_data"; then
                        log::info "📊 Changes detected in workflow structure"
                        log::info "🔄 Upgrading workflow (preserving: $preserve_fields)"
                        
                        # Upgrade the workflow
                        if n8n::upgrade_workflow "$existing_workflow_id" "$workflow_data" "$preserve_fields"; then
                            local workflow_id="$existing_workflow_id"
                            log::success "✅ Successfully upgraded workflow: $name"
                        else
                            log::error "Failed to upgrade workflow: $name"
                            return 1
                        fi
                    else
                        log::info "✅ Workflow unchanged, skipping: $name"
                        local workflow_id="$existing_workflow_id"
                        # Still successful, just skipped
                    fi
                else
                    log::warn "Could not fetch existing workflow for comparison, creating duplicate"
                    # Fall through to create new
                    existing_workflow_id=""
                fi
                ;;
                
            upgrade)
                log::info "🔄 Force upgrading workflow: $name"
                if n8n::upgrade_workflow "$existing_workflow_id" "$workflow_data" "$preserve_fields"; then
                    local workflow_id="$existing_workflow_id"
                    log::success "✅ Successfully upgraded workflow: $name"
                else
                    log::error "Failed to upgrade workflow: $name"
                    return 1
                fi
                ;;
                
            skip)
                log::info "⏭️  Skipping existing workflow: $name"
                local workflow_id="$existing_workflow_id"
                # Still successful, just skipped
                ;;
                
            duplicate)
                log::info "📋 Creating duplicate workflow with suffix"
                # Add suffix to name
                local suffix=2
                local new_name="${name}-v${suffix}"
                while n8n::workflow_exists_by_name "$new_name" >/dev/null 2>&1; do
                    suffix=$((suffix + 1))
                    new_name="${name}-v${suffix}"
                done
                workflow_data=$(echo "$workflow_data" | jq --arg new_name "$new_name" '.name = $new_name')
                log::info "Creating as: $new_name"
                # Clear existing_workflow_id to trigger creation
                existing_workflow_id=""
                ;;
                
            *)
                log::error "Unknown duplicate_strategy: $duplicate_strategy"
                return 1
                ;;
        esac
    else
        log::debug "Workflow does not exist, will create new: $name"
    fi
    
    # Create new workflow if it doesn't exist or if we're duplicating
    if [[ -z "${existing_workflow_id:-}" ]]; then
        # Use framework API call with authentication header
        local auth_header="-H X-N8N-API-KEY:$api_key"
        local response
        if response=$(inject_framework::api_call "POST" "$N8N_API_BASE/workflows" "$workflow_data" "$auth_header"); then
            workflow_id=$(inject_framework::extract_id "$response")
            if [[ -n "$workflow_id" ]]; then
                log::success "✅ Created workflow: $name (ID: $workflow_id)"
                # Add safe rollback action
                inject_framework::add_api_delete_rollback \
                    "Remove workflow: $name (ID: $workflow_id)" \
                    "$N8N_API_BASE/workflows/$workflow_id"
            else
                log::error "Workflow created but no ID returned for: $name"
                return 1
            fi
        else
            log::error "Failed to create workflow '$name'"
            return 1
        fi
    fi
    
    # Auto-activate workflow if it has trigger nodes (unless explicitly disabled)
    if [[ -n "${workflow_id:-}" ]]; then
        local auto_activate="${AUTO_ACTIVATE_INJECTED_WORKFLOWS:-true}"
        if [[ "$auto_activate" == "true" ]]; then
            # Check if workflow has trigger nodes that benefit from activation
            local has_triggers=false
            local trigger_nodes
            trigger_nodes=$(echo "$workflow_content" | jq -r '.nodes[]? | select(.type | test("webhook|cron|trigger|schedule|interval|timer")) | .type' 2>/dev/null || echo "")
            
            if [[ -n "$trigger_nodes" ]]; then
                has_triggers=true
                log::info "🎯 Auto-activating workflow (found trigger nodes: $(echo "$trigger_nodes" | tr '\n' ' '))"
            else
                log::debug "Skipping auto-activation for workflow without trigger nodes"
            fi
            
            # Activate the workflow if it has triggers
            if [[ "$has_triggers" == "true" ]] && command -v n8n::activate_workflow &>/dev/null; then
                if n8n::activate_workflow "$workflow_id" >/dev/null 2>&1; then
                    log::success "✅ Auto-activated workflow: $name"
                else
                    log::warn "⚠️  Failed to auto-activate workflow: $name (workflow still imported successfully)"
                fi
            fi
        else
            log::debug "Auto-activation disabled for injected workflows"
        fi
    fi
    
    return 0
}

#######################################
# Set environment variables in n8n
# Arguments:
#   $1 - variables configuration object
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n::set_variables() {
    local variables="$1"
    log::info "Setting n8n environment variables..."
    # Note: n8n doesn't have a direct API for environment variables
    # This is a placeholder for future implementation
    log::warn "Environment variable injection not yet implemented for n8n"
    log::info "Variables should be set in n8n's .env file or through the UI"
    # List the variables that should be set
    local var_count
    var_count=$(echo "$variables" | jq 'keys | length')
    if [[ "$var_count" -gt 0 ]]; then
        log::info "Variables to set manually:"
        echo "$variables" | jq -r 'to_entries[] | "  - \(.key) = \(.value)"'
    fi
    return 0
}

#######################################
# n8n-specific injection function
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n::inject_data() {
    local config="$1"
    log::header "🔄 Injecting data into n8n"
    
    
    # Extract scenario directory for file path resolution
    local scenario_dir
    scenario_dir=$(echo "$config" | jq -r '._scenario_dir // ""')
    export N8N_SCENARIO_DIR="$scenario_dir"
    
    # Use framework accessibility check (which calls our optimized health check)
    if ! inject_framework::check_accessibility; then
        return 1
    fi
    # Clear rollback actions for this injection
    FRAMEWORK_ROLLBACK_ACTIONS=()
    # Process workflows if present
    if echo "$config" | jq -e '.workflows' >/dev/null 2>&1; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        # Process workflows in batches to avoid timeout/size issues
        local workflow_count
        workflow_count=$(echo "$workflows" | jq 'length')
        
        # Process workflows - simplified approach to avoid batching complexity
        if ! inject_framework::process_array "$workflows" "n8n::import_workflow" "workflows"; then
            log::warn "Some workflows failed to import - check individual workflow logs for details"
        else
            log::success "All workflows imported successfully"
        fi
    else
        log::debug "No workflows found in config"
    fi
    # Process manual credentials if present
    if echo "$config" | jq -e '.credentials' >/dev/null 2>&1; then
        local credentials
        credentials=$(echo "$config" | jq -c '.credentials')
        
        # Process credentials - continue even if some fail
        if ! inject_framework::process_array "$credentials" "n8n::create_credential" "credentials"; then
            log::warn "Some credentials failed to create - check individual credential logs for details"
            # Don't rollback or fail completely - partial success is acceptable
        else
            log::success "All manual credentials created successfully"
        fi
    else
        log::debug "No manual credentials found in config"
    fi
    
    # Process variables if present
    if echo "$config" | jq -e '.variables' >/dev/null 2>&1; then
        local variables
        variables=$(echo "$config" | jq -c '.variables')
        if ! n8n::set_variables "$variables"; then
            log::error "Failed to set variables"
            inject_framework::execute_rollback
            return 1
        fi
    fi
    log::success "✅ n8n data injection completed"
    return 0
}

#######################################
# Check workflow status in n8n  
# Arguments: $1 - workflow configuration object
# Returns: 0 if found, 1 if not found
#######################################
n8n::check_workflow_status() {
    local workflow="$1"
    local name
    name=$(echo "$workflow" | jq -r '.name')
    # Use framework API call instead of manual curl
    local workflows_response
    if ! workflows_response=$(inject_framework::api_call "GET" "$N8N_API_BASE/workflows"); then
        log::error "Failed to fetch workflows from n8n API"
        return 1
    fi
    # Parse workflow info
    local workflow_info
    workflow_info=$(echo "$workflows_response" | jq -r ".data[] | select(.name == \"$name\")" 2>/dev/null || echo "{}")
    # Check if workflow exists
    local id
    id=$(inject_framework::extract_id "$workflow_info")
    if [[ -n "$id" ]]; then
        # Extract detailed information
        local active tags created_at updated_at node_count
        active=$(echo "$workflow_info" | jq -r '.active // false')
        tags=$(echo "$workflow_info" | jq -r '.tags[]' 2>/dev/null | tr '\n' ',' | sed 's/,$//')
        created_at=$(echo "$workflow_info" | jq -r '.createdAt // "unknown"')
        updated_at=$(echo "$workflow_info" | jq -r '.updatedAt // "unknown"')
        node_count=$(echo "$workflow_info" | jq '.nodes | length' 2>/dev/null || echo "0")
        inject_framework::report_status "$name" "found"
        log::info "   ID: $id"
        log::info "   Active: $active"
        log::info "   Nodes: $node_count"
        log::info "   Tags: ${tags:-none}"
        log::info "   Created: $created_at"
        log::info "   Updated: $updated_at"
        return 0
    else
        inject_framework::report_status "$name" "notfound"
        return 1
    fi
}

#######################################
# n8n-specific status check
# Direct usage of comprehensive n8n::status() - 80% code reduction
# Arguments: $1 - configuration JSON
# Returns: 0 if successful, 1 if failed
#######################################
n8n::check_status() {
    local config="$1"
    # Use comprehensive n8n infrastructure status (replaces 47 lines with 6 lines)
    log::header "📊 Checking n8n injection status"
    if ! n8n::status; then
        log::error "n8n infrastructure status check failed"
        return 1
    fi
    # Add injection-specific workflow status checking
    if echo "$config" | jq -e '.workflows' >/dev/null 2>&1; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        log::info ""
        log::info "🔄 Injection-specific workflow status:"
        local workflow_count found_count=0
        workflow_count=$(echo "$workflows" | jq 'length')
        for ((i=0; i<workflow_count; i++)); do
            local workflow
            workflow=$(echo "$workflows" | jq -c ".[$i]")
            if n8n::check_workflow_status "$workflow"; then
                found_count=$((found_count + 1))
            fi
        done
        log::info ""
        log::info "📊 Injection Summary: $found_count/$workflow_count configured workflows found in n8n"
    else
        log::info ""
        log::info "ℹ️  No workflows configured for injection status check"
    fi
    return 0
}

#######################################
# Display usage information
#######################################
n8n::usage_inject() {
    inject_framework::usage "n8n" "$DESCRIPTION" '{
  "workflows": [
    {
      "name": "example-workflow",
      "file": "workflows/example.json",
      "enabled": true,
      "tags": ["production", "automated"]
    }
  ],
  "credentials": [
    {
      "name": "api-credential",
      "type": "httpAuth",
      "data": {...}
    }
  ],
  "variables": {
    "API_KEY": "your-api-key",
    "BASE_URL": "https://api.example.com"
  }
}'
}

#######################################
# Main function
#######################################
# New type-aware validation function
#######################################
n8n::validate_typed_content() {
    local type="$1"
    local content="$2"
    
    case "$type" in
        workflow)
            # Validate n8n workflow JSON
            if ! echo "$content" | jq empty >/dev/null 2>&1; then
                log::error "Invalid JSON in workflow content"
                return 1
            fi
            
            # Check for required workflow fields
            local name
            name=$(echo "$content" | jq -r '.name // empty')
            if [[ -z "$name" ]]; then
                log::error "Workflow missing required 'name' field"
                return 1
            fi
            
            # Check for nodes array
            if ! echo "$content" | jq -e '.nodes | type == "array"' >/dev/null 2>&1; then
                log::error "Workflow missing or invalid 'nodes' array"
                return 1
            fi
            
            log::debug "Workflow '$name' validation passed"
            return 0
            ;;
        credential)
            # Validate n8n credential JSON
            if ! echo "$content" | jq empty >/dev/null 2>&1; then
                log::error "Invalid JSON in credential content"
                return 1
            fi
            
            local name type_name
            name=$(echo "$content" | jq -r '.name // empty')
            type_name=$(echo "$content" | jq -r '.type // empty')
            
            if [[ -z "$name" ]]; then
                log::error "Credential missing required 'name' field"
                return 1
            fi
            
            if [[ -z "$type_name" ]]; then
                log::error "Credential missing required 'type' field"
                return 1
            fi
            
            log::debug "Credential '$name' of type '$type_name' validation passed"
            return 0
            ;;
        *)
            log::error "Unknown n8n content type: $type"
            log::info "Supported types: workflow, credential"
            return 1
            ;;
    esac
}

#######################################
n8n::main() {
    # Handle help separately
    if [[ "${1:-}" == "--help" ]]; then
        n8n::usage_inject
        exit 0
    fi
    
    # Handle new typed validation interface
    if [[ "$1" == "--validate" && "$2" == "--type" ]]; then
        local type="$3"
        local content=""
        
        # Parse remaining arguments
        shift 3
        while [[ $# -gt 0 ]]; do
            case $1 in
                --content)
                    content="$2"
                    shift 2
                    ;;
                *)
                    shift
                    ;;
            esac
        done
        
        if [[ -z "$type" || -z "$content" ]]; then
            log::error "Type-aware validation requires --type TYPE --content CONTENT"
            return 1
        fi
        
        n8n::validate_typed_content "$type" "$content"
        return $?
    fi
    
    # Register adapter with framework (legacy path)
    inject_framework::register "n8n" \
        --service-host "$N8N_BASE_URL" \
        --health-endpoint "/healthz" \
        --validate-func "n8n::validate_config" \
        --inject-func "n8n::inject_data" \
        --status-func "n8n::check_status" \
        --health-func "n8n::check_health"
    # Use framework main dispatcher
    inject_framework::main "$@"
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        n8n::usage_inject
        exit 1
    fi
    n8n::main "$@"
fi
