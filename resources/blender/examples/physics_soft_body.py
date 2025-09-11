#!/usr/bin/env python3
"""
Soft Body Physics Simulation
Demonstrates deformable objects, cloth, and soft body dynamics
"""

import bpy
import random
from mathutils import Vector

def clear_scene():
    """Remove all objects from the scene"""
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()

def setup_physics_world():
    """Configure physics for soft body simulation"""
    scene = bpy.context.scene
    
    # Enable rigid body world for collision objects
    bpy.ops.rigidbody.world_add()
    
    rbw = scene.rigidbody_world
    rbw.point_cache.frame_start = 1
    rbw.point_cache.frame_end = 250
    rbw.substeps_per_frame = 5  # Blender 4.0+ compatible
    rbw.solver_iterations = 20
    
    # Standard gravity
    scene.gravity = (0, 0, -9.81)
    scene.render.fps = 30

def create_ground_bowl():
    """Create a bowl-shaped ground for soft bodies to fall into"""
    # Create UV sphere and scale it
    bpy.ops.mesh.primitive_uv_sphere_add(
        segments=32,
        ring_count=16,
        radius=5,
        location=(0, 0, -2)
    )
    bowl = bpy.context.active_object
    bowl.name = "Ground_Bowl"
    
    # Flatten the top to create bowl shape
    bowl.scale = (1, 1, 0.3)
    bpy.ops.object.transform_apply(scale=True)
    
    # Add solidify modifier for thickness
    solidify = bowl.modifiers.new("Solidify", 'SOLIDIFY')
    solidify.thickness = 0.1
    solidify.offset = -1
    
    # Add rigid body (passive)
    bpy.ops.rigidbody.object_add()
    bowl.rigid_body.type = 'PASSIVE'
    bowl.rigid_body.collision_shape = 'MESH'
    
    # Material
    mat = bpy.data.materials.new(name="Bowl_Material")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (0.8, 0.8, 0.9, 1.0)  # Light blue
    bowl.data.materials.append(mat)
    
    return bowl

def create_soft_sphere():
    """Create a soft deformable sphere"""
    bpy.ops.mesh.primitive_uv_sphere_add(
        segments=32,
        ring_count=16,
        radius=0.8,
        location=(0, 0, 5)
    )
    sphere = bpy.context.active_object
    sphere.name = "Soft_Sphere"
    
    # Add soft body physics
    bpy.ops.object.modifier_add(type='SOFT_BODY')
    soft_body = sphere.modifiers["Softbody"]
    
    # Configure soft body settings
    sb = sphere.soft_body
    sb.mass = 1.0
    sb.friction = 0.5
    
    # Stiffness settings
    sb.spring_length = 0.5  # How much springs can stretch
    sb.shear = 0.5  # Resistance to shearing
    sb.bend = 0.1  # Resistance to bending
    
    # Goal (shape retention)
    sb.use_goal = True
    sb.goal_default = 0.3  # How much to retain original shape
    sb.goal_spring = 0.5
    sb.goal_friction = 0.5
    
    # Collision settings
    sb.use_self_collision = True
    sb.collision_type = 'MANUAL'
    sb.ball_size = 0.05
    
    # Material
    mat = bpy.data.materials.new(name="Soft_Sphere_Mat")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (1.0, 0.3, 0.3, 1.0)  # Red
    bsdf.inputs[4].default_value = 0.8  # Metallic
    sphere.data.materials.append(mat)
    
    return sphere

