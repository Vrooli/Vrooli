#!/usr/bin/env bash
################################################################################
# Twilio Test Runner - v2.0 Universal Contract Implementation
# 
# Main test orchestrator for all Twilio test phases
################################################################################

set -euo pipefail

# Get the directory of this script
TEST_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${TEST_DIR}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
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