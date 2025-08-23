#!/usr/bin/env python3
"""
Blender batch processing example.
Demonstrates processing multiple 3D models with consistent settings.

Usage:
    vrooli resource blender inject examples/batch_processing.py
    vrooli resource blender run batch_processing.py
    vrooli resource blender export-all ./output_directory/
"""

import bpy
import os
import json
from pathlib import Path

def setup_render_settings():
    """Configure consistent render settings for all outputs."""
    render = bpy.context.scene.render
    render.engine = 'CYCLES'
    render.resolution_x = 1024
    render.resolution_y = 1024
    render.resolution_percentage = 100
    bpy.context.scene.cycles.samples = 64  # Lower samples for faster batch processing
    
def setup_lighting():
    """Add standard three-point lighting setup."""
    # Key light
    bpy.ops.object.light_add(type='AREA', location=(5, -5, 6))
    key_light = bpy.context.object
    key_light.name = "KeyLight"
    key_light.data.energy = 500
    key_light.data.size = 2
    
    # Fill light
    bpy.ops.object.light_add(type='AREA', location=(-3, -5, 4))
    fill_light = bpy.context.object
    fill_light.name = "FillLight"
    fill_light.data.energy = 200
    fill_light.data.size = 3
    
    # Rim light
    bpy.ops.object.light_add(type='SPOT', location=(0, 5, 5))
    rim_light = bpy.context.object
    rim_light.name = "RimLight"
    rim_light.data.energy = 300
    rim_light.data.spot_size = 1.0

def setup_camera():
    """Add and position camera."""
    bpy.ops.object.camera_add(location=(7, -7, 5))
    camera = bpy.context.object
    camera.name = "MainCamera"
    camera.rotation_euler = (1.1, 0, 0.785)  # Point at origin
    bpy.context.scene.camera = camera
    return camera

def create_object_variant(base_name, variant_num, color):
    """Create a variant of an object with different properties."""
    # Clear scene
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete(use_global=False)
    
    # Create object based on variant number
    if variant_num % 3 == 0:
        bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0))
        obj_type = "cube"
    elif variant_num % 3 == 1:
        bpy.ops.mesh.primitive_uv_sphere_add(location=(0, 0, 0))
        obj_type = "sphere"
    else:
        bpy.ops.mesh.primitive_torus_add(location=(0, 0, 0))
        obj_type = "torus"
    
    obj = bpy.context.object
    obj.name = f"{base_name}_{obj_type}_{variant_num}"
    
    # Apply subdivision surface for smoothness
    modifier = obj.modifiers.new(name="Subdivision", type='SUBSURF')
    modifier.levels = 2
    modifier.render_levels = 2
    
    # Create and apply material
    mat = bpy.data.materials.new(name=f"Material_{variant_num}")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (*color, 1.0)  # Base color
    bsdf.inputs[4].default_value = 0.5  # Metallic
    bsdf.inputs[7].default_value = 0.2  # Roughness
    obj.data.materials.append(mat)
    
    return obj

def render_variants():
    """Main batch processing function."""
    output_dir = Path('/output/batch')
    output_dir.mkdir(exist_ok=True)
    
    # Setup scene
    setup_render_settings()
    setup_lighting()
    camera = setup_camera()
    
    # Define color palette
    colors = [
        (0.8, 0.2, 0.2),  # Red
        (0.2, 0.8, 0.2),  # Green
        (0.2, 0.2, 0.8),  # Blue
        (0.8, 0.8, 0.2),  # Yellow
        (0.8, 0.2, 0.8),  # Magenta
        (0.2, 0.8, 0.8),  # Cyan
    ]
    
    # Batch processing metadata
    batch_info = {
        "batch_name": "3D_Object_Variants",
        "total_renders": len(colors),
        "render_settings": {
            "engine": "CYCLES",
            "resolution": "1024x1024",
            "samples": 64
        },
        "outputs": []
    }
    
    # Process each variant
    for i, color in enumerate(colors):
        print(f"Processing variant {i+1}/{len(colors)}...")
        
        # Create variant
        obj = create_object_variant("Object", i, color)
        
        # Setup lighting (needs to be re-added after clearing scene)
        setup_lighting()
        
        # Re-add camera
        camera = setup_camera()
        
        # Set output path
        output_path = output_dir / f"variant_{i:02d}.png"
        bpy.context.scene.render.filepath = str(output_path)
        
        # Render
        bpy.ops.render.render(write_still=True)
        
        # Save blend file for this variant
        blend_path = output_dir / f"variant_{i:02d}.blend"
        bpy.ops.wm.save_as_mainfile(filepath=str(blend_path))
        
        # Update metadata
        batch_info["outputs"].append({
            "index": i,
            "filename": output_path.name,
            "blend_file": blend_path.name,
            "object_type": obj.name.split('_')[1],
            "color": color,
            "timestamp": bpy.context.scene.frame_current
        })
        
        print(f"  ✓ Rendered: {output_path.name}")
        print(f"  ✓ Saved: {blend_path.name}")
    
    # Save batch metadata
    metadata_path = output_dir / "batch_metadata.json"
    with open(metadata_path, 'w') as f:
        json.dump(batch_info, f, indent=2)
    
    print("\n" + "="*50)
    print(f"Batch processing complete!")
    print(f"Total renders: {len(colors)}")
    print(f"Output directory: {output_dir}")
    print(f"Metadata saved: {metadata_path.name}")
    print("="*50)

# Run batch processing
if __name__ == "__main__":
    render_variants()