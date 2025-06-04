#!/usr/bin/env bash
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/env.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/system.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/var.sh"

docker::install() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping Docker installation due to sudo mode"
        return
    fi

    if system::is_command "docker"; then
        log::info "Detected: $(docker --version)"
        return 0
    fi

    log::info "Docker is not installed. Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    trap 'rm -f get-docker.sh' EXIT
    sudo sh get-docker.sh
    # Check if Docker installation failed
    if ! system::is_command "docker"; then
        log::error "Docker installation failed."
        return 1
    fi
}

docker::start() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping Docker start due to sudo mode"
        return
    fi

    # Try to start Docker (if already running, this should be a no-op)
    sudo service docker start

    # Verify Docker is running by attempting a command
    if ! system::is_command "docker"; then
        log::error "Failed to start Docker or Docker is not running. If you are in Windows Subsystem for Linux (WSL), please start Docker Desktop and try again."
        return 1
    fi
}

docker::restart() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping Docker restart due to sudo mode"
        return
    fi

    log::info "Restarting Docker..."
    sudo service docker restart
}

docker::kill_all() {
    # If docker is not running
    if ! system::is_command "docker"; then
        log::warning "Docker is not running"
        return
    fi

    # If there are no running containers, do nothing
    if [ -z "$(docker ps -q)" ]; then
        log::warning "No running containers found"
        return
    fi

    # Kill all running containers
    docker kill $(docker ps -q)
}

docker::setup_docker_compose() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping Docker Compose installation due to sudo mode"
        return
    fi

    if system::is_command "docker-compose"; then
        log::info "Detected: $(docker-compose --version)"
        return 0
    fi

    log::info "Docker Compose is not installed. Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/download/v2.15.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod a+rx /usr/local/bin/docker-compose
    # Check if Docker Compose installation failed
    if ! system::is_command "docker-compose"; then
        log::error "Docker Compose installation failed."
        return 1
    fi
}

docker::check_internet_access() {
    log::header "Checking Docker internet access..."
    if docker run --rm busybox ping -c 1 google.com &>/dev/null; then
        log::success "Docker internet access: OK"
    else
        log::warning "Docker internet access: FAILED"
        return 1
    fi
}

docker::show_daemon() {
    if [ -f /etc/docker/daemon.json ]; then
        log::info "Current /etc/docker/daemon.json:"
        cat /etc/docker/daemon.json
    else
        log::warning "/etc/docker/daemon.json does not exist."
    fi
}

docker::update_daemon() {
    if ! flow::can_run_sudo; then
        log::warning "Skipping Docker daemon update due to sudo mode"
        return
    fi

    log::info "Updating /etc/docker/daemon.json to use Google DNS (8.8.8.8)..."

    # Check if /etc/docker/daemon.json exists
    if [ -f /etc/docker/daemon.json ]; then
        # Backup existing file
        sudo cp /etc/docker/daemon.json /etc/docker/daemon.json.backup
        log::info "Backup created at /etc/docker/daemon.json.backup"
    fi

    # Write new config
    sudo bash -c 'cat > /etc/docker/daemon.json' <<EOF
{
  "dns": ["8.8.8.8"]
}
EOF

    log::info "/etc/docker/daemon.json updated."
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
    # Get total number of CPU cores and calculate CPU quota.
    N=$(nproc)
    # Calculate quota: (N - 0.5) * 100. This value is later appended with '%' .
    QUOTA=$(echo "($N - 0.5) * 100" | bc)
    CPU_QUOTA="${QUOTA}%" 

    # Get total memory (in MB) and calculate 80% of it.
    TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
    MEM_LIMIT=$(echo "$TOTAL_MEM * 0.8" | bc | cut -d. -f1)M
}

docker::define_config_files() {
    # This slice unit will hold the Docker daemon's resource limits.
    SLICE_FILE="/etc/systemd/system/docker.slice"

    # The service override drop-in file ensures docker.service is placed in the Docker slice.
    SERVICE_OVERRIDE_DIR="/etc/systemd/system/docker.service.d"
    SERVICE_OVERRIDE_FILE="${SERVICE_OVERRIDE_DIR}/slice.conf"
}

