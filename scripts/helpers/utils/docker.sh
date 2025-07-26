#!/usr/bin/env bash
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
for util in env exit_codes flow log system var; do source "${UTILS_DIR}/${util}.sh"; done

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
    trap 'rm -f get-docker.sh' EXIT
    sudo sh get-docker.sh
}

docker::install() {
    docker::_install_if_missing "docker" docker::_do_install_docker "Docker"
}

# Check if Docker is running
docker::_is_running() {
    system::is_command "docker" && docker::run version >/dev/null 2>&1
}

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
    local current_user="${SUDO_USER:-$USER}"
    
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

docker::start() {
    # First check if Docker is already running with current permissions
    if docker::_is_running; then
        log::info "Docker is already running"
        return 0
    fi
    
    # Check if we have docker group but it's not active
    local current_user="${SUDO_USER:-$USER}"
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
    local containers=$(docker::run ps -q)
    # shellcheck disable=SC2086
    [[ -n "$containers" ]] && docker::run kill $containers || log::warning "No running containers"
}

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

docker::manage_docker_group() {
    local actual_user="${SUDO_USER:-$USER}"
    
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
        sudo rm -f "${daemon_file}.tmp"
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
    SERVICE_OVERRIDE_DIR="/etc/systemd/system/docker.service.d"
    SERVICE_OVERRIDE_FILE="${SERVICE_OVERRIDE_DIR}/slice.conf"
}

# Check if systemd config would need updating (without writing)
docker::_check_systemd_config_needs_update() {
    local file="$1" section="$2"; shift 2
    local directives=("$@")
    
    # If file doesn't exist, we need to create it
    if [[ ! -f "$file" ]]; then
        return 0  # needs update
    fi
    
    # Check if section exists
    if ! grep -q "^\[$section\]" "$file"; then
        return 0  # needs update
    fi
    
    # Check each directive
    for directive in "${directives[@]}"; do
        local key="${directive%%=*}" value="${directive#*=}"
        if grep -q "^$key=" "$file"; then
            local old_value=$(grep "^$key=" "$file" | head -n 1 | cut -d= -f2-)
            [[ "$old_value" != "$value" ]] && return 0  # needs update
        else
            return 0  # needs update (directive missing)
        fi
    done
    
    return 1  # no updates needed
}

# Generic systemd config updater
docker::_update_systemd_config() {
    local file="$1" section="$2"; shift 2
    local directives=("$@")
    
    if [[ ! -f "$file" ]]; then
        { echo "[$section]"; printf '%s\n' "${directives[@]}"; } > "$file"
        changed=true
        return
    fi
    
    grep -q "^\[$section\]" "$file" || { sed -i "1i[$section]" "$file"; changed=true; }
    
    for directive in "${directives[@]}"; do
        local key="${directive%%=*}" value="${directive#*=}"
        if grep -q "^$key=" "$file"; then
            local old_value=$(grep "^$key=" "$file" | head -n 1 | cut -d= -f2-)
            [[ "$old_value" != "$value" ]] && { sed -i "s/^$key=.*/$directive/" "$file"; changed=true; }
        else
            sed -i "/^\[$section\]/a $directive" "$file"; changed=true
        fi
    done
}

docker::update_slice_file() {
    docker::_update_systemd_config "$SLICE_FILE" "Slice" "CPUQuota=${CPU_QUOTA}" "MemoryMax=${MEM_LIMIT}"
}

docker::update_service_override_file() {
    # Only create directory if it doesn't exist
    if [[ ! -d "$SERVICE_OVERRIDE_DIR" ]]; then
        mkdir -p "$SERVICE_OVERRIDE_DIR"
    fi
    docker::_update_systemd_config "$SERVICE_OVERRIDE_FILE" "Service" "Slice=docker.slice"
}

