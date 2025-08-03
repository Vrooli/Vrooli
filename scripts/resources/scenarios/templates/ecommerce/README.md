# E-commerce Platform Template

A complete e-commerce platform with inventory management, order processing, payment integration, and analytics.

## Features

- 📦 **Product Management**: Full catalog with categories, variants, and inventory tracking
- 🛒 **Shopping Cart**: Session-based cart with abandoned cart recovery
- 💳 **Payment Processing**: Stripe and PayPal integration
- 📧 **Notifications**: Email notifications for orders, shipping, and marketing
- 📊 **Analytics**: Real-time sales metrics and customer insights
- 🤖 **AI Support**: Customer service chatbot and product recommendations
- 🔄 **Automation**: Order processing, inventory alerts, and reporting workflows

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│                    (Your Application)                        │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Resource Layer                          │
├──────────────┬──────────────┬──────────────┬───────────────┤
│  PostgreSQL  │     MinIO    │   QuestDB    │    Qdrant     │
│   (Orders)   │   (Images)   │  (Metrics)   │   (Search)    │
├──────────────┼──────────────┼──────────────┼───────────────┤
│     n8n      │  Node-RED    │   Windmill   │    Vault      │
│ (Workflows)  │  (Events)    │   (Tasks)    │  (Secrets)    │
└──────────────┴──────────────┴──────────────┴───────────────┘
```

## Quick Start

### 1. Set Environment Variables

Create `.env` file:
```bash
# Required
DB_USER=ecommerce_user
DB_PASSWORD=secure_password_here
STRIPE_PUBLISHABLE_KEY=pk_test_xxx
STRIPE_SECRET_KEY=sk_test_xxx
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=notifications@example.com
SMTP_PASSWORD=app_specific_password

# Optional
STRIPE_WEBHOOK_SECRET=whsec_xxx
PAYPAL_CLIENT_ID=xxx
PAYPAL_CLIENT_SECRET=xxx
FEDEX_ACCOUNT=xxx
GA_TRACKING_ID=G-xxx
```

### 2. Deploy Template

```bash
# Copy template
cp scenario.json ~/.vrooli/scenarios.json

# Source environment variables
source .env

# Deploy all resources
../../index.sh --action inject-all
```

### 3. Verify Deployment

```bash
# Check resource status
../../index.sh --action status

# Test database connection
psql -h localhost -p 5432 -U ecommerce_user -d ecommerce

# Access services
# n8n: http://localhost:5678
# Node-RED: http://localhost:1880
# MinIO: http://localhost:9001
```

## Database Schema

### Core Tables

**users**
- User accounts and authentication
- Profile information
- Preferences and settings

**products**
- Product catalog
- Categories and tags
- Pricing and variants

**orders**
- Order management
- Status tracking
- Payment records

**inventory**
- Stock levels
- Warehouse locations
- Reorder points

**carts**
- Shopping cart sessions
- Saved items
- Abandoned cart tracking

### Sample Data

The template includes sample data for testing:
- 100 sample products across 10 categories
- 50 test customer accounts
- 200 historical orders
- Inventory levels for all products

## Workflows

### n8n Workflows

**Order Processing**
- Validates payment
- Updates inventory
- Sends confirmation email
- Creates shipping label

**Inventory Alerts**
- Monitors stock levels
- Sends reorder notifications
- Updates supplier systems

**Customer Notifications**
- Order confirmations
- Shipping updates
- Review requests
- Marketing campaigns

### Node-RED Flows

**Payment Processing**
- Real-time payment validation
- Fraud detection
- Payment method routing

**Inventory Sync**
- Multi-warehouse synchronization
- Real-time stock updates
- Back-order management

### Windmill Scripts

**Daily Sales Report**
- Aggregates daily metrics
- Generates PDF reports
- Emails to stakeholders

**Inventory Reorder**
- Calculates reorder points
- Creates purchase orders
- Manages supplier communication

## API Integration

### Payment Gateways

**Stripe Integration**
```javascript
// Webhook endpoint for Stripe events
POST /webhooks/stripe
Headers: stripe-signature: xxx

