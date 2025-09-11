# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Universal authentication and authorization service that enables any scenario to secure its endpoints and UI with user management, session handling, and access control. This becomes the foundational security layer that transforms any Vrooli scenario from a demo into a production-ready application.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Agents can now build multi-user applications with role-based permissions, user-specific data, and secure API access. This enables agents to create SaaS products, enterprise tools, and collaborative platforms - dramatically expanding the complexity and value of scenarios they can generate.

### Recursive Value
**What new scenarios become possible after this exists?**
- **SaaS Billing Hub**: Can track user subscriptions and payments per authenticated user
- **Team Collaboration Tools**: Multiple users working on shared projects with permissions
- **Enterprise Dashboards**: Department-level access control and audit trails
- **Personal Data Vaults**: User-specific encrypted storage with OAuth integration
- **API Gateway Manager**: Rate limiting and API key management per authenticated client

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] User registration with email/password
  - [ ] User login with JWT token generation
  - [ ] Token validation middleware for APIs
  - [ ] Password reset functionality with email
  - [ ] Session management via Redis
  - [ ] User storage in PostgreSQL
  - [ ] Direct API token validation endpoint
  - [ ] CLI commands for user management
  
- **Should Have (P1)**
  - [ ] OAuth2 provider support (Google, GitHub)
  - [ ] Role-based access control (RBAC)
  - [ ] API key generation for programmatic access
  - [ ] Rate limiting per user/API key
  - [ ] Audit logging of auth events
  - [ ] Two-factor authentication (2FA)
  
- **Nice to Have (P2)**
  - [ ] SAML/SSO enterprise integration
  - [ ] Biometric authentication support
  - [ ] Passwordless login via magic links
  - [ ] User impersonation for admins
  - [ ] Session replay protection

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 50ms for token validation | API monitoring |
| Throughput | 10,000 auth checks/second | Load testing |
| Session Capacity | 100,000 concurrent sessions | Redis monitoring |
| Token Generation | < 100ms including DB write | Performance test |
| Password Hashing | < 200ms with bcrypt | Unit tests |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with postgres and redis
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Example integration in at least one other scenario

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: User account storage and audit logs
    integration_pattern: Direct SQL via Go database/sql
    access_method: resource-postgres CLI for setup, direct connection for runtime
    
  - resource_name: redis
    purpose: Session storage and token blacklisting
    integration_pattern: Redis client library
    access_method: resource-redis CLI for setup, direct connection for runtime
    
optional:
  - resource_name: mailpit
    purpose: Email sending for password reset
    fallback: Log email content to console
    access_method: SMTP protocol
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_resource_cli:
    - command: resource-postgres create-database scenario-authenticator
      purpose: Initialize auth database
    - command: resource-redis set-config maxmemory 256mb
      purpose: Configure session storage
  
  2_direct_api:
    - justification: Real-time session validation needs sub-millisecond response
      endpoint: Redis GET/SET/EXPIRE commands
    - justification: Email sending for password reset
      endpoint: Mailpit SMTP or console logging

api_integration_patterns:
  - Auth validation via GET/POST /api/v1/auth/validate
  - Password reset via POST /api/v1/auth/reset-password
  - Token refresh via POST /api/v1/auth/refresh
  - Session management via Redis direct connection
