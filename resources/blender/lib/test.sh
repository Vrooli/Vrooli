#!/bin/bash
# Blender test functions

# Test the Blender installation and functionality
blender::test() {
    local format="${1:-text}"
    local test_results=()
    local test_count=0
    local pass_count=0
    local last_test_time
    
    # Get last test timestamp if it exists
    local test_timestamp_file="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}/.last_test"
    if [[ -f "$test_timestamp_file" ]]; then
        last_test_time=$(cat "$test_timestamp_file")
    else
        last_test_time="never"
    fi
    
    # Test 1: Check Blender installation (native or Docker)
    test_count=$((test_count + 1))
    if command -v blender >/dev/null 2>&1; then
        test_results+=("✅ Blender binary found (native)")
        pass_count=$((pass_count + 1))
    elif docker inspect vrooli-blender >/dev/null 2>&1; then
        test_results+=("✅ Blender found (Docker)")
        pass_count=$((pass_count + 1))
    else
        test_results+=("❌ Blender not found")
    fi
    
    # Test 2: Check Blender version
    test_count=$((test_count + 1))
    local blender_cmd="blender"
    if ! command -v blender >/dev/null 2>&1 && docker inspect vrooli-blender >/dev/null 2>&1; then
        blender_cmd="docker exec vrooli-blender blender"
    fi
    
    if $blender_cmd --version 2>&1 | grep -q "Blender"; then
        local version=$($blender_cmd --version 2>&1 | grep "^Blender" | grep -oP 'Blender \K[0-9.]+' || echo "unknown")
        test_results+=("✅ Blender version: $version")
        pass_count=$((pass_count + 1))
    else
        test_results+=("❌ Cannot get Blender version")
    fi
    
    # Test 3: Check Python API
    test_count=$((test_count + 1))
    local test_python_script="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}/temp/test_api.py"
    mkdir -p "${test_python_script%/*}"
    cat > "$test_python_script" << 'EOF'
import bpy
import sys
print("BLENDER_TEST_SUCCESS")
sys.exit(0)
EOF
    
    # Use correct path for Docker vs native
    local python_path="$test_python_script"
    if [[ "$blender_cmd" == *"docker"* ]]; then
        python_path="/config/temp/test_api.py"
    fi
    if timeout 10 $blender_cmd --background --python "$python_path" 2>/dev/null | grep -q "BLENDER_TEST_SUCCESS"; then
        test_results+=("✅ Python API working")
        pass_count=$((pass_count + 1))
    else
        test_results+=("❌ Python API not working")
    fi
    # Don't remove yet, will cleanup at end
    
    # Test 4: Check data directory exists and is writable
    test_count=$((test_count + 1))
    local data_dir="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}"
    if [[ -d "$data_dir" && -w "$data_dir" ]]; then
        test_results+=("✅ Data directory writable")
        pass_count=$((pass_count + 1))
    else
        test_results+=("❌ Data directory not writable")
    fi
    
    # Test 5: Test rendering capability
    test_count=$((test_count + 1))
    local test_render_script="${BLENDER_DATA_DIR:-${HOME}/.vrooli/blender}/temp/test_render.py"
    cat > "$test_render_script" << 'EOF'
import bpy
# Create a simple cube scene
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0))
# Set render engine to Cycles (CPU)
bpy.context.scene.render.engine = 'CYCLES'
bpy.context.scene.cycles.device = 'CPU'
# Set small render size for quick test
bpy.context.scene.render.resolution_x = 100
bpy.context.scene.render.resolution_y = 100
# Render test (without saving)
try:
    bpy.ops.render.render(write_still=False)
    print("RENDER_TEST_SUCCESS")
except Exception as e:
    print("RENDER_TEST_FAILED: " + str(e))
EOF
    
    # Use correct path for Docker vs native
    local render_path="$test_render_script"
    if [[ "$blender_cmd" == *"docker"* ]]; then
        render_path="/config/temp/test_render.py"
    fi
    # Run the render test with visible output for debugging
    local render_output
    render_output=$(timeout 30 $blender_cmd --background --python "$render_path" 2>&1)
    if echo "$render_output" | grep -q "RENDER_TEST_SUCCESS"; then
        test_results+=("✅ Rendering capability working")
        pass_count=$((pass_count + 1))
    else
        test_results+=("❌ Rendering capability not working")
    fi
    
    # Cleanup test files
    rm -f "$test_python_script" "$test_render_script"
    
    # Test 6: Check injected scripts directory
    test_count=$((test_count + 1))
    local scripts_dir="${data_dir}/scripts"
    if [[ -d "$scripts_dir" ]]; then
        local script_count=$(find "$scripts_dir" -name "*.py" 2>/dev/null | wc -l)
        test_results+=("✅ Scripts directory exists ($script_count scripts)")
        pass_count=$((pass_count + 1))
    else
        test_results+=("❌ Scripts directory missing")
    fi
    
    # Update test timestamp
    date -Iseconds > "$test_timestamp_file"
    
    # Calculate test summary
    local success_rate=$((pass_count * 100 / test_count))
    local overall_status="healthy"
    if [[ $success_rate -lt 50 ]]; then
        overall_status="unhealthy"
    elif [[ $success_rate -lt 100 ]]; then
        overall_status="degraded"
    fi
    
    # Output results based on format
    local first=true
    if [[ "$format" == "json" ]]; then
        echo "{"
        echo "  \"test_status\": \"$overall_status\","
        echo "  \"tests_total\": $test_count,"
        echo "  \"tests_passed\": $pass_count,"
        echo "  \"success_rate\": $success_rate,"
        echo "  \"last_test\": \"$(date -Iseconds)\","
        echo "  \"previous_test\": \"$last_test_time\","
        echo "  \"results\": ["
        first=true
        for result in "${test_results[@]}"; do
            if [[ "$first" == "false" ]]; then
                echo ","
            fi
            echo -n "    \"$result\""
            first=false
        done
        echo ""
        echo "  ]"
        echo "}"
    else
        echo "[HEADER]  🧪 Blender Test Results"
        echo ""
        echo "[INFO]    📊 Summary:"
        echo "[INFO]       Tests Run: $test_count"
        echo "[INFO]       Tests Passed: $pass_count"
        echo "[INFO]       Success Rate: ${success_rate}%"
        echo "[INFO]       Overall Status: $overall_status"
        echo "[INFO]       Last Test: $last_test_time"
        echo ""
        echo "[INFO]    📋 Test Results:"
        for result in "${test_results[@]}"; do
            echo "[INFO]       $result"
        done
    fi
    
    # Return non-zero if not all tests passed
    [[ $pass_count -eq $test_count ]] && return 0 || return 1
}

# Run tests and update status cache
blender::run_tests() {
    blender::test "$@"
}