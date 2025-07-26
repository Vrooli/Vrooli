#!/usr/bin/env bash
# ComfyUI GPU Detection and Management
# Handles GPU detection, NVIDIA runtime setup, and GPU-specific configurations

#######################################
# Detect GPU type without logging (for use in conditionals)
#######################################
comfyui::detect_gpu_silent() {
    # Check for NVIDIA GPU
    if system::is_command "nvidia-smi" && nvidia-smi >/dev/null 2>&1; then
        echo "nvidia"
        return 0
    fi
    
    # Check for AMD GPU
    if system::is_command "rocm-smi" && rocm-smi >/dev/null 2>&1; then
        echo "amd"
        return 0
    fi
    
    # Check via lspci if available
    if system::is_command "lspci"; then
        if lspci 2>/dev/null | grep -iE "(nvidia|geforce|rtx|gtx|quadro)" >/dev/null; then
            echo "nvidia"
            return 0
        elif lspci 2>/dev/null | grep -iE "(amd|radeon|vega)" | grep -i vga >/dev/null; then
            echo "amd"
            return 0
        fi
    fi
    
    # Default to CPU
    echo "cpu"
    return 0
}

#######################################
# Detect GPU type with user interaction
#######################################
comfyui::detect_gpu() {
    log::info "Detecting GPU type..."
    
    local detected_gpu
    detected_gpu=$(comfyui::detect_gpu_silent)
    
    case "$detected_gpu" in
        nvidia)
            log::success "NVIDIA GPU detected"
            if nvidia-smi >/dev/null 2>&1; then
                log::info "NVIDIA driver is installed and functional"
                nvidia-smi --query-gpu=name,driver_version,memory.total --format=csv,noheader || true
            else
                log::warn "NVIDIA GPU detected but nvidia-smi not functional"
            fi
            ;;
        amd)
            log::success "AMD GPU detected"
            if system::is_command "rocm-smi"; then
                log::info "ROCm is installed"
                rocm-smi --showid --showproductname 2>/dev/null || true
            else
                log::warn "AMD GPU detected but ROCm tools not found"
            fi
            ;;
        cpu)
            log::warn "No supported GPU detected, will use CPU mode"
            log::info "Image generation will be significantly slower in CPU mode"
            ;;
    esac
    
    echo "$detected_gpu"
}

#######################################
# Detect OS distribution for package management
#######################################
comfyui::detect_os_distribution() {
    if [[ -f /etc/os-release ]]; then
        # shellcheck disable=SC1091
        source /etc/os-release
        case "$ID" in
            ubuntu|debian)
                echo "apt"
                ;;
            centos|rhel|fedora|rocky|almalinux)
                echo "yum"
                ;;
            arch|manjaro)
                echo "pacman"
                ;;
            *)
                echo "unknown"
                ;;
        esac
    else
        echo "unknown"
    fi
}

#######################################
# Check if NVIDIA Container Runtime is installed
#######################################
comfyui::is_nvidia_runtime_installed() {
    # Check if nvidia-container-toolkit is installed
    if system::is_command "nvidia-container-toolkit"; then
        return 0
    fi
    
    # Check if nvidia-container-runtime is installed
    if system::is_command "nvidia-container-runtime"; then
        return 0
    fi
    
    # Check Docker daemon configuration
    if docker::run info 2>/dev/null | grep -q "nvidia"; then
        return 0
    fi
    
    return 1
}

