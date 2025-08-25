#!/usr/bin/env bash
# ComfyUI Installation and Uninstallation
# Handles the main installation flow and cleanup procedures

# Source required utilities using unique directory variable
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
COMFYUI_LIB_DIR="${APP_ROOT}/resources/comfyui/lib"
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Main installation function
#######################################
install::install() {
    log::header "🚀 Installing ComfyUI"
    
    # Start rollback context
    resources::start_rollback_context "comfyui_install"
    
    # Check if already installed
    if [[ "$FORCE" != "yes" ]] && common::container_exists; then
        log::warn "ComfyUI is already installed"
        log::info "Use --force yes to reinstall"
        
        # Check if it's running
        if ! common::is_running; then
            log::info "ComfyUI is installed but not running"
            log::info "Starting ComfyUI..."
            comfyui::docker::start
        else
            log::success "ComfyUI is already running"
            local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
            log::info "Access at: http://localhost:$port"
        fi
        
        return 0
    fi
    
    # Step 1: Check prerequisites
    log::info "Checking prerequisites..."
    
    # Check Docker
    if ! install::check_docker; then
        resources::handle_error \
            "Docker is not available or not running" \
            "system" \
            "Ensure Docker is installed and running: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Check disk space
    if ! install::check_disk_space; then
        return 1
    fi
    
    # Step 2: Detect and configure GPU
    log::info "Detecting GPU configuration..."
    
    # Use specified GPU type or auto-detect
    if [[ -z "$GPU_TYPE" ]] || [[ "$GPU_TYPE" == "auto" ]]; then
        GPU_TYPE=$(gpu::detect_gpu_silent)
        log::info "Auto-detected GPU type: $GPU_TYPE"
        # Show detailed GPU info
        gpu::detect_gpu >/dev/null
    else
        log::info "Using specified GPU type: $GPU_TYPE"
    fi
    
    # Export GPU_TYPE so it's available to all functions
    export GPU_TYPE
    
    # Handle NVIDIA-specific requirements
    if [[ "$GPU_TYPE" == "nvidia" ]]; then
        if ! gpu::validate_nvidia_requirements; then
            log::warn "NVIDIA GPU requirements not met"
            
            # Handle NVIDIA setup
            if ! gpu::handle_nvidia_failure; then
                resources::handle_error \
                    "Failed to set up NVIDIA GPU support" \
                    "system" \
                    "Check GPU setup or use --gpu cpu to force CPU mode"
                return 1
            fi
        fi
    fi
    
    # Step 3: Create directories
    if ! common::create_directories; then
        resources::handle_error \
            "Failed to create ComfyUI directories" \
            "permission" \
            "Check permissions on ${COMFYUI_DATA_DIR}"
        return 1
    fi
    
    # Add rollback action for directories
    resources::add_rollback_action \
        "Remove ComfyUI directories" \
        "trash::safe_remove '${COMFYUI_DATA_DIR}' --no-confirm" \
        5
    
    # Step 4: Pull Docker image
    if ! comfyui::docker::pull_image; then
        resources::handle_error \
            "Failed to pull ComfyUI Docker image" \
            "network" \
            "Check internet connection and Docker Hub access"
        return 1
    fi
    
    # Step 5: Start container
    if ! comfyui::docker::start_container; then
        resources::handle_error \
            "Failed to start ComfyUI container" \
            "system" \
            "Check Docker logs: docker logs $COMFYUI_CONTAINER_NAME"
        return 1
    fi
    
    # At this point, ComfyUI is successfully installed
    # Clear rollback context to prevent removing it if next steps fail
    log::info "ComfyUI core installation completed successfully"
    ROLLBACK_ACTIONS=()
    OPERATION_ID=""
    
    # Step 6: Download models (non-critical)
    if [[ "${SKIP_MODELS:-no}" != "yes" ]]; then
        log::info "Downloading default models..."
        if ! models::download_default_models; then
            log::warn "Model download failed, but ComfyUI is installed"
            log::info "You can download models later with: $0 --action download-models"
        fi
    else
        log::info "Skipping model downloads (--skip-models specified)"
    fi
    
    # Step 7: Update Vrooli configuration (non-critical)
    if ! common::update_config; then
        log::warn "Failed to update Vrooli configuration, but ComfyUI is installed"
        log::info "You may need to configure Vrooli manually to use this ComfyUI instance"
    fi
    
    # Step 8: Show final status
    echo
    status::status
    
    log::success "✅ ComfyUI installation completed!"
    
    # Show next steps
    echo
    log::header "🎯 Next Steps"
    log::info "1. Access ComfyUI Web UI: http://localhost:${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    log::info "2. Download additional models: $0 --action download-models"
    log::info "3. Import workflows: $0 --action import-workflow --workflow <file>"
    log::info "4. Check GPU status: $0 --action gpu-info"
    echo
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${COMFYUI_LIB_DIR}/.." 2>/dev/null || true
    
    return 0
}

