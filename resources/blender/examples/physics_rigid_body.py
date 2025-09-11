#!/usr/bin/env python3
"""
Rigid Body Physics Simulation
Demonstrates Blender's rigid body dynamics with falling cubes
"""

import bpy
import random
from mathutils import Vector

def clear_scene():
    """Remove all objects from the scene"""
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()
    
def create_ground():
    """Create a static ground plane for physics"""
    bpy.ops.mesh.primitive_plane_add(
        size=20,
        location=(0, 0, 0)
    )
    ground = bpy.context.active_object
    ground.name = "Ground"
    
    # Add rigid body physics (passive/static)
    bpy.ops.rigidbody.object_add()
    ground.rigid_body.type = 'PASSIVE'
    ground.rigid_body.collision_shape = 'BOX'
    
    return ground

def create_falling_cubes(count=10):
    """Create cubes that will fall with physics"""
    cubes = []
    
    for i in range(count):
        # Random position above ground
        x = random.uniform(-3, 3)
        y = random.uniform(-3, 3)
        z = random.uniform(3, 8)
        
        # Random rotation
        rot_x = random.uniform(0, 3.14)
        rot_y = random.uniform(0, 3.14)
        rot_z = random.uniform(0, 3.14)
        
        # Create cube
        bpy.ops.mesh.primitive_cube_add(
            size=1,
            location=(x, y, z),
            rotation=(rot_x, rot_y, rot_z)
        )
        
        cube = bpy.context.active_object
        cube.name = f"Cube_{i:03d}"
        
        # Add rigid body physics (active/dynamic)
        bpy.ops.rigidbody.object_add()
        cube.rigid_body.type = 'ACTIVE'
        cube.rigid_body.collision_shape = 'BOX'
        cube.rigid_body.mass = random.uniform(0.5, 2.0)
        cube.rigid_body.friction = 0.5
        cube.rigid_body.restitution = random.uniform(0.2, 0.8)  # Bounciness
        
        # Random material color
        mat = bpy.data.materials.new(name=f"Mat_Cube_{i}")
        mat.use_nodes = True
        bsdf = mat.node_tree.nodes["Principled BSDF"]
        bsdf.inputs[0].default_value = (
            random.random(),
            random.random(),
            random.random(),
            1.0
        )
        cube.data.materials.append(mat)
        
        cubes.append(cube)
    
    return cubes

def create_domino_chain():
    """Create a line of dominoes that will fall"""
    dominoes = []
    
    for i in range(20):
        x = i * 0.8 - 8
        
        # Create tall thin box (domino)
        bpy.ops.mesh.primitive_cube_add(
            size=1,
            location=(x, 0, 1)
        )
        
        domino = bpy.context.active_object
        domino.name = f"Domino_{i:03d}"
        domino.scale = (0.2, 1, 2)
        
        # Apply scale for physics
        bpy.ops.object.transform_apply(scale=True)
        
        # Add rigid body
        bpy.ops.rigidbody.object_add()
        domino.rigid_body.type = 'ACTIVE'
        domino.rigid_body.collision_shape = 'BOX'
        domino.rigid_body.mass = 0.5
        domino.rigid_body.friction = 0.7
        
        # Slight initial tilt for first domino
        if i == 0:
            domino.rotation_euler[1] = 0.1
        
        dominoes.append(domino)
    
    return dominoes

def create_pendulum():
    """Create a physics pendulum"""
    # Create pivot point (passive)
    bpy.ops.mesh.primitive_cube_add(
        size=0.3,
        location=(0, 5, 6)
    )
    pivot = bpy.context.active_object
    pivot.name = "Pendulum_Pivot"
    bpy.ops.rigidbody.object_add()
    pivot.rigid_body.type = 'PASSIVE'
    
    # Create pendulum bob (active)
    bpy.ops.mesh.primitive_uv_sphere_add(
        radius=0.5,
        location=(2, 5, 6)
    )
    bob = bpy.context.active_object
    bob.name = "Pendulum_Bob"
    bpy.ops.rigidbody.object_add()
    bob.rigid_body.type = 'ACTIVE'
    bob.rigid_body.mass = 2.0
    
    # Create constraint between pivot and bob
    bpy.ops.object.empty_add(location=(0, 5, 6))
    constraint = bpy.context.active_object
    constraint.name = "Pendulum_Constraint"
    
    bpy.ops.rigidbody.constraint_add()
    constraint.rigid_body_constraint.type = 'POINT'
    constraint.rigid_body_constraint.object1 = pivot
    constraint.rigid_body_constraint.object2 = bob
    
    return pivot, bob, constraint

