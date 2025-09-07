# Email Triage - Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
AI-powered email management system that intelligently processes, prioritizes, and acts on emails using natural language rules and semantic understanding. This transforms email from a time-sink into an intelligent assistant that automatically handles routine communications while surfacing only the most important messages that require human attention.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Agents can now build sophisticated communication workflows that understand context, intent, and priority. The semantic search and AI rule generation capabilities enable agents to create email-based automation that goes far beyond simple keyword matching - they can understand business context, relationship importance, and communication patterns to make intelligent decisions.

### Recursive Value
**What new scenarios become possible after this exists?**
- **Customer Support Automation**: AI agents that triage support tickets and auto-generate responses
- **Sales Pipeline Manager**: Automatically categorize leads, follow-ups, and opportunities
- **Executive Assistant**: AI that manages executive communications with intelligent prioritization
- **Legal Document Processor**: Identify contract deadlines, compliance issues, and urgent legal matters
- **Research Aggregator**: Automatically collect and organize research papers, news, and industry updates

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Multi-tenant user accounts with email profile isolation
  - [ ] IMAP/SMTP integration with mail-in-a-box for email access
  - [ ] AI-powered rule creation assistant using natural language
  - [ ] Email semantic search with vector embeddings in Qdrant
  - [ ] Smart email prioritization with ML scoring
  - [ ] Basic triage actions (forward, archive, mark important, auto-reply)
  - [ ] Real-time email processing and notification integration
  - [ ] Web dashboard for email management and rule configuration
  
- **Should Have (P1)**
  - [ ] Advanced automation rules (scheduled sends, follow-up reminders)
  - [ ] Email thread analysis and conversation tracking
  - [ ] Integration with external calendars for meeting context
  - [ ] Bulk email operations with undo capability
  - [ ] Email analytics and productivity insights
  - [ ] Mobile-responsive UI for on-the-go management
  
- **Nice to Have (P2)**
  - [ ] Email signature and template management
  - [ ] Integration with CRM systems via APIs
  - [ ] Advanced AI features (sentiment analysis, urgency detection)
  - [ ] Team collaboration on shared email accounts
  - [ ] Email scheduling and send-later functionality

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Email Processing Speed | < 2s per email | Processing pipeline monitoring |
| Search Response Time | < 500ms for semantic queries | Qdrant query performance |
| Rule Creation Time | < 30s with AI assistant | User workflow tracking |
| Dashboard Load Time | < 1.5s initial load | Frontend performance monitoring |
| Triage Accuracy | > 85% for AI-suggested actions | User feedback validation |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with mail-in-a-box, qdrant, and scenario-authenticator
- [ ] Performance targets met under load (1000+ emails/user)
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Multi-tenant isolation verified with security audit
- [ ] AI rule generation produces useful, actionable rules

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: User profiles, email metadata, rules, and audit logs
    integration_pattern: Direct SQL via Go database/sql
    access_method: resource-postgres CLI for setup, direct connection for runtime
    
  - resource_name: qdrant
    purpose: Vector embeddings for semantic email search
    integration_pattern: Qdrant Go client library
    access_method: resource-qdrant CLI for setup, direct API for runtime
    
  - resource_name: scenario-authenticator
    purpose: Multi-tenant user authentication and authorization
    integration_pattern: JWT token validation API
    access_method: API endpoint /api/v1/auth/validate
    
  - resource_name: mail-in-a-box
    purpose: IMAP/SMTP email server access
    integration_pattern: IMAP/SMTP protocol clients
    access_method: Direct protocol connection via Go libraries
    
optional:
  - resource_name: notification-hub
    purpose: Alert users about urgent emails and system events
    fallback: Console logging and basic email notifications
    access_method: API endpoint /api/v1/notifications/send
    
  - resource_name: ollama
    purpose: AI rule generation and email analysis
    fallback: Basic keyword-based rules only
    access_method: resource-ollama CLI for LLM inference
    
  - resource_name: redis
    purpose: Email processing queue and caching
    fallback: In-memory processing (limited scalability)
    access_method: Redis Go client for queue management
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_direct_api:          # Primary: Direct API integration for real-time performance
    - justification: Real-time email processing requires immediate response
      endpoints: 
        - Qdrant vector operations
        - PostgreSQL transactions
        - IMAP/SMTP protocols
  
  2_resource_cli:        # Secondary: CLI for setup and maintenance
    - command: resource-postgres create-database email-triage
      purpose: Initialize multi-tenant database
    - command: resource-qdrant create-collection emails
      purpose: Setup vector collection for semantic search
    - command: resource-ollama generate
      purpose: AI-powered rule generation
  
  3_shared_workflows:     # Avoided: No n8n workflows as requested
    - note: Direct Go implementation for all email processing workflows
