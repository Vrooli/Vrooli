# Product Requirements Document: LNbits Lightning Network Payments

## Executive Summary
**Product**: LNbits - Open-source Lightning Network wallet and accounts system for Bitcoin micropayments
**Purpose**: Enable scenarios to process instant, low-fee Bitcoin micropayments via Lightning Network with extensible payment infrastructure
**Target Users**: Scenarios requiring payment processing, monetization, subscriptions, pay-per-use features, or crypto-native applications
**Value Proposition**: Generate $25K-50K revenue potential through micropayment-enabled services, subscription models, and value-for-value content monetization
**Priority**: High - Critical infrastructure for Web3 monetization and decentralized payment processing

## Problem Statement
Vrooli scenarios currently lack efficient micropayment capabilities, limiting monetization options for pay-per-use services, content monetization, and real-time value exchange. Traditional payment processors have high fees (2-3%) making micropayments uneconomical. Lightning Network solves this with instant settlements and near-zero fees.

## Success Metrics
- **Completion**: 100% of P0 requirements functional, 80% of P1 requirements
- **Performance**: <100ms payment processing, <1s wallet creation
- **Reliability**: 99.9% uptime for payment processing
- **Integration**: Compatible with BTCPay, n8n workflows, and Vault for key management

## Technical Specifications

### Architecture
- **Core**: Python 3.9+ with FastAPI web framework
- **Database**: PostgreSQL for wallet/transaction data
- **Lightning Backend**: Multiple funding sources (LND, Core Lightning, Eclair)
- **Extensions**: Plugin system for additional functionality
- **API**: RESTful API with WebSocket support for real-time updates

### Dependencies
- Python 3.9+
- PostgreSQL 14+
- Redis for caching
- Docker for containerization
- Lightning node backend (optional - can use custodial)

### Resource Requirements
- **Memory**: 512MB minimum, 1GB recommended
- **CPU**: 1 core minimum, 2 cores recommended
- **Storage**: 1GB for application, 10GB+ for database
- **Ports**: 5001 (API), 5432 (PostgreSQL internal)

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Service Lifecycle**: Complete start/stop/restart with health checks responding within 5s
- [ ] **Wallet Management**: Create, list, and manage Lightning wallets via API
- [ ] **Payment Processing**: Send/receive Lightning payments with BOLT11 invoices
- [ ] **API Authentication**: Secure API key generation and management for wallet access
- [ ] **LNURL Support**: Handle LNURL-pay and LNURL-withdraw for seamless payments
- [ ] **Transaction History**: Store and retrieve payment history with filtering
- [ ] **Extension Framework**: Load and manage at least one core extension (e.g., LNURLp)

### P1 Requirements (Should Have)
- [ ] **NOSTR Integration**: NIP-57 zaps support for social payments
- [ ] **WebSocket Events**: Real-time payment notifications via WebSocket
- [ ] **Multi-Currency Display**: Show balances in BTC, sats, and fiat equivalents
- [ ] **Backup/Restore**: Export and import wallet configurations

### P2 Requirements (Nice to Have)
- [ ] **Lightning Address**: Support username@domain Lightning addresses
- [ ] **Point of Sale**: Basic POS interface for merchant payments
- [ ] **Split Payments**: Automatic payment splitting between multiple wallets

## User Scenarios

### Scenario 1: Content Monetization Platform
A content platform uses LNbits to enable creators to receive instant micropayments for articles, videos, or API calls. Users pay 100 sats per article view, with instant settlement to creator wallets.

### Scenario 2: IoT Device Payments
IoT devices use LNbits API to autonomously pay for resources like data storage, compute time, or sensor data access using programmatic Lightning payments.

### Scenario 3: Subscription Service
A SaaS platform implements pay-per-use billing where users fund a Lightning wallet and services deduct micropayments in real-time based on usage.

## Integration Points

### Inbound Integrations
- **BTCPay Server**: On-chain to Lightning swaps
- **Vault**: Secure storage of Lightning node credentials
- **PostgreSQL**: Shared database for transaction data
- **Redis**: Caching layer for performance

### Outbound Integrations
- **n8n**: Trigger workflows on payment events
- **Webhooks**: Payment notifications to external services
- **Prometheus**: Metrics export for monitoring

## Security Considerations
- API keys stored in environment variables only
- Rate limiting on payment endpoints
- Wallet isolation with unique API keys
- Optional macaroon authentication for LND
- TLS encryption for Lightning node connections

## Risks and Mitigation
- **Risk**: Lightning node connectivity issues
  - **Mitigation**: Support multiple funding sources, fallback to custodial
- **Risk**: Database corruption losing wallet data
  - **Mitigation**: Regular PostgreSQL backups, transaction logs
- **Risk**: High payment volume overwhelming system
  - **Mitigation**: Redis caching, connection pooling, horizontal scaling

## Revenue Model
- **Direct**: $25K from payment processing fees (0.1% on $25M volume)
- **Indirect**: $25K+ from enabling monetized scenarios
- **Total Potential**: $50K+ annually per deployment

## Development Phases
1. **Phase 1**: Core setup with PostgreSQL and basic wallet creation
2. **Phase 2**: Lightning payment processing with BOLT11
3. **Phase 3**: LNURL and extension framework
4. **Phase 4**: Advanced features (NOSTR, WebSockets)

## Testing Strategy
- Unit tests for wallet operations
- Integration tests with Lightning testnet
- Load testing for concurrent payments
- Security audit of API endpoints

## Documentation Requirements
- API reference with examples
- Integration guide for scenarios
- Security best practices
- Troubleshooting guide

## Future Enhancements
- Lightning Service Provider (LSP) integration
- Atomic Multi-Path Payments (AMP)
- Taproot asset support
- Cross-chain swaps

## Competitive Analysis
- **BTCPay**: Better for on-chain, LNbits better for Lightning
- **OpenNode**: Custodial only, LNbits supports self-custody
- **Strike API**: US-only, LNbits is global

## Success Criteria
- Process 1000+ payments daily
- <100ms payment latency
- 99.9% payment success rate
- 5+ scenarios integrated within 30 days

## Progress Tracking
- **Date**: 2025-01-11
- **Completion**: 14% (1/7 P0 requirements)
- **Status**: Basic scaffolding complete with lifecycle management
- **Next Steps**: Future improvers to implement wallet management and payment processing
- **Implementation Notes**: 
  - v2.0 contract compliant structure created
  - CLI interface with all required commands
  - Port allocated in registry (5001)
  - Docker-based deployment ready
  - Test framework established