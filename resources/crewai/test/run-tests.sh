#!/usr/bin/env bash
################################################################################
# CrewAI Test Runner - v2.0 Universal Contract Compliant
#
# Main test orchestrator for CrewAI resource validation
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CREWAI_ROOT="${APP_ROOT}/resources/crewai"
TEST_DIR="${CREWAI_ROOT}/test"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${CREWAI_ROOT}/lib/test.sh"

################################################################################
# Main Test Runner
################################################################################

main() {
    local test_phase="${1:-all}"
    
    log::header "CrewAI Test Runner"
    log::info "Test Phase: $test_phase"
    
    # Ensure service is running for tests
    if ! "${CREWAI_ROOT}/cli.sh" status | grep -q "Running: Yes"; then
        log::info "Starting CrewAI for testing..."
        "${CREWAI_ROOT}/cli.sh" manage start --wait
        sleep 2
    fi
    
    # Run requested test phase
    case "$test_phase" in
        smoke)
            crewai::test::smoke
            ;;
        integration)
            crewai::test::integration
            ;;
        unit)
            crewai::test::unit
            ;;
        all)
            crewai::test::all
            ;;
        *)
            log::error "Unknown test phase: $test_phase"
            log::info "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "Test phase '$test_phase' completed successfully!"
    else
        log::error "Test phase '$test_phase' failed with exit code $exit_code"
    fi
    
    return $exit_code
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi