#!/usr/bin/env python3
"""
Process Optimization Simulation
Advanced discrete-event simulation for optimizing complex workflows and resource allocation
"""

import simpy
import numpy as np
import json
from typing import List, Dict, Tuple, Optional
from dataclasses import dataclass, field
from enum import Enum
import random

class ResourceType(Enum):
    """Types of resources in the system"""
    CPU = "cpu"
    MEMORY = "memory"
    GPU = "gpu"
    NETWORK = "network"
    STORAGE = "storage"

@dataclass
class Task:
    """Represents a task in the workflow"""
    id: str
    name: str
    duration: float
    resource_requirements: Dict[ResourceType, float]
    priority: int = 1
    dependencies: List[str] = field(default_factory=list)
    
@dataclass
class WorkflowMetrics:
    """Metrics collected during simulation"""
    total_time: float = 0
    resource_utilization: Dict[ResourceType, float] = field(default_factory=dict)
    bottlenecks: List[Tuple[str, float]] = field(default_factory=list)
    waiting_times: Dict[str, float] = field(default_factory=dict)
    throughput: float = 0
    cost: float = 0

class ResourcePool:
    """Manages a pool of resources with dynamic allocation"""
    
    def __init__(self, env: simpy.Environment, resource_type: ResourceType, 
                 capacity: float, cost_per_unit: float = 1.0):
        self.env = env
        self.resource_type = resource_type
        self.capacity = capacity
        self.available = capacity
        self.cost_per_unit = cost_per_unit
        self.utilization_history = []
        self.allocation_events = []
        
    def allocate(self, amount: float, task_id: str) -> bool:
        """Try to allocate resources"""
        if self.available >= amount:
            self.available -= amount
            self.allocation_events.append({
                'time': self.env.now,
                'task': task_id,
                'amount': amount,
                'action': 'allocate'
            })
            return True
        return False
        
    def release(self, amount: float, task_id: str):
        """Release resources back to pool"""
        self.available = min(self.available + amount, self.capacity)
        self.allocation_events.append({
            'time': self.env.now,
            'task': task_id,
            'amount': amount,
            'action': 'release'
        })
        
    def get_utilization(self) -> float:
        """Calculate current utilization percentage"""
        return ((self.capacity - self.available) / self.capacity) * 100
        
    def record_utilization(self):
        """Record utilization snapshot"""
        self.utilization_history.append({
            'time': self.env.now,
            'utilization': self.get_utilization()
        })

class OptimizationEngine:
    """Engine for optimizing process execution"""
    
    def __init__(self, env: simpy.Environment):
        self.env = env
        self.resource_pools: Dict[ResourceType, ResourcePool] = {}
        self.tasks_completed = []
        self.tasks_failed = []
        self.metrics = WorkflowMetrics()
        
    def add_resource_pool(self, resource_type: ResourceType, capacity: float, 
                         cost_per_unit: float = 1.0):
        """Add a resource pool to the system"""
        self.resource_pools[resource_type] = ResourcePool(
            self.env, resource_type, capacity, cost_per_unit
        )
        
    def calculate_task_score(self, task: Task, wait_time: float) -> float:
        """Calculate optimization score for task scheduling"""
        # Multi-objective scoring function
        priority_weight = 0.4
        wait_weight = 0.3
        resource_weight = 0.3
        
        # Normalize components
        priority_score = task.priority / 10.0
        wait_score = min(wait_time / 10.0, 1.0)
        resource_score = sum(task.resource_requirements.values()) / 100.0
        
        return (priority_weight * priority_score + 
                wait_weight * wait_score + 
                resource_weight * (1 - resource_score))
        
    def allocate_resources(self, task: Task) -> bool:
        """Try to allocate all required resources for a task"""
        # Check availability
        for resource_type, amount in task.resource_requirements.items():
            if resource_type in self.resource_pools:
                if self.resource_pools[resource_type].available < amount:
                    return False
                    
        # Allocate all resources
        for resource_type, amount in task.resource_requirements.items():
            if resource_type in self.resource_pools:
                self.resource_pools[resource_type].allocate(amount, task.id)
                
        return True
        
    def release_resources(self, task: Task):
        """Release all resources used by a task"""
        for resource_type, amount in task.resource_requirements.items():
            if resource_type in self.resource_pools:
                self.resource_pools[resource_type].release(amount, task.id)
                
    def identify_bottlenecks(self) -> List[Tuple[ResourceType, float]]:
        """Identify resource bottlenecks based on utilization"""
        bottlenecks = []
        for resource_type, pool in self.resource_pools.items():
            avg_utilization = np.mean([h['utilization'] for h in pool.utilization_history])
            if avg_utilization > 80:  # Threshold for bottleneck
                bottlenecks.append((resource_type, avg_utilization))
        return sorted(bottlenecks, key=lambda x: x[1], reverse=True)

