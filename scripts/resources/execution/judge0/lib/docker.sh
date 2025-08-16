#!/usr/bin/env bash
# Judge0 Docker Management - Simplified with docker-resource-utils.sh
# Handles multi-service Judge0 container lifecycle operations

# Source simplified docker utilities
_JUDGE0_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_JUDGE0_DOCKER_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Start Judge0 services
#######################################
judge0::docker::start() {
    if ! judge0::is_installed; then
        log::error "$JUDGE0_MSG_STATUS_NOT_INSTALLED"
        return 1
    fi
    
    if judge0::is_running; then
        log::info "Judge0 is already running"
        return 0
    fi
    
    log::info "$JUDGE0_MSG_START"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    if docker compose -f "$compose_file" start >/dev/null 2>&1; then
        if judge0::wait_for_health; then
            log::success "Judge0 started successfully"
            return 0
        else
            log::error "Judge0 started but health check failed"
            return 1
        fi
    else
        log::error "Failed to start Judge0"
        return 1
    fi
}

#######################################
# Stop Judge0 services
#######################################
judge0::docker::stop() {
    if ! judge0::is_running; then
        log::info "Judge0 is not running"
        return 0
    fi
    
    log::info "$JUDGE0_MSG_STOP"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    if docker compose -f "$compose_file" stop >/dev/null 2>&1; then
        log::success "Judge0 stopped successfully"
        return 0
    else
        log::error "Failed to stop Judge0"
        return 1
    fi
}

#######################################
# Restart Judge0 services
#######################################
judge0::docker::restart() {
    log::info "$JUDGE0_MSG_RESTART"
    
    if judge0::docker::stop; then
        sleep ${JUDGE0_RESTART_WAIT:-2}
        judge0::docker::start
    else
        return 1
    fi
}

#######################################
# Show Judge0 logs
#######################################
judge0::docker::logs() {
    if ! judge0::is_installed; then
        log::error "$JUDGE0_MSG_STATUS_NOT_INSTALLED"
        return 1
    fi
    
    log::info "$JUDGE0_MSG_LOGS"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    # Show logs from all services
    docker compose -f "$compose_file" logs --tail=100 --follow
}

#######################################
# Uninstall Judge0
# Arguments:
#   $1 - Force uninstall (yes/no)
#######################################
judge0::docker::uninstall() {
    local force="${1:-no}"
    
    if ! judge0::is_installed; then
        log::info "Judge0 is not installed"
        return 0
    fi
    
    # Use simplified data removal confirmation
    if ! docker_resource::remove_data "Judge0" "${JUDGE0_DATA_DIR}" "$force"; then
        return 0  # User cancelled
    fi
    
    log::info "$JUDGE0_MSG_UNINSTALL"
    
    # Stop services
    judge0::docker::stop
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    # Remove containers and volumes using docker-compose
    if [[ -f "$compose_file" ]]; then
        docker compose -f "$compose_file" down -v >/dev/null 2>&1
    fi
    
    # Remove network and volume
    # Remove network only if empty
    docker::cleanup_network_if_empty "$JUDGE0_NETWORK_NAME"
    docker volume rm "$JUDGE0_VOLUME_NAME" >/dev/null 2>&1 || true
    
    log::success "Judge0 uninstalled successfully"
}

#######################################
# Update Judge0 to latest version
#######################################
judge0::docker::update() {
    log::info "Checking for Judge0 updates..."
    
    # Get current version
    local current_version=$(judge0::get_version)
    
    # Pull latest image using simplified utility
    local latest_image="${JUDGE0_IMAGE}:latest"
    if ! docker::pull_image "$latest_image"; then
        log::error "Failed to pull latest Judge0 image"
        return 1
    fi
    
    # Get latest version
    local latest_version=$(docker inspect "$latest_image" --format '{{.Config.Labels.version}}' 2>/dev/null || echo "unknown")
    
    if [[ "$current_version" == "$latest_version" ]]; then
        log::info "Judge0 is already up to date (version: $current_version)"
        return 0
    fi
    
    log::info "Updating Judge0 from $current_version to $latest_version..."
    
    # Stop current services
    judge0::docker::stop
    
    # Update compose file with new version
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    sed -i "s/${JUDGE0_IMAGE}:${JUDGE0_VERSION}/${JUDGE0_IMAGE}:${latest_version}/g" "$compose_file"
    
    # Update version in config
    export JUDGE0_VERSION="$latest_version"
    
    # Start updated services
    judge0::docker::start
    
    log::success "Judge0 updated to version $latest_version"
}

#######################################
# Scale Judge0 workers
# Arguments:
#   $1 - Number of workers
#######################################
judge0::docker::scale_workers() {
    local worker_count="${1:-$JUDGE0_WORKERS_COUNT}"
    
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    log::info "Scaling Judge0 workers to $worker_count..."
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    if docker compose -f "$compose_file" up -d --scale judge0-workers="$worker_count" >/dev/null 2>&1; then
        log::success "Successfully scaled workers to $worker_count"
        export JUDGE0_WORKERS_COUNT="$worker_count"
    else
        log::error "Failed to scale workers"
        return 1
    fi
}

