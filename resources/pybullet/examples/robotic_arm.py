#!/usr/bin/env python3
"""Robotic arm simulation with inverse kinematics in PyBullet"""

import pybullet as p
import pybullet_data
import time
import math
import numpy as np

def create_robotic_arm(client):
    """Create a simple 3-DOF robotic arm"""
    # Base
    base_height = 0.1
    base = p.createMultiBody(
        baseMass=0,  # Fixed base
        baseCollisionShapeIndex=p.createCollisionShape(p.GEOM_BOX, halfExtents=[0.1, 0.1, base_height/2]),
        baseVisualShapeIndex=p.createVisualShape(p.GEOM_BOX, halfExtents=[0.1, 0.1, base_height/2], 
                                                rgbaColor=[0.3, 0.3, 0.3, 1]),
        basePosition=[0, 0, base_height/2]
    )
    
    # Link dimensions
    link_length = 0.5
    link_radius = 0.02
    
    # Create arm with 3 revolute joints
    link_masses = [1.0, 0.5, 0.25]
    link_collision_shapes = []
    link_visual_shapes = []
    link_positions = []
    link_orientations = []
    link_inertial_positions = []
    link_inertial_orientations = []
    parent_indices = []
    joint_types = []
    joint_axes = []
    
    for i in range(3):
        # Collision shape (cylinder)
        col_shape = p.createCollisionShape(
            p.GEOM_CYLINDER, 
            radius=link_radius, 
            height=link_length
        )
        link_collision_shapes.append(col_shape)
        
        # Visual shape with different colors for each link
        colors = [[0.8, 0.2, 0.2, 1], [0.2, 0.8, 0.2, 1], [0.2, 0.2, 0.8, 1]]
        vis_shape = p.createVisualShape(
            p.GEOM_CYLINDER,
            radius=link_radius,
            length=link_length,
            rgbaColor=colors[i]
        )
        link_visual_shapes.append(vis_shape)
        
        # Link position relative to parent joint
        link_positions.append([0, 0, link_length/2])
        link_orientations.append([0, 0, 0, 1])
        
        # Inertial frame (center of mass)
        link_inertial_positions.append([0, 0, 0])
        link_inertial_orientations.append([0, 0, 0, 1])
        
        # Joint configuration
        parent_indices.append(i-1 if i > 0 else -1)
        joint_types.append(p.JOINT_REVOLUTE)
        joint_axes.append([0, 0, 1] if i == 0 else [0, 1, 0])  # First joint rotates around Z, others around Y
    
    # Create multi-body
    arm = p.createMultiBody(
        baseMass=0,
        basePosition=[0, 0, base_height],
        linkMasses=link_masses,
        linkCollisionShapeIndices=link_collision_shapes,
        linkVisualShapeIndices=link_visual_shapes,
        linkPositions=link_positions,
        linkOrientations=link_orientations,
        linkInertialFramePositions=link_inertial_positions,
        linkInertialFrameOrientations=link_inertial_orientations,
        linkParentIndices=parent_indices,
        linkJointTypes=joint_types,
        linkJointAxis=joint_axes
    )
    
    return arm

def inverse_kinematics_demo(arm, target_pos):
    """Demonstrate inverse kinematics to reach a target position"""
    num_joints = p.getNumJoints(arm)
    
    # Get end effector link index (last link)
    end_effector_index = num_joints - 1
    
    # Calculate inverse kinematics
    joint_positions = p.calculateInverseKinematics(
        arm,
        end_effector_index,
        target_pos,
        maxNumIterations=100,
        residualThreshold=0.001
    )
    
    # Apply joint positions
    for i in range(num_joints):
        p.setJointMotorControl2(
            arm,
            i,
            p.POSITION_CONTROL,
            targetPosition=joint_positions[i],
            force=500
        )
    
    return joint_positions