def create_cloth_plane():
    """Create a cloth simulation"""
    # Create plane for cloth
    bpy.ops.mesh.primitive_plane_add(
        size=3,
        location=(2, 0, 4)
    )
    cloth = bpy.context.active_object
    cloth.name = "Cloth"
    
    # Subdivide for better cloth simulation
    bpy.ops.object.mode_set(mode='EDIT')
    bpy.ops.mesh.subdivide(number_cuts=20)
    bpy.ops.object.mode_set(mode='OBJECT')
    
    # Add cloth modifier
    bpy.ops.object.modifier_add(type='CLOTH')
    cloth_mod = cloth.modifiers["Cloth"]
    
    # Configure cloth settings
    cloth_settings = cloth_mod.settings
    cloth_settings.quality = 10  # Higher quality simulation
    cloth_settings.mass = 0.3
    cloth_settings.air_damping = 1.0
    
    # Stiffness
    cloth_settings.tension_stiffness = 15
    cloth_settings.compression_stiffness = 15
    cloth_settings.shear_stiffness = 5
    cloth_settings.bending_stiffness = 0.5
    
    # Collision settings
    cloth_mod.collision_settings.collision_quality = 5
    cloth_mod.collision_settings.distance_min = 0.015
    cloth_mod.collision_settings.self_collision_quality = 5
    cloth_mod.collision_settings.use_self_collision = True
    
    # Material
    mat = bpy.data.materials.new(name="Cloth_Material")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (0.2, 0.5, 0.9, 1.0)  # Blue fabric
    bsdf.inputs[7].default_value = 0.8  # Roughness
    cloth.data.materials.append(mat)
    
    return cloth

def create_jelly_cube():
    """Create a jelly-like soft body cube"""
    bpy.ops.mesh.primitive_cube_add(
        size=1.5,
        location=(-2, 0, 6)
    )
    jelly = bpy.context.active_object
    jelly.name = "Jelly_Cube"
    
    # Subdivide for better deformation
    bpy.ops.object.mode_set(mode='EDIT')
    bpy.ops.mesh.subdivide(number_cuts=3)
    bpy.ops.object.mode_set(mode='OBJECT')
    
    # Add soft body
    bpy.ops.object.modifier_add(type='SOFT_BODY')
    
    # Configure for jelly-like behavior
    sb = jelly.soft_body
    sb.mass = 2.0
    sb.friction = 0.2
    
    # Very soft settings
    sb.spring_length = 0.8
    sb.shear = 0.1
    sb.bend = 0.01
    
    # Low shape retention for wobbliness
    sb.use_goal = True
    sb.goal_default = 0.1
    sb.goal_spring = 0.1
    
    # Collision
    sb.use_self_collision = True
    sb.collision_type = 'MANUAL'
    sb.ball_size = 0.1
    
    # Material (translucent jelly)
    mat = bpy.data.materials.new(name="Jelly_Material")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (0.9, 0.1, 0.9, 1.0)  # Purple
    bsdf.inputs[15].default_value = 0.8  # Transmission (translucency)
    bsdf.inputs[7].default_value = 0.1  # Low roughness
    jelly.data.materials.append(mat)
    
    return jelly

def create_soft_rope():
    """Create a soft body rope/chain"""
    # Create a bezier curve for the rope
    curve_data = bpy.data.curves.new(name="Rope_Curve", type='CURVE')
    curve_data.dimensions = '3D'
    
    # Create spline
    spline = curve_data.splines.new('BEZIER')
    spline.bezier_points.add(9)  # 10 points total
    
    # Position points to form a hanging rope
    for i, point in enumerate(spline.bezier_points):
        x = i * 0.5 - 2.25
        y = 3
        z = 7 - (i * 0.1)  # Slight sag
        point.co = (x, y, z)
        point.handle_left = (x - 0.2, y, z)
        point.handle_right = (x + 0.2, y, z)
    
    # Create object
    rope = bpy.data.objects.new("Soft_Rope", curve_data)
    bpy.context.collection.objects.link(rope)
    
    # Add thickness to curve
    curve_data.bevel_depth = 0.1
    curve_data.bevel_resolution = 4
    curve_data.resolution_u = 20
    
    # Convert to mesh for soft body
    rope.select_set(True)
    bpy.context.view_layer.objects.active = rope
    bpy.ops.object.convert(target='MESH')
    
    # Add soft body
    bpy.ops.object.modifier_add(type='SOFT_BODY')
    
    sb = rope.soft_body
    sb.mass = 0.5
    sb.friction = 0.3
    
    # Rope-like settings
    sb.spring_length = 0.5
    sb.shear = 0.8  # High shear for rope
    sb.bend = 0.1
    
    # Pin first point (fixed end)
    sb.use_goal = True
    sb.goal_default = 0.0
    
    # Create vertex group for pinning
    bpy.ops.object.mode_set(mode='EDIT')
    bpy.ops.mesh.select_all(action='DESELECT')
    bpy.ops.object.mode_set(mode='OBJECT')
    
    # Select first few vertices
    for i in range(5):
        rope.data.vertices[i].select = True
    
    # Create vertex group
    bpy.ops.object.vertex_group_add()
    bpy.ops.object.vertex_group_assign()
    rope.vertex_groups[0].name = "Pin"
    sb.vertex_group_goal = "Pin"
    
    # Material
    mat = bpy.data.materials.new(name="Rope_Material")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (0.6, 0.4, 0.2, 1.0)  # Brown rope
    bsdf.inputs[7].default_value = 0.9  # Rough
    rope.data.materials.append(mat)
    
    return rope

