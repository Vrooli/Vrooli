#!/usr/bin/env python3
"""
Simple Physics Test for Blender 4.0+
Tests basic rigid body physics functionality
"""

import bpy
import sys

def test_physics():
    """Simple physics test"""
    print("Starting simple physics test...")
    
    # Clear scene
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()
    
    # Create ground plane
    bpy.ops.mesh.primitive_plane_add(size=10, location=(0, 0, 0))
    ground = bpy.context.active_object
    ground.name = "Ground"
    
    # Add rigid body to ground
    bpy.ops.rigidbody.object_add()
    ground.rigid_body.type = 'PASSIVE'
    
    # Create falling cube
    bpy.ops.mesh.primitive_cube_add(location=(0, 0, 5))
    cube = bpy.context.active_object
    cube.name = "Falling_Cube"
    
    # Add rigid body to cube
    bpy.ops.rigidbody.object_add()
    cube.rigid_body.type = 'ACTIVE'
    
    # Setup physics world
    scene = bpy.context.scene
    if not scene.rigidbody_world:
        # This gets added automatically when we add rigid bodies
        pass
    
    rbw = scene.rigidbody_world
    rbw.point_cache.frame_start = 1
    rbw.point_cache.frame_end = 100
    
    # Bake physics - need to ensure we're in the right context
    print("Baking physics simulation...")
    
    # Set gravity explicitly
    scene.gravity = (0, 0, -9.81)
    
    # Force update and bake
    scene.frame_set(1)
    bpy.context.view_layer.update()
    
    # Use the free_all then bake pattern
    bpy.ops.ptcache.free_bake_all()
    bpy.ops.ptcache.bake_all(bake=True)
    
    # Check cube position at different frames
    results = []
    for frame in [1, 25, 50, 75, 100]:
        scene.frame_set(frame)
        z = cube.location.z
        results.append(f"Frame {frame}: Cube at z={z:.2f}")
        print(f"Frame {frame}: Cube at z={z:.2f}")
    
    # Verify physics worked (cube should fall)
    initial_z = 5.0
    scene.frame_set(50)
    final_z = cube.location.z
    
    if final_z < initial_z - 2.0:
        print("SUCCESS: Physics simulation working! Cube fell as expected.")
        return True
    else:
        print("FAILED: Cube did not fall. Physics may not be working.")
        return False

if __name__ == "__main__":
    try:
        success = test_physics()
        sys.exit(0 if success else 1)
    except Exception as e:
        print(f"ERROR: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)