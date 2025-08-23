#!/usr/bin/env python3
"""
Blender animation example.
Creates an animated scene with rotating objects.

Usage:
    vrooli resource blender inject examples/animation.py
    vrooli resource blender run animation.py
    vrooli resource blender export animation.mp4 ./my_animation.mp4
"""

import bpy
import math

# Clear existing objects
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)

# Set animation length
scene = bpy.context.scene
scene.frame_start = 1
scene.frame_end = 120  # 4 seconds at 30fps
scene.render.fps = 30

# Create central sphere
bpy.ops.mesh.primitive_uv_sphere_add(location=(0, 0, 0))
center_sphere = bpy.context.object
center_sphere.name = "CenterSphere"

# Add material
mat = bpy.data.materials.new(name="CenterMaterial")
mat.use_nodes = True
mat.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (1, 0.8, 0, 1)  # Gold color
mat.node_tree.nodes["Principled BSDF"].inputs[4].default_value = 0.8  # Metallic
center_sphere.data.materials.append(mat)

# Create orbiting cubes
orbit_radius = 3
num_cubes = 6

for i in range(num_cubes):
    angle = (2 * math.pi * i) / num_cubes
    x = orbit_radius * math.cos(angle)
    y = orbit_radius * math.sin(angle)
    
    bpy.ops.mesh.primitive_cube_add(location=(x, y, 0), scale=(0.5, 0.5, 0.5))
    cube = bpy.context.object
    cube.name = f"OrbitCube_{i}"
    
    # Add colorful material
    mat = bpy.data.materials.new(name=f"CubeMat_{i}")
    mat.use_nodes = True
    # Rainbow colors
    hue = i / num_cubes
    mat.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (
        abs(math.cos(hue * 2 * math.pi)),
        abs(math.cos((hue + 0.33) * 2 * math.pi)),
        abs(math.cos((hue + 0.66) * 2 * math.pi)),
        1
    )
    cube.data.materials.append(mat)
    
    # Animate rotation around center
    cube.rotation_euler = (0, 0, 0)
    cube.keyframe_insert(data_path="rotation_euler", frame=1)
    
    cube.rotation_euler = (0, 0, math.radians(360))
    cube.keyframe_insert(data_path="rotation_euler", frame=scene.frame_end)
    
    # Animate orbiting
    for frame in range(scene.frame_start, scene.frame_end + 1):
        scene.frame_set(frame)
        orbit_angle = angle + (2 * math.pi * frame / scene.frame_end)
        cube.location = (
            orbit_radius * math.cos(orbit_angle),
            orbit_radius * math.sin(orbit_angle),
            math.sin(frame * 0.1) * 0.5  # Vertical oscillation
        )
        cube.keyframe_insert(data_path="location", frame=frame)

# Animate center sphere scale
scene.frame_set(1)
center_sphere.scale = (1, 1, 1)
center_sphere.keyframe_insert(data_path="scale", frame=1)

scene.frame_set(30)
center_sphere.scale = (1.5, 1.5, 1.5)
center_sphere.keyframe_insert(data_path="scale", frame=30)

scene.frame_set(60)
center_sphere.scale = (0.8, 0.8, 0.8)
center_sphere.keyframe_insert(data_path="scale", frame=60)

scene.frame_set(90)
center_sphere.scale = (1.2, 1.2, 1.2)
center_sphere.keyframe_insert(data_path="scale", frame=90)

scene.frame_set(120)
center_sphere.scale = (1, 1, 1)
center_sphere.keyframe_insert(data_path="scale", frame=120)

# Add lighting
bpy.ops.object.light_add(type='SUN', location=(5, 5, 10))
sun = bpy.context.object
sun.data.energy = 2

# Add camera with animation
bpy.ops.object.camera_add(location=(8, -8, 6))
camera = bpy.context.object
camera.name = "AnimatedCamera"

# Point camera at center
constraint = camera.constraints.new(type='TRACK_TO')
constraint.target = center_sphere
constraint.track_axis = 'TRACK_NEGATIVE_Z'
constraint.up_axis = 'UP_Y'

# Animate camera position (circular dolly)
for frame in range(scene.frame_start, scene.frame_end + 1):
    scene.frame_set(frame)
    cam_angle = 2 * math.pi * frame / scene.frame_end
    camera.location = (
        8 * math.cos(cam_angle),
        8 * math.sin(cam_angle),
        6 + 2 * math.sin(frame * 0.05)
    )
    camera.keyframe_insert(data_path="location", frame=frame)

# Set camera as active
scene.camera = camera

# Configure render settings
render = scene.render
render.engine = 'EEVEE'  # Use EEVEE for faster animation rendering
render.resolution_x = 1280
render.resolution_y = 720
render.resolution_percentage = 100

# Set output format for animation
render.image_settings.file_format = 'FFMPEG'
render.ffmpeg.format = 'MPEG4'
render.ffmpeg.codec = 'H264'
render.filepath = '/output/animation.mp4'

print("Rendering animation...")
print(f"Frames: {scene.frame_start} to {scene.frame_end}")
print(f"Output: {render.filepath}")

# Render animation
bpy.ops.render.render(animation=True)

print("Animation rendering complete!")

# Also save the blend file
bpy.ops.wm.save_as_mainfile(filepath='/output/animation.blend')
print("Blend file saved to: /output/animation.blend")