# Sets up a dedicated systemd slice for the Docker daemon
# and assigns the docker.service to this slice. It calculates resource
# limits based on system characteristics:
#  - CPUQuota: (total cores - 0.5)*100 (%)
#  - MemoryMax: 80% of the total memory (in MB)
#
# The Docker slice will be defined in /etc/systemd/system/docker.slice,
# and the docker.service override will be in
# /etc/systemd/system/docker.service.d/slice.conf.
docker::configure_resource_limits() {
    log::header "Checking Docker resource limits"

    # Calculate limits and define config files (no sudo needed)
    docker::calculate_resource_limits
    docker::define_config_files

    # Check if any changes are needed (read-only operations)
    local needs_update=false
    
    # Check slice file
    if docker::_check_systemd_config_needs_update "$SLICE_FILE" "Slice" "CPUQuota=${CPU_QUOTA}" "MemoryMax=${MEM_LIMIT}"; then
        needs_update=true
    fi
    
    # Check service override file (also check if directory needs to be created)
    if [[ ! -d "$SERVICE_OVERRIDE_DIR" ]] || docker::_check_systemd_config_needs_update "$SERVICE_OVERRIDE_FILE" "Service" "Slice=docker.slice"; then
        needs_update=true
    fi
    
    # If no updates needed, we're done
    if [[ "$needs_update" = false ]]; then
        log::info "Docker resource limits are already configured correctly. No action needed."
        return 0
    fi
    
    # Updates are needed - now check for sudo
    if ! flow::can_run_sudo "Docker resource limits configuration"; then
        log::warning "Docker resource limits need updating but sudo is not available"
        log::info "The following changes would be made:"
        log::info "  - Set CPU quota to ${CPU_QUOTA}"
        log::info "  - Set memory limit to ${MEM_LIMIT}"
        log::info "To apply these changes manually, run with sudo or use --sudo-mode error"
        return 1
    fi

    # We have sudo and changes are needed - proceed with updates
    log::info "Updating Docker resource limits"
    
    changed=false
    docker::update_slice_file
    docker::update_service_override_file

    # Reload systemd if changes were made
    if [ "$changed" = true ]; then
        log::info "Docker slice configuration updated."
        systemctl daemon-reload
        systemctl restart docker.service
        log::success "Docker resource limits set up successfully."
    else
        # This shouldn't happen since we already checked for changes
        log::info "Docker slice configuration unchanged. No action taken."
    fi
}

# Configure Docker daemon for GPU runtime support
docker::configure_gpu_runtime() {
    # Check if NVIDIA container runtime is installed
    if ! system::is_command "nvidia-container-runtime"; then
        log::info "NVIDIA Container Runtime not installed, skipping GPU configuration"
        return 0
    fi
    
    # Check if NVIDIA GPU is present
    if ! system::has_nvidia_gpu; then
        log::info "No NVIDIA GPU detected, skipping GPU configuration"
        return 0
    fi
    
    log::info "Checking Docker daemon GPU configuration..."
    
    local daemon_file="/etc/docker/daemon.json"
    local needs_update=false
    
    # Check if daemon.json exists and already has correct GPU configuration
    if [[ -f "$daemon_file" ]]; then
        # Check if default-runtime is nvidia and nvidia runtime is configured
        local has_nvidia_default=$(jq -r '."default-runtime" // ""' "$daemon_file" 2>/dev/null)
        local has_nvidia_runtime=$(jq -r '.runtimes.nvidia // ""' "$daemon_file" 2>/dev/null)
        
        if [[ "$has_nvidia_default" == "nvidia" && -n "$has_nvidia_runtime" ]]; then
            log::info "Docker daemon already configured for GPU support. No changes needed."
            return 0
        else
            needs_update=true
        fi
    else
        needs_update=true
    fi
    
    # Only proceed if update is needed
    if [[ "$needs_update" == "true" ]]; then
        log::info "Configuring Docker daemon for NVIDIA GPU support..."
        
        local gpu_config='{
            "default-runtime": "nvidia",
            "runtimes": {
                "nvidia": {
                    "path": "nvidia-container-runtime",
                    "runtimeArgs": []
                }
            }
        }'
        
        # Create backup if file exists
        [[ -f "$daemon_file" ]] && flow::maybe_run_sudo cp "$daemon_file" "${daemon_file}.backup"
        
        if [[ -f "$daemon_file" ]]; then
            # Merge GPU config into existing config using jq
            if ! echo "$gpu_config" | flow::maybe_run_sudo jq -s '.[0] * .[1]' "$daemon_file" - | flow::maybe_run_sudo tee "${daemon_file}.tmp" >/dev/null 2>&1; then
                log::error "Failed to update daemon.json for GPU support"
                return 1
            fi
        else
            # Create new file with GPU config only
            echo "$gpu_config" | flow::maybe_run_sudo tee "${daemon_file}.tmp" >/dev/null
        fi
        
        # Validate JSON before replacing
        if jq empty "${daemon_file}.tmp" 2>/dev/null; then
            flow::maybe_run_sudo mv "${daemon_file}.tmp" "$daemon_file"
            log::success "Docker daemon configured for GPU support"
            
            # Restart Docker daemon to apply changes
            log::info "Restarting Docker daemon to apply GPU configuration..."
            docker::restart
        else
            log::error "Invalid JSON generated for GPU config, keeping original"
            flow::maybe_run_sudo rm -f "${daemon_file}.tmp"
            return 1
        fi
    fi
}

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

