# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Visitor-intelligence provides real-time website visitor identification, behavioral tracking, and retention marketing automation. This capability transforms anonymous traffic into actionable visitor profiles across all Vrooli scenarios, creating a permanent intelligence layer that remembers and learns from every interaction.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Every Vrooli scenario gains visitor context awareness - agents can understand user behavior patterns, preferences, and intent without explicit input. This creates compound intelligence where scenarios can proactively adapt to user needs, trigger retention workflows, and share insights across the entire ecosystem. Agents become predictive rather than reactive.

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Personalized Content Engine** - Dynamically adapt scenario UIs based on visitor behavior patterns
2. **Retention Campaign Orchestrator** - Automated email/SMS campaigns triggered by specific visitor actions
3. **A/B Testing Platform** - Real-time experimentation with visitor segmentation
4. **Behavioral Analytics Dashboard** - Cross-scenario user journey analysis and optimization
5. **Intent Prediction System** - ML models to predict visitor actions before they happen

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] JavaScript tracking pixel with fingerprinting (40-60% identification rate)
  - [ ] Real-time visitor session tracking with PostgreSQL storage
  - [ ] RESTful API for visitor data retrieval and management
  - [ ] One-line integration script for any scenario
  - [ ] Privacy-compliant data collection with GDPR controls
  
- **Should Have (P1)**
  - [ ] Real-time dashboard showing live visitor activity
  - [ ] Visitor profile enrichment with behavioral insights
  - [ ] Event-driven retention trigger system
  - [ ] Cross-scenario visitor journey tracking
  - [ ] Audience segmentation and export tools
  
- **Nice to Have (P2)**
  - [ ] Predictive analytics using Ollama for intent inference
  - [ ] WebSocket real-time notifications
  - [ ] Advanced fingerprinting with Canvas/WebGL
  - [ ] Integration with external marketing tools

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 50ms for tracking pixel | API monitoring |
| Throughput | 1000+ events/second | Load testing |
| Accuracy | 40-60% visitor identification | Validation suite |
| Resource Usage | < 2GB memory, < 25% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with PostgreSQL and Redis
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Visitor profiles, events, and session data storage
    integration_pattern: Direct SQL via Go database/sql
    access_method: TCP connection to PostgreSQL instance
    
  - resource_name: redis
    purpose: Real-time session caching and rate limiting
    integration_pattern: Go Redis client
    access_method: TCP connection to Redis instance
    
optional:
  - resource_name: ollama
    purpose: Visitor intent analysis and behavioral insights
    fallback: Basic rules-based classification
    access_method: CLI commands via resource-ollama
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows: []  # No n8n workflows for performance reasons
  
  2_resource_cli:
    - command: resource-ollama analyze-visitor-behavior
      purpose: Generate behavioral insights from visitor data
  
  3_direct_api:
    - justification: High-performance visitor tracking requires direct database access
      endpoint: PostgreSQL and Redis connections for real-time data ingestion
```

### Data Models
```yaml
primary_entities:
  - name: Visitor
    storage: postgres
    schema: |
      {
        id: UUID,
        fingerprint: string,
        first_seen: timestamp,
        last_seen: timestamp,
        session_count: integer,
        identified: boolean,
        email: string (optional),
        user_agent: string,
        ip_address: string,
        timezone: string,
        language: string,
        screen_resolution: string,
        device_type: string,
        tags: string[]
      }
    relationships: HasMany VisitorEvents, HasMany VisitorSessions
    
  - name: VisitorEvent
    storage: postgres  
    schema: |
      {
        id: UUID,
        visitor_id: UUID,
        session_id: UUID,
        scenario: string,
        event_type: string,
        page_url: string,
        timestamp: timestamp,
        properties: jsonb
      }
    relationships: BelongsTo Visitor, BelongsTo VisitorSession
    
  - name: VisitorSession
    storage: postgres
    schema: |
      {
        id: UUID,
        visitor_id: UUID,
        scenario: string,
        start_time: timestamp,
        end_time: timestamp,
        page_views: integer,
        duration: integer,
        referrer: string,
        utm_source: string,
        utm_medium: string,
        utm_campaign: string
      }
    relationships: BelongsTo Visitor, HasMany VisitorEvents
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/visitor/track
    purpose: Accept tracking pixel data and events
    input_schema: |
      {
        fingerprint: string,
        event_type: string,
        page_url: string,
        scenario: string,
        properties: object
      }
    output_schema: |
      {
        success: boolean,
        visitor_id: string
      }
    sla:
      response_time: 50ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/visitor/{id}
    purpose: Retrieve visitor profile and events
    input_schema: |
      {
        id: string,
        include_events: boolean
      }
    output_schema: |
      {
        visitor: VisitorProfile,
        events: VisitorEvent[],
        sessions: VisitorSession[]
      }
    sla:
      response_time: 100ms
      availability: 99.5%
      
  - method: GET
    path: /api/v1/analytics/scenario/{name}
    purpose: Get scenario-specific visitor analytics
    input_schema: |
      {
        scenario: string,
        timeframe: string,
        metrics: string[]
      }
    output_schema: |
      {
        total_visitors: integer,
        unique_visitors: integer,
        page_views: integer,
        avg_session_duration: float,
        top_pages: PageStats[]
      }
    sla:
      response_time: 200ms
      availability: 99%
