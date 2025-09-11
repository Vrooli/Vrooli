#!/usr/bin/env bash
################################################################################
# Gemini Resource Test Runner
# 
# Central test runner that delegates to test phases
################################################################################

set -euo pipefail

# Determine paths
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source resource configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Parse arguments
TEST_PHASE="${1:-all}"

# Run appropriate test phase
case "$TEST_PHASE" in
    smoke)
        log::info "Running smoke tests..."
        exec "${TEST_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        log::info "Running integration tests..."
        exec "${TEST_DIR}/phases/test-integration.sh"
        ;;
    unit)
        log::info "Running unit tests..."
        exec "${TEST_DIR}/phases/test-unit.sh"
        ;;
    cache)
        log::info "Running cache tests..."
        exec "${TEST_DIR}/phases/test-cache.sh"
        ;;
    all)
        log::info "Running all test phases..."
        tests_passed=0
        total_tests=4
        
        # Run each phase
        if "${TEST_DIR}/phases/test-unit.sh"; then
            ((tests_passed++))
        fi
        
        if "${TEST_DIR}/phases/test-smoke.sh"; then
            ((tests_passed++))
        fi
        
        if "${TEST_DIR}/phases/test-integration.sh"; then
            ((tests_passed++))
        fi
        
        if "${TEST_DIR}/phases/test-cache.sh"; then
            ((tests_passed++))
        fi
        
        # Report results
        if [[ $tests_passed -eq $total_tests ]]; then
            log::success "✓ All test phases passed ($tests_passed/$total_tests)"
            exit 0
        else
            log::error "✗ Some test phases failed ($tests_passed/$total_tests)"
            exit 1
        fi
        ;;
    *)
        log::error "Unknown test phase: $TEST_PHASE"
        echo "Usage: $0 [smoke|integration|unit|cache|all]"
        exit 1
        ;;
esac