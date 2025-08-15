#!/usr/bin/env bash
################################################################################
# ComfyUI Resource CLI
# 
# Lightweight CLI interface for ComfyUI using the CLI Command Framework
#
# Usage:
#   resource-comfyui <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
COMFYUI_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$COMFYUI_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$COMFYUI_CLI_DIR"
export COMFYUI_SCRIPT_DIR="$COMFYUI_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Source ComfyUI configuration
# shellcheck disable=SC1091
source "${COMFYUI_CLI_DIR}/config/defaults.sh" 2>/dev/null || true

# Source ComfyUI libraries
for lib in common gpu docker models workflows status install; do
    lib_file="${COMFYUI_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "comfyui" "ComfyUI AI image generation and manipulation"

# Override help to provide ComfyUI-specific examples
cli::register_command "help" "Show this help message with ComfyUI examples" "resource_cli::show_help"

# Register additional ComfyUI-specific commands
cli::register_command "inject" "Inject workflow into ComfyUI" "resource_cli::inject" "modifies-system"
cli::register_command "execute-workflow" "Execute a workflow" "resource_cli::execute_workflow" "modifies-system"
cli::register_command "list-workflows" "List available workflows" "resource_cli::list_workflows"
cli::register_command "download-models" "Download default models" "resource_cli::download_models" "modifies-system"
cli::register_command "list-models" "List available models" "resource_cli::list_models"
cli::register_command "gpu-info" "Show GPU information" "resource_cli::gpu_info"
cli::register_command "validate-nvidia" "Validate NVIDIA requirements" "resource_cli::validate_nvidia"
cli::register_command "logs" "Show ComfyUI logs" "resource_cli::logs"
cli::register_command "credentials" "Show n8n credentials for ComfyUI" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall ComfyUI (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject workflow into ComfyUI
resource_cli::inject() {
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
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing workflow import function
    export WORKFLOW_PATH="$file"
    if command -v workflows::import_workflow &>/dev/null; then
        workflows::import_workflow
    else
        log::error "ComfyUI workflow import function not available"
        return 1
    fi
}

# Validate ComfyUI configuration
resource_cli::validate() {
    if command -v common::check_ready &>/dev/null; then
        common::check_ready
    else
        # Basic validation
        log::header "Validating ComfyUI"
        local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name" || {
            log::error "ComfyUI container not running"
            return 1
        }
        log::success "ComfyUI is running"
    fi
}

# Show ComfyUI status
resource_cli::status() {
    if command -v status::status &>/dev/null; then
        status::status
    else
        # Basic status
        log::header "ComfyUI Status"
        local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start ComfyUI
resource_cli::start() {
    if command -v docker::start &>/dev/null; then
        docker::start
    else
        local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
        docker start "$container_name" || log::error "Failed to start ComfyUI"
    fi
}

# Stop ComfyUI
resource_cli::stop() {
    if command -v docker::stop &>/dev/null; then
        docker::stop
    else
        local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
        docker stop "$container_name" || log::error "Failed to stop ComfyUI"
    fi
}

# Install ComfyUI
resource_cli::install() {
    if command -v install::install &>/dev/null; then
        install::install
    else
        log::error "ComfyUI install function not available"
        return 1
    fi
}

# Uninstall ComfyUI
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove ComfyUI and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v install::uninstall &>/dev/null; then
        install::uninstall
    else
        local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "ComfyUI uninstalled"
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "comfyui"; return 0; }
        return 1
    fi
    
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
    
    local response
    response=$(credentials::build_response "comfyui" "$status" "$connections_array")
    credentials::format_output "$response"
}

# Execute workflow
resource_cli::execute_workflow() {
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
        workflow="${VROOLI_ROOT}/${workflow#shared:}"
    fi
    
    if [[ ! -f "$workflow" ]]; then
        log::error "Workflow file not found: $workflow"
        return 1
    fi
    
    export WORKFLOW_PATH="$workflow"
    if command -v workflows::execute_workflow &>/dev/null; then
        workflows::execute_workflow
    else
        log::error "Workflow execution not available"
        return 1
    fi
}

# List workflows
resource_cli::list_workflows() {
    if command -v workflows::list_workflows &>/dev/null; then
        workflows::list_workflows
    else
        log::error "Workflow listing not available"
        return 1
    fi
}

# Download models
resource_cli::download_models() {
    if command -v models::download_default_models &>/dev/null; then
        models::download_default_models
    elif command -v models::download_models &>/dev/null; then
        models::download_models
    else
        log::error "Model download not available"
        return 1
    fi
}

# List models
resource_cli::list_models() {
    if command -v models::list_models &>/dev/null; then
        models::list_models
    else
        log::error "Model listing not available"
        return 1
    fi
}

# GPU info
resource_cli::gpu_info() {
    if command -v gpu::get_gpu_info &>/dev/null; then
        gpu::get_gpu_info
    else
        log::error "GPU info not available"
        return 1
    fi
}

# Validate NVIDIA requirements
resource_cli::validate_nvidia() {
    if command -v gpu::validate_nvidia_requirements &>/dev/null; then
        gpu::validate_nvidia_requirements
    else
        log::error "NVIDIA validation not available"
        return 1
    fi
}

# Show logs
resource_cli::logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    local container_name="${COMFYUI_CONTAINER_NAME:-comfyui}"
    
    if command -v docker::logs &>/dev/null; then
        docker::logs
    else
        if [[ "$follow" == "true" ]]; then
            docker logs -f --tail "$lines" "$container_name" 2>/dev/null || log::error "Failed to show logs"
        else
            docker logs --tail "$lines" "$container_name" 2>/dev/null || log::error "Failed to show logs"
        fi
    fi
}

# Custom help function with ComfyUI-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add ComfyUI-specific examples
    echo ""
    echo "🎨 ComfyUI AI Image Generation Examples:"
    echo ""
    echo "Workflow Operations:"
    echo "  resource-comfyui execute-workflow basic_text_to_image.json  # Run workflow"
    echo "  resource-comfyui list-workflows                             # List workflows"
    echo "  resource-comfyui inject workflow.json                      # Import workflow"
    echo "  resource-comfyui inject shared:init/comfyui/workflow.json   # Import shared"
    echo ""
    echo "Model Management:"
    echo "  resource-comfyui download-models                           # Download default models"
    echo "  resource-comfyui list-models                               # List available models"
    echo ""
    echo "System & Hardware:"
    echo "  resource-comfyui gpu-info                                  # Show GPU details"
    echo "  resource-comfyui validate-nvidia                           # Check NVIDIA setup"
    echo "  resource-comfyui logs 100 true                             # Follow logs"
    echo ""
    echo "Management:"
    echo "  resource-comfyui status                                    # Check service status"
    echo "  resource-comfyui credentials                               # Get API details"
    echo ""
    echo "AI Features:"
    echo "  • Stable Diffusion and custom model support"
    echo "  • Node-based visual workflow editor"
    echo "  • GPU acceleration for fast generation"
    echo "  • Custom nodes and extensions ecosystem"
    echo ""
    echo "Default Port: 8188"
    echo "Web UI: http://localhost:8188"
    echo "Jupyter (if enabled): http://localhost:8889"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi