#!/usr/bin/env python3
"""Example of SimPy simulation with Qdrant integration for storing and searching results"""

import simpy
import random
import json
import numpy as np
from datetime import datetime

# Mock Qdrant client for demonstration (replace with actual when Qdrant is available)
class SimulationStore:
    """Store and search simulation results (Qdrant-ready)"""
    
    def __init__(self):
        self.simulations = []
    
    def store_simulation(self, simulation_data):
        """Store simulation results with embedding"""
        # Generate embedding from simulation parameters
        # In production, use a proper embedding model
        embedding = self.generate_embedding(simulation_data)
        
        # Store with metadata
        stored_sim = {
            'id': f"sim_{len(self.simulations)}",
            'timestamp': datetime.now().isoformat(),
            'embedding': embedding,
            'data': simulation_data
        }
        
        self.simulations.append(stored_sim)
        return stored_sim['id']
    
    def generate_embedding(self, data):
        """Generate embedding vector from simulation data"""
        # Create feature vector from simulation parameters
        features = [
            data.get('num_servers', 0),
            data.get('arrival_rate', 0),
            data.get('service_rate', 0),
            data.get('simulation_time', 0),
            data.get('total_customers', 0),
            data.get('avg_wait_time', 0),
            data.get('avg_service_time', 0),
            data.get('utilization', 0)
        ]
        
        # Normalize and pad to 128 dimensions
        embedding = np.zeros(128)
        embedding[:len(features)] = features
        return embedding.tolist()
    
    def search_similar(self, query_params, top_k=5):
        """Search for similar simulations"""
        query_embedding = self.generate_embedding(query_params)
        
        # Calculate cosine similarity
        results = []
        for sim in self.simulations:
            similarity = self.cosine_similarity(
                query_embedding, 
                sim['embedding']
            )
            results.append({
                'id': sim['id'],
                'similarity': similarity,
                'data': sim['data']
            })
        
        # Sort by similarity and return top_k
        results.sort(key=lambda x: x['similarity'], reverse=True)
        return results[:top_k]
    
    def cosine_similarity(self, vec1, vec2):
        """Calculate cosine similarity between two vectors"""
        vec1 = np.array(vec1)
        vec2 = np.array(vec2)
        
        dot_product = np.dot(vec1, vec2)
        norm1 = np.linalg.norm(vec1)
        norm2 = np.linalg.norm(vec2)
        
        if norm1 == 0 or norm2 == 0:
            return 0
        
        return dot_product / (norm1 * norm2)

class QueueingSystem:
    """Queueing system simulation with pattern storage"""
    
    def __init__(self, env, num_servers, service_rate):
        self.env = env
        self.servers = simpy.Resource(env, capacity=num_servers)
        self.num_servers = num_servers
        self.service_rate = service_rate
        self.customers_served = 0
        self.wait_times = []
        self.service_times = []
    
    def serve_customer(self, customer_id):
        """Serve a customer"""
        arrival_time = self.env.now
        
        with self.servers.request() as request:
            yield request
            
            wait_time = self.env.now - arrival_time
            self.wait_times.append(wait_time)
            
            # Service time
            service_time = random.expovariate(self.service_rate)
            self.service_times.append(service_time)
            yield self.env.timeout(service_time)
            
            self.customers_served += 1

def run_queueing_simulation(num_servers=2, arrival_rate=1.0, service_rate=1.5, sim_time=100):
    """Run queueing simulation with configurable parameters"""
    
    # Create environment
    env = simpy.Environment()
    system = QueueingSystem(env, num_servers, service_rate)
    
    # Customer arrival process
    def customer_arrivals():
        customer_id = 0
        while True:
            yield env.timeout(random.expovariate(arrival_rate))
            customer_id += 1
            env.process(system.serve_customer(customer_id))
    
    # Start arrivals
    env.process(customer_arrivals())
    
    # Run simulation
    env.run(until=sim_time)
    
    # Calculate metrics
    avg_wait = np.mean(system.wait_times) if system.wait_times else 0
    avg_service = np.mean(system.service_times) if system.service_times else 0
    utilization = (sum(system.service_times) / (sim_time * num_servers)) * 100
    
    return {
        'num_servers': num_servers,
        'arrival_rate': arrival_rate,
        'service_rate': service_rate,
        'simulation_time': sim_time,
        'total_customers': system.customers_served,
        'avg_wait_time': round(avg_wait, 2),
        'avg_service_time': round(avg_service, 2),
        'utilization': round(utilization, 2)
    }

def demonstrate_pattern_learning():
    """Demonstrate storing and searching simulation patterns"""
    
    print("SimPy + Qdrant Integration Demo")
    print("=" * 50)
    
    # Create simulation store (Qdrant interface)
    store = SimulationStore()
    
    # Run multiple simulations with different parameters
    print("\n1. Running simulations with various parameters...")
    
    configurations = [
        {'num_servers': 1, 'arrival_rate': 0.8, 'service_rate': 1.0},
        {'num_servers': 2, 'arrival_rate': 1.5, 'service_rate': 1.0},
        {'num_servers': 3, 'arrival_rate': 2.0, 'service_rate': 1.0},
        {'num_servers': 2, 'arrival_rate': 1.0, 'service_rate': 0.8},
        {'num_servers': 2, 'arrival_rate': 1.0, 'service_rate': 1.5},
    ]
    
    for i, config in enumerate(configurations):
        print(f"\n   Simulation {i+1}: {config}")
        
        # Run simulation
        result = run_queueing_simulation(
            num_servers=config['num_servers'],
            arrival_rate=config['arrival_rate'],
            service_rate=config['service_rate'],
            sim_time=100
        )
        
        # Store in Qdrant
        sim_id = store.store_simulation(result)
        print(f"   Stored as: {sim_id}")
        print(f"   Results: Customers={result['total_customers']}, "
              f"Avg Wait={result['avg_wait_time']}, "
              f"Utilization={result['utilization']}%")
    
    # Search for similar simulations
    print("\n2. Searching for similar simulation patterns...")
    
    query = {
        'num_servers': 2,
        'arrival_rate': 1.2,
        'service_rate': 1.1,
        'simulation_time': 100
    }
    
    print(f"\n   Query parameters: {query}")
    print("\n   Similar simulations found:")
    
    similar = store.search_similar(query, top_k=3)
    
    for rank, result in enumerate(similar, 1):
        print(f"\n   Rank {rank}: {result['id']}")
        print(f"   Similarity: {result['similarity']:.3f}")
        print(f"   Config: servers={result['data']['num_servers']}, "
              f"arrival={result['data']['arrival_rate']}, "
              f"service={result['data']['service_rate']}")
        print(f"   Performance: wait={result['data']['avg_wait_time']}, "
              f"utilization={result['data']['utilization']}%")
    
    print("\n" + "=" * 50)
    print("Demo complete! This pattern can be used with actual Qdrant")
    print("for persistent storage and advanced vector search capabilities.")
    
    return {
        'simulations_stored': len(store.simulations),
        'search_performed': True,
        'integration_ready': True
    }

if __name__ == '__main__':
    random.seed(42)
    result = demonstrate_pattern_learning()
    print(f"\nFinal result: {json.dumps(result, indent=2)}")