#!/usr/bin/env bash
################################################################################
# Vrooli-Specific Docker Functions
# 
# App-specific Docker utilities that extend the universal docker.sh
# This file sources the universal docker functions and adds Vrooli-specific ones.
################################################################################

set -euo pipefail

APP_UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_UTILS_DIR}/../../lib/utils/var.sh"

# Source universal docker functions
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/docker.sh"

# Source Vrooli-specific dependencies
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/env.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/repository.sh"

################################################################################
# Vrooli-Specific Docker Functions
################################################################################

# Get the appropriate compose file based on environment
docker::get_compose_file() {
    if env::in_production; then
        echo "${var_DOCKER_COMPOSE_PROD_FILE}"
    else
        echo "${var_DOCKER_COMPOSE_DEV_FILE}"
    fi
}

# Override docker::compose to handle docker directory
# This wrapper ensures we use the correct compose file from the project root
docker::compose() {
    local compose_file=$(docker::get_compose_file)
    
    # Verify compose file exists
    if [[ ! -f "$compose_file" ]]; then
        log::error "Docker Compose file not found: $compose_file"
        return 1
    fi
    
    # Get the compose command from universal docker.sh
    local compose_cmd
    if ! compose_cmd=$(docker::_get_compose_command); then
        log::error "No Docker Compose version found"
        return 1
    fi
    
    # Change to project root for correct relative paths in docker-compose.yml
    cd "$var_ROOT_DIR" || {
        log::error "Failed to change to project root: $var_ROOT_DIR"
        return 1
    }
    
    # Execute compose command with explicit compose file path
    local exit_code=0
    if [[ "$compose_cmd" == "docker compose" ]]; then
        docker::_execute_with_permissions "docker" "compose" "-f" "$compose_file" "$@" || exit_code=$?
    else
        docker::_execute_with_permissions "docker-compose" "-f" "$compose_file" "$@" || exit_code=$?
    fi
    
    return $exit_code
}

# Build Docker images for Vrooli services
docker::build_images() {
    log::header "Building Docker images"
    local compose_file=$(docker::get_compose_file)
    cd "$var_ROOT_DIR" || { log::error "Failed to change directory to project root"; exit "$ERROR_BUILD_FAILED"; }
    
    # Check if compose is available
    if ! docker::_get_compose_command >/dev/null 2>&1; then
        log::error "No Docker Compose available to build images"
        exit "$ERROR_BUILD_FAILED"
    fi
    
    # Get repository metadata for build labels
    local repo_url repo_branch git_commit build_date
    repo_url=$(repository::get_url 2>/dev/null || echo "unknown")
    repo_branch=$(repository::get_branch 2>/dev/null || echo "unknown")
    git_commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    build_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Build with repository metadata as labels
    log::info "Building with metadata: repo=$repo_url, branch=$repo_branch, commit=$git_commit"
    
    if ! docker::compose -f "$compose_file" build \
        --build-arg "BUILD_DATE=$build_date" \
        --build-arg "VCS_REF=$git_commit" \
        --build-arg "VCS_URL=$repo_url" \
        --build-arg "VCS_BRANCH=$repo_branch"; then
        log::error "Failed to build Docker images"
        exit "$ERROR_BUILD_FAILED"
    fi
    
    log::success "Docker images built successfully"
}

# Pull base images from Docker Hub
docker::pull_base_images() {
    log::header "Pulling base images from Docker Hub"
    local compose_file=$(docker::get_compose_file)
    cd "$var_ROOT_DIR" || { log::error "Failed to change directory to project root"; exit "$ERROR_BUILD_FAILED"; }
    
    # Pull all images defined in compose file
    if ! docker::compose -f "$compose_file" pull; then
        log::error "Failed to pull base images"
        exit "$ERROR_BUILD_FAILED"
    fi
    
    log::success "Base images pulled successfully"
}

# Save Docker images to tar files
docker::save_images() {
    log::header "Saving Docker images to tar files"
    
    local output_dir="${1:-$var_ROOT_DIR/docker-images}"
    mkdir -p "$output_dir"
    
    # Get list of services from compose file
    local compose_file=$(docker::get_compose_file)
    local services=$(docker::compose -f "$compose_file" config --services)
    
    for service in $services; do
        local image_name="${service}:latest"
        local output_file="$output_dir/${service}.tar"
        
        log::info "Saving image $image_name to $output_file"
        if docker save "$image_name" -o "$output_file"; then
            log::success "Saved $image_name"
        else
            log::warning "Failed to save $image_name (may not exist)"
        fi
    done
    
    log::success "Docker images saved to $output_dir"
}

