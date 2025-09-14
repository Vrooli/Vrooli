#!/bin/bash

# PyBullet Test Functions
# Implements v2.0 universal contract test requirements

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$RESOURCE_DIR/lib/core.sh"

# Handle test subcommands
handle_test() {
    local test_type="${1:-all}"
    shift || true
    
    case "$test_type" in
        smoke)
            test_smoke "$@"
            ;;
        integration)
            test_integration "$@"
            ;;
        unit)
            test_unit "$@"
            ;;
        all)
            test_all "$@"
            ;;
        *)
            echo "Error: Unknown test type '$test_type'"
            echo "Valid types: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Smoke test - quick health check (<30s)
test_smoke() {
    echo "Running PyBullet smoke tests..."
    echo "================================"
    
    local failures=0
    
    # Test 1: Check Python environment
    echo -n "1. Python environment... "
    if [[ -d "$VENV_DIR" ]] && [[ -f "$VENV_DIR/bin/python" ]]; then
        echo "✓"
    else
        echo "✗ (virtual environment not found)"
        ((failures++))
    fi
    
    # Test 2: Check PyBullet import
    echo -n "2. PyBullet import... "
    if source "$VENV_DIR/bin/activate" && python -c "import pybullet" 2>/dev/null; then
        echo "✓"
    else
        echo "✗ (import failed)"
        ((failures++))
    fi
    
    # Test 3: Check API server health
    echo -n "3. API server health... "
    if timeout 5 curl -sf "http://localhost:$PYBULLET_PORT/health" > /dev/null 2>&1; then
        echo "✓"
    else
        echo "✗ (health check failed)"
        ((failures++))
    fi
    
    # Test 4: Check basic physics simulation
    echo -n "4. Basic physics test... "
    if source "$VENV_DIR/bin/activate" && python -c "
import pybullet as p
client = p.connect(p.DIRECT)
p.setGravity(0, 0, -9.81)
p.stepSimulation()
p.disconnect()
print('OK')
" 2>/dev/null | grep -q "OK"; then
        echo "✓"
    else
        echo "✗ (simulation failed)"
        ((failures++))
    fi
    
    echo "================================"
    if [[ $failures -eq 0 ]]; then
        echo "Smoke tests: PASSED"
        return 0
    else
        echo "Smoke tests: FAILED ($failures failures)"
        return 1
    fi
}

# Integration test - full functionality
test_integration() {
    echo "Running PyBullet integration tests..."
    echo "===================================="
    
    local failures=0
    
    # Test 1: API create simulation
    echo -n "1. Create simulation via API... "
    local response=$(curl -sf -X POST "http://localhost:$PYBULLET_PORT/simulation/create" \
        -H "Content-Type: application/json" \
        -d '{"name": "test_sim", "gravity": [0, 0, -9.81]}' 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q '"status":"created"'; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 2: Spawn object
    echo -n "2. Spawn object in simulation... "
    response=$(curl -sf -X POST "http://localhost:$PYBULLET_PORT/simulation/test_sim/spawn" \
        -H "Content-Type: application/json" \
        -d '{"shape": "box", "position": [0, 0, 2]}' 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q '"body_id"'; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 3: Step simulation
    echo -n "3. Step simulation... "
    response=$(curl -sf -X POST "http://localhost:$PYBULLET_PORT/simulation/test_sim/step" \
        -H "Content-Type: application/json" \
        -d '{"steps": 10}' 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q '"status":"stepped"'; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 4: Get state
    echo -n "4. Get simulation state... "
    response=$(curl -sf "http://localhost:$PYBULLET_PORT/simulation/test_sim/state" 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q '"num_bodies"'; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 5: Destroy simulation
    echo -n "5. Destroy simulation... "
    response=$(curl -sf -X DELETE "http://localhost:$PYBULLET_PORT/simulation/test_sim" 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q '"status":"destroyed"'; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 6: Run example simulation
    echo -n "6. Run pendulum example... "
    if [[ -f "$EXAMPLES_DIR/pendulum.py" ]]; then
        if source "$VENV_DIR/bin/activate" && python "$EXAMPLES_DIR/pendulum.py" 2>/dev/null | grep -q "Simulation complete"; then
            echo "✓"
        else
            echo "✗"
            ((failures++))
        fi
    else
        echo "✗ (example not found)"
        ((failures++))
    fi
    
    echo "===================================="
    if [[ $failures -eq 0 ]]; then
        echo "Integration tests: PASSED"
        return 0
    else
        echo "Integration tests: FAILED ($failures failures)"
        return 1
    fi
}

# Unit test - test library functions
test_unit() {
    echo "Running PyBullet unit tests..."
    echo "=============================="
    
    local failures=0
    
    # Test 1: Port allocation
    echo -n "1. Port allocation... "
    if [[ -n "$PYBULLET_PORT" ]] && [[ "$PYBULLET_PORT" -gt 1024 ]]; then
        echo "✓ (port: $PYBULLET_PORT)"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 2: Configuration loading
    echo -n "2. Configuration files... "
    if [[ -f "$CONFIG_DIR/defaults.sh" ]] && [[ -f "$CONFIG_DIR/runtime.json" ]]; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 3: PyBullet version check
    echo -n "3. PyBullet version... "
    if source "$VENV_DIR/bin/activate" && python -c "
import pybullet as p
version = p.getVersionInfo()
if version: print('OK')
" 2>/dev/null | grep -q "OK"; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    # Test 4: URDF loading capability
    echo -n "4. URDF loading... "
    if source "$VENV_DIR/bin/activate" && python -c "
import pybullet as p
import pybullet_data
p.connect(p.DIRECT)
p.setAdditionalSearchPath(pybullet_data.getDataPath())
try:
    plane = p.loadURDF('plane.urdf')
    print('OK' if plane >= 0 else 'FAIL')
except:
    print('FAIL')
finally:
    p.disconnect()
" 2>/dev/null | grep -q "OK"; then
        echo "✓"
    else
        echo "✗"
        ((failures++))
    fi
    
    echo "=============================="
    if [[ $failures -eq 0 ]]; then
        echo "Unit tests: PASSED"
        return 0
    else
        echo "Unit tests: FAILED ($failures failures)"
        return 1
    fi
}

# Run all tests
test_all() {
    local total_failures=0
    
    echo "Running all PyBullet tests..."
    echo "============================="
    echo
    
    # Run each test type
    if ! test_smoke; then
        ((total_failures++))
    fi
    echo
    
    if ! test_unit; then
        ((total_failures++))
    fi
    echo
    
    if ! test_integration; then
        ((total_failures++))
    fi
    
    echo
    echo "============================="
    if [[ $total_failures -eq 0 ]]; then
        echo "All tests: PASSED"
        return 0
    else
        echo "All tests: FAILED ($total_failures test suites failed)"
        return 1
    fi
}