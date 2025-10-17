#!/bin/bash

# Mail-in-a-Box Test Runner
# Follows v2.0 contract test structure

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
PHASES_DIR="$SCRIPT_DIR/phases"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test phase to run (default: all)
PHASE="${1:-all}"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Run a test phase
run_phase() {
    local phase_name="$1"
    local phase_script="$PHASES_DIR/test-${phase_name}.sh"
    
    if [[ ! -f "$phase_script" ]]; then
        log_error "Test phase '$phase_name' not found at $phase_script"
        return 1
    fi
    
    log_info "Running $phase_name tests..."
    
    if bash "$phase_script"; then
        log_info "✓ $phase_name tests passed"
        return 0
    else
        log_error "✗ $phase_name tests failed"
        return 1
    fi
}

# Main test execution
main() {
    log_info "Mail-in-a-Box Resource Test Suite"
    log_info "Test phase: $PHASE"
    
    local exit_code=0
    
    case "$PHASE" in
        smoke)
            run_phase "smoke" || exit_code=$?
            ;;
        unit)
            run_phase "unit" || exit_code=$?
            ;;
        integration)
            run_phase "integration" || exit_code=$?
            ;;
        all)
            # Run all phases in order
            for phase in smoke unit integration; do
                if ! run_phase "$phase"; then
                    exit_code=1
                    log_warn "Continuing with remaining tests despite failure..."
                fi
            done
            ;;
        *)
            log_error "Unknown test phase: $PHASE"
            log_info "Available phases: smoke, unit, integration, all"
            exit 1
            ;;
    esac
    
    if [[ $exit_code -eq 0 ]]; then
        log_info "✓ All requested tests passed"
    else
        log_error "✗ Some tests failed"
    fi
    
    exit $exit_code
}

# Execute main function
main