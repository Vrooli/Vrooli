#!/usr/bin/env python3
"""
Optimized Physics Simulation
Performance-optimized physics simulations using GPU acceleration and efficient settings
"""

import bpy
import time
import json
from mathutils import Vector

class PhysicsOptimizer:
    """Optimize Blender physics simulations for performance"""
    
    def __init__(self):
        self.stats = {
            "start_time": 0,
            "end_time": 0,
            "object_count": 0,
            "frame_count": 0,
            "bake_time": 0,
            "render_time": 0
        }
    
    def clear_scene(self):
        """Efficiently clear the scene"""
        # Delete all mesh objects at once
        meshes = set()
        for obj in bpy.data.objects:
            if obj.type == 'MESH':
                meshes.add(obj.data)
        
        bpy.ops.object.select_all(action='SELECT')
        bpy.ops.object.delete()
        
        # Remove orphaned mesh data
        for mesh in meshes:
            if mesh.users == 0:
                bpy.data.meshes.remove(mesh)
    
    def optimize_viewport(self):
        """Optimize viewport for physics simulation"""
        # Disable viewport features not needed for physics
        for area in bpy.context.screen.areas:
            if area.type == 'VIEW_3D':
                for space in area.spaces:
                    if space.type == 'VIEW_3D':
                        # Simplify shading
                        space.shading.type = 'SOLID'
                        # Disable overlays during simulation
                        space.overlay.show_overlays = False
                        # Use bounding box display for faster viewport
                        space.shading.show_object_outline = False
    
    def setup_optimized_physics(self, quality='balanced'):
        """Setup physics with optimized settings"""
        scene = bpy.context.scene
        
        # Add rigid body world if not exists
        if not scene.rigidbody_world:
            bpy.ops.rigidbody.world_add()
        
        rbw = scene.rigidbody_world
        
        if quality == 'performance':
            # Fastest settings, lower accuracy
            rbw.substeps_per_frame = 2  # Blender 4.0+ compatible
            rbw.solver_iterations = 10
            rbw.point_cache.frame_step = 2  # Cache every 2nd frame
        elif quality == 'balanced':
            # Good balance of speed and accuracy
            rbw.substeps_per_frame = 5  # Blender 4.0+ compatible
            rbw.solver_iterations = 20
            rbw.point_cache.frame_step = 1
        elif quality == 'quality':
            # High accuracy, slower
            rbw.substeps_per_frame = 10  # Blender 4.0+ compatible
            rbw.solver_iterations = 50
            rbw.point_cache.frame_step = 1
        
        # Optimize cache settings
        rbw.point_cache.frame_start = 1
        rbw.point_cache.frame_end = 250
        rbw.point_cache.compression = 'LIGHT'  # Compress cache
        
        # Standard gravity
        scene.gravity = (0, 0, -9.81)
        
        # Set appropriate FPS
        scene.render.fps = 30 if quality == 'performance' else 60
    
    def create_optimized_mesh(self, mesh_type='cube', location=(0,0,0), subdivisions=0):
        """Create mesh with optimized settings for physics"""
        if mesh_type == 'cube':
            bpy.ops.mesh.primitive_cube_add(location=location)
        elif mesh_type == 'sphere':
            # Use lower subdivision for physics
            bpy.ops.mesh.primitive_uv_sphere_add(
                segments=8 if subdivisions == 0 else 16,
                ring_count=8 if subdivisions == 0 else 16,
                location=location
            )
        
        obj = bpy.context.active_object
        
        # Optimize mesh for physics
        if obj.type == 'MESH':
            # Apply modifiers and remove unnecessary data
            bpy.context.view_layer.objects.active = obj
            obj.select_set(True)
            
            # Remove unnecessary mesh data
            mesh = obj.data
            mesh.calc_normals_split()
            mesh.free_normals_split()
            
            # Clear UV maps and vertex colors if not needed
            mesh.uv_layers.clear()
            mesh.vertex_colors.clear()
        
        return obj
    
    def add_optimized_rigid_body(self, obj, body_type='ACTIVE', shape='CONVEX_HULL'):
        """Add rigid body with optimized collision shape"""
        obj.select_set(True)
        bpy.context.view_layer.objects.active = obj
        
        bpy.ops.rigidbody.object_add()
        rb = obj.rigid_body
        rb.type = body_type
        
        # Use optimal collision shapes
        if shape == 'AUTO':
            # Automatically choose best shape
            if 'sphere' in obj.name.lower():
                rb.collision_shape = 'SPHERE'
            elif 'cube' in obj.name.lower() or 'box' in obj.name.lower():
                rb.collision_shape = 'BOX'
            else:
                rb.collision_shape = 'CONVEX_HULL'
        else:
            rb.collision_shape = shape
        
        # Optimize collision margins
        rb.collision_margin = 0.04  # Slightly larger for stability
        
        # Enable deactivation for performance
        rb.use_deactivation = True
        rb.deactivate_linear_velocity = 0.4
        rb.deactivate_angular_velocity = 0.5
        
        return rb
    
    def create_instanced_physics_objects(self, base_object, count=100, scatter_radius=5):
        """Create many physics objects using instancing for performance"""
        import random
        
        instances = []
        
        # Create instances using dupli_groups for better performance
        for i in range(count):
            # Duplicate with linked mesh data (instancing)
            obj_copy = base_object.copy()
            obj_copy.data = base_object.data  # Share mesh data
            bpy.context.collection.objects.link(obj_copy)
            
            # Random position
            x = random.uniform(-scatter_radius, scatter_radius)
            y = random.uniform(-scatter_radius, scatter_radius)
            z = random.uniform(2, 10)
            obj_copy.location = (x, y, z)
            
            # Random rotation
            obj_copy.rotation_euler = (
                random.uniform(0, 3.14),
                random.uniform(0, 3.14),
                random.uniform(0, 3.14)
            )
            
            # Add physics
            self.add_optimized_rigid_body(obj_copy, shape='BOX')
            obj_copy.rigid_body.mass = random.uniform(0.5, 2.0)
            
            instances.append(obj_copy)
        
        return instances
    
    def batch_bake_physics(self, start_frame=1, end_frame=250):
        """Optimized physics baking with progress tracking"""
        self.stats["start_time"] = time.time()
        
        scene = bpy.context.scene
        scene.frame_set(start_frame)
        
        # Optimize baking
        rbw = scene.rigidbody_world
        rbw.point_cache.frame_start = start_frame
        rbw.point_cache.frame_end = end_frame
        
        print(f"Baking physics simulation ({start_frame}-{end_frame})...")
        
        # Use free bake for testing (doesn't lock the cache)
        bpy.ops.ptcache.free_bake_all()
        
        # Bake with progress
        bpy.ops.ptcache.bake_all(bake=True)
        
        self.stats["bake_time"] = time.time() - self.stats["start_time"]
        self.stats["frame_count"] = end_frame - start_frame + 1
        
        print(f"Baking complete in {self.stats['bake_time']:.2f} seconds")
        print(f"Performance: {self.stats['frame_count']/self.stats['bake_time']:.1f} frames/second")
    
    def render_optimized(self, output_path="/tmp/blender_optimized_physics.mp4", quality='balanced'):
        """Render with optimized settings"""
        render_start = time.time()
        
        scene = bpy.context.scene
        render = scene.render
        
        # Output settings
        render.filepath = output_path
        render.image_settings.file_format = 'FFMPEG'
        render.ffmpeg.format = 'MPEG4'
        
        if quality == 'performance':
            # Fastest rendering
            render.engine = 'BLENDER_WORKBENCH'
            render.resolution_percentage = 50
            render.ffmpeg.codec = 'H264'
            render.ffmpeg.constant_rate_factor = 'HIGH'
        elif quality == 'balanced':
            # Good quality, reasonable speed
            render.engine = 'BLENDER_EEVEE'
            scene.eevee.taa_render_samples = 16
            scene.eevee.use_motion_blur = False
            render.resolution_percentage = 75
        elif quality == 'quality':
            # Best quality
            render.engine = 'BLENDER_EEVEE'
            scene.eevee.taa_render_samples = 64
            scene.eevee.use_motion_blur = True
            render.resolution_percentage = 100
        
        # Render
        print(f"Rendering with {quality} settings...")
        bpy.ops.render.render(animation=True)
        
        self.stats["render_time"] = time.time() - render_start
        print(f"Rendering complete in {self.stats['render_time']:.2f} seconds")
    
    def export_performance_stats(self):
        """Export performance statistics"""
        self.stats["end_time"] = time.time()
        self.stats["total_time"] = self.stats["end_time"] - self.stats["start_time"]
        
        # Calculate performance metrics
        if self.stats["bake_time"] > 0:
            self.stats["physics_fps"] = self.stats["frame_count"] / self.stats["bake_time"]
        
        if self.stats["render_time"] > 0:
            self.stats["render_fps"] = self.stats["frame_count"] / self.stats["render_time"]
        
        # Save stats
        stats_path = "/tmp/blender_physics_performance.json"
        with open(stats_path, "w") as f:
            json.dump(self.stats, f, indent=2)
        
        print(f"\nPerformance stats exported to: {stats_path}")
        return self.stats

