#!/bin/bash

ffmpeg_uninstall() {
    local force="${1:-false}"
    local remove_data="${2:-no}"
    
    # Get the directory of this lib file
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
    local FFMPEG_LIB_DIR="${APP_ROOT}/resources/ffmpeg/lib"
    source "${APP_ROOT}/scripts/lib/utils/format.sh"
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
    source "${FFMPEG_LIB_DIR}/../config/defaults.sh"
    
    # Export configuration
    ffmpeg::export_config
    
    if [[ "$force" != "--force" && "$force" != "true" ]]; then
        log::error "Uninstall requires --force flag to prevent accidental removal"
        log::info "Usage: ffmpeg_uninstall --force [remove_data]"
        return 1
    fi
    
    log::header "🗑️ Uninstalling FFmpeg Resource"
    
    if ! command -v ffmpeg &> /dev/null; then
        log::info "FFmpeg is not installed"
    else
        log::info "Removing FFmpeg package..."
        sudo apt-get remove -y ffmpeg &>/dev/null || {
            log::error "Failed to uninstall FFmpeg package"
            return 1
        }
        log::success "FFmpeg package removed"
    fi
    
    # Handle data removal
    if [[ "$remove_data" == "yes" ]]; then
        log::warn "Removing all FFmpeg data directories..."
        
        if [[ -d "${FFMPEG_DATA_DIR}" ]]; then
            # Create backup of important data
            local backup_dir="${FFMPEG_DATA_DIR}_backup_$(date +%Y%m%d_%H%M%S)"
            if [[ -d "${FFMPEG_OUTPUT_DIR}" ]] && [[ -n "$(ls -A "${FFMPEG_OUTPUT_DIR}" 2>/dev/null)" ]]; then
                log::info "Creating backup of output files: $backup_dir"
                mkdir -p "$backup_dir"
                cp -r "${FFMPEG_OUTPUT_DIR}" "$backup_dir/" 2>/dev/null || true
            fi
            
            # Remove all directories
            rm -rf "${FFMPEG_DATA_DIR}" || {
                log::error "Failed to remove data directory: ${FFMPEG_DATA_DIR}"
            }
            
            if [[ -d "$backup_dir" ]]; then
                log::success "Data removed, backup created at: $backup_dir"
            else
                log::success "Data directory removed: ${FFMPEG_DATA_DIR}"
            fi
        else
            log::info "No data directory found to remove"
        fi
    else
        log::info "Data directories preserved:"
        [[ -d "${FFMPEG_DATA_DIR}" ]] && log::info "  Data: ${FFMPEG_DATA_DIR}"
        [[ -d "${FFMPEG_OUTPUT_DIR}" ]] && log::info "  Output: ${FFMPEG_OUTPUT_DIR}"
        log::info "To remove data, use: ffmpeg_uninstall --force yes"
    fi
    
    # Clean up temporary files
    if [[ -d "${FFMPEG_TEMP_DIR}" ]]; then
        log::info "Cleaning up temporary files..."
        rm -rf "${FFMPEG_TEMP_DIR}" || true
    fi
    
    log::success "FFmpeg resource uninstallation completed"
    return 0
}
