# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Automated referral/affiliate program generation and implementation for Vrooli scenarios, transforming any scenario into a revenue-multiplying business with intelligent branding analysis, commission optimization, and standardized implementation patterns.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Each referral program created builds a knowledge base of optimal conversion rates, UI patterns, and marketing copy across different verticals. Future scenarios inherit increasingly sophisticated monetization strategies, creating compound business intelligence where every successful program makes ALL future programs more effective.

### Recursive Value
**What new scenarios become possible after this exists?**
- **Cross-scenario Affiliate Networks**: Scenarios can promote each other creating ecosystem-wide revenue amplification
- **Revenue Attribution Intelligence**: Advanced analytics that track conversion paths across the entire Vrooli ecosystem
- **Business Model Pattern Recognition**: AI that automatically suggests optimal pricing/commission structures based on scenario type
- **Automated A/B Testing Scenarios**: Programs that continuously optimize referral strategies across deployments
- **Franchise Generator**: Templates for turning successful scenarios into multi-tenant business opportunities

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Analyze local scenario files to extract branding (colors, fonts, logos) and pricing models
  - [ ] Generate referral program configuration with optimal commission structures
  - [ ] Create branded referral materials (landing pages, tracking codes, email templates)
  - [ ] Detect existing referral logic in scenarios to prevent duplication
  - [ ] Integration with scenario-authenticator for referral account management
  
- **Should Have (P1)**
  - [ ] Spawn resource-claude-code agent to automatically implement referral logic in scenarios
  - [ ] Web crawling mode for deployed scenarios (GitHub repos, live sites)
  - [ ] Cross-scenario referral network setup and management
  - [ ] Analytics dashboard for tracking referral performance
  - [ ] Fraud detection and commission validation
  
- **Nice to Have (P2)**
  - [ ] A/B testing framework for referral landing pages
  - [ ] Competitive intelligence gathering for optimal commission rates
  - [ ] Multi-tier referral programs (referrer gets referrers)
  - [ ] Advanced attribution modeling across marketing channels
  - [ ] Automated tax compliance documentation generation

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Branding Analysis Time | < 30s for local scenarios | File system analysis |
| Program Generation Time | < 2 minutes end-to-end | Full workflow timing |
| Implementation Success Rate | > 95% for standard patterns | Automated validation |
| Revenue Attribution Accuracy | > 99% for tracked conversions | Database audit |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with scenario-authenticator and resource-claude-code
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can generate referral programs for other scenarios via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: scenario-authenticator
    purpose: User account management and referral profile creation
    integration_pattern: API calls for account creation and authentication
    access_method: API endpoints
    
  - resource_name: postgres
    purpose: Referral tracking, commission calculations, analytics storage
    integration_pattern: Direct database operations
    access_method: SQL queries via Go database/sql
    
  - resource_name: resource-claude-code
    purpose: Automated implementation of referral logic in target scenarios
    integration_pattern: CLI invocation with precise instructions
    access_method: CLI command spawning
    
optional:
  - resource_name: browserless
    purpose: Web crawling for deployed scenario analysis and screenshot generation
    fallback: Manual input mode for branding/pricing information
    access_method: CLI commands via resource-browserless
    
  - resource_name: qdrant
    purpose: Semantic search of optimal referral patterns and marketing copy
    fallback: Rule-based pattern matching
    access_method: Vector database queries
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_resource_cli:        # FIRST: Use resource CLI commands
    - command: scenario-authenticator create-referral-profile
      purpose: Create referral user accounts
    - command: resource-claude-code implement-referral-pattern
      purpose: Automated code generation
    - command: resource-browserless screenshot
      purpose: Validation screenshots
  
  2_direct_api:          # SECOND: Direct API for performance
    - justification: High-frequency referral tracking requires direct DB access
      endpoint: Direct PostgreSQL connections for analytics
