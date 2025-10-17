# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
The SaaS Landing Manager provides automated landing page generation, deployment, and optimization for all SaaS scenarios in the Vrooli ecosystem. It automatically detects which scenarios are SaaS businesses, generates professional landing pages using AI-powered templates, deploys them via Claude Code agents, and continuously optimizes them through A/B testing and analytics.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides landing page intelligence that compounds across scenarios:
- **Marketing Intelligence**: Learns what landing page patterns convert best for different SaaS types
- **SEO Knowledge**: Builds understanding of search optimization across industries
- **A/B Testing Framework**: Enables data-driven optimization for all user-facing scenarios
- **Template Patterns**: Creates reusable conversion-optimized components
- **Agent Orchestration**: Establishes patterns for spawning specialized agents for complex tasks

### Recursive Value
**What new scenarios become possible after this exists?**
- **marketing-automation-suite** - Can create complete marketing funnels with landing pages
- **saas-business-generator** - Complete SaaS businesses from idea to deployed landing page
- **conversion-optimization-lab** - Advanced A/B testing and optimization platform
- **seo-intelligence-hub** - Comprehensive SEO optimization across all Vrooli scenarios
- **brand-consistent-deployer** - Automated brand application across all customer touchpoints

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Automatic detection of SaaS scenarios based on PRD analysis and service.json tags
  - [ ] Template system with base SaaS templates (B2B tool, B2C app, API service, marketplace)
  - [ ] A/B testing infrastructure with traffic routing and analytics collection
  - [ ] Claude Code agent orchestration for automated landing page deployment
  - [ ] Landing page file structure standardization across all scenarios
  - [ ] SEO optimization with meta tags, structured data, and performance optimization
  
- **Should Have (P1)**
  - [ ] AI-powered content generation for landing page copy using Ollama
  - [ ] Brand integration with brand-manager scenario for consistent visual identity
  - [ ] Analytics dashboard showing conversion rates and performance metrics
  - [ ] Template marketplace with community-contributed templates
  - [ ] Responsive design system with mobile-first templates
  - [ ] Performance monitoring and Core Web Vitals optimization
  
- **Nice to Have (P2)**
  - [ ] Multi-language landing page generation
  - [ ] Advanced funnel integration with funnel-builder scenario
  - [ ] Real-time collaborative editing for landing page customization
  - [ ] Integration with external analytics platforms (Google Analytics, Mixpanel)
  - [ ] Automated screenshot comparison for A/B test visualization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Scenario Detection Accuracy | > 95% for SaaS scenarios | Automated validation |
| Landing Page Generation Time | < 2 minutes per page | API timing |
| Page Load Speed | < 3s for 95% of pages | Core Web Vitals |
| Conversion Rate Improvement | > 20% with A/B testing | Analytics tracking |
| Template Deployment Success | > 98% success rate | Deployment logs |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Successfully detects and generates landing pages for test SaaS scenarios
- [ ] A/B testing infrastructure functional with traffic splitting
- [ ] Claude Code integration working end-to-end
- [ ] Templates are mobile-responsive and pass Core Web Vitals
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store landing page metadata, A/B test results, scenario registry, analytics
    integration_pattern: Direct SQL via Go database/sql
    access_method: SQL queries for data persistence
    
  - resource_name: claude-code
    purpose: Automated deployment of landing pages to target scenarios
    integration_pattern: Agent spawning via API calls
    access_method: CLI commands and API endpoints for agent management
    
optional:
  - resource_name: ollama
    purpose: AI-powered content generation for landing page copy
    fallback: Use predefined template content
    access_method: Direct CLI calls to resource-ollama
    
  - resource_name: browserless
    purpose: Screenshot generation for template previews and A/B test validation
    fallback: Text-based template previews
    access_method: resource-browserless CLI commands
    
  - resource_name: brand-manager
    purpose: Brand asset integration for consistent visual identity
    fallback: Generic styling without brand assets
    access_method: API calls to brand-manager endpoints
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_direct_cli_access:     # FIRST: Use resource CLI commands
    - command: resource-claude-code spawn
      purpose: Deploy landing pages to target scenarios
    - command: resource-ollama generate
      purpose: Generate landing page content
    - command: resource-browserless screenshot
      purpose: Generate template previews
  
  2_direct_api:            # SECOND: Direct API when CLI insufficient
    - justification: Complex agent orchestration requires API control
      endpoint: POST /api/v1/agents/spawn
    - justification: Real-time A/B test traffic routing
      endpoint: GET /api/v1/landing/:variant

