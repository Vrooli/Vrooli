# Matrix Synapse Resource PRD

## Executive Summary
**What**: Matrix Synapse homeserver resource providing federated, encrypted, real-time communication infrastructure for Vrooli scenarios
**Why**: Enable secure decentralized communication, team collaboration, notification channels, and chatbot interfaces across all scenarios
**Who**: All multi-user scenarios, customer support applications, team collaboration tools, and any scenario requiring real-time messaging
**Value**: Enables $500K+ in scenario value through communication infrastructure (100+ scenarios × $5K average value from chat/collaboration features)
**Priority**: P0 - Core communication infrastructure required by many scenarios

## Differentiation from Existing Resources

### Similar Resources Analyzed
- **Twilio**: 10% overlap - SMS/voice only, not real-time chat or team collaboration
- **Pushover**: 5% overlap - One-way notifications only, no bidirectional communication
- **Redis**: 0% overlap - Data store, not communication platform

### Why This Isn't a Duplicate
Matrix Synapse provides unique federated, decentralized communication that no existing resource offers. It enables:
- Real-time bidirectional messaging with E2E encryption
- Federation with other Matrix servers globally
- Bot integration for automation scenarios
- Room-based collaboration with persistent history
- Bridge connections to Slack, Discord, IRC, and other platforms
- Voice/video calling capabilities through WebRTC

### Integration vs. Creation Decision
No existing resource provides real-time chat infrastructure. Matrix is the industry standard for federated communication and cannot be replaced by configuration of existing tools.

## Requirements

### P0 Requirements (Must Have)
- [x] **PostgreSQL Integration**: Use existing PostgreSQL resource as backend (not SQLite) for production readiness and data persistence ✅ 2025-01-10
- [x] **Health Check Endpoint**: Respond to health checks within 5 seconds at `/_matrix/client/versions` ✅ 2025-01-10
- [x] **v2.0 Contract Compliance**: Full implementation of all required CLI commands and lifecycle hooks ✅ 2025-01-10
- [ ] **User Registration**: Enable user registration and authentication via shared registration secret
- [ ] **Room Management**: Create, join, and manage Matrix rooms programmatically
- [ ] **Message Sending**: Send and receive messages through REST API
- [ ] **Federation Setup**: Configure server_name and well-known delegation for federation

### P1 Requirements (Should Have)  
- [ ] **TURN Server Integration**: Configure coturn for voice/video calls
- [ ] **Bot User Creation**: Automated bot user creation for scenarios
- [ ] **Bridge Management**: Setup bridges to Slack/Discord/IRC
- [ ] **Media Repository**: Configure media storage with size limits

### P2 Requirements (Nice to Have)
- [ ] **Dendrite Alternative**: Option to use lighter Dendrite for resource-constrained environments
- [ ] **Metrics Export**: Prometheus metrics endpoint for monitoring
- [ ] **Backup/Restore**: Database backup and restore commands

## Technical Specifications

### Architecture
- **Service**: Matrix Synapse homeserver v1.100+
- **Database**: PostgreSQL 14+ (via postgres resource)
- **Cache**: Redis (optional, via redis resource)
- **Storage**: Local filesystem for media (future: MinIO integration)
- **Port**: 8008 (client/federation API)
- **Protocol**: HTTPS with TLS for federation

### Dependencies
```yaml
resources:
  - postgres    # Required for database backend
  - redis       # Optional for caching
  - vault       # Optional for secret management
```

### Configuration
```yaml
# homeserver.yaml key configuration
server_name: "vrooli.local"  # Can be overridden
database:
  name: psycopg2
  args:
    host: localhost
    port: 5433  # Use postgres resource port
    database: synapse
    user: synapse
    password: ${SYNAPSE_DB_PASSWORD}
registration_shared_secret: ${SYNAPSE_REGISTRATION_SECRET}
enable_registration: true
federation_domain_whitelist: []  # Open federation by default
max_upload_size: "50M"
```

### API Endpoints
```yaml
# Client-Server API
GET  /_matrix/client/versions          # Version info & health
POST /_matrix/client/v3/register       # User registration
POST /_matrix/client/v3/login          # User login
POST /_matrix/client/v3/rooms          # Create room
PUT  /_matrix/client/v3/rooms/{roomId}/send/{eventType}/{txnId}  # Send message
GET  /_matrix/client/v3/sync           # Long-poll for events

# Admin API (requires admin user)
GET  /_synapse/admin/v1/users          # List users
POST /_synapse/admin/v1/register       # Register user with shared secret
```

### CLI Commands
```bash
# v2.0 Contract Required Commands
resource-matrix-synapse help
resource-matrix-synapse info
resource-matrix-synapse manage install/start/stop/restart/uninstall
resource-matrix-synapse test smoke/integration/unit/all
resource-matrix-synapse status
resource-matrix-synapse logs

# Content Management (Matrix-specific)
resource-matrix-synapse content add-user <username>      # Create user
resource-matrix-synapse content list-users               # List all users
resource-matrix-synapse content create-room <name>       # Create room
resource-matrix-synapse content send-message <room> <message>  # Send message
resource-matrix-synapse content add-bot <name>           # Create bot user
```

