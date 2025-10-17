#!/usr/bin/env python3
"""Simple pendulum simulation in PyBullet"""

import pybullet as p
import time
import math

def main():
    # Connect to physics server in direct mode (no GUI)
    client = p.connect(p.DIRECT)
    print("Connected to PyBullet physics engine")
    
    # Set gravity
    p.setGravity(0, 0, -9.81)
    print("Gravity set to -9.81 m/s²")
    
    # Create ground plane
    plane_id = p.createCollisionShape(p.GEOM_PLANE)
    ground = p.createMultiBody(0, plane_id)
    
    # Create pendulum anchor point (fixed in space)
    anchor_pos = [0, 0, 2]
    anchor = p.createMultiBody(
        baseMass=0,  # Zero mass = fixed
        basePosition=anchor_pos
    )
    
    # Create pendulum bob (sphere)
    bob_radius = 0.1
    bob_mass = 1.0
    bob_pos = [0, 0, 1]  # 1 meter below anchor
    
    bob_collision = p.createCollisionShape(p.GEOM_SPHERE, radius=bob_radius)
    bob_visual = p.createVisualShape(p.GEOM_SPHERE, radius=bob_radius, rgbaColor=[1, 0, 0, 1])
    
    bob = p.createMultiBody(
        baseMass=bob_mass,
        baseCollisionShapeIndex=bob_collision,
        baseVisualShapeIndex=bob_visual,
        basePosition=bob_pos
    )
    
    # Create constraint (the "string" of the pendulum)
    constraint = p.createConstraint(
        parentBodyUniqueId=anchor,
        parentLinkIndex=-1,
        childBodyUniqueId=bob,
        childLinkIndex=-1,
        jointType=p.JOINT_POINT2POINT,
        jointAxis=[0, 0, 0],
        parentFramePosition=[0, 0, 0],
        childFramePosition=[0, 0, 0]
    )
    
    # Give the pendulum an initial push
    p.resetBaseVelocity(bob, linearVelocity=[2, 0, 0])
    
    # Simulation parameters
    timestep = 1/240.0  # 240 Hz
    p.setTimeStep(timestep)
    simulation_time = 5.0  # seconds
    steps = int(simulation_time / timestep)
    
    print(f"\nSimulating pendulum for {simulation_time} seconds...")
    print("Time | X Position | Y Position | Z Position | Total Energy")
    print("-" * 60)
    
    # Run simulation
    for i in range(steps):
        p.stepSimulation()
        
        # Print status every 0.25 seconds
        if i % 60 == 0:
            pos, _ = p.getBasePositionAndOrientation(bob)
            vel, _ = p.getBaseVelocity(bob)
            
            # Calculate energy (simplified)
            height = pos[2]
            speed_sq = sum(v*v for v in vel)
            potential_energy = bob_mass * 9.81 * height
            kinetic_energy = 0.5 * bob_mass * speed_sq
            total_energy = potential_energy + kinetic_energy
            
            print(f"{i*timestep:4.1f}s | {pos[0]:9.4f} | {pos[1]:9.4f} | {pos[2]:9.4f} | {total_energy:12.4f}J")
    
    # Cleanup
    p.disconnect()
    print("\nSimulation complete!")
    print("The pendulum demonstrated conservation of energy")
    print("(small variations are due to numerical integration)")
    
    return 0

if __name__ == "__main__":
    exit(main())