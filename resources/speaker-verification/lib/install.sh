#!/usr/bin/env bash
# Speaker Verification - Install/Uninstall/Lifecycle

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

#######################################
# Install the speaker verification resource
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::install() {
    echo "=== Installing Speaker Verification Resource ==="
    echo

    # Check Docker
    if ! common::check_docker; then
        return 1
    fi

    # Check if already running
    if common::is_running; then
        if speaker_verification::is_healthy; then
            log::info "Speaker verification is already installed and running"
            speaker_verification::display_connection_info
            return 0
        fi
    fi

    # Check port availability
    if ! common::is_port_available "$SPEAKER_VERIFICATION_PORT"; then
        log::error "Port ${SPEAKER_VERIFICATION_PORT} is already in use"
        return 1
    fi

    # Create data directories
    speaker_verification::create_directories

    # Stop and remove existing container if present
    if common::container_exists; then
        log::info "Removing existing container..."
        speaker_verification::cleanup
    fi

    # Build the Docker image
    if ! speaker_verification::docker::build_image; then
        log::error "Failed to build Docker image"
        return 1
    fi

    # Start the container
    if ! speaker_verification::docker::start_container; then
        log::error "Failed to start container"
        return 1
    fi

    # Wait for health
    if ! speaker_verification::wait_for_health; then
        log::error "Service failed to become healthy"
        log::info "Check logs with: resource-speaker-verification logs"
        return 1
    fi

    log::success "$MSG_INSTALL_SUCCESS"
    speaker_verification::display_connection_info
    return 0
}
export -f speaker_verification::install

#######################################
# Uninstall the speaker verification resource
# Arguments: --force (optional, removes data too)
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::uninstall() {
    echo "=== Uninstalling Speaker Verification Resource ==="
    echo

    local force="false"
    for arg in "$@"; do
        [[ "$arg" == "--force" ]] && force="true"
    done

    # Stop container
    speaker_verification::docker::stop_container

    # Remove container
    if common::container_exists; then
        log::info "Removing container..."
        docker rm -f "$SPEAKER_VERIFICATION_CONTAINER_NAME" &>/dev/null || true
    fi

    # Remove image (optional)
    if [[ "$force" == "true" ]]; then
        log::info "Removing Docker image..."
        docker rmi "$SPEAKER_VERIFICATION_IMAGE" &>/dev/null || true

        log::info "Removing cache..."
        rm -rf "${SPEAKER_VERIFICATION_CACHE_DIR}" 2>/dev/null || true
    fi

    # Remove network
    docker network rm "$SPEAKER_VERIFICATION_NETWORK_NAME" &>/dev/null || true

    if [[ "$force" == "true" ]]; then
        log::warn "Profiles preserved at: ${SPEAKER_VERIFICATION_PROFILES_DIR}"
        log::info "To remove profiles: rm -rf ${SPEAKER_VERIFICATION_PROFILES_DIR}"
    fi

    log::success "Speaker verification uninstalled"
    return 0
}
export -f speaker_verification::uninstall

#######################################
# Start the speaker verification service
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::start() {
    log::info "$MSG_STARTING"

    if common::is_running; then
        log::info "Speaker verification is already running"
        return 0
    fi

    if ! common::check_docker; then
        return 1
    fi

    speaker_verification::create_directories

    # If container exists but is stopped, start it
    if common::container_exists; then
        log::info "Starting existing container..."
        docker start "$SPEAKER_VERIFICATION_CONTAINER_NAME" &>/dev/null
    else
        # Need to create a new container
        if ! speaker_verification::docker::start_container; then
            return 1
        fi
    fi

    # Wait for health
    if ! speaker_verification::wait_for_health; then
        log::error "Service failed to become healthy after start"
        return 1
    fi

    log::success "$MSG_START_SUCCESS"
    return 0
}
export -f speaker_verification::start

#######################################
# Stop the speaker verification service
# Returns: 0 on success
#######################################
speaker_verification::stop() {
    log::info "$MSG_STOPPING"
    speaker_verification::docker::stop_container
    log::success "$MSG_STOP_SUCCESS"
    return 0
}
export -f speaker_verification::stop

#######################################
# Restart the speaker verification service
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::restart() {
    log::info "Restarting speaker verification..."
    speaker_verification::stop
    sleep 2
    speaker_verification::start
}
export -f speaker_verification::restart

#######################################
# Show logs
#######################################
speaker_verification::show_logs() {
    speaker_verification::docker::show_logs "$@"
}
export -f speaker_verification::show_logs
