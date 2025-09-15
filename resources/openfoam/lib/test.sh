#!/bin/bash

# OpenFOAM Test Library
# Provides validation and testing functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source core library
source "$RESOURCE_DIR/lib/core.sh"

# Test phases
openfoam::test::smoke() {
    echo "Running OpenFOAM smoke tests..."
    local passed=0
    local failed=0
    
    # Test 1: Check if OpenFOAM is running
    echo -n "  1. Container running... "
    if openfoam::docker::is_running; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Container not running"
        ((failed++))
        # Try to start it
        openfoam::docker::start &>/dev/null || true
        sleep 5
    fi
    
    # Test 2: Health check
    echo -n "  2. Health check... "
    # Try to find the actual port being used
    local test_port="${OPENFOAM_PORT:-8090}"
    if openfoam::docker::is_running; then
        local container_port=$(docker port openfoam 2>/dev/null | grep -oP '\d+$' | head -1)
        if [[ -n "$container_port" ]]; then
            test_port="$container_port"
        fi
    fi
    
    if timeout 5 curl -sf "http://localhost:${test_port}/health" &>/dev/null; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Health endpoint not responding on port ${test_port}"
        ((failed++))
    fi
    
    # Test 3: OpenFOAM version check
    echo -n "  3. OpenFOAM version... "
    if openfoam::docker::is_running; then
        # Try both possible OpenFOAM installation paths
        if docker exec openfoam bash -c "source /opt/openfoam11/etc/bashrc && foamVersion" &>/dev/null || \
           docker exec openfoam bash -c "source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc && foamVersion" &>/dev/null; then
            echo "PASS"
            ((passed++))
        else
            echo "FAIL - Cannot execute foamVersion"
            ((failed++))
        fi
    else
        echo "SKIP - Container not running"
    fi
    
    # Test 4: Directory structure
    echo -n "  4. Directory structure... "
    if [[ -d "${OPENFOAM_CASES_DIR}" ]] && [[ -d "${OPENFOAM_DATA_DIR}" ]]; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Required directories missing"
        ((failed++))
    fi
    
    echo ""
    echo "Smoke Test Results: $passed passed, $failed failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

openfoam::test::integration() {
    echo "Running OpenFOAM integration tests..."
    local passed=0
    local failed=0
    
    # Ensure OpenFOAM is running
    if ! openfoam::docker::is_running; then
        echo "Starting OpenFOAM for integration tests..."
        openfoam::docker::start || {
            echo "Error: Failed to start OpenFOAM"
            return 1
        }
        sleep 10
    fi
    
    # Test 1: Create a test case
    echo -n "  1. Create test case... "
    if openfoam::case::create "test_cavity" "incompressible/simpleFoam" &>/dev/null; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Cannot create case"
        ((failed++))
    fi
    
    # Test 2: Generate mesh
    echo -n "  2. Generate mesh... "
    if [[ -d "${OPENFOAM_CASES_DIR}/test_cavity" ]]; then
        if openfoam::mesh::generate "test_cavity" &>/dev/null; then
            echo "PASS"
            ((passed++))
        else
            echo "FAIL - Mesh generation failed"
            ((failed++))
        fi
    else
        echo "SKIP - Test case not created"
    fi
    
    # Test 3: Run solver (limited iterations)
    echo -n "  3. Run solver... "
    if [[ -d "${OPENFOAM_CASES_DIR}/test_cavity" ]]; then
        # Modify controlDict for quick test
        docker exec openfoam bash -c "
            source /opt/openfoam11/etc/bashrc 2>/dev/null || source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc
            cd /cases/test_cavity
            sed -i 's/endTime.*/endTime 0.1;/' system/controlDict 2>/dev/null || true
            sed -i 's/writeInterval.*/writeInterval 0.1;/' system/controlDict 2>/dev/null || true
        " &>/dev/null || true
        
        if timeout 30 docker exec openfoam bash -c "
            source /opt/openfoam11/etc/bashrc 2>/dev/null || source /opt/OpenFOAM/OpenFOAM-v2312/etc/bashrc
            cd /cases/test_cavity
            simpleFoam -case . &>/dev/null || icoFoam -case . &>/dev/null
        "; then
            echo "PASS"
            ((passed++))
        else
            echo "FAIL - Solver execution failed"
            ((failed++))
        fi
    else
        echo "SKIP - Test case not available"
    fi
    
    # Test 4: Export results
    echo -n "  4. Export results... "
    if [[ -d "${OPENFOAM_CASES_DIR}/test_cavity" ]]; then
        if openfoam::results::export "test_cavity" "vtk" &>/dev/null; then
            echo "PASS"
            ((passed++))
        else
            echo "FAIL - Result export failed"
            ((failed++))
        fi
    else
        echo "SKIP - Test case not available"
    fi
    
    # Test 5: Content management
    echo -n "  5. Content management... "
    local content_test_passed=true
    
    # List cases
    openfoam::content::list &>/dev/null || content_test_passed=false
    
    # Add new case
    openfoam::content::add "test_content" &>/dev/null || content_test_passed=false
    
    # Get case
    openfoam::content::get "test_content" &>/dev/null || content_test_passed=false
    
    # Remove case
    openfoam::content::remove "test_content" &>/dev/null || content_test_passed=false
    
    if $content_test_passed; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Content operations failed"
        ((failed++))
    fi
    
    # Cleanup test cases
    rm -rf "${OPENFOAM_CASES_DIR}/test_cavity" &>/dev/null || true
    rm -rf "${OPENFOAM_CASES_DIR}/test_content" &>/dev/null || true
    
    echo ""
    echo "Integration Test Results: $passed passed, $failed failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

openfoam::test::unit() {
    echo "Running OpenFOAM unit tests..."
    local passed=0
    local failed=0
    
    # Test 1: Configuration loading
    echo -n "  1. Configuration loading... "
    if [[ -n "${OPENFOAM_PORT:-}" ]] && [[ -n "${OPENFOAM_DATA_DIR:-}" ]]; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Configuration not loaded"
        ((failed++))
    fi
    
    # Test 2: Docker command availability
    echo -n "  2. Docker availability... "
    if timeout 2 bash -c "command -v docker" &>/dev/null; then
        echo "PASS"
        ((passed++))
    else
        echo "SKIP - Docker not available"
    fi
    
    # Test 3: Directory creation
    echo -n "  3. Directory creation... "
    local test_dir="/tmp/openfoam_test_$$"
    mkdir -p "$test_dir"
    if [[ -d "$test_dir" ]]; then
        rm -rf "$test_dir"
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Cannot create directories"
        ((failed++))
    fi
    
    # Test 4: Port validation
    echo -n "  4. Port validation... "
    if [[ "${OPENFOAM_PORT:-8090}" -ge 1024 ]] && [[ "${OPENFOAM_PORT:-8090}" -le 65535 ]]; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Invalid port configuration"
        ((failed++))
    fi
    
    # Test 5: Memory limit parsing
    echo -n "  5. Memory limit parsing... "
    if [[ "${OPENFOAM_MEMORY_LIMIT:-4g}" =~ ^[0-9]+(m|g)$ ]]; then
        echo "PASS"
        ((passed++))
    else
        echo "FAIL - Invalid memory limit format"
        ((failed++))
    fi
    
    echo ""
    echo "Unit Test Results: $passed passed, $failed failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

openfoam::test::all() {
    echo "Running all OpenFOAM test phases..."
    echo "=================================="
    
    local smoke_result=0
    local unit_result=0
    local integration_result=0
    
    # Run smoke tests
    echo ""
    openfoam::test::smoke || smoke_result=$?
    
    # Run unit tests
    echo ""
    openfoam::test::unit || unit_result=$?
    
    # Run integration tests only if smoke passed
    if [[ $smoke_result -eq 0 ]]; then
        echo ""
        openfoam::test::integration || integration_result=$?
    else
        echo ""
        echo "Skipping integration tests due to smoke test failure"
        integration_result=1
    fi
    
    # Summary
    echo ""
    echo "=================================="
    echo "Test Summary:"
    echo "  Smoke:       $([ $smoke_result -eq 0 ] && echo "PASS" || echo "FAIL")"
    echo "  Unit:        $([ $unit_result -eq 0 ] && echo "PASS" || echo "FAIL")"
    echo "  Integration: $([ $integration_result -eq 0 ] && echo "PASS" || echo "FAIL")"
    
    # Return failure if any test failed
    [[ $smoke_result -eq 0 ]] && [[ $unit_result -eq 0 ]] && [[ $integration_result -eq 0 ]]
}