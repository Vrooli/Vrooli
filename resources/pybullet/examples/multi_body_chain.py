#!/usr/bin/env python3
"""Multi-body chain simulation demonstrating soft constraints in PyBullet"""

import pybullet as p
import time
import math
import random

def create_chain(num_links=10, link_length=0.1):
    """Create a chain of connected bodies"""
    bodies = []
    constraints = []
    
    # Create spherical links
    for i in range(num_links):
        # Alternate colors for visual appeal
        color = [1, 0.5, 0, 1] if i % 2 == 0 else [0, 0.5, 1, 1]
        
        # Create sphere collision and visual shapes
        radius = 0.02
        collision_shape = p.createCollisionShape(p.GEOM_SPHERE, radius=radius)
        visual_shape = p.createVisualShape(p.GEOM_SPHERE, radius=radius, rgbaColor=color)
        
        # Position links in a line initially
        position = [0, 0, 2.0 - i * link_length]
        
        # Create body
        mass = 0.1 if i > 0 else 0  # First link is fixed
        body = p.createMultiBody(
            baseMass=mass,
            baseCollisionShapeIndex=collision_shape,
            baseVisualShapeIndex=visual_shape,
            basePosition=position
        )
        bodies.append(body)
        
        # Create constraint between consecutive links
        if i > 0:
            constraint = p.createConstraint(
                parentBodyUniqueId=bodies[i-1],
                parentLinkIndex=-1,
                childBodyUniqueId=body,
                childLinkIndex=-1,
                jointType=p.JOINT_POINT2POINT,
                jointAxis=[0, 0, 0],
                parentFramePosition=[0, 0, 0],
                childFramePosition=[0, 0, 0]
            )
            
            # Set constraint parameters for rope-like behavior
            p.changeConstraint(constraint, maxForce=100)
            constraints.append(constraint)
    
    return bodies, constraints

def create_soft_body_cloth(width=10, height=10, spacing=0.05):
    """Create a cloth-like structure using connected particles"""
    particles = []
    constraints = []
    
    # Create grid of particles
    for i in range(width):
        row = []
        for j in range(height):
            # Position in a grid
            x = (i - width/2) * spacing
            y = (j - height/2) * spacing
            z = 2.0
            
            # Color gradient
            color = [
                0.5 + 0.5 * i / width,
                0.2,
                0.5 + 0.5 * j / height,
                1.0
            ]
            
            # Create particle
            collision_shape = p.createCollisionShape(p.GEOM_SPHERE, radius=0.01)
            visual_shape = p.createVisualShape(p.GEOM_SPHERE, radius=0.01, rgbaColor=color)
            
            # Fixed particles at top edge
            mass = 0 if j == height-1 else 0.01
            
            particle = p.createMultiBody(
                baseMass=mass,
                baseCollisionShapeIndex=collision_shape,
                baseVisualShapeIndex=visual_shape,
                basePosition=[x, y, z]
            )
            row.append(particle)
        particles.append(row)
    
    # Create constraints between neighboring particles
    for i in range(width):
        for j in range(height):
            # Horizontal constraint
            if i < width - 1:
                constraint = p.createConstraint(
                    parentBodyUniqueId=particles[i][j],
                    parentLinkIndex=-1,
                    childBodyUniqueId=particles[i+1][j],
                    childLinkIndex=-1,
                    jointType=p.JOINT_POINT2POINT,
                    jointAxis=[0, 0, 0],
                    parentFramePosition=[0, 0, 0],
                    childFramePosition=[0, 0, 0]
                )
                p.changeConstraint(constraint, maxForce=10)
                constraints.append(constraint)
            
            # Vertical constraint
            if j < height - 1:
                constraint = p.createConstraint(
                    parentBodyUniqueId=particles[i][j],
                    parentLinkIndex=-1,
                    childBodyUniqueId=particles[i][j+1],
                    childLinkIndex=-1,
                    jointType=p.JOINT_POINT2POINT,
                    jointAxis=[0, 0, 0],
                    parentFramePosition=[0, 0, 0],
                    childFramePosition=[0, 0, 0]
                )
                p.changeConstraint(constraint, maxForce=10)
                constraints.append(constraint)
    
    return particles, constraints