def setup_camera_and_lighting():
    """Setup camera and lights for rendering"""
    # Camera
    bpy.ops.object.camera_add(
        location=(15, -15, 10),
        rotation=(1.1, 0, 0.785)
    )
    camera = bpy.context.active_object
    bpy.context.scene.camera = camera
    
    # Sun light
    bpy.ops.object.light_add(
        type='SUN',
        location=(5, 5, 10)
    )
    sun = bpy.context.active_object
    sun.data.energy = 2
    
    # Area light for soft shadows
    bpy.ops.object.light_add(
        type='AREA',
        location=(-5, -5, 8)
    )
    area = bpy.context.active_object
    area.data.energy = 50
    area.data.size = 5

def setup_physics_world():
    """Configure physics simulation settings"""
    scene = bpy.context.scene
    
    # Enable rigid body world
    bpy.ops.rigidbody.world_add()
    
    # Physics settings
    rbw = scene.rigidbody_world
    rbw.point_cache.frame_start = 1
    rbw.point_cache.frame_end = 250
    
    # Gravity
    scene.gravity = (0, 0, -9.81)
    
    # Simulation quality (Blender 4.0+ compatible)
    rbw.substeps_per_frame = 5  # Higher = more accurate
    rbw.solver_iterations = 20  # Higher = more stable

def bake_physics():
    """Bake the physics simulation for faster playback"""
    bpy.ops.ptcache.bake_all(bake=True)

def render_animation():
    """Render the physics animation"""
    scene = bpy.context.scene
    scene.frame_start = 1
    scene.frame_end = 250
    scene.render.fps = 30
    scene.render.image_settings.file_format = 'FFMPEG'
    scene.render.ffmpeg.format = 'MPEG4'
    scene.render.filepath = "/tmp/blender_physics_simulation.mp4"
    
    # Use EEVEE for faster rendering
    scene.render.engine = 'BLENDER_EEVEE'
    scene.eevee.taa_render_samples = 32
    
    # Render animation
    bpy.ops.render.render(animation=True)

def main():
    """Main function to run the physics simulation"""
    print("Setting up rigid body physics simulation...")
    
    # Clear and setup
    clear_scene()
    setup_camera_and_lighting()
    setup_physics_world()
    
    # Create physics objects
    print("Creating ground plane...")
    ground = create_ground()
    
    print("Creating falling cubes...")
    cubes = create_falling_cubes(15)
    
    print("Creating domino chain...")
    dominoes = create_domino_chain()
    
    print("Creating pendulum...")
    pendulum = create_pendulum()
    
    # Bake physics for performance
    print("Baking physics simulation...")
    bake_physics()
    
    # Render
    print("Rendering animation...")
    render_animation()
    
    print("Physics simulation complete!")
    print("Output: /tmp/blender_physics_simulation.mp4")
    
    # Export physics data for analysis
    export_physics_data()

def export_physics_data():
    """Export physics simulation data for validation"""
    import json
    
    data = {
        "simulation": {
            "gravity": list(bpy.context.scene.gravity),
            "fps": bpy.context.scene.render.fps,
            "frames": bpy.context.scene.frame_end
        },
        "objects": []
    }
    
    # Sample physics data at key frames
    for frame in [1, 50, 100, 150, 200, 250]:
        bpy.context.scene.frame_set(frame)
        frame_data = []
        
        for obj in bpy.data.objects:
            if obj.rigid_body and obj.rigid_body.type == 'ACTIVE':
                frame_data.append({
                    "name": obj.name,
                    "frame": frame,
                    "location": list(obj.location),
                    "rotation": list(obj.rotation_euler),
                    "velocity": list(obj.rigid_body.linear_velocity) if hasattr(obj.rigid_body, 'linear_velocity') else [0,0,0]
                })
        
        data["objects"].extend(frame_data)
    
    # Save to JSON
    with open("/tmp/blender_physics_data.json", "w") as f:
        json.dump(data, f, indent=2)
    
    print("Physics data exported to: /tmp/blender_physics_data.json")

if __name__ == "__main__":
    main()