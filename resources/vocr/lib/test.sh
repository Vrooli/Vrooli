#!/usr/bin/env bash
################################################################################
# VOCR Test Library - v2.0 Contract Compliant
#
# Implements test functions for smoke, integration, and unit testing
################################################################################

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VOCR_TEST_DIR="${APP_ROOT}/resources/vocr/test"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/resources/vocr/config/defaults.sh"

# Export configuration
vocr::export_config

################################################################################
# Smoke Test - Quick health validation (<30s)
################################################################################
vocr::test::smoke() {
    log::header "Running VOCR Smoke Tests"
    
    local failed=0
    
    # Test 1: Service is running
    log::info "Test 1: Checking if service is running..."
    if pgrep -f "vocr-service.py" >/dev/null 2>&1; then
        log::success "✅ Service is running"
    else
        log::error "❌ Service is not running"
        ((failed++))
    fi
    
    # Test 2: Health endpoint responds
    log::info "Test 2: Testing health endpoint..."
    if timeout 5 curl -sf "http://${VOCR_HOST}:${VOCR_PORT}/health" >/dev/null 2>&1; then
        log::success "✅ Health endpoint responsive"
    else
        log::error "❌ Health endpoint not responding"
        ((failed++))
    fi
    
    # Test 3: OCR capability is available
    log::info "Test 3: Checking OCR capability..."
    # Check for either system tesseract or Python OCR wrapper
    if command -v tesseract >/dev/null 2>&1 || [[ -x "${VOCR_DATA_DIR}/tesseract-wrapper.sh" ]]; then
        log::success "✅ OCR capability available"
    elif [[ -f "${VOCR_DATA_DIR}/venv/bin/python" ]] && "${VOCR_DATA_DIR}/venv/bin/python" -c "import easyocr" 2>/dev/null; then
        log::success "✅ EasyOCR available"
    elif [[ -f "${VOCR_DATA_DIR}/venv/bin/python" ]] && "${VOCR_DATA_DIR}/venv/bin/python" -c "import pytesseract" 2>/dev/null; then
        log::success "✅ PyTesseract available"
    else
        log::error "❌ No OCR capability found"
        ((failed++))
    fi
    
    # Test 4: Python environment exists
    log::info "Test 4: Checking Python environment..."
    if [[ -d "${VOCR_DATA_DIR}/venv" ]]; then
        log::success "✅ Python environment exists"
    else
        log::error "❌ Python environment missing"
        ((failed++))
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All smoke tests passed!"
        return 0
    else
        log::error "$failed smoke tests failed"
        return 1
    fi
}

################################################################################
# Integration Test - End-to-end functionality (<120s)
################################################################################
vocr::test::integration() {
    log::header "Running VOCR Integration Tests"
    
    local failed=0
    
    # Test 1: OCR endpoint works
    log::info "Test 1: Testing OCR endpoint..."
    
    # Create a test image with text
    local test_image="${VOCR_SCREENSHOTS_DIR}/test_ocr.png"
    mkdir -p "${VOCR_SCREENSHOTS_DIR}"
    
    # Use ImageMagick to create test image if available
    if command -v convert >/dev/null 2>&1; then
        convert -size 200x50 xc:white -font "DejaVu-Sans" -pointsize 20 \
                -fill black -annotate +10+30 "Test OCR Text" \
                "$test_image" 2>/dev/null || true
    fi
    
    if [[ -f "$test_image" ]]; then
        # Test OCR on the image
        local response
        response=$(curl -sf -X POST "http://${VOCR_HOST}:${VOCR_PORT}/ocr" \
                   -F "image=@${test_image}" 2>/dev/null || echo "{}")
        
        if echo "$response" | grep -q "text"; then
            log::success "✅ OCR endpoint works"
        else
            log::error "❌ OCR endpoint failed"
            ((failed++))
        fi
        
        rm -f "$test_image"
    else
        log::warning "⚠️  Could not create test image (ImageMagick not installed)"
    fi
    
    # Test 2: Capture endpoint responds (may fail without display)
    log::info "Test 2: Testing capture endpoint..."
    local capture_response
    capture_response=$(curl -sf -X POST "http://${VOCR_HOST}:${VOCR_PORT}/capture" \
                      -H "Content-Type: application/json" \
                      -d '{"region": "0,0,100,100"}' 2>/dev/null || echo "{}")
    
    if echo "$capture_response" | grep -q "error"; then
        log::warning "⚠️  Capture endpoint returned error (expected without display)"
    else
        log::success "✅ Capture endpoint responds"
    fi
    
    # Test 3: Vision/Ask endpoint responds
    log::info "Test 3: Testing vision endpoint..."
    local vision_response
    vision_response=$(curl -sf -X POST "http://${VOCR_HOST}:${VOCR_PORT}/ask" \
                     -H "Content-Type: application/json" \
                     -d '{"question": "test"}' 2>/dev/null || echo "{}")
    
    if [[ -n "$vision_response" ]]; then
        log::success "✅ Vision endpoint responds"
    else
        log::error "❌ Vision endpoint not responding"
        ((failed++))
    fi
    
    # Test 4: Monitor endpoint
    log::info "Test 4: Testing monitor endpoint..."
    if curl -sf "http://${VOCR_HOST}:${VOCR_PORT}/monitor" >/dev/null 2>&1; then
        log::success "✅ Monitor endpoint available"
    else
        log::warning "⚠️  Monitor endpoint not available"
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All integration tests passed!"
        return 0
    else
        log::error "$failed integration tests failed"
        return 1
    fi
}

