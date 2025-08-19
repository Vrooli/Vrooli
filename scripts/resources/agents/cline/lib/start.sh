#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
CLINE_START_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$CLINE_START_DIR/common.sh"

# Start Cline (mainly ensures VS Code is configured)
cline::start() {
    log::info "Starting Cline configuration..."
    
    # Ensure config directory exists
    mkdir -p "$CLINE_CONFIG_DIR"
    
    # Check if VS Code is available
    if ! cline::check_vscode; then
        log::warn "VS Code is not installed. Setting up configuration for when it becomes available."
        # Continue with configuration setup
    elif ! cline::is_installed; then
        log::warn "Cline is not installed. Installing..."
        "$CLINE_START_DIR/install.sh"
        return $?
    fi
    
    # Ensure configured
    if ! cline::is_configured; then
        log::warn "Cline is not configured properly"
        
        # Try to configure
        local provider=$(cline::get_provider)
        if [[ "$provider" == "ollama" ]]; then
            # Check if Ollama is running
            if ! curl -s http://localhost:11434/api/version >/dev/null 2>&1; then
                log::error "Ollama is not running. Please start Ollama first."
                return 1
            fi
        fi
        
        log::info "Cline configured to use: $provider"
    fi
    
    # Mark as running in status file
    echo "running" > "$CLINE_CONFIG_DIR/.status"
    
    if cline::check_vscode; then
        log::success "Cline is ready to use in VS Code"
        log::info "Open VS Code and use Cmd/Ctrl+Shift+P -> 'Cline: Open Chat' to start"
    else
        log::success "Cline configuration prepared. Install VS Code to use Cline."
    fi
    
    return 0
}

# Main - only run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cline::start "$@"
fi