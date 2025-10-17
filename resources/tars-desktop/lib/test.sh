#!/bin/bash
# TARS-desktop test functionality - health checks and validation

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TARS_DESKTOP_TEST_DIR="${APP_ROOT}/resources/tars-desktop/lib"

# Source dependencies
source "${TARS_DESKTOP_TEST_DIR}/core.sh"

# Smoke test - basic health check
tars_desktop::test::smoke() {
    local verbose="${1:-false}"
    
    log::info "Running TARS-desktop smoke test..."
    
    # Check if installed
    if ! tars_desktop::is_installed; then
        log::error "TARS-desktop is not installed"
        return 1
    fi
    
    # Check if running
    if ! tars_desktop::is_running; then
        log::warn "TARS-desktop is not running - attempting to start"
        if ! tars_desktop::start "$verbose"; then
            log::error "Failed to start TARS-desktop"
            return 1
        fi
    fi
    
    # Health check
    if ! tars_desktop::health_check; then
        log::error "TARS-desktop health check failed"
        return 1
    fi
    
    log::success "TARS-desktop smoke test passed"
    return 0
}

# Integration test
tars_desktop::test::integration() {
    local verbose="${1:-false}"
    
    log::info "Running TARS-desktop integration test..."
    
    # First run smoke test
    if ! tars_desktop::test::smoke "$verbose"; then
        log::error "Smoke test failed"
        return 1
    fi
    
    # Test capabilities endpoint
    local capabilities
    capabilities=$(tars_desktop::get_capabilities 2>/dev/null)
    if [[ -z "$capabilities" ]]; then
        log::error "Failed to get capabilities"
        return 1
    fi
    
    [[ "$verbose" == "true" ]] && log::info "Capabilities: $capabilities"
    
    # Test screenshot functionality
    local temp_screenshot="/tmp/tars_test_screenshot_$(date +%s).png"
    if tars_desktop::screenshot "$temp_screenshot"; then
        if [[ -f "$temp_screenshot" ]]; then
            [[ "$verbose" == "true" ]] && log::info "Screenshot test passed"
            rm -f "$temp_screenshot"
        else
            log::error "Screenshot file not created"
            return 1
        fi
    else
        log::error "Screenshot test failed"
        return 1
    fi
    
    log::success "TARS-desktop integration test passed"
    return 0
}

# Unit tests (basic component validation)
tars_desktop::test::unit() {
    local verbose="${1:-false}"
    
    log::info "Running TARS-desktop unit tests..."
    
    # Test installation directory exists
    if [[ ! -d "$TARS_DESKTOP_INSTALL_DIR" ]]; then
        log::error "Installation directory missing: $TARS_DESKTOP_INSTALL_DIR"
        return 1
    fi
    
    # Test server script exists
    if [[ ! -f "${TARS_DESKTOP_INSTALL_DIR}/server.py" ]]; then
        log::error "Server script missing"
        return 1
    fi
    
    # Test Python dependencies
    local python_cmd
    if [[ -f "${TARS_DESKTOP_VENV_DIR}/bin/activate" ]]; then
        python_cmd="${TARS_DESKTOP_VENV_DIR}/bin/python"
    else
        python_cmd="python3"
    fi
    
    if ! $python_cmd -c "import pyautogui, fastapi, uvicorn" 2>/dev/null; then
        log::error "Required Python dependencies missing"
        return 1
    fi
    
    [[ "$verbose" == "true" ]] && log::info "All unit tests passed"
    
    log::success "TARS-desktop unit tests passed"
    return 0
}

# Run all tests
tars_desktop::test::all() {
    local verbose="${1:-false}"
    
    log::info "Running all TARS-desktop tests..."
    
    # Run unit tests
    if ! tars_desktop::test::unit "$verbose"; then
        log::error "Unit tests failed"
        return 1
    fi
    
    # Run smoke tests
    if ! tars_desktop::test::smoke "$verbose"; then
        log::error "Smoke tests failed"
        return 1
    fi
    
    # Run integration tests
    if ! tars_desktop::test::integration "$verbose"; then
        log::error "Integration tests failed"
        return 1
    fi
    
    log::success "All TARS-desktop tests passed"
    return 0
}

# Export functions
export -f tars_desktop::test::smoke
export -f tars_desktop::test::integration
export -f tars_desktop::test::unit
export -f tars_desktop::test::all