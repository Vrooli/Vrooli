#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CLINE_STOP_DIR="${APP_ROOT}/resources/cline/lib"

# Source common functions
source "$CLINE_STOP_DIR/common.sh"

# Stop Cline (Note: Cline is a VS Code extension, so this is mostly informational)
cline::stop() {
    log::info "Cline is a VS Code extension and doesn't run as a standalone service"
    log::info "To disable Cline, disable the extension in VS Code"
    
    # Ensure directory exists before writing status
    mkdir -p "$CLINE_CONFIG_DIR"
    
    # Mark as stopped in status
    echo "stopped" > "$CLINE_CONFIG_DIR/.status"
    
    return 0
}

# Main - only run if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cline::stop "$@"
fi