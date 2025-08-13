#!/usr/bin/env bash
# Browserless Simple Test Data Injection
# Minimal test data injection for service validation

#######################################  
# Inject minimal test data for validation
# Creates simple test files to verify Browserless is working
#######################################
browserless::inject() {
    log::info "Injecting test data into Browserless..."
    
    # Verify service is available
    if ! docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::error "Browserless container not running - start it first"
        log::info "Run: ./manage.sh --action start"
        return 1
    fi
    
    # Create test workspace
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    mkdir -p "$test_dir"
    
    log::info "Creating test validation script..."
    
    # Create minimal test script
    cat > "$test_dir/validation.js" << 'EOF'
module.exports = async ({ page }) => {
    await page.goto('https://httpbin.org/html');
    const title = await page.title();
    return { 
        success: true, 
        title, 
        url: page.url(),
        timestamp: new Date().toISOString() 
    };
};
EOF
    
    # Create simple screenshot test
    cat > "$test_dir/screenshot-test.js" << 'EOF'
module.exports = async ({ page }) => {
    await page.goto('https://httpbin.org/html');
    await page.screenshot({ path: '/workspace/screenshots/validation-test.png' });
    return { success: true, message: 'Screenshot saved to validation-test.png' };
};
EOF
    
    log::success "Test scripts created in $test_dir"
    
    # Validate injection worked
    if browserless::validate_injection; then
        log::success "✅ Test data injected and validated successfully"
        browserless::show_injection_info
        return 0
    else
        log::error "❌ Injection validation failed"
        return 1
    fi
}

#######################################
# Validate injection is working  
#######################################
browserless::validate_injection() {
    # Check test directory exists
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    if [[ ! -d "$test_dir" ]]; then
        log::error "Test data directory not found: $test_dir"
        return 1
    fi
    
    # Check test files exist
    if [[ ! -f "$test_dir/validation.js" ]]; then
        log::error "Validation script not found"
        return 1
    fi
    
    if [[ ! -f "$test_dir/screenshot-test.js" ]]; then
        log::error "Screenshot test script not found"  
        return 1
    fi
    
    # Check service responds
    if ! http::check_endpoint "http://localhost:$BROWSERLESS_PORT/pressure"; then
        log::error "Browserless API not responding"
        return 1
    fi
    
    log::success "All injection validation checks passed"
    return 0
}

#######################################
# Show injection status and information
#######################################  
browserless::injection_status() {
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    
    log::header "📊 Browserless Injection Status"
    
    if [[ -d "$test_dir" ]]; then
        local file_count=$(find "$test_dir" -name "*.js" | wc -l)
        log::info "Test data directory: $test_dir"
        log::info "Test script files: $file_count"
        
        # List test files
        if [[ $file_count -gt 0 ]]; then
            echo "Test files:"
            find "$test_dir" -name "*.js" -exec basename {} \; | sed 's/^/  - /'
        fi
        
        echo
        if browserless::validate_injection 2>/dev/null; then
            log::success "✅ Injection is healthy and validated"
        else
            log::warn "⚠️  Injection exists but validation failed"  
            log::info "Try: ./manage.sh --action inject (to recreate)"
        fi
    else
        log::info "❌ No test data found"
        log::info "Run: ./manage.sh --action inject"
    fi
}

#######################################
# Show information about injected test data
#######################################
browserless::show_injection_info() {
    echo
    echo "Test Data Information:"
    echo "  Location: ${BROWSERLESS_DATA_DIR}/test-data/"
    echo "  Files created: validation.js, screenshot-test.js"
    echo
    echo "Next steps:"
    echo "  ./manage.sh --action usage --usage-type screenshot  # Test screenshot API"
    echo "  ./manage.sh --action usage --usage-type all         # Test all APIs"
    echo "  ./manage.sh --action injection-status              # Check injection status"
}

#######################################
# Clean up test data
#######################################
browserless::cleanup_injection() {
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    
    if [[ -d "$test_dir" ]]; then
        log::info "Removing test data directory..."
        rm -rf "$test_dir"
        log::success "✅ Test data cleaned up"
        
        # Also clean up any test screenshots
        local screenshots_dir="${BROWSERLESS_DATA_DIR}/screenshots"
        if [[ -f "$screenshots_dir/validation-test.png" ]]; then
            rm -f "$screenshots_dir/validation-test.png"
            log::info "Removed validation screenshot"
        fi
    else
        log::info "No test data to clean up"
    fi
}