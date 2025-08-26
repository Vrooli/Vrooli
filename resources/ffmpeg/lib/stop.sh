#!/bin/bash

ffmpeg_stop() {
    # Get the directory of this lib file
    local APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
    local FFMPEG_LIB_DIR="${APP_ROOT}/resources/ffmpeg/lib"
    source "${APP_ROOT}/scripts/lib/utils/format.sh"
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
    
    # FFmpeg is a CLI tool, not a service - nothing to stop
    log::info "FFmpeg is a command-line tool and doesn't run as a service"
    return 0
}
