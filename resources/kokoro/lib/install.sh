#!/usr/bin/env bash
################################################################################
# Kokoro Installation Functions
#
# Functions for installing and managing Kokoro service
################################################################################

#######################################
# Install Kokoro service
#######################################
kokoro::install() {
    log::header "🔊 Installing Kokoro Text-to-Speech"

    # Check if already installed
    if common::container_exists "$KOKORO_CONTAINER_NAME" && common::is_running "$KOKORO_CONTAINER_NAME" && [[ "${FORCE:-false}" != "true" ]]; then
        log::info "Kokoro is already installed and running"
        log::info "Use FORCE=true to reinstall"
        return 0
    fi

    # Validate Docker is available
    if ! common::check_docker; then
        return 1
    fi

    # Create necessary directories
    kokoro::create_directories

    # Stop existing container if running
    if common::is_running "$KOKORO_CONTAINER_NAME"; then
        log::info "Stopping existing Kokoro container..."
        docker::stop_container "$KOKORO_CONTAINER_NAME"
    fi

    # Remove existing container if it exists
    if common::container_exists "$KOKORO_CONTAINER_NAME"; then
        log::info "Removing existing Kokoro container..."
        docker::remove_container "$KOKORO_CONTAINER_NAME" "true"
    fi

    # Pull the Docker image
    log::info "Pulling Kokoro Docker image..."
    if ! kokoro::docker::pull_image; then
        log::error "Failed to pull Kokoro Docker image"
        return 1
    fi

    # Start the container
    log::info "Starting Kokoro container..."
    if ! kokoro::docker::start_container; then
        log::error "Failed to start Kokoro container"
        return 1
    fi

    # Wait for service to be healthy
    log::info "Waiting for Kokoro service to be ready..."
    if kokoro::wait_for_health; then
        log::success "✅ Kokoro installed and running successfully"
        log::info "Service URL: ${KOKORO_BASE_URL:-http://localhost:8880}"
        log::info "Voices endpoint: ${KOKORO_BASE_URL:-http://localhost:8880}/v1/audio/voices"
        log::info "Default voice: ${KOKORO_DEFAULT_VOICE:-af_heart}"
    else
        log::error "Kokoro service failed to start properly"
        return 1
    fi
}

#######################################
# Uninstall Kokoro service
#######################################
kokoro::uninstall() {
    log::header "🗑️  Uninstalling Kokoro Text-to-Speech"

    # Stop container if running
    if common::is_running "$KOKORO_CONTAINER_NAME"; then
        log::info "Stopping Kokoro container..."
        docker::stop_container "$KOKORO_CONTAINER_NAME"
    fi

    # Remove container if it exists
    if common::container_exists "$KOKORO_CONTAINER_NAME"; then
        log::info "Removing Kokoro container..."
        docker::remove_container "$KOKORO_CONTAINER_NAME" "true"
    fi

    # Clean up kokoro cleanup
    kokoro::cleanup

    log::success "✅ Kokoro uninstalled successfully"
}

#######################################
# Start Kokoro service
#######################################
kokoro::start() {
    if common::is_running "$KOKORO_CONTAINER_NAME"; then
        log::info "Kokoro is already running"
        return 0
    fi

    if ! common::container_exists "$KOKORO_CONTAINER_NAME"; then
        log::error "Kokoro container does not exist. Install first with: resource-kokoro install"
        return 1
    fi

    log::info "Starting Kokoro container..."
    if kokoro::docker::start_container; then
        log::info "Waiting for service to be ready..."
        if kokoro::wait_for_health; then
            log::success "✅ Kokoro started successfully"
        else
            log::error "Service failed to start properly"
            return 1
        fi
    else
        log::error "Failed to start Kokoro container"
        return 1
    fi
}

#######################################
# Stop Kokoro service
#######################################
kokoro::stop() {
    if ! common::is_running "$KOKORO_CONTAINER_NAME"; then
        log::info "Kokoro is not running"
        return 0
    fi

    log::info "Stopping Kokoro container..."
    if docker::stop_container "$KOKORO_CONTAINER_NAME"; then
        log::success "✅ Kokoro stopped successfully"
    else
        log::error "Failed to stop Kokoro container"
        return 1
    fi
}

#######################################
# Restart Kokoro service
#######################################
kokoro::restart() {
    log::info "Restarting Kokoro service..."

    if kokoro::docker::restart_container; then
        log::info "Waiting for service to be ready..."
        if kokoro::wait_for_health; then
            log::success "✅ Kokoro restarted successfully"
        else
            log::error "Service failed to restart properly"
            return 1
        fi
    else
        log::error "Failed to restart Kokoro container"
        return 1
    fi
}

#######################################
# Show Kokoro logs
#######################################
kokoro::show_logs() {
    kokoro::docker::show_logs
}

#######################################
# Get Kokoro container stats
#######################################
kokoro::get_stats() {
    kokoro::docker::show_stats
}
