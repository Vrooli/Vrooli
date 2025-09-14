#!/usr/bin/env python3
"""
Physics Cache Streaming and Memory Optimization
Efficient caching strategies for large-scale physics simulations
"""

import bpy
import os
import json
import time
import tempfile
import shutil
from pathlib import Path

class PhysicsCacheManager:
    """Manage physics cache for optimal memory usage and performance"""
    
    def __init__(self, cache_dir=None):
        self.cache_dir = cache_dir or tempfile.mkdtemp(prefix="blender_physics_")
        self.cache_info = {
            "total_frames": 0,
            "cached_frames": 0,
            "cache_size_mb": 0,
            "compression": "LIGHT",
            "streaming_enabled": False
        }
        self.ensure_cache_dir()
    
    def ensure_cache_dir(self):
        """Ensure cache directory exists"""
        Path(self.cache_dir).mkdir(parents=True, exist_ok=True)
        print(f"[Cache] Using cache directory: {self.cache_dir}")
    
    def configure_cache_settings(self, compression="LIGHT", disk_cache=True):
        """Configure optimal cache settings"""
        scene = bpy.context.scene
        
        if not scene.rigidbody_world:
            bpy.ops.rigidbody.world_add()
        
        rbw = scene.rigidbody_world
        cache = rbw.point_cache
        
        # Set compression
        cache.compression = compression  # 'NO', 'LIGHT', 'HEAVY'
        self.cache_info["compression"] = compression
        
        # Configure disk cache
        if disk_cache:
            cache.use_disk_cache = True
            cache.use_library_path = False
            # Set custom cache path
            blend_file = bpy.data.filepath
            if blend_file:
                cache_name = Path(blend_file).stem + "_physics_cache"
                cache.name = cache_name
        
        # Optimize cache settings
        cache.use_quick_cache = True  # Faster caching
        
        print(f"[Cache] Configured: compression={compression}, disk_cache={disk_cache}")
        return cache
    
    def stream_cache_to_disk(self, start_frame=1, end_frame=250, chunk_size=50):
        """Stream physics cache to disk in chunks"""
        scene = bpy.context.scene
        rbw = scene.rigidbody_world
        cache = rbw.point_cache
        
        total_chunks = (end_frame - start_frame) // chunk_size + 1
        cached_chunks = []
        
        print(f"[Stream] Starting cache streaming ({total_chunks} chunks)")
        
        for chunk_idx in range(total_chunks):
            chunk_start = start_frame + chunk_idx * chunk_size
            chunk_end = min(chunk_start + chunk_size - 1, end_frame)
            
            # Set chunk range
            cache.frame_start = chunk_start
            cache.frame_end = chunk_end
            
            # Bake chunk
            print(f"[Stream] Caching chunk {chunk_idx+1}/{total_chunks} (frames {chunk_start}-{chunk_end})")
            bpy.ops.ptcache.bake(bake=True)
            
            # Save chunk info
            chunk_file = os.path.join(self.cache_dir, f"chunk_{chunk_idx:04d}.cache")
            cached_chunks.append({
                "index": chunk_idx,
                "start": chunk_start,
                "end": chunk_end,
                "file": chunk_file
            })
            
            # Free memory after each chunk
            if chunk_idx < total_chunks - 1:
                bpy.ops.ptcache.free_bake()
        
        self.cache_info["streaming_enabled"] = True
        self.cache_info["total_frames"] = end_frame - start_frame + 1
        self.cache_info["cached_frames"] = len(cached_chunks) * chunk_size
        
        # Save cache manifest
        manifest_path = os.path.join(self.cache_dir, "cache_manifest.json")
        with open(manifest_path, "w") as f:
            json.dump({
                "chunks": cached_chunks,
                "info": self.cache_info
            }, f, indent=2)
        
        print(f"[Stream] Cache streaming complete. Manifest: {manifest_path}")
        return cached_chunks
    
    def load_cache_chunk(self, chunk_index):
        """Load specific cache chunk from disk"""
        manifest_path = os.path.join(self.cache_dir, "cache_manifest.json")
        
        if not os.path.exists(manifest_path):
            print("[Cache] No cache manifest found")
            return False
        
        with open(manifest_path, "r") as f:
            manifest = json.load(f)
        
        if chunk_index >= len(manifest["chunks"]):
            print(f"[Cache] Invalid chunk index: {chunk_index}")
            return False
        
        chunk = manifest["chunks"][chunk_index]
        
        # Set frame range for chunk
        scene = bpy.context.scene
        rbw = scene.rigidbody_world
        cache = rbw.point_cache
        
        cache.frame_start = chunk["start"]
        cache.frame_end = chunk["end"]
        
        print(f"[Cache] Loaded chunk {chunk_index} (frames {chunk['start']}-{chunk['end']})")
        return True
    
    def optimize_memory_usage(self):
        """Optimize memory usage for physics simulation"""
        # Free unused data blocks
        for block in bpy.data.meshes:
            if block.users == 0:
                bpy.data.meshes.remove(block)
        
        for block in bpy.data.materials:
            if block.users == 0:
                bpy.data.materials.remove(block)
        
        for block in bpy.data.textures:
            if block.users == 0:
                bpy.data.textures.remove(block)
        
        for block in bpy.data.images:
            if block.users == 0:
                bpy.data.images.remove(block)
        
        # Purge orphaned data
        bpy.ops.outliner.orphans_purge(do_recursive=True)
        
        print("[Memory] Optimized memory usage")
    
    def create_cache_preview(self, output_path="/tmp/cache_preview.mp4"):
        """Create preview of cached physics simulation"""
        scene = bpy.context.scene
        
        # Setup render for preview
        render = scene.render
        render.filepath = output_path
        render.image_settings.file_format = 'FFMPEG'
        render.ffmpeg.format = 'MPEG4'
        render.ffmpeg.codec = 'H264'
        render.ffmpeg.constant_rate_factor = 'MEDIUM'
        
        # Use viewport render for speed
        render.engine = 'BLENDER_WORKBENCH'
        render.resolution_percentage = 50
        
        # Render animation
        print(f"[Preview] Rendering cache preview to {output_path}")
        bpy.ops.render.render(animation=True)
        
        return output_path
    
    def analyze_cache_performance(self):
        """Analyze cache performance metrics"""
        metrics = {
            "cache_directory": self.cache_dir,
            "total_size_mb": 0,
            "file_count": 0,
            "average_chunk_size_mb": 0,
            "compression_ratio": 0
        }
        
        # Calculate cache size
        total_size = 0
        file_count = 0
        
        for root, dirs, files in os.walk(self.cache_dir):
            for file in files:
                file_path = os.path.join(root, file)
                total_size += os.path.getsize(file_path)
                file_count += 1
        
        metrics["total_size_mb"] = total_size / (1024 * 1024)
        metrics["file_count"] = file_count
        
        if file_count > 0:
            metrics["average_chunk_size_mb"] = metrics["total_size_mb"] / file_count
        
        # Estimate compression ratio
        if self.cache_info["compression"] == "HEAVY":
            metrics["compression_ratio"] = 0.3
        elif self.cache_info["compression"] == "LIGHT":
            metrics["compression_ratio"] = 0.6
        else:
            metrics["compression_ratio"] = 1.0
        
        self.cache_info["cache_size_mb"] = metrics["total_size_mb"]
        
        return metrics
    
    def cleanup_cache(self):
        """Clean up cache directory"""
        if os.path.exists(self.cache_dir):
            shutil.rmtree(self.cache_dir)
            print(f"[Cache] Cleaned up cache directory: {self.cache_dir}")

