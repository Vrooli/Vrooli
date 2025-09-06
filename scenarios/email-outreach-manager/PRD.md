# Product Requirements Document (PRD) - Email Outreach Manager

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Email Outreach Manager adds **AI-powered personalized email campaign generation and execution** as a permanent capability. It transforms raw contact information and campaign goals into beautifully designed, personalized email templates with intelligent drip campaign automation. This capability leverages contact intelligence to create hyper-personalized outreach that scales with relationship awareness.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Contact-Aware Communications**: All future scenarios gain access to sophisticated email outreach with relationship context
- **Campaign Intelligence**: Templates and personalization strategies become reusable across scenarios
- **Social Graph Leverage**: Uses contact-book's relationship intelligence to optimize send timing and personalization depth
- **Template Evolution**: AI-generated templates improve over time based on engagement metrics and response patterns

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Sales Pipeline Manager**: Automated lead nurturing with intelligent follow-up sequences based on engagement
2. **Event Marketing Hub**: Conference/webinar promotion with attendee relationship-aware messaging
3. **Newsletter Intelligence**: Content distribution optimized by recipient preferences and relationship strength
4. **Customer Success Automation**: Proactive relationship maintenance with personalized check-ins
5. **Investor Relations Manager**: Fundraising outreach with relationship-mapped investor communications

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Generate beautiful email templates from campaign briefs and documents using Ollama
  - [ ] Integrate with contact-book to enrich recipient data (names, pronouns, preferences)
  - [ ] Send emails via mail-in-a-box with personalization tiers (full/partial/template-only)
  - [ ] Visual warning indicators for non-personalized emails
  - [ ] Support manual recipient data fallback when contact-book data unavailable
  - [ ] Campaign management UI for template creation and recipient list management
  - [ ] Basic drip campaign sequencing with time-based triggers
  
- **Should Have (P1)**
  - [ ] A/B testing for email templates with performance analytics
  - [ ] Open/click tracking and engagement analytics dashboard
  - [ ] Campaign performance optimization suggestions via AI analysis
  - [ ] Template library with categorization and reuse capabilities
  - [ ] Bulk recipient import (CSV/JSON) with automatic contact-book synchronization
  - [ ] Email preview with multiple device/client rendering
  
- **Nice to Have (P2)**
  - [ ] Dynamic content insertion based on recipient interests from contact-book
  - [ ] Send time optimization using recipient timezone and engagement patterns
  - [ ] Integration with personal-relationship-manager for relationship-aware follow-ups
  - [ ] Advanced drip campaign branching based on recipient engagement

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Template Generation Time | < 30s for complex templates | API timing |
| Email Send Rate | 100 emails/minute | Mail-in-a-box throughput |
| Contact Enrichment | < 2s per recipient lookup | Contact-book API response |
| UI Responsiveness | < 200ms for all interactions | Frontend monitoring |
| Campaign Setup Time | < 5 minutes end-to-end | User workflow tracking |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with contact-book and mail-in-a-box resources
- [ ] Performance targets met under load (1000+ recipient campaigns)
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI for automated outreach

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: ollama
    purpose: AI template generation and personalization
    integration_pattern: CLI command
    access_method: resource-ollama generate

  - resource_name: mail-in-a-box
    purpose: Email delivery infrastructure
    integration_pattern: CLI command
    access_method: resource-mail-in-a-box add-account, send-email

  - resource_name: postgres
    purpose: Campaign data, templates, and analytics storage
    integration_pattern: Database connection
    access_method: Direct SQL via Go database/sql

optional:
  - resource_name: contact-book
    purpose: Recipient intelligence and personalization data
    fallback: Manual recipient data entry with warning indicators
    access_method: CLI - contact-book search, API - /api/v1/contacts
    
  - resource_name: redis
    purpose: Campaign sending queue and rate limiting
    fallback: In-memory queuing with reduced throughput
    access_method: Redis CLI/API
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: Consistent AI model access for template generation
  
  2_resource_cli:
    - command: resource-ollama generate
      purpose: AI template generation and content personalization
    - command: resource-mail-in-a-box send-email
      purpose: Email delivery with proper SMTP handling
    - command: contact-book search
      purpose: Recipient data enrichment
  
  3_direct_api:
    - justification: contact-book CLI lacks JSON batch operations needed for campaign recipient processing
      endpoint: /api/v1/contacts (search and bulk lookup)
