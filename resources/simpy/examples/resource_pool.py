#!/usr/bin/env python3
"""
Resource Pool Simulation Example

Demonstrates how to model shared resources (like worker agents, servers, or GPUs)
being used by multiple processes with different requirements and priorities.
This is useful for capacity planning and understanding bottlenecks in Vrooli scenarios.
"""

import simpy
import random
import json
from typing import List, Dict, Any


class ResourcePool:
    """Manages a pool of shared resources with usage tracking"""
    
    def __init__(self, env: simpy.Environment, capacity: int):
        self.env = env
        self.resource = simpy.Resource(env, capacity=capacity)
        self.stats = {
            'total_requests': 0,
            'completed_requests': 0,
            'total_wait_time': 0,
            'max_wait_time': 0,
            'utilization_samples': []
        }
    
    def request_resource(self, task_id: str, duration: float, priority: int = 1):
        """Request a resource for a specific duration"""
        self.stats['total_requests'] += 1
        arrival_time = self.env.now
        
        with self.resource.request() as req:
            # Wait for resource
            yield req
            
            # Calculate wait time
            wait_time = self.env.now - arrival_time
            self.stats['total_wait_time'] += wait_time
            self.stats['max_wait_time'] = max(self.stats['max_wait_time'], wait_time)
            
            # Use resource
            print(f"[{self.env.now:.1f}] Task {task_id} started (waited {wait_time:.1f})")
            yield self.env.timeout(duration)
            
            # Release resource
            self.stats['completed_requests'] += 1
            print(f"[{self.env.now:.1f}] Task {task_id} completed")
    
    def monitor_utilization(self):
        """Continuously monitor resource utilization"""
        while True:
            utilization = self.resource.count / self.resource.capacity
            self.stats['utilization_samples'].append({
                'time': self.env.now,
                'utilization': utilization * 100
            })
            yield self.env.timeout(1)  # Sample every time unit


def generate_tasks(env: simpy.Environment, pool: ResourcePool, 
                   task_rate: float, task_duration_range: tuple):
    """Generate tasks at a specified rate"""
    task_id = 0
    while True:
        # Wait for next task
        yield env.timeout(random.expovariate(task_rate))
        
        # Create task with random duration
        task_id += 1
        duration = random.uniform(*task_duration_range)
        env.process(pool.request_resource(f"task_{task_id}", duration))


def run_simulation(config: Dict[str, Any]) -> Dict[str, Any]:
    """Run the resource pool simulation with given configuration"""
    
    # Create environment
    env = simpy.Environment()
    
    # Create resource pool
    pool = ResourcePool(env, capacity=config['pool_size'])
    
    # Start monitoring
    env.process(pool.monitor_utilization())
    
    # Start task generation
    env.process(generate_tasks(
        env, pool, 
        config['task_rate'],
        config['task_duration_range']
    ))
    
    # Run simulation
    env.run(until=config['simulation_time'])
    
    # Calculate final statistics
    avg_wait_time = pool.stats['total_wait_time'] / max(1, pool.stats['total_requests'])
    avg_utilization = sum(s['utilization'] for s in pool.stats['utilization_samples']) / len(pool.stats['utilization_samples'])
    
    return {
        'config': config,
        'results': {
            'total_requests': pool.stats['total_requests'],
            'completed_requests': pool.stats['completed_requests'],
            'average_wait_time': round(avg_wait_time, 2),
            'max_wait_time': round(pool.stats['max_wait_time'], 2),
            'average_utilization': round(avg_utilization, 2),
            'throughput': pool.stats['completed_requests'] / config['simulation_time']
        },
        'utilization_timeline': pool.stats['utilization_samples'][:50]  # First 50 samples
    }


if __name__ == "__main__":
    # Example configuration - simulating Vrooli agent pool
    config = {
        'pool_size': 3,  # Number of available agents/resources
        'task_rate': 0.5,  # Average tasks per time unit
        'task_duration_range': (2, 8),  # Task duration between 2-8 time units
        'simulation_time': 100  # Total simulation time
    }
    
    print("Starting Resource Pool Simulation...")
    print(f"Configuration: {json.dumps(config, indent=2)}")
    print("\n" + "="*50 + "\n")
    
    # Run simulation
    results = run_simulation(config)
    
    print("\n" + "="*50)
    print("\nSimulation Results:")
    print(json.dumps(results['results'], indent=2))
    
    # Analyze bottlenecks
    print("\n📊 Analysis:")
    if results['results']['average_utilization'] > 80:
        print("⚠️  High utilization detected - consider adding more resources")
    if results['results']['average_wait_time'] > 5:
        print("⚠️  Long wait times - tasks are queuing significantly")
    if results['results']['max_wait_time'] > results['results']['average_wait_time'] * 3:
        print("⚠️  High wait time variance - some tasks experience much longer delays")
    
    # Output JSON for further processing
    print("\n💾 Full results saved to JSON format")
    print(json.dumps(results, indent=2))