def demo_optimized_simulation():
    """Demonstrate optimized physics simulation"""
    optimizer = PhysicsOptimizer()
    
    print("=" * 50)
    print("OPTIMIZED PHYSICS SIMULATION DEMO")
    print("=" * 50)
    
    # Setup
    optimizer.clear_scene()
    optimizer.optimize_viewport()
    optimizer.setup_optimized_physics(quality='balanced')
    
    # Create ground
    ground = optimizer.create_optimized_mesh('cube', location=(0, 0, -0.5))
    ground.scale = (10, 10, 0.1)
    bpy.ops.object.transform_apply(scale=True)
    optimizer.add_optimized_rigid_body(ground, body_type='PASSIVE', shape='BOX')
    ground.name = "Ground"
    
    # Create base object for instancing
    base_cube = optimizer.create_optimized_mesh('cube', location=(0, 0, -100))
    base_cube.scale = (0.5, 0.5, 0.5)
    bpy.ops.object.transform_apply(scale=True)
    
    # Create many instances efficiently
    print("\nCreating 200 physics objects using instancing...")
    instances = optimizer.create_instanced_physics_objects(base_cube, count=200, scatter_radius=4)
    optimizer.stats["object_count"] = len(instances) + 1  # +1 for ground
    
    # Hide base object
    base_cube.hide_set(True)
    base_cube.hide_render = True
    
    # Setup camera
    bpy.ops.object.camera_add(location=(15, -15, 8), rotation=(1.1, 0, 0.785))
    bpy.context.scene.camera = bpy.context.active_object
    
    # Simple lighting
    bpy.ops.object.light_add(type='SUN', location=(5, 5, 10))
    sun = bpy.context.active_object
    sun.data.energy = 2
    
    # Bake physics
    optimizer.batch_bake_physics(start_frame=1, end_frame=150)
    
    # Render
    optimizer.render_optimized(quality='performance')
    
    # Export stats
    stats = optimizer.export_performance_stats()
    
    # Print summary
    print("\n" + "=" * 50)
    print("SIMULATION COMPLETE")
    print("=" * 50)
    print(f"Objects simulated: {stats['object_count']}")
    print(f"Frames simulated: {stats['frame_count']}")
    print(f"Physics performance: {stats.get('physics_fps', 0):.1f} fps")
    print(f"Render performance: {stats.get('render_fps', 0):.1f} fps")
    print(f"Total time: {stats.get('total_time', 0):.2f} seconds")
    
    return stats

