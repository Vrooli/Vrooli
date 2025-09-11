#!/usr/bin/env python3
"""
Blender Fluid Dynamics Simulation
Demonstrates liquid and gas simulation capabilities
"""

import bpy
import bmesh
import math
import json
import os
from datetime import datetime

# Clear existing scene
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete()

def create_fluid_domain():
    """Create the fluid simulation domain."""
    # Create domain cube
    bpy.ops.mesh.primitive_cube_add(size=4, location=(0, 0, 2))
    domain = bpy.context.active_object
    domain.name = "FluidDomain"
    
    # Add fluid modifier as domain
    modifier = domain.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'DOMAIN'
    
    # Configure domain settings
    settings = modifier.domain_settings
    settings.domain_type = 'LIQUID'  # Can be 'LIQUID' or 'GAS'
    settings.resolution_max = 64  # Resolution for simulation
    settings.use_adaptive_time_steps = True
    settings.use_collision_border_front = True
    settings.use_collision_border_back = True
    settings.use_collision_border_right = True
    settings.use_collision_border_left = True
    settings.use_collision_border_top = False  # Open top for splash
    settings.use_collision_border_bottom = True
    
    # Physics settings
    settings.gravity = (0, 0, -9.81)  # Gravity vector
    settings.viscosity_value = 5.0  # Water-like viscosity
    settings.surface_tension = 0.05
    
    # Cache settings
    settings.cache_directory = "/tmp/blender_fluid_cache"
    settings.cache_frame_start = 1
    settings.cache_frame_end = 100
    
    return domain

def create_fluid_inflow():
    """Create a fluid inflow source."""
    # Create inflow sphere
    bpy.ops.mesh.primitive_uv_sphere_add(radius=0.3, location=(0, 0, 3.5))
    inflow = bpy.context.active_object
    inflow.name = "FluidInflow"
    
    # Add fluid modifier as flow
    modifier = inflow.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'FLOW'
    
    # Configure flow settings
    settings = modifier.flow_settings
    settings.flow_type = 'LIQUID'  # Can be 'LIQUID', 'GAS', or 'BOTH'
    settings.flow_behavior = 'INFLOW'
    settings.use_initial_velocity = True
    settings.velocity_coord = (0, 0, -2)  # Initial velocity
    settings.volume_density = 1.0
    
    return inflow

def create_obstacle():
    """Create an obstacle for fluid interaction."""
    # Create obstacle cube
    bpy.ops.mesh.primitive_cube_add(size=0.8, location=(0.5, 0, 1))
    obstacle = bpy.context.active_object
    obstacle.name = "FluidObstacle"
    
    # Rotate for interesting interaction
    obstacle.rotation_euler = (0.3, 0.5, 0.2)
    
    # Add fluid modifier as effector
    modifier = obstacle.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'EFFECTOR'
    
    # Configure effector settings
    settings = modifier.effector_settings
    settings.effector_type = 'COLLISION'
    settings.use_effector = True
    settings.surface_distance = 0.0  # How close fluid gets to surface
    
    return obstacle

def setup_materials():
    """Create materials for visualization."""
    # Create water material
    water_mat = bpy.data.materials.new(name="Water")
    water_mat.use_nodes = True
    nodes = water_mat.node_tree.nodes
    
    # Clear default nodes
    nodes.clear()
    
    # Add principled BSDF
    bsdf = nodes.new(type='ShaderNodeBsdfPrincipled')
    bsdf.inputs['Base Color'].default_value = (0.5, 0.7, 1.0, 1.0)  # Light blue
    bsdf.inputs['Metallic'].default_value = 0.0
    bsdf.inputs['Roughness'].default_value = 0.1
    bsdf.inputs['IOR'].default_value = 1.333  # Water IOR
    bsdf.inputs['Transmission'].default_value = 0.9  # Transparent
    
    # Add output node
    output = nodes.new(type='ShaderNodeOutputMaterial')
    water_mat.node_tree.links.new(bsdf.outputs['BSDF'], output.inputs['Surface'])
    
    return water_mat

def setup_lighting():
    """Setup lighting for the scene."""
    # Add sun light
    bpy.ops.object.light_add(type='SUN', location=(5, 5, 10))
    sun = bpy.context.active_object
    sun.data.energy = 2.0
    
    # Add area light for soft shadows
    bpy.ops.object.light_add(type='AREA', location=(-3, -3, 5))
    area = bpy.context.active_object
    area.data.energy = 50
    area.data.size = 3
    
    # Setup world environment
    world = bpy.context.scene.world
    world.use_nodes = True
    bg = world.node_tree.nodes['Background']
    bg.inputs[0].default_value = (0.2, 0.3, 0.4, 1.0)  # Sky blue
    bg.inputs[1].default_value = 0.5  # Strength

