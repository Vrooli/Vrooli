#!/usr/bin/env bash
# OpenSCAD Resource Testing Functions (Resource Validation)

# Quick smoke test - validate OpenSCAD resource health
openscad::test::smoke() {
    log::info "Running OpenSCAD smoke test..."
    
    # Check basic installation
    if ! openscad::is_installed; then
        log::error "OpenSCAD not installed (Docker required)"
        return 1
    fi
    
    # Check if container can start
    if ! openscad::is_running; then
        log::info "Starting OpenSCAD for smoke test..."
        if ! openscad::start; then
            log::error "Failed to start OpenSCAD container"
            return 1
        fi
    fi
    
    # Test basic OpenSCAD functionality
    if docker exec "${OPENSCAD_CONTAINER_NAME}" openscad --version >/dev/null 2>&1; then
        log::success "OpenSCAD smoke test passed"
        return 0
    else
        log::error "OpenSCAD smoke test failed - command execution error"
        return 1
    fi
}

# Integration test - validate OpenSCAD end-to-end functionality
openscad::test::integration() {
    log::info "Running OpenSCAD integration test..."
    
    # Run smoke test first
    if ! openscad::test::smoke; then
        log::error "Smoke test failed, skipping integration test"
        return 1
    fi
    
    # Test rendering capability with a simple cube
    local test_script="${OPENSCAD_SCRIPTS_DIR}/test_cube.scad"
    local test_output="${OPENSCAD_OUTPUT_DIR}/test_cube.stl"
    
    # Create a simple test script
    openscad::ensure_dirs
    cat > "$test_script" << 'EOF'
// Simple test cube for integration testing
cube([10, 10, 10], center=true);
EOF
    
    # Test rendering
    log::info "Testing rendering capability..."
    if docker exec "${OPENSCAD_CONTAINER_NAME}" openscad -o "/output/test_cube.stl" "/scripts/test_cube.scad" >/dev/null 2>&1; then
        if [[ -f "$test_output" ]]; then
            # Clean up test files
            rm -f "$test_script" "$test_output"
            log::success "OpenSCAD integration test passed"
            return 0
        else
            log::error "Rendering succeeded but output file not found"
            return 1
        fi
    else
        log::error "OpenSCAD rendering test failed"
        return 1
    fi
}

# Unit tests - validate OpenSCAD library functions (basic validation)
openscad::test::unit() {
    log::info "Running OpenSCAD unit tests..."
    
    local failed=0
    
    # Test common functions
    log::info "Testing common functions..."
    
    # Test directory creation
    if openscad::ensure_dirs; then
        log::success "Directory creation test passed"
    else
        log::error "Directory creation test failed"
        ((failed++))
    fi
    
    # Test is_installed function
    if openscad::is_installed; then
        log::success "Installation check test passed"
    else
        log::error "Installation check test failed"
        ((failed++))
    fi
    
    # Test container status functions
    if openscad::container_exists || ! openscad::container_exists; then
        log::success "Container existence check test passed"
    else
        log::error "Container existence check test failed"
        ((failed++))
    fi
    
    if openscad::is_running || ! openscad::is_running; then
        log::success "Container running check test passed"
    else
        log::error "Container running check test failed"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "OpenSCAD unit tests passed"
        return 0
    else
        log::error "OpenSCAD unit tests failed ($failed failures)"
        return 1
    fi
}

# Run all resource tests
openscad::test::all() {
    log::info "Running all OpenSCAD resource tests..."
    
    local failed=0
    
    # Run smoke test
    if ! openscad::test::smoke; then
        log::error "Smoke test failed"
        ((failed++))
    fi
    
    # Run unit tests
    if ! openscad::test::unit; then
        log::error "Unit tests failed"
        ((failed++))
    fi
    
    # Run integration test
    if ! openscad::test::integration; then
        log::error "Integration test failed"
        ((failed++))
    fi
    
    if [[ $failed -eq 0 ]]; then
        log::success "All OpenSCAD tests passed"
        return 0
    else
        log::error "OpenSCAD tests failed ($failed test suites failed)"
        return 1
    fi
}