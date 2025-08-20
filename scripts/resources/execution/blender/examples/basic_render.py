#!/usr/bin/env python3
"""
Basic Blender rendering example.
Creates a simple scene with primitives and renders it.

Usage:
    vrooli resource blender inject examples/basic_render.py
    vrooli resource blender run basic_render.py
    vrooli resource blender export scene.png ./my_render.png
"""

import bpy
import math

# Clear existing mesh objects
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)

# Add a cube
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0))
cube = bpy.context.object
cube.name = "MainCube"

# Add material to cube
mat = bpy.data.materials.new(name="CubeMaterial")
mat.use_nodes = True
mat.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (0.5, 0.7, 1.0, 1.0)  # Blue color
cube.data.materials.append(mat)

# Add a sphere
bpy.ops.mesh.primitive_uv_sphere_add(location=(3, 0, 0))
sphere = bpy.context.object
sphere.name = "Sphere"

# Add material to sphere
mat2 = bpy.data.materials.new(name="SphereMaterial")
mat2.use_nodes = True
mat2.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (1.0, 0.5, 0.5, 1.0)  # Red color
sphere.data.materials.append(mat2)

# Add a cylinder
bpy.ops.mesh.primitive_cylinder_add(location=(-3, 0, 0))
cylinder = bpy.context.object
cylinder.name = "Cylinder"

# Add material to cylinder
mat3 = bpy.data.materials.new(name="CylinderMaterial")
mat3.use_nodes = True
mat3.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (0.5, 1.0, 0.5, 1.0)  # Green color
cylinder.data.materials.append(mat3)

# Add a plane (ground)
bpy.ops.mesh.primitive_plane_add(size=10, location=(0, 0, -2))
plane = bpy.context.object
plane.name = "Ground"

# Add a sun light
bpy.ops.object.light_add(type='SUN', location=(5, 5, 10))
sun = bpy.context.object
sun.data.energy = 2

# Add a point light
bpy.ops.object.light_add(type='POINT', location=(0, -5, 5))
point_light = bpy.context.object
point_light.data.energy = 500

# Set up camera
bpy.ops.object.camera_add(location=(7, -7, 5))
camera = bpy.context.object
camera.rotation_euler = (math.radians(60), 0, math.radians(45))

# Set camera as active
bpy.context.scene.camera = camera

# Configure render settings
render = bpy.context.scene.render
render.engine = 'CYCLES'  # Use Cycles renderer
render.resolution_x = 1920
render.resolution_y = 1080
render.resolution_percentage = 100
render.filepath = '/output/scene.png'

# Set cycles samples for quality
bpy.context.scene.cycles.samples = 128

# Render the scene
print("Rendering scene...")
bpy.ops.render.render(write_still=True)
print(f"Render complete! Output saved to: {render.filepath}")

# Also save the .blend file
bpy.ops.wm.save_as_mainfile(filepath='/output/scene.blend')
print("Blend file saved to: /output/scene.blend")