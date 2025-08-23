#!/usr/bin/env python3
"""
Basic Queue Simulation Example

This simulation models a simple M/M/1 queue system:
- Customers arrive at random intervals
- A single server processes them
- We track waiting times and queue lengths
"""

import simpy
import random
import statistics

# Configuration
RANDOM_SEED = 42
NUM_CUSTOMERS = 100
ARRIVAL_RATE = 2.0  # Average time between arrivals
SERVICE_RATE = 1.8  # Average service time
SIM_TIME = 200  # Total simulation time

# Metrics
waiting_times = []
queue_lengths = []

def customer(env, name, server):
    """Customer process"""
    arrival_time = env.now
    print(f"[{env.now:7.2f}] Customer {name} arrives")
    
    # Record queue length before joining
    queue_lengths.append(len(server.queue))
    
    with server.request() as request:
        # Wait for server
        yield request
        wait_time = env.now - arrival_time
        waiting_times.append(wait_time)
        print(f"[{env.now:7.2f}] Customer {name} begins service (waited {wait_time:.2f})")
        
        # Service time
        service_time = random.expovariate(1.0 / SERVICE_RATE)
        yield env.timeout(service_time)
        
        print(f"[{env.now:7.2f}] Customer {name} departs")

def customer_generator(env, server):
    """Generate customers at random intervals"""
    for i in range(NUM_CUSTOMERS):
        # Create customer
        env.process(customer(env, i, server))
        
        # Wait until next arrival
        arrival_interval = random.expovariate(1.0 / ARRIVAL_RATE)
        yield env.timeout(arrival_interval)

def main():
    """Run the simulation"""
    print("=" * 60)
    print("M/M/1 Queue Simulation")
    print("=" * 60)
    print(f"Number of customers: {NUM_CUSTOMERS}")
    print(f"Average arrival interval: {ARRIVAL_RATE}")
    print(f"Average service time: {SERVICE_RATE}")
    print(f"Simulation time: {SIM_TIME}")
    print("=" * 60)
    
    # Set random seed for reproducibility
    random.seed(RANDOM_SEED)
    
    # Create environment and server
    env = simpy.Environment()
    server = simpy.Resource(env, capacity=1)
    
    # Start customer generation
    env.process(customer_generator(env, server))
    
    # Run simulation
    env.run(until=SIM_TIME)
    
    # Report statistics
    print("\n" + "=" * 60)
    print("Simulation Results")
    print("=" * 60)
    
    if waiting_times:
        print(f"Customers served: {len(waiting_times)}")
        print(f"Average waiting time: {statistics.mean(waiting_times):.2f}")
        print(f"Maximum waiting time: {max(waiting_times):.2f}")
        print(f"Minimum waiting time: {min(waiting_times):.2f}")
        
    if queue_lengths:
        print(f"Average queue length: {statistics.mean(queue_lengths):.2f}")
        print(f"Maximum queue length: {max(queue_lengths)}")
    
    # Calculate utilization
    utilization = (len(waiting_times) * SERVICE_RATE) / SIM_TIME
    print(f"Server utilization: {utilization:.2%}")
    print("=" * 60)

if __name__ == "__main__":
    main()