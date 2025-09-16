# BTCPay Server Resource

Self-hosted, open-source cryptocurrency payment processor that allows businesses to accept Bitcoin and other cryptocurrencies directly, without fees or intermediaries.

## Features

- Accept Bitcoin and altcoin payments
- Multi-currency support (BTC, LTC, ETH)
- No transaction fees (only network fees)
- Direct, peer-to-peer payments
- Full control of private keys
- Privacy-focused design
- Lightning Network support
- Multi-store management
- Complete Point of Sale system with inventory management
- HTML-based POS terminal for retail
- Crowdfunding campaign management
- Embeddable payment buttons

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

## Lightning Network

BTCPay Server includes full Lightning Network support for instant, low-fee Bitcoin payments.

### Setting Up Lightning

```bash
# Initialize Lightning Network support
vrooli resource btcpay lightning setup

# Check Lightning status
vrooli resource btcpay lightning status

# View Lightning wallet balance
vrooli resource btcpay lightning balance
```

### Lightning Invoices

```bash
# Create a Lightning invoice (amount in satoshis)
vrooli resource btcpay lightning create-invoice 10000 "Coffee payment"

# Check Lightning invoice status
vrooli resource btcpay lightning check-invoice <invoice_hash>
```

### Channel Management

```bash
# List Lightning channels
vrooli resource btcpay lightning channels

# Open a new channel (amount in satoshis)
vrooli resource btcpay lightning open-channel <peer_pubkey> 100000
```

### Popular Lightning Nodes

When opening channels, you can connect to these well-known nodes:
- **ACINQ**: `03864ef025fde8fb587d989186ce6a4a186895ee44a926bfc370e2c366597a3f8f`
- **Bitrefill**: `030c3f19d742ca294a55c00376b3b355c3c90d61c6b6b39554dbc7ac19b141c14f`

## Multi-Currency Support

### Configure Multiple Cryptocurrencies

```bash
# Configure multi-currency support
vrooli resource btcpay multicurrency configure btc,ltc,eth

# Enable a specific currency
vrooli resource btcpay multicurrency enable LTC

# Disable a currency
vrooli resource btcpay multicurrency disable ETH

# List configured currencies
vrooli resource btcpay multicurrency list

# Apply configuration (restart with new settings)
vrooli resource btcpay multicurrency apply
```

## Point of Sale System

### Configure POS

```bash
# Set up POS system
vrooli resource btcpay pos configure "My Store" BTC

# Check POS status
vrooli resource btcpay pos status
```

### Manage Inventory

```bash
# Add items to inventory
vrooli resource btcpay pos add-item "Coffee" 5.99 USD "Fresh roasted coffee"
vrooli resource btcpay pos add-item "Sandwich" 8.99 USD "Turkey club sandwich"

# List all items
vrooli resource btcpay pos list-items

# Update item price
vrooli resource btcpay pos update-item <item_id> price 6.99

# Remove item
vrooli resource btcpay pos remove-item <item_id>

# Import items from CSV
vrooli resource btcpay pos import items.csv
```

### Generate POS Terminal

```bash
# Generate HTML-based POS terminal
vrooli resource btcpay pos generate

# Terminal will be available at:
# /home/user/Vrooli/data/resources/btcpay/config/pos-terminal.html
```

## Configuration

BTCPay Server uses environment variables for configuration:

- `BTCPAY_PORT`: API port (default: 23000)
- `BTCPAY_NETWORK`: Bitcoin network (mainnet/testnet/regtest)
- `BTCPAY_POSTGRES_USER`: Database username
- `BTCPAY_POSTGRES_PASSWORD`: Database password
- `BTCPAY_POSTGRES_DB`: Database name

## Crowdfunding Campaigns

### Creating a Campaign

```bash
# Create a crowdfunding campaign
vrooli resource btcpay crowdfunding create \
  "Bitcoin Education Fund" 10 BTC \
  "Support open-source Bitcoin education" \
  "2025-12-31"

# List all campaigns
vrooli resource btcpay crowdfunding list

# Check campaign status
vrooli resource btcpay crowdfunding status <campaign-id>

# Record contribution
vrooli resource btcpay crowdfunding contribute \
  <campaign-id> 0.5 "John Doe" "Great cause!"

# Generate embeddable widget
vrooli resource btcpay crowdfunding widget <campaign-id>

# Export campaign data
vrooli resource btcpay crowdfunding export <campaign-id> csv
```

## Payment Buttons

### Creating Payment Buttons

```bash
# Create a payment button
vrooli resource btcpay paybutton create \
  0.001 BTC "Coffee Donation" "Buy me a coffee"

# List all buttons
vrooli resource btcpay paybutton list

# Get embed code
vrooli resource btcpay paybutton get-code <button-id> iframe

# View button statistics
vrooli resource btcpay paybutton stats <button-id>

# Generate custom styles
vrooli resource btcpay paybutton generate-styles rounded

# Bulk create from CSV
vrooli resource btcpay paybutton bulk-create buttons.csv
```

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