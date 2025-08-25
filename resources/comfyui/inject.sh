#!/usr/bin/env bash
set -euo pipefail

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="$APP_ROOT/resources/comfyui"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# ComfyUI Workflow Injection Adapter
# This script handles injection of workflows and models into ComfyUI
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject workflows and models into ComfyUI image generation platform"

SCRIPT_DIR="$APP_ROOT/resources/comfyui"

# Source var.sh first to get all path variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source ComfyUI configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default ComfyUI settings
readonly DEFAULT_COMFYUI_HOST="http://localhost:8188"
readonly DEFAULT_COMFYUI_MODELS_DIR="/app/models"
readonly DEFAULT_COMFYUI_WORKFLOWS_DIR="/app/workflows"

# ComfyUI settings (can be overridden by environment)
COMFYUI_HOST="${COMFYUI_HOST:-$DEFAULT_COMFYUI_HOST}"
COMFYUI_MODELS_DIR="${COMFYUI_MODELS_DIR:-$DEFAULT_COMFYUI_MODELS_DIR}"
COMFYUI_WORKFLOWS_DIR="${COMFYUI_WORKFLOWS_DIR:-$DEFAULT_COMFYUI_WORKFLOWS_DIR}"

# Operation tracking
declare -a COMFYUI_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
inject::usage() {
    cat << EOF
ComfyUI Workflow Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects workflows and models into ComfyUI based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the workflow/model injection
    --status      Check status of injected workflows
    --rollback    Rollback injected workflows
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "workflows": [
        {
          "name": "text-to-image",
          "file": "path/to/workflow.json",
          "description": "Basic text to image workflow"
        }
      ],
      "models": [
        {
          "type": "checkpoint",
          "name": "sdxl_base",
          "url": "https://example.com/model.safetensors",
          "directory": "checkpoints"
        }
      ],
      "custom_nodes": [
        {
          "name": "ComfyUI-Manager",
          "git_url": "https://github.com/ltdrdata/ComfyUI-Manager"
        }
      ],
      "settings": [
        {
          "key": "auto_queue",
          "value": true
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"workflows": [{"name": "test", "file": "workflows/test.json"}]}'
    
    # Inject workflows and models
    $0 --inject '{"workflows": [{"name": "img2img", "file": "configs/img2img.json"}], "models": [{"type": "vae", "name": "vae-ft", "url": "https://example.com/vae.safetensors"}]}'

EOF
}

#######################################
# Check if ComfyUI is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
inject::check_accessibility() {
    # Check if ComfyUI API is accessible
    if curl -s --max-time 5 "${COMFYUI_HOST}/system_stats" >/dev/null 2>&1; then
        log::debug "ComfyUI is accessible at $COMFYUI_HOST"
        return 0
    else
        log::error "ComfyUI is not accessible at $COMFYUI_HOST"
        log::info "Ensure ComfyUI is running: ./resources/comfyui/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    COMFYUI_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added ComfyUI rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
inject::execute_rollback() {
    if [[ ${#COMFYUI_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No ComfyUI rollback actions to execute"
        return 0
    fi
    
    log::info "Executing ComfyUI rollback actions..."
    
    local success_count=0
    local total_count=${#COMFYUI_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#COMFYUI_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${COMFYUI_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "ComfyUI rollback completed: $success_count/$total_count actions successful"
    COMFYUI_ROLLBACK_ACTIONS=()
}

#######################################
# Validate workflows configuration
# Arguments:
#   $1 - workflows configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_workflows() {
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
        
        # Check if workflow file exists
        local workflow_path="$VROOLI_PROJECT_ROOT/$file"
        if [[ ! -f "$workflow_path" ]]; then
            log::error "Workflow file not found: $workflow_path"
            return 1
        fi
        
        # Validate workflow file is valid JSON
        if ! jq . "$workflow_path" >/dev/null 2>&1; then
            log::error "Workflow file is not valid JSON: $workflow_path"
            return 1
        fi
        
        log::debug "Workflow '$name' configuration is valid"
    done
    
    log::success "All workflow configurations are valid"
    return 0
}

#######################################
# Validate models configuration
# Arguments:
#   $1 - models configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_models() {
    local models_config="$1"
    
    log::debug "Validating model configurations..."
    
    # Check if models is an array
    local models_type
    models_type=$(echo "$models_config" | jq -r 'type')
    
    if [[ "$models_type" != "array" ]]; then
        log::error "Models configuration must be an array, got: $models_type"
        return 1
    fi
    
    # Validate each model
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        # Check required fields
        local type name
        type=$(echo "$model" | jq -r '.type // empty')
        name=$(echo "$model" | jq -r '.name // empty')
        
        if [[ -z "$type" ]]; then
            log::error "Model at index $i missing required 'type' field"
            return 1
        fi
        
        if [[ -z "$name" ]]; then
            log::error "Model at index $i missing required 'name' field"
            return 1
        fi
        
        # Validate model type
        case "$type" in
            checkpoint|vae|lora|controlnet|upscaler|embeddings)
                log::debug "Model '$name' has valid type: $type"
                ;;
            *)
                log::error "Model '$name' has invalid type: $type"
                return 1
                ;;
        esac
        
        # Check for either URL or local file
        local url file
        url=$(echo "$model" | jq -r '.url // empty')
        file=$(echo "$model" | jq -r '.file // empty')
        
        if [[ -z "$url" && -z "$file" ]]; then
            log::error "Model '$name' must have either 'url' or 'file' field"
            return 1
        fi
        
        # If local file, check it exists
        if [[ -n "$file" ]]; then
            local file_path="$VROOLI_PROJECT_ROOT/$file"
            if [[ ! -f "$file_path" ]]; then
                log::error "Model file not found: $file_path"
                return 1
            fi
        fi
        
        log::debug "Model '$name' configuration is valid"
    done
    
    log::success "All model configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_config() {
    local config="$1"
    
    log::info "Validating ComfyUI injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in ComfyUI injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_workflows has_models has_custom_nodes has_settings
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    has_custom_nodes=$(echo "$config" | jq -e '.custom_nodes' >/dev/null 2>&1 && echo "true" || echo "false")
    has_settings=$(echo "$config" | jq -e '.settings' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "false" && "$has_models" == "false" && "$has_custom_nodes" == "false" && "$has_settings" == "false" ]]; then
        log::error "ComfyUI injection configuration must have 'workflows', 'models', 'custom_nodes', or 'settings'"
        return 1
    fi
    
    # Validate workflows if present
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        if ! inject::validate_workflows "$workflows"; then
            return 1
        fi
    fi
    
    # Validate models if present
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! inject::validate_models "$models"; then
            return 1
        fi
    fi
    
    log::success "ComfyUI injection configuration is valid"
    return 0
}

#######################################
# Upload workflow to ComfyUI
# Arguments:
#   $1 - workflow configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::upload_workflow() {
    local workflow_config="$1"
    
    local name file description
    name=$(echo "$workflow_config" | jq -r '.name')
    file=$(echo "$workflow_config" | jq -r '.file')
    description=$(echo "$workflow_config" | jq -r '.description // ""')
    
    # Resolve workflow file path
    local workflow_path="$VROOLI_PROJECT_ROOT/$file"
    
    log::info "Uploading workflow: $name"
    
    # Read workflow content
    local workflow_content
    workflow_content=$(cat "$workflow_path")
    
    # Save workflow via ComfyUI API
    local api_format=$(jq -n \
        --arg name "$name" \
        --arg description "$description" \
        --argjson workflow "$workflow_content" \
        '{
            "name": $name,
            "description": $description,
            "workflow": $workflow
        }')
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$api_format" \
        "${COMFYUI_HOST}/api/workflows" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Uploaded workflow: $name"
        
        # Add rollback action
        inject::add_rollback_action \
            "Remove workflow: $name" \
            "curl -s -X DELETE '${COMFYUI_HOST}/api/workflows/${name}' >/dev/null 2>&1"
        
        return 0
    else
        log::error "Failed to upload workflow: $name"
        return 1
    fi
}

