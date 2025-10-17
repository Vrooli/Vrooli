#!/usr/bin/env bash
# Codex Install Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_INSTALL_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_INSTALL_DIR}/common.sh"

#######################################
# Install Codex service (API-based, no actual installation needed)
# Arguments:
#   --dry-run: Show what would be done without executing
# Returns:
#   0 on success, 1 on failure
#######################################
codex::install::execute() {
    local dry_run="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$dry_run" == "true" ]]; then
        log::info "Would install Codex (API-based service):"
        log::info "  - Verify API configuration"
        log::info "  - Create necessary directories"
        log::info "  - Set up status tracking"
        return 0
    fi
    
    log::info "Installing Codex service..."
    
    # Create necessary directories
    mkdir -p "${CODEX_SCRIPTS_DIR}" "${CODEX_OUTPUT_DIR}" "${CODEX_INJECTED_DIR}" 2>/dev/null || true
    
    # Check if API is configured
    if codex::is_configured; then
        log::success "Codex API configuration found"
        
        # Test API connection
        if codex::is_available; then
            codex::save_status "running"
            log::success "Codex service installed and running"
        else
            codex::save_status "stopped"
            log::warn "Codex installed but API not responding"
        fi
    else
        log::warn "Codex installed but API not configured"
        log::info "Please set OPENAI_API_KEY environment variable or configure Vault"
        codex::save_status "stopped"
    fi
    
    return 0
}

#######################################
# Uninstall Codex service
# Arguments:
#   --dry-run: Show what would be done without executing
# Returns:
#   0 on success, 1 on failure
#######################################
codex::install::uninstall() {
    local dry_run="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$dry_run" == "true" ]]; then
        log::info "Would uninstall Codex:"
        log::info "  - Stop service"
        log::info "  - Remove status file"
        log::info "  - Clean up temporary files"
        return 0
    fi
    
    log::info "Uninstalling Codex service..."
    
    # Stop service
    codex::save_status "stopped"
    
    # Remove status file
    if [[ -f "${CODEX_STATUS_FILE}" ]]; then
        rm -f "${CODEX_STATUS_FILE}"
    fi
    
    log::info "Codex service uninstalled (user data preserved)"
    log::info "To remove all data, manually delete: ${CODEX_SCRIPTS_DIR}, ${CODEX_OUTPUT_DIR}"
    
    return 0
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ "$1" == "install" ]]; then
        shift
        codex::install::execute "$@"
    elif [[ "$1" == "uninstall" ]]; then
        shift
        codex::install::uninstall "$@"
    fi
fi