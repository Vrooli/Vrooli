# LNbits - Lightning Network Bitcoin Payments

## Overview
LNbits is a free and open-source Lightning Network wallet and accounts system that enables instant Bitcoin micropayments with near-zero fees. It provides a powerful API and extensible framework for building Lightning-powered applications.

## Features
- **Lightning Wallets**: Create and manage multiple isolated wallets
- **Instant Payments**: Process Lightning payments in milliseconds
- **LNURL Support**: Seamless payment experiences with LNURL
- **Extension System**: Extend functionality with plugins
- **Multi-Backend**: Support for LND, Core Lightning, and more
- **API-First**: RESTful API for programmatic access

## Quick Start

### Installation
```bash
# Install LNbits resource
vrooli resource lnbits manage install

# Start the service
vrooli resource lnbits manage start --wait
```

### Basic Usage
```bash
# Check service status
vrooli resource lnbits status

# Create a new wallet
vrooli resource lnbits content add --type wallet --name "My Wallet"

# List wallets
vrooli resource lnbits content list --type wallet

# Generate invoice
vrooli resource lnbits content execute --action create-invoice --amount 1000 --memo "Test payment"

# View logs
vrooli resource lnbits logs
```

## Configuration

### Environment Variables
```bash
# Required
LNBITS_PORT=5001              # API port
LNBITS_POSTGRES_URL=          # PostgreSQL connection
LNBITS_DATA_FOLDER=           # Data directory

# Optional
LNBITS_BACKEND_WALLET=LndWallet  # Lightning backend
LND_GRPC_ENDPOINT=               # LND connection
LND_GRPC_MACAROON=               # LND authentication
```

### Runtime Configuration
See `config/runtime.json` for startup dependencies and timing.

## API Endpoints
- `GET /api/v1/health` - Health check
- `POST /api/v1/wallet` - Create wallet
- `GET /api/v1/wallet/{id}` - Get wallet info
- `POST /api/v1/payments` - Create invoice
- `GET /api/v1/payments/{hash}` - Check payment

## Integration Examples

### With n8n Workflow
```javascript
// Trigger workflow on payment received
const webhook = {
  url: 'http://n8n:5678/webhook/payment-received',
  event: 'invoice.paid',
  wallet_id: 'wallet-uuid'
};
```

### With Python Script
```python
import requests

# Create invoice
response = requests.post(
    'http://localhost:5001/api/v1/payments',
    headers={'X-Api-Key': 'your-api-key'},
    json={'amount': 1000, 'memo': 'Payment for service'}
)
invoice = response.json()
print(f"Pay this invoice: {invoice['payment_request']}")
```

## Testing
```bash
# Run all tests
vrooli resource lnbits test all

# Quick smoke test
vrooli resource lnbits test smoke

# Integration tests
vrooli resource lnbits test integration
```

## Troubleshooting

### Service Won't Start
- Check PostgreSQL is running: `vrooli resource postgres status`
- Verify port 5001 is available: `lsof -i :5001`
- Check logs: `vrooli resource lnbits logs`

### Payment Failures
- Ensure Lightning backend is connected
- Check wallet has sufficient balance
- Verify invoice hasn't expired

### Database Issues
- Check PostgreSQL connection string
- Ensure database exists and migrations ran
- Review PostgreSQL logs

## Security Notes
- Never expose API keys in code
- Use separate wallets for different applications
- Enable rate limiting in production
- Keep Lightning node keys secure

## Resources
- [LNbits Documentation](https://docs.lnbits.org)
- [Lightning Network Spec](https://github.com/lightning/bolts)
- [LNURL Specifications](https://github.com/lnurl/luds)

## Support
For issues specific to the Vrooli integration, check logs and ensure all dependencies are running. For LNbits-specific issues, consult the upstream documentation.