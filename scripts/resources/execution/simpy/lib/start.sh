#!/bin/bash

# SimPy Resource - Start Functions
set -euo pipefail

# Get the script directory
SIMPY_START_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$SIMPY_START_LIB_DIR/common.sh"

# Start SimPy (no-op for library)
start::main() {
    log::info "SimPy is a library resource - no service to start"
    log::info "SimPy is ready to run simulations"
    
    # Check if installed
    if simpy::is_installed; then
        log::success "✅ SimPy is installed and ready"
        return 0
    else
        log::warning "⚠️ SimPy is not installed"
        log::info "Run: vrooli resource simpy install"
        return 1
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    start::main
fi