#######################################
# Check if Docker is configured for NVIDIA
#######################################
comfyui::is_docker_nvidia_configured() {
    # Method 1: Check docker info for nvidia runtime
    if docker::run info 2>/dev/null | grep -q "nvidia"; then
        return 0
    fi
    
    # Method 2: Check daemon.json
    if [[ -f /etc/docker/daemon.json ]]; then
        if grep -q "nvidia-container-runtime" /etc/docker/daemon.json 2>/dev/null; then
            return 0
        fi
    fi
    
    # Method 3: Try to run a test container
    if docker::run run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Test NVIDIA runtime functionality
#######################################
comfyui::test_nvidia_runtime() {
    log::info "Testing NVIDIA runtime..."
    
    # First check if runtime is available
    if ! docker::run info 2>/dev/null | grep -q nvidia; then
        log::debug "NVIDIA runtime not found in docker info"
        return 1
    fi
    
    # Try a simple GPU test
    log::info "Running NVIDIA GPU test container..."
    
    if docker::run run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi; then
        log::success "✅ NVIDIA GPU is accessible in Docker"
        return 0
    else
        log::error "❌ Failed to access GPU in Docker container"
        
        # Try alternative test
        log::info "Trying alternative GPU test..."
        if docker::run run --rm --runtime=nvidia nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi 2>/dev/null; then
            log::success "✅ NVIDIA GPU accessible with --runtime=nvidia flag"
            log::warn "Note: You may need to use --runtime=nvidia instead of --gpus all"
            return 0
        fi
        
        return 1
    fi
}

#######################################
# Comprehensive NVIDIA requirements validation
#######################################
comfyui::validate_nvidia_requirements() {
    log::header "🔍 Validating NVIDIA Requirements"
    
    local validation_passed=true
    
    # 1. Check NVIDIA driver
    log::info "Checking NVIDIA driver..."
    if nvidia-smi >/dev/null 2>&1; then
        local driver_version
        driver_version=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits | head -n1)
        log::success "✅ NVIDIA driver installed: $driver_version"
        
        # Show GPU information
        nvidia-smi --query-gpu=name,memory.total --format=csv,noheader | while IFS=',' read -r gpu_name gpu_memory; do
            log::info "   GPU: $gpu_name (${gpu_memory})"
        done
    else
        log::error "❌ NVIDIA driver not functional"
        log::info "   Install NVIDIA drivers from: https://www.nvidia.com/Download/index.aspx"
        validation_passed=false
    fi
    
    echo
    
    # 2. Check NVIDIA Container Runtime
    log::info "Checking NVIDIA Container Runtime..."
    if comfyui::is_nvidia_runtime_installed; then
        log::success "✅ NVIDIA Container Runtime is installed"
    else
        log::error "❌ NVIDIA Container Runtime not found"
        validation_passed=false
    fi
    
    # 3. Check Docker NVIDIA configuration
    log::info "Checking Docker NVIDIA configuration..."
    if comfyui::is_docker_nvidia_configured; then
        log::success "✅ Docker is configured for NVIDIA GPUs"
    else
        log::error "❌ Docker is not configured for NVIDIA GPUs"
        validation_passed=false
    fi
    
    # 4. Test GPU access in container
    if [[ "$validation_passed" == "true" ]]; then
        log::info "Testing GPU access in Docker container..."
        if comfyui::test_nvidia_runtime; then
            log::success "✅ GPU access in containers is working"
        else
            log::error "❌ Cannot access GPU in containers"
            validation_passed=false
        fi
    fi
    
    echo
    
    if [[ "$validation_passed" == "true" ]]; then
        log::success "🎉 All NVIDIA requirements validated successfully!"
        return 0
    else
        log::error "❌ NVIDIA validation failed"
        log::info "Run '$0 --action install' to attempt automatic setup"
        return 1
    fi
}

#######################################
# Install NVIDIA Container Runtime
#######################################
comfyui::install_nvidia_runtime() {
    local distro
    distro=$(comfyui::detect_os_distribution)
    
    log::info "Detected distribution type: $distro"
    
    case "$distro" in
        apt)
            comfyui::install_nvidia_runtime_apt
            ;;
        yum)
            comfyui::install_nvidia_runtime_yum
            ;;
        pacman)
            comfyui::install_nvidia_runtime_pacman
            ;;
        *)
            log::error "Unsupported distribution for automatic installation"
            return 1
            ;;
    esac
}

