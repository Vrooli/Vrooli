#!/usr/bin/env python3
"""
Supply Chain Network Simulation
Complex multi-echelon supply chain with inventory management and demand forecasting
"""

import simpy
import numpy as np
import json
from typing import List, Dict, Optional, Tuple
from dataclasses import dataclass, field
from enum import Enum
import random

class NodeType(Enum):
    """Types of nodes in supply chain"""
    SUPPLIER = "supplier"
    MANUFACTURER = "manufacturer"
    WAREHOUSE = "warehouse"
    RETAILER = "retailer"
    CUSTOMER = "customer"

@dataclass
class Product:
    """Represents a product in the supply chain"""
    id: str
    name: str
    unit_cost: float
    holding_cost: float
    lead_time: float
    perishable: bool = False
    shelf_life: float = float('inf')

@dataclass
class Order:
    """Represents an order in the system"""
    id: str
    product_id: str
    quantity: int
    source_node: str
    destination_node: str
    order_time: float
    required_by: Optional[float] = None
    priority: int = 1
    
@dataclass
class InventoryPolicy:
    """Inventory management policy"""
    reorder_point: int
    order_quantity: int
    safety_stock: int
    review_period: float = 1.0

class SupplyChainNode:
    """Base class for supply chain nodes"""
    
    def __init__(self, env: simpy.Environment, node_id: str, name: str, 
                 node_type: NodeType, capacity: int = 1000):
        self.env = env
        self.node_id = node_id
        self.name = name
        self.node_type = node_type
        self.capacity = capacity
        self.inventory: Dict[str, int] = {}
        self.backlog: List[Order] = []
        self.suppliers: List['SupplyChainNode'] = []
        self.customers: List['SupplyChainNode'] = []
        self.policies: Dict[str, InventoryPolicy] = {}
        self.metrics = {
            'orders_received': 0,
            'orders_fulfilled': 0,
            'stockouts': 0,
            'holding_cost': 0.0,
            'service_level': 1.0,
            'inventory_turnover': 0.0
        }
        
    def add_supplier(self, supplier: 'SupplyChainNode'):
        """Add a supplier node"""
        self.suppliers.append(supplier)
        
    def add_customer(self, customer: 'SupplyChainNode'):
        """Add a customer node"""
        self.customers.append(customer)
        
    def set_inventory_policy(self, product_id: str, policy: InventoryPolicy):
        """Set inventory policy for a product"""
        self.policies[product_id] = policy
        
    def check_inventory(self, product_id: str) -> int:
        """Check current inventory level"""
        return self.inventory.get(product_id, 0)
        
    def consume_inventory(self, product_id: str, quantity: int) -> bool:
        """Consume inventory if available"""
        if self.check_inventory(product_id) >= quantity:
            self.inventory[product_id] -= quantity
            return True
        return False
        
    def add_inventory(self, product_id: str, quantity: int):
        """Add to inventory"""
        if product_id not in self.inventory:
            self.inventory[product_id] = 0
        self.inventory[product_id] = min(
            self.inventory[product_id] + quantity, 
            self.capacity
        )
        
    def process_order(self, order: Order):
        """Process an incoming order"""
        self.metrics['orders_received'] += 1
        
        # Check if we can fulfill
        if self.consume_inventory(order.product_id, order.quantity):
            self.metrics['orders_fulfilled'] += 1
            print(f"[{self.env.now:.2f}] {self.name} fulfilled order {order.id}")
            return True
        else:
            # Add to backlog
            self.backlog.append(order)
            self.metrics['stockouts'] += 1
            print(f"[{self.env.now:.2f}] {self.name} stockout for order {order.id}")
            return False
            
    def review_inventory(self, product: Product):
        """Review inventory and place orders if needed"""
        while True:
            if product.id in self.policies:
                policy = self.policies[product.id]
                current_level = self.check_inventory(product.id)
                
                # Check reorder point
                if current_level <= policy.reorder_point:
                    # Place order with supplier
                    if self.suppliers:
                        supplier = random.choice(self.suppliers)
                        order = Order(
                            id=f"ORD_{self.node_id}_{self.env.now}",
                            product_id=product.id,
                            quantity=policy.order_quantity,
                            source_node=supplier.node_id,
                            destination_node=self.node_id,
                            order_time=self.env.now
                        )
                        print(f"[{self.env.now:.2f}] {self.name} placing order for {policy.order_quantity} units")
                        self.env.process(self.place_order(order, supplier, product))
                        
                # Wait for review period
                yield self.env.timeout(policy.review_period)
            else:
                yield self.env.timeout(1.0)
                
    def place_order(self, order: Order, supplier: 'SupplyChainNode', product: Product):
        """Place order with supplier"""
        # Process at supplier
        if supplier.process_order(order):
            # Wait for lead time
            yield self.env.timeout(product.lead_time)
            # Receive inventory
            self.add_inventory(product.id, order.quantity)
            print(f"[{self.env.now:.2f}] {self.name} received {order.quantity} units of {product.id}")
            
            # Try to fulfill backlog
            self.process_backlog(product.id)
            
    def process_backlog(self, product_id: str):
        """Try to fulfill backlogged orders"""
        fulfilled = []
        for order in self.backlog:
            if order.product_id == product_id:
                if self.consume_inventory(product_id, order.quantity):
                    self.metrics['orders_fulfilled'] += 1
                    fulfilled.append(order)
                    print(f"[{self.env.now:.2f}] {self.name} fulfilled backlogged order {order.id}")
                    
        # Remove fulfilled orders from backlog
        for order in fulfilled:
            self.backlog.remove(order)
            
    def calculate_metrics(self) -> dict:
        """Calculate performance metrics"""
        total_orders = self.metrics['orders_received']
        if total_orders > 0:
            self.metrics['service_level'] = self.metrics['orders_fulfilled'] / total_orders
            
        # Calculate holding cost
        for product_id, quantity in self.inventory.items():
            self.metrics['holding_cost'] += quantity * 0.1  # Simple holding cost
            
        return self.metrics