# Load Docker images from tar files
docker::load_images_from_tar() {
    log::header "Loading Docker images from tar files"
    
    local input_dir="${1:-$var_ROOT_DIR/docker-images}"
    
    if [[ ! -d "$input_dir" ]]; then
        log::error "Directory not found: $input_dir"
        return 1
    fi
    
    for tar_file in "$input_dir"/*.tar; do
        if [[ -f "$tar_file" ]]; then
            log::info "Loading image from $tar_file"
            if docker load -i "$tar_file"; then
                log::success "Loaded $(basename "$tar_file")"
            else
                log::error "Failed to load $(basename "$tar_file")"
            fi
        fi
    done
    
    log::success "Docker images loaded"
}

# Build Docker artifacts (for deployment)
docker::build_artifacts() {
    log::header "Building Docker artifacts"
    
    # First build the images
    docker::build_images
    
    # Then save them as artifacts if requested
    if [[ "${SAVE_DOCKER_ARTIFACTS:-false}" == "true" ]]; then
        docker::save_images
    fi
    
    log::success "Docker artifacts built"
}

# Login to Docker Hub
docker::login_to_dockerhub() {
    log::info "Logging into Docker Hub..."
    echo "$DOCKERHUB_PASSWORD" | docker::run login -u "$DOCKERHUB_USERNAME" --password-stdin
    log::success "Successfully logged into Docker Hub"
}

# Tag and push images to Docker Hub
docker::tag_and_push_images() {
    local services=("$@")
    local current_version="${BUILD_VERSION:-latest}"
    local floating_tag
    
    log::header "Tagging and pushing Docker images to Docker Hub"
    
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

# Additional Docker utility functions for resources

# Check if Docker is available and running
docker::check() {
    local quiet="${1:-false}"
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        [[ "$quiet" != "true" ]] && log::error "Docker is not installed"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        [[ "$quiet" != "true" ]] && log::error "Docker daemon is not running"
        return 1
    fi
    
    [[ "$quiet" != "true" ]] && log::success "Docker is installed and running"
    return 0
}

# Duplicate alias for backward compatibility
docker::is_running() {
    docker::_is_running
}

# Check if Docker socket is exposed
docker::is_socket_exposed() {
    # Check if docker socket exists and is accessible
    if [[ -S /var/run/docker.sock ]]; then
        # Check if we can actually use it
        if docker version &>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Create a Docker network
docker::create_network() {
    local network_name="${1:-judge0}"
    
    # Check if network already exists
    if docker network ls --format '{{.Name}}' | grep -q "^${network_name}$"; then
        log::debug "Network ${network_name} already exists"
        return 0
    fi
    
    log::info "Creating Docker network: ${network_name}"
    if docker network create "${network_name}" >/dev/null 2>&1; then
        log::success "Created Docker network: ${network_name}"
        return 0
    else
        log::error "Failed to create Docker network: ${network_name}"
        return 1
    fi
}

# Remove a Docker network
docker::remove_network() {
    local network_name="${1:-judge0}"
    
    # Check if network exists
    if ! docker network ls --format '{{.Name}}' | grep -q "^${network_name}$"; then
        log::debug "Network ${network_name} does not exist"
        return 0
    fi
    
    log::info "Removing Docker network: ${network_name}"
    if docker network rm "${network_name}" >/dev/null 2>&1; then
        log::success "Removed Docker network: ${network_name}"
        return 0
    else
        log::error "Failed to remove Docker network: ${network_name}"
        return 1
    fi
}

# Fix volume permissions
docker::fix_volume_permissions() {
    local volume_path="$1"
    local user_id="${2:-1000}"
    local group_id="${3:-1000}"
    
    if [[ -z "$volume_path" ]]; then
        log::error "Volume path is required"
        return 1
    fi
    
    if [[ ! -d "$volume_path" ]]; then
        log::error "Volume path does not exist: $volume_path"
        return 1
    fi
    
    log::info "Fixing permissions for: $volume_path"
    
    # Use Docker to fix permissions to avoid sudo requirements
    if docker run --rm -v "$volume_path:/data" busybox chown -R "$user_id:$group_id" /data; then
        log::success "Fixed permissions for: $volume_path"
        return 0
    else
        log::error "Failed to fix permissions for: $volume_path"
        return 1
    fi
}

# Collect Docker images for bundling
docker::collect_images() {
    log::header "Collecting Docker images"
    
    local bundle_dir="${1:-$var_ROOT_DIR/bundle}"
    local images_dir="$bundle_dir/docker-images"
    
    mkdir -p "$images_dir"
    
    # Get list of images to save
    local compose_file=$(docker::get_compose_file)
    local services=$(docker::compose -f "$compose_file" config --services)
    
    for service in $services; do
        local image="${service}:latest"
        local tar_file="$images_dir/${service}.tar"
        
        if docker::run images -q "$image" >/dev/null 2>&1; then
            log::info "Saving $image to $tar_file"
            docker::run save "$image" -o "$tar_file"
        else
            log::warning "Image $image not found, skipping"
        fi
    done
    
    log::success "Docker images collected"
}

# Export functions for use by other scripts
export -f docker::get_compose_file
export -f docker::build_images
export -f docker::pull_base_images
export -f docker::save_images
export -f docker::load_images_from_tar
export -f docker::build_artifacts
export -f docker::login_to_dockerhub
export -f docker::tag_and_push_images
export -f docker::check
export -f docker::is_running
export -f docker::is_socket_exposed
export -f docker::create_network
export -f docker::remove_network
export -f docker::fix_volume_permissions
export -f docker::collect_images