```

### Data Models
```yaml
primary_entities:
  - name: Campaign
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        description: string,
        template_id: UUID,
        status: enum(draft, scheduled, sending, completed, paused),
        created_at: timestamp,
        send_schedule: timestamp,
        total_recipients: integer,
        sent_count: integer,
        analytics: jsonb
      }
    relationships: Has many EmailRecipients, belongs to Template

  - name: Template
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        subject: string,
        html_content: text,
        text_content: text,
        ai_generated_from: text,
        personalization_fields: string[],
        style_category: enum(professional, creative, casual),
        created_at: timestamp
      }
    relationships: Has many Campaigns

  - name: EmailRecipient
    storage: postgres
    schema: |
      {
        id: UUID,
        campaign_id: UUID,
        email: string,
        name: string,
        pronouns: string,
        contact_book_id: UUID nullable,
        personalization_data: jsonb,
        personalization_level: enum(full, partial, template_only),
        send_status: enum(pending, sent, failed, bounced),
        sent_at: timestamp nullable,
        opened_at: timestamp nullable,
        clicked_at: timestamp nullable
      }
    relationships: Belongs to Campaign, optionally linked to contact-book Person

  - name: DripSequence
    storage: postgres
    schema: |
      {
        id: UUID,
        campaign_id: UUID,
        step_number: integer,
        delay_days: integer,
        template_id: UUID,
        condition_type: enum(time_based, engagement_based),
        condition_criteria: jsonb
      }
    relationships: Belongs to Campaign, belongs to Template
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/campaigns
    purpose: Create new email campaign
    input_schema: |
      {
        name: string,
        description: string,
        template_requirements: {
          purpose: string,
          tone: enum(professional, friendly, casual),
          documents: string[] // URLs or content
        },
        recipients: [
          {
            email: string,
            name?: string,
            pronouns?: string,
            custom_data?: object
          }
        ],
        drip_sequence?: {
          enabled: boolean,
          steps: [{delay_days: integer, template_requirements: object}]
        }
      }
    output_schema: |
      {
        campaign_id: UUID,
        template_id: UUID,
        recipient_count: integer,
        personalization_stats: {
          full: integer,
          partial: integer,
          template_only: integer
        },
        estimated_send_time: string
      }
    sla:
      response_time: 30000ms
      availability: 99%

  - method: POST
    path: /api/v1/campaigns/{id}/send
    purpose: Execute campaign sending
    input_schema: |
      {
        schedule_time?: timestamp,
        test_mode?: boolean,
        batch_size?: integer
      }
    output_schema: |
      {
        send_job_id: UUID,
        status: string,
        estimated_completion: timestamp
      }
    sla:
      response_time: 1000ms
      availability: 99%

  - method: GET
    path: /api/v1/campaigns/{id}/analytics
    purpose: Get campaign performance metrics
    output_schema: |
      {
        campaign_id: UUID,
        metrics: {
          sent: integer,
          opened: integer,
          clicked: integer,
          bounced: integer,
          open_rate: float,
          click_rate: float
        },
        recipient_breakdown: {
          full_personalization: {count: integer, open_rate: float},
          partial_personalization: {count: integer, open_rate: float},
          template_only: {count: integer, open_rate: float}
        }
      }
    sla:
      response_time: 500ms
      availability: 99%
```

### Event Interface
```yaml
published_events:
  - name: email-outreach.campaign.sent
    payload: {campaign_id: UUID, recipient_count: integer}
    subscribers: [analytics-dashboard, personal-relationship-manager]
    
  - name: email-outreach.template.generated
    payload: {template_id: UUID, generation_prompt: string}
    subscribers: [template-library, ai-learning-system]
    
