#!/usr/bin/env bash
set -euo pipefail

# ComfyUI AI Image Generation Workflow Platform Setup and Management
# This script handles installation, configuration, and management of ComfyUI using Docker
#
# Environment Variables:
#   COMFYUI_NVIDIA_CHOICE - Non-interactive NVIDIA runtime choice (1-4):
#     1: Auto-install NVIDIA Container Runtime
#     2: Show manual installation instructions
#     3: Continue with CPU mode instead
#     4: Cancel installation

DESCRIPTION="Install and manage ComfyUI AI-powered image generation workflow platform using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/docker.sh"

# ComfyUI configuration
readonly COMFYUI_PORT="${COMFYUI_CUSTOM_PORT:-$(resources::get_default_port "comfyui")}"
readonly COMFYUI_BASE_URL="http://localhost:${COMFYUI_PORT}"
readonly COMFYUI_CONTAINER_NAME="comfyui"
readonly COMFYUI_DATA_DIR="${HOME}/.comfyui"
readonly COMFYUI_IMAGE_GPU="${COMFYUI_CUSTOM_IMAGE:-ghcr.io/ai-dock/comfyui:latest}"
readonly COMFYUI_IMAGE_CPU="${COMFYUI_CPU_IMAGE:-ghcr.io/ai-dock/comfyui:latest}"

# Directory paths
readonly COMFYUI_MODELS_DIR="${COMFYUI_DATA_DIR}/models"
readonly COMFYUI_CUSTOM_NODES_DIR="${COMFYUI_DATA_DIR}/custom_nodes"
readonly COMFYUI_OUTPUTS_DIR="${COMFYUI_DATA_DIR}/outputs"
readonly COMFYUI_WORKFLOWS_DIR="${COMFYUI_DATA_DIR}/workflows"
readonly COMFYUI_INPUT_DIR="${COMFYUI_DATA_DIR}/input"

# GPU configuration
readonly GPU_TYPE="${COMFYUI_GPU_TYPE:-auto}"  # auto, nvidia, amd, cpu
readonly VRAM_LIMIT="${COMFYUI_VRAM_LIMIT:-}"  # Optional VRAM limit in GB

# Model configuration
readonly DEFAULT_CHECKPOINT="sd_xl_base_1.0.safetensors"
readonly DEFAULT_VAE="sdxl_vae.safetensors"
readonly MIN_DISK_SPACE_GB=50

#######################################
# Parse command line arguments
#######################################
comfyui::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|download-models|execute-workflow|import-workflow|list-models|gpu-info|validate-nvidia" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if ComfyUI appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "gpu" \
        --desc "GPU type (auto, nvidia, amd, cpu)" \
        --type "value" \
        --options "auto|nvidia|amd|cpu" \
        --default "auto"
    
    args::register \
        --name "models" \
        --desc "Models to download (comma-separated list or 'default')" \
        --type "value" \
        --default "default"
    
    args::register \
        --name "workflow" \
        --desc "Path to workflow JSON file" \
        --type "value" \
        --default ""
    
    args::register \
        --name "output" \
        --desc "Output directory for generated images" \
        --type "value" \
        --default "${COMFYUI_OUTPUTS_DIR}"
    
    args::register \
        --name "client-id" \
        --desc "Client ID for API calls (auto-generated if not provided)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "headless" \
        --desc "Run in headless mode (no UI)" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    if args::is_asking_for_help "$@"; then
        comfyui::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export GPU_OVERRIDE=$(args::get "gpu")
    export MODELS_TO_DOWNLOAD=$(args::get "models")
    export WORKFLOW_FILE=$(args::get "workflow")
    export OUTPUT_DIR=$(args::get "output")
    export CLIENT_ID=$(args::get "client-id")
    export HEADLESS_MODE=$(args::get "headless")
}

#######################################
# Display usage information
#######################################
comfyui::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install ComfyUI with auto GPU detection"
    echo "  $0 --action install --gpu nvidia                 # Install with NVIDIA GPU support"
    echo "  $0 --action install --gpu cpu                    # Install CPU-only version"
    echo "  $0 --action download-models --models default     # Download default models"
    echo "  $0 --action execute-workflow --workflow workflow.json  # Execute a workflow"
    echo "  $0 --action status                               # Check ComfyUI status"
    echo "  $0 --action check-ready                          # Check if ComfyUI is ready to use"
    echo "  $0 --action gpu-info                             # Display GPU information"
    echo "  $0 --action validate-nvidia                      # Validate NVIDIA runtime setup"
    echo "  $0 --action uninstall                            # Remove ComfyUI"
}

#######################################
# Check if Docker is installed and accessible
# Returns: 0 if ready, 1 otherwise
#######################################
comfyui::check_docker() {
    # Use the docker.sh helpers for comprehensive checks
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        docker::install || {
            log::info "Please install Docker manually: https://docs.docker.com/get-docker/"
            return 1
        }
    fi
    
    # Try to start Docker if it's not running
    if ! docker::_is_running; then
        log::info "Docker is not running, attempting to start..."
        docker::start || {
            log::error "Failed to start Docker"
            docker::diagnose
            return 1
        }
    fi
    
    # Check permissions and try alternative access methods
    if ! docker::run ps >/dev/null 2>&1; then
        log::info "Checking Docker permissions..."
        docker::manage_docker_group
        
        # Try alternative access methods
        if ! docker::run ps >/dev/null 2>&1; then
            docker::_diagnose_permission_issue
            return 1
        fi
    fi
    
    log::success "Docker is installed and accessible"
    return 0
}

#######################################
# Check available disk space
# Returns: 0 if sufficient, 1 otherwise
#######################################
comfyui::check_disk_space() {
    local required_gb="${1:-$MIN_DISK_SPACE_GB}"
    local available_gb
    
    # Get available space in GB
    available_gb=$(df -BG "$HOME" | awk 'NR==2 {print $4}' | sed 's/G//')
    
    if [[ "$available_gb" -lt "$required_gb" ]]; then
        log::error "Insufficient disk space"
        log::info "Required: ${required_gb}GB, Available: ${available_gb}GB"
        log::info "ComfyUI needs space for models and generated images"
        return 1
    fi
    
    log::info "Disk space check passed (${available_gb}GB available)"
    return 0
}

#######################################
# Detect GPU type and capabilities (silent version)
# Returns: GPU type (nvidia, amd, cpu)
#######################################
comfyui::detect_gpu_silent() {
    if [[ "$GPU_OVERRIDE" != "auto" ]]; then
        echo "$GPU_OVERRIDE"
        return 0
    fi
    
    # Check for NVIDIA GPU
    if system::is_command "nvidia-smi"; then
        if nvidia-smi &>/dev/null; then
            echo "nvidia"
            return 0
        fi
    fi
    
    # Check for AMD GPU
    if [[ -f /sys/class/drm/card0/device/vendor ]]; then
        local vendor_id=$(cat /sys/class/drm/card0/device/vendor 2>/dev/null)
        if [[ "$vendor_id" == "0x1002" ]]; then
            echo "amd"
            return 0
        fi
    fi
    
    # Check using lspci
    if system::is_command "lspci"; then
        if lspci | grep -i "vga\|3d\|display" | grep -qi "nvidia"; then
            echo "nvidia"
            return 0
        elif lspci | grep -i "vga\|3d\|display" | grep -qi "amd"; then
            echo "amd"
            return 0
        fi
    fi
    
    echo "cpu"
    return 0
}

#######################################
# Detect GPU type and capabilities (with logging)
# Returns: GPU type (nvidia, amd, cpu)
#######################################
comfyui::detect_gpu() {
    if [[ "$GPU_OVERRIDE" != "auto" ]]; then
        echo "$GPU_OVERRIDE"
        return 0
    fi
    
    # Check for NVIDIA GPU
    if system::is_command "nvidia-smi"; then
        if nvidia-smi &>/dev/null; then
            log::info "NVIDIA GPU detected"
            echo "nvidia"
            return 0
        fi
    fi
    
    # Check for AMD GPU
    if [[ -f /sys/class/drm/card0/device/vendor ]]; then
        local vendor_id=$(cat /sys/class/drm/card0/device/vendor 2>/dev/null)
        if [[ "$vendor_id" == "0x1002" ]]; then
            log::info "AMD GPU detected"
            echo "amd"
            return 0
        fi
    fi
    
    # Check using lspci
    if system::is_command "lspci"; then
        if lspci | grep -i "vga\|3d\|display" | grep -qi "nvidia"; then
            log::info "NVIDIA GPU detected (via lspci)"
            echo "nvidia"
            return 0
        elif lspci | grep -i "vga\|3d\|display" | grep -qi "amd"; then
            log::info "AMD GPU detected (via lspci)"
            echo "amd"
            return 0
        fi
    fi
    
    log::warn "No GPU detected, will use CPU mode"
    echo "cpu"
    return 0
}

