#!/usr/bin/env bash
################################################################################
# Speaker Verification Installation Functions
#
# Functions for installing and managing the Speaker Verification service
################################################################################

#######################################
# Install Speaker Verification service
#######################################
speaker_verification::install() {
    log::header "🗣️  Installing Speaker Verification"

    # Check if already installed
    if common::container_exists && common::is_running && [[ "${FORCE:-false}" != "true" ]]; then
        log::info "Speaker Verification is already installed and running"
        log::info "Use FORCE=true to reinstall"
        return 0
    fi

    # Validate Docker is available
    if ! common::check_docker; then
        return 1
    fi

    # Create necessary directories
    speaker_verification::create_directories

    # Stop existing container if running
    if common::is_running; then
        log::info "Stopping existing Speaker Verification container..."
        docker::stop_container "$SPEAKER_VERIFICATION_CONTAINER_NAME"
    fi

    # Remove existing container if it exists
    if common::container_exists; then
        log::info "Removing existing Speaker Verification container..."
        docker::remove_container "$SPEAKER_VERIFICATION_CONTAINER_NAME" "true"
    fi

    # Build and start via compose (custom image build)
    log::info "${MSG_PULLING_IMAGE}"
    if ! speaker_verification::compose_up; then
        log::error "Failed to start Speaker Verification container"
        return 1
    fi

    # Wait for service to be healthy
    log::info "Waiting for Speaker Verification service to be ready..."
    if speaker_verification::wait_for_health; then
        log::success "✅ Speaker Verification installed and running successfully"
        log::info "Service URL: ${SPEAKER_VERIFICATION_BASE_URL:-http://localhost:11452}"
        log::info "Readiness endpoint: ${SPEAKER_VERIFICATION_BASE_URL:-http://localhost:11452}/ready"
        log::info "Model: ${SPEAKER_VERIFICATION_MODEL:-speechbrain/spkrec-ecapa-voxceleb}"
    else
        log::error "Speaker Verification service failed to start properly"
        return 1
    fi
}

#######################################
# Uninstall Speaker Verification service
#######################################
speaker_verification::uninstall() {
    log::header "🗑️  Uninstalling Speaker Verification"

    # Stop and remove via compose
    speaker_verification::compose_down "no"

    # Clean up container if still present
    speaker_verification::cleanup

    log::success "✅ Speaker Verification uninstalled successfully"
}

#######################################
# Start Speaker Verification service
#######################################
speaker_verification::start() {
    if common::is_running; then
        log::info "Speaker Verification is already running"
        return 0
    fi

    log::info "Starting Speaker Verification container..."
    if speaker_verification::compose_up; then
        log::info "Waiting for service to be ready..."
        if speaker_verification::wait_for_health; then
            log::success "✅ Speaker Verification started successfully"
        else
            log::error "Service failed to start properly"
            return 1
        fi
    else
        log::error "Failed to start Speaker Verification container"
        return 1
    fi
}

#######################################
# Stop Speaker Verification service
#######################################
speaker_verification::stop() {
    if ! common::is_running; then
        log::info "Speaker Verification is not running"
        return 0
    fi

    log::info "Stopping Speaker Verification container..."
    if docker::stop_container "$SPEAKER_VERIFICATION_CONTAINER_NAME"; then
        log::success "✅ Speaker Verification stopped successfully"
    else
        log::error "Failed to stop Speaker Verification container"
        return 1
    fi
}

#######################################
# Restart Speaker Verification service
#######################################
speaker_verification::restart() {
    log::info "Restarting Speaker Verification service..."

    speaker_verification::compose_down "no" >/dev/null 2>&1 || true

    if speaker_verification::compose_up; then
        log::info "Waiting for service to be ready..."
        if speaker_verification::wait_for_health; then
            log::success "✅ Speaker Verification restarted successfully"
        else
            log::error "Service failed to restart properly"
            return 1
        fi
    else
        log::error "Failed to restart Speaker Verification container"
        return 1
    fi
}

#######################################
# Show Speaker Verification logs
#######################################
speaker_verification::show_logs() {
    speaker_verification::docker::show_logs "$@"
}

#######################################
# Get Speaker Verification container stats
#######################################
speaker_verification::get_stats() {
    speaker_verification::docker::show_stats
}
