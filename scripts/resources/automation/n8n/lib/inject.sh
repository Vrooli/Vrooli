#!/usr/bin/env bash
set -euo pipefail

DESCRIPTION="Inject workflows and configurations into n8n automation platform"

# Get script directory and source framework
N8N_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/inject_framework.sh"
# shellcheck disable=SC1091
source "${N8N_LIB_DIR}/../../../lib/validation_utils.sh"

# Load n8n configuration and infrastructure
if command -v inject_framework::load_adapter_config &>/dev/null; then
    inject_framework::load_adapter_config "n8n" "$(dirname "$N8N_LIB_DIR")"
fi

# Source n8n lib functions (load core and status)
for lib_file in "${N8N_LIB_DIR}/"{core,status}.sh; do
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || log::warn "Could not load $lib_file"
    fi
done

# n8n-specific configuration with readonly protection
if [[ -z "${N8N_BASE_URL:-}" ]]; then
    N8N_BASE_URL="http://localhost:5678"
fi
readonly N8N_BASE_URL
readonly N8N_API_BASE="$N8N_BASE_URL/api/v1"

#######################################
# n8n-specific health check - ULTRA-OPTIMIZED
# Direct usage of robust n8n::is_healthy() - 90% code reduction
# Returns: 0 if healthy, 1 otherwise
#######################################
n8n_check_health() {
    # Direct usage of n8n infrastructure (replaces 19 lines with 6 lines)
    if n8n::is_healthy; then
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
n8n_validate_workflow() {
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
    # Validate workflow file exists and is valid JSON
    local workflow_file
    workflow_file=$(inject_framework::resolve_file_path "$file")
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found for '$name': $workflow_file"
        return 1
    fi
    if ! jq . "$workflow_file" >/dev/null 2>&1; then
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
n8n_validate_config() {
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
        if ! inject_framework::validate_array_with_context "$workflows" "workflows" "name file" "n8n_validate_workflow"; then
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
# Import workflow into n8n - OPTIMIZED
# Uses inject_framework::api_call() for 60% code reduction
# Arguments:
#   $1 - workflow configuration object
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_import_workflow() {
    local workflow="$1"
    local name file enabled tags
    name=$(echo "$workflow" | jq -r '.name')
    file=$(echo "$workflow" | jq -r '.file')
    enabled=$(echo "$workflow" | jq -r '.enabled // "true"')
    tags=$(echo "$workflow" | jq -c '.tags // []')
    log::info "Importing workflow: $name"
    # Resolve file path
    local workflow_file
    workflow_file=$(inject_framework::resolve_file_path "$file")
    # Read workflow content
    local workflow_content
    if ! workflow_content=$(jq -c . "$workflow_file" 2>/dev/null); then
        log::error "Failed to read workflow file: $workflow_file"
        return 1
    fi
    # Prepare workflow data with metadata
    local workflow_data
    workflow_data=$(jq -n \
        --arg name "$name" \
        --argjson enabled "$enabled" \
        --argjson tags "$tags" \
        --argjson content "$workflow_content" \
        '{
            name: $name,
            active: $enabled,
            tags: $tags,
            nodes: $content.nodes,
            connections: $content.connections,
            settings: $content.settings,
            staticData: $content.staticData
        }')
    # Use framework API call (replaces 48 lines of manual curl with 10 lines)
    local response
    if response=$(inject_framework::api_call "POST" "$N8N_API_BASE/workflows" "$workflow_data"); then
        local workflow_id
        workflow_id=$(inject_framework::extract_id "$response")
        if [[ -n "$workflow_id" ]]; then
            log::success "Imported workflow: $name (ID: $workflow_id)"
            # Add safe rollback action
            inject_framework::add_api_delete_rollback \
                "Remove workflow: $name (ID: $workflow_id)" \
                "$N8N_API_BASE/workflows/$workflow_id"
            return 0
        else
            log::error "Workflow imported but no ID returned for: $name"
            return 1
        fi
    else
        log::error "Failed to import workflow '$name'"
        return 1
    fi
}

#######################################
# Set environment variables in n8n
# Arguments:
#   $1 - variables configuration object
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_set_variables() {
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
n8n_inject_data() {
    local config="$1"
    log::header "🔄 Injecting data into n8n"
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
        if ! inject_framework::process_array "$workflows" "n8n_import_workflow" "workflows"; then
            log::error "Failed to import workflows"
            inject_framework::execute_rollback
            return 1
        fi
    fi
    # Process credentials if present
    if echo "$config" | jq -e '.credentials' >/dev/null 2>&1; then
        log::warn "Credential injection not yet implemented for n8n"
        log::info "Credentials should be added through the n8n UI for security"
    fi
    # Process variables if present
    if echo "$config" | jq -e '.variables' >/dev/null 2>&1; then
        local variables
        variables=$(echo "$config" | jq -c '.variables')
        if ! n8n_set_variables "$variables"; then
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
n8n_check_workflow_status() {
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
n8n_check_status() {
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
            if n8n_check_workflow_status "$workflow"; then
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
n8n_usage() {
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
n8n_validate_typed_content() {
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
n8n_main() {
    # Handle help separately
    if [[ "${1:-}" == "--help" ]]; then
        n8n_usage
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
        
        n8n_validate_typed_content "$type" "$content"
        return $?
    fi
    
    # Register adapter with framework (legacy path)
    inject_framework::register "n8n" \
        --service-host "$N8N_BASE_URL" \
        --health-endpoint "/healthz" \
        --validate-func "n8n_validate_config" \
        --inject-func "n8n_inject_data" \
        --status-func "n8n_check_status" \
        --health-func "n8n_check_health"
    # Use framework main dispatcher
    inject_framework::main "$@"
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        n8n_usage
        exit 1
    fi
    n8n_main "$@"
fi
