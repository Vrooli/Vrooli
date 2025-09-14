#!/bin/bash
# Test physics optimization features

set -euo pipefail

# Get absolute path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
BLENDER_TEST_DIR="${APP_ROOT}/resources/blender/test/phases"

# Source test utilities (skip if not found)
[[ -f "${APP_ROOT}/scripts/lib/utils/test.sh" ]] && source "${APP_ROOT}/scripts/lib/utils/test.sh"
source "${APP_ROOT}/resources/blender/lib/core.sh"

# Test configuration
export TEST_OUTPUT_DIR="/tmp/blender_physics_test_$$"
mkdir -p "$TEST_OUTPUT_DIR"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -n "[TEST] $test_name... "
    
    if eval "$test_command" &>/dev/null; then
        echo "✅ PASSED"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "❌ FAILED"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test GPU optimization module
test_gpu_optimization() {
    local script="${APP_ROOT}/resources/blender/examples/physics_gpu_optimized.py"
    local output="${TEST_OUTPUT_DIR}/gpu_optimization.json"
    
    if [[ ! -f "$script" ]]; then
        return 1
    fi
    
    # Run GPU optimization test
    timeout 60 blender::run_script "$script" --background --python-expr "
import sys
sys.path.append('${APP_ROOT}/resources/blender/examples')
from physics_gpu_optimized import GPUPhysicsOptimizer
import json

optimizer = GPUPhysicsOptimizer()
optimizer.optimize_scene_for_physics()
result = optimizer.setup_adaptive_physics('fast')

# Save result
with open('$output', 'w') as f:
    json.dump({
        'gpu_enabled': optimizer.stats['gpu_enabled'],
        'optimization_level': optimizer.stats['optimization_level'],
        'success': True
    }, f)
" 2>/dev/null
    
    # Check output exists
    [[ -f "$output" ]] && grep -q '"success": true' "$output"
}

# Test cache streaming
test_cache_streaming() {
    local script="${APP_ROOT}/resources/blender/examples/physics_cache_streaming.py"
    local output="${TEST_OUTPUT_DIR}/cache_streaming.json"
    
    if [[ ! -f "$script" ]]; then
        return 1
    fi
    
    # Run cache streaming test
    timeout 60 blender::run_script "$script" --background --python-expr "
import sys
import tempfile
sys.path.append('${APP_ROOT}/resources/blender/examples')
from physics_cache_streaming import PhysicsCacheManager
import json

# Test cache manager
cache_mgr = PhysicsCacheManager(tempfile.mkdtemp())
cache_mgr.configure_cache_settings('LIGHT', True)

# Simple test scene
import bpy
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 5))
bpy.ops.rigidbody.object_add()

# Test streaming (small test)
chunks = cache_mgr.stream_cache_to_disk(1, 10, 5)
metrics = cache_mgr.analyze_cache_performance()

# Save result
with open('$output', 'w') as f:
    json.dump({
        'chunks_created': len(chunks),
        'cache_size_mb': metrics['total_size_mb'],
        'success': len(chunks) > 0
    }, f)

# Cleanup
cache_mgr.cleanup_cache()
" 2>/dev/null
    
    # Check output
    [[ -f "$output" ]] && grep -q '"success": true' "$output"
}

# Test adaptive LOD
test_adaptive_lod() {
    local script="${APP_ROOT}/resources/blender/examples/physics_cache_streaming.py"
    local output="${TEST_OUTPUT_DIR}/adaptive_lod.json"
    
    # Run LOD test
    timeout 30 blender::run_script "$script" --background --python-expr "
import sys
sys.path.append('${APP_ROOT}/resources/blender/examples')
from physics_cache_streaming import AdaptivePhysicsLOD
import bpy
import json

# Test LOD system
lod_system = AdaptivePhysicsLOD()

# Create test object
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0))
cube = bpy.context.active_object

# Create LOD group
lods = lod_system.create_lod_group(cube, levels=3)

# Test LOD switching
success = lod_system.switch_lod_level(f'{cube.name}_LOD_Group', 1)

# Save result
with open('$output', 'w') as f:
    json.dump({
        'lod_count': len(lods),
        'switch_success': success,
        'success': len(lods) == 3 and success
    }, f)
" 2>/dev/null
    
    # Check output
    [[ -f "$output" ]] && grep -q '"success": true' "$output"
}