```

### Data Models
```yaml
primary_entities:
  - name: User
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        email: VARCHAR(255) UNIQUE NOT NULL,
        username: VARCHAR(100) UNIQUE,
        password_hash: VARCHAR(255) NOT NULL,
        roles: JSONB DEFAULT '["user"]',
        metadata: JSONB DEFAULT '{}',
        email_verified: BOOLEAN DEFAULT false,
        created_at: TIMESTAMP,
        updated_at: TIMESTAMP,
        last_login: TIMESTAMP
      }
    relationships: Has many Sessions, API Keys, Audit Logs
    
  - name: Session
    storage: redis
    schema: |
      {
        session_id: String (key),
        user_id: UUID,
        jwt_token: String,
        refresh_token: String,
        ip_address: String,
        user_agent: String,
        expires_at: Timestamp,
        created_at: Timestamp
      }
    relationships: Belongs to User
    
  - name: APIKey
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        user_id: UUID REFERENCES users(id),
        key_hash: VARCHAR(255) UNIQUE NOT NULL,
        name: VARCHAR(100),
        permissions: JSONB,
        last_used: TIMESTAMP,
        expires_at: TIMESTAMP,
        created_at: TIMESTAMP
      }
    relationships: Belongs to User
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/auth/register
    purpose: Create new user account
    input_schema: |
      {
        email: string (required),
        password: string (required, min 8 chars),
        username: string (optional),
        metadata: object (optional)
      }
    output_schema: |
      {
        success: boolean,
        user: { id, email, username },
        token: string (JWT),
        refresh_token: string
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/auth/login
    purpose: Authenticate user and create session
    input_schema: |
      {
        email: string (required),
        password: string (required)
      }
    output_schema: |
      {
        success: boolean,
        token: string (JWT),
        refresh_token: string,
        user: { id, email, roles }
      }
      
  - method: GET
    path: /api/v1/auth/validate
    purpose: Validate JWT token for other scenarios
    input_schema: |
      Headers: {
        Authorization: "Bearer <token>"
      }
    output_schema: |
      {
        valid: boolean,
        user_id: string,
        roles: array,
        expires_at: timestamp
      }
    sla:
      response_time: 50ms
      availability: 99.99%
      
  - method: POST
    path: /api/v1/auth/refresh
    purpose: Refresh expired JWT with refresh token
    
  - method: POST
    path: /api/v1/auth/logout
    purpose: Invalidate session and blacklist token
    
  - method: POST
    path: /api/v1/auth/reset-password
    purpose: Initiate password reset flow
```

### Event Interface
```yaml
published_events:
  - name: auth.user.registered
    payload: { user_id, email, timestamp }
    subscribers: [notification-hub, saas-billing-hub]
    
  - name: auth.user.logged_in
    payload: { user_id, ip_address, timestamp }
    subscribers: [audit-logger, usage-tracker]
    
  - name: auth.token.validated
    payload: { user_id, scenario_name, timestamp }
    subscribers: [rate-limiter, usage-metrics]
    
consumed_events:
  - name: user.data.deleted
    action: Remove user account and all sessions
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: scenario-authenticator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show auth service status and statistics
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: user create
    description: Create a new user account
    api_endpoint: /api/v1/auth/register
    arguments:
      - name: email
        type: string
        required: true
        description: User email address
      - name: password
        type: string
        required: true
        description: User password (min 8 chars)
    flags:
      - name: --username
        description: Optional username
      - name: --roles
        description: Comma-separated roles
    output: User ID and initial token
    
  - name: user list
    description: List all users with filters
    api_endpoint: /api/v1/users
    flags:
      - name: --role
        description: Filter by role
      - name: --active
        description: Only show active sessions
      - name: --limit
        description: Number of results
    output: Table of users or JSON array
    
  - name: token validate
    description: Validate a JWT token
    api_endpoint: /api/v1/auth/validate
    arguments:
      - name: token
        type: string
        required: true
        description: JWT token to validate
    output: Token validity and user info
    
  - name: session list
    description: List active sessions
    api_endpoint: /api/v1/sessions
    flags:
      - name: --user-id
        description: Filter by user
    output: Active session details
    
  - name: protect
    description: Generate integration code for a scenario
    arguments:
      - name: scenario
        type: string
        required: true
        description: Scenario name to protect
    flags:
      - name: --type
        description: Integration type (api|ui|workflow)
    output: Integration code snippet
```

### CLI-API Parity Requirements
- Every auth API endpoint has corresponding CLI command
- CLI provides additional utility commands (protect, generate-integration)
- All commands support JSON output for scripting
- Batch operations available via CLI for admin tasks

## 🔄 Integration Requirements

### Upstream Dependencies
- **PostgreSQL**: Database must be running for user storage
- **Redis**: Required for session management
- **Base Vrooli CLI**: Must be installed for resource commands

### Downstream Enablement
- **Every Business Scenario**: Can add authentication with 3 lines of code
- **SaaS Billing Hub**: User accounts enable subscription tracking
- **Team Collaboration**: Multi-user features with permissions
- **API Gateway**: Per-user rate limiting and API key management

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: ALL
    capability: User authentication and authorization
    interface: Direct API endpoints
    
  - scenario: saas-billing-hub
    capability: User accounts for subscription management
    interface: API user endpoints
    
  - scenario: notification-hub  
    capability: User-specific notification preferences
    interface: User ID from tokens
    
consumes_from:
  - scenario: notification-hub
    capability: Email sending for password reset
    fallback: Console logging of reset links
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Auth0 / Clerk - clean, trustworthy, secure
  
  visual_style:
    color_scheme: light with dark mode support
    typography: modern, clean sans-serif
    layout: centered card design for auth forms
    animations: subtle transitions, no flashy effects
  
  personality:
    tone: professional, secure, trustworthy
    mood: calm, reassuring
    target_feeling: Users should feel their data is safe

style_references:
  professional:
    - Similar to Clerk.dev login pages
    - Clean forms with clear validation
    - Professional without being cold
    - Mobile-responsive design
    - Accessibility compliant (WCAG 2.1 AA)
```

### Target Audience Alignment
- **Primary Users**: Developers integrating auth, end users logging in
- **User Expectations**: Fast, secure, familiar auth flow
- **Accessibility**: Full keyboard navigation, screen reader support
- **Responsive Design**: Mobile-first approach

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transforms any scenario into production-ready SaaS
- **Revenue Potential**: $10K-30K per secured application
- **Cost Savings**: 40-80 hours of development per scenario
- **Market Differentiator**: Zero-config auth for AI-generated apps

### Technical Value
- **Reusability Score**: 10/10 - Every scenario can use this
- **Complexity Reduction**: Auth becomes a one-line integration
- **Innovation Enablement**: Unlocks multi-tenant SaaS capabilities

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic authentication (register, login, logout)
- JWT tokens with refresh
- PostgreSQL user storage
- Redis session management
- Simple integration API

### Version 2.0 (Planned)
- OAuth2 providers (Google, GitHub, Microsoft)
- Role-based access control (RBAC)
- API key management
- Rate limiting per user
- Audit logging

### Long-term Vision
- Enterprise SSO/SAML support
- Biometric authentication
- Zero-knowledge proofs
- Decentralized identity integration
- AI-powered fraud detection

## 📈 Implementation Status

### ✅ Completed Features (v1.0)
- **Core Authentication**: User registration, login, logout fully functional
- **JWT Implementation**: Token generation with RS256 signing, refresh tokens
- **Database Integration**: PostgreSQL for user storage with proper schema
- **Session Management**: Redis-based session storage and invalidation
- **Password Security**: Bcrypt hashing with salt
- **API Endpoints**: All v1.0 endpoints implemented and tested
- **CLI Tool**: Full-featured CLI with dynamic port discovery
- **UI Dashboard**: Professional login/register interface
- **Health Monitoring**: Health check endpoints for orchestrator
- **Error Handling**: Consistent error messages and HTTP status codes
- **CORS Support**: Proper CORS headers for cross-origin requests
- **Integration Examples**: Code generation for protecting other scenarios

### 🚧 In Progress
- **Password Reset**: Email functionality (currently logs to console)
- **Rate Limiting**: Basic implementation needed
- **Audit Logging**: Database schema ready, implementation pending

### 📋 Not Yet Implemented (v2.0+)
- OAuth2 provider support (Google, GitHub)
- Advanced RBAC with custom permissions
- API key generation and management
- Two-factor authentication (2FA)
- Device fingerprinting
- Breach detection system

### 🔄 Recent Refactoring (2025-09-11)
- **Security Fix**: Removed hardcoded database password from schema.sql
- **Bug Fix**: Fixed UI server health check undefined variable
- **Test Coverage**: Added auth-flow.sh comprehensive test suite
- **Port Management**: Simplified port fallback logic with /config endpoint
- **Code Cleanup**: Removed redundant auth_processor.go file
- **Version Info**: Added version display to CLI commands
- **Error Consistency**: Verified consistent error message formatting

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Redis unavailable | Low | High | Fallback to PostgreSQL sessions |
| Token theft | Medium | High | Short expiry, refresh rotation |
| Password breach | Low | Critical | Bcrypt hashing, breach detection |
| Session hijacking | Medium | High | IP validation, device fingerprinting |

### Operational Risks
- **Security Updates**: Automated dependency scanning
- **Key Rotation**: Automatic JWT signing key rotation
- **Audit Compliance**: Comprehensive logging of all auth events
- **GDPR Compliance**: User data deletion capabilities

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: scenario-authenticator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/scenario-authenticator
    - cli/install.sh
    - initialization/postgres/schema.sql
    - ui/index.html
    - ui/auth.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/postgres
    - ui

resources:
  required: [postgres, redis]
  optional: [mailpit]
  health_timeout: 60

tests:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "User registration works"
    type: http
    service: api
    endpoint: /api/v1/auth/register
    method: POST
    body:
      email: test@example.com
      password: TestPass123!
    expect:
      status: 201
      body:
        success: true
        
  - name: "Token validation works"
    type: http
    service: api
    endpoint: /api/v1/auth/validate
    method: GET
    headers:
      Authorization: "Bearer ${token}"
    expect:
      status: 200
      body:
        valid: true
        
  - name: "CLI user creation works"
    type: exec
    command: ./cli/scenario-authenticator user create test2@example.com Pass456!
    expect:
      exit_code: 0
      output_contains: ["User created", "ID:"]
```

## 📝 Implementation Notes

### Design Decisions
**JWT vs Sessions**: Chose JWT for stateless validation and scenario portability
- Alternative considered: Server-side sessions only
- Decision driver: Scenarios need fast, independent validation
- Trade-offs: Slightly larger tokens for self-contained claims

**Redis for Sessions**: Fast session validation and token blacklisting
- Alternative considered: PostgreSQL only
- Decision driver: Sub-millisecond validation requirements
- Trade-offs: Additional dependency for significant performance gain

### Known Limitations
- **No LDAP/AD**: Focus on modern auth methods first
  - Workaround: OAuth2 proxy for enterprise
  - Future fix: Version 3.0 enterprise features
  
- **Single Region**: Sessions don't replicate across regions
  - Workaround: Sticky sessions at load balancer
  - Future fix: Redis cluster support

### Security Considerations
- **Data Protection**: Bcrypt for passwords, encrypted session data
- **Access Control**: Role-based permissions, API key scoping
- **Audit Trail**: Every auth action logged with context

## 🔗 References

### Documentation
- README.md - Quick start guide
- docs/integration.md - How to add auth to your scenario
- docs/api.md - Full API specification
- docs/security.md - Security best practices

### Related PRDs
- saas-billing-hub/PRD.md - Depends on user accounts
- notification-hub/PRD.md - Uses auth for user preferences
- api-gateway/PRD.md - Will integrate auth for rate limiting

---

**Last Updated**: 2025-01-04  
**Status**: In Development  
**Owner**: AI Agent  
**Review Cycle**: Weekly during initial development