#######################################
# Detect OS distribution for package management
# Returns: OS identifier (ubuntu, debian, centos, rhel, fedora, arch, unknown)
#######################################
comfyui::detect_os_distribution() {
    if [[ -f /etc/os-release ]]; then
        local id
        id=$(grep "^ID=" /etc/os-release | cut -d'=' -f2 | tr -d '"')
        case "$id" in
            ubuntu|debian|centos|rhel|fedora|arch)
                echo "$id"
                ;;
            *)
                # Check ID_LIKE for derivatives
                local id_like
                id_like=$(grep "^ID_LIKE=" /etc/os-release | cut -d'=' -f2 | tr -d '"')
                case "$id_like" in
                    *ubuntu*|*debian*)
                        echo "debian"
                        ;;
                    *rhel*|*fedora*)
                        echo "rhel"
                        ;;
                    *arch*)
                        echo "arch"
                        ;;
                    *)
                        echo "unknown"
                        ;;
                esac
                ;;
        esac
    else
        echo "unknown"
    fi
}

#######################################
# Check if NVIDIA Container Toolkit is installed
# Returns: 0 if installed, 1 otherwise
#######################################
comfyui::is_nvidia_runtime_installed() {
    # Check for nvidia-ctk command
    if system::is_command "nvidia-ctk"; then
        return 0
    fi
    
    # Check for nvidia-container-runtime
    if system::is_command "nvidia-container-runtime"; then
        return 0
    fi
    
    return 1
}

#######################################
# Check if Docker has NVIDIA runtime configured
# Returns: 0 if configured, 1 otherwise
#######################################
comfyui::is_docker_nvidia_configured() {
    # Try to get Docker info with proper permissions
    local docker_info
    if docker_info=$(docker::run info 2>/dev/null); then
        echo "$docker_info" | grep -q "nvidia"
        return $?
    elif docker_info=$(sg docker -c "docker info" 2>/dev/null); then
        echo "$docker_info" | grep -q "nvidia"
        return $?
    else
        log::warn "Cannot access Docker info to check NVIDIA configuration"
        return 1
    fi
}

#######################################
# Test NVIDIA runtime functionality
# Returns: 0 if working, 1 otherwise
#######################################
comfyui::test_nvidia_runtime() {
    log::info "Testing NVIDIA runtime functionality..."
    
    # Use a minimal CUDA container for testing
    local test_image="nvidia/cuda:11.8-base-ubuntu20.04"
    local test_result
    local exit_code
    
    # Try with docker::run first
    if test_result=$(timeout 60 docker::run run --rm --gpus all "$test_image" nvidia-smi --query-gpu=name --format=csv,noheader 2>&1); then
        exit_code=$?
    # Fallback to sg docker
    elif test_result=$(timeout 60 sg docker -c "docker run --rm --gpus all $test_image nvidia-smi --query-gpu=name --format=csv,noheader" 2>&1); then
        exit_code=$?
    else
        exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]] && [[ -n "$test_result" ]] && [[ "$test_result" != *"error"* ]]; then
        log::success "✅ NVIDIA runtime test passed"
        log::info "Detected GPU: $(echo "$test_result" | head -1)"
        return 0
    else
        log::error "❌ NVIDIA runtime test failed"
        if [[ -n "$test_result" ]]; then
            log::info "Test output: $test_result"
        fi
        return 1
    fi
}

#######################################
# Comprehensive NVIDIA requirements validation
# Arguments: $1 - GPU type
# Returns: 0=success, 1=drivers missing, 2=runtime missing, 3=config missing, 4=runtime broken
#######################################
comfyui::validate_nvidia_requirements() {
    local gpu_type="${1:-}"
    
    # If no GPU type specified, detect it
    if [[ -z "$gpu_type" ]]; then
        gpu_type=$(comfyui::detect_gpu_silent)
    fi
    
    if [[ "$gpu_type" != "nvidia" ]]; then
        log::info "GPU type: $gpu_type (not NVIDIA - skipping validation)"
        return 0  # Skip validation for non-NVIDIA
    fi
    
    log::header "🔍 Validating NVIDIA GPU Requirements"
    
    # Check 1: NVIDIA drivers
    if ! system::is_command "nvidia-smi"; then
        log::error "❌ NVIDIA drivers not installed"
        log::info "Install NVIDIA drivers first: https://www.nvidia.com/Download/index.aspx"
        return 1
    fi
    
    # Test driver functionality
    if ! nvidia-smi >/dev/null 2>&1; then
        log::error "❌ NVIDIA drivers not functional"
        log::info "Check driver installation and system reboot"
        return 1
    fi
    
    log::success "✅ NVIDIA drivers installed and functional"
    
    # Check 2: Container toolkit
    if ! comfyui::is_nvidia_runtime_installed; then
        log::warn "⚠️  NVIDIA Container Toolkit not installed"
        return 2  # Installable issue
    fi
    log::success "✅ NVIDIA Container Toolkit installed"
    
    # Check 3: Docker runtime configuration
    if ! comfyui::is_docker_nvidia_configured; then
        log::warn "⚠️  Docker NVIDIA runtime not configured"
        return 3  # Configuration issue
    fi
    
    log::success "✅ Docker NVIDIA runtime configured"
    
    # Check 4: Functional test
    if ! comfyui::test_nvidia_runtime; then
        log::error "❌ NVIDIA runtime not functional"
        return 4  # Runtime issue
    fi
    
    log::success "✅ All NVIDIA requirements validated"
    return 0
}

#######################################
# Install NVIDIA Container Runtime automatically
# Returns: 0 if successful, 1 otherwise
#######################################
comfyui::install_nvidia_runtime() {
    log::header "📦 Installing NVIDIA Container Runtime"
    
    # Check if already installed
    if comfyui::is_nvidia_runtime_installed && comfyui::is_docker_nvidia_configured; then
        log::info "NVIDIA Container Runtime already installed and configured"
        return 0
    fi
    
    # Detect OS/Distribution
    local os_info
    os_info=$(comfyui::detect_os_distribution)
    log::info "Detected OS: $os_info"
    
    case "$os_info" in
        "ubuntu"|"debian")
            comfyui::install_nvidia_runtime_apt
            ;;
        "centos"|"rhel"|"fedora")
            comfyui::install_nvidia_runtime_yum
            ;;
        "arch")
            comfyui::install_nvidia_runtime_pacman
            ;;
        *)
            log::error "Unsupported OS for automatic installation: $os_info"
            comfyui::show_manual_installation_guide
            return 1
            ;;
    esac
}

#######################################
# Install NVIDIA Container Runtime on Ubuntu/Debian
# Returns: 0 if successful, 1 otherwise
#######################################
comfyui::install_nvidia_runtime_apt() {
    log::info "Installing NVIDIA Container Toolkit (Ubuntu/Debian)..."
    
    # Start rollback context for this operation
    resources::start_rollback_context "install_nvidia_runtime_apt"
    
    # Check for required commands
    if ! system::is_command "curl"; then
        log::error "curl is required for installation"
        return 1
    fi
    
    # Add rollback actions for each step
    resources::add_rollback_action \
        "Remove NVIDIA container toolkit repository key" \
        "sudo rm -f /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg" \
        5
    
    resources::add_rollback_action \
        "Remove NVIDIA container toolkit repository" \
        "sudo rm -f /etc/apt/sources.list.d/nvidia-container-toolkit.list" \
        10
    
    # Install repository GPG key
    log::info "Adding NVIDIA repository key..."
    if ! curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
         sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg; then
        log::error "Failed to add NVIDIA repository key"
        return 1
    fi
    
    # Add repository
    log::info "Adding NVIDIA repository..."
    if ! curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
         sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
         sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list >/dev/null; then
        log::error "Failed to add NVIDIA repository"
        return 1
    fi
    
    # Update package lists
    log::info "Updating package lists..."
    if ! sudo apt-get update >/dev/null 2>&1; then
        log::error "Failed to update package lists"
        return 1
    fi
    
    # Install NVIDIA Container Toolkit
    log::info "Installing NVIDIA Container Toolkit..."
    if ! sudo apt-get install -y nvidia-container-toolkit >/dev/null 2>&1; then
        log::error "Failed to install NVIDIA Container Toolkit"
        return 1
    fi
    
    # Configure Docker runtime
    log::info "Configuring Docker runtime..."
    if ! sudo nvidia-ctk runtime configure --runtime=docker >/dev/null 2>&1; then
        log::error "Failed to configure Docker runtime"
        return 1
    fi
    
    # Restart Docker
    log::info "Restarting Docker daemon..."
    if ! sudo systemctl restart docker; then
        log::error "Failed to restart Docker daemon"
        return 1
    fi
    
    # Wait for Docker to be ready
    sleep 5
    
    # Validate installation
    if comfyui::test_nvidia_runtime; then
        log::success "✅ NVIDIA Container Runtime installed successfully"
        # Clear rollback actions on success
        ROLLBACK_ACTIONS=()
        OPERATION_ID=""
        return 0
    else
        log::error "NVIDIA Container Runtime installation failed validation"
        return 1
    fi
}

