#!/bin/bash

# SimPy Resource - Injection Functions
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SIMPY_INJECT_LIB_DIR="${APP_ROOT}/resources/simpy/lib"

# Source common functions
source "$SIMPY_INJECT_LIB_DIR/common.sh"

# Inject a simulation file
inject::main() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "No simulation file specified"
        echo "Usage: $0 <simulation.py>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Validate it's a Python file
    if [[ ! "$file" =~ \.py$ ]]; then
        log::warning "File does not have .py extension: $file"
    fi
    
    # Create simulations directory if needed
    mkdir -p "$SIMPY_SIMULATIONS_DIR"
    
    # Copy file to simulations directory
    local filename=$(basename "$file")
    local dest="$SIMPY_SIMULATIONS_DIR/$filename"
    
    log::info "Injecting simulation: $filename"
    cp "$file" "$dest"
    
    # Validate the simulation can be parsed
    if "$SIMPY_PYTHON" -m py_compile "$dest" &>/dev/null; then
        log::success "✅ Simulation injected successfully: $filename"
        log::info "Location: $dest"
        return 0
    else
        log::error "❌ Simulation has syntax errors"
        rm -f "$dest"
        return 1
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    inject::main "$@"
fi