```

### Data Models
```yaml
primary_entities:
  - name: ReferralProgram
    storage: postgres
    schema: |
      {
        id: UUID,
        scenario_name: string,
        commission_rate: decimal,
        tracking_code: string,
        landing_page_url: string,
        branding_config: jsonb,
        created_at: timestamp,
        updated_at: timestamp
      }
    relationships: Links to ReferralLink and Commission entities
    
  - name: ReferralLink
    storage: postgres
    schema: |
      {
        id: UUID,
        program_id: UUID,
        referrer_id: UUID,
        tracking_code: string,
        clicks: integer,
        conversions: integer,
        total_commission: decimal,
        created_at: timestamp
      }
    relationships: Belongs to ReferralProgram, owned by User
    
  - name: Commission
    storage: postgres
    schema: |
      {
        id: UUID,
        link_id: UUID,
        customer_id: UUID,
        amount: decimal,
        status: enum(pending, paid, disputed),
        transaction_date: timestamp,
        paid_date: timestamp
      }
    relationships: Links to ReferralLink and Customer
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/referral/analyze
    purpose: Analyze scenario for branding/pricing information
    input_schema: |
      {
        scenario_path: string (local) OR url: string (deployed),
        mode: enum(local, deployed)
      }
    output_schema: |
      {
        branding: {
          colors: {primary, secondary, accent},
          fonts: [font_families],
          logo_path: string
        },
        pricing: {
          model: enum(subscription, one-time, freemium),
          tiers: [pricing_objects]
        }
      }
    sla:
      response_time: 30000ms
      availability: 99%
      
  - method: POST
    path: /api/v1/referral/generate
    purpose: Generate complete referral program
    input_schema: |
      {
        scenario_info: analyzed_data,
        commission_rate: decimal (optional),
        custom_branding: object (optional)
      }
    output_schema: |
      {
        program_id: UUID,
        tracking_code: string,
        landing_page_html: string,
        email_templates: [template_objects],
        analytics_dashboard_url: string
      }
    sla:
      response_time: 120000ms
      availability: 99%
      
  - method: POST
    path: /api/v1/referral/implement
    purpose: Automatically implement referral logic in target scenario
    input_schema: |
      {
        program_id: UUID,
        scenario_path: string,
        auto_mode: boolean
      }
    output_schema: |
      {
        implementation_status: enum(success, partial, failed),
        files_modified: [file_paths],
        validation_results: object
      }
    sla:
      response_time: 300000ms
      availability: 95%
```

### Event Interface
```yaml
published_events:
  - name: referral.program.created
    payload: {program_id, scenario_name, commission_rate}
    subscribers: [analytics-dashboard, billing-system]
    
  - name: referral.conversion.completed
    payload: {link_id, customer_id, amount, timestamp}
    subscribers: [commission-calculator, notification-system]
    
consumed_events:
  - name: scenario_authenticator.user.created
    action: Check if user creation was from referral link and award commission
    
  - name: billing.payment.completed
    action: Calculate and process referral commissions
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: referral-program-generator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show referral program operational status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: analyze
    description: Analyze scenario for referral program generation
    api_endpoint: /api/v1/referral/analyze
    arguments:
      - name: path_or_url
        type: string
        required: true
        description: Local scenario path or deployed URL
    flags:
      - name: --mode
        description: Analysis mode (local|deployed)
      - name: --output
        description: Output format (json|summary)
    output: Branding and pricing analysis results
    
  - name: generate
    description: Generate referral program for scenario
    api_endpoint: /api/v1/referral/generate
    arguments:
      - name: scenario_path
        type: string
        required: true
        description: Path to scenario directory
    flags:
      - name: --commission-rate
        description: Override default commission rate (decimal)
      - name: --preview
        description: Preview without creating program
    output: Generated referral program configuration
    
  - name: implement
    description: Implement referral logic in target scenario
    api_endpoint: /api/v1/referral/implement
    arguments:
      - name: program_id
        type: string
        required: true
        description: ID of generated referral program
    flags:
      - name: --auto
        description: Fully automated implementation
      - name: --preview
        description: Show planned changes without applying
    output: Implementation status and file changes
    
  - name: list
    description: List all referral programs
    api_endpoint: /api/v1/referral/programs
    flags:
      - name: --scenario
        description: Filter by scenario name
    output: List of referral programs with stats
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **scenario-authenticator**: Required for user account management and referral profile creation
- **postgres**: Essential for referral tracking, commission calculations, and analytics storage
- **resource-claude-code**: Needed for automated implementation of referral logic in scenarios

