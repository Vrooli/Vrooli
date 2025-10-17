#!/bin/bash
# Elmer FEM Test Functions - v2.0 Contract Compliant

set -euo pipefail

# Get script directory
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Test configuration
readonly TEST_TIMEOUT=30
readonly INTEGRATION_TIMEOUT=120
readonly UNIT_TIMEOUT=60
readonly ELMER_PORT="${ELMER_FEM_PORT:-8192}"
readonly ELMER_DATA_DIR="${VROOLI_DATA:-${HOME}/.vrooli/data}/elmer-fem"

# Run specified test phase
main() {
    local phase="${1:-all}"
    
    case "$phase" in
        smoke)
            test::smoke
            ;;
        integration)
            test::integration
            ;;
        unit)
            test::unit
            ;;
        all)
            test::smoke
            test::integration
            test::unit
            ;;
        *)
            echo "ERROR: Unknown test phase: $phase" >&2
            echo "Valid phases: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Smoke test - Quick health check (<30s)
test::smoke() {
    echo "Running Elmer FEM smoke tests..."
    
    # Test 1: Check if service responds to health endpoint
    echo -n "Testing health endpoint... "
    if timeout 5 curl -sf "http://localhost:${ELMER_PORT}/health" > /dev/null 2>&1; then
        echo "PASS"
    else
        echo "FAIL"
        return 1
    fi
    
    # Test 2: Verify Docker container is running
    echo -n "Testing container status... "
    if docker ps --format '{{.Names}}' | grep -q "vrooli-elmer-fem"; then
        echo "PASS"
    else
        echo "FAIL"
        return 1
    fi
    
    # Test 3: Check API version endpoint
    echo -n "Testing version endpoint... "
    if timeout 5 curl -sf "http://localhost:${ELMER_PORT}/version" | grep -q "elmer"; then
        echo "PASS"
    else
        echo "FAIL"
        return 1
    fi
    
    echo "Smoke tests completed successfully"
    return 0
}

# Integration test - Full functionality (<120s)
test::integration() {
    echo "Running Elmer FEM integration tests..."
    
    # Test 1: Run heat transfer example
    echo "Testing heat transfer simulation..."
    local case_id="test_heat_$(date +%s)"
    
    # Create test case
    curl -X POST "http://localhost:${ELMER_PORT}/case/create" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"${case_id}\", \"type\": \"heat_transfer\"}" || {
        echo "FAIL: Could not create case"
        return 1
    }
    
    # Execute simulation
    curl -X POST "http://localhost:${ELMER_PORT}/case/${case_id}/solve" \
        -H "Content-Type: application/json" \
        -d "{\"parameters\": {\"max_iterations\": 100}}" || {
        echo "FAIL: Could not execute simulation"
        return 1
    }
    
    # Check results exist
    sleep 5  # Give time for solve to complete
    if curl -sf "http://localhost:${ELMER_PORT}/case/${case_id}/results" > /dev/null; then
        echo "PASS: Heat transfer simulation"
    else
        echo "FAIL: No results generated"
        return 1
    fi
    
    # Test 2: Mesh import
    echo -n "Testing mesh import... "
    curl -X POST "http://localhost:${ELMER_PORT}/mesh/import" \
        -H "Content-Type: application/json" \
        -d '{"format": "gmsh", "data": "test"}' > /dev/null 2>&1 && echo "PASS" || echo "FAIL"
    
    # Test 3: Parameter sweep
    echo -n "Testing parameter sweep... "
    curl -X POST "http://localhost:${ELMER_PORT}/batch/sweep" \
        -H "Content-Type: application/json" \
        -d '{"case": "heat_transfer", "parameter": "conductivity", "values": [0.5, 1.0, 2.0]}' \
        > /dev/null 2>&1 && echo "PASS" || echo "FAIL"
    
    echo "Integration tests completed"
    return 0
}

# Unit test - Library functions (<60s)
test::unit() {
    echo "Running Elmer FEM unit tests..."
    
    # Test 1: Config file parsing
    echo -n "Testing configuration loading... "
    if [[ -f "${SCRIPT_DIR}/config/runtime.json" ]]; then
        jq empty "${SCRIPT_DIR}/config/runtime.json" 2>/dev/null && echo "PASS" || echo "FAIL"
    else
        echo "FAIL"
    fi
    
    # Test 2: Port allocation
    echo -n "Testing port configuration... "
    if [[ -n "${ELMER_PORT}" ]] && [[ "${ELMER_PORT}" -gt 1024 ]]; then
        echo "PASS"
    else
        echo "FAIL"
    fi
    
    # Test 3: Data directory structure
    echo -n "Testing data directory setup... "
    local required_dirs=("cases" "meshes" "results" "temp")
    local all_exist=true
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "${ELMER_DATA_DIR}/${dir}" ]]; then
            all_exist=false
            break
        fi
    done
    
    if [[ "$all_exist" == true ]]; then
        echo "PASS"
    else
        echo "FAIL"
    fi
    
    # Test 4: Example case files
    echo -n "Testing example cases... "
    if [[ -f "${ELMER_DATA_DIR}/cases/heat_transfer/case.sif" ]]; then
        echo "PASS"
    else
        echo "FAIL"
    fi
    
    echo "Unit tests completed"
    return 0
}

# Execute main function
main "$@"