docker::get_compose_file() {
    if env::in_production; then
        echo "${var_DOCKER_COMPOSE_PROD_FILE}"
    else
        echo "${var_DOCKER_COMPOSE_DEV_FILE}"
    fi
}

docker::build_images() {
    log::header "Building Docker images"
    local compose_file=$(docker::get_compose_file)
    cd "$var_ROOT_DIR" || { log::error "Failed to change directory to project root"; exit "$ERROR_BUILD_FAILED"; }
    
    # Check if compose is available
    if ! docker::_get_compose_command >/dev/null 2>&1; then
        log::error "No Docker Compose available to build images"
        exit "$ERROR_BUILD_FAILED"
    fi
    
    # Build with appropriate arguments based on version
    # The wrapper handles version detection, we just need to handle version-specific args
    local compose_cmd=$(docker::_get_compose_command)
    if [[ "$compose_cmd" == "docker compose" ]]; then
        # Plugin version supports --progress flag
        docker::compose -f "$compose_file" build --no-cache --progress=plain
    else
        # Standalone version doesn't support --progress
        docker::compose -f "$compose_file" build --no-cache
    fi
    
    log::success "Docker images built successfully"
}

docker::pull_base_images() {
    log::header "Pulling Docker base images"
    local base_images=("redis:7.4.0-alpine" "pgvector/pgvector:pg15" "steelcityamir/safe-content-ai:1.1.0")
    for img in "${base_images[@]}"; do docker::run pull "$img"; done
    log::success "Docker base images pulled successfully"
}