#######################################
# Install NVIDIA Container Runtime on CentOS/RHEL/Fedora
# Returns: 0 if successful, 1 otherwise
#######################################
comfyui::install_nvidia_runtime_yum() {
    log::info "Installing NVIDIA Container Toolkit (CentOS/RHEL/Fedora)..."
    
    # Start rollback context
    resources::start_rollback_context "install_nvidia_runtime_yum"
    
    # Detect package manager
    local pkg_mgr
    if system::is_command "dnf"; then
        pkg_mgr="dnf"
    elif system::is_command "yum"; then
        pkg_mgr="yum"
    else
        log::error "No supported package manager found (yum/dnf)"
        return 1
    fi
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove NVIDIA container toolkit repository" \
        "sudo rm -f /etc/yum.repos.d/nvidia-container-toolkit.repo" \
        5
    
    # Add repository
    log::info "Adding NVIDIA repository..."
    if ! curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \
         sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo >/dev/null; then
        log::error "Failed to add NVIDIA repository"
        return 1
    fi
    
    # Install NVIDIA Container Toolkit
    log::info "Installing NVIDIA Container Toolkit..."
    if ! sudo $pkg_mgr install -y nvidia-container-toolkit >/dev/null 2>&1; then
        log::error "Failed to install NVIDIA Container Toolkit"
        return 1
    fi
    
    # Configure and restart Docker
    log::info "Configuring Docker runtime..."
    if ! sudo nvidia-ctk runtime configure --runtime=docker >/dev/null 2>&1; then
        log::error "Failed to configure Docker runtime"
        return 1
    fi
    
    log::info "Restarting Docker daemon..."
    if ! sudo systemctl restart docker; then
        log::error "Failed to restart Docker daemon"
        return 1
    fi
    
    sleep 5
    
    # Validate installation
    if comfyui::test_nvidia_runtime; then
        log::success "✅ NVIDIA Container Runtime installed successfully"
        ROLLBACK_ACTIONS=()
        OPERATION_ID=""
        return 0
    else
        log::error "NVIDIA Container Runtime installation failed validation"
        return 1
    fi
}

#######################################
# Install NVIDIA Container Runtime on Arch Linux
# Returns: 0 if successful, 1 otherwise
#######################################
comfyui::install_nvidia_runtime_pacman() {
    log::info "Installing NVIDIA Container Toolkit (Arch Linux)..."
    
    # Note: Arch Linux nvidia-container-toolkit is available in AUR
    log::warn "Arch Linux requires manual installation from AUR"
    log::info "Install nvidia-container-toolkit from AUR:"
    log::info "  yay -S nvidia-container-toolkit"
    log::info "  or use your preferred AUR helper"
    
    comfyui::show_manual_installation_guide
    return 1
}

#######################################
# Show manual installation guide for NVIDIA Container Runtime
#######################################
comfyui::show_manual_installation_guide() {
    cat << 'EOF'

=== Manual NVIDIA Container Runtime Installation ===

Prerequisites:
- NVIDIA GPU with compatible drivers installed
- Docker installed and running
- Sudo privileges

Ubuntu/Debian:
  curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
    sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
  
  curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
    sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
  
  sudo apt-get update
  sudo apt-get install -y nvidia-container-toolkit
  sudo nvidia-ctk runtime configure --runtime=docker
  sudo systemctl restart docker

CentOS/RHEL/Fedora:
  curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \
    sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo
  
  sudo yum install -y nvidia-container-toolkit  # or dnf
  sudo nvidia-ctk runtime configure --runtime=docker
  sudo systemctl restart docker

Arch Linux:
  yay -S nvidia-container-toolkit
  sudo nvidia-ctk runtime configure --runtime=docker
  sudo systemctl restart docker

Verification:
  docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi

For more information:
  https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html

EOF
}

#######################################
# Handle NVIDIA requirements with user interaction
# Arguments: $1 - GPU type
# Returns: 0 if requirements satisfied, 1 otherwise
#######################################
comfyui::handle_nvidia_requirements() {
    local gpu_type="$1"
    local validation_result
    
    echo "DEBUG: Starting handle_nvidia_requirements with gpu_type=$gpu_type"
    
    # Validate current state
    echo "DEBUG: About to call validate_nvidia_requirements"
    comfyui::validate_nvidia_requirements "$gpu_type"
    validation_result=$?
    
    echo "DEBUG: validation_result=$validation_result"
    
    case "$validation_result" in
        0)
            log::success "✅ NVIDIA requirements satisfied"
            return 0
            ;;
        1)
            log::error "❌ NVIDIA drivers not installed"
            comfyui::show_driver_installation_guide
            return 1
            ;;
        2|3)
            log::warn "⚠️  NVIDIA Container Runtime missing or misconfigured"
            echo "DEBUG: About to call prompt_runtime_installation"
            comfyui::prompt_runtime_installation
            local prompt_result=$?
            echo "DEBUG: prompt_runtime_installation returned $prompt_result"
            return $prompt_result
            ;;
        4)
            log::error "❌ NVIDIA runtime not functional"
            comfyui::show_troubleshooting_guide
            return 1
            ;;
        *)
            log::error "Unknown validation error"
            return 1
            ;;
    esac
}

#######################################
# Prompt user for NVIDIA runtime installation
# Returns: 0 if successful, 1 otherwise
#######################################
comfyui::prompt_runtime_installation() {
    echo
    log::info "ComfyUI requires NVIDIA Container Runtime for GPU support."
    log::info "This allows Docker containers to access your GPU."
    echo
    
    local choice
    
    if flow::is_yes "$YES"; then
        # Auto-install in non-interactive mode
        log::info "Auto-installing NVIDIA Container Runtime..."
        comfyui::install_nvidia_runtime
        return $?
    elif [[ -n "${COMFYUI_NVIDIA_CHOICE:-}" ]]; then
        # Use provided choice (for testing/automation)
        choice="$COMFYUI_NVIDIA_CHOICE"
        log::info "Using provided choice: $choice"
        echo "DEBUG: About to process choice $choice"
    else
        # Interactive prompt
        echo "Options:"
        echo "  1) Auto-install NVIDIA Container Runtime (requires sudo)"
        echo "  2) Show manual installation instructions"
        echo "  3) Continue with CPU mode instead"
        echo "  4) Cancel installation"
        echo
        
        read -p "Choose option [1-4]: " -r choice
    fi
        
    case "$choice" in
            1)
                comfyui::install_nvidia_runtime
                return $?
                ;;
            2)
                comfyui::show_manual_installation_guide
                echo
                read -p "Press Enter after completing manual setup..."
                echo
                # Re-validate after manual setup
                comfyui::validate_nvidia_requirements "nvidia"
                return $?
                ;;
            3)
                log::info "Switching to CPU mode..."
                echo "DEBUG: Returning 2 for CPU mode switch"
                return 2  # Special return code for CPU mode switch
                ;;
            4)
                log::info "Installation cancelled"
                return 1
                ;;
            *)
                log::error "Invalid choice"
                return 1
                ;;
    esac
}

#######################################
# Prompt user about CPU mode fallback
# Returns: 0 if yes, 1 if no
#######################################
comfyui::prompt_cpu_fallback() {
    echo
    log::warn "GPU mode setup failed. Would you like to continue with CPU mode?"
    log::info "Note: CPU mode will be significantly slower for image generation."
    echo
    
    if flow::is_yes "$YES"; then
        return 0  # Auto-accept in non-interactive mode
    else
        read -p "Continue with CPU mode? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            return 0
        else
            return 1
        fi
    fi
}

#######################################
# Show NVIDIA driver installation guide
#######################################
comfyui::show_driver_installation_guide() {
    cat << 'EOF'

=== NVIDIA Driver Installation Required ===

ComfyUI requires NVIDIA drivers to be installed first.

Installation options:

Ubuntu/Debian:
  # Automatic driver installation
  sudo ubuntu-drivers autoinstall
  # OR install specific driver version
  sudo apt install nvidia-driver-535

  # Reboot after installation
  sudo reboot

CentOS/RHEL/Fedora:
  # Enable EPEL repository (RHEL/CentOS)
  sudo yum install epel-release
  
  # Install NVIDIA drivers
  sudo yum install nvidia-driver nvidia-settings
  # OR use dnf on newer systems
  sudo dnf install nvidia-driver nvidia-settings

  # Reboot after installation
  sudo reboot

Manual Download:
  Visit: https://www.nvidia.com/Download/index.aspx
  Select your GPU model and download the appropriate driver

Verification:
  After installation and reboot, test with:
  nvidia-smi

EOF
}

