#!/usr/bin/env python3
"""
Example Blender script: Create a simple cube with material
This demonstrates basic Blender Python API usage.
"""

import bpy
import math

def create_cube_scene():
    """Create a simple scene with a textured cube."""
    
    # Clear existing mesh objects
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete(use_global=False)
    
    # Create a cube
    bpy.ops.mesh.primitive_cube_add(
        size=2,
        location=(0, 0, 0),
        rotation=(math.radians(30), math.radians(45), 0)
    )
    cube = bpy.context.active_object
    cube.name = "VrooliCube"
    
    # Create a material
    material = bpy.data.materials.new(name="CubeMaterial")
    material.use_nodes = True
    
    # Get the principled BSDF node
    bsdf = material.node_tree.nodes["Principled BSDF"]
    bsdf.inputs['Base Color'].default_value = (0.5, 0.2, 0.8, 1.0)  # Purple color
    bsdf.inputs['Metallic'].default_value = 0.5
    bsdf.inputs['Roughness'].default_value = 0.3
    
    # Assign material to cube
    cube.data.materials.append(material)
    
    # Add a light
    bpy.ops.object.light_add(
        type='SUN',
        location=(5, 5, 5),
        rotation=(math.radians(-45), math.radians(-45), 0)
    )
    light = bpy.context.active_object
    light.name = "SunLight"
    light.data.energy = 2.0
    
    # Set up camera
    bpy.ops.object.camera_add(
        location=(7, -7, 5),
        rotation=(math.radians(60), 0, math.radians(45))
    )
    camera = bpy.context.active_object
    camera.name = "MainCamera"
    
    # Point camera at cube
    bpy.context.scene.camera = camera
    
    # Set render settings
    scene = bpy.context.scene
    scene.render.engine = 'CYCLES'
    scene.cycles.samples = 128
    scene.render.resolution_x = 1920
    scene.render.resolution_y = 1080
    scene.render.resolution_percentage = 100
    
    # Save the blend file
    output_path = "/output/cube_scene.blend"
    bpy.ops.wm.save_as_mainfile(filepath=output_path)
    print(f"Scene saved to: {output_path}")
    
    # Export as STL
    bpy.ops.object.select_all(action='DESELECT')
    cube.select_set(True)
    bpy.context.view_layer.objects.active = cube
    
    stl_path = "/output/cube.stl"
    bpy.ops.export_mesh.stl(
        filepath=stl_path,
        use_selection=True
    )
    print(f"Cube exported to: {stl_path}")
    
    # Render an image
    image_path = "/output/cube_render.png"
    scene.render.filepath = image_path
    bpy.ops.render.render(write_still=True)
    print(f"Render saved to: {image_path}")

if __name__ == "__main__":
    create_cube_scene()
    print("✅ Cube scene created successfully!")