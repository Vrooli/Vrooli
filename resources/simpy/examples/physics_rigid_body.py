#!/usr/bin/env python3
"""
Physics Simulation: Rigid Body Dynamics
Simulates rigid body physics including collisions, gravity, and momentum conservation
"""

import simpy
import numpy as np
import json
from dataclasses import dataclass, asdict
from typing import List, Tuple

@dataclass
class RigidBody:
    """Represents a rigid body with physics properties"""
    id: str
    mass: float
    position: np.ndarray
    velocity: np.ndarray
    force: np.ndarray
    radius: float  # For collision detection
    restitution: float = 0.8  # Bounciness

class PhysicsWorld:
    """Physics simulation environment"""
    
    def __init__(self, env: simpy.Environment, gravity: float = -9.81, dt: float = 0.01):
        self.env = env
        self.gravity = gravity
        self.dt = dt
        self.bodies: List[RigidBody] = []
        self.collision_events = []
        self.energy_history = []
        
    def add_body(self, body: RigidBody):
        """Add a rigid body to the simulation"""
        self.bodies.append(body)
        
    def apply_gravity(self, body: RigidBody):
        """Apply gravitational force"""
        body.force[1] += body.mass * self.gravity
        
    def integrate(self, body: RigidBody):
        """Euler integration for motion"""
        # F = ma -> a = F/m
        acceleration = body.force / body.mass
        
        # Update velocity and position
        body.velocity += acceleration * self.dt
        body.position += body.velocity * self.dt
        
        # Reset forces for next frame
        body.force = np.zeros(3)
        
    def check_collisions(self):
        """Check and resolve collisions between bodies"""
        for i, body1 in enumerate(self.bodies):
            for body2 in self.bodies[i+1:]:
                distance = np.linalg.norm(body1.position - body2.position)
                min_distance = body1.radius + body2.radius
                
                if distance < min_distance:
                    self.resolve_collision(body1, body2)
                    self.collision_events.append({
                        'time': self.env.now,
                        'bodies': [body1.id, body2.id],
                        'impact_velocity': np.linalg.norm(body1.velocity - body2.velocity)
                    })
                    
    def resolve_collision(self, body1: RigidBody, body2: RigidBody):
        """Resolve collision using conservation of momentum"""
        # Calculate relative velocity
        relative_velocity = body1.velocity - body2.velocity
        
        # Calculate collision normal
        collision_normal = (body1.position - body2.position)
        collision_normal = collision_normal / np.linalg.norm(collision_normal)
        
        # Calculate relative velocity in collision normal direction
        velocity_along_normal = np.dot(relative_velocity, collision_normal)
        
        # Don't resolve if bodies are moving apart
        if velocity_along_normal > 0:
            return
            
        # Calculate restitution (bounciness)
        e = min(body1.restitution, body2.restitution)
        
        # Calculate impulse scalar
        j = -(1 + e) * velocity_along_normal
        j /= 1/body1.mass + 1/body2.mass
        
        # Apply impulse
        impulse = j * collision_normal
        body1.velocity += impulse / body1.mass
        body2.velocity -= impulse / body2.mass
        
        # Separate bodies to prevent overlap
        overlap = (body1.radius + body2.radius) - np.linalg.norm(body1.position - body2.position)
        separation = collision_normal * (overlap / 2)
        body1.position += separation
        body2.position -= separation
        
    def check_boundaries(self, body: RigidBody, bounds: Tuple[float, float, float]):
        """Check and handle boundary collisions"""
        for i in range(3):
            if body.position[i] - body.radius < -bounds[i]:
                body.position[i] = -bounds[i] + body.radius
                body.velocity[i] *= -body.restitution
            elif body.position[i] + body.radius > bounds[i]:
                body.position[i] = bounds[i] - body.radius
                body.velocity[i] *= -body.restitution
                
    def calculate_energy(self) -> dict:
        """Calculate total system energy"""
        kinetic_energy = sum(0.5 * b.mass * np.linalg.norm(b.velocity)**2 for b in self.bodies)
        potential_energy = sum(b.mass * abs(self.gravity) * (b.position[1] + 10) for b in self.bodies)
        return {
            'kinetic': kinetic_energy,
            'potential': potential_energy,
            'total': kinetic_energy + potential_energy
        }
        
    def simulate_step(self):
        """Simulate one physics step"""
        # Apply forces
        for body in self.bodies:
            self.apply_gravity(body)
            
        # Check collisions
        self.check_collisions()
        
        # Integrate motion
        for body in self.bodies:
            self.integrate(body)
            self.check_boundaries(body, (10, 10, 10))
            
        # Record energy
        self.energy_history.append({
            'time': self.env.now,
            **self.calculate_energy()
        })

def physics_simulation_process(env: simpy.Environment, world: PhysicsWorld, duration: float):
    """Main simulation process"""
    steps = int(duration / world.dt)
    
    for _ in range(steps):
        world.simulate_step()
        yield env.timeout(world.dt)
        
        # Log progress every second
        if int(env.now) > int(env.now - world.dt):
            energy = world.calculate_energy()
            print(f"[{env.now:.1f}s] Bodies: {len(world.bodies)}, "
                  f"Energy: {energy['total']:.2f}J, "
                  f"Collisions: {len(world.collision_events)}")

def create_demo_scenario() -> PhysicsWorld:
    """Create a demonstration scenario with multiple bodies"""
    env = simpy.Environment()
    world = PhysicsWorld(env)
    
    # Create bouncing balls
    bodies = [
        RigidBody("ball1", mass=1.0, 
                 position=np.array([0.0, 5.0, 0.0]),
                 velocity=np.array([2.0, 0.0, 0.0]),
                 force=np.zeros(3), radius=0.5),
        RigidBody("ball2", mass=2.0,
                 position=np.array([3.0, 8.0, 0.0]),
                 velocity=np.array([-1.0, 0.0, 0.0]),
                 force=np.zeros(3), radius=0.7),
        RigidBody("ball3", mass=0.5,
                 position=np.array([-2.0, 3.0, 0.0]),
                 velocity=np.array([0.0, 5.0, 0.0]),
                 force=np.zeros(3), radius=0.3)
    ]
    
    for body in bodies:
        world.add_body(body)
        
    return env, world

def main():
    """Run physics simulation"""
    print("="*60)
    print("Rigid Body Physics Simulation")
    print("="*60)
    
    # Create scenario
    env, world = create_demo_scenario()
    
    # Run simulation
    duration = 10.0  # 10 seconds
    env.process(physics_simulation_process(env, world, duration))
    env.run(until=duration)
    
    # Report results
    results = {
        "simulation_type": "rigid_body_physics",
        "duration": duration,
        "dt": world.dt,
        "num_bodies": len(world.bodies),
        "total_collisions": len(world.collision_events),
        "final_energy": world.calculate_energy(),
        "energy_conservation": {
            "initial": world.energy_history[0]['total'] if world.energy_history else 0,
            "final": world.energy_history[-1]['total'] if world.energy_history else 0,
            "drift_percentage": abs(
                (world.energy_history[-1]['total'] - world.energy_history[0]['total']) 
                / world.energy_history[0]['total'] * 100
            ) if world.energy_history and world.energy_history[0]['total'] != 0 else 0
        },
        "collision_summary": {
            "total": len(world.collision_events),
            "average_impact_velocity": np.mean([c['impact_velocity'] for c in world.collision_events])
                if world.collision_events else 0
        }
    }
    
    print("\n" + "="*60)
    print("Simulation Results")
    print("="*60)
    print(json.dumps(results, indent=2))
    
    return results

if __name__ == "__main__":
    main()