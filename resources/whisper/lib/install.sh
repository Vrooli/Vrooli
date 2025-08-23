#!/usr/bin/env bash
################################################################################
# Whisper Installation Functions
# 
# Functions for installing and managing Whisper service
################################################################################

#######################################
# Install Whisper service
#######################################
whisper::install() {
    log::header "🎤 Installing Whisper Speech-to-Text"
    
    # Check if already installed
    if common::container_exists "$WHISPER_CONTAINER_NAME" && common::is_running "$WHISPER_CONTAINER_NAME" && [[ "${FORCE:-false}" != "true" ]]; then
        log::info "Whisper is already installed and running"
        log::info "Use FORCE=true to reinstall"
        return 0
    fi
    
    # Validate Docker is available
    if ! common::check_docker; then
        return 1
    fi
    
    # Install ffmpeg if not present
    if ! system::is_command "ffmpeg"; then
        log::info "Installing ffmpeg for audio format conversion..."
        local pm
        pm=$(system::detect_pm)
        
        case "$pm" in
            "apt-get")
                if sudo apt-get update && sudo apt-get install -y ffmpeg; then
                    log::success "ffmpeg installed successfully"
                else
                    log::warn "Failed to install ffmpeg. Audio format conversion may be limited."
                fi
                ;;
            "dnf"|"yum")
                if sudo "$pm" install -y ffmpeg; then
                    log::success "ffmpeg installed successfully"
                else
                    log::warn "Failed to install ffmpeg. Audio format conversion may be limited."
                fi
                ;;
            "pacman")
                if sudo pacman -S --noconfirm ffmpeg; then
                    log::success "ffmpeg installed successfully"
                else
                    log::warn "Failed to install ffmpeg. Audio format conversion may be limited."
                fi
                ;;
            *)
                log::warn "Unable to auto-install ffmpeg on this system (package manager: $pm). Please install manually for full audio format support."
                ;;
        esac
    fi
    
    # Create necessary directories
    whisper::create_directories
    
    # Stop existing container if running
    if common::is_running "$WHISPER_CONTAINER_NAME"; then
        log::info "Stopping existing Whisper container..."
        docker::stop_container "$WHISPER_CONTAINER_NAME"
    fi
    
    # Remove existing container if it exists
    if common::container_exists "$WHISPER_CONTAINER_NAME"; then
        log::info "Removing existing Whisper container..."
        docker::remove_container "$WHISPER_CONTAINER_NAME" "true"
    fi
    
    # Pull the Docker image
    log::info "Pulling Whisper Docker image..."
    if ! whisper::docker::pull_image; then
        log::error "Failed to pull Whisper Docker image"
        return 1
    fi
    
    # Start the container
    log::info "Starting Whisper container..."
    if ! whisper::docker::start_container; then
        log::error "Failed to start Whisper container"
        return 1
    fi
    
    # Wait for service to be healthy
    log::info "Waiting for Whisper service to be ready..."
    if whisper::wait_for_health; then
        log::success "✅ Whisper installed and running successfully"
        log::info "Service URL: ${WHISPER_BASE_URL:-http://localhost:9000}"
        log::info "API Documentation: ${WHISPER_BASE_URL:-http://localhost:9000}/docs"
        log::info "Default model: ${WHISPER_DEFAULT_MODEL:-base}"
    else
        log::error "Whisper service failed to start properly"
        return 1
    fi
}

#######################################
# Uninstall Whisper service
#######################################
whisper::uninstall() {
    log::header "🗑️  Uninstalling Whisper Speech-to-Text"
    
    # Stop container if running
    if common::is_running "$WHISPER_CONTAINER_NAME"; then
        log::info "Stopping Whisper container..."
        docker::stop_container "$WHISPER_CONTAINER_NAME"
    fi
    
    # Remove container if it exists
    if common::container_exists "$WHISPER_CONTAINER_NAME"; then
        log::info "Removing Whisper container..."
        docker::remove_container "$WHISPER_CONTAINER_NAME" "true"
    fi
    
    # Clean up whisper cleanup
    whisper::cleanup
    
    log::success "✅ Whisper uninstalled successfully"
}

#######################################
# Start Whisper service
#######################################
whisper::start() {
    if common::is_running "$WHISPER_CONTAINER_NAME"; then
        log::info "Whisper is already running"
        return 0
    fi
    
    if ! common::container_exists "$WHISPER_CONTAINER_NAME"; then
        log::error "Whisper container does not exist. Install first with: resource-whisper install"
        return 1
    fi
    
    log::info "Starting Whisper container..."
    if whisper::docker::start_container; then
        log::info "Waiting for service to be ready..."
        if whisper::wait_for_health; then
            log::success "✅ Whisper started successfully"
        else
            log::error "Service failed to start properly"
            return 1
        fi
    else
        log::error "Failed to start Whisper container"
        return 1
    fi
}

#######################################
# Stop Whisper service
#######################################
whisper::stop() {
    if ! common::is_running "$WHISPER_CONTAINER_NAME"; then
        log::info "Whisper is not running"
        return 0
    fi
    
    log::info "Stopping Whisper container..."
    if docker::stop_container "$WHISPER_CONTAINER_NAME"; then
        log::success "✅ Whisper stopped successfully"
    else
        log::error "Failed to stop Whisper container"
        return 1
    fi
}

#######################################
# Restart Whisper service
#######################################
whisper::restart() {
    log::info "Restarting Whisper service..."
    
    if whisper::docker::restart_container; then
        log::info "Waiting for service to be ready..."
        if whisper::wait_for_health; then
            log::success "✅ Whisper restarted successfully"
        else
            log::error "Service failed to restart properly"
            return 1
        fi
    else
        log::error "Failed to restart Whisper container"
        return 1
    fi
}

#######################################
# Show Whisper logs
#######################################
whisper::show_logs() {
    whisper::docker::show_logs
}

#######################################
# Get Whisper container stats
#######################################
whisper::get_stats() {
    whisper::docker::show_stats
}