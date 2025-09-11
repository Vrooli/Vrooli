#!/usr/bin/env bash
################################################################################
# Blender Smoke Test - Quick Health Validation
# 
# Validates basic Blender functionality within 30 seconds
# Tests: health check, basic service availability, critical features
#
################################################################################

set -euo pipefail

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and Blender libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/docker.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_start() {
    echo "[TEST] $1"
    ((TESTS_RUN++))
}

test_pass() {
    echo "  ✓ $1"
    ((TESTS_PASSED++))
}

test_fail() {
    echo "  ✗ $1"
    ((TESTS_FAILED++))
}

# Test 1: Check Blender service is running
test_service_running() {
    test_start "Blender service status"
    
    if docker ps --format "table {{.Names}}" | grep -q "${BLENDER_CONTAINER_NAME}"; then
        test_pass "Blender container is running"
    else
        # Try native installation
        if command -v blender &>/dev/null; then
            test_pass "Blender native installation found"
        else
            test_fail "Blender is not running (neither Docker nor native)"
            return 1
        fi
    fi
}

# Test 2: Quick health check
test_health_check() {
    test_start "Health check response"
    
    # For Blender, we check if we can execute a simple command
    local test_script=$(mktemp /tmp/blender_health_XXXXX.py)
    cat > "$test_script" << 'EOF'
import bpy
import sys
print("HEALTH_OK")
sys.exit(0)
EOF
    
    if timeout 5 blender::run_script "$test_script" &>/dev/null; then
        test_pass "Blender responds to commands"
        rm -f "$test_script"
    else
        test_fail "Blender not responding"
        rm -f "$test_script"
        return 1
    fi
}

# Test 3: Check required directories exist
test_directories() {
    test_start "Required directories"
    
    local all_exist=true
    for dir in "$BLENDER_DATA_DIR" "$BLENDER_SCRIPTS_DIR" "$BLENDER_OUTPUT_DIR"; do
        if [[ -d "$dir" ]]; then
            test_pass "Directory exists: $dir"
        else
            test_fail "Directory missing: $dir"
            all_exist=false
        fi
    done
    
    [[ "$all_exist" == true ]]
}

# Test 4: Verify Python API accessibility
test_python_api() {
    test_start "Python API accessibility"
    
    local test_script=$(mktemp /tmp/blender_api_XXXXX.py)
    cat > "$test_script" << 'EOF'
import bpy
# Test basic API modules
modules = ['bpy.data', 'bpy.ops', 'bpy.context']
for module in modules:
    if not eval(f"hasattr({module.split('.')[0]}, '{module.split('.')[1]}')"):
        print(f"ERROR: {module} not accessible")
        exit(1)
print("API_OK")
EOF
    
    if timeout 5 blender::run_script "$test_script" 2>/dev/null | grep -q "API_OK"; then
        test_pass "Python API is accessible"
        rm -f "$test_script"
    else
        test_fail "Python API not accessible"
        rm -f "$test_script"
        return 1
    fi
}

# Test 5: Quick render capability test
test_render_capability() {
    test_start "Render capability"
    
    local test_script=$(mktemp /tmp/blender_render_XXXXX.py)
    cat > "$test_script" << 'EOF'
import bpy
# Create a simple scene
bpy.ops.mesh.primitive_cube_add()
# Set render settings for speed
bpy.context.scene.render.engine = 'BLENDER_WORKBENCH'
bpy.context.scene.render.resolution_x = 32
bpy.context.scene.render.resolution_y = 32
# Test render (don't save)
bpy.ops.render.render()
print("RENDER_OK")
EOF
    
    if timeout 10 blender::run_script "$test_script" 2>/dev/null | grep -q "RENDER_OK"; then
        test_pass "Render engine works"
        rm -f "$test_script"
    else
        test_fail "Render engine not working"
        rm -f "$test_script"
        return 1
    fi
}

# Main test execution
main() {
    echo "======================================"
    echo "Blender Smoke Tests"
    echo "======================================"
    
    # Run all tests
    test_service_running || true
    test_health_check || true
    test_directories || true
    test_python_api || true
    test_render_capability || true
    
    # Summary
    echo "======================================"
    echo "Tests Run: $TESTS_RUN"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo "======================================"
    
    # Return appropriate exit code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "[SUCCESS] All smoke tests passed"
        exit 0
    else
        echo "[FAILURE] $TESTS_FAILED test(s) failed"
        exit 1
    fi
}

main "$@"