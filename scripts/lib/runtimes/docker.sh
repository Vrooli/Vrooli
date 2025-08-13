#!/usr/bin/env bash
################################################################################
# Universal Docker Management Functions
# 
# Generic Docker utilities that work for any application.
# No app-specific logic should be in this file.
################################################################################

set -euo pipefail

# Get script directory
RUNTIME_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${RUNTIME_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

################################################################################
# Core Docker Command Wrappers
################################################################################

# Generic command wrapper that handles permission issues automatically
docker::_execute_with_permissions() {
    local cmd="$1"
    shift
    if [[ -n "${DOCKER_CMD:-}" ]]; then
        if [[ "$DOCKER_CMD" == "sg docker -c" ]]; then
            sg docker -c "$cmd $*"
        else
            # shellcheck disable=SC2086
            $DOCKER_CMD "$cmd" "$@"
        fi
    else
        "$cmd" "$@"
    fi
}

# Docker command wrapper
docker::run() {
    docker::_execute_with_permissions "docker" "$@"
}

# Helper to determine which compose command to use
docker::_get_compose_command() {
    # Check if we've already determined the command
    if [[ -n "${DOCKER_COMPOSE_CMD:-}" ]]; then
        echo "$DOCKER_COMPOSE_CMD"
        return 0
    fi
    
    # Try plugin version first (preferred)
    if docker::run compose version >/dev/null 2>&1; then
        export DOCKER_COMPOSE_CMD="docker compose"
        echo "docker compose"
        return 0
    fi
    
    # Fall back to standalone
    if system::is_command "docker-compose"; then
        export DOCKER_COMPOSE_CMD="docker-compose"
        echo "docker-compose"
        return 0
    fi
    
    # Neither found
    return 1
}

# Docker-compose command wrapper with automatic version detection
docker::compose() {
    local compose_cmd
    if ! compose_cmd=$(docker::_get_compose_command); then
        log::error "No Docker Compose version found"
        return 1
    fi
    
    # Handle plugin vs standalone invocation
    if [[ "$compose_cmd" == "docker compose" ]]; then
        docker::_execute_with_permissions "docker" "compose" "$@"
    else
        docker::_execute_with_permissions "docker-compose" "$@"
    fi
}

# Clear cached compose version (useful for testing)
docker::_reset_compose_detection() {
    unset DOCKER_COMPOSE_CMD
}

################################################################################
# Docker Installation & Setup
################################################################################

# Generic installation helper
docker::_install_if_missing() {
    local cmd="$1" install_func="$2" name="$3"
    system::is_command "$cmd" && { log::info "$name: $($cmd --version 2>&1)"; return 0; }
    flow::can_run_sudo "$name installation" || { log::error "$name needs sudo to install"; return 1; }
    log::info "Installing $name..."
    $install_func && system::is_command "$cmd" && log::success "$name installed" || { log::error "$name install failed"; return 1; }
}

# Docker installation function
docker::_do_install_docker() {
    curl -fsSL https://get.docker.com -o get-docker.sh
    trap 'trash::safe_remove get-docker.sh --no-confirm' EXIT
    sudo sh get-docker.sh
}

docker::install() {
    docker::_install_if_missing "docker" docker::_do_install_docker "Docker"
}

# Check if Docker is running
docker::_is_running() {
    system::is_command "docker" && docker::run version >/dev/null 2>&1
}

################################################################################
# Docker Permission Management
################################################################################

# Try alternative access methods when docker group is not active in session
docker::_try_alternative_access() {
    log::info "Docker group detected but not active in session. Attempting to run with group privileges..."
    
    # Try running docker with sg (switch group) which doesn't require a new shell
    if command -v sg >/dev/null 2>&1; then
        # Note: We can't use docker::run here because we're testing sg itself
        if sg docker -c "docker version" >/dev/null 2>&1; then
            log::success "Can access Docker using sg command"
            export DOCKER_CMD="sg docker -c"
            return 0
        fi
    fi
    
    # If sg didn't work, try using sudo to run docker commands
    if flow::can_run_sudo "Docker access"; then
        # Note: We can't use docker::run here because we're testing sudo itself
        if sudo docker version >/dev/null 2>&1; then
            log::info "Using sudo for Docker access"
            export DOCKER_CMD="sudo"
            return 0
        fi
    fi
    
    return 1
}

# Diagnose Docker permission issues
docker::_diagnose_permission_issue() {
    local current_user
    current_user=$(sudo::get_actual_user)
    
    # Check if user is in docker group (system-wide)
    if groups "$current_user" 2>/dev/null | grep -q docker; then
        # User IS in docker group but session doesn't have it active
        if ! groups | grep -q docker; then
            log::error "Docker permission issue: You're in the docker group but it's not active in this session"
            log::warning "Quick fix: Run 'newgrp docker' to activate it now"
            log::info "Or open a new terminal for permanent activation"
        else
            log::error "Docker permission denied despite being in docker group"
            log::info "Try restarting Docker service: sudo systemctl restart docker"
            log::info "Or check socket ownership: ls -l /var/run/docker.sock"
        fi
    else
        # User is NOT in docker group at all
        log::error "Docker permission denied: You're not in the docker group"
        log::error "Fix: sudo usermod -aG docker $current_user"
        log::error "Then: newgrp docker (or open new terminal)"
    fi
}

docker::manage_docker_group() {
    local actual_user
    actual_user=$(sudo::get_actual_user)
    
    # Check if user is already in docker group
    if groups "$actual_user" 2>/dev/null | grep -q docker; then
        log::info "User '$actual_user' is already in docker group"
        ! groups | grep -q docker && log::warning "Run 'newgrp docker' to activate in current session"
        return 0
    fi
    
    log::warning "User '$actual_user' is not in the docker group"
    
    # Check sudo availability
    if ! flow::can_run_sudo "Add user to docker group"; then
        log::warning "Manual fix: sudo usermod -aG docker $actual_user && newgrp docker"
        return 0
    fi
    
    # Confirm with user
    if ! flow::is_yes "$YES" && ! flow::confirm "Add '$actual_user' to docker group?"; then
        log::warning "Skipping docker group addition"
        return 0
    fi
    
    # Add user to docker group
    if sudo usermod -aG docker "$actual_user"; then
        log::success "Added '$actual_user' to docker group. Run 'newgrp docker' or restart terminal"
    else
        log::error "Failed to add user to docker group"
        return 1
    fi
}

################################################################################
# Docker Service Management
################################################################################

docker::start() {
    # First check if Docker is already running with current permissions
    if docker::_is_running; then
        log::info "Docker is already running"
        return 0
    fi
    
    # Check if we have docker group but it's not active
    local current_user
    current_user=$(sudo::get_actual_user)
    if groups "$current_user" 2>/dev/null | grep -q docker && ! groups | grep -q docker; then
        if docker::_try_alternative_access; then
            return 0
        fi
    fi

    # Docker is not running, check if we can start it
    if ! flow::can_run_sudo "Docker service start"; then
        log::warning "Docker is not running and sudo is not available to start it"
        log::warning "Please start Docker manually or ensure Docker Desktop is running"
        return 1
    fi

    # Try to start Docker
    log::info "Starting Docker service..."
    sudo service docker start

    # Wait a moment for Docker to fully start
    sleep 2
    
    # Verify Docker is now running
    if ! docker::run version >/dev/null 2>&1; then
        # Check if it's a permission issue
        if [[ -S /var/run/docker.sock ]]; then
            docker::_diagnose_permission_issue
        else
            log::error "Failed to start Docker or Docker is not running. If you are in Windows Subsystem for Linux (WSL), please start Docker Desktop and try again."
        fi
        return 1
    fi
    
    log::success "Docker service started successfully"
}

docker::restart() {
    if ! flow::can_run_sudo "Docker service restart"; then
        log::warning "Skipping Docker restart due to sudo mode"
        return
    fi

    log::info "Restarting Docker..."
    sudo service docker restart
}

docker::kill_all() {
    system::is_command "docker" || return
    local containers
    containers=$(docker::run ps -q)
    # shellcheck disable=SC2086
    [[ -n "$containers" ]] && docker::run kill $containers || log::warning "No running containers"
}

################################################################################
# Docker Compose Setup
################################################################################

# Docker Compose installation function
docker::_do_install_docker_compose() {
    sudo curl -L "https://github.com/docker/compose/releases/download/v2.15.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod a+rx /usr/local/bin/docker-compose
}

docker::setup_docker_compose() {
    # Check what version is available using our detection helper
    local compose_cmd
    if compose_cmd=$(docker::_get_compose_command); then
        if [[ "$compose_cmd" == "docker compose" ]]; then
            log::info "Detected Docker Compose plugin: $(docker::run compose version 2>&1 | head -1)"
        else
            log::info "Detected Docker Compose standalone: $(docker-compose --version)"
        fi
        return 0
    fi
    
    # Neither version found, try to install standalone version
    log::info "No Docker Compose found, installing standalone version..."
    docker::_install_if_missing "docker-compose" docker::_do_install_docker_compose "Docker Compose"
    
    # Reset detection cache after installation
    docker::_reset_compose_detection
}

################################################################################
# Docker Network & Internet Access
################################################################################

docker::check_internet_access() {
    docker::run run --rm busybox ping -c 1 google.com &>/dev/null
}

docker::show_daemon() {
    [[ -f /etc/docker/daemon.json ]] && cat /etc/docker/daemon.json || log::warning "No daemon.json"
}

docker::update_daemon() {
    if ! flow::can_run_sudo "Docker daemon configuration update"; then
        log::warning "Skipping Docker daemon update due to sudo mode"
        return
    fi

    log::info "Updating Docker daemon DNS configuration..."
    
    local daemon_file="/etc/docker/daemon.json"
    local dns_servers='["8.8.8.8", "8.8.4.4"]'
    
    # Create backup if file exists
    [[ -f "$daemon_file" ]] && sudo cp "$daemon_file" "${daemon_file}.backup"
    
    if [[ -f "$daemon_file" ]]; then
        # Merge DNS into existing config using jq
        if ! sudo jq --argjson dns "$dns_servers" '. + {dns: $dns}' "$daemon_file" | sudo tee "${daemon_file}.tmp" >/dev/null 2>&1; then
            log::error "Failed to update daemon.json (invalid JSON?)"
            return 1
        fi
    else
        # Create new file with DNS only
        echo "{\"dns\": $dns_servers}" | sudo tee "${daemon_file}.tmp" >/dev/null
    fi
    
    # Validate JSON before replacing
    if jq empty "${daemon_file}.tmp" 2>/dev/null; then
        sudo mv "${daemon_file}.tmp" "$daemon_file"
        log::success "Docker daemon DNS updated (preserved existing config)"
    else
        log::error "Invalid JSON generated, keeping original"
        trash::safe_remove "${daemon_file}.tmp" --no-confirm
        return 1
    fi
}

docker::setup_internet_access() {
    if docker::check_internet_access; then
        log::success "Docker internet access: OK"
        return 0
    fi

    log::error "Docker cannot access the internet. This may be a DNS issue."
    docker::show_daemon

    if flow::is_yes "${YES:-}"; then
        docker::update_daemon
        docker::restart
        log::info "Docker DNS updated. Retesting Docker internet access..."
        docker::check_internet_access && log::success "Docker internet access is now working!" || log::error "Docker internet access still failing."
    else
        log::prompt "Would you like to update /etc/docker/daemon.json to use Google DNS (8.8.8.8)? (y/n): " choice
        read -n1 -r choice
        echo
        if flow::is_yes "$choice"; then
            docker::update_daemon
            docker::restart
            log::info "Docker DNS updated. Retesting Docker internet access..."
            docker::check_internet_access && log::success "Docker internet access is now working!" || log::error "Docker internet access still failing."
        else
            log::info "No changes made."
        fi
    fi
}

################################################################################
# Docker Resource Management
################################################################################

docker::calculate_resource_limits() {
    # Check if bc is available
    if ! system::is_command "bc"; then
        log::warning "bc not found, using fallback resource calculations"
        # Fallback: use awk for calculations
        N=$(nproc)
        CPU_QUOTA=$(awk -v n="$N" 'BEGIN {printf "%d%%", (n - 0.5) * 100}')
        MEM_LIMIT=$(free -m | awk '/^Mem:/{printf "%dM", int($2 * 0.8)}')
    else
        # Get total number of CPU cores and calculate CPU quota.
        N=$(nproc)
        # Calculate quota: (N - 0.5) * 100. This value is later appended with '%' .
        QUOTA=$(echo "($N - 0.5) * 100" | bc)
        CPU_QUOTA="${QUOTA}%" 

        # Get total memory (in MB) and calculate 80% of it.
        TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
        MEM_LIMIT=$(echo "$TOTAL_MEM * 0.8" | bc | cut -d. -f1)M
    fi
}

docker::define_config_files() {
    # This slice unit will hold the Docker daemon's resource limits.
    SLICE_FILE="/etc/systemd/system/docker.slice"

    # The service override drop-in file ensures docker.service is placed in the Docker slice.
    DOCKER_SERVICE_DIR="/etc/systemd/system/docker.service.d"
    DOCKER_OVERRIDE_FILE="${DOCKER_SERVICE_DIR}/override.conf"
}

docker::create_docker_slice() {
    docker::calculate_resource_limits
    docker::define_config_files

    log::info "Creating Docker slice with resource limits (CPU: ${CPU_QUOTA}, Memory: ${MEM_LIMIT})..."
    
    # Create the slice file
    sudo bash -c "cat > ${SLICE_FILE}" << EOF
[Unit]
Description=Slice for Docker services
Before=slices.target

[Slice]
CPUQuota=${CPU_QUOTA}
MemoryMax=${MEM_LIMIT}
EOF

    log::success "Docker slice created with resource limits"
}

docker::create_docker_override() {
    docker::define_config_files

    log::info "Creating Docker service override to use resource slice..."
    
    # Create the service drop-in directory if it doesn't exist
    sudo mkdir -p "${DOCKER_SERVICE_DIR}"
    
    # Create the override configuration
    sudo bash -c "cat > ${DOCKER_OVERRIDE_FILE}" << EOF
[Service]
Slice=docker.slice
EOF

    log::success "Docker service override created"
}

docker::apply_systemd_changes() {
    log::info "Applying systemd configuration changes..."
    sudo systemctl daemon-reload
    
    # Only restart Docker if it's already running
    if systemctl is-active --quiet docker; then
        log::info "Restarting Docker to apply resource limits..."
        sudo systemctl restart docker
        log::success "Docker restarted with resource limits applied"
    else
        log::info "Docker is not running. Resource limits will be applied when Docker starts."
    fi
}

docker::configure_resource_limits() {
    # Check if we're in a CI environment or sudo is not available
    if flow::is_yes "${IS_CI:-}" || ! flow::can_run_sudo "Docker resource limits configuration"; then
        log::info "Skipping Docker resource limits configuration (CI or no sudo)"
        return 0
    fi

    # Check if systemd is available
    if ! system::is_command "systemctl"; then
        log::info "Systemd not available, skipping Docker resource limits configuration"
        return 0
    fi

    log::header "Configuring Docker Resource Limits"
    
    docker::create_docker_slice
    docker::create_docker_override
    docker::apply_systemd_changes
    
    log::success "Docker resource limits configured successfully"
}

################################################################################
# Docker GPU Support
################################################################################

docker::configure_gpu_runtime() {
    # Check if we're in a CI environment
    if flow::is_yes "${IS_CI:-}"; then
        log::info "Skipping GPU runtime configuration (CI environment)"
        return 0
    fi

    # Check if nvidia-smi is available
    if ! system::is_command "nvidia-smi"; then
        log::info "No NVIDIA GPU detected, skipping GPU runtime configuration"
        return 0
    fi

    # Early check if GPU support is already working (before starting configuration)
    if docker::run run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi >/dev/null 2>&1; then
        log::success "Docker GPU runtime is already working"
        return 0
    fi

    log::header "Configuring Docker GPU Runtime"
    
    # Check if nvidia-container-runtime is available (alternative to nvidia-docker2)
    if system::is_command "nvidia-container-runtime"; then
        log::info "nvidia-container-runtime detected, testing GPU access..."
        # Try with the runtime flag instead of --gpus
        if docker::run run --rm --runtime=nvidia nvidia/cuda:11.0-base nvidia-smi >/dev/null 2>&1; then
            log::success "Docker GPU runtime working with nvidia-container-runtime"
            return 0
        fi
    fi
    
    # Check if nvidia-docker2 is installed
    if ! dpkg -l 2>/dev/null | grep -q nvidia-docker2; then
        log::info "nvidia-docker2 not installed"
        
        if ! flow::can_run_sudo "nvidia-docker2 installation"; then
            log::warning "Cannot install nvidia-docker2 without sudo"
            log::info "GPU support may be limited. To enable full GPU support, run:"
            log::info "  sudo apt-get install nvidia-docker2"
            log::info "  sudo systemctl restart docker"
            # Don't fail setup - GPU support is optional
            return 0
        fi
        
        # Add NVIDIA Docker repository
        distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
        curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
        curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
        
        # Install nvidia-docker2
        sudo apt-get update
        sudo apt-get install -y nvidia-docker2
        
        # Restart Docker to apply changes
        sudo systemctl restart docker
        
        log::success "nvidia-docker2 installed successfully"
    else
        log::info "nvidia-docker2 is already installed"
    fi
    
    # Test GPU access
    if docker::run run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi >/dev/null 2>&1; then
        log::success "Docker GPU runtime configured successfully"
    else
        log::warning "GPU runtime configured but test failed. You may need to restart Docker."
    fi
}

################################################################################
# Docker Diagnostics
################################################################################

docker::diagnose() {
    log::header "Docker Diagnostics"
    system::is_command "docker" || { log::error "Docker not installed"; return 1; }
    log::info "Version: $(docker::run --version 2>&1)"
    
    if docker::run version >/dev/null 2>&1; then
        log::success "Daemon: RUNNING"
    else
        log::error "Daemon: NOT ACCESSIBLE"
        command -v systemctl >/dev/null 2>&1 && systemctl status docker --no-pager 2>&1 | head -10
    fi
    
    [[ -S /var/run/docker.sock ]] && log::info "Socket: $(ls -l /var/run/docker.sock)" || log::warning "Socket: NOT FOUND"
    groups | grep -q docker && log::success "In docker group" || log::warning "NOT in docker group"
}

################################################################################
# Main Docker Setup Function
################################################################################

docker::setup() {
    docker::install
    docker::manage_docker_group
    docker::start
    docker::setup_docker_compose
    if ! flow::is_yes "${IS_CI:-}"; then
        docker::setup_internet_access
        docker::configure_resource_limits
        docker::configure_gpu_runtime
    fi
}

# Clean interface for setup.sh
docker::ensure_installed() {
    docker::setup
}

# Export functions for use by other scripts
export -f docker::run
export -f docker::compose
export -f docker::setup
export -f docker::diagnose
export -f docker::start
export -f docker::restart
export -f docker::kill_all