shared_workflow_criteria:
  - No n8n workflows used per architectural decision
  - All orchestration handled directly in Go API
  - CLI commands wrap API functionality
```

### Data Models
```yaml
primary_entities:
  - name: SaaSScenario
    storage: postgres
    schema: |
      {
        id: UUID
        scenario_name: string
        display_name: string
        description: string
        saas_type: enum(b2b_tool, b2c_app, api_service, marketplace)
        industry: string
        revenue_potential: string
        has_landing_page: boolean
        landing_page_url: string
        last_scan: timestamp
        confidence_score: number
        metadata: JSONB
      }
    relationships: Has many LandingPages and ABTests
    
  - name: LandingPage
    storage: postgres
    schema: |
      {
        id: UUID
        scenario_id: UUID
        template_id: UUID
        variant: string (a, b, control)
        title: string
        description: string
        content: JSONB
        seo_metadata: JSONB
        performance_metrics: JSONB
        status: enum(draft, active, archived)
        created_at: timestamp
        updated_at: timestamp
      }
    relationships: Belongs to SaaSScenario, has many ABTestResults
    
  - name: Template
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        category: enum(base, industry, component)
        saas_type: string
        industry: string
        html_content: text
        css_content: text
        js_content: text
        config_schema: JSONB
        preview_url: string
        usage_count: number
        rating: number
        created_at: timestamp
      }
    relationships: Used by many LandingPages
    
  - name: ABTestResult
    storage: postgres
    schema: |
      {
        id: UUID
        landing_page_id: UUID
        variant: string
        metric_name: string
        metric_value: number
        timestamp: datetime
        session_id: string
        user_agent: string
      }
    relationships: Belongs to LandingPage
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/scenarios/scan
    purpose: Scan and detect SaaS scenarios for landing page opportunities
    input_schema: |
      {
        force_rescan: boolean
        scenario_filter: string (optional)
      }
    output_schema: |
      {
        total_scenarios: number
        saas_scenarios: number
        newly_detected: number
        scenarios: [SaaSScenario]
      }
    sla:
      response_time: 10000ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/landing-pages/generate
    purpose: Generate landing page for a SaaS scenario
    input_schema: |
      {
        scenario_id: UUID
        template_id: UUID (optional)
        custom_content: object (optional)
        enable_ab_testing: boolean
      }
    output_schema: |
      {
        landing_page_id: UUID
        preview_url: string
        deployment_status: string
        ab_test_variants: [string]
      }
      
  - method: POST
    path: /api/v1/landing-pages/:id/deploy
    purpose: Deploy landing page to target scenario using Claude Code
    input_schema: |
      {
        target_scenario: string
        deployment_method: enum(direct, claude_agent)
        backup_existing: boolean
      }
    output_schema: |
      {
        deployment_id: UUID
        agent_session_id: string (if using claude_agent)
        status: string
        estimated_completion: datetime
      }
      
  - method: GET
    path: /api/v1/analytics/dashboard
    purpose: Get landing page performance analytics
    output_schema: |
      {
        total_pages: number
        active_ab_tests: number
        average_conversion_rate: number
        top_performing_templates: [Template]
        recent_deployments: [DeploymentStatus]
      }
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: saas-landing-manager
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show operational status and service health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: scan
    description: Scan scenarios and detect SaaS opportunities
    api_endpoint: /api/v1/scenarios/scan
    arguments:
      - name: scenario
        type: string
        required: false
        description: Specific scenario to scan (scans all if omitted)
    flags:
      - name: --force
        description: Force rescan even if recently scanned
      - name: --dry-run
        description: Show what would be detected without saving
    output: List of detected SaaS scenarios with confidence scores
    
  - name: generate
    description: Generate landing page for a SaaS scenario
    api_endpoint: /api/v1/landing-pages/generate
    arguments:
      - name: scenario
        type: string
        required: true
        description: Scenario name to generate landing page for
    flags:
      - name: --template
        description: Template ID or name to use
      - name: --ab-test
        description: Enable A/B testing with variants
      - name: --preview-only
        description: Generate preview without deploying
    output: Landing page ID and preview URL
    
  - name: deploy
    description: Deploy landing page to target scenario
    api_endpoint: /api/v1/landing-pages/:id/deploy
    arguments:
      - name: landing_page_id
        type: string
        required: true
        description: Landing page ID to deploy
      - name: target_scenario
        type: string
        required: true
        description: Target scenario for deployment
    flags:
      - name: --method
        description: Deployment method (direct, claude-agent)
      - name: --backup
        description: Backup existing landing page before deployment
    output: Deployment status and progress information
    
  - name: template
    description: Manage landing page templates
    subcommands:
      - name: list
        description: List available templates
        flags: [--category, --saas-type, --industry]
      - name: create
        description: Create new template
        arguments: [name, category]
      - name: preview
        description: Preview template
        arguments: [template_id]
    
  - name: analytics
    description: View landing page performance analytics
    api_endpoint: /api/v1/analytics/dashboard
    flags:
      - name: --scenario
        description: Filter analytics by scenario
      - name: --timeframe
        description: Analytics timeframe (7d, 30d, 90d)
      - name: --format
        description: Output format (table, json, csv)
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **PostgreSQL**: Required for all data persistence and analytics
- **Claude Code**: Enables automated deployment to target scenarios
- **Scenarios with SaaS characteristics**: Provides targets for landing page generation

