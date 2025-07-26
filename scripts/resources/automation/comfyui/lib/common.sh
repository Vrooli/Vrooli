#!/usr/bin/env bash
# ComfyUI Common Utilities
# Shared functions used across ComfyUI modules

# Get the ComfyUI script directory
COMFYUI_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Source configuration
# shellcheck disable=SC1091
source "${COMFYUI_SCRIPT_DIR}/config/defaults.sh"

# Source Vrooli common utilities
RESOURCES_DIR="${COMFYUI_SCRIPT_DIR}/../.."
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

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
comfyui::parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --action|-a)
                ACTION="$2"
                shift 2
                ;;
            --force|-f)
                FORCE="${2:-yes}"
                [[ "$FORCE" != "yes" && "$FORCE" != "no" ]] && FORCE="no"
                shift 2
                ;;
            --yes|-y)
                YES="${2:-yes}"
                [[ "$YES" != "yes" && "$YES" != "no" ]] && YES="no"
                shift 2
                ;;
            --gpu)
                GPU_TYPE="$2"
                shift 2
                ;;
            --workflow)
                WORKFLOW_PATH="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --prompt-id)
                PROMPT_ID="$2"
                shift 2
                ;;
            --help|-h)
                ACTION="help"
                shift
                ;;
            *)
                log::error "Unknown argument: $1"
                comfyui::usage
                exit 1
                ;;
        esac
    done
    
    # Set default action if not specified
    ACTION="${ACTION:-help}"
    
    # Apply environment variable overrides
    GPU_TYPE="${COMFYUI_GPU_TYPE:-${GPU_TYPE:-auto}}"
}

#######################################
# Display usage information
#######################################
comfyui::usage() {
    cat << EOF
Usage: $0 [OPTIONS]

ComfyUI Resource Management Script

OPTIONS:
    --action, -a <action>     Action to perform (required)
                             Available actions:
                               install    - Install ComfyUI
                               uninstall  - Remove ComfyUI
                               start      - Start ComfyUI service
                               stop       - Stop ComfyUI service
                               restart    - Restart ComfyUI service
                               status     - Check ComfyUI status
                               logs       - View ComfyUI logs
                               info       - Display resource information
                               download-models - Download default models
                               list-models - List installed models
                               execute-workflow - Execute a workflow
                               import-workflow - Import a workflow file
                               gpu-info   - Display GPU information
                               validate-nvidia - Validate NVIDIA setup
                               check-ready - Check if ComfyUI is ready
                               cleanup-help - Show cleanup instructions
    
    --force, -f [yes|no]     Force action even if already done (default: no)
    --yes, -y [yes|no]       Skip confirmation prompts (default: no)
    --gpu <type>             GPU type: auto|nvidia|amd|cpu (default: auto)
    --workflow <path>        Path to workflow file (for execute/import)
    --output <dir>           Output directory for generated images
    --help, -h               Display this help message

ENVIRONMENT VARIABLES:
    COMFYUI_CUSTOM_PORT      Override default port (default: 5679)
    COMFYUI_GPU_TYPE         Force GPU type (auto/nvidia/amd/cpu)
    COMFYUI_CUSTOM_IMAGE     Use custom Docker image
    COMFYUI_VRAM_LIMIT       Limit VRAM usage in GB
    COMFYUI_NVIDIA_CHOICE    Non-interactive NVIDIA choice (1-4)

EXAMPLES:
    # Install with automatic GPU detection
    $0 --action install
    
    # Install with specific GPU type
    $0 --action install --gpu nvidia
    
    # Execute a workflow
    $0 --action execute-workflow --workflow my-workflow.json
    
    # Check GPU information
    $0 --action gpu-info

EOF
}

#######################################
# Check if ComfyUI container exists
#######################################
comfyui::container_exists() {
    docker::run ps -a --filter "name=^${COMFYUI_CONTAINER_NAME}$" --format "{{.Names}}" | grep -q "^${COMFYUI_CONTAINER_NAME}$"
}

#######################################
# Check if ComfyUI is running
#######################################
comfyui::is_running() {
    docker::run ps --filter "name=^${COMFYUI_CONTAINER_NAME}$" --filter "status=running" --format "{{.Names}}" | grep -q "^${COMFYUI_CONTAINER_NAME}$"
}

#######################################
# Check if ComfyUI API is healthy
#######################################
comfyui::is_healthy() {
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
comfyui::check_ready() {
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
comfyui::create_directories() {
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
comfyui::update_config() {
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
comfyui::cleanup_help() {
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
    echo "   rm -rf ${COMFYUI_DATA_DIR}"
    echo
    echo "4. Remove from Vrooli configuration:"
    echo "   Edit ~/.vrooli/resources.local.json and remove the 'comfyui' entry"
    echo
    echo "5. Clean up any orphaned volumes:"
    echo "   docker volume prune"
    echo
    log::warn "Always backup important data before cleanup!"
}