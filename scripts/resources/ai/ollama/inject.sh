#!/usr/bin/env bash
set -euo pipefail

# Ollama Model Injection Adapter
# This script handles injection of models and configurations into Ollama
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject models and configurations into Ollama AI server"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source Ollama configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Ollama settings
readonly DEFAULT_OLLAMA_HOST="http://localhost:11434"

# Ollama settings (can be overridden by environment)
OLLAMA_HOST="${OLLAMA_HOST:-$DEFAULT_OLLAMA_HOST}"

# Operation tracking
declare -a OLLAMA_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
ollama_inject::usage() {
    cat << EOF
Ollama Model Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects models and configurations into Ollama based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the model injection
    --status      Check status of injected models
    --rollback    Rollback injected models
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "models": [
        {
          "name": "llama2",
          "tag": "latest",
          "pull": true
        },
        {
          "name": "custom-model",
          "modelfile": "path/to/Modelfile",
          "build": true
        }
      ],
      "configurations": [
        {
          "key": "gpu_layers",
          "value": 35
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"models": [{"name": "llama2", "pull": true}]}'
    
    # Pull and inject models
    $0 --inject '{"models": [{"name": "codellama", "tag": "7b", "pull": true}]}'
    
    # Build custom model from Modelfile
    $0 --inject '{"models": [{"name": "my-model", "modelfile": "configs/Modelfile", "build": true}]}'

EOF
}

#######################################
# Check if Ollama is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
ollama_inject::check_accessibility() {
    # Check if Ollama is running
    if curl -s --max-time 5 "${OLLAMA_HOST}/api/tags" >/dev/null 2>&1; then
        log::debug "Ollama is accessible at $OLLAMA_HOST"
        return 0
    else
        log::error "Ollama is not accessible at $OLLAMA_HOST"
        log::info "Ensure Ollama is running: ./scripts/resources/ai/ollama/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
ollama_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    OLLAMA_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Ollama rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
ollama_inject::execute_rollback() {
    if [[ ${#OLLAMA_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Ollama rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Ollama rollback actions..."
    
    local success_count=0
    local total_count=${#OLLAMA_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#OLLAMA_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${OLLAMA_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Ollama rollback completed: $success_count/$total_count actions successful"
    OLLAMA_ROLLBACK_ACTIONS=()
}

#######################################
# Validate model configuration
# Arguments:
#   $1 - models configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
ollama_inject::validate_models() {
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
        local name pull build
        name=$(echo "$model" | jq -r '.name // empty')
        pull=$(echo "$model" | jq -r '.pull // false')
        build=$(echo "$model" | jq -r '.build // false')
        
        if [[ -z "$name" ]]; then
            log::error "Model at index $i missing required 'name' field"
            return 1
        fi
        
        # Check that either pull or build is specified
        if [[ "$pull" == "false" && "$build" == "false" ]]; then
            log::error "Model '$name' must have either 'pull' or 'build' set to true"
            return 1
        fi
        
        # If build is true, check for modelfile
        if [[ "$build" == "true" ]]; then
            local modelfile
            modelfile=$(echo "$model" | jq -r '.modelfile // empty')
            
            if [[ -z "$modelfile" ]]; then
                log::error "Model '$name' with build=true missing required 'modelfile' field"
                return 1
            fi
            
            # Check if modelfile exists
            local modelfile_path="$VROOLI_PROJECT_ROOT/$modelfile"
            if [[ ! -f "$modelfile_path" ]]; then
                log::error "Modelfile not found: $modelfile_path"
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
ollama_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Ollama injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Ollama injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_models has_configurations
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Ollama injection configuration must have 'models' or 'configurations'"
        return 1
    fi
    
    # Validate models if present
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! ollama_inject::validate_models "$models"; then
            return 1
        fi
    fi
    
    # Validate configurations if present
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        # Basic validation for configurations
        local config_type
        config_type=$(echo "$configurations" | jq -r 'type')
        
        if [[ "$config_type" != "array" ]]; then
            log::error "Configurations must be an array"
            return 1
        fi
    fi
    
    log::success "Ollama injection configuration is valid"
    return 0
}

#######################################
# Pull model from registry
# Arguments:
#   $1 - model configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
ollama_inject::pull_model() {
    local model_config="$1"
    
    local name tag
    name=$(echo "$model_config" | jq -r '.name')
    tag=$(echo "$model_config" | jq -r '.tag // "latest"')
    
    local model_spec="${name}:${tag}"
    
    log::info "Pulling model: $model_spec"
    
    # Pull model using Ollama API
    local pull_response
    pull_response=$(curl -s -X POST "${OLLAMA_HOST}/api/pull" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$model_spec\"}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Pulled model: $model_spec"
        
        # Add rollback action
        ollama_inject::add_rollback_action \
            "Remove model: $model_spec" \
            "curl -s -X DELETE '${OLLAMA_HOST}/api/delete' -d '{\"name\": \"$model_spec\"}' >/dev/null 2>&1"
        
        return 0
    else
        log::error "Failed to pull model: $model_spec"
        return 1
    fi
}

#######################################
# Build model from Modelfile
# Arguments:
#   $1 - model configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
ollama_inject::build_model() {
    local model_config="$1"
    
    local name modelfile
    name=$(echo "$model_config" | jq -r '.name')
    modelfile=$(echo "$model_config" | jq -r '.modelfile')
    
    # Resolve modelfile path
    local modelfile_path="$VROOLI_PROJECT_ROOT/$modelfile"
    
    log::info "Building model '$name' from Modelfile: $modelfile"
    
    # Read modelfile content
    local modelfile_content
    modelfile_content=$(cat "$modelfile_path")
    
    # Build model using Ollama API
    local build_response
    build_response=$(curl -s -X POST "${OLLAMA_HOST}/api/create" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$name\", \"modelfile\": \"$modelfile_content\"}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Built model: $name"
        
        # Add rollback action
        ollama_inject::add_rollback_action \
            "Remove model: $name" \
            "curl -s -X DELETE '${OLLAMA_HOST}/api/delete' -d '{\"name\": \"$name\"}' >/dev/null 2>&1"
        
        return 0
    else
        log::error "Failed to build model: $name"
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
ollama_inject::inject_models() {
    local models_config="$1"
    
    log::info "Injecting Ollama models..."
    
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    if [[ "$model_count" -eq 0 ]]; then
        log::info "No models to inject"
        return 0
    fi
    
    local failed_models=()
    
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        local name pull build
        name=$(echo "$model" | jq -r '.name')
        pull=$(echo "$model" | jq -r '.pull // false')
        build=$(echo "$model" | jq -r '.build // false')
        
        if [[ "$pull" == "true" ]]; then
            if ! ollama_inject::pull_model "$model"; then
                failed_models+=("$name")
            fi
        elif [[ "$build" == "true" ]]; then
            if ! ollama_inject::build_model "$model"; then
                failed_models+=("$name")
            fi
        fi
    done
    
    if [[ ${#failed_models[@]} -eq 0 ]]; then
        log::success "All models injected successfully"
        return 0
    else
        log::error "Failed to inject models: ${failed_models[*]}"
        return 1
    fi
}

#######################################
# Apply configurations
# Arguments:
#   $1 - configurations JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
ollama_inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Ollama configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Note: Ollama configuration is typically done via environment variables
    # This is a placeholder for future configuration injection capabilities
    log::warn "Configuration injection not yet implemented for Ollama"
    log::info "Set environment variables before starting Ollama for configuration"
    
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
ollama_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Ollama"
    
    # Check Ollama accessibility
    if ! ollama_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    OLLAMA_ROLLBACK_ACTIONS=()
    
    # Inject models if present
    local has_models
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! ollama_inject::inject_models "$models"; then
            log::error "Failed to inject models"
            ollama_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! ollama_inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            ollama_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Ollama data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
ollama_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Ollama injection status"
    
    # Check Ollama accessibility
    if ! ollama_inject::check_accessibility; then
        return 1
    fi
    
    # Get list of installed models
    log::info "Fetching installed models..."
    
    local installed_models
    installed_models=$(curl -s "${OLLAMA_HOST}/api/tags" | jq -r '.models[]?.name' 2>/dev/null)
    
    if [[ -z "$installed_models" ]]; then
        log::warn "No models currently installed"
    else
        log::info "Installed models:"
        echo "$installed_models" | while read -r model; do
            log::info "  - $model"
        done
    fi
    
    # Check models from configuration
    local has_models
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        log::info "Checking configured models..."
        
        local model_count
        model_count=$(echo "$models" | jq 'length')
        
        for ((i=0; i<model_count; i++)); do
            local model
            model=$(echo "$models" | jq -c ".[$i]")
            
            local name tag
            name=$(echo "$model" | jq -r '.name')
            tag=$(echo "$model" | jq -r '.tag // "latest"')
            
            local model_spec="${name}:${tag}"
            
            # Check if model is installed
            if echo "$installed_models" | grep -q "^${model_spec}$"; then
                log::success "✅ Model '$model_spec' is installed"
            else
                log::error "❌ Model '$model_spec' not found"
            fi
        done
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
ollama_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        ollama_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            ollama_inject::validate_config "$config"
            ;;
        "--inject")
            ollama_inject::inject_data "$config"
            ;;
        "--status")
            ollama_inject::check_status "$config"
            ;;
        "--rollback")
            ollama_inject::execute_rollback
            ;;
        "--help")
            ollama_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            ollama_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        ollama_inject::usage
        exit 1
    fi
    
    ollama_inject::main "$@"
fi