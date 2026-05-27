#!/usr/bin/env bash
################################################################################
# Kyutai STT Installation Functions
#
# Functions for installing and managing the Kyutai STT service.
################################################################################

#######################################
# Install Kyutai STT service
#######################################
kyutai_stt::install() {
    log::header "🎤 Installing Kyutai STT (streaming speech-to-text)"

    # Check if already installed
    if common::container_exists && common::is_running && [[ "${FORCE:-false}" != "true" ]]; then
        log::info "${MSG_ALREADY_INSTALLED}"
        log::info "Use FORCE=true to reinstall"
        return 0
    fi

    # Validate Docker is available
    if ! common::check_docker; then
        return 1
    fi

    # Warn (not block) when no GPU: model needs CUDA for real-time streaming.
    if ! kyutai_stt::is_gpu_available; then
        log::warn "${MSG_GPU_NOT_AVAILABLE}"
    fi

    # Create necessary directories (HF cache bind-mount target)
    kyutai_stt::create_directories

    # Stop existing container if running
    if common::is_running; then
        log::info "Stopping existing Kyutai STT container..."
        docker::stop_container "$KYUTAI_STT_CONTAINER_NAME"
    fi

    # Remove existing container if it exists
    if common::container_exists; then
        log::info "Removing existing Kyutai STT container..."
        docker::remove_container "$KYUTAI_STT_CONTAINER_NAME" "true"
    fi

    # Start the container (compose builds the image on first run)
    log::info "${MSG_STARTING_CONTAINER}"
    if ! kyutai_stt::docker::start_container; then
        log::error "${MSG_START_CONTAINER_FAILED}"
        return 1
    fi

    # Wait for service to be healthy (first run downloads weights)
    log::info "${MSG_WAITING_INIT}"
    if kyutai_stt::wait_for_health; then
        log::success "${MSG_INSTALL_SUCCESS}"
        log::info "Service URL: ${KYUTAI_STT_BASE_URL:-http://localhost:8094}"
        log::info "Stream endpoint (WS): ${KYUTAI_STT_WS_URL:-ws://localhost:8094/v1/stream}"
        log::info "Model: ${KYUTAI_STT_HF_REPO:-kyutai/stt-1b-en_fr}"
    else
        log::error "${MSG_STARTED_NOT_HEALTHY}"
        return 1
    fi
}

#######################################
# Uninstall Kyutai STT service
#######################################
kyutai_stt::uninstall() {
    log::header "🗑️  Uninstalling Kyutai STT"

    # Stop container if running
    if common::is_running; then
        log::info "Stopping Kyutai STT container..."
        docker::stop_container "$KYUTAI_STT_CONTAINER_NAME"
    fi

    # Remove container if it exists
    if common::container_exists; then
        log::info "Removing Kyutai STT container..."
        docker::remove_container "$KYUTAI_STT_CONTAINER_NAME" "true"
    fi

    # Cleanup
    kyutai_stt::cleanup

    log::success "${MSG_UNINSTALL_SUCCESS}"
}

#######################################
# Start Kyutai STT service
#######################################
kyutai_stt::start() {
    if common::is_running; then
        log::info "${MSG_ALREADY_RUNNING}"
        return 0
    fi

    log::info "${MSG_STARTING_CONTAINER}"
    if kyutai_stt::docker::start_container; then
        log::info "${MSG_WAITING_INIT}"
        if kyutai_stt::wait_for_health; then
            log::success "${MSG_START_SUCCESS}"
        else
            log::error "${MSG_STARTED_NOT_HEALTHY}"
            return 1
        fi
    else
        log::error "${MSG_START_CONTAINER_FAILED}"
        return 1
    fi
}

#######################################
# Stop Kyutai STT service
#######################################
kyutai_stt::stop() {
    if ! common::is_running; then
        log::info "${MSG_NOT_RUNNING}"
        return 0
    fi

    log::info "${MSG_STOPPING}"
    if docker::stop_container "$KYUTAI_STT_CONTAINER_NAME"; then
        log::success "${MSG_STOP_SUCCESS}"
    else
        log::error "${MSG_STOP_FAILED}"
        return 1
    fi
}

#######################################
# Restart Kyutai STT service
#######################################
kyutai_stt::restart() {
    log::info "${MSG_RESTARTING}"

    if kyutai_stt::docker::restart_container; then
        log::info "${MSG_WAITING_INIT}"
        if kyutai_stt::wait_for_health; then
            log::success "${MSG_RESTART_SUCCESS}"
        else
            log::error "${MSG_STARTED_NOT_HEALTHY}"
            return 1
        fi
    else
        log::error "${MSG_START_CONTAINER_FAILED}"
        return 1
    fi
}

#######################################
# Show Kyutai STT logs
#######################################
kyutai_stt::show_logs() {
    kyutai_stt::docker::show_logs
}

#######################################
# Get Kyutai STT container stats
#######################################
kyutai_stt::get_stats() {
    kyutai_stt::docker::show_stats
}
