#!/usr/bin/env python3
"""
Example: Create a customer in Odoo via XML-RPC API
"""

import xmlrpc.client
import sys

# Configuration
URL = "http://localhost:8069"
DB = "odoo"
USERNAME = "admin@example.com"
PASSWORD = "admin"

def create_customer(name, email, phone=None):
    """Create a new customer in Odoo"""
    try:
        # Authenticate
        common = xmlrpc.client.ServerProxy(f'{URL}/xmlrpc/2/common')
        uid = common.authenticate(DB, USERNAME, PASSWORD, {})
        
        if not uid:
            print("Authentication failed")
            return None
        
        # Create customer
        models = xmlrpc.client.ServerProxy(f'{URL}/xmlrpc/2/object')
        customer_data = {
            'name': name,
            'email': email,
            'is_company': False,
            'customer_rank': 1,
        }
        
        if phone:
            customer_data['phone'] = phone
        
        customer_id = models.execute_kw(DB, uid, PASSWORD,
            'res.partner', 'create', [customer_data])
        
        print(f"Customer created successfully with ID: {customer_id}")
        return customer_id
        
    except Exception as e:
        print(f"Error creating customer: {e}")
        return None

if __name__ == "__main__":
    # Example usage
    customer_id = create_customer(
        name="John Doe",
        email="john.doe@example.com",
        phone="+1-555-0123"
    )
    
    if customer_id:
        print(f"Success! Customer ID: {customer_id}")
    else:
        print("Failed to create customer")
        sys.exit(1)