#######################################
# Install NVIDIA runtime on Ubuntu/Debian
#######################################
comfyui::install_nvidia_runtime_apt() {
    log::info "Installing NVIDIA Container Runtime for Ubuntu/Debian..."
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove NVIDIA Container Runtime repository" \
        "flow::maybe_run_sudo rm -f /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg /etc/apt/sources.list.d/nvidia-container-toolkit.list" \
        5
    
    # Set up the repository
    log::info "Setting up NVIDIA repository..."
    
    # Download and add GPG key
    if ! curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
         flow::maybe_run_sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg; then
        log::error "Failed to add NVIDIA GPG key"
        return 1
    fi
    
    # Add repository
    if ! curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
         sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
         flow::maybe_run_sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list >/dev/null; then
        log::error "Failed to add NVIDIA repository"
        return 1
    fi
    
    # Update package list
    log::info "Updating package list..."
    if ! flow::maybe_run_sudo apt-get update; then
        log::error "Failed to update package list"
        return 1
    fi
    
    # Install nvidia-container-toolkit
    log::info "Installing nvidia-container-toolkit..."
    if ! flow::maybe_run_sudo apt-get install -y nvidia-container-toolkit; then
        log::error "Failed to install nvidia-container-toolkit"
        return 1
    fi
    
    # Configure Docker
    log::info "Configuring Docker for NVIDIA runtime..."
    if ! flow::maybe_run_sudo nvidia-ctk runtime configure --runtime=docker; then
        log::error "Failed to configure Docker for NVIDIA"
        return 1
    fi
    
    # Restart Docker
    log::info "Restarting Docker service..."
    if ! flow::maybe_run_sudo systemctl restart docker; then
        log::error "Failed to restart Docker"
        return 1
    fi
    
    # Wait for Docker to be ready
    sleep 5
    
    log::success "NVIDIA Container Runtime installed successfully"
    return 0
}

#######################################
# Install NVIDIA runtime on CentOS/RHEL/Fedora
#######################################
comfyui::install_nvidia_runtime_yum() {
    log::info "Installing NVIDIA Container Runtime for CentOS/RHEL/Fedora..."
    
    # Set up the repository
    log::info "Setting up NVIDIA repository..."
    
    local repo_url="https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo"
    
    if ! curl -s -L "$repo_url" | flow::maybe_run_sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo >/dev/null; then
        log::error "Failed to add NVIDIA repository"
        return 1
    fi
    
    # Clean yum cache
    flow::maybe_run_sudo yum clean expire-cache
    
    # Install nvidia-container-toolkit
    log::info "Installing nvidia-container-toolkit..."
    if ! flow::maybe_run_sudo yum install -y nvidia-container-toolkit; then
        log::error "Failed to install nvidia-container-toolkit"
        return 1
    fi
    
    # Configure Docker
    log::info "Configuring Docker for NVIDIA runtime..."
    if ! flow::maybe_run_sudo nvidia-ctk runtime configure --runtime=docker; then
        log::error "Failed to configure Docker for NVIDIA"
        return 1
    fi
    
    # Restart Docker
    log::info "Restarting Docker service..."
    if ! flow::maybe_run_sudo systemctl restart docker; then
        log::error "Failed to restart Docker"
        return 1
    fi
    
    # Wait for Docker to be ready
    sleep 5
    
    log::success "NVIDIA Container Runtime installed successfully"
    return 0
}

#######################################
# Install NVIDIA runtime on Arch Linux
#######################################
comfyui::install_nvidia_runtime_pacman() {
    log::info "Installing NVIDIA Container Runtime for Arch Linux..."
    
    # Install from AUR
    log::info "Installing nvidia-container-toolkit from AUR..."
    
    if system::is_command "yay"; then
        yay -S --noconfirm nvidia-container-toolkit
    elif system::is_command "paru"; then
        paru -S --noconfirm nvidia-container-toolkit
    else
        log::error "No AUR helper found. Please install yay or paru first."
        return 1
    fi
    
    # Configure Docker
    log::info "Configuring Docker for NVIDIA runtime..."
    if ! flow::maybe_run_sudo nvidia-ctk runtime configure --runtime=docker; then
        log::error "Failed to configure Docker for NVIDIA"
        return 1
    fi
    
    # Restart Docker
    log::info "Restarting Docker service..."
    if ! flow::maybe_run_sudo systemctl restart docker; then
        log::error "Failed to restart Docker"
        return 1
    fi
    
    log::success "NVIDIA Container Runtime installed successfully"
    return 0
}