docker::collect_images() {
    local tag=$(env::in_production && echo "prod" || echo "dev")
    local images=("ui:$tag" "server:$tag" "jobs:$tag" "redis:7.4.0-alpine" "pgvector/pgvector:pg15")
    local available_images=()
    
    for img in "${images[@]}"; do
        docker::run image inspect "$img" >/dev/null 2>&1 && available_images+=("$img") || log::warning "Image $img not found"
    done
    
    [[ ${#available_images[@]} -eq 0 ]] && { log::error "No Docker images available"; exit "$ERROR_BUILD_FAILED"; }
    echo "${available_images[@]}"
}

docker::save_images() {
    local local_artifacts_dir="$1"
    log::header "Saving Docker images"
    local images=()
    # Correctly read space-separated image names into the 'images' array
    read -r -a images <<< "$(docker::collect_images)"
    local images_tar="$local_artifacts_dir/docker-images.tar"

    # Check if the images array is empty after attempting to populate it
    if [[ ${#images[@]} -eq 0 ]]; then
        log::error "No Docker images were collected to save. docker::collect_images might have returned empty or failed."
        # Optionally, exit here if this is a critical error
        # exit "$ERROR_BUILD_FAILED" # Ensure ERROR_BUILD_FAILED is defined or use a generic exit code
        return 1 # Or return an error code
    fi

    docker::run save -o "$images_tar" "${images[@]}"
    log::success "Docker images saved to $images_tar"
}

docker::build_artifacts() {
    log::header "Building Docker artifacts..."
    docker::build_images
    docker::pull_base_images
}

docker::load_images_from_tar() {
    local tar_path="$1"

    if [ ! -f "$tar_path" ]; then
        log::error "Docker images tarball not found at: ${tar_path}"
        return 1
    fi

    log::info "Loading Docker images from ${tar_path}..."
    if docker::run load -i "$tar_path"; then
        log::success "Successfully loaded Docker images from ${tar_path}."
        return 0
    else
        log::error "Failed to load Docker images from ${tar_path}."
        return 1
    fi
}

# Docker login. Required for sending images to Docker Hub.
docker::login_to_dockerhub() {
    [[ -z "${DOCKERHUB_USERNAME:-}" || -z "${DOCKERHUB_TOKEN:-}" ]] && { log::error "DOCKERHUB_USERNAME and DOCKERHUB_TOKEN required"; exit "$ERROR_DOCKER_LOGIN_FAILED"; }
    echo "${DOCKERHUB_TOKEN}" | docker::run login -u "${DOCKERHUB_USERNAME}" --password-stdin
}

# Tags and pushes Docker images to Docker Hub. Required for deploying to K8s, 
# since we can't send zipped images like we can for deployment to a single VPS.
docker::tag_and_push_images() {
    log::header "Tagging and pushing Docker images..."
  
    if [ -z "${DOCKERHUB_USERNAME:-}" ]; then
        log::error "DOCKERHUB_USERNAME must be set."
        exit "$ERROR_DOCKER_LOGIN_FAILED"
    fi
    
    local current_version="${PROJECT_VERSION:-}" # Use PROJECT_VERSION set by build.sh
    if [ -z "$current_version" ]; then
        log::warning "PROJECT_VERSION is not set by the calling script (e.g., build.sh). Using 'latest' as the version tag."
        current_version="latest"
    fi
  
    local services=("server" "ui" "jobs") # Define services with Docker images
    local image_name # local variable for image name
    local version_tag # specific version tag
    local floating_tag # dev or prod tag

    # Determine the floating tag (dev/prod) based on the global ENVIRONMENT variable
    # env::in_production sources this global variable
    if env::in_production; then # env::in_production doesn't take an argument, uses global $ENVIRONMENT
        floating_tag="prod"
    else
        floating_tag="dev"
    fi
  
    for service in "${services[@]}"; do
        # Local image name uses service:environment format
        image_name="${service}:${floating_tag}"
        
        # Tag with specific version
        version_tag="${DOCKERHUB_USERNAME}/vrooli-${service}:${current_version}"
        log::info "Tagging image ${image_name} as ${version_tag}"
        if ! docker::run tag "${image_name}" "${version_tag}"; then
            log::error "Failed to tag image ${image_name} with version ${current_version}. Does the local image exist?"
            continue 
        fi
    
        log::info "Pushing image ${version_tag} to Docker Hub..."
        if ! docker::run push "${version_tag}"; then
            log::error "Failed to push image ${version_tag}."
        else
            log::success "Successfully pushed ${version_tag}"
        fi

        # Tag with floating tag (dev/prod)
        local floating_tag_full="${DOCKERHUB_USERNAME}/vrooli-${service}:${floating_tag}"
        log::info "Tagging image ${image_name} as ${floating_tag_full}"
        if ! docker::run tag "${image_name}" "${floating_tag_full}"; then
            log::error "Failed to tag image ${image_name} with floating tag ${floating_tag}."
            # This failure is less critical than the versioned tag, so don't continue here necessarily
        else
            log::info "Pushing image ${floating_tag_full} to Docker Hub..."
            if ! docker::run push "${floating_tag_full}"; then
                log::error "Failed to push image ${floating_tag_full}."
            else
                log::success "Successfully pushed ${floating_tag_full}"
            fi
        fi
    done
  
    log::success "Finished tagging and pushing images."
}