```

### Data Models
```yaml
primary_entities:
  - name: User
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        email_profiles: JSONB[] // Multiple email accounts per user
        preferences: JSONB DEFAULT '{}',
        plan_type: VARCHAR(50) DEFAULT 'free',
        created_at: TIMESTAMP,
        updated_at: TIMESTAMP
      }
    relationships: Has many EmailAccounts, Rules, ProcessedEmails
    
  - name: EmailAccount
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        user_id: UUID REFERENCES users(id),
        email_address: VARCHAR(255) NOT NULL,
        imap_settings: JSONB NOT NULL,
        smtp_settings: JSONB NOT NULL,
        last_sync: TIMESTAMP,
        sync_enabled: BOOLEAN DEFAULT true
      }
    relationships: Belongs to User, Has many ProcessedEmails
    
  - name: TriageRule
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        user_id: UUID REFERENCES users(id),
        name: VARCHAR(255) NOT NULL,
        conditions: JSONB NOT NULL, // AI-generated conditions
        actions: JSONB NOT NULL,    // Forward, archive, label, etc.
        priority: INTEGER DEFAULT 100,
        enabled: BOOLEAN DEFAULT true,
        created_by_ai: BOOLEAN DEFAULT false,
        created_at: TIMESTAMP
      }
    relationships: Belongs to User
    
  - name: ProcessedEmail
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        account_id: UUID REFERENCES email_accounts(id),
        message_id: VARCHAR(500) UNIQUE NOT NULL,
        subject: TEXT,
        sender_email: VARCHAR(255),
        recipient_emails: TEXT[],
        body_preview: TEXT,
        priority_score: FLOAT DEFAULT 0.5,
        vector_id: UUID, // Reference to Qdrant vector
        processed_at: TIMESTAMP,
        actions_taken: JSONB DEFAULT '[]'
      }
    relationships: Belongs to EmailAccount
    
  - name: EmailVector
    storage: qdrant
    schema: |
      {
        id: UUID,
        vector: FLOAT[384], // Email content embedding
        payload: {
          email_id: UUID,
          user_id: UUID,
          subject: String,
          sender: String,
          timestamp: Integer,
          priority_score: Float
        }
      }
    relationships: References ProcessedEmail
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/accounts
    purpose: Connect new email account to user profile
    input_schema: |
      {
        email_address: string (required),
        imap_server: string (required),
        imap_port: number (required),
        smtp_server: string (required), 
        smtp_port: number (required),
        password: string (required),
        use_tls: boolean (default: true)
      }
    output_schema: |
      {
        success: boolean,
        account_id: string,
        status: string // "connected" | "testing" | "failed"
      }
    sla:
      response_time: 5000ms  // Email connection testing can be slow
      availability: 99.5%
      
  - method: POST
    path: /api/v1/rules
    purpose: Create new triage rule with AI assistance
    input_schema: |
      {
        description: string (required), // Natural language rule description
        use_ai_generation: boolean (default: true),
        manual_conditions: object (optional),
        actions: array (required) // forward, archive, label, etc.
      }
    output_schema: |
      {
        success: boolean,
        rule_id: string,
        generated_conditions: object,
        ai_confidence: float,
        preview_matches: number
      }
    sla:
      response_time: 15000ms  // AI processing can take time
      availability: 99%
      
  - method: GET
    path: /api/v1/emails/search
    purpose: Semantic search across user's processed emails
    input_schema: |
      Query params: {
        q: string (required),     // Search query
        limit: number (default: 20),
        account_id: string (optional),
        date_from: string (optional),
        date_to: string (optional)
      }
    output_schema: |
      {
        results: [{
          email_id: string,
          subject: string,
          sender: string,
          preview: string,
          similarity_score: float,
          processed_at: string
        }],
        total: number,
        query_time_ms: number
      }
    sla:
      response_time: 500ms
      availability: 99.9%
```

### Event Interface
```yaml
published_events:
  - name: email.processed
    payload: { email_id, user_id, priority_score, actions_taken, timestamp }
    subscribers: [notification-hub, analytics-collector]
    
  - name: rule.triggered
    payload: { rule_id, email_id, actions, confidence_score, timestamp }
    subscribers: [audit-logger, performance-tracker]
    
  - name: user.email_quota_reached
    payload: { user_id, plan_type, current_usage, limit, timestamp }
    subscribers: [notification-hub, billing-system]
    
