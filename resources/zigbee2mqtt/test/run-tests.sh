#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT Test Runner
# 
# Main test orchestrator for Zigbee2MQTT resource
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test framework
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/lib/test/framework.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test configuration
TEST_SUITE="${1:-all}"
TEST_TIMEOUT="${TEST_TIMEOUT:-600}"

################################################################################
# Test Execution
################################################################################

main() {
    log::info "Starting Zigbee2MQTT test suite: $TEST_SUITE"
    
    # Set up test environment
    export TEST_MODE=true
    export ZIGBEE2MQTT_PORT="${ZIGBEE2MQTT_PORT:-8090}"
    
    # Run requested test suite
    case "$TEST_SUITE" in
        smoke)
            timeout "$TEST_TIMEOUT" bash "${SCRIPT_DIR}/phases/test-smoke.sh"
            ;;
        integration)
            timeout "$TEST_TIMEOUT" bash "${SCRIPT_DIR}/phases/test-integration.sh"
            ;;
        unit)
            timeout "$TEST_TIMEOUT" bash "${SCRIPT_DIR}/phases/test-unit.sh"
            ;;
        all)
            local failed=0
            
            # Run all test phases
            timeout "$TEST_TIMEOUT" bash "${SCRIPT_DIR}/phases/test-smoke.sh" || ((failed++))
            timeout "$TEST_TIMEOUT" bash "${SCRIPT_DIR}/phases/test-unit.sh" || ((failed++))
            timeout "$TEST_TIMEOUT" bash "${SCRIPT_DIR}/phases/test-integration.sh" || ((failed++))
            
            if [[ $failed -gt 0 ]]; then
                log::error "$failed test phase(s) failed"
                exit 1
            fi
            
            log::success "All test phases passed"
            ;;
        *)
            log::error "Unknown test suite: $TEST_SUITE"
            echo "Valid options: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"