class DemandGenerator:
    """Generates customer demand"""
    
    def __init__(self, env: simpy.Environment, mean_demand: float = 10, 
                 variability: float = 3, seasonality: bool = True):
        self.env = env
        self.mean_demand = mean_demand
        self.variability = variability
        self.seasonality = seasonality
        
    def generate_demand(self) -> int:
        """Generate demand based on patterns"""
        base_demand = np.random.poisson(self.mean_demand)
        
        if self.seasonality:
            # Add seasonal pattern
            season_factor = 1 + 0.3 * np.sin(2 * np.pi * self.env.now / 30)
            base_demand = int(base_demand * season_factor)
            
        # Add random variability
        noise = np.random.normal(0, self.variability)
        demand = max(0, int(base_demand + noise))
        
        return demand

class SupplyChainSimulator:
    """Simulates the entire supply chain network"""
    
    def __init__(self, env: simpy.Environment):
        self.env = env
        self.nodes: Dict[str, SupplyChainNode] = {}
        self.products: Dict[str, Product] = {}
        self.demand_generator = DemandGenerator(env)
        self.metrics = {
            'total_orders': 0,
            'fulfilled_orders': 0,
            'total_stockouts': 0,
            'average_lead_time': 0,
            'total_cost': 0,
            'bullwhip_effect': 0
        }
        
    def add_node(self, node: SupplyChainNode):
        """Add node to network"""
        self.nodes[node.node_id] = node
        
    def add_product(self, product: Product):
        """Add product to catalog"""
        self.products[product.id] = product
        
    def create_network(self):
        """Create a multi-echelon supply chain network"""
        # Create nodes
        supplier = SupplyChainNode(self.env, "SUP1", "Main Supplier", 
                                  NodeType.SUPPLIER, capacity=10000)
        manufacturer = SupplyChainNode(self.env, "MAN1", "Manufacturer", 
                                      NodeType.MANUFACTURER, capacity=5000)
        warehouse = SupplyChainNode(self.env, "WAR1", "Central Warehouse", 
                                   NodeType.WAREHOUSE, capacity=3000)
        retailer1 = SupplyChainNode(self.env, "RET1", "Retailer 1", 
                                   NodeType.RETAILER, capacity=1000)
        retailer2 = SupplyChainNode(self.env, "RET2", "Retailer 2", 
                                   NodeType.RETAILER, capacity=1000)
        
        # Set up relationships
        manufacturer.add_supplier(supplier)
        warehouse.add_supplier(manufacturer)
        retailer1.add_supplier(warehouse)
        retailer2.add_supplier(warehouse)
        
        supplier.add_customer(manufacturer)
        manufacturer.add_customer(warehouse)
        warehouse.add_customer(retailer1)
        warehouse.add_customer(retailer2)
        
        # Add to network
        for node in [supplier, manufacturer, warehouse, retailer1, retailer2]:
            self.add_node(node)
            
        # Create products
        product = Product("PROD1", "Widget A", unit_cost=10, holding_cost=0.5, lead_time=2.0)
        self.add_product(product)
        
        # Set inventory policies
        supplier.inventory["PROD1"] = 1000  # Initial inventory
        supplier.set_inventory_policy("PROD1", InventoryPolicy(100, 500, 50))
        
        manufacturer.set_inventory_policy("PROD1", InventoryPolicy(200, 400, 100))
        warehouse.set_inventory_policy("PROD1", InventoryPolicy(150, 300, 75))
        retailer1.set_inventory_policy("PROD1", InventoryPolicy(50, 100, 25))
        retailer2.set_inventory_policy("PROD1", InventoryPolicy(50, 100, 25))
        
        # Start inventory review processes
        for node in self.nodes.values():
            if node.node_type != NodeType.SUPPLIER:
                self.env.process(node.review_inventory(product))
                
    def customer_demand_process(self):
        """Generate customer demand at retailers"""
        retailers = [n for n in self.nodes.values() if n.node_type == NodeType.RETAILER]
        
        while True:
            # Generate demand for each retailer
            for retailer in retailers:
                demand = self.demand_generator.generate_demand()
                
                if demand > 0:
                    order = Order(
                        id=f"CUST_{self.env.now}_{retailer.node_id}",
                        product_id="PROD1",
                        quantity=demand,
                        source_node=retailer.node_id,
                        destination_node="CUSTOMER",
                        order_time=self.env.now
                    )
                    
                    retailer.process_order(order)
                    self.metrics['total_orders'] += 1
                    
                    print(f"[{self.env.now:.2f}] Customer demand: {demand} units at {retailer.name}")
                    
            # Wait before next demand
            yield self.env.timeout(1.0)
            
    def calculate_bullwhip_effect(self) -> float:
        """Calculate bullwhip effect (demand amplification)"""
        # Compare demand variability at different echelons
        retailer_demands = []
        warehouse_demands = []
        manufacturer_demands = []
        
        # This is simplified - in reality would track actual order quantities over time
        return 1.0  # Placeholder
        
    def run_simulation(self, duration: float):
        """Run the supply chain simulation"""
        # Create network
        self.create_network()
        
        # Start demand generation
        self.env.process(self.customer_demand_process())
        
        # Run simulation
        self.env.run(until=duration)
        
        # Calculate final metrics
        for node in self.nodes.values():
            node_metrics = node.calculate_metrics()
            self.metrics['total_stockouts'] += node_metrics['stockouts']
            if node_metrics['orders_received'] > 0:
                self.metrics['fulfilled_orders'] += node_metrics['orders_fulfilled']
                
        # Calculate service level
        if self.metrics['total_orders'] > 0:
            self.metrics['service_level'] = (
                self.metrics['fulfilled_orders'] / self.metrics['total_orders']
            )
        else:
            self.metrics['service_level'] = 0
            
        return self.metrics

