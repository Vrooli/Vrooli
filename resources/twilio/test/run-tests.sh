#!/usr/bin/env bash
################################################################################
# Twilio Test Runner - v2.0 Universal Contract Implementation
# 
# Main test orchestrator for all Twilio test phases
################################################################################

set -euo pipefail

# Get the directory of this script
TEST_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
TWILIO_DIR="${RESOURCE_DIR}"

# Source required libraries
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
source "${TWILIO_DIR}/lib/test.sh"

# Parse command line arguments
TEST_PHASE="${1:-all}"

# Main test execution
main() {
    case "$TEST_PHASE" in
        smoke)
            twilio::test::smoke
            ;;
        integration)
            twilio::test::integration
            ;;
        unit)
            twilio::test::unit
            ;;
        all)
            twilio::test::all
            ;;
        *)
            log::error "Unknown test phase: $TEST_PHASE"
            log::info "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"