def create_collision_objects():
    """Create rigid objects for soft bodies to collide with"""
    objects = []
    
    # Rotating cylinder
    bpy.ops.mesh.primitive_cylinder_add(
        radius=0.5,
        depth=4,
        location=(0, 0, 2),
        rotation=(1.57, 0, 0)  # Horizontal
    )
    cylinder = bpy.context.active_object
    cylinder.name = "Rotating_Cylinder"
    
    # Add rigid body
    bpy.ops.rigidbody.object_add()
    cylinder.rigid_body.type = 'PASSIVE'
    cylinder.rigid_body.kinematic = True
    
    # Animate rotation
    cylinder.rotation_euler = (1.57, 0, 0)
    cylinder.keyframe_insert(data_path="rotation_euler", frame=1)
    cylinder.rotation_euler = (1.57, 0, 6.28)
    cylinder.keyframe_insert(data_path="rotation_euler", frame=250)
    
    # Set interpolation to linear
    if cylinder.animation_data:
        for fcurve in cylinder.animation_data.action.fcurves:
            for keyframe in fcurve.keyframe_points:
                keyframe.interpolation = 'LINEAR'
    
    objects.append(cylinder)
    
    # Static sphere obstacles
    for i in range(3):
        angle = i * 2.09  # 120 degrees apart
        x = 2 * bpy.math.cos(angle)
        y = 2 * bpy.math.sin(angle)
        
        bpy.ops.mesh.primitive_uv_sphere_add(
            radius=0.4,
            location=(x, y, 1)
        )
        sphere = bpy.context.active_object
        sphere.name = f"Obstacle_Sphere_{i}"
        
        bpy.ops.rigidbody.object_add()
        sphere.rigid_body.type = 'PASSIVE'
        
        objects.append(sphere)
    
    return objects

def setup_camera_and_lighting():
    """Setup camera and lights optimized for soft body visualization"""
    # Camera
    bpy.ops.object.camera_add(
        location=(10, -10, 8),
        rotation=(1.1, 0, 0.785)
    )
    camera = bpy.context.active_object
    bpy.context.scene.camera = camera
    
    # Key light
    bpy.ops.object.light_add(
        type='AREA',
        location=(5, -5, 10)
    )
    key_light = bpy.context.active_object
    key_light.data.energy = 100
    key_light.data.size = 3
    
    # Fill light
    bpy.ops.object.light_add(
        type='AREA',
        location=(-5, 5, 6)
    )
    fill_light = bpy.context.active_object
    fill_light.data.energy = 30
    fill_light.data.size = 2
    
    # Rim light for depth
    bpy.ops.object.light_add(
        type='SPOT',
        location=(0, 8, 5),
        rotation=(1.0, 0, 0)
    )
    rim_light = bpy.context.active_object
    rim_light.data.energy = 80
    rim_light.data.spot_size = 1.0

def bake_all_physics():
    """Bake all physics simulations"""
    print("Baking soft body physics...")
    
    # Bake rigid body first
    bpy.ops.ptcache.bake_all(bake=True)
    
    # Then bake soft body and cloth
    for obj in bpy.data.objects:
        if obj.soft_body:
            obj.select_set(True)
            bpy.context.view_layer.objects.active = obj
            bpy.ops.ptcache.bake(bake=True)
            obj.select_set(False)
        
        # Check for cloth modifier
        for mod in obj.modifiers:
            if mod.type == 'CLOTH':
                obj.select_set(True)
                bpy.context.view_layer.objects.active = obj
                bpy.ops.ptcache.bake(bake=True)
                obj.select_set(False)