```

### Event Interface
```yaml
published_events:
  - name: visitor.identified
    payload: {visitor_id: string, scenario: string, email: string}
    subscribers: [retention-campaigns, personalization-engine]
    
  - name: visitor.abandoned
    payload: {visitor_id: string, scenario: string, page: string, duration: integer}
    subscribers: [retention-campaigns, analytics-dashboard]
    
consumed_events:
  - name: scenario.user_action
    action: Track user interaction and update visitor profile
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: visitor-intelligence
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show visitor tracking system status and resource health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: track
    description: Manually track a visitor event
    api_endpoint: /api/v1/visitor/track
    arguments:
      - name: event_type
        type: string
        required: true
        description: Type of event to track (pageview, click, etc.)
      - name: scenario
        type: string
        required: true
        description: Scenario name generating the event
    flags:
      - name: --properties
        description: JSON properties object for the event
    output: Visitor ID and tracking confirmation
    
  - name: profile
    description: Retrieve visitor profile and analytics
    api_endpoint: /api/v1/visitor/{id}
    arguments:
      - name: visitor_id
        type: string
        required: true
        description: Visitor ID to retrieve
    flags:
      - name: --events
        description: Include visitor events in output
      - name: --sessions  
        description: Include session data in output
    output: Complete visitor profile with history
    
  - name: analytics
    description: Get analytics for a specific scenario
    api_endpoint: /api/v1/analytics/scenario/{name}
    arguments:
      - name: scenario
        type: string
        required: true
        description: Scenario name for analytics
    flags:
      - name: --timeframe
        description: Time period (1d, 7d, 30d, 90d)
      - name: --format
        description: Output format (table, json, csv)
    output: Visitor analytics and metrics
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **PostgreSQL**: Required for persistent visitor data storage and complex queries
- **Redis**: Required for session caching and real-time data processing

### Downstream Enablement
**What future capabilities does this unlock?**
- **Retention Marketing**: Automated campaigns based on visitor behavior triggers
- **Personalization Engine**: Dynamic content adaptation based on visitor profiles
- **Predictive Analytics**: ML models for visitor intent and conversion probability

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: any_scenario
    capability: Visitor tracking integration via single script tag
    interface: JavaScript library
    
  - scenario: marketing_scenarios
    capability: Visitor profiles and behavioral data
    interface: API/CLI
    
consumes_from:
  - scenario: any_scenario
    capability: User interaction events
    fallback: Basic pageview tracking only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Modern analytics dashboards (Mixpanel, Amplitude)
  
  visual_style:
    color_scheme: dark
    typography: modern
    layout: dashboard
    animations: subtle
  
  personality:
    tone: technical
    mood: focused
    target_feeling: Empowered with data insights