#######################################
# Show troubleshooting guide for NVIDIA runtime issues
#######################################
comfyui::show_troubleshooting_guide() {
    cat << 'EOF'

=== NVIDIA Runtime Troubleshooting ===

The NVIDIA Container Runtime is installed but not working properly.

Common solutions:

1. Restart Docker daemon:
   sudo systemctl restart docker

2. Check Docker daemon configuration:
   docker info | grep nvidia

3. Test NVIDIA runtime manually:
   docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi

4. Check NVIDIA driver status:
   nvidia-smi

5. Verify container toolkit configuration:
   sudo nvidia-ctk runtime configure --runtime=docker
   sudo systemctl restart docker

6. Check Docker logs:
   sudo journalctl -u docker.service

If problems persist:
- Reboot the system
- Reinstall NVIDIA drivers
- Reinstall NVIDIA Container Toolkit

For detailed troubleshooting:
https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/troubleshooting.html

EOF
}

#######################################
# Handle NVIDIA setup failure with recovery options
# Arguments: $1 - error type
# Returns: 1 (always fails, provides guidance)
#######################################
comfyui::handle_nvidia_failure() {
    local error_type="$1"
    
    log::error "NVIDIA setup failed: $error_type"
    echo
    
    case "$error_type" in
        "drivers")
            log::info "Install NVIDIA drivers first:"
            log::info "  Ubuntu: sudo ubuntu-drivers autoinstall"
            log::info "  Visit: https://www.nvidia.com/Download/index.aspx"
            ;;
        "permissions")
            log::info "Fix Docker permissions:"
            log::info "  sudo usermod -aG docker $USER"
            log::info "  Then log out and back in"
            ;;
        "runtime")
            log::info "NVIDIA Container Runtime installation failed"
            log::info "Try the manual installation guide above"
            ;;
        "network")
            log::info "Network connectivity issue"
            log::info "Check internet connection and try again"
            ;;
    esac
    
    echo
    log::info "Alternative options:"
    log::info "  1) Use CPU mode: --gpu cpu"
    log::info "  2) Use remote GPU service"
    log::info "  3) Install on a different machine"
    
    return 1
}

#######################################
# Get GPU information
#######################################
comfyui::get_gpu_info() {
    echo "=== GPU Information ==="
    
    # Detect GPU type using silent version for clean output
    local gpu_type
    gpu_type=$(comfyui::detect_gpu_silent)
    
    echo "Detected GPU Type: $gpu_type"
    echo
    
    case "$gpu_type" in
        "nvidia")
            if system::is_command "nvidia-smi"; then
                echo "NVIDIA GPU Details:"
                nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader
                echo
                echo "CUDA Version:"
                nvidia-smi | grep "CUDA Version" | awk '{print $9}'
            fi
            ;;
        "amd")
            echo "AMD GPU Details:"
            if system::is_command "rocm-smi"; then
                rocm-smi --showproductname
            else
                lspci | grep -i "vga\|3d\|display" | grep -i "amd"
            fi
            ;;
        "cpu")
            echo "CPU Mode - No GPU acceleration available"
            echo "CPU Info:"
            lscpu | grep "Model name" | cut -d':' -f2 | xargs
            ;;
    esac
}

#######################################
# Check if ComfyUI container exists
# Returns: 0 if exists, 1 otherwise
#######################################
comfyui::container_exists() {
    docker::run ps -a --format '{{.Names}}' | grep -q "^${COMFYUI_CONTAINER_NAME}$"
}

#######################################
# Check if ComfyUI is running
# Returns: 0 if running, 1 otherwise
#######################################
comfyui::is_running() {
    docker::run ps --format '{{.Names}}' | grep -q "^${COMFYUI_CONTAINER_NAME}$"
}

#######################################
# Check if ComfyUI API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
comfyui::is_healthy() {
    # Add a quick mode parameter (default: false)
    local quick_mode="${1:-false}"
    
    if system::is_command "curl"; then
        # AI-dock containers may take time to fully initialize
        # Try multiple times with different endpoints
        local attempts=3
        local timeout=15
        local sleep_time=10
        
        # Quick mode for status checks
        if [[ "$quick_mode" == "true" ]]; then
            attempts=1
            timeout=3
            sleep_time=0
        fi
        
        for ((i=1; i<=attempts; i++)); do
            # Try root endpoint first (Caddy proxy should respond)
            local response
            response=$(curl -f -s --max-time "$timeout" "$COMFYUI_BASE_URL/" 2>/dev/null)
            if [[ $? -eq 0 ]]; then
                return 0
            fi
            
            # Try history endpoint as fallback
            response=$(curl -f -s --max-time "$timeout" "$COMFYUI_BASE_URL/history" 2>/dev/null)
            if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
                # Check if response is valid JSON (even empty {} is valid)
                if echo "$response" | jq . >/dev/null 2>&1; then
                    return 0
                fi
            fi
            
            # Wait before retry (except on last attempt)
            if [[ $i -lt $attempts ]] && [[ $sleep_time -gt 0 ]]; then
                sleep "$sleep_time"
            fi
        done
    fi
    return 1
}

#######################################
# Create ComfyUI directories
#######################################
comfyui::create_directories() {
    log::info "Creating ComfyUI directories..."
    
    # Create all required directories
    local dirs=(
        "$COMFYUI_DATA_DIR"
        "$COMFYUI_MODELS_DIR"
        "$COMFYUI_MODELS_DIR/checkpoints"
        "$COMFYUI_MODELS_DIR/vae"
        "$COMFYUI_MODELS_DIR/loras"
        "$COMFYUI_MODELS_DIR/embeddings"
        "$COMFYUI_MODELS_DIR/controlnet"
        "$COMFYUI_MODELS_DIR/clip"
        "$COMFYUI_MODELS_DIR/clip_vision"
        "$COMFYUI_MODELS_DIR/upscale_models"
        "$COMFYUI_CUSTOM_NODES_DIR"
        "$COMFYUI_OUTPUTS_DIR"
        "$COMFYUI_WORKFLOWS_DIR"
        "$COMFYUI_INPUT_DIR"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir" || {
            log::error "Failed to create directory: $dir"
            return 1
        }
    done
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove ComfyUI directories" \
        "rm -rf $COMFYUI_DATA_DIR 2>/dev/null || true" \
        10
    
    log::success "ComfyUI directories created"
    return 0
}

#######################################
# Start ComfyUI container
#######################################
comfyui::start_container() {
    local gpu_type="$1"
    
    log::info "Starting ComfyUI container..."
    
    # Remove existing container if it exists
    if comfyui::container_exists; then
        log::info "Removing existing ComfyUI container..."
        docker::run stop "$COMFYUI_CONTAINER_NAME" 2>/dev/null || true
        docker::run rm "$COMFYUI_CONTAINER_NAME" 2>/dev/null || true
    fi
    
    # Build docker run arguments
    local -a docker_args=(
        "run" "-d"
        "--name" "$COMFYUI_CONTAINER_NAME"
        "-p" "${COMFYUI_PORT}:8188"
        "--restart" "unless-stopped"
    )
    
    # GPU configuration
    case "$gpu_type" in
        "nvidia")
            docker_args+=("--gpus" "all" "-e" "NVIDIA_VISIBLE_DEVICES=all")
            ;;
        "amd")
            docker_args+=(
                "--device=/dev/kfd"
                "--device=/dev/dri"
                "--group-add" "video"
                "-e" "HSA_OVERRIDE_GFX_VERSION=10.3.0"
            )
            ;;
        "cpu")
            docker_args+=("-e" "CUDA_VISIBLE_DEVICES=")
            docker_args+=("-e" "TORCH_DEVICE=cpu")
            docker_args+=("-e" "FORCE_CPU=1")
            docker_args+=("-e" "PYTORCH_CUDA_ALLOC_CONF=")
            docker_args+=("-e" "TORCH_CUDA_AVAILABLE=false")
            log::warn "Running in CPU mode - generation will be slower"
            ;;
    esac
    
    # Volume mounts
    docker_args+=(
        "-v" "${COMFYUI_MODELS_DIR}:/comfyui/models"
        "-v" "${COMFYUI_CUSTOM_NODES_DIR}:/comfyui/custom_nodes"
        "-v" "${COMFYUI_OUTPUTS_DIR}:/comfyui/output"
        "-v" "${COMFYUI_WORKFLOWS_DIR}:/comfyui/workflows"
        "-v" "${COMFYUI_INPUT_DIR}:/comfyui/input"
    )
    
    # Environment variables
    docker_args+=("-e" "COMFYUI_PORT_HOST=${COMFYUI_PORT}")  # AI-dock needs this to match host port
    docker_args+=("-e" "AUTO_UPDATE=false")  # Skip auto-update to speed startup
    
    # Configure CLI args based on GPU type
    if [[ "$gpu_type" == "cpu" ]]; then
        docker_args+=("-e" "CLI_ARGS=--cpu")
        docker_args+=("-m" "8g")  # Limit to 8GB RAM
    else
        docker_args+=("-e" "CLI_ARGS=")
    fi
    
    # Image selection based on GPU type
    if [[ "$gpu_type" == "cpu" ]]; then
        docker_args+=("$COMFYUI_IMAGE_CPU")
    else
        docker_args+=("$COMFYUI_IMAGE_GPU")
    fi
    
    # Execute docker run with all arguments
    local docker_output
    docker_output=$(docker::run "${docker_args[@]}" 2>&1)
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "ComfyUI container started"
        
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove ComfyUI container" \
            "docker::run stop $COMFYUI_CONTAINER_NAME 2>/dev/null; docker::run rm $COMFYUI_CONTAINER_NAME 2>/dev/null || true" \
            20
        
        return 0
    else
        log::error "Failed to start ComfyUI container"
        if [[ -n "$docker_output" ]]; then
            log::error "Docker error: $docker_output"
        fi
        return 1
    fi
}

