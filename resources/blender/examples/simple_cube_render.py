#!/usr/bin/env python3
"""
Simple Blender cube rendering example.
This script creates a basic 3D scene with a cube, light, and camera,
then renders it to demonstrate Blender's functionality.
"""

import bpy
import os
from pathlib import Path

def setup_scene():
    """Set up a basic 3D scene with cube, light, and camera."""
    
    # Clear existing mesh objects
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete(use_global=False)
    
    # Add a cube
    bpy.ops.mesh.primitive_cube_add(size=2, location=(0, 0, 0))
    cube = bpy.context.active_object
    cube.name = "TestCube"
    
    # Add a material to the cube
    mat = bpy.data.materials.new(name="CubeMaterial")
    mat.use_nodes = True
    mat.node_tree.nodes["Principled BSDF"].inputs[0].default_value = (0.5, 0.7, 1.0, 1.0)  # Blue color
    cube.data.materials.append(mat)
    
    # Add a light
    bpy.ops.object.light_add(type='SUN', location=(5, 5, 5))
    light = bpy.context.active_object
    light.data.energy = 1.5
    
    # Add a camera
    bpy.ops.object.camera_add(location=(7, -7, 5))
    camera = bpy.context.active_object
    camera.rotation_euler = (1.1, 0, 0.785)  # Point at cube
    
    # Set camera as active
    bpy.context.scene.camera = camera
    
    # Configure render settings
    render = bpy.context.scene.render
    render.engine = 'CYCLES'  # Use Cycles renderer
    render.resolution_x = 640
    render.resolution_y = 480
    render.resolution_percentage = 100
    
    # Use CPU for compatibility
    bpy.context.scene.cycles.device = 'CPU'
    bpy.context.scene.cycles.samples = 32  # Low samples for quick render
    
    print("Scene setup complete!")
    return True

def render_scene(output_path=None):
    """Render the scene to an image file."""
    
    if output_path is None:
        # Default to output directory
        output_dir = Path(os.path.expanduser("~/.vrooli/blender/output"))
        output_dir.mkdir(parents=True, exist_ok=True)
        output_path = str(output_dir / "test_render.png")
    
    # Set output path
    bpy.context.scene.render.filepath = output_path
    bpy.context.scene.render.image_settings.file_format = 'PNG'
    
    print(f"Rendering to: {output_path}")
    
    # Render the scene
    try:
        bpy.ops.render.render(write_still=True)
        print(f"Render complete! Image saved to: {output_path}")
        return True
    except Exception as e:
        print(f"Render failed: {e}")
        return False

def main():
    """Main function to set up and render a simple scene."""
    
    print("Starting Blender rendering example...")
    
    # Set up the scene
    if not setup_scene():
        print("Failed to set up scene")
        return 1
    
    # Render the scene
    if not render_scene():
        print("Failed to render scene")
        return 1
    
    print("Example completed successfully!")
    return 0

if __name__ == "__main__":
    import sys
    sys.exit(main())