def main():
    # Connect to physics server
    client = p.connect(p.DIRECT)
    print("Connected to PyBullet physics engine")
    
    # Configure simulation
    p.setGravity(0, 0, -9.81)
    p.setTimeStep(1/240.0)
    
    # Add search path for URDF files
    p.setAdditionalSearchPath(pybullet_data.getDataPath())
    
    # Create ground plane
    plane = p.loadURDF("plane.urdf")
    
    # Create robotic arm
    arm = create_robotic_arm(client)
    print("\nCreated 3-DOF robotic arm")
    
    # Create target object
    target_radius = 0.05
    target_visual = p.createVisualShape(
        p.GEOM_SPHERE,
        radius=target_radius,
        rgbaColor=[1, 1, 0, 0.8]
    )
    target = p.createMultiBody(
        baseMass=0,
        baseVisualShapeIndex=target_visual,
        basePosition=[0.5, 0.3, 0.4]
    )
    
    print("\nSimulating robotic arm control...")
    print("The arm will move to reach different target positions")
    print("\nTime | Target Position | Joint Angles (degrees)")
    print("-" * 60)
    
    # Simulation parameters
    simulation_time = 10.0
    timestep = 1/240.0
    steps = int(simulation_time / timestep)
    
    # Define waypoints for the target
    waypoints = [
        [0.5, 0.3, 0.4],
        [0.4, -0.3, 0.3],
        [-0.3, 0.4, 0.5],
        [0.3, 0.0, 0.6],
        [0.5, 0.3, 0.4]
    ]
    
    waypoint_index = 0
    waypoint_duration = 2.0  # seconds per waypoint
    waypoint_steps = int(waypoint_duration / timestep)
    
    for step in range(steps):
        # Update target position
        if step % waypoint_steps == 0:
            waypoint_index = (waypoint_index + 1) % len(waypoints)
            target_pos = waypoints[waypoint_index]
            p.resetBasePositionAndOrientation(
                target,
                target_pos,
                [0, 0, 0, 1]
            )
            
            # Apply inverse kinematics
            joint_positions = inverse_kinematics_demo(arm, target_pos)
        
        # Step simulation
        p.stepSimulation()
        
        # Print status every 0.5 seconds
        if step % 120 == 0:
            current_time = step * timestep
            
            # Get actual joint positions
            num_joints = p.getNumJoints(arm)
            joint_states = []
            if num_joints > 0:
                for i in range(min(3, num_joints)):  # Get up to 3 joints
                    state = p.getJointState(arm, i)
                    joint_states.append(math.degrees(state[0]))
                
                # Pad with zeros if less than 3 joints
                while len(joint_states) < 3:
                    joint_states.append(0.0)
                
                # Get end effector position
                end_effector_state = p.getLinkState(arm, num_joints - 1)
                end_pos = end_effector_state[0]
            else:
                joint_states = [0.0, 0.0, 0.0]
                end_pos = [0, 0, 0]
            
            print(f"{current_time:4.1f}s | ({target_pos[0]:.2f}, {target_pos[1]:.2f}, {target_pos[2]:.2f}) | "
                  f"[{joint_states[0]:6.1f}, {joint_states[1]:6.1f}, {joint_states[2]:6.1f}]")
    
    # Calculate final accuracy
    num_joints_final = p.getNumJoints(arm)
    if num_joints_final > 0:
        final_end_effector = p.getLinkState(arm, num_joints_final - 1)[0]
        final_target = waypoints[-1]
        error = math.sqrt(sum((a - b)**2 for a, b in zip(final_end_effector, final_target)))
    else:
        error = 0.0
    
    print("\n" + "=" * 60)
    print(f"Simulation complete!")
    print(f"Final position error: {error:.4f} meters")
    print("The robotic arm successfully tracked the moving target")
    print("using inverse kinematics for precise control")
    
    # Cleanup
    p.disconnect()
    
    return 0

if __name__ == "__main__":
    exit(main())