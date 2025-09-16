# Mifos X Resource

Digital finance platform for microfinance institutions and financial inclusion, powered by Apache Fineract.

## Overview

Mifos X provides a complete core banking solution for:
- Microfinance institutions
- Credit unions
- Community banks
- Fintech startups
- Financial inclusion initiatives

The platform includes client management, loan processing, savings accounts, and comprehensive reporting.

## Quick Start

```bash
# Install and start Mifos
vrooli resource mifos manage install
vrooli resource mifos manage start --wait

# Check status
vrooli resource mifos status

# View credentials
vrooli resource mifos credentials
```

## Features

### Core Banking Functions
- **Client Management**: Create and manage customer records
- **Loan Products**: Define and manage loan products
- **Savings Accounts**: Savings products with interest calculation
- **Payment Processing**: Transaction management and reconciliation
- **Financial Reporting**: Comprehensive accounting reports

### API Capabilities
- RESTful API for all operations
- Multi-tenant architecture
- Batch API for bulk operations
- Webhook support for events

### Web Interface
- Modern Angular-based UI
- Mobile-responsive design
- Role-based access control
- Multi-language support

## Usage

### CLI Commands

```bash
# Lifecycle Management
vrooli resource mifos manage install      # Install Mifos
vrooli resource mifos manage start        # Start services
vrooli resource mifos manage stop         # Stop services
vrooli resource mifos manage restart      # Restart services
vrooli resource mifos manage uninstall    # Uninstall Mifos

# Information
vrooli resource mifos status              # Show status
vrooli resource mifos credentials         # Show credentials
vrooli resource mifos logs                # View logs

# Testing
vrooli resource mifos test smoke          # Quick health check
vrooli resource mifos test integration    # Full integration test
vrooli resource mifos test all            # Run all tests

# Content Management
vrooli resource mifos content list        # List content types
vrooli resource mifos content add client  # Add a client
vrooli resource mifos content get loans   # Get loan information
```

### Creating a Client

```bash
# Create a new client
vrooli resource mifos content add client '{
  "firstname": "John",
  "lastname": "Doe",
  "dateOfBirth": "15 March 1985"
}'

# Or use the specific command
vrooli resource mifos content create-client "John" "Doe"
```

### Managing Loans

```bash
# Create a loan product
vrooli resource mifos products create-loan "Small Business Loan" 12

# Disburse a loan
vrooli resource mifos content disburse-loan 12345 10000

# Check payment status
vrooli resource mifos content payment-status 12345
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MIFOS_PORT` | 8030 | API server port |
| `MIFOS_WEBAPP_PORT` | 8031 | Web UI port |
| `FINERACT_DB_HOST` | postgres | Database host |
| `FINERACT_DB_PORT` | 5433 | Database port |
| `FINERACT_DB_NAME` | fineract | Database name |
| `MIFOS_DEFAULT_USER` | mifos | Default username |
| `MIFOS_DEFAULT_PASSWORD` | password | Default password |
| `MIFOS_SEED_DEMO_DATA` | true | Seed demo data on install |

### Multi-Currency Support

Configure supported currencies:
```bash
export MIFOS_CURRENCIES="USD,EUR,GBP,INR"
export MIFOS_BASE_CURRENCY="USD"
```

## API Access

### Authentication

```bash
# Get authentication token
curl -X POST http://localhost:8030/fineract-provider/api/v1/authentication \
  -H "Content-Type: application/json" \
  -H "Fineract-Platform-TenantId: default" \
  -d '{"username": "mifos", "password": "password"}'
```

### Example API Calls

```bash
# List clients
curl -X GET http://localhost:8030/fineract-provider/api/v1/clients \
  -H "Authorization: Basic <token>" \
  -H "Fineract-Platform-TenantId: default"

# Create loan application
curl -X POST http://localhost:8030/fineract-provider/api/v1/loans \
  -H "Authorization: Basic <token>" \
  -H "Content-Type: application/json" \
  -d '{"clientId": 1, "productId": 1, "principal": 10000, ...}'
```

## Web UI Access

After starting Mifos, access the web interface at:
- URL: `http://localhost:8031`
- Username: `mifos`
- Password: `password`
- Tenant: `default`

## Integration

### With Other Vrooli Resources

- **PostgreSQL**: Database backend (required)
- **Redis**: Caching layer (optional)
- **ERPNext**: ERP integration for accounting
- **BTCPay**: Cryptocurrency payment processing
- **Vault**: Secure credential storage

### Webhook Configuration

Configure webhooks for real-time notifications:
```bash
# Configure webhook endpoint
vrooli resource mifos content execute configure-webhook \
  --url "https://your-endpoint.com/webhook" \
  --events "CLIENT_CREATED,LOAN_APPROVED,PAYMENT_RECEIVED"
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check what's using the port
   lsof -i :8030
   # Change port if needed
   export MIFOS_PORT=8035
   ```

2. **Database Connection Failed**
   ```bash
   # Ensure PostgreSQL is running
   vrooli resource postgres status
   vrooli resource postgres manage start
   ```

3. **Authentication Failed**
   ```bash
   # Reset to default credentials
   vrooli resource mifos credentials
   # Check tenant configuration
   export FINERACT_TENANT=default
   ```

## Performance Tuning

### Database Optimization
```bash
# Increase connection pool
export MIFOS_MAX_CONNECTIONS=200

# Tune PostgreSQL
vrooli resource postgres content execute optimize-for-mifos
```

### Memory Configuration
```bash
# Increase Java heap size
export JAVA_TOOL_OPTIONS="-Xmx4G -Xms2G"
```

## Security

- All credentials stored in environment variables
- Multi-tenant isolation supported
- SSL/TLS can be enabled
- Regular security updates via Docker images
- Audit logging for all transactions

## Resources

- [Apache Fineract Documentation](https://fineract.apache.org/docs/)
- [Mifos Initiative](https://mifos.org)
- [API Documentation](https://demo.mifos.io/api-docs/apiLive.htm)
- [Community Forum](https://mifosforge.jira.com)

## License

Apache Fineract is licensed under Apache License 2.0.
This Vrooli resource wrapper is part of the Vrooli ecosystem.