#######################################
# Download and install model
# Arguments:
#   $1 - model configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::install_model() {
    local model_config="$1"
    
    local type name url file directory
    type=$(echo "$model_config" | jq -r '.type')
    name=$(echo "$model_config" | jq -r '.name')
    url=$(echo "$model_config" | jq -r '.url // empty')
    file=$(echo "$model_config" | jq -r '.file // empty')
    directory=$(echo "$model_config" | jq -r '.directory // empty')
    
    # Determine target directory based on type
    if [[ -z "$directory" ]]; then
        case "$type" in
            checkpoint) directory="checkpoints" ;;
            vae) directory="vae" ;;
            lora) directory="loras" ;;
            controlnet) directory="controlnet" ;;
            upscaler) directory="upscale_models" ;;
            embeddings) directory="embeddings" ;;
            *) directory="custom" ;;
        esac
    fi
    
    local target_dir="${COMFYUI_MODELS_DIR}/${directory}"
    local target_path="${target_dir}/${name}"
    
    log::info "Installing model '$name' (type: $type)"
    
    if [[ -n "$url" ]]; then
        # Download model from URL
        log::info "Downloading model from: $url"
        
        if curl -s -L -o "$target_path" "$url"; then
            log::success "Downloaded model: $name"
            
            # Add rollback action
            inject::add_rollback_action \
                "Remove model: $name" \
                "rm -f '${target_path}'"
            
            return 0
        else
            log::error "Failed to download model: $name"
            return 1
        fi
    elif [[ -n "$file" ]]; then
        # Copy local model file
        local source_path="$VROOLI_PROJECT_ROOT/$file"
        
        log::info "Copying model from: $file"
        
        if cp "$source_path" "$target_path"; then
            log::success "Copied model: $name"
            
            # Add rollback action
            inject::add_rollback_action \
                "Remove model: $name" \
                "rm -f '${target_path}'"
            
            return 0
        else
            log::error "Failed to copy model: $name"
            return 1
        fi
    else
        log::error "Model '$name' has no source (url or file)"
        return 1
    fi
}

