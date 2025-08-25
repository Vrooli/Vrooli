#!/usr/bin/env bash
# FFmpeg Configuration Defaults

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
FFMPEG_CONFIG_DIR="${APP_ROOT}/resources/ffmpeg/config"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# Source port registry for dynamic port allocation (if needed)
# shellcheck disable=SC1091
source "${FFMPEG_CONFIG_DIR}/../../../port_registry.sh"

# Export configuration function
ffmpeg::export_config() {
    # Basic configuration
    export FFMPEG_NAME="ffmpeg"
    export FFMPEG_DISPLAY_NAME="FFmpeg"
    export FFMPEG_DESCRIPTION="Universal media processing framework"
    export FFMPEG_CATEGORY="execution"
    
    # Paths
    export FFMPEG_DATA_DIR="${FFMPEG_DATA_DIR:-${var_DATA_DIR}/resources/ffmpeg}"
    export FFMPEG_TEMP_DIR="${FFMPEG_TEMP_DIR:-${FFMPEG_DATA_DIR}/temp}"
    export FFMPEG_OUTPUT_DIR="${FFMPEG_OUTPUT_DIR:-${FFMPEG_DATA_DIR}/output}"
    export FFMPEG_LOGS_DIR="${FFMPEG_LOGS_DIR:-${FFMPEG_DATA_DIR}/logs}"
    
    # Processing defaults
    export FFMPEG_DEFAULT_VIDEO_CODEC="${FFMPEG_DEFAULT_VIDEO_CODEC:-libx264}"
    export FFMPEG_DEFAULT_AUDIO_CODEC="${FFMPEG_DEFAULT_AUDIO_CODEC:-aac}"
    export FFMPEG_DEFAULT_QUALITY="${FFMPEG_DEFAULT_QUALITY:-23}"  # CRF value
    export FFMPEG_DEFAULT_PRESET="${FFMPEG_DEFAULT_PRESET:-medium}"
    export FFMPEG_DEFAULT_THREADS="${FFMPEG_DEFAULT_THREADS:-0}"  # 0 = auto
    
    # Hardware acceleration
    export FFMPEG_HW_ACCEL="${FFMPEG_HW_ACCEL:-auto}"  # auto, vaapi, nvenc, none
    export FFMPEG_HW_DEVICE="${FFMPEG_HW_DEVICE:-/dev/dri/renderD128}"
    
    # Limits
    export FFMPEG_MAX_FILE_SIZE="${FFMPEG_MAX_FILE_SIZE:-5G}"
    export FFMPEG_TIMEOUT="${FFMPEG_TIMEOUT:-3600}"  # 1 hour default
    
    # Log level
    export FFMPEG_LOG_LEVEL="${FFMPEG_LOG_LEVEL:-warning}"
}