#######################################
# Download default models
#######################################
comfyui::download_default_models() {
    log::header "📥 Downloading Default Models"
    
    # Model URLs (using HuggingFace for reliability)
    local sdxl_base_url="https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0/resolve/main/sd_xl_base_1.0.safetensors"
    local sdxl_vae_url="https://huggingface.co/stabilityai/sdxl-vae/resolve/main/sdxl_vae.safetensors"
    
    # Check if models already exist
    local need_download=0
    
    if [[ ! -f "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT" ]]; then
        log::info "SDXL base model not found"
        need_download=1
    else
        log::info "✓ SDXL base model already exists"
    fi
    
    if [[ ! -f "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE" ]]; then
        log::info "SDXL VAE not found"
        need_download=1
    else
        log::info "✓ SDXL VAE already exists"
    fi
    
    if [[ "$need_download" -eq 0 ]]; then
        log::success "All default models are already downloaded"
        return 0
    fi
    
    # Check disk space (models are ~6.5GB each)
    if ! comfyui::check_disk_space 15; then
        log::error "Insufficient disk space for model downloads"
        return 1
    fi
    
    # Download SDXL base model
    if [[ ! -f "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT" ]]; then
        log::info "Downloading SDXL base model (~6.5GB)..."
        log::info "This may take a while depending on your internet connection"
        
        if system::is_command "wget"; then
            wget -c -O "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT.tmp" \
                --progress=bar:force \
                "$sdxl_base_url" || {
                log::error "Failed to download SDXL base model"
                rm -f "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT.tmp"
                return 1
            }
        elif system::is_command "curl"; then
            curl -L -C - -o "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT.tmp" \
                --progress-bar \
                "$sdxl_base_url" || {
                log::error "Failed to download SDXL base model"
                rm -f "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT.tmp"
                return 1
            }
        else
            log::error "Neither wget nor curl is available for downloading"
            return 1
        fi
        
        # Move to final location
        mv "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT.tmp" \
           "$COMFYUI_MODELS_DIR/checkpoints/$DEFAULT_CHECKPOINT"
        log::success "✓ SDXL base model downloaded"
    fi
    
    # Download SDXL VAE
    if [[ ! -f "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE" ]]; then
        log::info "Downloading SDXL VAE (~335MB)..."
        
        if system::is_command "wget"; then
            wget -c -O "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE.tmp" \
                --progress=bar:force \
                "$sdxl_vae_url" || {
                log::error "Failed to download SDXL VAE"
                rm -f "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE.tmp"
                return 1
            }
        elif system::is_command "curl"; then
            curl -L -C - -o "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE.tmp" \
                --progress-bar \
                "$sdxl_vae_url" || {
                log::error "Failed to download SDXL VAE"
                rm -f "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE.tmp"
                return 1
            }
        fi
        
        # Move to final location
        mv "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE.tmp" \
           "$COMFYUI_MODELS_DIR/vae/$DEFAULT_VAE"
        log::success "✓ SDXL VAE downloaded"
    fi
    
    log::success "✅ Default models downloaded successfully"
    return 0
}

#######################################
# Update Vrooli configuration
#######################################
comfyui::update_config() {
    local gpu_type="$1"
    
    # Determine image based on GPU type
    local image_name
    if [[ "$gpu_type" == "cpu" ]]; then
        image_name="$COMFYUI_IMAGE_CPU"
    else
        image_name="$COMFYUI_IMAGE_GPU"
    fi
    
    # Create JSON with ComfyUI configuration
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "workflows": true,
        "api": true,
        "gpu": $([ "$gpu_type" != "cpu" ] && echo "true" || echo "false"),
        "models": {
            "checkpoints": ["$DEFAULT_CHECKPOINT"],
            "vae": ["$DEFAULT_VAE"],
            "loras": []
        }
    },
    "ui": {
        "endpoint": "/",
        "port": "$COMFYUI_PORT"
    },
    "api": {
        "prompt": "/prompt",
        "history": "/history",
        "view": "/view",
        "upload": "/upload/image",
        "system_stats": "/system_stats"
    },
    "paths": {
        "models": "$COMFYUI_MODELS_DIR",
        "outputs": "$COMFYUI_OUTPUTS_DIR",
        "custom_nodes": "$COMFYUI_CUSTOM_NODES_DIR",
        "workflows": "$COMFYUI_WORKFLOWS_DIR"
    },
    "hardware": {
        "gpu_type": "$gpu_type"
    },
    "container": {
        "name": "$COMFYUI_CONTAINER_NAME",
        "image": "$image_name"
    }
}
EOF
)
    
    resources::update_config "automation" "comfyui" "$COMFYUI_BASE_URL" "$additional_config"
}