consumed_events:
  - name: contact-book.person.updated
    action: Update recipient personalization data for active campaigns
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: email-outreach-manager
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show operational status and resource health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: create-campaign
    description: Create new email outreach campaign
    api_endpoint: /api/v1/campaigns
    arguments:
      - name: name
        type: string
        required: true
        description: Campaign name
      - name: purpose
        type: string
        required: true
        description: Campaign purpose/goal
    flags:
      - name: --recipients-file
        description: CSV/JSON file with recipient list
      - name: --template-docs
        description: Documents to inform template generation
      - name: --tone
        description: Email tone (professional, friendly, casual)
      - name: --drip-sequence
        description: Enable drip campaign with specified steps
    output: Campaign ID and setup summary

  - name: send-campaign
    description: Execute campaign sending
    api_endpoint: /api/v1/campaigns/{id}/send
    arguments:
      - name: campaign_id
        type: string
        required: true
        description: Campaign to send
    flags:
      - name: --schedule
        description: Schedule send time (RFC3339 format)
      - name: --test-mode
        description: Send only to test recipients
      - name: --batch-size
        description: Number of emails to send per batch
    output: Send job status

  - name: analytics
    description: Show campaign performance metrics
    api_endpoint: /api/v1/campaigns/{id}/analytics
    arguments:
      - name: campaign_id
        type: string
        required: true
        description: Campaign to analyze
    flags:
      - name: --format
        description: Output format (table, json, csv)
      - name: --detailed
        description: Include per-recipient breakdown
    output: Performance metrics and insights

  - name: list-campaigns
    description: List all campaigns with status
    api_endpoint: /api/v1/campaigns
    flags:
      - name: --status
        description: Filter by campaign status
      - name: --limit
        description: Maximum campaigns to show
    output: Campaign list with status and metrics

  - name: preview-template
    description: Generate and preview email template
    arguments:
      - name: purpose
        type: string
        required: true
        description: Template purpose
    flags:
      - name: --docs
        description: Documents to inform template
      - name: --tone
        description: Email tone
      - name: --recipient-sample
        description: Sample recipient data for personalization preview
    output: HTML/text template preview
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Contact-Book Scenario**: Provides recipient intelligence and personalization data
- **Ollama Resource**: Enables AI template generation and content personalization
- **Mail-in-a-Box Resource**: Email delivery infrastructure with SMTP/domain management
- **PostgreSQL Resource**: Campaign and analytics data persistence

### Downstream Enablement
**What future capabilities does this unlock?**
- **Sales Automation Platform**: CRM-integrated lead nurturing with email sequences
- **Event Marketing Suite**: Conference/webinar promotion with relationship-aware messaging
- **Customer Success Manager**: Proactive relationship maintenance via automated email check-ins
- **Investor Outreach System**: Fundraising campaigns with relationship-mapped communications

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: sales-pipeline-manager
    capability: Automated lead nurturing email sequences
    interface: API
    
  - scenario: personal-relationship-manager
    capability: Relationship maintenance email automation
    interface: API + Events
    
  - scenario: event-marketing-hub
    capability: Attendee invitation and follow-up campaigns
    interface: API
    
consumes_from:
  - scenario: contact-book
    capability: Recipient data enrichment and relationship intelligence
    fallback: Manual data entry with warning indicators
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "ConvertKit meets Mailchimp - clean email marketing interface"
  
  visual_style:
    color_scheme: light
    typography: modern
    layout: dashboard
    animations: subtle
  
  personality:
    tone: friendly
    mood: focused
    target_feeling: "Confident and efficient in email marketing"

style_references:
  professional: 
    - app-debugger: "Clean dashboard layout with data tables"
    - product-manager: "Modern SaaS interface with action-oriented design"
