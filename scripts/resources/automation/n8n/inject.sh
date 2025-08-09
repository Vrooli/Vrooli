#!/usr/bin/env bash
set -euo pipefail

# n8n Data Injection Adapter
# This script handles injection of workflows, credentials, and configurations into n8n
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject workflows and credentials into n8n workflow automation platform"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source n8n configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
    
    # Export n8n configuration if function exists
    if declare -f n8n::export_config >/dev/null 2>&1; then
        n8n::export_config 2>/dev/null || true
    fi
fi

# Default n8n API settings (check if already set to avoid readonly conflicts)
if [[ -z "${N8N_BASE_URL:-}" ]]; then
    N8N_BASE_URL="http://localhost:5678"
fi
readonly N8N_API_BASE="$N8N_BASE_URL/api/v1"

# Operation tracking
declare -a N8N_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
n8n_inject::usage() {
    cat << EOF
n8n Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects workflows, credentials, and configurations into n8n based on
    scenario configuration. Supports validation, injection, status checks,
    and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    The configuration should be a JSON object with the following structure:
    {
      "workflows": [
        {
          "name": "workflow-name",
          "file": "path/to/workflow.json",
          "enabled": true,
          "tags": ["tag1", "tag2"]
        }
      ],
      "credentials": [
        {
          "name": "credential-name", 
          "type": "httpAuth",
          "config": {
            "url": "http://example.com",
            "authType": "bearer"
          }
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"workflows": [{"name": "test", "file": "test.json"}]}'
    
    # Inject data
    $0 --inject '{"workflows": [{"name": "test", "file": "test.json", "enabled": true}]}'
    
    # Check injection status
    $0 --status '{"workflows": [{"name": "test"}]}'

EOF
}

#######################################
# Check if n8n is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
n8n_inject::check_accessibility() {
    if ! system::is_command "curl"; then
        log::error "curl command not available"
        return 1
    fi
    
    local health_check
    if health_check=$(curl -s --max-time 5 "$N8N_BASE_URL/healthz" 2>/dev/null); then
        if [[ "$health_check" == *"ok"* ]] || [[ "$health_check" == *"healthy"* ]]; then
            log::debug "n8n is accessible at $N8N_BASE_URL"
            return 0
        fi
    fi
    
    log::error "n8n is not accessible at $N8N_BASE_URL"
    log::info "Ensure n8n is running: ./scripts/resources/automation/n8n/manage.sh --action start"
    return 1
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
n8n_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    N8N_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added n8n rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
n8n_inject::execute_rollback() {
    if [[ ${#N8N_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No n8n rollback actions to execute"
        return 0
    fi
    
    log::info "Executing n8n rollback actions..."
    
    local success_count=0
    local total_count=${#N8N_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#N8N_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${N8N_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "n8n rollback completed: $success_count/$total_count actions successful"
    N8N_ROLLBACK_ACTIONS=()
}

#######################################
# Validate workflow configuration
# Arguments:
#   $1 - workflow configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
n8n_inject::validate_workflows() {
    local workflows_config="$1"
    
    log::debug "Validating workflow configurations..."
    
    # Check if workflows is an array
    local workflows_type
    workflows_type=$(echo "$workflows_config" | jq -r 'type')
    
    if [[ "$workflows_type" != "array" ]]; then
        log::error "Workflows configuration must be an array, got: $workflows_type"
        return 1
    fi
    
    # Validate each workflow
    local workflow_count
    workflow_count=$(echo "$workflows_config" | jq 'length')
    
    for ((i=0; i<workflow_count; i++)); do
        local workflow
        workflow=$(echo "$workflows_config" | jq -c ".[$i]")
        
        # Check required fields
        local name file
        name=$(echo "$workflow" | jq -r '.name // empty')
        file=$(echo "$workflow" | jq -r '.file // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Workflow at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Workflow '$name' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists (resolve relative to Vrooli root)
        local workflow_file="$var_ROOT_DIR/$file"
        if [[ ! -f "$workflow_file" ]]; then
            log::error "Workflow file not found: $workflow_file"
            return 1
        fi
        
        # Validate workflow file is valid JSON
        if ! jq . "$workflow_file" >/dev/null 2>&1; then
            log::error "Workflow file contains invalid JSON: $workflow_file"
            return 1
        fi
        
        log::debug "Workflow '$name' configuration is valid"
    done
    
    log::success "All workflow configurations are valid"
    return 0
}

#######################################
# Validate credential configuration
# Arguments:
#   $1 - credentials configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
n8n_inject::validate_credentials() {
    local credentials_config="$1"
    
    log::debug "Validating credential configurations..."
    
    # Check if credentials is an array
    local credentials_type
    credentials_type=$(echo "$credentials_config" | jq -r 'type')
    
    if [[ "$credentials_type" != "array" ]]; then
        log::error "Credentials configuration must be an array, got: $credentials_type"
        return 1
    fi
    
    # Validate each credential
    local credential_count
    credential_count=$(echo "$credentials_config" | jq 'length')
    
    for ((i=0; i<credential_count; i++)); do
        local credential
        credential=$(echo "$credentials_config" | jq -c ".[$i]")
        
        # Check required fields
        local name type
        name=$(echo "$credential" | jq -r '.name // empty')
        type=$(echo "$credential" | jq -r '.type // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Credential at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$type" ]]; then
            log::error "Credential '$name' missing required 'type' field"
            return 1
        fi
        
        # Validate credential type
        case "$type" in
            httpAuth|apiKey|oauth2|basic|bearer)
                log::debug "Credential '$name' has valid type: $type"
                ;;
            *)
                log::error "Credential '$name' has unsupported type: $type"
                return 1
                ;;
        esac
    done
    
    log::success "All credential configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
n8n_inject::validate_config() {
    local config="$1"
    
    log::info "Validating n8n injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in n8n injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_workflows has_credentials
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    has_credentials=$(echo "$config" | jq -e '.credentials' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "false" && "$has_credentials" == "false" ]]; then
        log::error "n8n injection configuration must have 'workflows' or 'credentials'"
        return 1
    fi
    
    # Validate workflows if present
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        if ! n8n_inject::validate_workflows "$workflows"; then
            return 1
        fi
    fi
    
    # Validate credentials if present
    if [[ "$has_credentials" == "true" ]]; then
        local credentials
        credentials=$(echo "$config" | jq -c '.credentials')
        
        if ! n8n_inject::validate_credentials "$credentials"; then
            return 1
        fi
    fi
    
    log::success "n8n injection configuration is valid"
    return 0
}

#######################################
# Import workflow into n8n
# Arguments:
#   $1 - workflow configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_inject::import_workflow() {
    local workflow_config="$1"
    
    local name file enabled
    name=$(echo "$workflow_config" | jq -r '.name')
    file=$(echo "$workflow_config" | jq -r '.file')
    enabled=$(echo "$workflow_config" | jq -r '.enabled // true')
    
    log::info "Importing workflow: $name"
    
    # Resolve file path
    local workflow_file="$var_ROOT_DIR/$file"
    
    # Check if workflow already exists
    local existing_workflow
    existing_workflow=$(curl -s "$N8N_API_BASE/workflows" | jq -r ".data[] | select(.name == \"$name\") | .id" 2>/dev/null || echo "")
    
    if [[ -n "$existing_workflow" ]]; then
        log::warn "Workflow '$name' already exists with ID: $existing_workflow"
        log::info "Updating existing workflow instead of creating new one"
        
        # Update existing workflow
        local response
        if response=$(curl -s -X PUT \
            -H "Content-Type: application/json" \
            -d @"$workflow_file" \
            "$N8N_API_BASE/workflows/$existing_workflow" 2>/dev/null); then
            
            log::success "Updated workflow: $name"
            
            # Add rollback action
            n8n_inject::add_rollback_action \
                "Restore original workflow: $name" \
                "echo 'Manual restoration required for workflow: $name'"
        else
            log::error "Failed to update workflow: $name"
            return 1
        fi
    else
        # Import new workflow
        local response
        if response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d @"$workflow_file" \
            "$N8N_API_BASE/workflows/import" 2>/dev/null); then
            
            local workflow_id
            workflow_id=$(echo "$response" | jq -r '.id // empty')
            
            if [[ -n "$workflow_id" ]]; then
                log::success "Imported workflow: $name (ID: $workflow_id)"
                
                # Add rollback action
                n8n_inject::add_rollback_action \
                    "Remove workflow: $name" \
                    "curl -s -X DELETE '$N8N_API_BASE/workflows/$workflow_id' >/dev/null 2>&1"
            else
                log::error "Failed to get workflow ID for: $name"
                return 1
            fi
        else
            log::error "Failed to import workflow: $name"
            return 1
        fi
    fi
    
    # Activate workflow if enabled
    if [[ "$enabled" == "true" ]]; then
        local workflow_id
        workflow_id=$(curl -s "$N8N_API_BASE/workflows" | jq -r ".data[] | select(.name == \"$name\") | .id" 2>/dev/null || echo "")
        
        if [[ -n "$workflow_id" ]]; then
            if curl -s -X POST "$N8N_API_BASE/workflows/$workflow_id/activate" >/dev/null 2>&1; then
                log::info "Activated workflow: $name"
                
                # Add rollback action
                n8n_inject::add_rollback_action \
                    "Deactivate workflow: $name" \
                    "curl -s -X POST '$N8N_API_BASE/workflows/$workflow_id/deactivate' >/dev/null 2>&1"
            else
                log::warn "Failed to activate workflow: $name"
            fi
        fi
    fi
    
    return 0
}

#######################################
# Inject workflows into n8n
# Arguments:
#   $1 - workflows configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_inject::inject_workflows() {
    local workflows_config="$1"
    
    log::info "Injecting workflows into n8n..."
    
    local workflow_count
    workflow_count=$(echo "$workflows_config" | jq 'length')
    
    if [[ "$workflow_count" -eq 0 ]]; then
        log::info "No workflows to inject"
        return 0
    fi
    
    local failed_workflows=()
    
    for ((i=0; i<workflow_count; i++)); do
        local workflow
        workflow=$(echo "$workflows_config" | jq -c ".[$i]")
        
        local workflow_name
        workflow_name=$(echo "$workflow" | jq -r '.name')
        
        if ! n8n_inject::import_workflow "$workflow"; then
            failed_workflows+=("$workflow_name")
        fi
    done
    
    if [[ ${#failed_workflows[@]} -eq 0 ]]; then
        log::success "All workflows injected successfully"
        return 0
    else
        log::error "Failed to inject workflows: ${failed_workflows[*]}"
        return 1
    fi
}

#######################################
# Inject credentials into n8n
# Arguments:
#   $1 - credentials configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_inject::inject_credentials() {
    local credentials_config="$1"
    
    log::info "Injecting credentials into n8n..."
    
    local credential_count
    credential_count=$(echo "$credentials_config" | jq 'length')
    
    if [[ "$credential_count" -eq 0 ]]; then
        log::info "No credentials to inject"
        return 0
    fi
    
    log::warn "Credential injection not yet implemented"
    log::info "This requires n8n API authentication and proper credential handling"
    log::info "Manually configure credentials in n8n UI for now"
    
    # TODO: Implement credential injection
    # This requires:
    # 1. n8n API authentication setup
    # 2. Proper credential encryption/decryption
    # 3. Credential type-specific handling
    
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into n8n"
    
    # Check n8n accessibility
    if ! n8n_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    N8N_ROLLBACK_ACTIONS=()
    
    # Inject workflows if present
    local has_workflows
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        if ! n8n_inject::inject_workflows "$workflows"; then
            log::error "Failed to inject workflows"
            n8n_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject credentials if present
    local has_credentials
    has_credentials=$(echo "$config" | jq -e '.credentials' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_credentials" == "true" ]]; then
        local credentials
        credentials=$(echo "$config" | jq -c '.credentials')
        
        if ! n8n_inject::inject_credentials "$credentials"; then
            log::warn "Credential injection failed (not fully implemented)"
            # Don't fail the entire injection for credential issues
        fi
    fi
    
    log::success "✅ n8n data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
n8n_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking n8n injection status"
    
    # Check n8n accessibility
    if ! n8n_inject::check_accessibility; then
        return 1
    fi
    
    # Check workflow status
    local has_workflows
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        log::info "Checking workflow status..."
        
        local workflow_count
        workflow_count=$(echo "$workflows" | jq 'length')
        
        for ((i=0; i<workflow_count; i++)); do
            local workflow
            workflow=$(echo "$workflows" | jq -c ".[$i]")
            
            local name
            name=$(echo "$workflow" | jq -r '.name')
            
            # Check if workflow exists in n8n
            local existing_workflow
            existing_workflow=$(curl -s "$N8N_API_BASE/workflows" | jq -r ".data[] | select(.name == \"$name\") | .id" 2>/dev/null || echo "")
            
            if [[ -n "$existing_workflow" ]]; then
                log::success "✅ Workflow '$name' found (ID: $existing_workflow)"
            else
                log::error "❌ Workflow '$name' not found"
            fi
        done
    fi
    
    # Note: Credential status checking would require API authentication
    log::info "Credential status checking requires n8n API authentication"
    
    return 0
}

#######################################
# Main execution function
#######################################
n8n_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        n8n_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            n8n_inject::validate_config "$config"
            ;;
        "--inject")
            n8n_inject::inject_data "$config"
            ;;
        "--status")
            n8n_inject::check_status "$config"
            ;;
        "--rollback")
            n8n_inject::execute_rollback
            ;;
        "--help")
            n8n_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            n8n_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        n8n_inject::usage
        exit 1
    fi
    
    n8n_inject::main "$@"
fi