// Process payment
POST /api/payments/stripe
{
  "amount": 9999,
  "currency": "usd",
  "payment_method": "pm_xxx"
}
```

**PayPal Integration**
```javascript
// Webhook endpoint for PayPal
POST /webhooks/paypal

// Create order
POST /api/payments/paypal/orders
{
  "amount": "99.99",
  "currency": "USD"
}
```

### Shipping Providers

**FedEx Integration**
- Rate calculation
- Label generation
- Tracking updates

## Analytics

### QuestDB Metrics

Real-time metrics available:
- Order volume and revenue
- Product popularity
- Cart abandonment rates
- Customer lifetime value
- Inventory turnover

### Sample Queries

```sql
-- Daily revenue
SELECT 
  timestamp,
  sum(amount) as revenue,
  count(*) as orders
FROM order_metrics
WHERE timestamp >= dateadd('d', -30, now())
SAMPLE BY 1d
ALIGN TO CALENDAR;

-- Top products
SELECT 
  product_id,
  count(*) as views
FROM product_views
WHERE timestamp >= dateadd('d', -7, now())
GROUP BY product_id
ORDER BY views DESC
LIMIT 10;
```

## Customization

### Adding Payment Methods

1. Add credentials to Vault:
```json
{
  "path": "secret/data/payment/newmethod",
  "data": {
    "api_key": "${NEW_METHOD_KEY}"
  }
}
```

2. Create workflow in n8n:
- Copy existing payment workflow
- Update API endpoints
- Configure webhook handling

### Extending Product Schema

1. Modify `assets/schema.sql`:
```sql
ALTER TABLE products ADD COLUMN custom_field VARCHAR(255);
```

2. Update seed data in `assets/seed-data.sql`

3. Re-run injection:
```bash
../../index.sh --action inject --resource postgres
```

### Custom Workflows

Add new workflows to the template:

1. Create workflow file:
```bash
assets/workflows/n8n/my-workflow.json
```

2. Update scenario.json:
```json
{
  "name": "My Custom Workflow",
  "file": "assets/workflows/n8n/my-workflow.json",
  "active": true
}
```

## Monitoring

### Health Checks

```bash
# Database health
psql -c "SELECT 1" -h localhost -p 5432 -U ecommerce_user -d ecommerce

# Workflow status
curl http://localhost:5678/healthz

# Storage status
curl http://localhost:9000/minio/health/live
```

### Performance Metrics

Monitor key metrics:
- Response times
- Order processing time
- Inventory sync latency
- Payment success rate

## Troubleshooting

### Common Issues

**Database Connection Failed**
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Verify credentials
psql -h localhost -p 5432 -U ecommerce_user
```

**Workflows Not Triggering**
```bash
# Check n8n webhook URL
curl http://localhost:5678/webhook/xxx

# Verify workflow is active
# Access n8n UI at http://localhost:5678
```

**Storage Issues**
```bash
# Check MinIO is accessible
curl http://localhost:9000/minio/health/live

# Verify buckets exist
mc ls local/
```

### Debug Mode

Enable debug logging:
```bash
DEBUG=1 ../../index.sh --action inject-all
```

### Reset and Retry

```bash
# Rollback all injections
../../index.sh --action rollback

# Clean data directories
rm -rf ~/.vrooli/data/ecommerce

# Retry deployment
../../index.sh --action inject-all
```

## Security Considerations

1. **Change default passwords** before production use
2. **Enable SSL/TLS** for all external connections
3. **Configure firewall rules** to restrict access
4. **Rotate API keys** regularly
5. **Enable audit logging** for compliance
6. **Implement rate limiting** on public endpoints
7. **Use Vault policies** to restrict secret access

## Production Deployment

### Scaling Considerations

- Use managed PostgreSQL for production
- Configure MinIO with multiple nodes
- Deploy workflows across multiple workers
- Implement caching layer (Redis)
- Use CDN for static assets

### Backup Strategy

```bash
# Database backup
pg_dump -h localhost -U ecommerce_user ecommerce > backup.sql

# MinIO backup
mc mirror local/product-images /backup/images

# Workflow export
n8n export:workflow --all > workflows-backup.json
```

## Support

For issues or questions:
1. Check the [main documentation](../../README.md)
2. Review resource-specific docs
3. Check workflow logs
4. Enable debug mode for detailed output