```

### Target Audience Alignment
- **Primary Users**: Entrepreneurs, marketers, and relationship managers
- **User Expectations**: Professional email marketing interface with AI enhancement
- **Accessibility**: WCAG 2.1 AA compliance
- **Responsive Design**: Desktop-first with mobile dashboard access

### Brand Consistency Rules
- **Scenario Identity**: Professional email marketing tool with AI intelligence
- **Vrooli Integration**: Consistent with professional business scenarios
- **Professional vs Fun**: Business-focused design with subtle personality in AI-generated content suggestions

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transform time-intensive email campaign creation into AI-assisted automation
- **Revenue Potential**: $15K - $40K per deployment (email marketing SaaS pricing)
- **Cost Savings**: 80% reduction in campaign setup time, 60% improvement in personalization quality
- **Market Differentiator**: Contact-book integration enables relationship-aware email marketing

### Technical Value
- **Reusability Score**: High - any scenario needing email communication can leverage this
- **Complexity Reduction**: Complex email campaign management becomes template-driven automation
- **Innovation Enablement**: Enables relationship-intelligent communication across all scenarios

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core template generation and campaign management
- Contact-book integration for personalization
- Basic drip campaign functionality
- Email delivery via mail-in-a-box

### Version 2.0 (Planned)
- Advanced analytics with engagement prediction
- Dynamic content insertion based on recipient behavior
- Integration with social media for multi-channel campaigns
- AI-powered send time optimization

### Long-term Vision
- Self-optimizing campaigns that learn from recipient engagement
- Integration with all communication scenarios for unified relationship intelligence
- Predictive content generation based on relationship evolution

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata
    - All required initialization files
    - Deployment scripts (startup.sh, monitor.sh)
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose based
    - kubernetes: Helm chart generation
    - cloud: AWS/GCP/Azure templates
    
  revenue_model:
    - type: subscription
    - pricing_tiers: [starter: $29/mo, professional: $79/mo, enterprise: $199/mo]
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: email-outreach-manager
    category: automation
    capabilities: [email_campaigns, template_generation, contact_personalization, drip_sequences]
    interfaces:
      - api: /api/v1/campaigns
      - cli: email-outreach-manager
      - events: email-outreach.*
      
  metadata:
    description: "AI-powered personalized email campaigns with contact intelligence"
    keywords: [email, marketing, campaigns, personalization, automation]
    dependencies: [contact-book, ollama, mail-in-a-box]
    enhances: [sales-pipeline-manager, personal-relationship-manager]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Mail-in-a-box delivery issues | Medium | High | Fallback to SMTP providers, delivery status monitoring |
| Contact-book unavailability | Low | Medium | Graceful degradation to manual data entry |
| Ollama response delays | Medium | Medium | Request queuing, template caching |
| Large campaign performance | Low | High | Batch processing, Redis queuing |

### Operational Risks
- **Email Deliverability**: Implement proper SPF/DKIM/DMARC configuration validation
- **Privacy Compliance**: GDPR/CAN-SPAM compliance with opt-out management
- **Rate Limiting**: Respect email provider limits to maintain sender reputation
- **Template Quality**: AI-generated content review and approval workflows

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: email-outreach-manager

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/email-outreach-manager
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    - ui/index.html
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - ui

resources:
  required: [postgres, ollama, mail-in-a-box]
  optional: [contact-book, redis]
  health_timeout: 120

tests:
  - name: "API health check responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200

  - name: "Template generation works"
    type: http
    service: api
    endpoint: /api/v1/templates/generate
    method: POST
    body:
      purpose: "Product announcement"
      tone: "professional"
    expect:
      status: 201
      body:
        template_id: "*"

  - name: "CLI status command works"
    type: exec
    command: ./cli/email-outreach-manager status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]

  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('campaigns', 'templates', 'email_recipients')"
    expect:
      rows: 
        - count: 3
```

### Performance Validation
- [ ] Template generation completes within 30 seconds
- [ ] Campaign with 1000 recipients processes within 10 minutes
- [ ] Contact-book lookups average under 2 seconds
- [ ] UI loads and responds within 200ms
- [ ] Email send rate maintains 100 emails/minute

### Integration Validation
- [ ] Contact-book integration enriches recipient data correctly
- [ ] Mail-in-a-box successfully delivers test emails
- [ ] Ollama generates coherent email templates
- [ ] Redis queuing handles large campaigns gracefully
- [ ] CLI commands mirror API functionality completely

### Capability Verification
- [ ] Successfully creates personalized email campaigns
- [ ] Handles contact-book unavailability gracefully with fallbacks
- [ ] Template quality meets professional standards
- [ ] Campaign analytics provide actionable insights
- [ ] Drip sequences execute on schedule

## 📝 Implementation Notes

### Design Decisions
**Template Generation Strategy**: Use Ollama with structured prompts for consistent quality
- Alternative considered: External API services (less reliable, cost concerns)
- Decision driver: Local control, no external dependencies, cost efficiency
- Trade-offs: Local compute requirements vs external service reliability

**Personalization Tiers**: Three-level system (full, partial, template-only)
- Alternative considered: Binary personalized/not-personalized
- Decision driver: Transparency about personalization quality
- Trade-offs: UI complexity vs user awareness

### Known Limitations
- **Email Client Rendering**: HTML templates may render differently across email clients
  - Workaround: Use email-safe CSS and provide text alternatives
  - Future fix: Email client preview integration

- **Large Recipient Lists**: Performance degrades with 10,000+ recipients
  - Workaround: Batch processing with progress indicators
  - Future fix: Background job processing with Redis

### Security Considerations
- **Data Protection**: Recipient data encrypted at rest, minimal data retention
- **Access Control**: Campaign access restricted to creator or shared users
- **Audit Trail**: All campaign actions logged with user attribution

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/api.md - Complete API specification
- docs/cli.md - CLI command documentation
- docs/integration.md - Contact-book and mail-in-a-box integration guide

### Related PRDs
- contact-book/PRD.md - Recipient intelligence source
- personal-relationship-manager/PRD.md - Downstream integration target

### External Resources
- ConvertKit API Documentation - Email marketing platform patterns
- Mailchimp Template Guidelines - Email design best practices
- CAN-SPAM Act Compliance - Email marketing legal requirements

---

**Last Updated**: 2025-09-06  
**Status**: Draft  
**Owner**: AI Agent Claude  
**Review Cycle**: After initial implementation and first deployment