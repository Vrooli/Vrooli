#!/bin/bash
# GGWave Test Runner
# Main test orchestrator following v2.0 contract

set -euo pipefail

# Get the directory of this script
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test phase to run
PHASE="${1:-all}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Log functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Main test execution
main() {
    log_info "Starting GGWave test suite - Phase: ${PHASE}"
    
    case "$PHASE" in
        smoke)
            log_info "Running smoke tests..."
            "${TEST_DIR}/phases/test-smoke.sh"
            ;;
        integration)
            log_info "Running integration tests..."
            "${TEST_DIR}/phases/test-integration.sh"
            ;;
        unit)
            log_info "Running unit tests..."
            "${TEST_DIR}/phases/test-unit.sh"
            ;;
        all)
            log_info "Running all test phases..."
            local errors=0
            
            # Run smoke tests
            log_info "Phase 1/3: Smoke tests"
            if ! "${TEST_DIR}/phases/test-smoke.sh"; then
                ((errors++))
                log_error "Smoke tests failed"
            fi
            
            # Run unit tests
            log_info "Phase 2/3: Unit tests"
            if ! "${TEST_DIR}/phases/test-unit.sh"; then
                ((errors++))
                log_error "Unit tests failed"
            fi
            
            # Run integration tests
            log_info "Phase 3/3: Integration tests"
            if ! "${TEST_DIR}/phases/test-integration.sh"; then
                ((errors++))
                log_error "Integration tests failed"
            fi
            
            # Summary
            echo ""
            if [[ $errors -eq 0 ]]; then
                log_info "All test phases completed successfully ✓"
                exit 0
            else
                log_error "$errors test phase(s) failed ✗"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown test phase: ${PHASE}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
}

# Run main
main