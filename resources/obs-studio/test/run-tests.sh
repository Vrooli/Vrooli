#!/usr/bin/env bash
################################################################################
# OBS Studio Test Runner - v2.0 Contract Compliant
#
# Main test runner for OBS Studio resource validation
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test phase to run
PHASE="${1:-all}"

# Main test execution
main() {
    case "$PHASE" in
        smoke)
            exec "${SCRIPT_DIR}/phases/test-smoke.sh"
            ;;
        integration)
            exec "${SCRIPT_DIR}/phases/test-integration.sh"
            ;;
        unit)
            exec "${SCRIPT_DIR}/phases/test-unit.sh"
            ;;
        all)
            local failed=0
            
            log::header "Running All OBS Studio Tests"
            
            # Run smoke tests
            if "${SCRIPT_DIR}/phases/test-smoke.sh"; then
                log::success "✅ Smoke tests passed"
            else
                log::error "❌ Smoke tests failed"
                ((failed++))
            fi
            
            # Run unit tests
            if "${SCRIPT_DIR}/phases/test-unit.sh"; then
                log::success "✅ Unit tests passed"
            else
                log::error "❌ Unit tests failed"
                ((failed++))
            fi
            
            # Run integration tests
            if "${SCRIPT_DIR}/phases/test-integration.sh"; then
                log::success "✅ Integration tests passed"
            else
                log::error "❌ Integration tests failed"
                ((failed++))
            fi
            
            if [[ $failed -eq 0 ]]; then
                log::success "All test phases passed"
                exit 0
            else
                log::error "$failed test phases failed"
                exit 1
            fi
            ;;
        *)
            log::error "Unknown test phase: $PHASE"
            log::info "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

main "$@"