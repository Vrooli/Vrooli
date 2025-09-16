# ERPNext Resource

Complete open-source ERP with accounting, inventory, HR, CRM, and project management capabilities.

**Status**: ✅ v2.0 Contract Compliant | 100% PRD Complete | All Tests Passing

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
# Install dependencies
vrooli resource erpnext manage install

# Start ERPNext (auto-initializes site on first run)
vrooli resource erpnext manage start --wait

# Check status (includes site and API status)
vrooli resource erpnext status

# Run tests
vrooli resource erpnext test smoke
vrooli resource erpnext test all

# Stop ERPNext
vrooli resource erpnext manage stop
```

### Site Initialization

ERPNext automatically creates and configures a site named `vrooli.local` on first startup. The site includes:
- Database setup with MariaDB
- ERPNext app installation
- Default admin user (admin/admin)
- Basic configuration

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

## Workflow Management

ERPNext includes a powerful workflow engine for business process automation:

```bash
# List available workflows
vrooli resource erpnext workflow list

# Get workflow details
vrooli resource erpnext workflow get "Purchase Order"

# Create new workflow from JSON
vrooli resource erpnext workflow create workflow.json

# Execute workflow transition
vrooli resource erpnext workflow transition "Purchase Order" "PO-001" "Approve"
```

## Reporting and Analytics

Built-in reporting module with custom report builder:

```bash
# List available reports
vrooli resource erpnext report list

# Get report details
vrooli resource erpnext report get "Sales Register"

# Execute report with filters
vrooli resource erpnext report execute "Sales Register" '{"from_date":"2025-01-01","to_date":"2025-12-31"}'

# Export report data
vrooli resource erpnext report export "Sales Register" csv

# Create custom report from JSON
vrooli resource erpnext report create report.json
```

## E-commerce Module

Manage online store and shopping cart functionality:

```bash
# Product management
vrooli resource erpnext ecommerce list-products       # List online products
vrooli resource erpnext ecommerce add-product "ITEM-001" "Product Name" 99.99 "Description"

# Shopping cart operations
vrooli resource erpnext ecommerce get-cart           # View cart contents
vrooli resource erpnext ecommerce add-to-cart "ITEM-001" 2  # Add 2 units

# Store configuration
vrooli resource erpnext ecommerce configure "My Store" "USD"
```

## Manufacturing Module

Production planning and control features:

```bash
# Bill of Materials (BOM) management
vrooli resource erpnext manufacturing list-boms       # List all BOMs
vrooli resource erpnext manufacturing create-bom "PROD-001" 1
vrooli resource erpnext manufacturing add-bom-item "BOM-001" "COMP-001" 5

# Work order management
vrooli resource erpnext manufacturing list-work-orders  # List work orders
vrooli resource erpnext manufacturing create-work-order "PROD-001" 10 "BOM-001"

# Production planning
vrooli resource erpnext manufacturing production-plan "2025-01-01" "2025-01-31"
vrooli resource erpnext manufacturing stock-entries "WO-001"
```

## Multi-tenant Support

Manage multiple companies and organizations:

```bash
# Company management
vrooli resource erpnext multi-tenant list-companies    # List all companies
vrooli resource erpnext multi-tenant create-company "Acme Corp" "ACME" "USD" "United States"
vrooli resource erpnext multi-tenant switch-company "Acme Corp"

# User assignment
vrooli resource erpnext multi-tenant assign-user "user@example.com" "Acme Corp" "Manager"

# Company-specific data
vrooli resource erpnext multi-tenant get-data "Acme Corp" "Customer"
vrooli resource erpnext multi-tenant configure "Acme Corp" "default_currency" "EUR"
```

## Mobile UI Configuration

Enable and configure mobile-responsive interface:

```bash
# Enable responsive UI
vrooli resource erpnext mobile-ui enable

# Configure mobile theme
vrooli resource erpnext mobile-ui configure-theme "Mobile Theme" "#5e64ff" "#ffffff"

# Set up Progressive Web App
vrooli resource erpnext mobile-ui configure-pwa "ERPNext Mobile" "ERP"

# Configure mobile menu
vrooli resource erpnext mobile-ui configure-menu

# Optimize for touch devices
vrooli resource erpnext mobile-ui optimize-touch

# Create mobile dashboard
vrooli resource erpnext mobile-ui create-dashboard "Executive Dashboard"
```

## API Access

### Multi-Tenant Architecture
ERPNext uses a multi-tenant architecture where the `Host` header determines which site to serve. All API requests must include:
```bash
-H "Host: vrooli.local"
```

### API Endpoints
- **Ping**: `curl -H "Host: vrooli.local" http://localhost:8020/api/method/frappe.ping`
- **Login**: `curl -X POST -H "Host: vrooli.local" -H "Content-Type: application/json" -d '{"usr":"Administrator","pwd":"admin"}' http://localhost:8020/api/method/login`
- **Resources**: `curl -H "Host: vrooli.local" -H "Cookie: sid=SESSION_ID" http://localhost:8020/api/resource/DocType`

### Using the API Helper
```bash
# Source the API helper
source /path/to/erpnext/lib/api.sh

# Login and get session
SESSION_ID=$(erpnext::api::login "Administrator" "admin")

# Make API calls
erpnext::api::get "/api/resource/User" "$SESSION_ID"
```

## Documentation

- [Product Requirements Document](PRD.md) - Complete feature specifications and progress
- [Configuration Guide](docs/configuration.md)
- [API Reference](docs/api.md)
- [Custom App Development](docs/custom-apps.md)
- [Injection Guide](docs/injection.md)

## Resource Requirements

- Memory: 2GB minimum, 4GB recommended
- Storage: 5GB minimum
- CPU: 2 cores minimum
- Dependencies: PostgreSQL, Redis (managed by Vrooli)

## v2.0 Contract Compliance

This resource fully implements the Vrooli v2.0 Universal Contract:
- ✅ Complete CLI command structure (help, info, manage, test, content, status, logs)
- ✅ Core library implementation (lib/core.sh)
- ✅ Test structure with phases (smoke, unit, integration)
- ✅ Configuration schema and runtime.json
- ✅ Health checks with <1s response time
- ✅ Proper exit codes and timeout handling

## Known Issues

### Multi-Tenant Routing
ERPNext's multi-tenant architecture requires special handling:

**For Browser Access:**
Add to `/etc/hosts`:
```
127.0.0.1 vrooli.local
```
Then access: http://vrooli.local:8020

**For API Access:**
Always include the Host header:
```bash
curl -H "Host: vrooli.local" http://localhost:8020/api/method/frappe.ping
```

This is a fundamental design aspect of Frappe/ERPNext and ensures proper site isolation in multi-tenant deployments.

## Troubleshooting

### Browser Access Setup
1. Add `127.0.0.1 vrooli.local` to `/etc/hosts`
2. Access http://vrooli.local:8020 in browser
3. Login with Administrator/admin

### API Testing
```bash
# Test API endpoint (with required Host header)
curl -H "Host: vrooli.local" http://localhost:8020/api/method/frappe.ping

# Authenticate
curl -X POST -H "Host: vrooli.local" \
  -H "Content-Type: application/json" \
  -d '{"usr":"Administrator","pwd":"admin"}' \
  http://localhost:8020/api/method/login
```

### Container Management
- Uses Docker Compose with MariaDB, Redis, and ERPNext containers
- Port 8020 is exposed for web access
- Logs available via `vrooli resource erpnext logs`