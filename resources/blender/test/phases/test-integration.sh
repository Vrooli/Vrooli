#!/usr/bin/env bash
################################################################################
# Blender Integration Test - End-to-End Functionality
# 
# Tests complete workflows and integration points
# Includes: rendering, physics, export, script injection
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
source "${RESOURCE_DIR}/lib/inject.sh"
source "${RESOURCE_DIR}/lib/test.sh"

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

# Test 1: Complete render workflow
test_render_workflow() {
    test_start "Complete render workflow"
    
    # Test rendering with example script
    local render_script="${RESOURCE_DIR}/examples/simple_cube_render.py"
    if [[ -f "$render_script" ]]; then
        if blender::run_script "$render_script"; then
            # Check if output was created
            if ls "${BLENDER_OUTPUT_DIR}"/cube_render*.png &>/dev/null; then
                test_pass "Render workflow completed with output"
            else
                test_fail "Render completed but no output generated"
            fi
        else
            test_fail "Render workflow failed"
        fi
    else
        test_fail "Render example script not found"
    fi
}

# Test 2: Physics simulation workflow
test_physics_workflow() {
    test_start "Physics simulation workflow"
    
    # Run physics test from test.sh
    if blender::test::physics; then
        test_pass "Physics simulation works"
    else
        test_fail "Physics simulation failed"
    fi
}

# Test 3: Script injection and execution
test_script_injection() {
    test_start "Script injection and execution"
    
    # Create a test script
    local test_script=$(mktemp /tmp/blender_inject_XXXXX.py)
    cat > "$test_script" << 'EOF'
import bpy
import json
# Create test data
data = {"test": "injection_successful", "frames": 10}
# Save to output
with open("/tmp/injection_test.json", "w") as f:
    json.dump(data, f)
print("INJECTION_TEST_COMPLETE")
EOF
    
    # Inject the script
    local script_name="test_injection_$(date +%s).py"
    if blender::inject "$test_script" "$script_name"; then
        test_pass "Script injected successfully"
        
        # Run the injected script
        if blender::run_script "${BLENDER_SCRIPTS_DIR}/$script_name" 2>/dev/null | grep -q "INJECTION_TEST_COMPLETE"; then
            test_pass "Injected script executed"
            
            # Check output
            if [[ -f "/tmp/injection_test.json" ]]; then
                test_pass "Script output generated"
                rm -f "/tmp/injection_test.json"
            else
                test_fail "Script output not found"
            fi
        else
            test_fail "Injected script execution failed"
        fi
        
        # Clean up
        blender::remove_injected "$script_name" &>/dev/null || true
    else
        test_fail "Script injection failed"
    fi
    
    rm -f "$test_script"
}

# Test 4: Export capabilities
test_export_formats() {
    test_start "Export format support"
    
    local test_script=$(mktemp /tmp/blender_export_XXXXX.py)
    cat > "$test_script" << 'EOF'
import bpy
import os
# Create a simple object
bpy.ops.mesh.primitive_cube_add()
# Test different export formats
formats = {
    ".obj": "bpy.ops.export_scene.obj",
    ".stl": "bpy.ops.export_mesh.stl",
    ".fbx": "bpy.ops.export_scene.fbx"
}
success_count = 0
for ext, op in formats.items():
    try:
        filepath = f"/tmp/export_test{ext}"
        eval(f"{op}(filepath='{filepath}')")
        if os.path.exists(filepath):
            success_count += 1
            os.remove(filepath)
    except:
        pass
print(f"EXPORT_SUCCESS:{success_count}")
EOF
    
    local result=$(timeout 15 blender::run_script "$test_script" 2>/dev/null | grep "EXPORT_SUCCESS" | cut -d: -f2)
    if [[ -n "$result" && "$result" -gt 0 ]]; then
        test_pass "Export formats working ($result/3)"
    else
        test_fail "Export formats not working"
    fi
    
    rm -f "$test_script"
}

# Test 5: Batch processing capability
test_batch_processing() {
    test_start "Batch processing capability"
    
    local batch_script="${RESOURCE_DIR}/examples/batch_processing.py"
    if [[ -f "$batch_script" ]]; then
        # Create test input files
        mkdir -p /tmp/blender_batch_test
        for i in {1..3}; do
            echo "Test file $i" > "/tmp/blender_batch_test/file_$i.txt"
        done
        
        if timeout 20 blender::run_script "$batch_script" &>/dev/null; then
            test_pass "Batch processing completed"
        else
            test_fail "Batch processing failed"
        fi
        
        # Clean up
        rm -rf /tmp/blender_batch_test
    else
        test_fail "Batch processing example not found"
    fi
}

# Test 6: Resource limits and cleanup
test_resource_management() {
    test_start "Resource management"
    
    # Check memory usage is reasonable
    if docker ps -q -f name="${BLENDER_CONTAINER_NAME}" &>/dev/null; then
        local mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${BLENDER_CONTAINER_NAME}" | cut -d/ -f1)
        test_pass "Memory usage: $mem_usage"
    else
        test_pass "Native installation (resource limits managed by OS)"
    fi
    
    # Check cleanup of temporary files
    local temp_count=$(find "${BLENDER_DATA_DIR}/temp" -type f 2>/dev/null | wc -l)
    if [[ $temp_count -lt 100 ]]; then
        test_pass "Temp file cleanup working ($temp_count files)"
    else
        test_fail "Too many temp files ($temp_count)"
    fi
}

# Main test execution
main() {
    echo "======================================"
    echo "Blender Integration Tests"
    echo "======================================"
    
    # Run all tests
    test_render_workflow || true
    test_physics_workflow || true
    test_script_injection || true
    test_export_formats || true
    test_batch_processing || true
    test_resource_management || true
    
    # Summary
    echo "======================================"
    echo "Tests Run: $TESTS_RUN"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    echo "======================================"
    
    # Return appropriate exit code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "[SUCCESS] All integration tests passed"
        exit 0
    else
        echo "[FAILURE] $TESTS_FAILED test(s) failed"
        exit 1
    fi
}

main "$@"