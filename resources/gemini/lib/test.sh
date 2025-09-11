#!/usr/bin/env bash
# Gemini Resource Testing Functions (Resource Validation, not AI Generation)

# ==================== RESOURCE VALIDATION FUNCTIONS ====================
# These test Gemini as a resource (health, connectivity, configuration)

# Quick smoke test - validate Gemini API connectivity and health
gemini::test::smoke() {
    log::info "Running Gemini API smoke test..."
    
    # Initialize configuration
    gemini::init >/dev/null 2>&1
    
    if gemini::test_connection; then
        log::success "✓ Gemini API smoke test passed"
        return 0
    else
        log::error "✗ Gemini API smoke test failed"
        return 1
    fi
}

# Full integration test - validate Gemini end-to-end functionality  
gemini::test::integration() {
    log::info "Running Gemini API integration test..."
    
    # Test 1: Configuration and initialization
    if ! gemini::init >/dev/null 2>&1; then
        log::error "✗ Failed to initialize Gemini configuration"
        return 1
    fi
    log::info "✓ Configuration initialized"
    
    # Test 2: API connectivity
    if ! gemini::test_connection; then
        log::error "✗ Failed to connect to Gemini API"
        return 1
    fi
    log::info "✓ API connectivity verified"
    
    # Test 3: Model listing (if API key is valid)
    if [[ "$GEMINI_API_KEY" != "placeholder-gemini-key" ]]; then
        if ! gemini::list_models >/dev/null 2>&1; then
            log::error "✗ Failed to list Gemini models"
            return 1
        fi
        log::info "✓ Model listing verified"
    else
        log::info "~ Model listing skipped (placeholder API key)"
    fi
    
    # Test 4: Content storage functionality
    if ! gemini::content::list >/dev/null 2>&1; then
        log::error "✗ Failed to access content storage"
        return 1
    fi
    log::info "✓ Content storage accessible"
    
    log::success "✓ Gemini API integration test passed"
    return 0
}

# Unit tests - validate Gemini library functions
gemini::test::unit() {
    log::info "Running Gemini library unit tests..."
    
    # Test configuration loading
    local test_passed=0
    
    # Test 1: Default configuration
    if [[ -n "$GEMINI_API_BASE" ]] && [[ -n "$GEMINI_DEFAULT_MODEL" ]]; then
        log::info "✓ Default configuration loaded"
        ((test_passed++))
    else
        log::error "✗ Default configuration not loaded properly"
    fi
    
    # Test 2: Content storage initialization
    local temp_storage="/tmp/gemini-test-$$"
    export GEMINI_CONTENT_STORAGE="$temp_storage"
    if gemini::content::init 2>/dev/null; then
        if [[ -d "$temp_storage/prompts" ]] && [[ -d "$temp_storage/templates" ]]; then
            log::info "✓ Content storage initialization works"
            ((test_passed++))
        else
            log::error "✗ Content storage directories not created"
        fi
        rm -rf "$temp_storage"
    else
        log::error "✗ Content storage initialization failed"
    fi
    
    if [[ $test_passed -eq 2 ]]; then
        log::success "✓ Gemini library unit tests passed ($test_passed/2)"
        return 0
    else
        log::error "✗ Gemini library unit tests failed ($test_passed/2)"
        return 1
    fi
}

# Cache tests - validate Redis integration and caching functionality
gemini::test::cache() {
    log::info "Running Gemini cache tests..."
    
    # Run the cache test script
    local test_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    if [[ -x "${test_dir}/test/phases/test-cache.sh" ]]; then
        "${test_dir}/test/phases/test-cache.sh"
        return $?
    else
        log::error "Cache test script not found or not executable"
        return 1
    fi
}

# Run all resource tests
gemini::test::all() {
    log::info "Running all Gemini resource tests..."
    
    local tests_passed=0
    local total_tests=4
    
    if gemini::test::unit; then
        ((tests_passed++))
    fi
    
    if gemini::test::smoke; then
        ((tests_passed++))
    fi
    
    if gemini::test::integration; then
        ((tests_passed++))
    fi
    
    if gemini::test::cache; then
        ((tests_passed++))
    fi
    
    if [[ $tests_passed -eq $total_tests ]]; then
        log::success "✓ All Gemini resource tests passed ($tests_passed/$total_tests)"
        return 0
    else
        log::error "✗ Some Gemini resource tests failed ($tests_passed/$total_tests)"
        return 1
    fi
}

# Export functions
export -f gemini::test::smoke
export -f gemini::test::integration
export -f gemini::test::unit
export -f gemini::test::cache
export -f gemini::test::all