docker::update_slice_file() {
    # We want the slice file to contain a [Slice] section with CPUQuota and MemoryMax.
    if [[ ! -f "$SLICE_FILE" ]]; then
        cat <<EOF > "$SLICE_FILE"
[Slice]
CPUQuota=${CPU_QUOTA}
MemoryMax=${MEM_LIMIT}
EOF
        changed=true
    else
        # Ensure the [Slice] header exists.
        if ! grep -q "^\[Slice\]" "$SLICE_FILE"; then
            sed -i "1i[Slice]" "$SLICE_FILE"
            changed=true
        fi
        # Update or add CPUQuota setting.
        if grep -q '^CPUQuota=' "$SLICE_FILE"; then
            old_cpu=$(grep '^CPUQuota=' "$SLICE_FILE" | head -n 1 | cut -d= -f2-)
            if [[ "$old_cpu" != "$CPU_QUOTA" ]]; then
                sed -i "s/^CPUQuota=.*/CPUQuota=${CPU_QUOTA}/" "$SLICE_FILE"
                changed=true
            fi
        else
            sed -i "/^\[Slice\]/a CPUQuota=${CPU_QUOTA}" "$SLICE_FILE"
            changed=true
        fi

        # Update or add MemoryMax setting.
        if grep -q '^MemoryMax=' "$SLICE_FILE"; then
            old_mem=$(grep '^MemoryMax=' "$SLICE_FILE" | head -n 1 | cut -d= -f2-)
            if [[ "$old_mem" != "$MEM_LIMIT" ]]; then
                sed -i "s/^MemoryMax=.*/MemoryMax=${MEM_LIMIT}/" "$SLICE_FILE"
                changed=true
            fi
        else
            sed -i "/^\[Slice\]/a MemoryMax=${MEM_LIMIT}" "$SLICE_FILE"
            changed=true
        fi
    fi
}

docker::update_service_override_file() {
    # Make sure the directory exists.
    mkdir -p "$SERVICE_OVERRIDE_DIR"

    # The override file should assign docker.service to the docker.slice.
    if [[ ! -f "$SERVICE_OVERRIDE_FILE" ]]; then
        cat <<EOF > "$SERVICE_OVERRIDE_FILE"
[Service]
Slice=docker.slice
EOF
        changed=true
    else
        # Ensure the [Service] header exists.
        if ! grep -q "^\[Service\]" "$SERVICE_OVERRIDE_FILE"; then
        sed -i "1i[Service]" "$SERVICE_OVERRIDE_FILE"
        changed=true
        fi

        # Check and update the Slice directive.
        if grep -q '^Slice=' "$SERVICE_OVERRIDE_FILE"; then
            old_slice=$(grep '^Slice=' "$SERVICE_OVERRIDE_FILE" | head -n 1 | cut -d= -f2-)
            if [[ "$old_slice" != "docker.slice" ]]; then
                sed -i "s/^Slice=.*/Slice=docker.slice/" "$SERVICE_OVERRIDE_FILE"
                changed=true
            fi
        else
            sed -i "/^\[Service\]/a Slice=docker.slice" "$SERVICE_OVERRIDE_FILE"
            changed=true
        fi
    fi
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
    if ! flow::can_run_sudo; then
        log::warning "Skipping Docker resource limits setup due to sudo mode"
        return
    fi

    log::header "Setting up Docker resource limits"

    changed=false
    docker::calculate_resource_limits
    docker::define_config_files
    docker::update_slice_file
    docker::update_service_override_file

    # Reload systemd if changes were made
    if [ "$changed" = true ]; then
        log::info "Docker slice configuration updated."
        systemctl daemon-reload
        systemctl restart docker.service
        log::success "Docker resource limits set up successfully."
    else
        log::info "Docker slice configuration unchanged. No action taken."
    fi
}

