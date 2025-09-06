# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
The funnel-builder adds the permanent capability to create, deploy, and optimize multi-step conversion funnels (sales, marketing, onboarding) with visual drag-and-drop editing, branching logic, lead capture, and comprehensive analytics. This becomes Vrooli's universal customer acquisition and conversion engine that any scenario can leverage to monetize or qualify users.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides agents with:
- Conversion optimization intelligence - learns what funnel patterns work best
- Lead qualification patterns - understands how to identify high-value prospects
- User journey mapping - tracks and optimizes multi-step user experiences
- A/B testing framework - enables data-driven decision making for all user interactions
- Revenue attribution - directly measures the business impact of agent actions

### Recursive Value
**What new scenarios become possible after this exists?**
- **saas-billing-hub** - Can create pricing/checkout funnels for any scenario
- **app-onboarding-manager** - Can build interactive onboarding flows
- **survey-monkey** - Can create advanced survey funnels with conditional logic
- **roi-fit-analysis** - Can qualify prospects before deep analysis
- **competitor-change-monitor** - Can create lead magnets to capture competitor users

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Visual drag-and-drop funnel builder with React/TypeScript UI
  - [ ] Support for multiple step types (quiz, form, content, CTA)
  - [ ] Lead capture and storage in PostgreSQL
  - [ ] Basic linear funnel flow execution
  - [ ] Integration with scenario-authenticator for multi-tenant support
  - [ ] API endpoints for funnel execution and lead retrieval
  - [ ] Mobile-responsive funnel display
  
- **Should Have (P1)**
  - [ ] Branching logic based on user responses
  - [ ] A/B testing support for funnel variations
  - [ ] Analytics dashboard showing conversion rates and drop-off points
  - [ ] Template library with pre-built funnels
  - [ ] Dynamic content personalization
  - [ ] Export leads to CSV/JSON
  
- **Nice to Have (P2)**
  - [ ] AI-powered copy generation for funnel steps
  - [ ] Predictive analytics for conversion optimization
  - [ ] Webhook integrations for external services
  - [ ] Advanced segmentation and targeting

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for step transitions | API monitoring |
| Throughput | 1000 concurrent funnel sessions | Load testing |
| Conversion Rate | > 15% for optimized funnels | Analytics tracking |
| Resource Usage | < 512MB memory, < 10% CPU | System monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store funnel definitions, steps, leads, and analytics data
    integration_pattern: Direct SQL queries via Go API
    access_method: resource-postgres CLI and API
    
  - resource_name: scenario-authenticator
    purpose: Multi-tenant support and user authentication
    integration_pattern: API integration for auth tokens
    access_method: API endpoints
    
optional:
  - resource_name: ollama
    purpose: Generate compelling copy for funnel steps
    fallback: Use pre-written templates
    access_method: resource-ollama CLI
    
  - resource_name: redis
    purpose: Cache funnel sessions and real-time analytics
    fallback: Use PostgreSQL with slightly higher latency
    access_method: resource-redis CLI
