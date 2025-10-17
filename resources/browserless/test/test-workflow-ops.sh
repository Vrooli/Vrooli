#!/usr/bin/env bash
# Test script for enhanced workflow operations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${BROWSERLESS_DIR}/lib/browser-ops.sh"
source "${BROWSERLESS_DIR}/lib/workflow-ops.sh"

log::info "Testing enhanced workflow operations..."

# Test directory for outputs
TEST_OUTPUT_DIR="/tmp/browserless-test-$(date +%s)"
mkdir -p "$TEST_OUTPUT_DIR"

# Test 1: Basic navigation and screenshot
test_basic_operations() {
    log::info "Test 1: Basic navigation and screenshot"
    
    local session_id="test-session-$(date +%s)"
    
    # Navigate to Google
    if browser::navigate "https://www.google.com" "$session_id"; then
        log::success "Navigation successful"
    else
        log::error "Navigation failed"
        return 1
    fi
    
    # Take full page screenshot
    if browser::screenshot_full_page "$TEST_OUTPUT_DIR/google-full.png" "$session_id"; then
        log::success "Full page screenshot saved"
    else
        log::error "Full page screenshot failed"
        return 1
    fi
    
    return 0
}

# Test 2: Element extraction
test_element_extraction() {
    log::info "Test 2: Element extraction"
    
    local session_id="test-extract-$(date +%s)"
    
    # Navigate to a page
    browser::navigate "https://www.example.com" "$session_id"
    
    # Extract text from h1
    local title
    if title=$(browser::extract_text "h1" "$session_id"); then
        log::success "Extracted title: $title"
    else
        log::error "Failed to extract title"
        return 1
    fi
    
    # Extract page content
    local content
    if content=$(browser::extract_text "body" "$session_id"); then
        log::success "Extracted body text (${#content} chars)"
        echo "$content" > "$TEST_OUTPUT_DIR/body-text.txt"
    else
        log::error "Failed to extract body text"
        return 1
    fi
    
    return 0
}

# Test 3: Form interaction
test_form_interaction() {
    log::info "Test 3: Form interaction with delays"
    
    local session_id="test-form-$(date +%s)"
    
    # Navigate to Google
    browser::navigate "https://www.google.com" "$session_id"
    
    # Type with delay into search box
    if browser::type_with_delay "input[name='q'], textarea[name='q']" "Vrooli AI platform" 50 "$session_id"; then
        log::success "Typed search query with delay"
    else
        log::error "Failed to type search query"
        return 1
    fi
    
    # Take screenshot of search box
    if browser::screenshot_element "input[name='q'], textarea[name='q']" "$TEST_OUTPUT_DIR/search-box.png" "$session_id"; then
        log::success "Element screenshot saved"
    else
        log::warning "Element screenshot failed (might be textarea)"
    fi
    
    return 0
}

# Test 4: Wait for URL change
test_url_change() {
    log::info "Test 4: URL change detection"
    
    local session_id="test-url-$(date +%s)"
    
    # Navigate to Google
    browser::navigate "https://www.google.com" "$session_id"
    
    # Click on "About" link (if exists) and wait for navigation
    if browser::element_exists "a[href*='about']" "$session_id"; then
        if browser::click_and_wait "a[href*='about']" "$session_id"; then
            log::success "Clicked link and waited for navigation"
        else
            log::warning "Click and wait failed"
        fi
    else
        log::info "About link not found, skipping URL change test"
    fi
    
    return 0
}

# Run all tests
main() {
    local failed=0
    
    log::info "Starting enhanced workflow operations tests..."
    log::info "Output directory: $TEST_OUTPUT_DIR"
    
    # Run tests
    if ! test_basic_operations; then
        ((failed++))
    fi
    
    if ! test_element_extraction; then
        ((failed++))
    fi
    
    if ! test_form_interaction; then
        ((failed++))
    fi
    
    if ! test_url_change; then
        ((failed++))
    fi
    
    # Summary
    log::info "═══════════════════════════════════════"
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed!"
        log::info "Test outputs saved to: $TEST_OUTPUT_DIR"
        ls -la "$TEST_OUTPUT_DIR"
    else
        log::error "$failed test(s) failed"
        exit 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi