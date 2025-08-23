#!/bin/bash

# SimPy Resource - Run Simulation Functions
set -euo pipefail

# Get the script directory
SIMPY_RUN_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$SIMPY_RUN_LIB_DIR/common.sh"

# Run a simulation
run::main() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "No simulation file specified"
        echo "Usage: $0 <simulation.py>"
        return 1
    fi
    
    # Check if SimPy is installed
    if ! simpy::is_installed; then
        log::error "SimPy is not installed. Run: vrooli resource simpy install"
        return 1
    fi
    
    # Find the simulation file
    local sim_file=""
    if [[ -f "$file" ]]; then
        sim_file="$file"
    elif [[ -f "$SIMPY_SIMULATIONS_DIR/$file" ]]; then
        sim_file="$SIMPY_SIMULATIONS_DIR/$file"
    elif [[ -f "$SIMPY_EXAMPLES_DIR/$file" ]]; then
        sim_file="$SIMPY_EXAMPLES_DIR/$file"
    else
        log::error "Simulation file not found: $file"
        log::info "Available simulations:"
        ls -1 "$SIMPY_SIMULATIONS_DIR" 2>/dev/null | grep "\.py$" || echo "  No simulations found"
        return 1
    fi
    
    # Create output directory
    mkdir -p "$SIMPY_OUTPUTS_DIR"
    
    # Run the simulation
    log::info "Running simulation: $(basename "$sim_file")"
    
    # Prepare volumes for Docker
    local sim_dir=$(dirname "$sim_file")
    local sim_name=$(basename "$sim_file")
    
    # Run with Docker
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="$SIMPY_OUTPUTS_DIR/$(basename "$sim_file" .py)_${timestamp}.log"
    
    if docker run --rm \
        -v "$sim_dir:/simulations:ro" \
        -v "$SIMPY_OUTPUTS_DIR:/outputs" \
        -v "$SIMPY_LOGS_DIR:/logs" \
        "$SIMPY_IMAGE_NAME" \
        python "/simulations/$sim_name" 2>&1 | tee "$output_file"; then
        log::success "✅ Simulation completed successfully"
        log::info "Output saved to: $output_file"
        return 0
    else
        log::error "❌ Simulation failed"
        return 1
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run::main "$@"
fi