### Performance Requirements
- **Startup Time**: <30 seconds for health check response
- **Message Latency**: <100ms for local message delivery
- **Concurrent Users**: Support 100+ concurrent connections
- **Database Size**: Handle 1GB+ message history
- **Federation Latency**: <500ms for federated message delivery

### Security Requirements
- **TLS**: Required for federation (self-signed allowed for local)
- **Registration Secret**: Secure random string for user creation
- **Database Credentials**: Stored in environment variables only
- **Rate Limiting**: Enabled by default to prevent abuse
- **E2E Encryption**: Enabled by default for private rooms

## Success Metrics

### Completion Metrics
- **P0 Completion**: 100% of P0 requirements functional
- **Test Coverage**: All test phases pass (smoke, integration, unit)
- **Health Check**: Responds within 5 seconds consistently
- **CLI Compliance**: All v2.0 commands functional

### Quality Metrics
- **Startup Success Rate**: >95% successful starts
- **Message Delivery Rate**: >99.9% successful delivery
- **API Response Time**: <100ms for 95th percentile
- **Federation Success**: Can federate with matrix.org

### Business Metrics
- **Scenario Adoption**: 20+ scenarios using Matrix within 3 months
- **User Capacity**: Support 1000+ users across all scenarios
- **Message Volume**: Handle 100K+ messages/day
- **Bridge Connections**: 5+ platform bridges configured

## Implementation Approach

### Phase 1: Core Setup (Week 1)
1. Implement v2.0 directory structure
2. Create PostgreSQL database setup
3. Configure basic Synapse installation
4. Implement health check endpoint

### Phase 2: API Integration (Week 2)
1. User registration with shared secret
2. Room creation and management
3. Message sending/receiving
4. Basic federation setup

### Phase 3: Advanced Features (Week 3)
1. Bot user management
2. Bridge configuration templates
3. Backup/restore functionality
4. Performance optimization

## Risk Mitigation

### Technical Risks
- **Database Performance**: Use connection pooling and indexing
- **Federation Complexity**: Start with local-only, add federation later
- **Resource Usage**: Monitor memory/CPU, consider Dendrite for light deployments
- **Port Conflicts**: Use port 8008 (uncommon, unlikely to conflict)

### Security Risks
- **Open Registration**: Use shared secret, disable public registration
- **Federation Exposure**: Whitelist trusted servers initially
- **Media Uploads**: Set size limits and scan for malware
- **Database Access**: Use restricted database user

## Testing Strategy

### Smoke Tests (<30s)
- Health endpoint responds
- Database connection works
- Can create test user
- Can create test room

### Integration Tests (<120s)
- Full user lifecycle (register/login/logout)
- Message send/receive flow
- Room creation and joining
- PostgreSQL integration verified

### Unit Tests (<60s)
- Configuration validation
- CLI command parsing
- Database connection handling
- API client functions

## Documentation Requirements

### README.md
- Quick start guide
- Configuration examples
- Common use cases
- Troubleshooting guide

### API Documentation
- Authentication flow
- Message format examples
- Room management guide
- Bot creation tutorial

### Integration Guides
- PostgreSQL setup
- Bridge configuration
- Federation setup
- Security hardening

## Future Enhancements

### Version 2.0
- Element Web UI integration
- Voice/video calling with TURN
- Full bridge support (Slack, Discord, IRC)
- Prometheus metrics export

### Version 3.0
- Dendrite alternative option
- MinIO media storage
- Kubernetes deployment
- Multi-server federation mesh

## Revenue Model

### Direct Value
- **Scenario Enhancement**: Each scenario gains $5K value from chat features
- **Multi-user Enablement**: Unlocks collaborative scenarios worth $50K+ each
- **Customer Support**: Enables support desk scenarios worth $100K+

### Indirect Value
- **Network Effects**: More users → more valuable platform
- **Integration Hub**: Bridges connect to external platforms
- **Automation Platform**: Bots enable workflow automation

### Monetization Opportunities
- **Hosted Matrix Service**: $50/month per organization
- **Bridge Connectors**: $20/month per bridge
- **Bot Development**: $500 per custom bot
- **Support Contracts**: $200/month for managed service

## Dependencies and Integration Points

### Required Resources
- **postgres**: Database backend (required)
- **redis**: Caching layer (optional but recommended)

### Scenario Integrations
- **customer-support**: Ticket chat channels
- **team-workspace**: Collaboration rooms
- **notification-hub**: Alert distribution
- **bot-framework**: Automation interface
- **social-dashboard**: Message analytics

## Appendix

### Matrix Specification
- Spec Version: v1.5
- Room Version: 10
- Federation Version: v2

### Useful References
- [Matrix Specification](https://spec.matrix.org/)
- [Synapse Documentation](https://matrix-org.github.io/synapse/)
- [Federation Tester](https://federationtester.matrix.org/)
- [Element Web Client](https://element.io/)

### Configuration Templates
- Basic homeserver.yaml
- PostgreSQL database setup
- Federation well-known files
- Nginx reverse proxy config