def main():
    """Run supply chain simulation"""
    print("="*60)
    print("Supply Chain Network Simulation")
    print("="*60)
    
    # Setup
    env = simpy.Environment()
    simulator = SupplyChainSimulator(env)
    
    # Run simulation
    duration = 100.0
    metrics = simulator.run_simulation(duration)
    
    # Compile results
    results = {
        "simulation_type": "supply_chain_network",
        "duration": duration,
        "network_size": len(simulator.nodes),
        "total_orders": metrics['total_orders'],
        "fulfilled_orders": metrics['fulfilled_orders'],
        "service_level": f"{metrics.get('service_level', 0) * 100:.2f}%",
        "total_stockouts": metrics['total_stockouts'],
        "node_performance": {}
    }
    
    # Add node-specific metrics
    for node_id, node in simulator.nodes.items():
        node_metrics = node.calculate_metrics()
        results["node_performance"][node.name] = {
            "orders_received": node_metrics['orders_received'],
            "orders_fulfilled": node_metrics['orders_fulfilled'],
            "stockouts": node_metrics['stockouts'],
            "service_level": f"{node_metrics['service_level'] * 100:.2f}%",
            "current_inventory": sum(node.inventory.values()),
            "backlog_size": len(node.backlog)
        }
        
    # Add recommendations
    results["optimization_recommendations"] = []
    
    if metrics['total_stockouts'] > metrics['total_orders'] * 0.1:
        results["optimization_recommendations"].append(
            "High stockout rate detected - consider increasing safety stock levels"
        )
        
    if metrics.get('service_level', 0) < 0.95:
        results["optimization_recommendations"].append(
            "Service level below 95% - review inventory policies and lead times"
        )
        
    print("\n" + "="*60)
    print("Simulation Results")
    print("="*60)
    print(json.dumps(results, indent=2))
    
    return results

if __name__ == "__main__":
    main()