#!/bin/bash

# SimPy Resource - List Functions
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SIMPY_LIST_LIB_DIR="${APP_ROOT}/resources/simpy/lib"

# Source common functions
source "$SIMPY_LIST_LIB_DIR/common.sh"

# List simulations or outputs
list::main() {
    local type="${1:-simulations}"
    
    case "$type" in
        simulations)
            log::header "📋 Available Simulations"
            if [[ -d "$SIMPY_SIMULATIONS_DIR" ]]; then
                find "$SIMPY_SIMULATIONS_DIR" -name "*.py" -type f -exec basename {} \; 2>/dev/null | sort
            else
                log::info "No simulations found"
            fi
            ;;
        outputs)
            log::header "📊 Simulation Outputs"
            if [[ -d "$SIMPY_OUTPUTS_DIR" ]]; then
                find "$SIMPY_OUTPUTS_DIR" -type f -exec basename {} \; 2>/dev/null | sort
            else
                log::info "No outputs found"
            fi
            ;;
        *)
            log::error "Unknown list type: $type"
            echo "Usage: $0 [simulations|outputs]"
            return 1
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    list::main "$@"
fi