docker::setup() {
    docker::install
    docker::start
    docker::setup_docker_compose
    if ! flow::is_yes "${IS_CI:-}"; then
        docker::setup_internet_access
        docker::configure_resource_limits
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
    if system::is_command "docker compose"; then
        docker compose -f "$compose_file" build --no-cache --progress=plain
    elif system::is_command "docker-compose"; then
        docker-compose -f "$compose_file" build --no-cache
    else
        log::error "No Docker Compose available to build images"
        exit "$ERROR_BUILD_FAILED"
    fi
    log::success "Docker images built successfully"
}

docker::pull_base_images() {
    log::header "Pulling Docker base images"
    local base_images=()
    if env::in_production; then
        base_images=(
            "redis:7.4.0-alpine"
            "pgvector/pgvector:pg15"
            "steelcityamir/safe-content-ai:1.1.0"
            # add production-only base images here
        )
    else
        base_images=(
            "redis:7.4.0-alpine"
            "pgvector/pgvector:pg15"
            "steelcityamir/safe-content-ai:1.1.0"
            # add development-only base images here
        )
    fi
    for img in "${base_images[@]}"; do
        docker pull "$img"
    done
    log::success "Docker base images pulled successfully"
}

docker::collect_images() {
    local images=()
     if env::in_production; then
        images=(
            "ui:prod"
            "server:prod"
            "jobs:prod"
            "redis:7.4.0-alpine"
            "pgvector/pgvector:pg15"
        )
    else
        images=(
            "ui:dev"
            "server:dev"
            "jobs:dev"
            "redis:7.4.0-alpine"
            "pgvector/pgvector:pg15"
        )
    fi

    local available_images=()
    for img in "${images[@]}"; do
        if docker image inspect "$img" >/dev/null 2>&1; then
            available_images+=("$img")
        else
            log::warning "Image $img not found, skipping"
        fi
    done
    if [[ ${#available_images[@]} -eq 0 ]]; then
        log::error "No Docker images available to save"
        exit "$ERROR_BUILD_FAILED"
    fi

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

    docker save -o "$images_tar" "${images[@]}"
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
    if docker load -i "$tar_path"; then
        log::success "Successfully loaded Docker images from ${tar_path}."
        return 0
    else
        log::error "Failed to load Docker images from ${tar_path}."
        return 1
    fi
}

# Docker login. Required for sending images to Docker Hub.
docker::login_to_dockerhub() {
    log::header "Logging into Docker Hub..."
    if [ -z "${DOCKERHUB_USERNAME:-}" ] || [ -z "${DOCKERHUB_TOKEN:-}" ]; then
        log::error "DOCKERHUB_USERNAME and DOCKERHUB_TOKEN must be set."
        exit "$ERROR_DOCKER_LOGIN_FAILED"
    fi
    echo "${DOCKERHUB_TOKEN}" | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin
    log::success "Successfully logged into Docker Hub."
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
        image_name="@vrooli/${service}" # This is the local image name convention used by pnpm + Docker
        
        # Tag with specific version
        version_tag="${DOCKERHUB_USERNAME}/${service}:${current_version}"
        log::info "Tagging image ${image_name} as ${version_tag}"
        if ! docker tag "${image_name}" "${version_tag}"; then
            log::error "Failed to tag image ${image_name} with version ${current_version}. Does the local image exist?"
            continue 
        fi
    
        log::info "Pushing image ${version_tag} to Docker Hub..."
        if ! docker push "${version_tag}"; then
            log::error "Failed to push image ${version_tag}."
        else
            log::success "Successfully pushed ${version_tag}"
        fi

        # Tag with floating tag (dev/prod)
        local floating_tag_full="${DOCKERHUB_USERNAME}/${service}:${floating_tag}"
        log::info "Tagging image ${image_name} as ${floating_tag_full}"
        if ! docker tag "${image_name}" "${floating_tag_full}"; then
            log::error "Failed to tag image ${image_name} with floating tag ${floating_tag}."
            # This failure is less critical than the versioned tag, so don't continue here necessarily
        else
            log::info "Pushing image ${floating_tag_full} to Docker Hub..."
            if ! docker push "${floating_tag_full}"; then
                log::error "Failed to push image ${floating_tag_full}."
            else
                log::success "Successfully pushed ${floating_tag_full}"
            fi
        fi
    done
  
    log::success "Finished tagging and pushing images."
}