```

### Target Audience Alignment
- **Primary Users**: Developers integrating tracking, marketers analyzing visitor data
- **User Expectations**: Clean, data-dense interface with powerful filtering and export
- **Accessibility**: WCAG AA compliance for dashboard components
- **Responsive Design**: Desktop-first for analytics, mobile-optimized tracking script

### Brand Consistency Rules
- **Scenario Identity**: Professional analytics tool with emphasis on privacy and performance
- **Vrooli Integration**: Seamlessly integrates with any Vrooli scenario
- **Professional Focus**: Function over form, but polished and trustworthy

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transform anonymous traffic into actionable customer intelligence
- **Revenue Potential**: $15K - $50K per deployment (similar to retention.com pricing)
- **Cost Savings**: Eliminate need for external visitor identification services
- **Market Differentiator**: Privacy-first, self-hosted alternative to commercial tools

### Technical Value
- **Reusability Score**: 10/10 - Every Vrooli scenario can leverage visitor intelligence
- **Complexity Reduction**: One-line integration replaces complex analytics setups
- **Innovation Enablement**: Foundation for AI-driven personalization and retention

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core visitor identification and tracking
- PostgreSQL/Redis integration
- Basic dashboard and API

### Version 2.0 (Planned)
- ML-powered visitor intent prediction
- Advanced segmentation and cohort analysis
- Real-time personalization triggers

### Long-term Vision
- Predictive visitor behavior modeling
- Cross-platform visitor tracking (mobile apps, desktop)
- AI-generated retention strategies

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - .vrooli/service.json with PostgreSQL and Redis requirements
    - initialization/storage/postgres/schema.sql
    - Tracking script distribution endpoint
    
  deployment_targets:
    - local: Docker Compose with database services
    - kubernetes: Helm chart with StatefulSets
    - cloud: Managed database integration
    
  revenue_model:
    - type: subscription
    - pricing_tiers: [starter, professional, enterprise]
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: visitor-intelligence
    category: analytics
    capabilities: [visitor-tracking, behavioral-analysis, retention-triggers]
    interfaces:
      - api: /api/v1/visitor/*
      - cli: visitor-intelligence
      - events: visitor.*
      
  metadata:
    description: Real-time visitor identification and behavioral tracking
    keywords: [analytics, tracking, retention, visitors, behavior]
    dependencies: [postgres, redis]
    enhances: [all scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| High tracking volume overwhelms DB | Medium | High | Redis caching, batch writes, connection pooling |
| Fingerprinting blocked by browsers | High | Medium | Graceful degradation to session-based tracking |
| GDPR compliance violations | Low | High | Built-in consent management, data purging |

### Operational Risks
- **Privacy Regulations**: Automatic data retention limits, consent management
- **Performance Impact**: Lightweight tracking script, async loading
- **Cross-browser Compatibility**: Extensive testing, progressive enhancement

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: visitor-intelligence

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/visitor-intelligence
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - ui/tracker.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - ui

resources:
  required: [postgres, redis]
  optional: [ollama]
  health_timeout: 60

tests:
  - name: "PostgreSQL is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Redis is accessible"
    type: tcp
    service: redis
    port: 6379
    expect:
      connected: true
      
  - name: "Visitor tracking API responds"
    type: http
    service: api
    endpoint: /api/v1/visitor/track
    method: POST
    body:
      fingerprint: test123
      event_type: pageview
      scenario: test
    expect:
      status: 201
      body:
        success: true
        
  - name: "CLI status command works"
    type: exec
    command: ./cli/visitor-intelligence status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
      
  - name: "Database schema initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('visitors', 'visitor_events', 'visitor_sessions')"
    expect:
      rows: 
        - count: 3
```

### Performance Validation
- [ ] Tracking pixel loads in <100ms
- [ ] API handles 1000+ events/second
- [ ] Database queries optimized with proper indexing
- [ ] Memory usage stable under sustained load

### Integration Validation
- [ ] One-line script integration works in sample scenarios
- [ ] Visitor profiles accurately track cross-scenario journeys
- [ ] Privacy controls function correctly
- [ ] Analytics dashboard shows real-time data

## 📝 Implementation Notes

### Design Decisions
**Tracking Architecture**: Direct database writes instead of n8n workflows for performance
- Alternative considered: Event queuing through n8n
- Decision driver: Need sub-100ms response times for tracking pixel
- Trade-offs: More complex Go code, but 10x performance improvement

**Fingerprinting Strategy**: Client-side JavaScript fingerprinting using open source libraries
- Alternative considered: Server-side IP+UserAgent hashing
- Decision driver: Better accuracy and persistence across sessions
- Trade-offs: Client-side code can be blocked, but provides richer identification

### Known Limitations
- **Browser Blocking**: Ad blockers and privacy extensions will block tracking
  - Workaround: Graceful degradation to server-side session tracking
  - Future fix: First-party domain hosting reduces blocking

- **Identification Rate**: 40-60% visitor identification vs commercial tools' 80%+
  - Workaround: Focus on behavior patterns rather than individual identification
  - Future fix: Enhanced fingerprinting techniques, ML correlation

### Security Considerations
- **Data Protection**: All PII encrypted at rest, visitor IDs are UUIDs not fingerprints
- **Access Control**: API requires authentication, dashboard has role-based access
- **Audit Trail**: All visitor data access logged with timestamps and sources

## 🔗 References

### Documentation
- README.md - Integration guide and overview
- docs/api.md - Complete API specification
- docs/privacy.md - GDPR compliance guide
- docs/integration.md - Scenario integration examples

### Related PRDs
- Future retention-campaign-orchestrator PRD
- Future personalization-engine PRD

### External Resources
- FingerprintJS documentation for identification techniques
- GDPR compliance guidelines for visitor tracking
- Web Analytics Association privacy best practices

---

**Last Updated**: 2025-09-06  
**Status**: Draft  
**Owner**: Claude Code AI Agent  
**Review Cycle**: Every implementation milestone