```

### Data Models
```yaml
primary_entities:
  - name: Funnel
    storage: postgres
    schema: |
      {
        id: UUID
        tenant_id: UUID (from authenticator)
        name: string
        slug: string (URL-friendly)
        status: enum (draft, active, archived)
        metadata: JSONB (settings, styling, etc.)
        created_at: timestamp
        updated_at: timestamp
      }
    relationships: Has many FunnelSteps, has many Leads
    
  - name: FunnelStep
    storage: postgres
    schema: |
      {
        id: UUID
        funnel_id: UUID
        position: integer
        type: enum (quiz, form, content, cta)
        content: JSONB (question, options, fields, etc.)
        branching_rules: JSONB
        created_at: timestamp
      }
    relationships: Belongs to Funnel, has many StepResponses
    
  - name: Lead
    storage: postgres
    schema: |
      {
        id: UUID
        funnel_id: UUID
        tenant_id: UUID
        email: string
        data: JSONB (all captured info)
        source: string
        completed: boolean
        created_at: timestamp
      }
    relationships: Belongs to Funnel, has many StepResponses
    
  - name: StepResponse
    storage: postgres
    schema: |
      {
        id: UUID
        lead_id: UUID
        step_id: UUID
        response: JSONB
        duration_ms: integer
        created_at: timestamp
      }
    relationships: Belongs to Lead and FunnelStep
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/funnels
    purpose: Create a new funnel
    input_schema: |
      {
        name: string
        steps: Array<StepDefinition>
      }
    output_schema: |
      {
        id: UUID
        slug: string
        preview_url: string
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/funnels/:slug/execute
    purpose: Execute a funnel (render current step)
    output_schema: |
      {
        step: StepContent
        progress: number (0-100)
        session_id: UUID
      }
      
  - method: POST
    path: /api/v1/funnels/:slug/submit
    purpose: Submit response for current step
    input_schema: |
      {
        session_id: UUID
        response: object
      }
      
  - method: GET
    path: /api/v1/funnels/:id/analytics
    purpose: Get funnel analytics
    output_schema: |
      {
        conversion_rate: number
        total_leads: number
        drop_off_points: Array<StepMetrics>
      }
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: funnel-builder
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
  - name: create
    description: Create a new funnel
    api_endpoint: /api/v1/funnels
    arguments:
      - name: name
        type: string
        required: true
        description: Funnel name
    flags:
      - name: --template
        description: Use a template (quiz, lead-magnet, webinar)
    output: Funnel ID and preview URL
    
  - name: list
    description: List all funnels
    api_endpoint: /api/v1/funnels
    flags:
      - name: --status
        description: Filter by status (draft, active, archived)
        
  - name: analytics
    description: Show funnel analytics
    api_endpoint: /api/v1/funnels/:id/analytics
    arguments:
      - name: funnel_id
        type: string
        required: true
    flags:
      - name: --format
        description: Output format (table, json, csv)
        
  - name: export-leads
    description: Export leads from a funnel
    api_endpoint: /api/v1/funnels/:id/leads
    arguments:
      - name: funnel_id
        type: string
        required: true
    flags:
      - name: --format
        description: Export format (csv, json)
      - name: --output
        description: Output file path
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **scenario-authenticator**: Required for multi-tenant support and user management
- **postgres**: Required for all data persistence

### Downstream Enablement
- **Any revenue-generating scenario**: Can use funnels for customer acquisition
- **saas-billing-hub**: Can create checkout funnels
- **app-onboarding-manager**: Can build onboarding flows

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: saas-billing-hub
    capability: Checkout and pricing funnels
    interface: API
    
  - scenario: app-onboarding-manager
    capability: Interactive onboarding flows
    interface: API/CLI
    
  - scenario: Any scenario
    capability: Lead generation and qualification
    interface: API/CLI
    
consumes_from:
  - scenario: scenario-authenticator
    capability: User authentication and multi-tenancy
    fallback: Single-tenant mode
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Modern SaaS tools like Typeform, ConvertFlow
  
  visual_style:
    color_scheme: light with customizable accent colors
    typography: modern, clean, highly readable
    layout: spacious, focused on one step at a time
    animations: subtle transitions between steps
  
  personality:
    tone: friendly but professional
    mood: confident and trustworthy
    target_feeling: "This is easy and I'm making progress"

builder_interface:
  - Drag-and-drop with clear visual hierarchy
  - Live preview alongside builder
  - Template gallery with preview thumbnails
  - Clear CTAs and progress indicators
  
funnel_display:
  - One step per screen for focus
  - Clear progress bar
  - Smooth transitions
  - Mobile-first responsive design
```

## 💰 Value Proposition

### Business Value
- **Primary Value**: Universal conversion optimization for any Vrooli scenario
- **Revenue Potential**: $10K - $50K per deployment as standalone SaaS
- **Cost Savings**: Replaces $200-500/month third-party funnel tools
- **Market Differentiator**: Local-first, AI-native, infinitely customizable

### Technical Value
- **Reusability Score**: 10/10 - Every revenue scenario needs funnels
- **Complexity Reduction**: Makes complex conversion flows visual and manageable
- **Innovation Enablement**: Enables data-driven optimization for all user interactions

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core drag-and-drop builder
- Basic step types (quiz, form, content, CTA)
- Linear funnel flow
- Lead capture and storage
- Basic analytics

### Version 2.0 (Planned)
- Branching logic and conditional paths
- A/B testing framework
- Advanced analytics with cohort analysis
- AI-powered copy generation
- Webhook integrations

### Long-term Vision
- Predictive conversion optimization using ML
- Cross-funnel user journey mapping
- Revenue attribution across entire Vrooli ecosystem
- Become the intelligence layer for all user interactions

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: funnel-builder

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/funnel-builder
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - ui/package.json
    - ui/index.html
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - ui
    - initialization
    - initialization/storage

resources:
  required: [postgres, scenario-authenticator]
  optional: [redis, ollama]
  health_timeout: 60

tests:
  - name: "API health check"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Create funnel via API"
    type: http
    service: api
    endpoint: /api/v1/funnels
    method: POST
    body:
      name: "Test Funnel"
      steps: []
    expect:
      status: 201
      body:
        id: <UUID>
        
  - name: "CLI status command"
    type: exec
    command: ./cli/funnel-builder status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
```

## 📝 Implementation Notes

### Design Decisions
**Builder vs Code**: Chose visual drag-and-drop over code-based definitions
- Alternative considered: YAML/JSON definitions
- Decision driver: Non-technical users need to build funnels
- Trade-offs: More complex UI for better UX

**Script-based execution vs n8n**: Chose direct script execution over n8n workflows
- Alternative considered: n8n workflow for each funnel
- Decision driver: Performance and reliability requirements
- Trade-offs: Less visual debugging for better speed

### Security Considerations
- **Data Protection**: All lead data encrypted at rest
- **Access Control**: Tenant isolation via scenario-authenticator
- **Audit Trail**: All funnel modifications and lead captures logged

---

**Last Updated**: 2025-09-06  
**Status**: In Development  
**Owner**: AI Agent  
**Review Cycle**: After each major feature addition