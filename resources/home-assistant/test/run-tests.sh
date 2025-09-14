#!/usr/bin/env bash
################################################################################
# Home Assistant Test Runner
# Executes all test phases for Home Assistant resource
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test configuration
TEST_PHASES_DIR="$SCRIPT_DIR/phases"
PHASE="${1:-all}"

# Available test phases
declare -A TEST_PHASES=(
    ["smoke"]="Quick health validation"
    ["integration"]="End-to-end functionality"
    ["unit"]="Library function validation"
    ["voice"]="Voice control configuration tests"
    ["all"]="Run all test phases"
)

#######################################
# Show usage information
#######################################
show_usage() {
    echo "Usage: $0 [PHASE]"
    echo
    echo "Available phases:"
    for phase in "${!TEST_PHASES[@]}"; do
        printf "  %-15s %s\n" "$phase" "${TEST_PHASES[$phase]}"
    done
    echo
    echo "Examples:"
    echo "  $0 smoke        # Run quick health checks"
    echo "  $0 integration  # Run integration tests"
    echo "  $0 all          # Run all tests"
}

#######################################
# Run specific test phase
#######################################
run_phase() {
    local phase="$1"
    local phase_script="$TEST_PHASES_DIR/test-${phase}.sh"
    
    if [[ "$phase" == "all" ]]; then
        # Run all phases in sequence
        local failed=0
        for test_phase in smoke integration unit voice; do
            if ! run_phase "$test_phase"; then
                ((failed++))
            fi
        done
        
        if [[ $failed -eq 0 ]]; then
            log::success "All test phases completed successfully!"
            return 0
        else
            log::error "$failed test phase(s) failed"
            return 1
        fi
    elif [[ -f "$phase_script" ]]; then
        log::header "Running $phase tests"
        bash "$phase_script"
        return $?
    else
        log::error "Test phase not found: $phase"
        return 1
    fi
}

#######################################
# Main execution
#######################################
main() {
    # Validate phase argument
    if [[ ! -v "TEST_PHASES[$PHASE]" ]]; then
        log::error "Invalid test phase: $PHASE"
        show_usage
        exit 1
    fi
    
    # Ensure Home Assistant is running for tests
    if [[ "$PHASE" != "unit" ]]; then
        if ! docker ps --format "{{.Names}}" | grep -q "^home-assistant$"; then
            log::warning "Home Assistant is not running. Starting it for tests..."
            "${RESOURCE_DIR}/cli.sh" manage start --wait
            
            if [ $? -ne 0 ]; then
                log::error "Failed to start Home Assistant for testing"
                exit 1
            fi
        fi
    fi
    
    # Run the requested phase
    run_phase "$PHASE"
    exit $?
}

# Show help if requested
if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
    show_usage
    exit 0
fi

# Run main function
main