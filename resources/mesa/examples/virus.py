"""
Virus on Network Model

Simulates virus spread through a network of connected agents.
"""

import random
import networkx as nx
from mesa import Agent, Model
from mesa.time import RandomActivation
from mesa.space import NetworkGrid
from mesa.datacollection import DataCollector


class VirusAgent(Agent):
    """An agent in the network that can be infected."""
    
    def __init__(self, unique_id, model, initial_state):
        super().__init__(unique_id, model)
        self.state = initial_state  # 0: susceptible, 1: infected, 2: resistant
    
    def try_to_infect_neighbors(self):
        """Try to infect neighbors if infected."""
        neighbors = self.model.grid.get_neighbors(self.pos, include_center=False)
        for neighbor in neighbors:
            if neighbor.state == 0:  # Susceptible
                if random.random() < self.model.virus_spread_chance:
                    neighbor.state = 1  # Infected
    
    def try_to_recover(self):
        """Try to recover from infection."""
        if random.random() < self.model.recovery_chance:
            if random.random() < self.model.gain_resistance_chance:
                self.state = 2  # Resistant
            else:
                self.state = 0  # Susceptible
    
    def step(self):
        """Agent step: spread virus or recover."""
        if self.state == 1:  # Infected
            self.try_to_infect_neighbors()
            self.try_to_recover()


class VirusOnNetwork(Model):
    """Virus spread model on a network."""
    
    def __init__(self, num_nodes=50, avg_node_degree=3, 
                 initial_infection=0.1, virus_spread_chance=0.4,
                 recovery_chance=0.3, gain_resistance_chance=0.05, seed=None):
        super().__init__()
        
        if seed is not None:
            random.seed(seed)
        
        self.num_nodes = num_nodes
        self.virus_spread_chance = virus_spread_chance
        self.recovery_chance = recovery_chance
        self.gain_resistance_chance = gain_resistance_chance
        
        # Create network
        prob = avg_node_degree / num_nodes
        self.G = nx.erdos_renyi_graph(n=num_nodes, p=prob, seed=seed)
        self.grid = NetworkGrid(self.G)
        self.schedule = RandomActivation(self)
        
        # Create agents
        self.nodes = []
        for i, node in enumerate(self.G.nodes()):
            # Initial infection
            initial_state = 1 if random.random() < initial_infection else 0
            agent = VirusAgent(i, self, initial_state)
            self.schedule.add(agent)
            self.grid.place_agent(agent, node)
            self.nodes.append(agent)
        
        self.datacollector = DataCollector(
            model_reporters={
                "Infected": lambda m: sum(1 for a in m.nodes if a.state == 1),
                "Susceptible": lambda m: sum(1 for a in m.nodes if a.state == 0),
                "Resistant": lambda m: sum(1 for a in m.nodes if a.state == 2)
            }
        )
    
    def step(self):
        """Advance model by one step."""
        self.datacollector.collect(self)
        self.schedule.step()