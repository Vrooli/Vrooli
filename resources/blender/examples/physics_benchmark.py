#!/usr/bin/env python3
"""
Physics Performance Benchmarking
Measure and validate physics simulation performance
"""

import bpy
import time
import json
import sys

def benchmark_rigid_body(object_count=100):
    """Benchmark rigid body simulation performance"""
    start_time = time.time()
    
    # Clear scene
    bpy.ops.wm.read_factory_settings(use_empty=True)
    
    # Create objects
    for i in range(object_count):
        bpy.ops.mesh.primitive_cube_add(
            location=(i % 10 * 2, i // 10 * 2, 10)
        )
        bpy.ops.rigidbody.object_add()
    
    # Create ground
    bpy.ops.mesh.primitive_plane_add(size=50, location=(0, 0, 0))
    bpy.ops.rigidbody.object_add(type='PASSIVE')
    
    # Run simulation
    scene = bpy.context.scene
    scene.frame_end = 100
    
    sim_start = time.time()
    for frame in range(1, 101):
        scene.frame_set(frame)
    sim_end = time.time()
    
    return {
        "setup_time": sim_start - start_time,
        "simulation_time": sim_end - sim_start,
        "fps": 100 / (sim_end - sim_start),
        "object_count": object_count
    }

def benchmark_soft_body(vertex_count=1000):
    """Benchmark soft body simulation performance"""
    start_time = time.time()
    
    # Clear scene
    bpy.ops.wm.read_factory_settings(use_empty=True)
    
    # Create soft body object
    bpy.ops.mesh.primitive_uv_sphere_add(
        segments=int(vertex_count ** 0.5),
        ring_count=int(vertex_count ** 0.5),
        location=(0, 0, 5)
    )
    obj = bpy.context.active_object
    
    # Add soft body modifier
    bpy.ops.object.modifier_add(type='SOFT_BODY')
    
    # Configure soft body settings
    soft_body = obj.modifiers["Softbody"].settings
    soft_body.mass = 1.0
    soft_body.friction = 0.5
    
    # Run simulation
    scene = bpy.context.scene
    scene.frame_end = 50
    
    sim_start = time.time()
    for frame in range(1, 51):
        scene.frame_set(frame)
    sim_end = time.time()
    
    return {
        "setup_time": sim_start - start_time,
        "simulation_time": sim_end - sim_start,
        "fps": 50 / (sim_end - sim_start),
        "vertex_count": len(obj.data.vertices)
    }

def benchmark_cache_performance():
    """Benchmark physics cache read/write performance"""
    import tempfile
    
    start_time = time.time()
    
    # Setup scene with rigid bodies
    benchmark_rigid_body(50)
    
    scene = bpy.context.scene
    rbw = scene.rigidbody_world
    cache = rbw.point_cache
    
    # Test cache write
    cache_write_start = time.time()
    bpy.ops.ptcache.bake_all(bake=True)
    cache_write_end = time.time()
    
    # Clear cache
    bpy.ops.ptcache.free_bake_all()
    
    # Test cache read (re-bake)
    cache_read_start = time.time()
    bpy.ops.ptcache.bake_all(bake=True)
    cache_read_end = time.time()
    
    return {
        "cache_write_time": cache_write_end - cache_write_start,
        "cache_read_time": cache_read_end - cache_read_start,
        "total_time": time.time() - start_time
    }

def main():
    """Main benchmark execution"""
    results = {
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "blender_version": bpy.app.version_string,
        "benchmarks": {}
    }
    
    # Get benchmark type from environment or default
    import os
    benchmark_type = os.environ.get('BENCHMARK_TYPE', 'all')
    
    if benchmark_type in ["all", "rigid"]:
        print("[INFO] Running rigid body benchmark...")
        results["benchmarks"]["rigid_body"] = benchmark_rigid_body(100)
        
    if benchmark_type in ["all", "soft"]:
        print("[INFO] Running soft body benchmark...")
        results["benchmarks"]["soft_body"] = benchmark_soft_body(500)
        
    if benchmark_type in ["all", "cache"]:
        print("[INFO] Running cache benchmark...")
        results["benchmarks"]["cache"] = benchmark_cache_performance()
    
    # Save results
    output_file = os.environ.get('BENCHMARK_OUTPUT', '/tmp/blender_physics_benchmark.json')
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)
    
    print("BENCHMARK_COMPLETE")
    print(f"Results saved to: {output_file}")
    
    # Print summary
    for bench_name, bench_data in results["benchmarks"].items():
        if "fps" in bench_data:
            print(f"[RESULT] {bench_name}: {bench_data['fps']:.2f} FPS")

if __name__ == "__main__":
    main()