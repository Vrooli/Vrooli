# BTCPay Server Resource

Self-hosted, open-source cryptocurrency payment processor that allows businesses to accept Bitcoin and other cryptocurrencies directly, without fees or intermediaries.

## Features

- Accept Bitcoin and altcoin payments
- No transaction fees (only network fees)
- Direct, peer-to-peer payments
- Full control of private keys
- Privacy-focused design
- Lightning Network support
- Multi-store management
- Point of Sale applications

## Quick Start

```bash
# Install BTCPay Server
vrooli resource btcpay manage install

# Start the service
vrooli resource btcpay manage start --wait

# Check status
vrooli resource btcpay status

# View credentials
vrooli resource btcpay credentials
```

## Usage

### Store Management

```bash
# Create a new store
vrooli resource btcpay content create-store \
  --name "My Store" \
  --currency BTC

# List all stores
vrooli resource btcpay content list-stores
```

### Creating an Invoice

```bash
# Create a payment invoice
vrooli resource btcpay content create-invoice \
  --store-id <store_id> \
  --amount 10.00 \
  --currency USD \
  --description "Product purchase"
```

### Checking Payment Status

```bash
# Check if payment was received
vrooli resource btcpay content check-payment \
  --store-id <store_id> \
  --invoice-id <invoice_id>
```

### Webhook Configuration

```bash
# Configure webhook for payment notifications
vrooli resource btcpay content configure-webhook \
  --store-id <store_id> \
  --url https://your-site.com/webhook \
  --events "InvoiceCreated,InvoiceSettled"
```

### Managing Stores

```bash
# List payment stores
vrooli resource btcpay content list

# Add a new store
vrooli resource btcpay content add --name "My Store"
```

## Configuration

BTCPay Server uses environment variables for configuration:

- `BTCPAY_PORT`: API port (default: 23000)
- `BTCPAY_NETWORK`: Bitcoin network (mainnet/testnet/regtest)
- `BTCPAY_POSTGRES_USER`: Database username
- `BTCPAY_POSTGRES_PASSWORD`: Database password
- `BTCPAY_POSTGRES_DB`: Database name

## Testing

```bash
# Run all tests
vrooli resource btcpay test all

# Quick health check
vrooli resource btcpay test smoke

# Integration tests
vrooli resource btcpay test integration
```

## API Integration

BTCPay provides a RESTful API for payment processing:

```javascript
// Create invoice
const invoice = await fetch('http://localhost:23000/api/v1/stores/store1/invoices', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <api-key>'
  },
  body: JSON.stringify({
    amount: 10.00,
    currency: 'USD'
  })
});

// Check payment status
const status = await fetch(`http://localhost:23000/api/v1/stores/store1/invoices/${invoiceId}`);
```

## Troubleshooting

### Service Won't Start
- Check Docker is running: `docker info`
- Verify port 23000 is available: `ss -tuln | grep 23000`
- Check logs: `vrooli resource btcpay logs`

### Database Connection Issues
- Verify PostgreSQL is running: `docker ps | grep btcpay-postgres`
- Check database credentials in environment
- Test connection: `vrooli resource btcpay test smoke`

### Payment Not Confirming
- Check blockchain sync status
- Verify network configuration (mainnet/testnet)
- Review webhook configuration

## Security Considerations

- Always use HTTPS in production
- Secure database credentials
- Regular security updates
- Backup private keys and seed phrases
- Use API keys for authentication
- Implement rate limiting

## Support

For issues or questions:
- BTCPay Docs: https://docs.btcpayserver.org
- GitHub Issues: https://github.com/btcpayserver/btcpayserver
- Community Chat: https://chat.btcpayserver.org