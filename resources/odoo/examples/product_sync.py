#!/usr/bin/env python3
"""
Example: Sync products to Odoo from external source
"""

import xmlrpc.client
import json

# Configuration
URL = "http://localhost:8069"
DB = "odoo"
USERNAME = "admin@example.com"
PASSWORD = "admin"

class OdooProductSync:
    def __init__(self):
        self.common = xmlrpc.client.ServerProxy(f'{URL}/xmlrpc/2/common')
        self.uid = self.common.authenticate(DB, USERNAME, PASSWORD, {})
        self.models = xmlrpc.client.ServerProxy(f'{URL}/xmlrpc/2/object')
        
        if not self.uid:
            raise Exception("Authentication failed")
    
    def find_product(self, sku):
        """Find product by SKU"""
        product_ids = self.models.execute_kw(DB, self.uid, PASSWORD,
            'product.product', 'search', [[['default_code', '=', sku]]])
        return product_ids[0] if product_ids else None
    
    def create_product(self, product_data):
        """Create a new product"""
        return self.models.execute_kw(DB, self.uid, PASSWORD,
            'product.product', 'create', [product_data])
    
    def update_product(self, product_id, product_data):
        """Update existing product"""
        return self.models.execute_kw(DB, self.uid, PASSWORD,
            'product.product', 'write', [[product_id], product_data])
    
    def sync_product(self, sku, name, price, qty=0, description=""):
        """Sync a single product"""
        product_data = {
            'name': name,
            'default_code': sku,
            'list_price': price,
            'description': description,
            'type': 'product',
            'sale_ok': True,
            'purchase_ok': True,
        }
        
        # Check if product exists
        product_id = self.find_product(sku)
        
        if product_id:
            # Update existing
            self.update_product(product_id, product_data)
            print(f"Updated product: {sku}")
        else:
            # Create new
            product_id = self.create_product(product_data)
            print(f"Created product: {sku}")
        
        # Update stock quantity if provided
        if qty > 0 and product_id:
            self.update_stock(product_id, qty)
        
        return product_id
    
    def update_stock(self, product_id, quantity):
        """Update product stock quantity"""
        # This would typically involve creating stock.quant records
        # Simplified for example
        print(f"Stock updated for product {product_id}: {quantity} units")

if __name__ == "__main__":
    # Example products to sync
    products = [
        {
            'sku': 'PROD-001',
            'name': 'Laptop Pro 15"',
            'price': 1299.99,
            'qty': 50,
            'description': 'High-performance laptop'
        },
        {
            'sku': 'PROD-002',
            'name': 'Wireless Mouse',
            'price': 29.99,
            'qty': 200,
            'description': 'Ergonomic wireless mouse'
        },
    ]
    
    try:
        sync = OdooProductSync()
        
        for product in products:
            sync.sync_product(**product)
        
        print(f"\nSuccessfully synced {len(products)} products")
        
    except Exception as e:
        print(f"Error: {e}")
        exit(1)