#######################################
# Show manual NVIDIA installation guide
#######################################
comfyui::show_manual_installation_guide() {
    local distro
    distro=$(comfyui::detect_os_distribution)
    
    log::header "📖 Manual NVIDIA Container Runtime Installation"
    
    echo "Your system configuration requires manual installation of NVIDIA Container Runtime."
    echo
    
    case "$distro" in
        apt)
            echo "For Ubuntu/Debian systems:"
            echo
            echo "1. Add the NVIDIA GPG key:"
            echo "   curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg"
            echo
            echo "2. Add the repository:"
            echo "   curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \\"
            echo "     sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \\"
            echo "     sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list"
            echo
            echo "3. Update and install:"
            echo "   sudo apt-get update"
            echo "   sudo apt-get install -y nvidia-container-toolkit"
            ;;
        yum)
            echo "For CentOS/RHEL/Fedora systems:"
            echo
            echo "1. Add the repository:"
            echo "   curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \\"
            echo "     sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo"
            echo
            echo "2. Install the toolkit:"
            echo "   sudo yum clean expire-cache"
            echo "   sudo yum install -y nvidia-container-toolkit"
            ;;
        *)
            echo "For other distributions, visit:"
            echo "https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html"
            ;;
    esac
    
    echo
    echo "4. Configure Docker:"
    echo "   sudo nvidia-ctk runtime configure --runtime=docker"
    echo
    echo "5. Restart Docker:"
    echo "   sudo systemctl restart docker"
    echo
    echo "6. Verify installation:"
    echo "   docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi"
    echo
    log::info "After manual installation, run this script again to continue ComfyUI setup."
}

#######################################
# Handle NVIDIA requirements interactively
#######################################
comfyui::handle_nvidia_requirements() {
    log::header "🎮 NVIDIA GPU Setup Required"
    
    log::info "ComfyUI requires NVIDIA Container Runtime to use your GPU."
    log::info "Without it, ComfyUI will run in CPU mode (very slow)."
    echo
    
    # Check if we're in non-interactive mode
    if [[ "$YES" == "yes" ]]; then
        # Check for COMFYUI_NVIDIA_CHOICE environment variable
        if [[ -n "$COMFYUI_NVIDIA_CHOICE" ]]; then
            case "$COMFYUI_NVIDIA_CHOICE" in
                1) comfyui::install_nvidia_runtime && return 0 || return 1 ;;
                2) comfyui::show_manual_installation_guide; return 1 ;;
                3) log::warn "Continuing with CPU mode"; GPU_TYPE="cpu"; return 0 ;;
                4) log::info "Installation cancelled"; return 1 ;;
                *) log::error "Invalid COMFYUI_NVIDIA_CHOICE: $COMFYUI_NVIDIA_CHOICE"; return 1 ;;
            esac
        else
            # Default to CPU mode in non-interactive
            log::warn "Non-interactive mode: Falling back to CPU mode"
            log::info "Set COMFYUI_NVIDIA_CHOICE=1 to auto-install NVIDIA runtime"
            GPU_TYPE="cpu"
            return 0
        fi
    fi
    
    # Interactive mode
    if ! comfyui::prompt_runtime_installation; then
        return 1
    fi
    
    return 0
}