#######################################
# Complete ComfyUI installation
#######################################
comfyui::install() {
    log::header "🎨 Installing ComfyUI AI Image Generation Platform (Docker)"
    
    # Start rollback context
    resources::start_rollback_context "install_comfyui_docker"
    
    # Check if already installed
    if comfyui::container_exists && comfyui::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "ComfyUI is already installed and running"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Check Docker
    if ! comfyui::check_docker; then
        return 1
    fi
    
    # Check disk space
    if ! comfyui::check_disk_space; then
        return 1
    fi
    
    # Validate port assignment
    if ! resources::validate_port "comfyui" "$COMFYUI_PORT"; then
        log::error "Port validation failed for ComfyUI"
        log::info "You can set a custom port with: export COMFYUI_CUSTOM_PORT=<port>"
        return 1
    fi
    
    # Detect GPU (with logging)
    local gpu_type
    gpu_type=$(comfyui::detect_gpu)
    log::info "GPU Configuration: $gpu_type"
    
    # Handle NVIDIA requirements for GPU mode
    if [[ "$gpu_type" == "nvidia" ]]; then
        local nvidia_result
        comfyui::handle_nvidia_requirements "$gpu_type"
        nvidia_result=$?
        
        case "$nvidia_result" in
            0)
                log::success "✅ NVIDIA requirements satisfied"
                ;;
            2)
                # User chose CPU mode fallback
                log::info "Switching to CPU mode as requested"
                gpu_type="cpu"
                ;;
            *)
                # NVIDIA setup failed, ask about CPU fallback
                if comfyui::prompt_cpu_fallback; then
                    log::info "Continuing with CPU mode"
                    gpu_type="cpu"
                else
                    log::info "Installation cancelled"
                    return 1
                fi
                ;;
        esac
    fi
    
    log::info "Final GPU configuration: $gpu_type"
    
    # Set image based on GPU type
    local COMFYUI_IMAGE
    if [[ "$gpu_type" == "cpu" ]]; then
        COMFYUI_IMAGE="$COMFYUI_IMAGE_CPU"
    else
        COMFYUI_IMAGE="$COMFYUI_IMAGE_GPU"
    fi
    
    # Create directories
    if ! comfyui::create_directories; then
        resources::handle_error \
            "Failed to create ComfyUI directories" \
            "system" \
            "Check directory permissions"
        return 1
    fi
    
    # Pull Docker image
    log::info "Pulling ComfyUI Docker image..."
    if ! docker::run pull "$COMFYUI_IMAGE"; then
        resources::handle_error \
            "Failed to pull ComfyUI image" \
            "network" \
            "Check internet connection and Docker Hub access"
        return 1
    fi
    
    # Start container
    if ! comfyui::start_container "$gpu_type"; then
        resources::handle_error \
            "Failed to start ComfyUI container" \
            "system" \
            "Check Docker logs: docker::run logs $COMFYUI_CONTAINER_NAME"
        return 1
    fi
    
    # Wait for service to be ready (extended timeout for AI-dock initialization)
    log::info "Waiting for ComfyUI to start (this may take 3-5 minutes for initial setup)..."
    if resources::wait_for_service "comfyui" "$COMFYUI_PORT" 300; then
        if comfyui::is_healthy; then
            log::success "✅ ComfyUI is running and healthy on port $COMFYUI_PORT"
            
            # Update Vrooli configuration
            if ! comfyui::update_config "$gpu_type"; then
                log::warn "Failed to update Vrooli configuration"
                log::info "ComfyUI is installed but may need manual configuration in Vrooli"
            fi
            
            # Clear rollback context on success
            ROLLBACK_ACTIONS=()
            OPERATION_ID=""
            
            # Display access information
            echo
            log::header "🌐 ComfyUI Access Information"
            log::info "URL: $COMFYUI_BASE_URL"
            log::info "GPU Mode: $gpu_type"
            log::info "Models Directory: $COMFYUI_MODELS_DIR"
            log::info "Outputs Directory: $COMFYUI_OUTPUTS_DIR"
            
            # Show next steps
            echo
            log::header "🎯 Next Steps"
            log::info "1. Access ComfyUI at: $COMFYUI_BASE_URL"
            log::info "2. Download models: $0 --action download-models"
            log::info "3. Import workflows from: $COMFYUI_WORKFLOWS_DIR"
            log::info "4. Generated images saved to: $COMFYUI_OUTPUTS_DIR"
            
            # Model download reminder
            if [[ "$MODELS_TO_DOWNLOAD" == "default" ]]; then
                echo
                log::info "💡 Tip: Download default models with:"
                log::info "   $0 --action download-models --models default"
            fi
            
            return 0
        else
            # AI-dock containers have special initialization behavior
            log::info "✅ ComfyUI container started successfully"
            echo
            log::warn "⏳ INITIALIZATION IN PROGRESS (5-10 minutes)"
            log::info "AI-dock containers need time to:"
            log::info "  • Set up internal services (Caddy proxy, workspace sync)"
            log::info "  • Configure GPU drivers and CUDA libraries"
            log::info "  • Initialize ComfyUI and load extensions"
            echo
            log::info "📊 Current Status:"
            log::info "  • Container: Running ✅"
            log::info "  • Port $COMFYUI_PORT: Listening ✅"
            log::info "  • Web UI: Initializing... (shows 'Connection reset' until ready)"
            echo
            log::header "🔍 How to Monitor Progress"
            log::info "Watch the initialization in real-time:"
            log::info "  docker logs -f $COMFYUI_CONTAINER_NAME"
            echo
            log::info "Look for messages like:"
            log::info "  • 'ComfyUI is ready'"
            log::info "  • 'Starting ComfyUI on port 8188'"
            log::info "  • 'Execution provider: CUDA' (for GPU mode)"
            echo
            log::header "🌐 Access Information"
            log::info "Once initialization completes, access ComfyUI at:"
            log::info "  $COMFYUI_BASE_URL"
            echo
            log::success "Installation completed - ComfyUI is starting up!"
            return 0
        fi
    else
        resources::handle_error \
            "ComfyUI failed to start within timeout" \
            "system" \
            "Check container logs for errors"
        return 1
    fi
}

#######################################
# Stop ComfyUI
#######################################
comfyui::stop() {
    if ! comfyui::is_running; then
        log::info "ComfyUI is not running"
        return 0
    fi
    
    log::info "Stopping ComfyUI..."
    
    if docker::run stop "$COMFYUI_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "ComfyUI stopped"
    else
        log::error "Failed to stop ComfyUI"
        return 1
    fi
}

#######################################
# Start ComfyUI
#######################################
comfyui::start() {
    if comfyui::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "ComfyUI is already running on port $COMFYUI_PORT"
        return 0
    fi
    
    log::info "Starting ComfyUI..."
    
    # Check if container exists
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist. Run install first."
        return 1
    fi
    
    # Start container
    if docker::run start "$COMFYUI_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "ComfyUI started"
        
        # Wait for service to be ready
        if resources::wait_for_service "comfyui" "$COMFYUI_PORT" 30; then
            log::success "✅ ComfyUI is running on port $COMFYUI_PORT"
            log::info "Access ComfyUI at: $COMFYUI_BASE_URL"
        else
            log::warn "ComfyUI started but may not be fully ready yet"
        fi
    else
        log::error "Failed to start ComfyUI"
        return 1
    fi
}

#######################################
# Restart ComfyUI
#######################################
comfyui::restart() {
    log::info "Restarting ComfyUI..."
    comfyui::stop
    sleep 2
    comfyui::start
}

#######################################
# Show ComfyUI logs
#######################################
comfyui::logs() {
    if ! comfyui::container_exists; then
        log::error "ComfyUI container does not exist"
        return 1
    fi
    
    log::info "Showing ComfyUI logs (Ctrl+C to exit)..."
    docker::run logs -f "$COMFYUI_CONTAINER_NAME"
}

