#!/bin/bash
# SU2 Test Functionality

# Run smoke tests
test_smoke() {
    echo "Running SU2 smoke tests..."
    
    # Test 1: Check if service is running
    echo -n "  [1/3] Service running... "
    if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        echo "✓"
    else
        echo "✗ (Service not running)"
        return 1
    fi
    
    # Test 2: Health check
    echo -n "  [2/3] Health check... "
    if timeout 5 curl -sf "http://localhost:${SU2_PORT}/health" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (Health check failed)"
        return 1
    fi
    
    # Test 3: API endpoints
    echo -n "  [3/3] API endpoints... "
    if timeout 5 curl -sf "http://localhost:${SU2_PORT}/api/status" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (API not responding)"
        return 1
    fi
    
    echo "Smoke tests passed"
    return 0
}

# Run integration tests
test_integration() {
    echo "Running SU2 integration tests..."
    
    # Test 1: List operations
    echo -n "  [1/4] List operations... "
    if content_list > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test 2: Check example files
    echo -n "  [2/4] Example files... "
    if [[ -f "${SU2_MESHES_DIR}/naca0012.su2" ]] && [[ -f "${SU2_CONFIGS_DIR}/naca0012.cfg" ]]; then
        echo "✓"
    else
        echo "✗ (Example files missing)"
        # Try to download them
        download_examples
    fi
    
    # Test 3: Submit simulation test
    echo -n "  [3/4] Simulation submission... "
    if docker ps --format '{{.Names}}' | grep -q "^${SU2_CONTAINER_NAME}$"; then
        local response=$(curl -sf -X POST "http://localhost:${SU2_PORT}/api/simulate" \
            -H "Content-Type: application/json" \
            -d '{"mesh":"naca0012.su2","config":"test.cfg"}' 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "✓"
        else
            echo "✗ (Simulation submission failed)"
            return 1
        fi
    else
        echo "✗ (Service not running)"
        return 1
    fi
    
    # Test 4: Check MPI availability
    echo -n "  [4/4] MPI functionality... "
    if docker exec "${SU2_CONTAINER_NAME}" mpirun --version > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (MPI not available)"
    fi
    
    echo "Integration tests passed"
    return 0
}

# Run unit tests
test_unit() {
    echo "Running SU2 unit tests..."
    
    # Test configuration loading
    echo -n "  [1/3] Configuration loading... "
    if [[ -n "$SU2_PORT" ]] && [[ -n "$SU2_DATA_DIR" ]]; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test directory structure
    echo -n "  [2/3] Directory structure... "
    local all_dirs_exist=true
    for dir in "${SU2_MESHES_DIR}" "${SU2_CONFIGS_DIR}" "${SU2_RESULTS_DIR}"; do
        if [[ ! -d "$dir" ]]; then
            all_dirs_exist=false
            break
        fi
    done
    
    if [[ "$all_dirs_exist" == "true" ]]; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    # Test Docker availability
    echo -n "  [3/3] Docker availability... "
    if docker --version > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗"
        return 1
    fi
    
    echo "Unit tests passed"
    return 0
}

# Run all tests
test_all() {
    local failed=0
    
    echo "Running all SU2 tests..."
    echo
    
    test_unit || ((failed++))
    echo
    
    test_smoke || ((failed++))
    echo
    
    test_integration || ((failed++))
    echo
    
    if [[ $failed -eq 0 ]]; then
        echo "All tests passed successfully"
        return 0
    else
        echo "Warning: $failed test suite(s) failed"
        return 1
    fi
}