#######################################
# Prompt user for NVIDIA runtime installation
#######################################
comfyui::prompt_runtime_installation() {
    log::info "Options:"
    echo "  1) Auto-install NVIDIA Container Runtime (recommended)"
    echo "  2) Show manual installation instructions"
    echo "  3) Continue with CPU mode instead (slow)"
    echo "  4) Cancel installation"
    echo
    
    local choice
    read -rp "Enter your choice (1-4): " choice
    
    case "$choice" in
        1)
            log::info "Attempting automatic installation..."
            if comfyui::install_nvidia_runtime; then
                # Verify installation
                if comfyui::test_nvidia_runtime; then
                    log::success "NVIDIA runtime installed and working!"
                    return 0
                else
                    log::error "Installation completed but GPU test failed"
                    comfyui::show_troubleshooting_guide
                    return 1
                fi
            else
                log::error "Automatic installation failed"
                echo
                if comfyui::prompt_manual_or_cpu; then
                    return 0
                else
                    return 1
                fi
            fi
            ;;
        2)
            comfyui::show_manual_installation_guide
            return 1
            ;;
        3)
            log::warn "Continuing with CPU mode"
            log::info "Note: Image generation will be significantly slower"
            GPU_TYPE="cpu"
            return 0
            ;;
        4)
            log::info "Installation cancelled"
            return 1
            ;;
        *)
            log::error "Invalid choice"
            return comfyui::prompt_runtime_installation
            ;;
    esac
}

#######################################
# Prompt for manual installation or CPU fallback
#######################################
comfyui::prompt_cpu_fallback() {
    echo
    log::info "Would you like to:"
    echo "  1) See manual installation instructions"
    echo "  2) Continue with CPU mode (slow)"
    echo "  3) Cancel installation"
    echo
    
    local choice
    read -rp "Enter your choice (1-3): " choice
    
    case "$choice" in
        1)
            comfyui::show_manual_installation_guide
            return 1
            ;;
        2)
            GPU_TYPE="cpu"
            return 0
            ;;
        3)
            return 1
            ;;
        *)
            log::error "Invalid choice"
            return comfyui::prompt_cpu_fallback
            ;;
    esac
}

#######################################
# Show NVIDIA driver installation guide
#######################################
comfyui::show_driver_installation_guide() {
    log::header "📖 NVIDIA Driver Installation Guide"
    
    echo "NVIDIA drivers are not installed or not functional on your system."
    echo
    echo "To install NVIDIA drivers:"
    echo
    
    local distro
    distro=$(comfyui::detect_os_distribution)
    
    case "$distro" in
        apt)
            echo "For Ubuntu/Debian:"
            echo "1. Update your system:"
            echo "   sudo apt update && sudo apt upgrade"
            echo
            echo "2. Install drivers:"
            echo "   sudo apt install nvidia-driver-530"  # Or latest version
            echo
            echo "3. Reboot your system"
            ;;
        yum)
            echo "For CentOS/RHEL/Fedora:"
            echo "1. Enable RPM Fusion:"
            echo "   sudo dnf install https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-\$(rpm -E %fedora).noarch.rpm"
            echo
            echo "2. Install drivers:"
            echo "   sudo dnf install akmod-nvidia"
            echo
            echo "3. Reboot your system"
            ;;
        *)
            echo "Visit NVIDIA's official driver download page:"
            echo "https://www.nvidia.com/Download/index.aspx"
            ;;
    esac
    
    echo
    echo "After installing drivers:"
    echo "1. Reboot your system"
    echo "2. Verify with: nvidia-smi"
    echo "3. Run this script again"
}

#######################################
# Show troubleshooting guide for GPU issues
#######################################
comfyui::show_troubleshooting_guide() {
    log::header "🔧 GPU Troubleshooting Guide"
    
    echo "If GPU access is not working, try these steps:"
    echo
    echo "1. Verify NVIDIA drivers:"
    echo "   nvidia-smi"
    echo
    echo "2. Check Docker daemon status:"
    echo "   sudo systemctl status docker"
    echo
    echo "3. Test GPU in Docker:"
    echo "   docker run --rm --gpus all nvidia/cuda:11.8-base-ubuntu20.04 nvidia-smi"
    echo
    echo "4. Check Docker runtime configuration:"
    echo "   docker info | grep -i nvidia"
    echo
    echo "5. If using WSL2, ensure GPU support is enabled:"
    echo "   - Update to latest Windows 11 or Windows 10 (Build 21H2 or later)"
    echo "   - Install WSL2 GPU drivers from NVIDIA"
    echo
    echo "6. Common fixes:"
    echo "   - Restart Docker: sudo systemctl restart docker"
    echo "   - Re-run configuration: sudo nvidia-ctk runtime configure --runtime=docker"
    echo "   - Check daemon.json: cat /etc/docker/daemon.json"
    echo
    echo "For more help, see:"
    echo "https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/troubleshooting.html"
}