class AdaptivePhysicsLOD:
    """Adaptive Level-of-Detail system for physics optimization"""
    
    def __init__(self):
        self.lod_groups = {}
        self.active_lod_level = 0
    
    def create_lod_group(self, base_object, levels=3):
        """Create LOD group for physics object"""
        group_name = f"{base_object.name}_LOD_Group"
        lod_objects = []
        
        for level in range(levels):
            if level == 0:
                # Use original for LOD0
                lod_obj = base_object
            else:
                # Create simplified version
                lod_obj = base_object.copy()
                lod_obj.data = base_object.data.copy()
                bpy.context.collection.objects.link(lod_obj)
                lod_obj.name = f"{base_object.name}_LOD{level}"
                
                # Apply simplification
                decimate = lod_obj.modifiers.new(f"Decimate_LOD{level}", 'DECIMATE')
                decimate.ratio = 1.0 / (2 ** level)
                
                # Apply modifier
                bpy.context.view_layer.objects.active = lod_obj
                bpy.ops.object.modifier_apply(modifier=f"Decimate_LOD{level}")
                
                # Hide by default
                lod_obj.hide_set(True)
                lod_obj.hide_render = True
            
            # Setup physics for LOD
            if not hasattr(lod_obj, 'rigid_body'):
                bpy.context.view_layer.objects.active = lod_obj
                bpy.ops.rigidbody.object_add()
            
            # Optimize collision shape based on LOD
            if level == 0:
                lod_obj.rigid_body.collision_shape = 'MESH'
            elif level == 1:
                lod_obj.rigid_body.collision_shape = 'CONVEX_HULL'
            else:
                lod_obj.rigid_body.collision_shape = 'BOX'
            
            lod_objects.append(lod_obj)
        
        self.lod_groups[group_name] = {
            "objects": lod_objects,
            "base": base_object,
            "active_level": 0
        }
        
        return lod_objects
    
    def switch_lod_level(self, group_name, level):
        """Switch to different LOD level"""
        if group_name not in self.lod_groups:
            return False
        
        group = self.lod_groups[group_name]
        
        if level >= len(group["objects"]):
            level = len(group["objects"]) - 1
        
        # Hide all LODs
        for obj in group["objects"]:
            obj.hide_set(True)
            obj.hide_render = True
            if hasattr(obj, 'rigid_body'):
                obj.rigid_body.enabled = False
        
        # Show selected LOD
        active_obj = group["objects"][level]
        active_obj.hide_set(False)
        active_obj.hide_render = False
        if hasattr(active_obj, 'rigid_body'):
            active_obj.rigid_body.enabled = True
        
        group["active_level"] = level
        
        return True
    
    def auto_switch_lod_by_distance(self, camera_obj):
        """Automatically switch LOD based on camera distance"""
        for group_name, group in self.lod_groups.items():
            base_obj = group["base"]
            
            # Calculate distance to camera
            distance = (camera_obj.location - base_obj.location).length
            
            # Determine LOD level
            if distance < 5:
                level = 0  # High detail
            elif distance < 15:
                level = 1  # Medium detail
            else:
                level = 2  # Low detail
            
            # Switch if needed
            if group["active_level"] != level:
                self.switch_lod_level(group_name, level)
                print(f"[LOD] {group_name}: Switched to level {level} (distance: {distance:.1f})")