def benchmark_physics_performance():
    """Benchmark physics performance with different settings"""
    results = []
    
    for quality in ['performance', 'balanced', 'quality']:
        print(f"\n{'='*50}")
        print(f"BENCHMARKING: {quality.upper()} MODE")
        print(f"{'='*50}")
        
        optimizer = PhysicsOptimizer()
        optimizer.clear_scene()
        optimizer.setup_optimized_physics(quality=quality)
        
        # Standard test scene
        ground = optimizer.create_optimized_mesh('cube', location=(0, 0, -0.5))
        ground.scale = (10, 10, 0.1)
        bpy.ops.object.transform_apply(scale=True)
        optimizer.add_optimized_rigid_body(ground, body_type='PASSIVE')
        
        # Create test objects
        base = optimizer.create_optimized_mesh('sphere', location=(0, 0, -100))
        instances = optimizer.create_instanced_physics_objects(base, count=100)
        base.hide_set(True)
        
        # Benchmark baking
        start = time.time()
        optimizer.batch_bake_physics(1, 100)
        bake_time = time.time() - start
        
        results.append({
            "quality": quality,
            "objects": len(instances),
            "frames": 100,
            "bake_time": bake_time,
            "fps": 100 / bake_time
        })
    
    # Export benchmark results
    with open("/tmp/blender_physics_benchmark.json", "w") as f:
        json.dump(results, f, indent=2)
    
    print("\n" + "=" * 50)
    print("BENCHMARK RESULTS")
    print("=" * 50)
    
    for result in results:
        print(f"{result['quality'].capitalize():12} - {result['fps']:.1f} fps ({result['bake_time']:.2f}s for 100 frames)")
    
    print(f"\nBenchmark data: /tmp/blender_physics_benchmark.json")

if __name__ == "__main__":
    # Run optimized demo
    demo_optimized_simulation()
    
    # Run benchmarks
    benchmark_physics_performance()