### Downstream Enablement
**What future capabilities does this unlock?**
- **Cross-Scenario Marketing Networks**: Scenarios can promote each other creating ecosystem revenue
- **Business Intelligence Platform**: Advanced analytics across all Vrooli deployments
- **Automated Revenue Optimization**: AI-driven commission and pricing adjustments
- **Franchise/Multi-tenant Scenarios**: Template for scaling successful scenarios

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: saas-billing-hub
    capability: Referral commission integration with payment processing
    interface: API
    
  - scenario: analytics-dashboard
    capability: Referral performance metrics and attribution data
    interface: Event streaming
    
  - scenario: email-campaign-manager
    capability: Automated referral marketing email sequences
    interface: API
    
consumes_from:
  - scenario: scenario-authenticator
    capability: User account management and authentication
    fallback: Basic tracking without full user profiles
    
  - scenario: seo-optimizer
    capability: Landing page optimization for better conversion rates
    fallback: Basic HTML templates without SEO optimization
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Modern SaaS dashboard with business intelligence focus
  
  visual_style:
    color_scheme: dynamic (adapts to analyzed scenario branding)
    typography: modern (clean, business-focused fonts)
    layout: dashboard (multi-panel analytics layout)
    animations: subtle (smooth transitions, loading states)
  
  personality:
    tone: professional (authoritative but approachable)
    mood: focused (business-oriented, results-driven)
    target_feeling: confidence in revenue generation capabilities

style_references:
  professional: 
    - product-manager: "Modern SaaS dashboard aesthetic with revenue focus"
    - analytics-dashboard: "Clean metrics presentation and business intelligence"