def render_soft_body_animation():
    """Render the soft body animation"""
    scene = bpy.context.scene
    scene.frame_start = 1
    scene.frame_end = 250
    scene.render.fps = 30
    
    # Output settings
    scene.render.image_settings.file_format = 'FFMPEG'
    scene.render.ffmpeg.format = 'MPEG4'
    scene.render.filepath = "/tmp/blender_soft_body_physics.mp4"
    
    # Use EEVEE for good quality with reasonable speed
    scene.render.engine = 'BLENDER_EEVEE'
    scene.eevee.taa_render_samples = 32
    scene.eevee.use_ssr = True  # Screen space reflections
    scene.eevee.use_bloom = True  # Bloom for nice lighting
    
    print("Rendering soft body animation...")
    bpy.ops.render.render(animation=True)

def main():
    """Main function to run soft body simulation"""
    print("=" * 50)
    print("SOFT BODY PHYSICS SIMULATION")
    print("=" * 50)
    
    # Setup
    clear_scene()
    setup_physics_world()
    setup_camera_and_lighting()
    
    # Create physics objects
    print("Creating ground bowl...")
    bowl = create_ground_bowl()
    
    print("Creating soft sphere...")
    soft_sphere = create_soft_sphere()
    
    print("Creating cloth plane...")
    cloth = create_cloth_plane()
    
    print("Creating jelly cube...")
    jelly = create_jelly_cube()
    
    print("Creating soft rope...")
    rope = create_soft_rope()
    
    print("Creating collision objects...")
    obstacles = create_collision_objects()
    
    # Bake all physics
    print("\nBaking physics simulations...")
    bake_all_physics()
    
    # Render animation
    print("\nRendering animation...")
    render_soft_body_animation()
    
    print("\n" + "=" * 50)
    print("SOFT BODY SIMULATION COMPLETE")
    print("=" * 50)
    print("Output: /tmp/blender_soft_body_physics.mp4")
    
    # Export simulation data
    export_soft_body_data()

def export_soft_body_data():
    """Export soft body deformation data for analysis"""
    import json
    
    data = {
        "simulation": {
            "type": "soft_body",
            "frames": bpy.context.scene.frame_end,
            "objects": []
        }
    }
    
    # Track deformation of soft bodies
    for obj in bpy.data.objects:
        if obj.soft_body or any(mod.type == 'CLOTH' for mod in obj.modifiers):
            obj_data = {
                "name": obj.name,
                "type": "soft_body" if obj.soft_body else "cloth",
                "vertex_count": len(obj.data.vertices),
                "deformation_samples": []
            }
            
            # Sample deformation at key frames
            for frame in [1, 50, 100, 150, 200, 250]:
                bpy.context.scene.frame_set(frame)
                
                # Calculate average vertex displacement
                if frame == 1:
                    # Store initial positions
                    initial_positions = [v.co.copy() for v in obj.data.vertices]
                    obj_data["initial_positions"] = [[p.x, p.y, p.z] for p in initial_positions[:5]]  # Sample
                else:
                    # Calculate deformation
                    total_displacement = 0
                    for i, v in enumerate(obj.data.vertices[:min(100, len(obj.data.vertices))]):
                        if i < len(initial_positions):
                            displacement = (v.co - initial_positions[i]).length
                            total_displacement += displacement
                    
                    avg_displacement = total_displacement / min(100, len(obj.data.vertices))
                    obj_data["deformation_samples"].append({
                        "frame": frame,
                        "avg_displacement": avg_displacement
                    })
            
            data["objects"].append(obj_data)
    
    # Save data
    with open("/tmp/blender_soft_body_data.json", "w") as f:
        json.dump(data, f, indent=2)
    
    print("Soft body data exported to: /tmp/blender_soft_body_data.json")

if __name__ == "__main__":
    main()