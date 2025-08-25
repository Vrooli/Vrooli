#!/bin/bash

ffmpeg_install() {
    local force="${1:-false}"
    
    # Get the directory of this lib file
    local APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
    local FFMPEG_LIB_DIR="${APP_ROOT}/resources/ffmpeg/lib"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/format.sh"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh"
    
    log::header "🎬 Installing FFmpeg"
    
    # Check if already installed
    if command -v ffmpeg &> /dev/null; then
        local version=$(ffmpeg -version 2>&1 | head -n1 | cut -d' ' -f3)
        log::success "FFmpeg ${version} already installed"
        return 0
    fi
    
    log::info "Installing FFmpeg via apt..."
    
    # Update package list
    sudo apt-get update &>/dev/null || {
        log::error "Failed to update package list"
        return 1
    }
    
    # Install FFmpeg
    sudo apt-get install -y ffmpeg &>/dev/null || {
        log::error "Failed to install FFmpeg"
        return 1
    }
    
    # Verify installation
    if command -v ffmpeg &> /dev/null; then
        local version=$(ffmpeg -version 2>&1 | head -n1 | cut -d' ' -f3)
        log::success "FFmpeg ${version} installed successfully"
        return 0
    else
        log::error "FFmpeg installation verification failed"
        return 1
    fi
}
