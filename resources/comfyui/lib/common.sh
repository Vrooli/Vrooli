#!/usr/bin/env bash
# ComfyUI Common Utilities

# Source required utilities using unique directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
COMFYUI_LIB_DIR="${APP_ROOT}/resources/comfyui/lib"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# Shared functions used across ComfyUI modules

# Get the ComfyUI script directory
COMFYUI_SCRIPT_DIR="${COMFYUI_LIB_DIR}/.."

# Source var.sh first if not already sourced
if [[ -z "${var_LIB_UTILS_DIR:-}" ]]; then
    # shellcheck disable=SC1091
    source "${COMFYUI_SCRIPT_DIR}/../../../lib/utils/var.sh"
fi

# Source configuration
# shellcheck disable=SC1091
source "${COMFYUI_SCRIPT_DIR}/config/defaults.sh"

# Source Vrooli common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Global variables for script state
declare -g ACTION=""
declare -g FORCE="no"
declare -g YES="no"
declare -g GPU_TYPE=""
declare -g WORKFLOW_PATH=""
declare -g OUTPUT_DIR=""
declare -g PROMPT_ID=""

#######################################
# Parse command line arguments
#######################################
common::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|download-models|list-models|execute-workflow|import-workflow|gpu-info|validate-nvidia|check-ready|cleanup-help" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if already done" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "gpu" \
        --desc "GPU type" \
        --type "value" \
        --options "auto|nvidia|amd|cpu" \
        --default "auto"
    
    args::register \
        --name "workflow" \
        --desc "Path to workflow file (for execute/import)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "output" \
        --desc "Output directory for generated images" \
        --type "value" \
        --default ""
    
    args::register \
        --name "prompt-id" \
        --desc "Prompt ID for workflow execution" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        args::usage "$DESCRIPTION"
        echo
        echo "Examples:"
        echo "  # Install with automatic GPU detection"
        echo "  $0 --action install"
        echo
        echo "  # Install with specific GPU type"
        echo "  $0 --action install --gpu nvidia"
        echo
        echo "  # Execute a workflow"
        echo "  $0 --action execute-workflow --workflow my-workflow.json"
        echo
        echo "Environment Variables:"
        echo "  COMFYUI_CUSTOM_PORT      Override default port (default: 8188)"
        echo "  COMFYUI_GPU_TYPE         Force GPU type (auto/nvidia/amd/cpu)"
        echo "  COMFYUI_CUSTOM_IMAGE     Use custom Docker image"
        echo "  COMFYUI_VRAM_LIMIT       Limit VRAM usage in GB"
        echo "  COMFYUI_NVIDIA_CHOICE    Non-interactive NVIDIA choice (1-4)"
        echo
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    YES=$(args::get "yes")
    FORCE=$(args::get "force")
    GPU_TYPE=$(args::get "gpu")
    WORKFLOW_PATH=$(args::get "workflow")
    OUTPUT_DIR=$(args::get "output")
    PROMPT_ID=$(args::get "prompt-id")
    export ACTION YES FORCE GPU_TYPE WORKFLOW_PATH OUTPUT_DIR PROMPT_ID
    
    # Apply environment variable overrides
    GPU_TYPE="${COMFYUI_GPU_TYPE:-${GPU_TYPE}}"
}


#######################################
# Check if ComfyUI container exists
#######################################
common::container_exists() {
    docker::run ps -a --filter "name=^${COMFYUI_CONTAINER_NAME}$" --format "{{.Names}}" | grep -q "^${COMFYUI_CONTAINER_NAME}$"
}

#######################################
# Check if ComfyUI is running
#######################################
common::is_running() {
    docker::run ps --filter "name=^${COMFYUI_CONTAINER_NAME}$" --filter "status=running" --format "{{.Names}}" | grep -q "^${COMFYUI_CONTAINER_NAME}$"
}

#######################################
# Check if ComfyUI API is healthy
#######################################
common::is_healthy() {
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -sf "${COMFYUI_API_BASE}${COMFYUI_HEALTH_ENDPOINT}" >/dev/null 2>&1; then
            return 0
        fi
        
        # Also check the direct ComfyUI port
        if curl -sf "http://localhost:${COMFYUI_DIRECT_PORT}" >/dev/null 2>&1; then
            return 0
        fi
        
        attempt=$((attempt + 1))
        if [[ $attempt -lt $max_attempts ]]; then
            sleep 2
        fi
    done
    
    return 1
}

#######################################
# Validate ComfyUI is ready for operations
#######################################
common::check_ready() {
    log::info "Checking if ComfyUI is ready..."
    
    # Check container exists
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist"
        log::info "Run: $0 --action install"
        return 1
    fi
    
    # Check container is running
    if ! comfyui::is_running; then
        log::error "ComfyUI container is not running"
        log::info "Run: $0 --action start"
        return 1
    fi
    
    # Check API is healthy
    if ! comfyui::is_healthy; then
        log::error "ComfyUI API is not responding"
        log::info "Check logs: $0 --action logs"
        return 1
    fi
    
    log::success "ComfyUI is ready for operations"
    return 0
}

#######################################
# Create required directories
#######################################
common::create_directories() {
    log::info "Creating ComfyUI directories..."
    
    for dir in "${COMFYUI_DIRS[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log::debug "Creating directory: $dir"
            mkdir -p "$dir" || {
                log::error "Failed to create directory: $dir"
                return 1
            }
        fi
    done
    
    # Set permissions to ensure Docker can access
    chmod -R 755 "${COMFYUI_DATA_DIR}" || {
        log::error "Failed to set permissions on ${COMFYUI_DATA_DIR}"
        return 1
    }
    
    log::success "Directories created successfully"
    return 0
}

#######################################
# Update Vrooli configuration
#######################################
common::update_config() {
    log::info "Updating Vrooli configuration for ComfyUI..."
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local base_url="http://localhost:$port"
    
    # Get GPU features
    local gpu_enabled="false"
    if [[ "$GPU_TYPE" != "cpu" ]]; then
        gpu_enabled="true"
    fi
    
    local additional_config="{\"features\":{\"workflows\":true,\"api\":true,\"gpu\":$gpu_enabled}}"
    
    if resources::update_config "$COMFYUI_CATEGORY" "$COMFYUI_RESOURCE_NAME" "$base_url" "$additional_config"; then
        log::success "Configuration updated successfully"
        return 0
    else
        log::error "Failed to update configuration"
        return 1
    fi
}

#######################################
# Display cleanup help
#######################################
common::cleanup_help() {
    log::header "🧹 ComfyUI Manual Cleanup Instructions"
    
    echo "If the uninstall script fails or you need to manually clean up ComfyUI:"
    echo
    echo "1. Stop and remove the container:"
    echo "   docker stop ${COMFYUI_CONTAINER_NAME} 2>/dev/null || true"
    echo "   docker rm ${COMFYUI_CONTAINER_NAME} 2>/dev/null || true"
    echo
    echo "2. Remove the Docker image (optional):"
    echo "   docker rmi ${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE} 2>/dev/null || true"
    echo
    echo "3. Remove ComfyUI data directory (WARNING: This deletes all models and outputs!):"
    echo "   trash::safe_remove ${COMFYUI_DATA_DIR} --no-confirm"
    echo
    echo "4. Remove from Vrooli configuration:"
    echo "   Edit ~/.vrooli/service.json and remove the 'comfyui' entry"
    echo
    echo "5. Clean up any orphaned volumes:"
    echo "   docker volume prune"
    echo
    log::warn "Always backup important data before cleanup!"
}