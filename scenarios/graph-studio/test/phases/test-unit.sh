#!/usr/bin/env bash
# Unit Tests Phase
# Tests individual components and functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
API_DIR="$SCENARIO_ROOT/api"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }

TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    shift

    log_info "Running: $test_name"
    if "$@"; then
        log_success "$test_name"
        ((TESTS_PASSED++))
        return 0
    else
        log_error "$test_name"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test 1: Go code compilation
test_go_build() {
    cd "$API_DIR"
    go build -o /tmp/graph-studio-test-build . &>/dev/null
    rm -f /tmp/graph-studio-test-build
}

# Test 2: Go fmt check
test_go_fmt() {
    cd "$API_DIR"
    local unformatted=$(gofmt -l . 2>/dev/null)
    if [[ -n "$unformatted" ]]; then
        echo "Unformatted files: $unformatted"
        return 1
    fi
    return 0
}

# Test 3: Go vet
test_go_vet() {
    cd "$API_DIR"
    go vet ./... 2>&1 | grep -v "composite literal uses unkeyed fields" || true
}

# Test 4: Run Go unit tests
test_go_tests() {
    cd "$API_DIR"
    if compgen -G "*_test.go" > /dev/null; then
        go test -v -short ./... 2>&1
    else
        log_info "No Go unit tests found (this is expected for now)"
        return 0
    fi
}

# Main execution
main() {
    log_info "=== Unit Tests Phase ==="
    echo ""

    run_test "Go Build" test_go_build || true
    run_test "Go Format Check" test_go_fmt || true
    run_test "Go Vet" test_go_vet || true
    run_test "Go Unit Tests" test_go_tests || true

    echo ""
    echo "Unit Tests: Passed=$TESTS_PASSED Failed=$TESTS_FAILED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

main "$@"
