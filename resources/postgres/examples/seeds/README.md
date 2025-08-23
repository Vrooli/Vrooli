# PostgreSQL Seed Data for Client Templates

This directory contains pre-built seed data for different client project types, designed to accelerate the setup of client-specific automation workflows.

## 📁 Directory Structure

```
seeds/
├── real-estate/          # Real estate and property management
├── ecommerce/            # Online store and retail automation  
├── saas/                 # Multi-tenant SaaS applications
└── README.md             # This file
```

## 🏠 Real Estate Seeds

**Purpose**: Lead generation, property management, CRM automation

**Files**:
- `001_properties.sql` - Property listings, features, MLS data
- `002_leads_contacts.sql` - Lead management, agents, activities

**Schema Includes**:
- Properties with MLS integration
- Lead capture and qualification system
- Agent assignment and activity tracking
- Property features and search optimization

**Use Cases**:
- Automated lead follow-up workflows
- Property listing syndication
- Market analysis and reporting
- Customer relationship management

## 🛒 Ecommerce Seeds

**Purpose**: Online store automation, inventory management, order processing

**Files**:
- `001_products_inventory.sql` - Product catalog, categories, inventory
- `002_customers_orders.sql` - Customer data, orders, abandoned carts

**Schema Includes**:
- Multi-category product catalog
- Inventory tracking with reorder points
- Customer profiles and order history
- Abandoned cart recovery system

**Use Cases**:
- Inventory alerts and reordering
- Customer segmentation and marketing
- Order fulfillment automation
- Abandoned cart recovery emails

## 🏢 SaaS Seeds

**Purpose**: Multi-tenant applications, subscription management, usage tracking

**Files**:
- `001_tenants_users.sql` - Multi-tenant architecture, user management
- `002_features_billing.sql` - Feature flags, billing, support tickets

**Schema Includes**:
- Row-level security for tenant isolation
- Subscription and billing management
- Feature flag system for A/B testing
- Usage metrics and analytics
- Support ticket tracking

**Use Cases**:
- Automated billing and invoicing
- Feature rollout and A/B testing  
- Usage-based pricing alerts
- Customer support automation

## 🚀 Usage Instructions

### Quick Start with Template
```bash
# Create instance with client template
./manage.sh --action create --instance my-client --template real-estate

# Apply template-specific seed data
./manage.sh --action seed --instance my-client --seed-path ./examples/seeds/real-estate/
```

### Manual Seeding
```bash
# Seed specific files in order
./manage.sh --action seed --instance my-client --seed-path ./examples/seeds/real-estate/001_properties.sql
./manage.sh --action seed --instance my-client --seed-path ./examples/seeds/real-estate/002_leads_contacts.sql
```

### Verify Seeded Data
```bash
# Check tables and data
./manage.sh --action connect --instance my-client
# Then run SQL: SELECT COUNT(*) FROM properties;
```

## 🎯 Integration with Automation

### n8n Workflows
The seeded data is designed to work with n8n automation workflows:

- **Real Estate**: Lead capture forms → CRM updates → Follow-up emails
- **Ecommerce**: Order webhooks → Inventory updates → Customer notifications  
- **SaaS**: Usage metrics → Billing triggers → Support ticket routing

### Node-RED Dashboards
Create real-time monitoring dashboards:

- **Real Estate**: Property views, lead conversion rates, agent performance
- **Ecommerce**: Sales metrics, inventory levels, customer activity
- **SaaS**: Tenant usage, subscription health, support queue

### Example n8n Connection
```javascript
// n8n PostgreSQL node configuration
{
  "host": "postgres-my-client",
  "port": 5432,
  "database": "vrooli_client", 
  "user": "vrooli",
  "password": "[from instance config]"
}
```

## 🔧 Customization

### Adding Your Own Seed Data
1. Create new directory: `seeds/your-industry/`
2. Add SQL files with sequential numbering
3. Include table creation and sample data
4. Update this README with your use case

### Template Integration
The seed data automatically works with client templates:
- `real-estate` template → `seeds/real-estate/`
- `ecommerce` template → `seeds/ecommerce/`
- `saas` template → `seeds/saas/`

### Data Privacy
- All seed data uses fictional names and addresses
- No real customer or business information included
- Safe for development and testing environments

## 📊 Sample Data Overview

| Template | Tables | Records | Key Features |
|----------|---------|---------|--------------|
| Real Estate | 5 | ~50 | Properties, leads, agents, activities |
| Ecommerce | 8 | ~75 | Products, orders, customers, inventory |
| SaaS | 10 | ~100 | Multi-tenancy, billing, feature flags |

## 🔗 Related Documentation

- [Client Template Guide](../README.md#client-templates)
- [Multi-tenant Examples](../multi-tenant.sh)
- [n8n Integration](../n8n-integration.md)
- [Node-RED Integration](../node-red-integration.md)