#######################################
# Show ComfyUI status
#######################################
comfyui::status() {
    log::header "📊 ComfyUI Status"
    
    # Check Docker
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        return 1
    fi
    
    # Try to check Docker status using group permissions if needed
    if ! docker::run version >/dev/null 2>&1; then
        if ! sg docker -c "docker version" >/dev/null 2>&1; then
            log::error "Docker daemon is not running or not accessible"
            return 1
        fi
    fi
    
    # Check container status
    if comfyui::container_exists; then
        if comfyui::is_running; then
            log::success "✅ ComfyUI container is running"
            
            # Get container stats
            local stats
            stats=$(docker::run stats "$COMFYUI_CONTAINER_NAME" --no-stream --format "CPU: {{.CPUPerc}} | Memory: {{.MemUsage}}" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                log::info "Resource usage: $stats"
            fi
            
            # Check health (use quick mode for status checks)
            if comfyui::is_healthy "true"; then
                log::success "✅ ComfyUI API is healthy"
                
                # Get basic API info (history endpoint as health indicator)
                local api_response
                api_response=$(curl -s "$COMFYUI_BASE_URL/history" 2>/dev/null || echo "{}")
                if [[ -n "$api_response" ]] && echo "$api_response" | jq . >/dev/null 2>&1; then
                    echo
                    log::info "API Status: Responsive"
                    local history_count
                    history_count=$(echo "$api_response" | jq 'length' 2>/dev/null || echo "0")
                    log::info "  Completed workflows: $history_count"
                fi
            else
                log::info "⏳ ComfyUI API is still initializing"
                log::info "  This is normal for AI-dock containers"
                log::info "  Check progress: docker logs -f $COMFYUI_CONTAINER_NAME"
            fi
            
            # Additional details
            echo
            log::info "ComfyUI Details:"
            log::info "  Web UI: $COMFYUI_BASE_URL"
            log::info "  Container: $COMFYUI_CONTAINER_NAME"
            
            # GPU info
            local gpu_type
            gpu_type=$(docker::run inspect "$COMFYUI_CONTAINER_NAME" --format='{{range .HostConfig.Devices}}{{.PathOnHost}}{{end}}' 2>/dev/null)
            if [[ -n "$gpu_type" ]]; then
                log::info "  GPU: Enabled"
            else
                log::info "  GPU: Disabled (CPU mode)"
            fi
            
            # Show logs command
            echo
            log::info "View logs: $0 --action logs"
        else
            log::warn "⚠️  ComfyUI container exists but is not running"
            log::info "Start with: $0 --action start"
        fi
    else
        log::error "❌ ComfyUI is not installed"
        log::info "Install with: $0 --action install"
    fi
}

#######################################
# Check if ComfyUI is ready to use
#######################################
comfyui::check_ready() {
    log::header "🔍 Checking ComfyUI Readiness"
    
    # Check if container exists
    if ! comfyui::container_exists; then
        log::error "ComfyUI is not installed"
        log::info "Install with: $0 --action install"
        return 1
    fi
    
    # Check if container is running
    if ! comfyui::is_running; then
        log::error "ComfyUI container is not running"
        log::info "Start with: $0 --action start"
        return 1
    fi
    
    log::info "Container Status: Running ✅"
    log::info "Checking API availability..."
    
    # Check API with detailed feedback
    local response
    response=$(curl -s -w "\n%{http_code}" --max-time 5 "$COMFYUI_BASE_URL/" 2>/dev/null)
    local http_code=$(echo "$response" | tail -n 1)
    local body=$(echo "$response" | head -n -1)
    
    if [[ "$http_code" == "200" ]] || [[ "$http_code" == "301" ]] || [[ "$http_code" == "302" ]]; then
        log::success "✅ ComfyUI is READY!"
        log::info "Access the web interface at: $COMFYUI_BASE_URL"
        return 0
    elif [[ -z "$http_code" ]] || [[ "$http_code" == "000" ]]; then
        log::warn "⏳ ComfyUI is still initializing..."
        echo
        log::info "This is normal for AI-dock containers. They need time to:"
        log::info "  • Set up Caddy reverse proxy"
        log::info "  • Configure GPU drivers"
        log::info "  • Initialize ComfyUI and extensions"
        echo
        log::info "💡 Tips:"
        log::info "  1. Watch live progress: docker logs -f $COMFYUI_CONTAINER_NAME"
        log::info "  2. Wait for 'ComfyUI is ready' message in logs"
        log::info "  3. Run this command again in a few minutes"
        return 1
    else
        log::warn "Unexpected response (HTTP $http_code)"
        log::info "Check logs: docker logs $COMFYUI_CONTAINER_NAME"
        return 1
    fi
}

#######################################
# Show ComfyUI information
#######################################
comfyui::info() {
    # Determine which image is currently in use
    local current_image="Unknown"
    if docker inspect "$COMFYUI_CONTAINER_NAME" &>/dev/null; then
        current_image=$(docker inspect "$COMFYUI_CONTAINER_NAME" --format='{{.Config.Image}}' 2>/dev/null || echo "Unknown")
    fi
    
    cat << EOF
=== ComfyUI Resource Information ===

ID: comfyui
Category: automation
Display Name: ComfyUI
Description: AI-powered image generation workflow platform

Service Details:
- Container Name: $COMFYUI_CONTAINER_NAME
- Service Port: $COMFYUI_PORT
- Service URL: $COMFYUI_BASE_URL
- Docker Image: $current_image
- Data Directory: $COMFYUI_DATA_DIR

Endpoints:
- Web UI: $COMFYUI_BASE_URL
- Queue Prompt: $COMFYUI_BASE_URL/prompt (POST)
- Get History: $COMFYUI_BASE_URL/history/{prompt_id} (GET)
- View Image: $COMFYUI_BASE_URL/view (GET)
- Upload Image: $COMFYUI_BASE_URL/upload/image (POST)
- System Stats: $COMFYUI_BASE_URL/system_stats (GET)
- WebSocket: ws://localhost:$COMFYUI_PORT/ws

Configuration:
- GPU Support: ${GPU_TYPE:-auto}
- Headless Mode: ${HEADLESS_MODE:-yes}
- Models Directory: $COMFYUI_MODELS_DIR
- Outputs Directory: $COMFYUI_OUTPUTS_DIR

ComfyUI Features:
- Node-based workflow editor
- Support for SD, SDXL, and custom models
- ControlNet, LoRA, and VAE support
- Batch processing
- Custom node development
- API for automation
- Real-time preview
- Workflow templates

Example Usage:
# Access the web UI
Open $COMFYUI_BASE_URL in your browser

# Execute a workflow via API
curl -X POST $COMFYUI_BASE_URL/prompt \\
  -H "Content-Type: application/json" \\
  -d @workflow.json

# Check system stats
curl $COMFYUI_BASE_URL/system_stats

# WebSocket for real-time updates
wscat -c ws://localhost:$COMFYUI_PORT/ws

For more information, visit: https://github.com/comfyanonymous/ComfyUI
EOF
}

#######################################
# Execute workflow via API
#######################################
comfyui::execute_workflow() {
    log::header "🚀 ComfyUI Workflow Execution"
    
    # Check if ComfyUI is running
    if ! comfyui::is_running; then
        log::error "ComfyUI is not running. Start it with: $0 --action start"
        return 1
    fi
    
    # Check workflow file
    if [[ -z "$WORKFLOW_FILE" ]]; then
        log::error "No workflow file specified"
        echo "Usage: $0 --action execute-workflow --workflow path/to/workflow.json"
        return 1
    fi
    
    if [[ ! -f "$WORKFLOW_FILE" ]]; then
        log::error "Workflow file not found: $WORKFLOW_FILE"
        return 1
    fi
    
    # Generate client ID if not provided
    if [[ -z "$CLIENT_ID" ]]; then
        CLIENT_ID="cli-$(date +%s)-$$"
    fi
    
    log::info "Loading workflow: $WORKFLOW_FILE"
    log::info "Client ID: $CLIENT_ID"
    
    # Read and validate workflow JSON
    local workflow_json
    workflow_json=$(cat "$WORKFLOW_FILE" 2>/dev/null)
    
    if ! echo "$workflow_json" | jq empty 2>/dev/null; then
        log::error "Invalid JSON in workflow file"
        return 1
    fi
    
    # Prepare prompt payload
    local prompt_payload
    prompt_payload=$(jq -n \
        --arg client_id "$CLIENT_ID" \
        --argjson prompt "$workflow_json" \
        '{
            "client_id": $client_id,
            "prompt": $prompt
        }')
    
    # Submit workflow to queue
    log::info "Submitting workflow to queue..."
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$prompt_payload" \
        "$COMFYUI_BASE_URL/prompt" 2>/dev/null)
    
    # Check for errors
    if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error')
        log::error "ComfyUI error: $error_msg"
        
        # Check for missing models
        if echo "$error_msg" | grep -qi "model.*not found\|missing.*model"; then
            log::info "Tip: Download default models with: $0 --action download-models"
        fi
        return 1
    fi
    
    # Extract prompt ID
    local prompt_id
    prompt_id=$(echo "$response" | jq -r '.prompt_id' 2>/dev/null)
    
    if [[ -z "$prompt_id" ]]; then
        log::error "Failed to get prompt ID from response"
        echo "Response: $response"
        return 1
    fi
    
    log::success "Workflow queued successfully"
    log::info "Prompt ID: $prompt_id"
    
    # Monitor execution progress
    log::info "Monitoring execution progress..."
    
    local check_count=0
    local max_checks=300  # 5 minutes timeout
    local completed=0
    
    while [[ $check_count -lt $max_checks ]]; do
        # Check history for completion
        local history_response
        history_response=$(curl -s "$COMFYUI_BASE_URL/history/$prompt_id" 2>/dev/null)
        
        if [[ -n "$history_response" ]] && [[ "$history_response" != "{}" ]]; then
            # Check if execution completed
            local status
            status=$(echo "$history_response" | jq -r ".[\"$prompt_id\"].status.status_str" 2>/dev/null)
            
            if [[ "$status" == "success" ]]; then
                completed=1
                break
            elif [[ "$status" == "error" ]]; then
                log::error "Workflow execution failed"
                local error_details
                error_details=$(echo "$history_response" | jq -r ".[\"$prompt_id\"].status.messages" 2>/dev/null)
                if [[ -n "$error_details" ]]; then
                    echo "Error details: $error_details"
                fi
                return 1
            fi
        fi
        
        # Show progress indicator
        echo -n "."
        sleep 1
        ((check_count++))
    done
    echo
    
    if [[ $completed -eq 0 ]]; then
        log::error "Workflow execution timed out"
        return 1
    fi
    
    log::success "✅ Workflow execution completed"
    
    # Extract output information
    local outputs
    outputs=$(echo "$history_response" | jq -r ".[\"$prompt_id\"].outputs" 2>/dev/null)
    
    if [[ -n "$outputs" ]] && [[ "$outputs" != "null" ]]; then
        # Find generated images
        local images
        images=$(echo "$outputs" | jq -r '.. | objects | select(has("images")) | .images[] | "\(.filename) (subfolder: \(.subfolder // "output"))"' 2>/dev/null)
        
        if [[ -n "$images" ]]; then
            echo
            log::header "📸 Generated Images"
            echo "$images" | while IFS= read -r image_info; do
                log::info "  - $image_info"
            done
            
            # Copy images to output directory if different from default
            if [[ "$OUTPUT_DIR" != "$COMFYUI_OUTPUTS_DIR" ]]; then
                mkdir -p "$OUTPUT_DIR"
                local image_count=0
                
                echo "$outputs" | jq -r '.. | objects | select(has("images")) | .images[] | .filename' 2>/dev/null | while IFS= read -r filename; do
                    if [[ -f "$COMFYUI_OUTPUTS_DIR/$filename" ]]; then
                        cp "$COMFYUI_OUTPUTS_DIR/$filename" "$OUTPUT_DIR/"
                        ((image_count++))
                    fi
                done
                
                log::info "Images copied to: $OUTPUT_DIR"
            else
                log::info "Images saved to: $COMFYUI_OUTPUTS_DIR"
            fi
        fi
    fi
    
    return 0
}

#######################################
# Import workflow
#######################################
comfyui::import_workflow() {
    log::header "📥 Import ComfyUI Workflow"
    
    # Check if workflow file is provided
    if [[ -z "$WORKFLOW_FILE" ]]; then
        log::error "No workflow file specified"
        echo "Usage: $0 --action import-workflow --workflow path/to/workflow.json"
        return 1
    fi
    
    if [[ ! -f "$WORKFLOW_FILE" ]]; then
        log::error "Workflow file not found: $WORKFLOW_FILE"
        return 1
    fi
    
    # Validate JSON
    if ! jq empty "$WORKFLOW_FILE" 2>/dev/null; then
        log::error "Invalid JSON in workflow file"
        return 1
    fi
    
    # Create workflows directory if it doesn't exist
    mkdir -p "$COMFYUI_WORKFLOWS_DIR"
    
    # Generate filename
    local basename
    basename=$(basename "$WORKFLOW_FILE")
    local dest_file="$COMFYUI_WORKFLOWS_DIR/$basename"
    
    # Check if file already exists
    if [[ -f "$dest_file" ]]; then
        log::warn "Workflow already exists: $basename"
        read -p "Overwrite? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Import cancelled"
            return 0
        fi
    fi
    
    # Copy workflow file
    if cp "$WORKFLOW_FILE" "$dest_file"; then
        log::success "✅ Workflow imported successfully"
        log::info "Saved to: $dest_file"
        
        # Analyze workflow for required models
        log::info "Analyzing workflow requirements..."
        
        # Extract model references from workflow
        local checkpoints
        checkpoints=$(jq -r '.. | objects | select(has("ckpt_name")) | .ckpt_name' "$dest_file" 2>/dev/null | sort -u)
        
        local vaes
        vaes=$(jq -r '.. | objects | select(has("vae_name")) | .vae_name' "$dest_file" 2>/dev/null | sort -u)
        
        local loras
        loras=$(jq -r '.. | objects | select(has("lora_name")) | .lora_name' "$dest_file" 2>/dev/null | sort -u)
        
        # Check which models are missing
        local missing_models=0
        
        if [[ -n "$checkpoints" ]]; then
            echo
            log::info "Required Checkpoints:"
            echo "$checkpoints" | while IFS= read -r model; do
                if [[ -f "$COMFYUI_MODELS_DIR/checkpoints/$model" ]]; then
                    log::success "  ✓ $model"
                else
                    log::warn "  ✗ $model (missing)"
                    missing_models=1
                fi
            done
        fi
        
        if [[ -n "$vaes" ]]; then
            echo
            log::info "Required VAEs:"
            echo "$vaes" | while IFS= read -r model; do
                if [[ -f "$COMFYUI_MODELS_DIR/vae/$model" ]]; then
                    log::success "  ✓ $model"
                else
                    log::warn "  ✗ $model (missing)"
                    missing_models=1
                fi
            done
        fi
        
        if [[ -n "$loras" ]]; then
            echo
            log::info "Required LoRAs:"
            echo "$loras" | while IFS= read -r model; do
                if [[ -f "$COMFYUI_MODELS_DIR/loras/$model" ]]; then
                    log::success "  ✓ $model"
                else
                    log::warn "  ✗ $model (missing)"
                    missing_models=1
                fi
            done
        fi
        
        if [[ $missing_models -eq 1 ]]; then
            echo
            log::warn "Some required models are missing"
            log::info "Download models manually or use: $0 --action download-models"
        fi
        
        echo
        log::info "To execute this workflow:"
        log::info "  $0 --action execute-workflow --workflow $dest_file"
        
        return 0
    else
        log::error "Failed to import workflow"
        return 1
    fi
}

#######################################
# List models
#######################################
comfyui::list_models() {
    log::header "📦 ComfyUI Models"
    
    if [[ ! -d "$COMFYUI_MODELS_DIR" ]]; then
        log::error "Models directory does not exist"
        return 1
    fi
    
    echo "Models Directory: $COMFYUI_MODELS_DIR"
    echo
    
    # List checkpoints
    echo "Checkpoints:"
    if [[ -d "$COMFYUI_MODELS_DIR/checkpoints" ]]; then
        local checkpoints=$(ls -1 "$COMFYUI_MODELS_DIR/checkpoints" 2>/dev/null | grep -E '\.(safetensors|ckpt|pt)$' || echo "  None")
        echo "$checkpoints" | sed 's/^/  /'
    else
        echo "  Directory not found"
    fi
    echo
    
    # List VAEs
    echo "VAE Models:"
    if [[ -d "$COMFYUI_MODELS_DIR/vae" ]]; then
        local vaes=$(ls -1 "$COMFYUI_MODELS_DIR/vae" 2>/dev/null | grep -E '\.(safetensors|ckpt|pt)$' || echo "  None")
        echo "$vaes" | sed 's/^/  /'
    else
        echo "  Directory not found"
    fi
    echo
    
    # List LoRAs
    echo "LoRA Models:"
    if [[ -d "$COMFYUI_MODELS_DIR/loras" ]]; then
        local loras=$(ls -1 "$COMFYUI_MODELS_DIR/loras" 2>/dev/null | grep -E '\.(safetensors|ckpt|pt)$' || echo "  None")
        echo "$loras" | sed 's/^/  /'
    else
        echo "  Directory not found"
    fi
}

#######################################
# Uninstall ComfyUI
#######################################
comfyui::uninstall() {
    log::header "🗑️  Uninstalling ComfyUI"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove ComfyUI and optionally delete all models and outputs"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop and remove container
    if comfyui::container_exists; then
        log::info "Removing ComfyUI container..."
        docker::run stop "$COMFYUI_CONTAINER_NAME" 2>/dev/null || true
        docker::run rm "$COMFYUI_CONTAINER_NAME" 2>/dev/null || true
        log::success "ComfyUI container removed"
    fi
    
    # Ask about data removal
    if [[ -d "$COMFYUI_DATA_DIR" ]]; then
        echo
        log::warn "ComfyUI data directory contains models and outputs"
        log::info "Directory size: $(du -sh "$COMFYUI_DATA_DIR" 2>/dev/null | cut -f1)"
        read -p "Remove ComfyUI data directory? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            # Backup outputs before removal
            if [[ -d "$COMFYUI_OUTPUTS_DIR" ]] && [[ "$(ls -A "$COMFYUI_OUTPUTS_DIR" 2>/dev/null)" ]]; then
                local backup_dir="$HOME/comfyui-outputs-backup-$(date +%Y%m%d-%H%M%S)"
                log::info "Backing up outputs to: $backup_dir"
                cp -r "$COMFYUI_OUTPUTS_DIR" "$backup_dir" 2>/dev/null || true
            fi
            
            log::info "Removing ComfyUI data directory..."
            rm -rf "$COMFYUI_DATA_DIR" 2>/dev/null || true
            log::info "Data directory removed"
        else
            log::info "Data directory preserved at: $COMFYUI_DATA_DIR"
        fi
    fi
    
    # Remove from Vrooli config
    resources::remove_config "automation" "comfyui"
    
    log::success "✅ ComfyUI uninstalled successfully"
}

#######################################
# Download models
#######################################
comfyui::download_models() {
    log::header "📥 Download ComfyUI Models"
    
    # Check if ComfyUI is installed
    if [[ ! -d "$COMFYUI_MODELS_DIR" ]]; then
        log::error "ComfyUI is not installed. Please install it first."
        return 1
    fi
    
    # Parse models to download
    local models="${MODELS_TO_DOWNLOAD:-default}"
    
    case "$models" in
        "default")
            comfyui::download_default_models
            ;;
        "none"|"skip")
            log::info "Skipping model download"
            ;;
        *)
            log::warn "Custom model download not yet implemented"
            log::info "Available options: default, none"
            echo
            log::info "To manually add models:"
            log::info "1. Download models from HuggingFace, CivitAI, or other sources"
            log::info "2. Place checkpoint files in: $COMFYUI_MODELS_DIR/checkpoints/"
            log::info "3. Place VAE files in: $COMFYUI_MODELS_DIR/vae/"
            log::info "4. Place LoRA files in: $COMFYUI_MODELS_DIR/loras/"
            ;;
    esac
    
    return 0
}

#######################################
# Main execution function
#######################################
comfyui::main() {
    comfyui::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            comfyui::install
            ;;
        "uninstall")
            comfyui::uninstall
            ;;
        "start")
            comfyui::start
            ;;
        "stop")
            comfyui::stop
            ;;
        "restart")
            comfyui::restart
            ;;
        "status")
            comfyui::status
            ;;
        "logs")
            comfyui::logs
            ;;
        "info")
            comfyui::info
            ;;
        "download-models")
            comfyui::download_models
            ;;
        "execute-workflow")
            comfyui::execute_workflow
            ;;
        "import-workflow")
            comfyui::import_workflow
            ;;
        "list-models")
            comfyui::list_models
            ;;
        "gpu-info")
            comfyui::get_gpu_info
            ;;
        "validate-nvidia")
            comfyui::validate_nvidia_requirements
            ;;
        "check-ready")
            comfyui::check_ready
            ;;
        *)
            log::error "Unknown action: $ACTION"
            comfyui::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    comfyui::main "$@"
fi