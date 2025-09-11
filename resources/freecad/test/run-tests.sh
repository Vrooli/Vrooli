#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Main Test Runner
# 
# Orchestrates all test phases for FreeCAD resource
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source common utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Test phase to run
PHASE="${1:-all}"

# Run specified test phase
case "$PHASE" in
    smoke)
        log::info "Running smoke tests..."
        bash "${SCRIPT_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        log::info "Running integration tests..."
        bash "${SCRIPT_DIR}/phases/test-integration.sh"
        ;;
    unit)
        log::info "Running unit tests..."
        bash "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    all)
        log::info "Running all test phases..."
        failed=0
        
        bash "${SCRIPT_DIR}/phases/test-smoke.sh" || ((failed++))
        bash "${SCRIPT_DIR}/phases/test-integration.sh" || ((failed++))
        bash "${SCRIPT_DIR}/phases/test-unit.sh" || ((failed++))
        
        if [[ $failed -eq 0 ]]; then
            log::info "All tests passed"
            exit 0
        else
            log::error "$failed test phase(s) failed"
            exit 1
        fi
        ;;
    *)
        log::error "Unknown test phase: $PHASE"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac