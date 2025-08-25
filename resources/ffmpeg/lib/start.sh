#!/bin/bash

ffmpeg_start() {
    # Get the directory of this lib file
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
    local FFMPEG_LIB_DIR="${APP_ROOT}/resources/ffmpeg/lib"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/format.sh"
    source "${FFMPEG_LIB_DIR}/../../../../lib/utils/log.sh"
    source "${FFMPEG_LIB_DIR}/../config/defaults.sh"
    source "${FFMPEG_LIB_DIR}/hardware.sh"
    
    # Export configuration
    ffmpeg::export_config
    
    log::header "🎬 Starting FFmpeg Resource"
    
    # FFmpeg is a CLI tool, not a service
    # "Starting" means ensuring it's installed, configured, and ready
    
    if ! command -v ffmpeg &> /dev/null; then
        log::error "FFmpeg is not installed"
        log::info "Run: vrooli resource ffmpeg install"
        return 1
    fi
    
    # Create necessary directories
    log::info "Creating directory structure..."
    mkdir -p "${FFMPEG_DATA_DIR}" "${FFMPEG_OUTPUT_DIR}" "${FFMPEG_TEMP_DIR}" "${FFMPEG_LOGS_DIR}"
    
    # Set proper permissions
    chmod 755 "${FFMPEG_DATA_DIR}" "${FFMPEG_OUTPUT_DIR}" "${FFMPEG_TEMP_DIR}" "${FFMPEG_LOGS_DIR}"
    
    log::success "Directories created:"
    log::info "  Data: ${FFMPEG_DATA_DIR}"
    log::info "  Output: ${FFMPEG_OUTPUT_DIR}"
    log::info "  Temp: ${FFMPEG_TEMP_DIR}"
    log::info "  Logs: ${FFMPEG_LOGS_DIR}"
    
    # Configure hardware acceleration
    log::info "Configuring hardware acceleration..."
    ffmpeg::hardware::configure
    
    # Clean up old temporary files (older than 1 day)
    log::info "Cleaning up temporary files..."
    find "${FFMPEG_TEMP_DIR}" -type f -mtime +1 -delete 2>/dev/null || true
    
    # Log current configuration
    local log_file="${FFMPEG_LOGS_DIR}/startup-$(date +%Y%m%d-%H%M%S).log"
    {
        echo "FFmpeg Resource Started: $(date)"
        echo "Version: $(ffmpeg -version 2>&1 | head -n1)"
        echo "Data Directory: ${FFMPEG_DATA_DIR}"
        echo "Hardware Acceleration: ${FFMPEG_HW_ACCEL:-none}"
        echo "Default Codec: ${FFMPEG_DEFAULT_VIDEO_CODEC}"
        echo "Default Quality: ${FFMPEG_DEFAULT_QUALITY}"
        echo "Default Preset: ${FFMPEG_DEFAULT_PRESET}"
    } > "$log_file"
    
    log::success "FFmpeg is ready for use"
    log::info "Configuration logged to: $log_file"
    
    return 0
}
