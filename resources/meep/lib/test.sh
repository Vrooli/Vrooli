#!/usr/bin/env bash
# MEEP Test Library Functions

set -euo pipefail

# Ensure core functions are available
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi
source "${SCRIPT_DIR}/lib/core.sh"

#######################################
# Main test dispatcher
#######################################
meep::test() {
    local phase="${1:-all}"
    shift || true
    
    case "$phase" in
        smoke)
            meep::test_smoke "$@"
            ;;
        integration)
            meep::test_integration "$@"
            ;;
        unit)
            meep::test_unit "$@"
            ;;
        all)
            meep::test_all "$@"
            ;;
        *)
            echo "Error: Unknown test phase: $phase" >&2
            echo "Available: smoke, integration, unit, all" >&2
            return 1
            ;;
    esac
}

#######################################
# Smoke test - Quick health check
#######################################
meep::test_smoke() {
    echo "Running MEEP smoke tests..."
    echo "=========================="
    
    local passed=0
    local failed=0
    
    # Test 1: Check if service is running
    echo -n "Test 1: Service running... "
    if meep::is_running; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Service not running"
        ((failed++))
    fi
    
    # Test 2: Health endpoint responds
    echo -n "Test 2: Health check... "
    if timeout 5 curl -sf "http://localhost:${MEEP_PORT}/health" > /dev/null; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Health check failed"
        ((failed++))
    fi
    
    # Test 3: API responds with correct status
    echo -n "Test 3: API status... "
    local health_response
    health_response=$(timeout 5 curl -sf "http://localhost:${MEEP_PORT}/health" 2>/dev/null || echo "{}")
    if echo "$health_response" | grep -q '"status":"healthy"'; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - API status incorrect"
        ((failed++))
    fi
    
    # Test 4: Templates endpoint works
    echo -n "Test 4: Templates listing... "
    if timeout 5 curl -sf "http://localhost:${MEEP_PORT}/templates" > /dev/null; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Templates endpoint failed"
        ((failed++))
    fi
    
    # Test 5: Data directories exist
    echo -n "Test 5: Data directories... "
    if [[ -d "${MEEP_DATA_DIR}" ]] && [[ -d "${MEEP_RESULTS_DIR}" ]]; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Data directories missing"
        ((failed++))
    fi
    
    # Summary
    echo
    echo "Smoke Test Results"
    echo "=================="
    echo "Passed: $passed"
    echo "Failed: $failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Integration test - Full functionality
#######################################
meep::test_integration() {
    echo "Running MEEP integration tests..."
    echo "================================="
    
    local passed=0
    local failed=0
    
    # Test 1: Create simulation
    echo -n "Test 1: Create simulation... "
    local sim_response
    sim_response=$(curl -sf -X POST \
        "http://localhost:${MEEP_PORT}/simulation/create" \
        -H "Content-Type: application/json" \
        -d '{"template": "waveguide", "resolution": 30, "runtime": 50}' 2>/dev/null || echo "{}")
    
    if echo "$sim_response" | grep -q '"simulation_id"'; then
        echo "PASSED"
        ((passed++))
        local sim_id=$(echo "$sim_response" | jq -r '.simulation_id')
    else
        echo "FAILED - Could not create simulation"
        ((failed++))
        local sim_id=""
    fi
    
    # Test 2: Run simulation (if created)
    if [[ -n "$sim_id" ]]; then
        echo -n "Test 2: Run simulation... "
        if curl -sf -X POST "http://localhost:${MEEP_PORT}/simulation/${sim_id}/run" > /dev/null; then
            echo "PASSED"
            ((passed++))
        else
            echo "FAILED - Could not run simulation"
            ((failed++))
        fi
        
        # Test 3: Check simulation status
        echo -n "Test 3: Check status... "
        sleep 2
        local status_response
        status_response=$(curl -sf "http://localhost:${MEEP_PORT}/simulation/${sim_id}/status" 2>/dev/null || echo "{}")
        
        if echo "$status_response" | grep -q '"status"'; then
            echo "PASSED"
            ((passed++))
        else
            echo "FAILED - Could not get status"
            ((failed++))
        fi
        
        # Test 4: Get spectra data
        echo -n "Test 4: Get spectra... "
        if curl -sf "http://localhost:${MEEP_PORT}/simulation/${sim_id}/spectra" > /dev/null; then
            echo "PASSED"
            ((passed++))
        else
            echo "FAILED - Could not get spectra"
            ((failed++))
        fi
    else
        echo "Skipping tests 2-4 due to simulation creation failure"
        ((failed+=3))
    fi
    
    # Test 5: Parameter sweep
    echo -n "Test 5: Parameter sweep... "
    local sweep_response
    sweep_response=$(curl -sf -X POST \
        "http://localhost:${MEEP_PORT}/batch/sweep" \
        -H "Content-Type: application/json" \
        -d '{"template": "waveguide", "parameter": "resolution", "values": [20, 30, 40]}' 2>/dev/null || echo "{}")
    
    if echo "$sweep_response" | grep -q '"sweep_id"'; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Parameter sweep failed"
        ((failed++))
    fi
    
    # Test 6: Template management
    echo -n "Test 6: Template operations... "
    local templates_response
    templates_response=$(curl -sf "http://localhost:${MEEP_PORT}/templates" 2>/dev/null || echo "{}")
    
    if echo "$templates_response" | grep -q '"waveguide"'; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Template operations failed"
        ((failed++))
    fi
    
    # Summary
    echo
    echo "Integration Test Results"
    echo "========================"
    echo "Passed: $passed"
    echo "Failed: $failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Unit test - Library functions
#######################################
meep::test_unit() {
    echo "Running MEEP unit tests..."
    echo "========================="
    
    local passed=0
    local failed=0
    
    # Test 1: Configuration loading
    echo -n "Test 1: Configuration... "
    if [[ -n "${MEEP_PORT:-}" ]] && [[ "${MEEP_PORT}" == "8193" ]]; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Configuration not loaded"
        ((failed++))
    fi
    
    # Test 2: Directory creation
    echo -n "Test 2: Directory structure... "
    local test_dir="/tmp/meep_test_$$"
    MEEP_DATA_DIR="$test_dir" MEEP_RESULTS_DIR="$test_dir/results" MEEP_TEMPLATES_DIR="$test_dir/templates"
    
    mkdir -p "${MEEP_RESULTS_DIR}" "${MEEP_TEMPLATES_DIR}"
    if [[ -d "$test_dir" ]] && [[ -d "$test_dir/results" ]]; then
        echo "PASSED"
        ((passed++))
        rm -rf "$test_dir"
    else
        echo "FAILED - Directory creation failed"
        ((failed++))
    fi
    
    # Test 3: Docker availability
    echo -n "Test 3: Docker available... "
    if command -v docker &> /dev/null; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Docker not available"
        ((failed++))
    fi
    
    # Test 4: Port availability check
    echo -n "Test 4: Port validation... "
    if [[ "${MEEP_PORT}" -ge 1024 ]] && [[ "${MEEP_PORT}" -le 65535 ]]; then
        echo "PASSED"
        ((passed++))
    else
        echo "FAILED - Invalid port number"
        ((failed++))
    fi
    
    # Test 5: JSON validation
    echo -n "Test 5: Runtime config... "
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        if jq empty "${SCRIPT_DIR}/config/runtime.json" 2>/dev/null; then
            echo "PASSED"
            ((passed++))
        else
            echo "FAILED - Invalid JSON"
            ((failed++))
        fi
    else
        echo "FAILED - Runtime config missing"
        ((failed++))
    fi
    
    # Summary
    echo
    echo "Unit Test Results"
    echo "================="
    echo "Passed: $passed"
    echo "Failed: $failed"
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Run all test phases
#######################################
meep::test_all() {
    echo "Running all MEEP tests..."
    echo "========================"
    echo
    
    local total_passed=0
    local total_failed=0
    
    # Run smoke tests
    if meep::test_smoke; then
        ((total_passed++))
    else
        ((total_failed++))
    fi
    
    echo
    
    # Run integration tests
    if meep::test_integration; then
        ((total_passed++))
    else
        ((total_failed++))
    fi
    
    echo
    
    # Run unit tests
    if meep::test_unit; then
        ((total_passed++))
    else
        ((total_failed++))
    fi
    
    # Overall summary
    echo
    echo "Overall Test Results"
    echo "===================="
    echo "Test Suites Passed: $total_passed/3"
    echo "Test Suites Failed: $total_failed/3"
    
    [[ $total_failed -eq 0 ]] && return 0 || return 1
}

# Export test functions
export -f meep::test
export -f meep::test_smoke
export -f meep::test_integration
export -f meep::test_unit
export -f meep::test_all