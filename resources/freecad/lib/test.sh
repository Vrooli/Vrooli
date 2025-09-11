#!/usr/bin/env bash
################################################################################
# FreeCAD Resource - Test Library Functions
################################################################################

# Run all tests
freecad::test::all() {
    log::info "Running all FreeCAD tests..."
    
    local failed=0
    
    # Run test phases
    freecad::test::smoke || ((failed++))
    freecad::test::integration || ((failed++))
    freecad::test::unit || ((failed++))
    
    if [[ $failed -eq 0 ]]; then
        log::info "All tests passed"
        return 0
    else
        log::error "$failed test suite(s) failed"
        return 1
    fi
}

# Smoke test - quick health check
freecad::test::smoke() {
    log::info "Running FreeCAD smoke tests..."
    
    # Delegate to test phase script if it exists
    local test_script="${FREECAD_CLI_DIR}/test/phases/test-smoke.sh"
    if [[ -f "$test_script" ]]; then
        bash "$test_script"
        return $?
    fi
    
    # Fallback to basic health check
    if freecad::health_check; then
        log::info "Smoke test passed"
        return 0
    else
        log::error "Smoke test failed"
        return 1
    fi
}

# Integration tests
freecad::test::integration() {
    log::info "Running FreeCAD integration tests..."
    
    # Delegate to test phase script if it exists
    local test_script="${FREECAD_CLI_DIR}/test/phases/test-integration.sh"
    if [[ -f "$test_script" ]]; then
        bash "$test_script"
        return $?
    fi
    
    # Basic integration test
    if ! freecad::is_running; then
        log::error "FreeCAD is not running"
        return 1
    fi
    
    # Test Python API availability
    local test_py="/tmp/freecad_test_${RANDOM}.py"
    cat > "$test_py" <<'EOF'
import FreeCAD
print("FreeCAD version:", FreeCAD.Version())
EOF
    
    if freecad::content::execute "$test_py"; then
        rm "$test_py"
        log::info "Integration test passed"
        return 0
    else
        rm "$test_py"
        log::error "Integration test failed"
        return 1
    fi
}

# Unit tests
freecad::test::unit() {
    log::info "Running FreeCAD unit tests..."
    
    # Delegate to test phase script if it exists
    local test_script="${FREECAD_CLI_DIR}/test/phases/test-unit.sh"
    if [[ -f "$test_script" ]]; then
        bash "$test_script"
        return $?
    fi
    
    # No unit tests defined yet
    log::info "No unit tests available"
    return 2
}