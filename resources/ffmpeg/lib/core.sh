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
# Data directories are now managed by defaults.sh

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

# Wrapper functions for CLI v2.0 shortcuts
ffmpeg::info() {
    local file_path="$1"
    if [[ -z "$file_path" ]]; then
        log::error "Usage: ffmpeg info <file_path>"
        return 1
    fi
    ffmpeg::inject::get_detailed_info "$file_path"
}

ffmpeg::transcode() {
    local file_path="$1"
    if [[ -z "$file_path" ]]; then
        log::error "Usage: ffmpeg transcode <file_path>"
        return 1
    fi
    # Initialize inject system
    ffmpeg::inject::init || return 1
    ffmpeg_inject "$file_path" "transcode"
}

ffmpeg::extract() {
    local file_path="$1"
    if [[ -z "$file_path" ]]; then
        log::error "Usage: ffmpeg extract <file_path>"
        return 1
    fi
    # Initialize inject system
    ffmpeg::inject::init || return 1
    ffmpeg_inject "$file_path" "extract"
}

ffmpeg::logs() {
    local tail_lines="${1:-50}"
    local follow="${2:-false}"
    
    # Initialize config
    ffmpeg::export_config
    
    if [[ ! -d "${FFMPEG_LOGS_DIR}" ]]; then
        log::info "No logs directory found. Run 'resource-ffmpeg manage start' first."
        return 0
    fi
    
    local latest_log=$(find "${FFMPEG_LOGS_DIR}" -name "*.log" -type f -printf '%T@ %p\n' 2>/dev/null | sort -n | tail -1 | cut -d' ' -f2-)
    
    if [[ -z "$latest_log" || ! -f "$latest_log" ]]; then
        log::info "No log files found in ${FFMPEG_LOGS_DIR}"
        return 0
    fi
    
    log::info "Showing logs from: $(basename "$latest_log")"
    
    if [[ "$follow" == "true" || "$follow" == "--follow" ]]; then
        tail -f -n "$tail_lines" "$latest_log"
    else
        tail -n "$tail_lines" "$latest_log"
    fi
}

# Export functions
export -f ffmpeg::init
export -f ffmpeg::test_installation
export -f ffmpeg::get_version
export -f ffmpeg::info
export -f ffmpeg::transcode
export -f ffmpeg::extract
export -f ffmpeg::logs