class WorkflowSimulator:
    """Simulates workflow execution with optimization"""
    
    def __init__(self, env: simpy.Environment, engine: OptimizationEngine):
        self.env = env
        self.engine = engine
        self.task_queue: List[Task] = []
        self.completed_tasks: List[str] = []
        
    def add_task(self, task: Task):
        """Add task to workflow"""
        self.task_queue.append(task)
        
    def check_dependencies(self, task: Task) -> bool:
        """Check if all task dependencies are completed"""
        return all(dep in self.completed_tasks for dep in task.dependencies)
        
    def execute_task(self, task: Task):
        """Process to execute a single task"""
        start_time = self.env.now
        
        # Wait for dependencies
        while not self.check_dependencies(task):
            yield self.env.timeout(0.1)
            
        # Wait for resource allocation
        wait_start = self.env.now
        while not self.engine.allocate_resources(task):
            yield self.env.timeout(0.1)
            
        wait_time = self.env.now - wait_start
        self.engine.metrics.waiting_times[task.id] = wait_time
        
        print(f"[{self.env.now:.2f}] Task {task.name} started (waited {wait_time:.2f}s)")
        
        # Execute task
        yield self.env.timeout(task.duration)
        
        # Release resources
        self.engine.release_resources(task)
        
        # Mark as completed
        self.completed_tasks.append(task.id)
        self.engine.tasks_completed.append(task)
        
        execution_time = self.env.now - start_time
        print(f"[{self.env.now:.2f}] Task {task.name} completed (total time: {execution_time:.2f}s)")
        
    def optimize_schedule(self) -> List[Task]:
        """Optimize task scheduling based on multiple criteria"""
        # Sort tasks by optimization score
        scheduled = []
        remaining = self.task_queue.copy()
        
        while remaining:
            # Find tasks with satisfied dependencies
            ready_tasks = [t for t in remaining if self.check_dependencies(t)]
            
            if ready_tasks:
                # Score and sort ready tasks
                scored_tasks = [(t, self.engine.calculate_task_score(t, 0)) 
                               for t in ready_tasks]
                scored_tasks.sort(key=lambda x: x[1], reverse=True)
                
                # Select best task
                best_task = scored_tasks[0][0]
                scheduled.append(best_task)
                remaining.remove(best_task)
                self.completed_tasks.append(best_task.id)  # Simulate completion
                
        # Reset completed tasks for actual execution
        self.completed_tasks = []
        return scheduled
        
    def run_simulation(self):
        """Run the workflow simulation"""
        # Optimize schedule
        optimized_schedule = self.optimize_schedule()
        
        print(f"Optimized schedule: {[t.name for t in optimized_schedule]}")
        
        # Start task execution processes
        for task in optimized_schedule:
            self.env.process(self.execute_task(task))
            
        # Monitor resource utilization
        self.env.process(self.monitor_resources())
        
    def monitor_resources(self):
        """Monitor and record resource utilization"""
        while True:
            for pool in self.engine.resource_pools.values():
                pool.record_utilization()
            yield self.env.timeout(0.5)  # Sample every 0.5 seconds

