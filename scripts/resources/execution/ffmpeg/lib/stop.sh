#!/bin/bash

ffmpeg_stop() {
    # Get the directory of this lib file
    local FFMPEG_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/format.sh"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh"
    
    # FFmpeg is a CLI tool, not a service - nothing to stop
    log::info "FFmpeg is a command-line tool and doesn't run as a service"
    return 0
}
