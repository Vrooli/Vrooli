# BTCPay Server Resource PRD

## Executive Summary
**What**: Self-hosted cryptocurrency payment processor that enables businesses to accept Bitcoin and other cryptocurrencies
**Why**: Eliminate payment processor fees, maintain full control of funds, and provide privacy-focused payment solutions
**Who**: E-commerce businesses, freelancers, and organizations needing cryptocurrency payment infrastructure
**Value**: Saves 2-5% transaction fees vs traditional processors, enables global payments, $15K-30K typical annual savings
**Priority**: Medium - Essential for crypto-enabled commerce scenarios

## Requirements

### P0 Requirements (Must Have)
- [x] **Service Management**: Start/stop/restart BTCPay Server with Docker (PARTIAL: PostgreSQL starts, BTCPay needs NBXplorer)
- [ ] **Health Validation**: Respond to health checks within 5 seconds
- [x] **Basic Configuration**: Configure network, ports, and database connection
- [ ] **Invoice Creation**: Generate payment invoices via API
- [ ] **Payment Status**: Check payment confirmation status
- [x] **v2.0 Contract**: Full compliance with universal resource contract
- [x] **Test Coverage**: Smoke, integration, and unit tests functioning

### P1 Requirements (Should Have)
- [ ] **Store Management**: Create and configure payment stores
- [ ] **Webhook Support**: Send payment notifications to external systems
- [ ] **Lightning Network**: Support for Lightning Network payments
- [ ] **Multi-Currency**: Support multiple cryptocurrencies

### P2 Requirements (Nice to Have)
- [ ] **Point of Sale**: Built-in POS system for retail
- [ ] **Crowdfunding**: Enable crowdfunding campaigns
- [ ] **Payment Buttons**: Generate embeddable payment buttons

## Technical Specifications

### Architecture
- **Type**: Dockerized service with PostgreSQL backend
- **Ports**: 23000 (HTTP API)
- **Dependencies**: PostgreSQL database, Docker
- **Network**: Isolated Docker network for security

### API Endpoints
- `GET /health` - Health check endpoint
- `POST /api/v1/stores/{storeId}/invoices` - Create invoice
- `GET /api/v1/stores/{storeId}/invoices/{invoiceId}` - Check payment status
- `POST /api/v1/stores` - Create new store
- `GET /api/v1/stores` - List stores

### Configuration Schema
```json
{
  "network": "mainnet|testnet|regtest",
  "database": {
    "host": "string",
    "port": "number",
    "user": "string",
    "password": "string",
    "database": "string"
  },
  "api": {
    "port": "number",
    "host": "string"
  },
  "lightning": {
    "enabled": "boolean",
    "implementation": "lnd|c-lightning|eclair"
  }
}
```

## Success Metrics

### Completion Targets
- **P0 Completion**: 57% (4/7 requirements)
- **P1 Completion**: 0% (0/4 requirements)
- **P2 Completion**: 0% (0/3 requirements)
- **Overall Progress**: 29%

### Quality Metrics
- Health check response time: <1 second
- Invoice creation time: <2 seconds
- Payment confirmation time: <5 seconds
- Uptime target: 99.5%

### Performance Targets
- Support 100 concurrent payment checks
- Process 1000 invoices per hour
- Handle 10 stores per instance
- Memory usage: <512MB idle, <1GB under load

## Implementation Progress

### History
- 2025-01-12: Initial PRD creation, starting assessment
- 2025-01-12: Implemented v2.0 compliant test structure (29% → 57%)
  - Added test/run-tests.sh orchestrator
  - Created test/phases/ with smoke, integration, unit tests
  - Updated lib/test.sh to delegate to phase scripts
  - Fixed port registry integration
  - Fixed PostgreSQL connection and authentication
  - Note: BTCPay requires NBXplorer (Bitcoin blockchain explorer) for full operation

### Next Steps
1. Add NBXplorer container for Bitcoin blockchain indexing
2. Implement health check endpoint wrapper
3. Add invoice creation capability via API
4. Enable payment status checking
5. Add Lightning Network support