def demo_cache_streaming():
    """Demonstrate physics cache streaming"""
    print("=" * 60)
    print("PHYSICS CACHE STREAMING DEMO")
    print("=" * 60)
    
    # Initialize cache manager
    cache_mgr = PhysicsCacheManager()
    
    # Clear scene
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()
    
    # Create scene
    # Ground
    bpy.ops.mesh.primitive_plane_add(size=15, location=(0, 0, 0))
    ground = bpy.context.active_object
    ground.name = "Ground"
    bpy.ops.rigidbody.object_add()
    ground.rigid_body.type = 'PASSIVE'
    
    # Create many physics objects
    print("[Setup] Creating 200 physics objects...")
    for i in range(200):
        x = (i % 20 - 10) * 0.6
        y = (i // 20 - 5) * 0.6
        z = 3 + (i % 5) * 2
        
        bpy.ops.mesh.primitive_cube_add(size=0.5, location=(x, y, z))
        obj = bpy.context.active_object
        obj.name = f"Cube_{i:03d}"
        
        bpy.ops.rigidbody.object_add()
        obj.rigid_body.mass = 0.5 + (i % 3) * 0.5
    
    # Configure cache
    cache_mgr.configure_cache_settings(compression="LIGHT", disk_cache=True)
    
    # Optimize memory
    cache_mgr.optimize_memory_usage()
    
    # Stream cache to disk
    chunks = cache_mgr.stream_cache_to_disk(start_frame=1, end_frame=250, chunk_size=50)
    
    # Analyze performance
    metrics = cache_mgr.analyze_cache_performance()
    
    print("\n" + "=" * 60)
    print("CACHE STREAMING COMPLETE")
    print("=" * 60)
    print(f"Total chunks: {len(chunks)}")
    print(f"Cache size: {metrics['total_size_mb']:.2f} MB")
    print(f"Compression: {cache_mgr.cache_info['compression']}")
    print(f"Cache directory: {cache_mgr.cache_dir}")
    
    # Save metrics
    metrics_path = "/tmp/blender_cache_metrics.json"
    with open(metrics_path, "w") as f:
        json.dump(metrics, f, indent=2)
    
    print(f"Metrics saved to: {metrics_path}")
    
    return cache_mgr

def demo_adaptive_lod():
    """Demonstrate adaptive LOD system"""
    print("=" * 60)
    print("ADAPTIVE LOD PHYSICS DEMO")
    print("=" * 60)
    
    # Initialize LOD system
    lod_system = AdaptivePhysicsLOD()
    
    # Clear scene
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()
    
    # Create ground
    bpy.ops.mesh.primitive_plane_add(size=20, location=(0, 0, 0))
    ground = bpy.context.active_object
    ground.name = "Ground"
    bpy.ops.rigidbody.object_add()
    ground.rigid_body.type = 'PASSIVE'
    
    # Create complex object with LODs
    bpy.ops.mesh.primitive_monkey_add(location=(0, 0, 5))
    suzanne = bpy.context.active_object
    suzanne.name = "Suzanne"
    
    # Subdivide for detail
    suzanne.modifiers.new("Subdivision", 'SUBSURF')
    suzanne.modifiers["Subdivision"].levels = 2
    bpy.ops.object.modifier_apply(modifier="Subdivision")
    
    # Create LOD group
    lods = lod_system.create_lod_group(suzanne, levels=3)
    
    print(f"[LOD] Created {len(lods)} LOD levels:")
    for i, lod in enumerate(lods):
        poly_count = len(lod.data.polygons) if lod.type == 'MESH' else 0
        print(f"  LOD{i}: {poly_count} polygons")
    
    # Create camera
    bpy.ops.object.camera_add(location=(10, -10, 5), rotation=(1.1, 0, 0.785))
    camera = bpy.context.active_object
    bpy.context.scene.camera = camera
    
    # Demonstrate LOD switching
    for distance in [3, 10, 20]:
        camera.location.x = distance
        lod_system.auto_switch_lod_by_distance(camera)
        time.sleep(0.5)  # Pause for visualization
    
    print("\n[LOD] Adaptive LOD system configured")
    
    return lod_system

if __name__ == "__main__":
    # Run cache streaming demo
    cache_mgr = demo_cache_streaming()
    
    # Run adaptive LOD demo
    lod_system = demo_adaptive_lod()
    
    print("\n" + "=" * 60)
    print("ALL OPTIMIZATIONS COMPLETE")
    print("=" * 60)