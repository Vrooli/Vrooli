# BTCPay Server Resource PRD

## Executive Summary
**What**: Self-hosted cryptocurrency payment processor that enables businesses to accept Bitcoin and other cryptocurrencies
**Why**: Eliminate payment processor fees, maintain full control of funds, and provide privacy-focused payment solutions
**Who**: E-commerce businesses, freelancers, and organizations needing cryptocurrency payment infrastructure
**Value**: Saves 2-5% transaction fees vs traditional processors, enables global payments, $15K-30K typical annual savings
**Priority**: Medium - Essential for crypto-enabled commerce scenarios

## Requirements

### P0 Requirements (Must Have)
- [x] **Service Management**: Start/stop/restart BTCPay Server with Docker (✅ NBXplorer integrated)
- [x] **Health Validation**: Respond to health checks within 5 seconds (✅ timeout added)
- [x] **Basic Configuration**: Configure network, ports, and database connection
- [x] **Invoice Creation**: Generate payment invoices via API (✅ implemented with CLI wrapper)
- [x] **Payment Status**: Check payment confirmation status (✅ implemented with CLI wrapper)
- [x] **v2.0 Contract**: Full compliance with universal resource contract
- [x] **Test Coverage**: Smoke, integration, and unit tests functioning

### P1 Requirements (Should Have)
- [x] **Store Management**: Create and configure payment stores (✅ API functions implemented)
- [x] **Webhook Support**: Send payment notifications to external systems (✅ webhook configuration added)
- [ ] **Lightning Network**: Support for Lightning Network payments (PARTIAL: Container starts but needs Bitcoin node)
- [x] **Multi-Currency**: Support multiple cryptocurrencies (✅ BTC, LTC configuration implemented)

### P2 Requirements (Nice to Have)
- [x] **Point of Sale**: Built-in POS system for retail (✅ Complete POS system with inventory management)
- [x] **Crowdfunding**: Enable crowdfunding campaigns (✅ Campaign management with flag and positional argument support)
- [x] **Payment Buttons**: Generate embeddable payment buttons (✅ Button generator with multiple styles)

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
- **P0 Completion**: 100% (7/7 requirements)
- **P1 Completion**: 75% (3/4 requirements)
- **P2 Completion**: 100% (3/3 requirements)
- **Overall Progress**: 93%

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
- 2025-09-13: Completed all P0 requirements (57% → 100% P0, 50% overall)
  - Added timeout handling to health checks per v2.0 contract
  - Implemented invoice creation functionality with API support
  - Implemented payment status checking with API support
  - Added CLI wrapper functions for invoice operations
  - All tests passing (smoke, integration, unit)
- 2025-09-14: Implemented P1 requirements (50% → 64% overall)
  - Integrated NBXplorer container for blockchain synchronization
  - Added store management API functionality (create, list stores)
  - Implemented webhook configuration support for payment notifications
  - Fixed NBXplorer PostgreSQL connection requirements
  - All integration tests passing with NBXplorer running
- 2025-09-14: Implemented Lightning Network support (64% → 71% overall)
  - Added LND (Lightning Network Daemon) integration
  - Created comprehensive Lightning CLI commands (setup, status, invoices, channels)
  - Implemented Lightning-specific test suite
  - Added balance checking and channel management features
  - Configured automatic LND container management
- 2025-09-15: Completed all P1 requirements and added POS system (71% → 83% overall)
  - Implemented multi-currency support with BTC, LTC, ETH configuration
  - Added dynamic chain configuration for docker containers
  - Created multicurrency CLI command group with full management capabilities
  - Implemented complete Point of Sale system with inventory management
  - Added POS terminal HTML generation for web-based checkout
  - Created item management system with add/remove/update/import capabilities
  - All P1 requirements now complete (100%)

- 2025-09-16: Completed all P2 requirements (83% → 100% overall claimed)
  - Implemented crowdfunding campaign management with full CRUD operations
  - Added campaign contribution tracking and export functionality
  - Created embeddable widget generator for campaigns
  - Implemented payment button generator with multiple styles
  - Added bulk button creation from CSV
  - Created button statistics tracking
  - All 10 integration tests passing

- 2025-09-26: Validation and corrections (100% claimed → 93% actual)
  - Fixed missing timeouts in lib/test.sh and lib/start.sh per v2.0 contract
  - Validated all P0 requirements working correctly
  - Lightning Network requires Bitcoin node for full functionality (marked PARTIAL)
  - Crowdfunding works but has CLI argument parsing issue with flags
  - All smoke and integration tests passing

- 2025-09-26: Improvements and bug fixes (93% → 93%)
  - Fixed crowdfunding CLI parsing to support both positional and flag-based arguments
  - Cleaned up malformed campaign JSON files from previous parsing errors
  - Verified all health checks have proper timeout wrappers per v2.0 contract
  - Updated documentation to reflect resolved issues

- 2025-09-26: Final validation and improvements (93% maintained)
  - Fixed POS configuration argument parsing to support both formats
  - Corrected POS config file with swapped values (storeName/currency)
  - All tests passing: 5/5 smoke, 10/10 integration, 7/7 unit
  - Full v2.0 contract compliance confirmed
  - Resource fully operational except Lightning (needs Bitcoin Core)

### Current Status
The resource is highly functional with:
- Full v2.0 contract compliance
- Complete payment processing capabilities (invoices, status checking)
- Working merchant tools (POS, crowdfunding, payment buttons)
- Multi-currency support (BTC, LTC)
- Lightning Network infrastructure (needs Bitcoin node for activation)
- All tests passing (5/5 smoke, 10/10 integration, 7/7 unit)