def main():
    # Connect to physics server
    client = p.connect(p.DIRECT)
    print("Connected to PyBullet physics engine")
    
    # Configure simulation
    p.setGravity(0, 0, -9.81)
    timestep = 1/480.0  # Higher frequency for stable soft body simulation
    p.setTimeStep(timestep)
    
    # Create ground plane
    plane_shape = p.createCollisionShape(p.GEOM_PLANE)
    ground = p.createMultiBody(0, plane_shape)
    
    print("\nCreating multi-body systems...")
    
    # Create chain
    chain_bodies, chain_constraints = create_chain(num_links=15, link_length=0.08)
    print(f"Created chain with {len(chain_bodies)} links")
    
    # Create cloth
    cloth_particles, cloth_constraints = create_soft_body_cloth(width=8, height=8, spacing=0.04)
    print(f"Created cloth with {len(cloth_particles) * len(cloth_particles[0])} particles")
    
    # Create some obstacles
    obstacles = []
    for i in range(3):
        # Random obstacle properties
        size = random.uniform(0.1, 0.3)
        pos = [
            random.uniform(-0.5, 0.5),
            random.uniform(-0.5, 0.5),
            random.uniform(0.5, 1.5)
        ]
        color = [random.random(), random.random(), random.random(), 1]
        
        # Create obstacle
        collision_shape = p.createCollisionShape(p.GEOM_BOX, halfExtents=[size/2]*3)
        visual_shape = p.createVisualShape(p.GEOM_BOX, halfExtents=[size/2]*3, rgbaColor=color)
        
        obstacle = p.createMultiBody(
            baseMass=0.5,
            baseCollisionShapeIndex=collision_shape,
            baseVisualShapeIndex=visual_shape,
            basePosition=pos
        )
        obstacles.append(obstacle)
    
    print(f"Created {len(obstacles)} obstacles")
    
    # Apply initial disturbance to chain
    if len(chain_bodies) > 5:
        p.applyExternalForce(
            chain_bodies[5],
            -1,
            [50, 0, 0],
            [0, 0, 0],
            p.LINK_FRAME
        )
    
    print("\nSimulating multi-body dynamics...")
    print("Watch as the chain swings and the cloth falls onto obstacles")
    print("\nTime | Chain Energy | Cloth Center Height | Active Constraints")
    print("-" * 70)
    
    # Simulation loop
    simulation_time = 5.0
    steps = int(simulation_time / timestep)
    
    for step in range(steps):
        # Apply wind force to cloth periodically
        if step % 500 == 0 and step > 0:
            wind_force = [
                random.uniform(-2, 2),
                random.uniform(-2, 2),
                random.uniform(-1, 1)
            ]
            # Apply wind to center particles
            for i in range(2, 6):
                for j in range(2, 6):
                    if i < len(cloth_particles) and j < len(cloth_particles[0]):
                        p.applyExternalForce(
                            cloth_particles[i][j],
                            -1,
                            wind_force,
                            [0, 0, 0],
                            p.LINK_FRAME
                        )
        
        # Step simulation
        p.stepSimulation()
        
        # Print status every 0.5 seconds
        if step % 240 == 0:
            current_time = step * timestep
            
            # Calculate chain energy
            chain_energy = 0
            for body in chain_bodies[1:]:  # Skip fixed first link
                vel, _ = p.getBaseVelocity(body)
                pos, _ = p.getBasePositionAndOrientation(body)
                speed_sq = sum(v*v for v in vel)
                chain_energy += 0.1 * (0.5 * speed_sq + 9.81 * pos[2])
            
            # Get cloth center position
            center_i = len(cloth_particles) // 2
            center_j = len(cloth_particles[0]) // 2
            cloth_center_pos, _ = p.getBasePositionAndOrientation(
                cloth_particles[center_i][center_j]
            )
            
            # Count active constraints
            active_constraints = len(chain_constraints) + len(cloth_constraints)
            
            print(f"{current_time:4.1f}s | {chain_energy:12.4f}J | {cloth_center_pos[2]:18.4f}m | {active_constraints:10d}")
    
    print("\n" + "=" * 70)
    print("Simulation complete!")
    print("The multi-body systems demonstrated:")
    print("- Chain dynamics with point-to-point constraints")
    print("- Cloth-like behavior using particle grids")
    print("- Collision interactions with obstacles")
    print("- Energy conservation and dissipation")
    
    # Cleanup
    p.disconnect()
    
    return 0

if __name__ == "__main__":
    exit(main())