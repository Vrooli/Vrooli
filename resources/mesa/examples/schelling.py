"""
Schelling Segregation Model

Simple agent-based model demonstrating how mild preferences 
for similar neighbors can lead to segregation.
"""

import random
from mesa import Agent, Model
from mesa.time import RandomActivation
from mesa.space import SingleGrid
from mesa.datacollection import DataCollector


class SchellingAgent(Agent):
    """An agent with a type (0 or 1) and happiness based on neighbors."""
    
    def __init__(self, unique_id, model, agent_type):
        super().__init__(unique_id, model)
        self.type = agent_type
        self.happy = False
    
    def step(self):
        """Check if happy and move if not."""
        similar = 0
        neighbors = self.model.grid.get_neighbors(
            self.pos, moore=True, include_center=False
        )
        
        if neighbors:
            for neighbor in neighbors:
                if neighbor.type == self.type:
                    similar += 1
            
            # Happy if at least homophily similar neighbors
            self.happy = similar >= self.model.homophily
        
        # If unhappy, move to empty cell
        if not self.happy:
            self.model.grid.move_to_empty(self)


class Schelling(Model):
    """Schelling segregation model."""
    
    def __init__(self, width=20, height=20, density=0.8, 
                 minority_fraction=0.2, homophily=3, seed=None):
        super().__init__()
        
        if seed is not None:
            random.seed(seed)
        
        self.width = width
        self.height = height
        self.density = density
        self.minority_fraction = minority_fraction
        self.homophily = homophily
        
        self.schedule = RandomActivation(self)
        self.grid = SingleGrid(width, height, torus=False)
        
        # Create agents
        agent_count = 0
        for x in range(width):
            for y in range(height):
                if random.random() < density:
                    agent_type = 1 if random.random() < minority_fraction else 0
                    agent = SchellingAgent(agent_count, self, agent_type)
                    agent_count += 1
                    self.grid.place_agent(agent, (x, y))
                    self.schedule.add(agent)
        
        self.datacollector = DataCollector(
            model_reporters={"Happy": lambda m: sum(a.happy for a in m.schedule.agents)},
            agent_reporters={"Type": "type", "Happy": "happy"}
        )
    
    def step(self):
        """Advance model by one step."""
        self.datacollector.collect(self)
        self.schedule.step()