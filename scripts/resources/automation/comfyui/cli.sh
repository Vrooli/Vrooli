#!/usr/bin/env bash
################################################################################
# ComfyUI Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-comfyui <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    COMFYUI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    COMFYUI_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
COMFYUI_CLI_DIR="$(cd "$(dirname "$COMFYUI_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source ComfyUI configuration
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
comfyui::export_config 2>/dev/null || true

# Source ComfyUI libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/gpu.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/models.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/workflows.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/lib/install.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "comfyui" "ComfyUI AI image generation and manipulation"

# Override help to provide ComfyUI-specific examples
cli::register_command "help" "Show this help message with examples" "comfyui::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install ComfyUI" "comfyui::install" "modifies-system"
cli::register_command "uninstall" "Uninstall ComfyUI" "comfyui::cli_uninstall" "modifies-system"
cli::register_command "start" "Start ComfyUI" "comfyui::start" "modifies-system"
cli::register_command "stop" "Stop ComfyUI" "comfyui::stop" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "comfyui::status"
cli::register_command "validate" "Validate installation" "comfyui::validate"
cli::register_command "logs" "Show ComfyUI logs" "comfyui::cli_logs"

# Register workflow and model commands
cli::register_command "inject" "Inject workflow into ComfyUI" "comfyui::cli_inject" "modifies-system"
cli::register_command "execute-workflow" "Execute a workflow" "comfyui::cli_execute_workflow" "modifies-system"
cli::register_command "list-workflows" "List available workflows" "workflows::list_workflows"
cli::register_command "download-models" "Download default models" "models::download_default" "modifies-system"
cli::register_command "list-models" "List available models" "models::list"

# Register GPU and hardware commands
cli::register_command "gpu-info" "Show GPU information" "gpu::info"
cli::register_command "validate-nvidia" "Validate NVIDIA requirements" "gpu::validate_nvidia"

# Register utility commands
cli::register_command "credentials" "Show ComfyUI credentials for integration" "comfyui::cli_credentials"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Inject workflow into ComfyUI
comfyui::cli_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-comfyui inject <workflow.json>"
        echo ""
        echo "Examples:"
        echo "  resource-comfyui inject workflow.json"
        echo "  resource-comfyui inject shared:initialization/automation/comfyui/workflow.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing workflow import function
    export WORKFLOW_PATH="$file"
    workflows::import_workflow
}

# Uninstall with force confirmation
comfyui::cli_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove ComfyUI and all its data. Use --force to confirm."
        return 1
    fi
    
    comfyui::uninstall
}

# Show logs with line count
comfyui::cli_logs() {
    local lines="${1:-50}"
    local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
    
    if command -v comfyui::docker::logs &>/dev/null; then
        comfyui::docker::logs
    else
        docker logs --tail "$lines" "$container_name"
    fi
}

# Show credentials for ComfyUI integration
comfyui::cli_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${COMFYUI_CONTAINER_NAME:-comfyui}")
    
    # Build connections array - ComfyUI runs without auth by default
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # ComfyUI HTTP API connection (no authentication by default)
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${COMFYUI_DIRECT_PORT:-8188}" \
            --arg path "/api" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "ComfyUI AI image generation and manipulation" \
            --arg web_ui "${COMFYUI_BASE_URL:-http://localhost:8188}" \
            --argjson jupyter_port "${COMFYUI_JUPYTER_PORT:-8889}" \
            --arg models_dir "${COMFYUI_MODELS_DIR:-/opt/ComfyUI/models}" \
            '{
                description: $description,
                web_ui_url: $web_ui,
                jupyter_port: $jupyter_port,
                models_directory: $models_dir,
                auth_enabled: false
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "ComfyUI API" \
            "httpRequest" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "comfyui" "$status" "$connections_array")
    
    credentials::format_output "$response"
}

# Execute workflow with validation
comfyui::cli_execute_workflow() {
    local workflow="${1:-}"
    
    if [[ -z "$workflow" ]]; then
        log::error "Workflow file required"
        echo "Usage: resource-comfyui execute-workflow <workflow.json>"
        echo ""
        echo "Examples:"
        echo "  resource-comfyui execute-workflow basic_text_to_image.json"
        echo "  resource-comfyui execute-workflow shared:examples/comfyui/workflow.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$workflow" == shared:* ]]; then
        workflow="${var_VROOLI_ROOT}/${workflow#shared:}"
    fi
    
    if [[ ! -f "$workflow" ]]; then
        log::error "Workflow file not found: $workflow"
        return 1
    fi
    
    export WORKFLOW_PATH="$workflow"
    workflows::execute_workflow
}


# Custom help function with examples
comfyui::show_help() {
    cli::_handle_help
    
    echo ""
    echo "⚡ Examples:"
    echo ""
    echo "  # Workflow operations"
    echo "  resource-comfyui inject workflow.json"
    echo "  resource-comfyui inject shared:initialization/comfyui/workflow.json"
    echo "  resource-comfyui execute-workflow basic_text_to_image.json"
    echo "  resource-comfyui list-workflows"
    echo ""
    echo "  # Model management"
    echo "  resource-comfyui download-models"
    echo "  resource-comfyui list-models"
    echo ""
    echo "  # System and hardware"
    echo "  resource-comfyui gpu-info"
    echo "  resource-comfyui validate-nvidia"
    echo ""
    echo "  # Management"
    echo "  resource-comfyui status"
    echo "  resource-comfyui logs 100"
    echo "  resource-comfyui credentials"
    echo ""
    echo "  # Dangerous operations"
    echo "  resource-comfyui uninstall --force"
    echo ""
    echo "Default Port: ${COMFYUI_DIRECT_PORT:-8188}"
    echo "Web UI: http://localhost:${COMFYUI_DIRECT_PORT:-8188}"
    echo "Features: AI image generation, GPU acceleration, custom nodes"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi