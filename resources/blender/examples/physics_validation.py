#!/usr/bin/env python3
"""
Physics Validation Suite
Validates Blender physics accuracy against analytical solutions
"""

import bpy
import math
import json
from mathutils import Vector

def clear_scene():
    """Remove all objects from the scene"""
    bpy.ops.object.select_all(action='SELECT')
    bpy.ops.object.delete()

def setup_physics_world():
    """Configure physics for validation tests"""
    scene = bpy.context.scene
    
    # Enable rigid body world (check if already exists)
    if not scene.rigidbody_world:
        # Need to be in object mode for this to work
        if bpy.context.mode != 'OBJECT':
            bpy.ops.object.mode_set(mode='OBJECT')
        bpy.ops.rigidbody.world_add()
    
    # Precise physics settings for validation
    rbw = scene.rigidbody_world
    rbw.point_cache.frame_start = 1
    rbw.point_cache.frame_end = 300
    # Note: steps_per_second removed in Blender 4.0+
    # Using substeps_per_frame instead for higher precision
    rbw.substeps_per_frame = 10  # High precision
    rbw.solver_iterations = 50  # Maximum stability
    
    # Standard gravity
    scene.gravity = (0, 0, -9.81)
    
    # Set frame rate for precise time steps
    scene.render.fps = 60

def test_free_fall():
    """Test free fall against kinematic equations"""
    print("\n=== Free Fall Validation ===")
    
    # Create falling sphere at height h
    h = 10.0  # meters
    bpy.ops.mesh.primitive_uv_sphere_add(
        radius=0.5,
        location=(0, 0, h)
    )
    sphere = bpy.context.active_object
    sphere.name = "FreeFall_Sphere"
    
    # Add physics
    bpy.ops.rigidbody.object_add()
    sphere.rigid_body.type = 'ACTIVE'
    sphere.rigid_body.mass = 1.0
    sphere.rigid_body.linear_damping = 0  # No air resistance
    sphere.rigid_body.angular_damping = 0
    
    # Calculate expected time to ground
    g = 9.81
    t_expected = math.sqrt(2 * h / g)
    
    # Bake and measure
    bpy.ops.ptcache.bake_all(bake=True)
    
    # Find frame when sphere hits ground
    fps = bpy.context.scene.render.fps
    results = []
    
    for frame in range(1, 300):
        bpy.context.scene.frame_set(frame)
        z = sphere.location.z - 0.5  # Account for radius
        
        if z <= 0.01:  # Hit ground
            t_measured = frame / fps
            error = abs(t_measured - t_expected) / t_expected * 100
            
            results.append({
                "test": "free_fall",
                "expected_time": t_expected,
                "measured_time": t_measured,
                "error_percent": error,
                "passed": error < 5.0  # 5% tolerance
            })
            
            print(f"Expected time: {t_expected:.3f}s")
            print(f"Measured time: {t_measured:.3f}s")
            print(f"Error: {error:.2f}%")
            print(f"Test: {'PASSED' if error < 5.0 else 'FAILED'}")
            break
    
    return results[0] if results else {"test": "free_fall", "passed": False, "error": "No ground impact detected"}

def test_projectile_motion():
    """Test projectile motion against analytical solution"""
    print("\n=== Projectile Motion Validation ===")
    
    clear_scene()
    setup_physics_world()
    
    # Initial conditions
    v0 = 20.0  # m/s initial velocity
    angle = 45  # degrees
    angle_rad = math.radians(angle)
    
    # Create projectile
    bpy.ops.mesh.primitive_uv_sphere_add(
        radius=0.25,
        location=(0, 0, 1)
    )
    projectile = bpy.context.active_object
    projectile.name = "Projectile"
    
    # Add physics with initial velocity
    bpy.ops.rigidbody.object_add()
    projectile.rigid_body.type = 'ACTIVE'
    projectile.rigid_body.mass = 1.0
    projectile.rigid_body.linear_damping = 0
    projectile.rigid_body.kinematic = True  # Start kinematic
    
    # Set initial velocity
    bpy.context.scene.frame_set(1)
    projectile.rigid_body.kinematic = False
    vx = v0 * math.cos(angle_rad)
    vy = 0
    vz = v0 * math.sin(angle_rad)
    projectile.rigid_body.linear_velocity = (vx, vy, vz)
    
    # Calculate expected range
    g = 9.81
    range_expected = (v0 * v0 * math.sin(2 * angle_rad)) / g
    max_height_expected = (v0 * v0 * math.sin(angle_rad)**2) / (2 * g) + 1  # +1 for initial height
    
    # Bake and measure
    bpy.ops.ptcache.bake_all(bake=True)
    
    # Track trajectory
    max_height_measured = 1.0
    range_measured = 0
    
    for frame in range(1, 300):
        bpy.context.scene.frame_set(frame)
        z = projectile.location.z
        x = projectile.location.x
        
        max_height_measured = max(max_height_measured, z)
        
        if z <= 0.26:  # Hit ground (accounting for radius)
            range_measured = x
            break
    
    # Calculate errors
    range_error = abs(range_measured - range_expected) / range_expected * 100 if range_expected > 0 else 100
    height_error = abs(max_height_measured - max_height_expected) / max_height_expected * 100
    
    result = {
        "test": "projectile_motion",
        "range_expected": range_expected,
        "range_measured": range_measured,
        "range_error_percent": range_error,
        "height_expected": max_height_expected,
        "height_measured": max_height_measured,
        "height_error_percent": height_error,
        "passed": range_error < 5.0 and height_error < 5.0
    }
    
    print(f"Expected range: {range_expected:.2f}m")
    print(f"Measured range: {range_measured:.2f}m")
    print(f"Range error: {range_error:.2f}%")
    print(f"Expected max height: {max_height_expected:.2f}m")
    print(f"Measured max height: {max_height_measured:.2f}m")
    print(f"Height error: {height_error:.2f}%")
    print(f"Test: {'PASSED' if result['passed'] else 'FAILED'}")
    
    return result

def test_conservation_of_energy():
    """Test energy conservation in pendulum motion"""
    print("\n=== Energy Conservation Validation ===")
    
    clear_scene()
    setup_physics_world()
    
    # Pendulum parameters
    length = 2.0  # meters
    angle_initial = 30  # degrees
    
    # Create pivot (passive)
    bpy.ops.mesh.primitive_cube_add(
        size=0.1,
        location=(0, 0, 5)
    )
    pivot = bpy.context.active_object
    pivot.name = "Pivot"
    bpy.ops.rigidbody.object_add()
    pivot.rigid_body.type = 'PASSIVE'
    
    # Create pendulum bob
    x_initial = length * math.sin(math.radians(angle_initial))
    z_initial = 5 - length * math.cos(math.radians(angle_initial))
    
    bpy.ops.mesh.primitive_uv_sphere_add(
        radius=0.2,
        location=(x_initial, 0, z_initial)
    )
    bob = bpy.context.active_object
    bob.name = "Pendulum_Bob"
    bpy.ops.rigidbody.object_add()
    bob.rigid_body.type = 'ACTIVE'
    bob.rigid_body.mass = 1.0
    bob.rigid_body.linear_damping = 0  # No damping for energy conservation
    bob.rigid_body.angular_damping = 0
    
    # Create constraint
    bpy.ops.object.empty_add(location=(0, 0, 5))
    constraint = bpy.context.active_object
    constraint.name = "Pendulum_Constraint"
    bpy.ops.rigidbody.constraint_add()
    constraint.rigid_body_constraint.type = 'POINT'
    constraint.rigid_body_constraint.object1 = pivot
    constraint.rigid_body_constraint.object2 = bob
    
    # Calculate initial energy
    g = 9.81
    m = 1.0
    h_initial = 5 - z_initial
    E_initial = m * g * h_initial  # Initial potential energy (v=0)
    
    # Bake and measure energy at different points
    bpy.ops.ptcache.bake_all(bake=True)
    
    energies = []
    for frame in range(1, 120):  # Two complete swings
        bpy.context.scene.frame_set(frame)
        
        # Get position and velocity
        z = bob.location.z
        h = 5 - z  # Height below pivot
        
        # Calculate energies
        PE = m * g * h
        
        # Get velocity magnitude (approximation)
        if frame > 1:
            bpy.context.scene.frame_set(frame - 1)
            pos_prev = bob.location.copy()
            bpy.context.scene.frame_set(frame)
            pos_curr = bob.location
            dt = 1.0 / bpy.context.scene.render.fps
            velocity = (pos_curr - pos_prev) / dt
            v = velocity.length
            KE = 0.5 * m * v * v
        else:
            KE = 0
        
        E_total = PE + KE
        energies.append(E_total)
    
    # Check energy conservation
    E_avg = sum(energies) / len(energies)
    E_std = math.sqrt(sum((e - E_avg)**2 for e in energies) / len(energies))
    conservation_error = (E_std / E_initial) * 100 if E_initial > 0 else 100
    
    result = {
        "test": "energy_conservation",
        "initial_energy": E_initial,
        "average_energy": E_avg,
        "energy_std_dev": E_std,
        "conservation_error_percent": conservation_error,
        "passed": conservation_error < 10.0  # 10% tolerance for discrete simulation
    }
    
    print(f"Initial energy: {E_initial:.2f}J")
    print(f"Average energy: {E_avg:.2f}J")
    print(f"Energy std dev: {E_std:.3f}J")
    print(f"Conservation error: {conservation_error:.2f}%")
    print(f"Test: {'PASSED' if result['passed'] else 'FAILED'}")
    
    return result

def test_collision_elasticity():
    """Test elastic collision conservation laws"""
    print("\n=== Elastic Collision Validation ===")
    
    clear_scene()
    setup_physics_world()
    
    # Create ground
    bpy.ops.mesh.primitive_plane_add(size=20, location=(0, 0, 0))
    ground = bpy.context.active_object
    bpy.ops.rigidbody.object_add()
    ground.rigid_body.type = 'PASSIVE'
    
    # Create two colliding spheres
    # Sphere 1 - moving
    bpy.ops.mesh.primitive_uv_sphere_add(
        radius=0.5,
        location=(-3, 0, 0.5)
    )
    sphere1 = bpy.context.active_object
    sphere1.name = "Sphere1"
    bpy.ops.rigidbody.object_add()
    sphere1.rigid_body.type = 'ACTIVE'
    sphere1.rigid_body.mass = 1.0
    sphere1.rigid_body.restitution = 1.0  # Perfect elasticity
    sphere1.rigid_body.friction = 0
    sphere1.rigid_body.linear_damping = 0
    
    # Sphere 2 - stationary
    bpy.ops.mesh.primitive_uv_sphere_add(
        radius=0.5,
        location=(0, 0, 0.5)
    )
    sphere2 = bpy.context.active_object
    sphere2.name = "Sphere2"
    bpy.ops.rigidbody.object_add()
    sphere2.rigid_body.type = 'ACTIVE'
    sphere2.rigid_body.mass = 1.0
    sphere2.rigid_body.restitution = 1.0
    sphere2.rigid_body.friction = 0
    sphere2.rigid_body.linear_damping = 0
    
    # Set initial velocity for sphere1
    bpy.context.scene.frame_set(1)
    sphere1.rigid_body.linear_velocity = (5, 0, 0)
    
    # Initial momentum
    p_initial = 5.0  # m1*v1 + m2*v2 = 1*5 + 1*0
    
    # Bake simulation
    bpy.ops.ptcache.bake_all(bake=True)
    
    # Measure momentum after collision
    collision_frame = None
    for frame in range(1, 100):
        bpy.context.scene.frame_set(frame)
        
        # Check for collision (spheres getting close then separating)
        dist = (sphere1.location - sphere2.location).length
        if dist < 1.1 and collision_frame is None:
            collision_frame = frame + 5  # A few frames after collision
            break
    
    if collision_frame:
        bpy.context.scene.frame_set(collision_frame)
        
        # Get velocities after collision (approximate)
        fps = bpy.context.scene.render.fps
        dt = 1.0 / fps
        
        # Sphere 1 velocity
        bpy.context.scene.frame_set(collision_frame - 1)
        pos1_prev = sphere1.location.copy()
        bpy.context.scene.frame_set(collision_frame)
        v1 = ((sphere1.location - pos1_prev) / dt).x
        
        # Sphere 2 velocity  
        bpy.context.scene.frame_set(collision_frame - 1)
        pos2_prev = sphere2.location.copy()
        bpy.context.scene.frame_set(collision_frame)
        v2 = ((sphere2.location - pos2_prev) / dt).x
        
        p_final = 1.0 * v1 + 1.0 * v2
        momentum_error = abs(p_final - p_initial) / p_initial * 100
        
        result = {
            "test": "elastic_collision",
            "initial_momentum": p_initial,
            "final_momentum": p_final,
            "momentum_error_percent": momentum_error,
            "v1_after": v1,
            "v2_after": v2,
            "passed": momentum_error < 10.0
        }
        
        print(f"Initial momentum: {p_initial:.2f} kg*m/s")
        print(f"Final momentum: {p_final:.2f} kg*m/s")
        print(f"Momentum error: {momentum_error:.2f}%")
        print(f"v1 after: {v1:.2f} m/s")
        print(f"v2 after: {v2:.2f} m/s")
        print(f"Test: {'PASSED' if result['passed'] else 'FAILED'}")
    else:
        result = {"test": "elastic_collision", "passed": False, "error": "No collision detected"}
        print("Test: FAILED - No collision detected")
    
    return result

def run_validation_suite():
    """Run all physics validation tests"""
    print("=" * 50)
    print("BLENDER PHYSICS VALIDATION SUITE")
    print("=" * 50)
    
    results = []
    
    # Test 1: Free fall
    clear_scene()
    setup_physics_world()
    results.append(test_free_fall())
    
    # Test 2: Projectile motion
    results.append(test_projectile_motion())
    
    # Test 3: Energy conservation
    results.append(test_conservation_of_energy())
    
    # Test 4: Elastic collision
    results.append(test_collision_elasticity())
    
    # Summary
    print("\n" + "=" * 50)
    print("VALIDATION SUMMARY")
    print("=" * 50)
    
    passed = sum(1 for r in results if r.get('passed', False))
    total = len(results)
    
    for result in results:
        status = "✓ PASSED" if result.get('passed', False) else "✗ FAILED"
        print(f"{result['test']}: {status}")
    
    print(f"\nOverall: {passed}/{total} tests passed")
    accuracy = (passed / total) * 100 if total > 0 else 0
    print(f"Physics Accuracy: {accuracy:.1f}%")
    
    # Export results
    with open("/tmp/blender_physics_validation.json", "w") as f:
        json.dump({
            "results": results,
            "summary": {
                "passed": passed,
                "total": total,
                "accuracy_percent": accuracy
            }
        }, f, indent=2)
    
    print("\nResults exported to: /tmp/blender_physics_validation.json")
    
    return accuracy >= 95.0  # Target: 95% accuracy

if __name__ == "__main__":
    success = run_validation_suite()
    exit(0 if success else 1)