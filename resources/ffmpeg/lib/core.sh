#!/bin/bash
# FFmpeg core functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
FFMPEG_CORE_DIR="${APP_ROOT}/resources/ffmpeg/lib"

# Source shared utilities (using relative path from lib to scripts)
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Configuration
export FFMPEG_HOME="${FFMPEG_HOME:-$HOME/.ffmpeg}"
export FFMPEG_BIN="${FFMPEG_HOME}/bin/ffmpeg"
export FFMPEG_PROBE_BIN="${FFMPEG_HOME}/bin/ffprobe"
export FFMPEG_VERSION="${FFMPEG_VERSION:-6.1.1}"
export FFMPEG_DATA_DIR="${var_VROOLI_BASE_DIR:-/root/Vrooli}/.vrooli/ffmpeg"
export FFMPEG_SCRIPTS_DIR="${FFMPEG_DATA_DIR}/scripts"
export FFMPEG_OUTPUT_DIR="${FFMPEG_DATA_DIR}/output"

# Initialize FFmpeg environment
ffmpeg::init() {
    # Create necessary directories
    mkdir -p "$FFMPEG_HOME" "$FFMPEG_DATA_DIR" "$FFMPEG_SCRIPTS_DIR" "$FFMPEG_OUTPUT_DIR"
    
    # Add FFmpeg to PATH if not already there
    if [[ ":$PATH:" != *":$FFMPEG_HOME/bin:"* ]]; then
        export PATH="$FFMPEG_HOME/bin:$PATH"
    fi
    
    return 0
}

# Test FFmpeg installation
ffmpeg::test_installation() {
    if [[ -x "$FFMPEG_BIN" ]]; then
        "$FFMPEG_BIN" -version >/dev/null 2>&1
        return $?
    elif command -v ffmpeg >/dev/null 2>&1; then
        ffmpeg -version >/dev/null 2>&1
        return $?
    fi
    return 1
}

# Get FFmpeg version
ffmpeg::get_version() {
    if [[ -x "$FFMPEG_BIN" ]]; then
        "$FFMPEG_BIN" -version 2>/dev/null | head -n1 | awk '{print $3}'
    elif command -v ffmpeg >/dev/null 2>&1; then
        ffmpeg -version 2>/dev/null | head -n1 | awk '{print $3}'
    else
        echo "not_installed"
    fi
}

# Export functions
export -f ffmpeg::init
export -f ffmpeg::test_installation
export -f ffmpeg::get_version