#######################################
# Uninstall ComfyUI
#######################################
install::uninstall() {
    log::header "🗑️ Uninstalling ComfyUI"
    
    # Check if installed
    if ! common::container_exists && [[ ! -d "$COMFYUI_DATA_DIR" ]]; then
        log::info "ComfyUI is not installed"
        return 0
    fi
    
    # Confirm uninstallation
    if [[ "$YES" != "yes" ]] && [[ "$FORCE" != "yes" ]]; then
        log::warn "This will remove ComfyUI and optionally all its data"
        echo
        read -rp "Are you sure you want to uninstall ComfyUI? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy] ]]; then
            log::info "Uninstallation cancelled"
            return 0
        fi
    fi
    
    # Step 1: Remove container
    if common::container_exists; then
        if ! comfyui::docker::remove_container; then
            log::error "Failed to remove container"
            log::info "You may need to remove it manually:"
            log::info "  docker stop $COMFYUI_CONTAINER_NAME"
            log::info "  docker rm $COMFYUI_CONTAINER_NAME"
        fi
    fi
    
    # Step 2: Remove from Vrooli configuration
    log::info "Removing from Vrooli configuration..."
    if ! resources::remove_config "$COMFYUI_CATEGORY" "$COMFYUI_RESOURCE_NAME"; then
        log::warn "Failed to remove from configuration"
    fi
    
    # Step 3: Handle data directory
    if [[ -d "$COMFYUI_DATA_DIR" ]]; then
        local remove_data="no"
        
        if [[ "$FORCE" == "yes" ]]; then
            remove_data="yes"
        elif [[ "$YES" != "yes" ]]; then
            # Check if there are models or outputs
            local has_models=false
            local has_outputs=false
            
            if [[ -d "${COMFYUI_DATA_DIR}/models" ]] && [[ -n "$(ls -A "${COMFYUI_DATA_DIR}/models" 2>/dev/null)" ]]; then
                has_models=true
            fi
            
            if [[ -d "${COMFYUI_DATA_DIR}/outputs" ]] && [[ -n "$(ls -A "${COMFYUI_DATA_DIR}/outputs" 2>/dev/null)" ]]; then
                has_outputs=true
            fi
            
            if [[ "$has_models" == "true" ]] || [[ "$has_outputs" == "true" ]]; then
                echo
                log::warn "ComfyUI data directory contains:"
                [[ "$has_models" == "true" ]] && log::info "  - Downloaded models"
                [[ "$has_outputs" == "true" ]] && log::info "  - Generated outputs"
                echo
                read -rp "Remove all ComfyUI data? (y/N): " confirm_data
                if [[ "$confirm_data" =~ ^[Yy] ]]; then
                    remove_data="yes"
                fi
            else
                remove_data="yes"
            fi
        fi
        
        if [[ "$remove_data" == "yes" ]]; then
            log::info "Removing ComfyUI data directory..."
            if trash::safe_remove "$COMFYUI_DATA_DIR" --no-confirm; then
                log::success "Data directory removed"
            else
                log::error "Failed to remove data directory"
                log::info "You may need to remove it manually: sudo trash::safe_remove $COMFYUI_DATA_DIR --no-confirm"
            fi
        else
            log::info "Keeping ComfyUI data directory: $COMFYUI_DATA_DIR"
            log::info "To remove it later: trash::safe_remove $COMFYUI_DATA_DIR --no-confirm"
        fi
    fi
    
    # Step 4: Optionally remove Docker image
    if [[ "$YES" != "yes" ]] && [[ "$FORCE" != "yes" ]]; then
        local image="${COMFYUI_CUSTOM_IMAGE:-$COMFYUI_DEFAULT_IMAGE}"
        echo
        read -rp "Remove ComfyUI Docker image ($image)? (y/N): " confirm_image
        if [[ "$confirm_image" =~ ^[Yy] ]]; then
            log::info "Removing Docker image..."
            if docker::run rmi "$image"; then
                log::success "Docker image removed"
            else
                log::warn "Failed to remove Docker image (may be in use by other containers)"
            fi
        fi
    fi
    
    log::success "✅ ComfyUI uninstalled successfully"
    
    # Show cleanup help
    echo
    log::info "If you need to perform additional cleanup:"
    log::info "Run: $0 --action cleanup-help"
    
    return 0
}

#######################################
# Check Docker availability
#######################################
install::check_docker() {
    if ! resources::ensure_docker; then
        log::error "Docker is required but not available"
        return 1
    fi
    
    # Check Docker version
    local docker_version
    docker_version=$(docker::run version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
    log::info "Docker version: $docker_version"
    
    # Check Docker Compose (optional, for future use)
    if system::is_command "docker-compose"; then
        local compose_version
        compose_version=$(docker-compose version --short 2>/dev/null || echo "unknown")
        log::debug "Docker Compose version: $compose_version"
    fi
    
    return 0
}

#######################################
# Check available disk space
#######################################
install::check_disk_space() {
    log::info "Checking disk space requirements..."
    
    local data_dir_parent
    data_dir_parent=${COMFYUI_DATA_DIR%/*
    
    # Get available space in GB
    local available_gb
    available_gb=$(df -BG "$data_dir_parent" | awk 'NR==2 {print $4}' | sed 's/G//')
    
    log::info "Available disk space: ${available_gb}GB"
    log::info "Minimum required: ${COMFYUI_MIN_DISK_GB}GB"
    log::info "Recommended: ${COMFYUI_RECOMMENDED_DISK_GB}GB"
    
    if [[ $available_gb -lt $COMFYUI_MIN_DISK_GB ]]; then
        log::error "Insufficient disk space"
        log::info "ComfyUI requires at least ${COMFYUI_MIN_DISK_GB}GB for models and outputs"
        return 1
    elif [[ $available_gb -lt $COMFYUI_RECOMMENDED_DISK_GB ]]; then
        log::warn "Disk space is below recommended amount"
        log::info "Some large models may not fit"
    else
        log::success "Disk space check passed"
    fi
    
    return 0
}