### Downstream Enablement
- **Marketing Automation**: Landing pages become lead capture endpoints
- **SaaS Business Generation**: Complete business deployment with landing pages
- **Conversion Optimization**: A/B testing framework for all scenarios
- **SEO Intelligence**: Search optimization patterns across the ecosystem

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: marketing-automation-suite
    capability: Professional landing pages with conversion tracking
    interface: API/Landing page URLs
    
  - scenario: saas-business-generator
    capability: Complete business landing page deployment
    interface: API/Agent orchestration
    
  - scenario: analytics-dashboard
    capability: Landing page performance metrics
    interface: API/Analytics endpoints
    
consumes_from:
  - scenario: brand-manager
    capability: Brand assets and visual identity
    fallback: Generic styling without brand consistency
    
  - scenario: funnel-builder
    capability: Advanced conversion funnels
    fallback: Simple call-to-action forms
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "Modern SaaS marketing tool (HubSpot/Unbounce) meets AI automation platform"
  
  visual_style:
    color_scheme: 
      primary: "#6366F1" # Indigo for professionalism
      secondary: "#10B981" # Emerald for success/growth
      accent: "#F59E0B" # Amber for attention/CTA
      neutral_dark: "#1F2937"
      neutral_light: "#F9FAFB"
      background: "Clean white with subtle gradients"
    
    typography:
      primary: "Inter" # Clean, modern, highly readable
      secondary: "JetBrains Mono" # For code/technical content
      display: "Poppins" # For headings and marketing copy
    
    layout:
      structure: "Dashboard layout with sidebar navigation"
      spacing: "Generous whitespace with card-based components"
      components: "Clean cards with subtle shadows and hover states"
    
    animations:
      philosophy: "Smooth, purposeful interactions that enhance UX"
      examples: 
        - "Template preview animations"
        - "A/B test result transitions"
        - "Deployment progress indicators"

  personality:
    tone: professional_friendly
    mood: confident_empowering
    target_feeling: "I can create professional marketing assets effortlessly"
