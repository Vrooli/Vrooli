#!/usr/bin/env python3
"""
Blender Gas/Smoke Simulation
Demonstrates gas and smoke simulation capabilities
"""

import bpy
import math
import json
import os
from datetime import datetime

# Clear existing scene
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete()

def create_smoke_domain():
    """Create the smoke simulation domain."""
    # Create domain cube
    bpy.ops.mesh.primitive_cube_add(size=6, location=(0, 0, 3))
    domain = bpy.context.active_object
    domain.name = "SmokeDomain"
    
    # Add fluid modifier as domain
    modifier = domain.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'DOMAIN'
    
    # Configure domain settings for gas
    settings = modifier.domain_settings
    settings.domain_type = 'GAS'  # Gas simulation
    settings.resolution_max = 64
    settings.use_adaptive_time_steps = True
    
    # Gas physics settings
    settings.alpha = -0.5  # Buoyancy factor (negative = smoke rises)
    settings.beta = 0.2  # Temperature difference factor
    settings.vorticity = 10  # Swirling motion
    settings.dissolve_speed = 10  # How fast smoke disappears
    settings.use_dissolve_smoke = True
    
    # Wind force
    settings.wind_velocity = (0.5, 0, 0)  # Slight wind in X direction
    
    # Cache settings
    settings.cache_directory = "/tmp/blender_smoke_cache"
    settings.cache_frame_start = 1
    settings.cache_frame_end = 150
    
    return domain

def create_smoke_emitter():
    """Create a smoke emitter source."""
    # Create emitter sphere
    bpy.ops.mesh.primitive_cylinder_add(radius=0.5, depth=0.2, location=(0, 0, 0.1))
    emitter = bpy.context.active_object
    emitter.name = "SmokeEmitter"
    
    # Add fluid modifier as flow
    modifier = emitter.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'FLOW'
    
    # Configure flow settings for smoke
    settings = modifier.flow_settings
    settings.flow_type = 'SMOKE'  # Smoke type
    settings.flow_behavior = 'INFLOW'
    settings.use_initial_velocity = True
    settings.velocity_normal = 2.0  # Upward velocity
    settings.velocity_random = 0.5  # Add randomness
    
    # Smoke properties
    settings.density = 1.0
    settings.temperature = 2.0  # Hot smoke rises faster
    settings.smoke_color = (0.8, 0.8, 0.8)  # Light gray smoke
    settings.use_texture = False
    settings.surface_distance = 0.0
    
    return emitter

def create_fire_emitter():
    """Create a fire emitter for flame simulation."""
    # Create fire source
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=2, radius=0.3, location=(2, 0, 0.5))
    fire = bpy.context.active_object
    fire.name = "FireEmitter"
    
    # Add fluid modifier as flow
    modifier = fire.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'FLOW'
    
    # Configure for fire
    settings = modifier.flow_settings
    settings.flow_type = 'SMOKE_FIRE'  # Both smoke and fire
    settings.flow_behavior = 'INFLOW'
    settings.fuel_amount = 1.0  # Amount of fuel for fire
    settings.temperature = 3.0  # Very hot for flames
    settings.density = 0.5  # Less dense smoke from fire
    settings.smoke_color = (0.1, 0.1, 0.1)  # Dark smoke from fire
    
    return fire

def create_collision_object():
    """Create an object that deflects smoke."""
    # Create deflector torus
    bpy.ops.mesh.primitive_torus_add(major_radius=1.5, minor_radius=0.3, location=(0, 0, 2))
    deflector = bpy.context.active_object
    deflector.name = "SmokeDeflector"
    deflector.rotation_euler = (0.5, 0, 0)
    
    # Add fluid modifier as effector
    modifier = deflector.modifiers.new(name="Fluid", type='FLUID')
    modifier.fluid_type = 'EFFECTOR'
    
    # Configure effector settings
    settings = modifier.effector_settings
    settings.effector_type = 'COLLISION'
    settings.use_effector = True
    
    return deflector

def setup_volumetric_material():
    """Create volumetric material for smoke visualization."""
    # Create smoke material
    smoke_mat = bpy.data.materials.new(name="SmokeMaterial")
    smoke_mat.use_nodes = True
    nodes = smoke_mat.node_tree.nodes
    links = smoke_mat.node_tree.links
    
    # Clear default nodes
    nodes.clear()
    
    # Add Principled Volume shader
    volume = nodes.new(type='ShaderNodeVolumePrincipled')
    volume.inputs['Density'].default_value = 5.0
    volume.inputs['Color'].default_value = (0.5, 0.5, 0.5, 1.0)
    volume.inputs['Anisotropy'].default_value = 0.5
    
    # Add emission for fire
    volume.inputs['Blackbody Intensity'].default_value = 1.0
    volume.inputs['Temperature'].default_value = 1500  # Fire temperature
    
    # Add output node
    output = nodes.new(type='ShaderNodeOutputMaterial')
    links.new(volume.outputs['Volume'], output.inputs['Volume'])
    
    return smoke_mat