def setup_camera():
    """Setup camera for rendering."""
    # Add camera
    bpy.ops.object.camera_add(location=(6, -6, 4))
    camera = bpy.context.active_object
    camera.rotation_euler = (1.1, 0, 0.785)
    
    # Set as active camera
    bpy.context.scene.camera = camera
    
    # Configure camera settings
    camera.data.lens = 35
    camera.data.sensor_width = 36

def run_simulation():
    """Bake the fluid simulation."""
    print("[INFO] Starting fluid simulation bake...")
    
    # Get domain object
    domain = bpy.data.objects.get("FluidDomain")
    if not domain:
        print("[ERROR] Fluid domain not found")
        return False
    
    # Select domain
    bpy.context.view_layer.objects.active = domain
    
    # Bake the simulation
    try:
        # Set frame range
        bpy.context.scene.frame_start = 1
        bpy.context.scene.frame_end = 100
        
        # Bake all data
        bpy.ops.fluid.bake_all()
        print("[SUCCESS] Fluid simulation baked successfully")
        return True
    except Exception as e:
        print(f"[ERROR] Failed to bake simulation: {e}")
        return False

def render_animation():
    """Render the fluid animation."""
    # Configure render settings
    scene = bpy.context.scene
    scene.render.engine = 'CYCLES'  # Use Cycles for realistic fluid
    scene.cycles.samples = 64  # Lower samples for faster render
    scene.render.resolution_x = 640
    scene.render.resolution_y = 480
    scene.render.resolution_percentage = 100
    
    # Set output format
    scene.render.image_settings.file_format = 'FFMPEG'
    scene.render.ffmpeg.format = 'MPEG4'
    scene.render.ffmpeg.codec = 'H264'
    scene.render.filepath = "/tmp/blender_fluid_simulation.mp4"
    
    # Render animation
    print("[INFO] Rendering fluid animation...")
    bpy.ops.render.render(animation=True)
    print(f"[SUCCESS] Animation saved to: {scene.render.filepath}")

def export_simulation_data():
    """Export simulation data for analysis."""
    data = {
        "simulation_type": "fluid_dynamics",
        "timestamp": datetime.now().isoformat(),
        "parameters": {
            "domain_type": "LIQUID",
            "resolution": 64,
            "viscosity": 5.0,
            "surface_tension": 0.05,
            "gravity": [-9.81, 0, 0],
            "frames": 100
        },
        "objects": {
            "domain": "FluidDomain",
            "inflow": "FluidInflow",
            "obstacles": ["FluidObstacle"]
        },
        "performance": {
            "bake_time_estimate": "2-5 minutes",
            "render_time_estimate": "5-10 minutes",
            "cache_size_estimate": "100-200 MB"
        }
    }
    
    # Save to JSON
    output_file = "/tmp/blender_fluid_data.json"
    with open(output_file, 'w') as f:
        json.dump(data, f, indent=2)
    
    print(f"[INFO] Simulation data exported to: {output_file}")
    
    # Also save to Blender output directory if available
    output_dir = os.getenv('BLENDER_OUTPUT_DIR', '/output')
    if os.path.exists(output_dir):
        output_file2 = os.path.join(output_dir, 'fluid_simulation_data.json')
        with open(output_file2, 'w') as f:
            json.dump(data, f, indent=2)
        print(f"[INFO] Data also saved to: {output_file2}")

def main():
    """Main execution function."""
    print("=" * 60)
    print("Blender Fluid Dynamics Simulation")
    print("=" * 60)
    
    # Create scene
    print("[INFO] Setting up fluid simulation scene...")
    domain = create_fluid_domain()
    inflow = create_fluid_inflow()
    obstacle = create_obstacle()
    
    # Setup visuals
    water_mat = setup_materials()
    setup_lighting()
    setup_camera()
    
    # Export setup data
    export_simulation_data()
    
    # Run simulation (comment out for quick testing)
    # Uncomment the following lines to actually bake and render:
    # if run_simulation():
    #     render_animation()
    
    print("[INFO] Fluid simulation setup complete")
    print("[INFO] To bake: Uncomment simulation lines in script")
    print("[INFO] Domain created with liquid physics")
    print("[INFO] Inflow source and obstacle configured")
    print("=" * 60)

if __name__ == "__main__":
    main()