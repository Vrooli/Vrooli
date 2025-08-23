# ERPNext Resource

Complete open-source ERP with accounting, inventory, HR, CRM, and project management capabilities.

## Overview

ERPNext is a comprehensive business automation suite that provides:
- Accounting & Finance
- Inventory Management
- Human Resources
- Customer Relationship Management (CRM)
- Project Management
- Manufacturing
- E-commerce
- And more...

## Features

- **Multi-tenant Architecture**: Support multiple companies/organizations
- **REST APIs**: Comprehensive API for all modules
- **Workflow Engine**: Custom business process automation
- **Reporting**: Built-in analytics and custom report builder
- **Mobile Friendly**: Responsive design for all devices

## Quick Start

```bash
# Start ERPNext
vrooli resource erpnext start

# Check status
vrooli resource erpnext status

# Stop ERPNext
vrooli resource erpnext stop
```

## Injection Support

ERPNext supports injecting custom apps, DocTypes, and scripts:

```bash
# Inject a custom app
vrooli resource erpnext inject --type app myapp.tar.gz

# Inject a custom DocType
vrooli resource erpnext inject --type doctype customer_extension.json

# Inject a server script
vrooli resource erpnext inject --type script custom_workflow.py
```

## API Access

ERPNext provides REST APIs at:
- Main API: http://localhost:8020/api
- Auth: http://localhost:8020/api/method/login

## Documentation

- [Configuration Guide](docs/configuration.md)
- [API Reference](docs/api.md)
- [Custom App Development](docs/custom-apps.md)
- [Injection Guide](docs/injection.md)

## Resource Requirements

- Memory: 2GB minimum, 4GB recommended
- Storage: 5GB minimum
- CPU: 2 cores minimum