def setup_environment():
    """Setup lighting and environment for smoke rendering."""
    # Add strong key light
    bpy.ops.object.light_add(type='SPOT', location=(5, -5, 8))
    spot = bpy.context.active_object
    spot.data.energy = 1000
    spot.data.spot_size = 1.0
    spot.rotation_euler = (-0.3, 0.3, 0)
    
    # Add rim light
    bpy.ops.object.light_add(type='AREA', location=(-5, 3, 5))
    area = bpy.context.active_object
    area.data.energy = 200
    area.data.size = 4
    
    # Setup world with dark background
    world = bpy.context.scene.world
    world.use_nodes = True
    bg = world.node_tree.nodes['Background']
    bg.inputs[0].default_value = (0.05, 0.05, 0.1, 1.0)  # Dark blue
    bg.inputs[1].default_value = 0.2  # Low strength

def setup_camera():
    """Setup camera for dramatic smoke rendering."""
    # Add camera
    bpy.ops.object.camera_add(location=(7, -7, 5))
    camera = bpy.context.active_object
    camera.rotation_euler = (1.2, 0, 0.785)
    
    # Set as active camera
    bpy.context.scene.camera = camera
    
    # Configure for cinematic look
    camera.data.lens = 50
    camera.data.sensor_width = 36
    camera.data.dof.use_dof = True
    camera.data.dof.focus_distance = 8
    camera.data.dof.aperture_fstop = 4

def configure_render_settings():
    """Configure render settings for volumetrics."""
    scene = bpy.context.scene
    
    # Use Cycles for better volumetrics
    scene.render.engine = 'CYCLES'
    scene.cycles.samples = 32  # Lower samples for faster preview
    scene.cycles.volume_step_rate = 0.1  # Better volume quality
    scene.cycles.volume_preview_step_rate = 0.5
    
    # Resolution
    scene.render.resolution_x = 640
    scene.render.resolution_y = 480
    scene.render.resolution_percentage = 100
    
    # Animation settings
    scene.frame_start = 1
    scene.frame_end = 150
    scene.render.fps = 24
    
    # Output settings
    scene.render.image_settings.file_format = 'FFMPEG'
    scene.render.ffmpeg.format = 'MPEG4'
    scene.render.ffmpeg.codec = 'H264'
    scene.render.filepath = "/tmp/blender_gas_simulation.mp4"

def export_gas_simulation_data():
    """Export gas simulation parameters."""
    data = {
        "simulation_type": "gas_dynamics",
        "timestamp": datetime.now().isoformat(),
        "parameters": {
            "domain_type": "GAS",
            "resolution": 64,
            "buoyancy_alpha": -0.5,
            "temperature_beta": 0.2,
            "vorticity": 10,
            "dissolve_speed": 10,
            "wind_velocity": [0.5, 0, 0],
            "frames": 150
        },
        "emitters": {
            "smoke": {
                "type": "SMOKE",
                "density": 1.0,
                "temperature": 2.0,
                "velocity": 2.0
            },
            "fire": {
                "type": "SMOKE_FIRE",
                "fuel": 1.0,
                "temperature": 3.0,
                "density": 0.5
            }
        },
        "performance": {
            "bake_time_estimate": "3-7 minutes",
            "render_time_estimate": "10-15 minutes",
            "cache_size_estimate": "200-400 MB",
            "volumetric_overhead": "high"
        }
    }
    
    # Save to JSON
    output_file = "/tmp/blender_gas_simulation_data.json"
    with open(output_file, 'w') as f:
        json.dump(data, f, indent=2)
    
    print(f"[INFO] Gas simulation data exported to: {output_file}")
    
    # Also save to Blender output directory
    output_dir = os.getenv('BLENDER_OUTPUT_DIR', '/output')
    if os.path.exists(output_dir):
        output_file2 = os.path.join(output_dir, 'gas_simulation_data.json')
        with open(output_file2, 'w') as f:
            json.dump(data, f, indent=2)
        print(f"[INFO] Data also saved to: {output_file2}")

def main():
    """Main execution function."""
    print("=" * 60)
    print("Blender Gas/Smoke Simulation")
    print("=" * 60)
    
    # Create simulation
    print("[INFO] Setting up gas simulation scene...")
    domain = create_smoke_domain()
    smoke = create_smoke_emitter()
    fire = create_fire_emitter()
    deflector = create_collision_object()
    
    # Setup rendering
    material = setup_volumetric_material()
    setup_environment()
    setup_camera()
    configure_render_settings()
    
    # Assign material to domain
    domain.data.materials.append(material)
    
    # Export configuration
    export_gas_simulation_data()
    
    print("[SUCCESS] Gas simulation setup complete")
    print("[INFO] Scene includes:")
    print("  - Smoke emitter with upward flow")
    print("  - Fire emitter with fuel combustion")
    print("  - Deflector object for interaction")
    print("  - Volumetric shading with emission")
    print("[INFO] To bake: Select domain and run fluid.bake_all()")
    print("=" * 60)

if __name__ == "__main__":
    main()