```

### Target Audience Alignment
- **Primary Users**: Entrepreneurs, developers deploying scenarios as businesses
- **User Expectations**: Professional, trustworthy interface that inspires confidence in revenue generation
- **Accessibility**: WCAG 2.1 AA compliance for business tool accessibility
- **Responsive Design**: Desktop-first (primary use), with mobile-friendly analytics viewing

### Brand Consistency Rules
- **Scenario Identity**: Professional revenue-generation tool with enterprise feel
- **Vrooli Integration**: Seamlessly integrates with ecosystem while maintaining business focus
- **Professional Design**: Business/enterprise tool → Professional design with revenue-focused UX

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transforms any Vrooli scenario into a revenue-generating business with referral programs
- **Revenue Potential**: $15K - $75K per deployment (amplifies existing scenario revenue 2-5x)
- **Cost Savings**: Eliminates manual referral program setup (typically $5K-15K in development costs)
- **Market Differentiator**: Only platform that auto-generates referral programs with AI-driven optimization

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from referral program capability
- **Complexity Reduction**: Reduces referral program setup from weeks to minutes
- **Innovation Enablement**: Creates compound revenue growth across entire Vrooli ecosystem

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core branding analysis and program generation
- Basic referral tracking and commission calculation
- Integration with scenario-authenticator and resource-claude-code

### Version 2.0 (Planned)
- Cross-scenario affiliate networks and revenue sharing
- Advanced A/B testing and conversion optimization
- Competitive intelligence and market analysis integration

### Long-term Vision
- AI-driven business model optimization across all scenarios
- Automated franchise/licensing program generation
- Revenue attribution across complex multi-scenario customer journeys

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
    - pricing_tiers: [Basic: $29/mo, Pro: $99/mo, Enterprise: $299/mo]
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: referral-program-generator
    category: monetization
    capabilities: [referral-generation, affiliate-management, revenue-optimization]
    interfaces:
      - api: /api/v1/referral
      - cli: referral-program-generator
      - events: referral.*
      
  metadata:
    description: "Automated referral program generation and implementation for scenarios"
    keywords: [referral, affiliate, monetization, revenue, business-intelligence]
    dependencies: [scenario-authenticator, postgres]
    enhances: [ALL scenarios - adds referral capability]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| scenario-authenticator unavailability | Low | High | Graceful degradation to basic tracking |
| Commission calculation errors | Medium | High | Automated validation and audit trails |
| Branding analysis failures | Medium | Medium | Manual fallback input mode |
| Claude Code integration failures | Medium | High | Preview mode and rollback capabilities |

### Operational Risks
- **Implementation Consistency**: Standardized referral patterns with validation
- **Revenue Attribution**: Double-entry accounting with audit trails
- **Fraud Prevention**: Rate limiting, IP tracking, conversion validation
- **Compliance**: Tax documentation and affiliate marketing law compliance

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: referral-program-generator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/referral-program-generator
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - scripts
    - ui
    - test

resources:
  required: [scenario-authenticator, postgres, resource-claude-code]
  optional: [browserless, qdrant]
  health_timeout: 60

tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Branding analysis endpoint works"
    type: http
    service: api
    endpoint: /api/v1/referral/analyze
    method: POST
    body:
      scenario_path: "../test-scenario-1"
      mode: "local"
    expect:
      status: 200
      body:
        branding: {}
        
  - name: "CLI analyze command executes"
    type: exec
    command: ./cli/referral-program-generator analyze ../test-scenario-1 --mode local
    expect:
      exit_code: 0
      output_contains: ["branding", "pricing"]
      
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('referral_programs', 'referral_links', 'commissions')"
    expect:
      rows: 
        - count: 3
```

## 📝 Implementation Notes

### Design Decisions
**Standardized Referral Pattern**: Chosen consistent file structure and API patterns over flexible customization
- Alternative considered: Fully customizable referral implementations per scenario
- Decision driver: Maintainability and cross-scenario compatibility
- Trade-offs: Less flexibility for better consistency and reliability

**Direct Claude Code Integration**: Using resource-claude-code for automated implementation rather than templates
- Alternative considered: Static template-based code generation
- Decision driver: Adaptive implementation that handles scenario variations
- Trade-offs: Dependency on Claude Code availability for better quality implementations

### Known Limitations
- **Initial Pattern Support**: V1 only supports standard web-app scenarios with REST APIs
  - Workaround: Manual implementation mode for non-standard scenarios
  - Future fix: Extended pattern recognition in v2.0

### Security Considerations
- **Commission Tracking**: All commission calculations use double-entry accounting with audit trails
- **Fraud Prevention**: Rate limiting, IP tracking, and conversion validation
- **Data Protection**: PII encryption for referral user data

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start guide
- docs/api.md - Complete API specification with examples
- docs/cli.md - CLI documentation with usage examples
- docs/architecture.md - Technical deep-dive and integration patterns

### Related PRDs
- scenario-authenticator/PRD.md - User authentication and profile management
- saas-billing-hub/PRD.md - Payment processing and billing integration

### External Resources
- [Affiliate Marketing Legal Guidelines](https://www.ftc.gov/tips-advice/business-center/guidance/ftcs-endorsement-guides-what-people-are-asking)
- [Double-Entry Accounting Principles](https://en.wikipedia.org/wiki/Double-entry_bookkeeping)
- [Referral Program Best Practices](https://blog.referralsaasquatch.com/referral-program-best-practices)

---

**Last Updated**: 2025-09-06  
**Status**: Draft  
**Owner**: Claude Code Agent  
**Review Cycle**: Weekly during development, monthly after deployment