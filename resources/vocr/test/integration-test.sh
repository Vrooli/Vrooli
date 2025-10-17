#\!/usr/bin/env bash
# VOCR Integration Tests

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TEST_DIR="${APP_ROOT}/resources/vocr/test"
VOCR_DIR="${APP_ROOT}/resources/vocr"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test health endpoint
test_health() {
    log::info "Testing health endpoint..."
    if curl -s -f http://localhost:9420/health >/dev/null 2>&1; then
        log::success "Health endpoint working"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "Health endpoint failed"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test OCR endpoint
test_ocr() {
    log::info "Testing OCR endpoint..."
    # Create a test image with text
    echo "TEST TEXT" > /tmp/test.txt
    convert -size 200x50 xc:white -font Helvetica -pointsize 20 \
            -draw "text 10,30 'Hello VOCR'" /tmp/test-ocr.png 2>/dev/null || {
        log::warning "ImageMagick not available, skipping OCR test"
        return 0
    }
    
    local response
    response=$(curl -s -X POST http://localhost:9420/ocr \
        -H "Content-Type: application/json" \
        -d "{\"image\": \"/tmp/test-ocr.png\"}" 2>/dev/null || echo "{}")
    
    if echo "$response" | grep -q "success"; then
        log::success "OCR endpoint working"
        ((TESTS_PASSED++))
        return 0
    else
        log::warning "OCR endpoint returned: $response"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main test runner
main() {
    log::header "Running VOCR Integration Tests"
    
    test_health || true
    test_ocr || true
    
    # Save results
    local test_status="failed"
    if [[ $TESTS_FAILED -eq 0 ]]; then
        test_status="passed"
    elif [[ $TESTS_PASSED -gt 0 ]]; then
        test_status="partial"
    fi
    
    cat > "${VROOLI_ROOT:-${HOME}/Vrooli}/data/vocr/test-results.json" << JSON
{
    "status": "$test_status",
    "passed": $TESTS_PASSED,
    "failed": $TESTS_FAILED,
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
JSON
    
    echo ""
    log::info "Test Results:"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "All tests passed\!"
        return 0
    else
        log::warning "Some tests failed"
        return 1
    fi
}

main "$@"