```

### Target Audience Alignment
- **Primary Users**: SaaS founders, marketing teams, growth hackers
- **User Expectations**: Professional, conversion-focused design tools
- **Accessibility**: WCAG AA compliant for business use
- **Responsive Design**: Desktop-first with mobile optimization

## 💰 Value Proposition

### Business Value
- **Primary Value**: Professional landing pages for every SaaS scenario without design/development costs
- **Revenue Potential**: $15K - $75K per deployment for marketing agencies
- **Cost Savings**: Replaces $5K - $50K custom landing page development
- **Market Differentiator**: AI-powered landing pages with automatic deployment and optimization

### Technical Value
- **Reusability Score**: 10/10 - Every SaaS scenario needs landing pages
- **Complexity Reduction**: Makes professional marketing pages accessible to non-designers
- **Innovation Enablement**: Foundation for complete automated SaaS business generation

## 🧬 Evolution Path

### Version 1.0 (Current)
- SaaS scenario detection and registry
- Template system with base SaaS templates
- A/B testing infrastructure
- Claude Code integration for deployment
- Basic analytics and performance tracking

### Version 2.0 (Planned)
- AI-powered content generation with Ollama
- Brand integration with brand-manager
- Advanced analytics dashboard
- Template marketplace with community contributions
- Multi-language support

### Long-term Vision
- Complete marketing automation integration
- Real-time conversion optimization using ML
- Cross-scenario marketing intelligence
- Become the marketing brain for all SaaS scenarios

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete resource dependencies
    - Database schemas for scenario registry and analytics
    - Template system with base templates
    - CLI for scenario management
    
  deployment_targets:
    - local: Direct service deployment
    - kubernetes: Marketing services pod
    - cloud: Marketing automation SaaS
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
      - starter: $200/month (5 scenarios, basic templates)
      - professional: $800/month (unlimited scenarios, custom templates)
      - enterprise: $2500/month (white-label, custom branding)
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: saas-landing-manager
    category: marketing
    capabilities: [landing_page_generation, ab_testing, seo_optimization, agent_orchestration]
    interfaces:
      - api: http://localhost:${SAAS_LANDING_API_PORT}/api/v1
      - cli: saas-landing-manager
      - ui: http://localhost:${SAAS_LANDING_UI_PORT}
      
  metadata:
    description: Automated landing page generation and optimization for SaaS scenarios
    keywords: [saas, landing-pages, marketing, ab-testing, conversion-optimization]
    dependencies: [postgres, claude-code]
    enhances: [ALL_SAAS_SCENARIOS]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Claude Code deployment failures | Medium | High | Fallback to direct file generation with manual deployment |
| Template rendering issues | Low | Medium | Comprehensive template validation and preview system |
| A/B test traffic routing errors | Low | High | Graceful fallback to control variant, detailed logging |

### Operational Risks
- **Template Quality Control**: Implement template validation and community rating system
- **SEO Compliance**: Automated checks for SEO best practices and performance metrics
- **Landing Page Consistency**: Standardized file structure and deployment patterns

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: saas-landing-manager

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/saas-landing-manager
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - templates/base/b2b-tool.html
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - ui
    - initialization/storage/postgres
    - templates/base
    - templates/industry
    - templates/components

resources:
  required: [postgres]
  optional: [claude-code, ollama, browserless]
  health_timeout: 60

tests:
  - name: "API health check"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "SaaS scenario detection"
    type: http
    service: api
    endpoint: /api/v1/scenarios/scan
    method: POST
    body:
      force_rescan: true
    expect:
      status: 200
      body:
        total_scenarios: ">0"
        
  - name: "Template listing"
    type: http
    service: api
    endpoint: /api/v1/templates
    method: GET
    expect:
      status: 200
      body:
        templates: "array"
        
  - name: "CLI scan command"
    type: exec
    command: ./cli/saas-landing-manager scan --dry-run --json
    expect:
      exit_code: 0
      output_contains: ["saas_scenarios"]
```

## 📝 Implementation Notes

### Design Decisions
**No n8n Workflows**: Direct Go API implementation for better reliability
- Alternative considered: n8n workflow orchestration
- Decision driver: Architectural decision to move away from n8n dependencies
- Trade-offs: More Go code but better performance and reliability

**Claude Code Integration**: Agent spawning for complex deployments
- Alternative considered: Direct file copying
- Decision driver: Need for intelligent deployment with scenario-specific customization
- Trade-offs: More complex but enables intelligent adaptation

### Security Considerations
- **Landing Page Content**: Validate all user-generated content to prevent XSS
- **Agent Spawning**: Secure Claude Code agent authentication and authorization
- **A/B Test Data**: Anonymize analytics data and respect privacy requirements

## 🔗 References

### Documentation
- README.md - User-facing overview and getting started guide
- docs/api.md - Complete API specification
- docs/templates.md - Template system documentation
- docs/deployment.md - Landing page deployment guide

### Related PRDs
- funnel-builder PRD (integration target for advanced funnels)
- brand-manager PRD (integration source for brand assets)
- marketing-automation-suite PRD (downstream consumer)

### External Resources
- Landing page design best practices
- A/B testing statistical significance guidelines
- Core Web Vitals optimization techniques

---

**Last Updated**: 2025-09-06  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After major feature implementations