#######################################
# Install custom node
# Arguments:
#   $1 - custom node configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::install_custom_node() {
    local node_config="$1"
    
    local name git_url
    name=$(echo "$node_config" | jq -r '.name')
    git_url=$(echo "$node_config" | jq -r '.git_url')
    
    log::info "Installing custom node: $name"
    
    # Clone custom node repository
    local custom_nodes_dir="/app/custom_nodes"
    local target_dir="${custom_nodes_dir}/${name}"
    
    if git clone "$git_url" "$target_dir" 2>/dev/null; then
        log::success "Installed custom node: $name"
        
        # Install requirements if present
        if [[ -f "${target_dir}/requirements.txt" ]]; then
            log::info "Installing requirements for: $name"
            pip install -r "${target_dir}/requirements.txt" >/dev/null 2>&1
        fi
        
        # Add rollback action
        inject::add_rollback_action \
            "Remove custom node: $name" \
            "trash::safe_remove '${target_dir}' --no-confirm"
        
        return 0
    else
        # Check if already installed
        if [[ -d "$target_dir" ]]; then
            log::warn "Custom node '$name' already installed"
            return 0
        else
            log::error "Failed to install custom node: $name"
            return 1
        fi
    fi
}

#######################################
# Apply settings
# Arguments:
#   $1 - settings configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::apply_settings() {
    local settings_config="$1"
    
    log::info "Applying ComfyUI settings..."
    
    local setting_count
    setting_count=$(echo "$settings_config" | jq 'length')
    
    if [[ "$setting_count" -eq 0 ]]; then
        log::info "No settings to apply"
        return 0
    fi
    
    # Note: ComfyUI settings API may vary
    # This is a placeholder implementation
    for ((i=0; i<setting_count; i++)); do
        local setting
        setting=$(echo "$settings_config" | jq -c ".[$i]")
        
        local key value
        key=$(echo "$setting" | jq -r '.key')
        value=$(echo "$setting" | jq -r '.value')
        
        log::info "Setting $key = $value"
        
        # Apply setting via API if available
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "{\"$key\": $value}" \
            "${COMFYUI_HOST}/api/settings" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            log::success "Applied setting: $key"
        else
            log::warn "Could not apply setting: $key (API may not support it)"
        fi
    done
    
    return 0
}

