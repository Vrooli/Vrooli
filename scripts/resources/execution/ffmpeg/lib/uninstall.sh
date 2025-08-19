#!/bin/bash

ffmpeg_uninstall() {
    local force="${1:-false}"
    
    # Get the directory of this lib file
    local FFMPEG_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/format.sh"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh"
    
    if [[ "$force" != "--force" ]]; then
        log::error "Uninstall requires --force flag to prevent accidental removal"
        return 1
    fi
    
    log::header "🗑️ Uninstalling FFmpeg"
    
    if ! command -v ffmpeg &> /dev/null; then
        log::info "FFmpeg is not installed"
        return 0
    fi
    
    log::info "Removing FFmpeg..."
    sudo apt-get remove -y ffmpeg &>/dev/null || {
        log::error "Failed to uninstall FFmpeg"
        return 1
    }
    
    log::success "FFmpeg uninstalled successfully"
    return 0
}