consumed_events:
  - name: auth.user.deleted
    action: Remove all user data including email accounts and processed emails
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: email-triage
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show email processing status and account health
    flags: [--json, --verbose, --account <id>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: account add
    description: Add new email account for processing
    api_endpoint: /api/v1/accounts
    arguments:
      - name: email
        type: string
        required: true
        description: Email address to connect
      - name: password
        type: string
        required: true
        description: Email account password
    flags:
      - name: --imap-server
        description: IMAP server hostname (auto-detected if not provided)
      - name: --smtp-server
        description: SMTP server hostname (auto-detected if not provided)
    output: Account ID and connection status
    
  - name: rule create
    description: Create new email triage rule with AI assistance
    api_endpoint: /api/v1/rules
    arguments:
      - name: description
        type: string
        required: true
        description: Natural language rule description
    flags:
      - name: --no-ai
        description: Disable AI rule generation
      - name: --preview
        description: Show rule preview without saving
    output: Generated rule conditions and estimated matches
    
  - name: search
    description: Search emails using semantic similarity
    api_endpoint: /api/v1/emails/search
    arguments:
      - name: query
        type: string
        required: true
        description: Search query (natural language)
    flags:
      - name: --limit
        description: Number of results (default 10)
      - name: --account
        description: Restrict to specific email account
    output: Matching emails with similarity scores
    
  - name: sync
    description: Force email sync for all or specific accounts
    api_endpoint: /api/v1/sync
    flags:
      - name: --account
        description: Sync specific account only
      - name: --full
        description: Full resync (may take time)
    output: Sync status and processed email count
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **scenario-authenticator**: User account management and JWT token validation
- **mail-in-a-box**: Email server infrastructure with IMAP/SMTP access
- **Qdrant**: Vector database for semantic search capabilities
- **PostgreSQL**: Relational data storage for structured email metadata

### Downstream Enablement
- **Customer Support Scenarios**: AI triage for support ticket management
- **Executive Assistant**: Intelligent email management for busy professionals  
- **Sales Automation**: Lead qualification and follow-up automation
- **Research Aggregation**: Automatic collection and organization of research materials

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: ALL
    capability: Intelligent email processing and semantic search
    interface: API endpoints for email analysis and rule generation
    
  - scenario: customer-support-hub
    capability: Automated ticket triage and categorization
    interface: Email classification API
    
  - scenario: sales-pipeline-manager
    capability: Lead email analysis and qualification
    interface: Priority scoring and contact extraction
    
consumes_from:
  - scenario: scenario-authenticator
    capability: Multi-tenant user authentication
    fallback: Single-user mode with basic auth
    
  - scenario: notification-hub
    capability: User notifications for urgent emails
    fallback: Basic email notifications only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Gmail meets Superhuman - clean, fast, intelligent
  
  visual_style:
    color_scheme: light with dark mode support
    typography: modern sans-serif, excellent readability
    layout: three-panel design (accounts, email list, detail view)
    animations: subtle transitions, loading states for AI processing
  
  personality:
    tone: professional, efficient, intelligent
    mood: calm, organized, empowering
    target_feeling: Users should feel in control of their email chaos

style_references:
  professional:
    - "Superhuman email client - fast, keyboard-driven, AI-powered"
    - "Linear issue tracker - clean, modern, focused on productivity"
    - "Notion interface - friendly but professional, great information density"
```

### Target Audience Alignment
- **Primary Users**: Business professionals, executives, customer support teams
- **User Expectations**: Fast, reliable, intelligent automation that saves time
- **Accessibility**: Full keyboard navigation, screen reader support
- **Responsive Design**: Desktop-first but mobile-friendly for quick triage

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transform email from time-sink to intelligent assistant
- **Revenue Potential**: $20K - $50K per enterprise deployment
- **Cost Savings**: 2-4 hours/day saved on email management per user
- **Market Differentiator**: AI-powered rules that understand context, not just keywords

### Technical Value
- **Reusability Score**: 9/10 - Email processing useful across many scenarios
- **Complexity Reduction**: Natural language rules instead of complex filters
- **Innovation Enablement**: Semantic search enables new types of email automation

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic email account connection and sync
- AI-powered rule generation with Ollama
- Semantic search with Qdrant embeddings
- Multi-tenant architecture with scenario-authenticator
- Web dashboard for email triage

### Version 2.0 (Planned)
- Advanced AI features (sentiment, urgency detection)
- Integration with calendar and CRM systems
- Team collaboration on shared email accounts
- Mobile app for iOS/Android
- Advanced analytics and productivity insights

### Long-term Vision
- Predictive email management (AI suggests actions before you read)
- Voice-controlled email triage
- Integration with all major email providers (not just mail-in-a-box)
- Enterprise features (SSO, admin controls, compliance)

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Email provider connectivity issues | Medium | High | Connection pooling, retry logic, offline mode |
| AI rule generation accuracy | Medium | Medium | Human validation, confidence scoring, easy editing |
| Vector search performance degradation | Low | Medium | Qdrant indexing optimization, query caching |
| Multi-tenant data isolation breach | Low | Critical | Comprehensive security audit, row-level security |

### Operational Risks
- **Email Privacy**: End-to-end encryption for sensitive data, clear privacy policy
- **Scalability**: Horizontal scaling with Redis queues and database sharding
- **Compliance**: GDPR/CCPA compliance with data deletion and export capabilities
- **Performance**: Email processing queues to handle high-volume scenarios

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: email-triage

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/email-triage
    - cli/install.sh
    - initialization/postgres/schema.sql
    - ui/src/index.html
    - ui/src/main.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/postgres
    - ui

resources:
  required: [postgres, qdrant, scenario-authenticator, mail-in-a-box]
  optional: [notification-hub, ollama, redis]
  health_timeout: 90

tests:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "User can connect email account"
    type: http
    service: api
    endpoint: /api/v1/accounts
    method: POST
    body:
      email_address: test@example.com
      imap_server: localhost
      imap_port: 143
      smtp_server: localhost
      smtp_port: 25
      password: testpass
    expect:
      status: 201
      body:
        success: true
        
  - name: "AI rule generation works"
    type: http
    service: api
    endpoint: /api/v1/rules
    method: POST
    body:
      description: "Archive all newsletters and promotional emails"
      use_ai_generation: true
    expect:
      status: 201
      body:
        success: true
        ai_confidence: "> 0.7"
        
  - name: "Semantic email search works"
    type: http
    service: api
    endpoint: /api/v1/emails/search?q=urgent meeting tomorrow
    method: GET
    expect:
      status: 200
      body:
        results: "array"
        
  - name: "CLI status command works"
    type: exec
    command: ./cli/email-triage status --json
    expect:
      exit_code: 0
      output_contains: ["healthy", "accounts"]
```

## 📝 Implementation Notes

### Design Decisions
**Direct API Integration**: Chose direct Go API over n8n workflows
- Alternative considered: n8n-based email processing workflows
- Decision driver: User requirement to avoid n8n due to performance concerns
- Trade-offs: More code to write but better performance and control

**Qdrant for Semantic Search**: Vector database for email embeddings
- Alternative considered: PostgreSQL with pgvector extension
- Decision driver: Qdrant's superior performance for high-dimensional vectors
- Trade-offs: Additional dependency but significantly better search quality

### Known Limitations
- **Single Email Provider**: Initially only supports mail-in-a-box
  - Workaround: Users must use mail-in-a-box as email server
  - Future fix: Version 2.0 will add Gmail, Outlook, other IMAP providers

- **AI Rule Accuracy**: Generated rules may need human refinement
  - Workaround: Confidence scoring and easy rule editing interface
  - Future fix: Improved AI models and user feedback training

### Security Considerations
- **Email Credentials**: Encrypted at rest, separate key management
- **Multi-tenant Isolation**: Row-level security in PostgreSQL
- **API Authentication**: JWT tokens from scenario-authenticator required
- **Data Privacy**: Users can opt out of AI analysis for sensitive emails

## 🔗 References

### Documentation
- README.md - Quick start and user guide
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference
- docs/ai-rules.md - AI rule generation guide

### Related PRDs
- scenario-authenticator/PRD.md - Authentication foundation
- notification-hub/PRD.md - Notification integration
- mail-in-a-box/README.md - Email server capabilities

### External Resources
- IMAP/SMTP Protocol Documentation
- Qdrant Vector Database Documentation
- OpenAI Embeddings API Reference
- Email Security Best Practices (RFC standards)

---

**Last Updated**: 2025-01-09
**Status**: Draft
**Owner**: AI Agent
**Review Cycle**: Weekly during initial development