#######################################
# Inject workflows
# Arguments:
#   $1 - workflows configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_workflows() {
    local workflows_config="$1"
    
    log::info "Uploading ComfyUI workflows..."
    
    local workflow_count
    workflow_count=$(echo "$workflows_config" | jq 'length')
    
    if [[ "$workflow_count" -eq 0 ]]; then
        log::info "No workflows to upload"
        return 0
    fi
    
    local failed_workflows=()
    
    for ((i=0; i<workflow_count; i++)); do
        local workflow
        workflow=$(echo "$workflows_config" | jq -c ".[$i]")
        
        local workflow_name
        workflow_name=$(echo "$workflow" | jq -r '.name')
        
        if ! inject::upload_workflow "$workflow"; then
            failed_workflows+=("$workflow_name")
        fi
    done
    
    if [[ ${#failed_workflows[@]} -eq 0 ]]; then
        log::success "All workflows uploaded successfully"
        return 0
    else
        log::error "Failed to upload workflows: ${failed_workflows[*]}"
        return 1
    fi
}

#######################################
# Inject models
# Arguments:
#   $1 - models configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_models() {
    local models_config="$1"
    
    log::info "Installing ComfyUI models..."
    
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    if [[ "$model_count" -eq 0 ]]; then
        log::info "No models to install"
        return 0
    fi
    
    local failed_models=()
    
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        local model_name
        model_name=$(echo "$model" | jq -r '.name')
        
        if ! inject::install_model "$model"; then
            failed_models+=("$model_name")
        fi
    done
    
    if [[ ${#failed_models[@]} -eq 0 ]]; then
        log::success "All models installed successfully"
        return 0
    else
        log::error "Failed to install models: ${failed_models[*]}"
        return 1
    fi
}

#######################################
# Inject custom nodes
# Arguments:
#   $1 - custom nodes configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_custom_nodes() {
    local nodes_config="$1"
    
    log::info "Installing ComfyUI custom nodes..."
    
    local node_count
    node_count=$(echo "$nodes_config" | jq 'length')
    
    if [[ "$node_count" -eq 0 ]]; then
        log::info "No custom nodes to install"
        return 0
    fi
    
    local failed_nodes=()
    
    for ((i=0; i<node_count; i++)); do
        local node
        node=$(echo "$nodes_config" | jq -c ".[$i]")
        
        local node_name
        node_name=$(echo "$node" | jq -r '.name')
        
        if ! inject::install_custom_node "$node"; then
            failed_nodes+=("$node_name")
        fi
    done
    
    if [[ ${#failed_nodes[@]} -eq 0 ]]; then
        log::success "All custom nodes installed successfully"
        return 0
    else
        log::error "Failed to install custom nodes: ${failed_nodes[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into ComfyUI"
    
    # Check ComfyUI accessibility
    if ! inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    COMFYUI_ROLLBACK_ACTIONS=()
    
    # Inject custom nodes if present (do this first as workflows may depend on them)
    local has_custom_nodes
    has_custom_nodes=$(echo "$config" | jq -e '.custom_nodes' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_custom_nodes" == "true" ]]; then
        local custom_nodes
        custom_nodes=$(echo "$config" | jq -c '.custom_nodes')
        
        if ! inject::inject_custom_nodes "$custom_nodes"; then
            log::error "Failed to install custom nodes"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject models if present
    local has_models
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! inject::inject_models "$models"; then
            log::error "Failed to install models"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject workflows if present
    local has_workflows
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        if ! inject::inject_workflows "$workflows"; then
            log::error "Failed to upload workflows"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply settings if present
    local has_settings
    has_settings=$(echo "$config" | jq -e '.settings' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_settings" == "true" ]]; then
        local settings
        settings=$(echo "$config" | jq -c '.settings')
        
        if ! inject::apply_settings "$settings"; then
            log::warn "Some settings could not be applied"
            # Don't fail on settings as they're optional
        fi
    fi
    
    log::success "✅ ComfyUI data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking ComfyUI injection status"
    
    # Check ComfyUI accessibility
    if ! inject::check_accessibility; then
        return 1
    fi
    
    # Get system stats
    log::info "Fetching ComfyUI system status..."
    
    local stats_response
    stats_response=$(curl -s "${COMFYUI_HOST}/system_stats" 2>/dev/null)
    
    if [[ -n "$stats_response" ]]; then
        local devices vram_free vram_total
        devices=$(echo "$stats_response" | jq -r '.devices // []')
        vram_free=$(echo "$stats_response" | jq -r '.vram_free // 0')
        vram_total=$(echo "$stats_response" | jq -r '.vram_total // 0')
        
        log::info "System Status:"
        log::info "  Devices: $devices"
        log::info "  VRAM: ${vram_free}/${vram_total} MB free"
    fi
    
    # Check workflows status
    local has_workflows
    has_workflows=$(echo "$config" | jq -e '.workflows' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_workflows" == "true" ]]; then
        local workflows
        workflows=$(echo "$config" | jq -c '.workflows')
        
        log::info "Checking workflows status..."
        
        local workflow_count
        workflow_count=$(echo "$workflows" | jq 'length')
        
        for ((i=0; i<workflow_count; i++)); do
            local workflow
            workflow=$(echo "$workflows" | jq -c ".[$i]")
            
            local name
            name=$(echo "$workflow" | jq -r '.name')
            
            # Check if workflow exists (API dependent)
            log::info "  Workflow: $name - status check not implemented (API dependent)"
        done
    fi
    
    # Check models status
    local has_models
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        log::info "Checking models status..."
        
        # Get list of available models
        local models_response
        models_response=$(curl -s "${COMFYUI_HOST}/object_info" 2>/dev/null)
        
        if [[ -n "$models_response" ]]; then
            log::success "✅ Models endpoint accessible"
        else
            log::warn "⚠️  Could not fetch model information"
        fi
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            inject::validate_config "$config"
            ;;
        "--inject")
            inject::inject_data "$config"
            ;;
        "--status")
            inject::check_status "$config"
            ;;
        "--rollback")
            inject::execute_rollback
            ;;
        "--help")
            inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        inject::usage
        exit 1
    fi
    
    inject::main "$@"
fi