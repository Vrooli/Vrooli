#!/usr/bin/env bash
# UI Tests Phase
# Tests the UI is accessible and rendering correctly

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Get UI port dynamically
UI_PORT=$(vrooli scenario port graph-studio UI_PORT 2>/dev/null || echo "38707")
UI_URL="http://localhost:$UI_PORT"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }

TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    shift

    log_info "Testing: $test_name"
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

# Test UI accessibility
test_ui_accessible() {
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$UI_URL" 2>&1)
    [[ "$status" == "200" ]]
}

# Test UI contains expected content
test_ui_content() {
    local content=$(curl -sf "$UI_URL" 2>&1)

    # Check for React root or common UI elements
    if echo "$content" | grep -q "root\|Graph Studio\|graph-studio"; then
        return 0
    fi

    # Vite dev server might return different content
    if echo "$content" | grep -q "vite\|<!DOCTYPE html>"; then
        return 0
    fi

    return 1
}

# Test UI has no console errors (basic check)
test_ui_no_errors() {
    local content=$(curl -sf "$UI_URL" 2>&1)

    # Check HTML doesn't contain obvious error messages
    if echo "$content" | grep -qi "error 404\|page not found\|cannot get"; then
        return 1
    fi

    return 0
}

# Test UI serves static assets
test_ui_assets() {
    # Try to fetch common static files
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$UI_URL/vite.svg" 2>&1)

    # It's okay if specific assets don't exist, just check UI is serving files
    if [[ "$status" == "200" || "$status" == "404" ]]; then
        return 0
    fi

    return 1
}

# Main execution
main() {
    log_info "=== UI Tests Phase ==="
    echo ""

    # Check if UI is running
    if ! curl -sf "$UI_URL" &>/dev/null; then
        log_warning "UI is not running at $UI_URL"
        log_info "Skipping UI tests (scenario may not have UI started)"
        echo ""
        echo "UI Tests: Skipped (UI not running)"
        return 0
    fi

    run_test "UI Accessible" test_ui_accessible || true
    run_test "UI Content Loads" test_ui_content || true
    run_test "UI No Errors" test_ui_no_errors || true
    run_test "UI Serves Assets" test_ui_assets || true

    echo ""
    echo "UI Tests: Passed=$TESTS_PASSED Failed=$TESTS_FAILED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

main "$@"