def create_complex_workflow() -> List[Task]:
    """Create a complex workflow with dependencies"""
    tasks = [
        Task("t1", "Data Ingestion", 2.0, 
             {ResourceType.CPU: 20, ResourceType.NETWORK: 50}, 
             priority=5),
        Task("t2", "Data Validation", 1.5,
             {ResourceType.CPU: 10, ResourceType.MEMORY: 30},
             priority=4, dependencies=["t1"]),
        Task("t3", "Feature Engineering", 3.0,
             {ResourceType.CPU: 40, ResourceType.MEMORY: 60},
             priority=3, dependencies=["t2"]),
        Task("t4", "Model Training", 5.0,
             {ResourceType.GPU: 80, ResourceType.MEMORY: 70},
             priority=5, dependencies=["t3"]),
        Task("t5", "Model Validation", 2.0,
             {ResourceType.CPU: 30, ResourceType.GPU: 20},
             priority=4, dependencies=["t4"]),
        Task("t6", "Results Storage", 1.0,
             {ResourceType.STORAGE: 40, ResourceType.NETWORK: 20},
             priority=2, dependencies=["t5"]),
        Task("t7", "Report Generation", 1.5,
             {ResourceType.CPU: 15, ResourceType.MEMORY: 20},
             priority=3, dependencies=["t5"]),
        Task("t8", "Notification", 0.5,
             {ResourceType.NETWORK: 10},
             priority=1, dependencies=["t6", "t7"])
    ]
    return tasks

def main():
    """Run process optimization simulation"""
    print("="*60)
    print("Process Optimization Simulation")
    print("="*60)
    
    # Setup environment
    env = simpy.Environment()
    engine = OptimizationEngine(env)
    
    # Configure resource pools
    engine.add_resource_pool(ResourceType.CPU, 100, cost_per_unit=0.1)
    engine.add_resource_pool(ResourceType.MEMORY, 100, cost_per_unit=0.05)
    engine.add_resource_pool(ResourceType.GPU, 100, cost_per_unit=0.5)
    engine.add_resource_pool(ResourceType.NETWORK, 100, cost_per_unit=0.02)
    engine.add_resource_pool(ResourceType.STORAGE, 100, cost_per_unit=0.01)
    
    # Create workflow
    workflow = create_complex_workflow()
    simulator = WorkflowSimulator(env, engine)
    
    for task in workflow:
        simulator.add_task(task)
        
    # Run simulation
    simulator.run_simulation()
    env.run(until=30)
    
    # Calculate metrics
    engine.metrics.total_time = env.now
    engine.metrics.throughput = len(engine.tasks_completed) / env.now
    
    # Calculate resource utilization
    for resource_type, pool in engine.resource_pools.items():
        if pool.utilization_history:
            avg_utilization = np.mean([h['utilization'] for h in pool.utilization_history])
            engine.metrics.resource_utilization[resource_type] = avg_utilization
            
    # Identify bottlenecks
    bottlenecks = engine.identify_bottlenecks()
    engine.metrics.bottlenecks = [(r.value, u) for r, u in bottlenecks]
    
    # Calculate total cost
    total_cost = sum(
        len(pool.allocation_events) * pool.cost_per_unit 
        for pool in engine.resource_pools.values()
    )
    engine.metrics.cost = total_cost
    
    # Generate results
    results = {
        "simulation_type": "process_optimization",
        "total_time": engine.metrics.total_time,
        "tasks_completed": len(engine.tasks_completed),
        "tasks_failed": len(engine.tasks_failed),
        "throughput": engine.metrics.throughput,
        "total_cost": engine.metrics.cost,
        "resource_utilization": {
            k.value: f"{v:.2f}%" for k, v in engine.metrics.resource_utilization.items()
        },
        "bottlenecks": engine.metrics.bottlenecks,
        "average_wait_time": np.mean(list(engine.metrics.waiting_times.values()))
            if engine.metrics.waiting_times else 0,
        "optimization_recommendations": []
    }
    
    # Generate optimization recommendations
    if bottlenecks:
        results["optimization_recommendations"].append(
            f"Increase {bottlenecks[0][0]} capacity to reduce bottleneck"
        )
    
    if results["average_wait_time"] > 2.0:
        results["optimization_recommendations"].append(
            "Consider parallel task execution to reduce wait times"
        )
        
    print("\n" + "="*60)
    print("Simulation Results")
    print("="*60)
    print(json.dumps(results, indent=2))
    
    return results

if __name__ == "__main__":
    main()