#######################################
# Handle NVIDIA GPU setup failure
#######################################
comfyui::handle_nvidia_failure() {
    log::header "⚠️ NVIDIA GPU Setup Issue"
    
    local detected_gpu
    detected_gpu=$(comfyui::detect_gpu_silent)
    
    if [[ "$detected_gpu" == "nvidia" ]]; then
        # NVIDIA GPU present but setup failed
        if ! nvidia-smi >/dev/null 2>&1; then
            log::error "NVIDIA GPU detected but drivers not functional"
            comfyui::show_driver_installation_guide
        elif ! comfyui::is_nvidia_runtime_installed; then
            log::error "NVIDIA drivers working but Container Runtime not installed"
            
            if [[ "$YES" != "yes" ]] || [[ -n "$COMFYUI_NVIDIA_CHOICE" ]]; then
                if comfyui::handle_nvidia_requirements; then
                    return 0
                fi
            fi
        else
            log::error "NVIDIA setup appears complete but GPU access failing"
            comfyui::show_troubleshooting_guide
        fi
    fi
    
    # Offer CPU fallback
    if [[ "$YES" == "yes" ]]; then
        log::warn "Falling back to CPU mode"
        GPU_TYPE="cpu"
        return 0
    else
        if comfyui::prompt_cpu_fallback; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Get GPU information for display
#######################################
comfyui::get_gpu_info() {
    log::header "🎮 GPU Information"
    
    local gpu_type
    gpu_type=$(comfyui::detect_gpu_silent)
    
    echo "Detected GPU Type: $gpu_type"
    echo
    
    case "$gpu_type" in
        nvidia)
            if nvidia-smi >/dev/null 2>&1; then
                echo "NVIDIA GPU Details:"
                nvidia-smi --query-gpu=name,driver_version,memory.total,compute_cap --format=csv
                echo
                echo "Current GPU Usage:"
                nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total --format=csv
            else
                echo "NVIDIA GPU detected but nvidia-smi not available"
                echo "Please install NVIDIA drivers"
            fi
            
            echo
            echo "Docker NVIDIA Runtime Status:"
            if comfyui::is_docker_nvidia_configured; then
                echo "✅ Docker is configured for NVIDIA GPUs"
            else
                echo "❌ Docker is not configured for NVIDIA GPUs"
                echo "Run: $0 --action install"
            fi
            ;;
        amd)
            echo "AMD GPU Details:"
            if system::is_command "rocm-smi"; then
                rocm-smi --showid --showproductname --showmeminfo vram
            else
                echo "ROCm not installed - GPU details unavailable"
            fi
            
            echo
            log::warn "Note: AMD GPU support in ComfyUI may require additional configuration"
            ;;
        cpu)
            echo "No supported GPU detected"
            echo
            echo "CPU Information:"
            if [[ -f /proc/cpuinfo ]]; then
                echo "Model: $(grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)"
                echo "Cores: $(grep -c "processor" /proc/cpuinfo)"
            fi
            echo
            echo "Available Memory:"
            free -h | grep "^Mem:"
            echo
            log::warn "ComfyUI will run in CPU mode - expect slow performance"
            ;;
    esac
    
    echo
    echo "For more GPU options, set COMFYUI_GPU_TYPE environment variable:"
    echo "  export COMFYUI_GPU_TYPE=nvidia  # Force NVIDIA"
    echo "  export COMFYUI_GPU_TYPE=amd     # Force AMD"
    echo "  export COMFYUI_GPU_TYPE=cpu     # Force CPU"
}