# Test performance benchmarking
test_performance_benchmark() {
    local output="${TEST_OUTPUT_DIR}/benchmark.json"
    
    # Run simple benchmark
    timeout 30 blender::run_script "" --background --python-expr "
import bpy
import time
import json

# Clear scene
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete()

# Create simple physics scene
bpy.ops.mesh.primitive_plane_add(size=10, location=(0, 0, 0))
ground = bpy.context.active_object
bpy.ops.rigidbody.object_add()
ground.rigid_body.type = 'PASSIVE'

# Add test objects
object_count = 20
for i in range(object_count):
    bpy.ops.mesh.primitive_cube_add(
        size=0.5,
        location=(0, 0, 2 + i * 0.5)
    )
    obj = bpy.context.active_object
    bpy.ops.rigidbody.object_add()

# Benchmark baking
start = time.time()
scene = bpy.context.scene
if not scene.rigidbody_world:
    bpy.ops.rigidbody.world_add()

rbw = scene.rigidbody_world
rbw.point_cache.frame_start = 1
rbw.point_cache.frame_end = 10

bpy.ops.ptcache.bake(bake=True)
elapsed = time.time() - start

# Save benchmark
with open('$output', 'w') as f:
    json.dump({
        'object_count': object_count,
        'frames': 10,
        'time': elapsed,
        'fps': 10 / elapsed if elapsed > 0 else 0,
        'success': elapsed > 0
    }, f)
" 2>/dev/null
    
    # Check benchmark completed
    [[ -f "$output" ]] && grep -q '"success": true' "$output"
}

# Test collision shape optimization
test_collision_optimization() {
    local output="${TEST_OUTPUT_DIR}/collision_opt.json"
    
    # Run collision optimization test
    timeout 30 blender::run_script "" --background --python-expr "
import bpy
import json

# Create test objects with different complexities
objects = []

# Simple cube
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 1))
cube = bpy.context.active_object
bpy.ops.rigidbody.object_add()
cube.rigid_body.collision_shape = 'BOX'
objects.append(cube)

# Complex sphere
bpy.ops.mesh.primitive_uv_sphere_add(segments=32, ring_count=16, location=(2, 0, 1))
sphere = bpy.context.active_object
bpy.ops.rigidbody.object_add()
sphere.rigid_body.collision_shape = 'SPHERE'
objects.append(sphere)

# Convex hull object
bpy.ops.mesh.primitive_cylinder_add(location=(4, 0, 1))
cylinder = bpy.context.active_object
bpy.ops.rigidbody.object_add()
cylinder.rigid_body.collision_shape = 'CONVEX_HULL'
objects.append(cylinder)

# Verify collision shapes
success = all(obj.rigid_body is not None for obj in objects)

# Save result
with open('$output', 'w') as f:
    json.dump({
        'objects_created': len(objects),
        'collision_shapes': [obj.rigid_body.collision_shape for obj in objects],
        'success': success
    }, f)
" 2>/dev/null
    
    # Check result
    [[ -f "$output" ]] && grep -q '"success": true' "$output"
}

# Test instanced physics
test_instanced_physics() {
    local output="${TEST_OUTPUT_DIR}/instanced.json"
    
    # Run instanced physics test
    timeout 30 blender::run_script "" --background --python-expr "
import bpy
import json

# Clear scene
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete()

# Create template
bpy.ops.mesh.primitive_cube_add(size=0.5, location=(0, 0, -100))
template = bpy.context.active_object

# Create instances
instances = []
for i in range(10):
    obj = template.copy()
    obj.data = template.data  # Share mesh data for efficiency
    bpy.context.collection.objects.link(obj)
    obj.location = (i * 0.7, 0, 3)
    
    # Add physics
    bpy.context.view_layer.objects.active = obj
    bpy.ops.rigidbody.object_add()
    
    instances.append(obj)

# Hide template
template.hide_set(True)

# Verify instances share mesh data
shared_data = all(obj.data == template.data for obj in instances)

# Save result
with open('$output', 'w') as f:
    json.dump({
        'instance_count': len(instances),
        'shared_mesh_data': shared_data,
        'success': len(instances) == 10 and shared_data
    }, f)
" 2>/dev/null
    
    # Check result
    [[ -f "$output" ]] && grep -q '"success": true' "$output"
}

# Main test execution
main() {
    echo "=" "=" | tr " " "=" | head -c 60 && echo
    echo "PHYSICS OPTIMIZATION TESTS"
    echo "=" "=" | tr " " "=" | head -c 60 && echo
    
    # Check Blender availability
    if ! blender::is_installed; then
        echo "[SKIP] Blender not installed"
        exit 0
    fi
    
    # Run tests
    run_test "GPU Optimization Module" "test_gpu_optimization"
    run_test "Cache Streaming" "test_cache_streaming"
    run_test "Adaptive LOD System" "test_adaptive_lod"
    run_test "Performance Benchmark" "test_performance_benchmark"
    run_test "Collision Optimization" "test_collision_optimization"
    run_test "Instanced Physics" "test_instanced_physics"
    
    # Summary
    echo
    echo "=" "=" | tr " " "=" | head -c 60 && echo
    echo "TEST SUMMARY"
    echo "=" "=" | tr " " "=" | head -c 60 && echo
    echo "Tests Run: $TESTS_RUN"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    # Cleanup
    rm -rf "$TEST_OUTPUT_DIR"
    
    # Exit code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo
        echo "✅ All physics optimization tests passed!"
        exit 0
    else
        echo
        echo "❌ Some tests failed"
        exit 1
    fi
}

# Run main
main "$@"