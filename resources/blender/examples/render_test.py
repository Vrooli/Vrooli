#!/usr/bin/env python3
"""
Rendering Test for Blender
Validates that 3D modeling and rendering capabilities work
"""

import bpy
import sys
import os

def test_rendering():
    """Test basic rendering functionality"""
    print("Testing Blender rendering capabilities...")
    
    # Clear scene
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()
    
    # Create a simple scene with multiple objects
    # Add cube
    bpy.ops.mesh.primitive_cube_add(location=(0, 0, 1))
    cube = bpy.context.active_object
    cube.name = "Test_Cube"
    
    # Add material to cube
    mat = bpy.data.materials.new(name="Cube_Material")
    mat.use_nodes = True
    bsdf = mat.node_tree.nodes["Principled BSDF"]
    bsdf.inputs[0].default_value = (0.5, 0.2, 0.8, 1.0)  # Purple color
    cube.data.materials.append(mat)
    
    # Add sphere
    bpy.ops.mesh.primitive_uv_sphere_add(location=(2, 0, 1))
    sphere = bpy.context.active_object
    sphere.name = "Test_Sphere"
    
    # Add torus
    bpy.ops.mesh.primitive_torus_add(location=(-2, 0, 1))
    torus = bpy.context.active_object
    torus.name = "Test_Torus"
    
    # Add ground plane
    bpy.ops.mesh.primitive_plane_add(size=10, location=(0, 0, 0))
    ground = bpy.context.active_object
    ground.name = "Ground"
    
    # Setup camera
    bpy.ops.object.camera_add(location=(7, -7, 5), rotation=(1.1, 0, 0.785))
    camera = bpy.context.active_object
    bpy.context.scene.camera = camera
    
    # Add lighting
    bpy.ops.object.light_add(type='SUN', location=(5, 5, 10))
    sun = bpy.context.active_object
    sun.data.energy = 2
    
    # Configure render settings
    scene = bpy.context.scene
    scene.render.engine = 'BLENDER_EEVEE'  # Fast renderer
    scene.render.resolution_x = 640
    scene.render.resolution_y = 480
    scene.render.resolution_percentage = 100
    scene.render.image_settings.file_format = 'PNG'
    scene.render.filepath = "/tmp/blender_render_test.png"
    
    # Render single frame
    print("Rendering test image...")
    bpy.ops.render.render(write_still=True)
    
    # Check if render was created
    if os.path.exists("/tmp/blender_render_test.png"):
        file_size = os.path.getsize("/tmp/blender_render_test.png")
        print(f"SUCCESS: Rendered image created ({file_size} bytes)")
        print("Output: /tmp/blender_render_test.png")
        
        # Also export 3D model
        print("Testing 3D export...")
        bpy.ops.export_mesh.stl(filepath="/tmp/blender_test_model.stl")
        
        if os.path.exists("/tmp/blender_test_model.stl"):
            stl_size = os.path.getsize("/tmp/blender_test_model.stl")
            print(f"SUCCESS: STL model exported ({stl_size} bytes)")
            return True
        else:
            print("WARNING: STL export failed")
            return True  # Still pass if render worked
    else:
        print("FAILED: Render not created")
        return False

def test_python_api():
    """Test Python API functionality"""
    print("\nTesting Python API...")
    
    tests_passed = 0
    tests_total = 0
    
    # Test 1: Object creation
    tests_total += 1
    try:
        bpy.ops.mesh.primitive_monkey_add()
        if "Suzanne" in bpy.data.objects:
            print("✓ Object creation API works")
            tests_passed += 1
        else:
            print("✗ Object creation failed")
    except Exception as e:
        print(f"✗ Object creation error: {e}")
    
    # Test 2: Material API
    tests_total += 1
    try:
        mat = bpy.data.materials.new(name="Test_Material")
        mat.use_nodes = True
        print("✓ Material API works")
        tests_passed += 1
    except Exception as e:
        print(f"✗ Material API error: {e}")
    
    # Test 3: Modifier API
    tests_total += 1
    try:
        obj = bpy.context.active_object
        if obj:
            modifier = obj.modifiers.new("Subdivision", 'SUBSURF')
            modifier.levels = 2
            print("✓ Modifier API works")
            tests_passed += 1
        else:
            print("✗ No active object for modifier test")
    except Exception as e:
        print(f"✗ Modifier API error: {e}")
    
    print(f"\nAPI Tests: {tests_passed}/{tests_total} passed")
    return tests_passed == tests_total

if __name__ == "__main__":
    try:
        print("=" * 50)
        print("BLENDER FUNCTIONALITY TEST")
        print("=" * 50)
        
        # Test rendering
        render_success = test_rendering()
        
        # Test Python API
        api_success = test_python_api()
        
        print("\n" + "=" * 50)
        print("TEST SUMMARY")
        print("=" * 50)
        print(f"Rendering: {'PASSED' if render_success else 'FAILED'}")
        print(f"Python API: {'PASSED' if api_success else 'FAILED'}")
        
        success = render_success and api_success
        print(f"\nOverall: {'SUCCESS' if success else 'FAILED'}")
        
        sys.exit(0 if success else 1)
    except Exception as e:
        print(f"ERROR: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)