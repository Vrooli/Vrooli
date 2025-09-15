#!/usr/bin/env python3
"""Bouncing ball simulation demonstrating restitution and damping"""

import pybullet as p
import pybullet_data
import time

def main():
    # Connect to physics server
    client = p.connect(p.DIRECT)
    print("Connected to PyBullet physics engine")
    
    # Set gravity
    p.setGravity(0, 0, -9.81)
    
    # Load plane from PyBullet data
    p.setAdditionalSearchPath(pybullet_data.getDataPath())
    plane_id = p.loadURDF("plane.urdf")
    print("Ground plane loaded")
    
    # Create three balls with different properties
    balls = []
    properties = [
        {"pos": [-1, 0, 3], "restitution": 0.9, "color": [1, 0, 0, 1], "name": "High bounce (red)"},
        {"pos": [0, 0, 3], "restitution": 0.5, "color": [0, 1, 0, 1], "name": "Medium bounce (green)"},
        {"pos": [1, 0, 3], "restitution": 0.1, "color": [0, 0, 1, 1], "name": "Low bounce (blue)"},
    ]
    
    for prop in properties:
        # Create ball
        ball_radius = 0.2
        ball_mass = 1.0
        
        ball_collision = p.createCollisionShape(p.GEOM_SPHERE, radius=ball_radius)
        ball_visual = p.createVisualShape(
            p.GEOM_SPHERE, 
            radius=ball_radius, 
            rgbaColor=prop["color"]
        )
        
        ball_id = p.createMultiBody(
            baseMass=ball_mass,
            baseCollisionShapeIndex=ball_collision,
            baseVisualShapeIndex=ball_visual,
            basePosition=prop["pos"]
        )
        
        # Set restitution (bounciness)
        p.changeDynamics(ball_id, -1, restitution=prop["restitution"])
        
        balls.append({"id": ball_id, "name": prop["name"], "initial_height": prop["pos"][2]})
        print(f"Created ball: {prop['name']}")
    
    # Simulation parameters
    timestep = 1/240.0
    p.setTimeStep(timestep)
    simulation_time = 10.0
    steps = int(simulation_time / timestep)
    
    print(f"\nSimulating {len(balls)} bouncing balls for {simulation_time} seconds...")
    print("Watch how different restitution values affect bouncing behavior!\n")
    print("Time | Red Ball Height | Green Ball Height | Blue Ball Height")
    print("-" * 65)
    
    # Run simulation
    max_heights = {ball["id"]: 0 for ball in balls}
    
    for i in range(steps):
        p.stepSimulation()
        
        # Update max heights
        for ball in balls:
            pos, _ = p.getBasePositionAndOrientation(ball["id"])
            if pos[2] > max_heights[ball["id"]]:
                max_heights[ball["id"]] = pos[2]
        
        # Print status every 0.5 seconds
        if i % 120 == 0:
            heights = []
            for ball in balls:
                pos, _ = p.getBasePositionAndOrientation(ball["id"])
                heights.append(pos[2])
            
            print(f"{i*timestep:4.1f}s | {heights[0]:15.4f} | {heights[1]:17.4f} | {heights[2]:16.4f}")
    
    # Final summary
    print("\n" + "=" * 65)
    print("SIMULATION COMPLETE")
    print("=" * 65)
    print("\nMax bounce heights achieved:")
    for ball in balls:
        max_h = max_heights[ball["id"]]
        efficiency = (max_h / ball["initial_height"]) * 100
        print(f"  {ball['name']:25s}: {max_h:.3f}m ({efficiency:.1f}% of initial)")
    
    print("\nKey observations:")
    print("- Higher restitution = more energy retained after bounce")
    print("- Restitution of 1.0 would be perfectly elastic (no energy loss)")
    print("- Real materials have restitution < 1.0 due to energy dissipation")
    
    # Cleanup
    p.disconnect()
    return 0

if __name__ == "__main__":
    exit(main())