# Odoo Community Resource

## Overview
Odoo Community is an integrated business management suite providing e-commerce, inventory, CRM, accounting, and more through a modular architecture. This resource enables Vrooli scenarios to build complete business automation workflows.

## Features
- **E-commerce**: Full online store with products, cart, checkout
- **Inventory Management**: Stock tracking, warehouses, transfers
- **CRM**: Customer relationship management with leads and opportunities
- **Accounting**: Invoicing, payments, financial reports
- **Manufacturing**: Bill of materials, work orders, production planning
- **HR**: Employee management, timesheets, leaves
- **Project Management**: Tasks, milestones, Gantt charts

## Architecture
- Python-based using Odoo framework
- PostgreSQL database backend
- XML-RPC and JSON-RPC APIs for automation
- Modular app ecosystem (install only what you need)
- Multi-tenant with database isolation

## Quick Start
```bash
# Install and start Odoo
vrooli resource odoo install
vrooli resource odoo start

# Check status
vrooli resource odoo status

# Access web interface
# Default: http://localhost:8069
# Admin email: admin@example.com
# Password: admin (change immediately)
```

## API Integration
```python
# Example: Create a customer via XML-RPC
import xmlrpc.client

url = "http://localhost:8069"
db = "odoo"
username = "admin@example.com"
password = "admin"

common = xmlrpc.client.ServerProxy(f'{url}/xmlrpc/2/common')
uid = common.authenticate(db, username, password, {})

models = xmlrpc.client.ServerProxy(f'{url}/xmlrpc/2/object')
customer_id = models.execute_kw(db, uid, password,
    'res.partner', 'create', [{
        'name': 'New Customer',
        'email': 'customer@example.com'
    }])
```

## Scenario Use Cases
- **E-commerce Automation**: Product sync, order processing, fulfillment
- **Inventory Sync**: Multi-channel stock management
- **Financial Automation**: Invoice generation, payment reconciliation
- **CRM Workflows**: Lead scoring, opportunity tracking
- **HR Automation**: Onboarding, timesheet approval

## Documentation
- [Installation Guide](docs/installation.md)
- [API Reference](docs/api.md)
- [Module Configuration](docs/modules.md)
- [Security Best Practices](docs/security.md)

## Resource Commands
```bash
vrooli resource odoo status      # Check health
vrooli resource odoo start       # Start service
vrooli resource odoo stop        # Stop service
vrooli resource odoo logs        # View logs
vrooli resource odoo inject      # Install modules/data
vrooli resource odoo test        # Run tests
```