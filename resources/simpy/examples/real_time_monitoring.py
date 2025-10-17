#!/usr/bin/env python3
"""Real-time monitoring example for SimPy simulations"""

import simpy
import random
import json
import time

class MonitoredFactory:
    """A factory with real-time production monitoring"""
    
    def __init__(self, env, num_machines=3):
        self.env = env
        self.machines = simpy.Resource(env, capacity=num_machines)
        self.products_completed = 0
        self.production_log = []
    
    def produce_item(self, item_id):
        """Produce a single item with monitoring"""
        arrival_time = self.env.now
        
        # Log arrival
        self.log_event('item_arrived', {
            'item_id': item_id,
            'time': arrival_time
        })
        
        with self.machines.request() as request:
            yield request
            
            # Log production start
            wait_time = self.env.now - arrival_time
            self.log_event('production_started', {
                'item_id': item_id,
                'wait_time': wait_time,
                'time': self.env.now
            })
            
            # Production time
            production_time = random.uniform(2, 5)
            yield self.env.timeout(production_time)
            
            # Log completion
            self.products_completed += 1
            self.log_event('production_completed', {
                'item_id': item_id,
                'production_time': production_time,
                'total_time': self.env.now - arrival_time,
                'time': self.env.now,
                'total_completed': self.products_completed
            })
    
    def log_event(self, event_type, data):
        """Log production events"""
        event = {
            'timestamp': self.env.now,
            'type': event_type,
            'data': data
        }
        self.production_log.append(event)
        
        # Print for real-time monitoring
        print(f"[{self.env.now:.2f}] {event_type}: {json.dumps(data)}")
    
    def item_generator(self):
        """Generate items for production"""
        item_id = 0
        while True:
            # Items arrive randomly
            yield self.env.timeout(random.expovariate(1/2))
            item_id += 1
            self.env.process(self.produce_item(item_id))
    
    def monitor_utilization(self):
        """Monitor machine utilization"""
        while True:
            yield self.env.timeout(10)  # Check every 10 time units
            
            utilization = (self.machines.count / self.machines.capacity) * 100
            self.log_event('utilization_check', {
                'machines_busy': self.machines.count,
                'machines_total': self.machines.capacity,
                'utilization_percent': utilization,
                'queue_length': len(self.machines.queue)
            })

def run_monitored_simulation():
    """Run factory simulation with real-time monitoring"""
    print("Starting real-time monitored factory simulation...")
    print("-" * 50)
    
    # Set random seed for reproducibility
    random.seed(42)
    
    # Create environment
    env = simpy.Environment()
    
    # Create factory
    factory = MonitoredFactory(env, num_machines=3)
    
    # Start processes
    env.process(factory.item_generator())
    env.process(factory.monitor_utilization())
    
    # Run simulation
    env.run(until=100)
    
    print("-" * 50)
    print("Simulation completed!")
    
    # Calculate statistics
    total_items = factory.products_completed
    avg_production_time = sum(
        e['data']['production_time'] 
        for e in factory.production_log 
        if e['type'] == 'production_completed'
    ) / total_items if total_items > 0 else 0
    
    avg_wait_time = sum(
        e['data']['wait_time'] 
        for e in factory.production_log 
        if e['type'] == 'production_started'
    ) / total_items if total_items > 0 else 0
    
    # Summary statistics
    summary = {
        'simulation_time': 100,
        'total_items_produced': total_items,
        'average_production_time': round(avg_production_time, 2),
        'average_wait_time': round(avg_wait_time, 2),
        'throughput': round(total_items / 100, 2),
        'total_events_logged': len(factory.production_log)
    }
    
    print("\nSummary Statistics:")
    print(json.dumps(summary, indent=2))
    
    return summary

if __name__ == '__main__':
    result = run_monitored_simulation()