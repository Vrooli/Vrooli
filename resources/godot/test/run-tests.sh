#!/usr/bin/env bash
# Godot Engine Resource - Test Runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Test phase to run
readonly PHASE="${1:-all}"

# Test result tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

run_test_phase() {
    local phase="$1"
    local script="${SCRIPT_DIR}/phases/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        log_warning "Test phase '$phase' not found, skipping"
        return 2
    fi
    
    log_info "Running $phase tests..."
    
    if bash "$script"; then
        ((TESTS_PASSED++))
        log_info "✅ $phase tests passed"
        return 0
    else
        ((TESTS_FAILED++))
        log_error "❌ $phase tests failed"
        return 1
    fi
}

# Main test execution
main() {
    log_info "🧪 Godot Resource Test Suite"
    log_info "================================"
    
    case "$PHASE" in
        all)
            log_info "Running all test phases..."
            run_test_phase "smoke" || true
            run_test_phase "unit" || true
            run_test_phase "integration" || true
            ;;
        smoke|unit|integration)
            run_test_phase "$PHASE"
            ;;
        *)
            log_error "Unknown test phase: $PHASE"
            echo "Usage: $0 [all|smoke|unit|integration]"
            exit 1
            ;;
    esac
    
    # Summary
    log_info "================================"
    log_info "Test Summary:"
    log_info "  Passed: $TESTS_PASSED"
    log_info "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log_error "Tests failed!"
        exit 1
    else
        log_info "All tests passed!"
        exit 0
    fi
}

# Execute main function
main "$@"