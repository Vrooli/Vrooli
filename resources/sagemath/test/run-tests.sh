#!/usr/bin/env bash
################################################################################
# SageMath Test Runner - v2.0 Universal Contract Compliant
#
# Main test orchestrator for the SageMath resource
# Delegates to specific test phases as required by the contract
################################################################################

set -euo pipefail

# Setup paths
TEST_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
APP_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || {
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
}

# Test phase selection
run_test_phase() {
    local phase="${1:-all}"
    local phase_script="${TEST_DIR}/phases/test-${phase}.sh"
    
    case "$phase" in
        all)
            log::info "Running all test phases..."
            local all_passed=true
            
            for test_phase in smoke unit integration; do
                if [[ -f "${TEST_DIR}/phases/test-${test_phase}.sh" ]]; then
                    log::info "Running ${test_phase} tests..."
                    if ! bash "${TEST_DIR}/phases/test-${test_phase}.sh"; then
                        log::error "${test_phase} tests failed"
                        all_passed=false
                    fi
                fi
            done
            
            if [[ "$all_passed" == "true" ]]; then
                log::success "All tests passed"
                return 0
            else
                log::error "Some tests failed"
                return 1
            fi
            ;;
            
        smoke|unit|integration|performance)
            if [[ -f "$phase_script" ]]; then
                log::info "Running ${phase} tests..."
                bash "$phase_script"
            else
                log::error "Test phase '${phase}' not found: ${phase_script}"
                return 2
            fi
            ;;
            
        *)
            log::error "Unknown test phase: ${phase}"
            echo "Usage: $0 [all|smoke|unit|integration|performance]"
            return 1
            ;;
    esac
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    phase="${1:-all}"
    run_test_phase "$phase"
fi