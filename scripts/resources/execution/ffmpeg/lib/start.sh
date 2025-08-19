#!/bin/bash

ffmpeg_start() {
    # Get the directory of this lib file
    local FFMPEG_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/format.sh"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh"
    
    # FFmpeg is a CLI tool, not a service
    # "Starting" means ensuring it's installed and available
    
    if command -v ffmpeg &> /dev/null; then
        log::success "FFmpeg is available and ready to use"
        return 0
    else
        log::error "FFmpeg is not installed"
        log::info "Run: vrooli resource ffmpeg install"
        return 1
    fi
}