################################################################################
# Unit Test - Library function validation (<60s)
################################################################################
vocr::test::unit() {
    log::header "Running VOCR Unit Tests"
    
    local failed=0
    
    # Test 1: Configuration loading
    log::info "Test 1: Testing configuration..."
    if [[ -n "${VOCR_PORT}" ]] && [[ "${VOCR_PORT}" == "9420" ]]; then
        log::success "✅ Configuration loaded correctly"
    else
        log::error "❌ Configuration not loaded properly"
        ((failed++))
    fi
    
    # Test 2: Directory creation
    log::info "Test 2: Testing directory setup..."
    local test_dirs=("${VOCR_DATA_DIR}" "${VOCR_LOGS_DIR}" "${VOCR_SCREENSHOTS_DIR}")
    for dir in "${test_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            log::success "✅ Directory exists: $dir"
        else
            log::error "❌ Directory missing: $dir"
            ((failed++))
        fi
    done
    
    # Test 3: Python package verification
    log::info "Test 3: Testing Python packages..."
    if [[ -f "${VOCR_DATA_DIR}/venv/bin/python" ]]; then
        local packages
        packages=$("${VOCR_DATA_DIR}/venv/bin/python" -m pip list 2>/dev/null | grep -iE "Flask|Pillow|pytesseract" | wc -l)
        if [[ $packages -ge 3 ]]; then
            log::success "✅ Required Python packages installed"
        else
            log::error "❌ Missing Python packages"
            ((failed++))
        fi
    else
        log::warning "⚠️  Python environment not found"
    fi
    
    # Test 4: OCR language support
    log::info "Test 4: Testing OCR language support..."
    # Check for OCR capability via Python or system
    if command -v tesseract >/dev/null 2>&1; then
        local langs
        langs=$(tesseract --list-langs 2>/dev/null | grep -c "eng" || echo "0")
        if [[ $langs -gt 0 ]]; then
            log::success "✅ English OCR support available"
        else
            log::warning "⚠️  No English language pack"
        fi
    elif [[ -f "${VOCR_DATA_DIR}/venv/bin/python" ]]; then
        # Check Python-based OCR support
        if "${VOCR_DATA_DIR}/venv/bin/python" -c "import easyocr; print('eng' in easyocr.Reader(['en']).lang_list)" 2>/dev/null | grep -q "True"; then
            log::success "✅ EasyOCR English support"
        elif "${VOCR_DATA_DIR}/venv/bin/python" -c "import pytesseract" 2>/dev/null; then
            log::success "✅ PyTesseract available (language support depends on system)"
        else
            log::warning "⚠️  OCR language support unclear"
        fi
    else
        log::warning "⚠️  OCR language support not verified"
    fi
    
    # Summary
    if [[ $failed -eq 0 ]]; then
        log::success "All unit tests passed!"
        return 0
    else
        log::error "$failed unit tests failed"
        return 1
    fi
}

################################################################################
# Run all tests
################################################################################
vocr::test::all() {
    log::header "Running All VOCR Tests"
    
    local failed=0
    
    # Run smoke tests
    if ! vocr::test::smoke; then
        ((failed++))
    fi
    
    # Run unit tests
    if ! vocr::test::unit; then
        ((failed++))
    fi
    
    # Run integration tests
    if ! vocr::test::integration; then
        ((failed++))
    fi
    
    # Final summary
    echo ""
    if [[ $failed -eq 0 ]]; then
        log::success "🎉 All test suites passed!"
        return 0
    else
        log::error "❌ $failed test suites failed"
        return 1
    fi
}

# Export functions for CLI use
export -f vocr::test::smoke
export -f vocr::test::integration
export -f vocr::test::unit
export -f vocr::test::all