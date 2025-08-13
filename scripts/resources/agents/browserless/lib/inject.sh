#!/usr/bin/env bash
# Browserless Data Injection Functions
# Handles test data and configuration injection

#######################################
# Inject test data into Browserless
#######################################
browserless::inject() {
    log::info "Injecting test data into Browserless..."
    
    if ! docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::error "Browserless is not running"
        return 1
    fi
    
    # Create test scripts directory
    local scripts_dir="${BROWSERLESS_DATA_DIR}/scripts"
    mkdir -p "$scripts_dir"
    
    # Create sample screenshot script
    cat > "$scripts_dir/test_screenshot.js" << 'EOF'
module.exports = async ({ page }) => {
    await page.goto('https://example.com');
    await page.screenshot({ path: '/workspace/screenshots/test.png' });
    return { success: true, message: 'Screenshot captured' };
};
EOF
    
    # Create sample scraping script
    cat > "$scripts_dir/test_scrape.js" << 'EOF'
module.exports = async ({ page }) => {
    await page.goto('https://example.com');
    const title = await page.title();
    const content = await page.evaluate(() => document.body.innerText);
    return { title, content: content.substring(0, 200) };
};
EOF
    
    log::success "Test scripts injected successfully"
    
    # Test the injection
    if browserless::validate_injection; then
        log::success "Injection validated successfully"
    else
        log::warn "Injection validation failed"
    fi
    
    return 0
}

#######################################
# Validate injected data
#######################################
browserless::validate_injection() {
    log::info "Validating injected data..."
    
    # Check if scripts directory exists
    if [[ ! -d "${BROWSERLESS_DATA_DIR}/scripts" ]]; then
        log::error "Scripts directory not found"
        return 1
    fi
    
    # Check if test scripts exist
    local scripts_found=0
    if [[ -f "${BROWSERLESS_DATA_DIR}/scripts/test_screenshot.js" ]]; then
        ((scripts_found++))
        log::success "Found test_screenshot.js"
    fi
    
    if [[ -f "${BROWSERLESS_DATA_DIR}/scripts/test_scrape.js" ]]; then
        ((scripts_found++))
        log::success "Found test_scrape.js"
    fi
    
    if [[ $scripts_found -eq 0 ]]; then
        log::error "No test scripts found"
        return 1
    fi
    
    log::success "Validation completed: $scripts_found scripts found"
    return 0
}

#######################################
# Run usage examples
#######################################
browserless::run_usage_example() {
    local usage_type="${1:-help}"
    
    case "$usage_type" in
        screenshot)
            browserless::test_screenshot_example
            ;;
        pdf)
            browserless::test_pdf_example
            ;;
        scrape)
            browserless::test_scrape_example
            ;;
        pressure)
            browserless::test_pressure_example
            ;;
        function)
            browserless::test_function_example
            ;;
        all)
            browserless::test_screenshot_example
            browserless::test_pdf_example
            browserless::test_scrape_example
            browserless::test_pressure_example
            browserless::test_function_example
            ;;
        help|*)
            browserless::show_usage_help
            ;;
    esac
}

#######################################
# Show usage help
#######################################
browserless::show_usage_help() {
    echo "Browserless Usage Examples"
    echo "========================="
    echo
    echo "Available examples:"
    echo "  screenshot - Test screenshot capture"
    echo "  pdf       - Test PDF generation"
    echo "  scrape    - Test web scraping"
    echo "  pressure  - Test browser pressure"
    echo "  function  - Test function execution"
    echo "  all       - Run all examples"
    echo
    echo "Usage:"
    echo "  ./manage.sh --action usage --usage-type screenshot"
    echo "  ./manage.sh --action usage --usage-type all"
}

#######################################
# Test screenshot example
#######################################
browserless::test_screenshot_example() {
    log::info "Testing screenshot capture..."
    
    local url="${URL:-https://example.com}"
    local output="${OUTPUT:-/tmp/browserless_screenshot.png}"
    
    if browserless::screenshot "$url" "$output"; then
        log::success "Screenshot saved to: $output"
        
        if [[ -f "$output" ]]; then
            local size
            size=$(stat -f%z "$output" 2>/dev/null || stat -c%s "$output" 2>/dev/null || echo "0")
            log::info "File size: $size bytes"
        fi
    else
        log::error "Screenshot capture failed"
        return 1
    fi
}

#######################################
# Test PDF example
#######################################
browserless::test_pdf_example() {
    log::info "Testing PDF generation..."
    
    local url="${URL:-https://example.com}"
    local output="${OUTPUT:-/tmp/browserless_document.pdf}"
    
    if browserless::pdf "$url" "$output"; then
        log::success "PDF saved to: $output"
        
        if [[ -f "$output" ]]; then
            local size
            size=$(stat -f%z "$output" 2>/dev/null || stat -c%s "$output" 2>/dev/null || echo "0")
            log::info "File size: $size bytes"
        fi
    else
        log::error "PDF generation failed"
        return 1
    fi
}

#######################################
# Test scrape example
#######################################
browserless::test_scrape_example() {
    log::info "Testing web scraping..."
    
    local url="${URL:-https://example.com}"
    local selector="${SELECTOR:-.main-content}"
    
    if content=$(browserless::scrape "$url" "$selector"); then
        log::success "Content scraped successfully"
        echo "Content preview:"
        echo "$content" | head -5
    else
        log::error "Web scraping failed"
        return 1
    fi
}

#######################################
# Test pressure example
#######################################
browserless::test_pressure_example() {
    log::info "Testing browser pressure..."
    
    if browserless::pressure_test; then
        log::success "Pressure test completed"
    else
        log::error "Pressure test failed"
        return 1
    fi
}

#######################################
# Test function example
#######################################
browserless::test_function_example() {
    log::info "Testing function execution..."
    
    local code='() => { return { title: document.title, url: window.location.href }; }'
    
    local response
    response=$(curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/function" \
        -H "Content-Type: application/json" \
        -d "{\"code\":\"$code\"}" \
        2>/dev/null)
    
    if [[ -n "$response" ]]; then
        log::success "Function executed successfully"
        echo "Result: $response"
    else
        log::error "Function execution failed"
        return 1
    fi
}

#######################################
# Screenshot helper function
#######################################
browserless::screenshot() {
    local url="${1:-https://example.com}"
    local output="${2:-/tmp/screenshot.png}"
    
    curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$url\"}" \
        -o "$output"
}

#######################################
# PDF helper function
#######################################
browserless::pdf() {
    local url="${1:-https://example.com}"
    local output="${2:-/tmp/document.pdf}"
    
    curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/pdf" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$url\"}" \
        -o "$output"
}

#######################################
# Scrape helper function
#######################################
browserless::scrape() {
    local url="${1:-https://example.com}"
    local selector="${2:-}"
    
    local data="{\"url\":\"$url\""
    if [[ -n "$selector" ]]; then
        data+=",\"selector\":\"$selector\""
    fi
    data+="}"
    
    curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/content" \
        -H "Content-Type: application/json" \
        -d "$data"
}

#######################################
# Pressure test helper function
#######################################
browserless::pressure_test() {
    local pressure_data
    pressure_data=$(curl -s "http://localhost:$BROWSERLESS_PORT/pressure")
    
    if [[ -n "$pressure_data" ]]; then
        echo "Browser Pressure Status:"
        echo "$pressure_data" | jq '.' 2>/dev/null || echo "$pressure_data"
        return 0
    else
        return 1
    fi
}