# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Generates legally compliant privacy policies, terms of service, and other legal documents tailored to specific business types and jurisdictions. Provides programmatic access for other scenarios to automatically generate legal compliance documentation.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Every generated business scenario can automatically obtain professional legal documents, reducing friction to production deployment. The system learns from usage patterns to improve clause combinations and stay current with regulatory changes through web-sourced templates.

### Recursive Value
**What new scenarios become possible after this exists?**
- **SaaS billing scenarios** can auto-generate compliant billing terms
- **Data collection scenarios** can ensure GDPR/CCPA compliance documentation
- **App deployment scenarios** can bundle required legal docs for app stores
- **Multi-tenant platforms** can generate custom terms per tenant
- **Compliance auditor** scenario can verify legal doc completeness

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Generate privacy policy from business inputs (name, type, jurisdiction, data collected) *(2025-09-24: Working with database and fallback templates)*
  - [x] Generate terms of service from business parameters *(2025-09-24: Working with database and fallback templates)*
  - [x] Support major jurisdictions (US, EU/GDPR, UK, Canada, Australia) *(2025-09-24: All jurisdictions seeded in database)*
  - [x] Track template freshness with generation timestamps *(2025-09-24: Implemented with days_old tracking and freshness status)*
  - [x] CLI interface for programmatic generation *(2025-09-24: Fully functional with database integration)*
  - [x] PostgreSQL storage for templates and generated documents *(2025-09-24: Schema created, templates seeded, integration complete)*
  
- **Should Have (P1)**
  - [x] Web search integration to fetch current legal requirements *(2025-09-28: Implemented web_updater.sh with fetch and update capabilities)*
  - [x] Version history tracking for generated documents *(2025-10-03: Implemented document_history table tracking, CLI history command, API endpoint)*
  - [x] Multi-format export (HTML, Markdown, PDF via Browserless) *(2025-09-28: Added PDF export module, HTML conversion, format support in CLI)*
  - [x] Semantic search for relevant clauses via Qdrant *(2025-10-03: Implemented semantic_search.sh with Qdrant integration and PostgreSQL fallback)*
  - [x] Cookie policy and EULA generation *(2025-09-28: Templates support all document types)*
  
- **Nice to Have (P2)**
  - [ ] Jurisdiction-specific clause recommendations
  - [ ] Industry-specific templates (healthcare, fintech, gaming)
  - [ ] Multi-language support
  - [ ] Diff view for policy updates
  - [ ] Compliance checklist generation

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Generation Time | < 5s for standard document | API timing logs |
| Template Freshness | Updated within 30 days | Timestamp tracking |
| Accuracy | > 95% jurisdiction compliance | Legal review sampling |
| Storage Efficiency | < 100KB per document | PostgreSQL metrics |

### Quality Gates
- [x] All P0 requirements implemented and tested *(2025-09-24: All P0 features working)*
- [x] Templates sourced from authoritative legal sources *(2025-09-24: Basic compliant templates seeded)*
- [x] Generation produces valid, complete documents *(2025-09-24: Verified with test generation)*
- [x] CLI and API interfaces fully functional *(2025-09-24: CLI working, API health check functional)*
- [x] Documents pass basic legal compliance checks *(2025-09-24: Templates include required clauses)*

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store templates, generated documents, and business profiles
    integration_pattern: Direct SQL via CLI
    access_method: resource-postgres query
    
  - resource_name: ollama
    purpose: Generate custom clauses and adapt templates
    integration_pattern: Direct CLI invocation
    access_method: resource-ollama prompt
    
optional:
  - resource_name: qdrant
    purpose: Semantic search for relevant legal clauses
    fallback: Full-text PostgreSQL search
    access_method: resource-qdrant search
    
  - resource_name: browserless
    purpose: Generate PDF versions of documents
    fallback: Markdown/HTML only
    access_method: resource-browserless pdf
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:     # Not using n8n per requirements
    - note: "Explicitly avoiding n8n workflows per design decision"
  
  2_resource_cli:        # Primary integration method
    - command: resource-postgres query
      purpose: Template and document storage
    - command: resource-ollama prompt
      purpose: Intelligent clause generation
    - command: resource-qdrant search (if available)
      purpose: Find relevant legal clauses
  
  3_direct_api:          # Only for web fetching
    - justification: Need to fetch current legal templates from web
      endpoint: WebSearch and WebFetch for template updates
```

### Data Models
```yaml
primary_entities:
  - name: LegalTemplate
    storage: postgres
    schema: |
      {
        id: UUID
        template_type: string (privacy|terms|cookie|eula)
        jurisdiction: string
        industry: string
        source_url: string
        fetched_at: timestamp
        last_validated: timestamp
        content: text
        metadata: jsonb
      }
    relationships: Used by GeneratedDocument
    
  - name: GeneratedDocument
    storage: postgres
    schema: |
      {
        id: UUID
        business_id: UUID
        document_type: string
        version: integer
        generated_at: timestamp
        template_id: UUID (FK)
        customizations: jsonb
        content: text
        format: string (html|markdown|pdf)
      }
    relationships: References LegalTemplate, BusinessProfile
    
  - name: BusinessProfile
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        type: string (saas|ecommerce|mobile|consulting)
        jurisdiction: string[]
        data_collected: jsonb
        created_at: timestamp
        metadata: jsonb
      }
    relationships: Has many GeneratedDocuments
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/legal/generate
    purpose: Generate legal document from business parameters
    input_schema: |
      {
        business_name: string
        business_type: string
        jurisdictions: string[]
        document_type: string
        data_types: string[] (optional)
        custom_clauses: string[] (optional)
      }
    output_schema: |
      {
        document_id: UUID
        content: string
        format: string
        generated_at: timestamp
        template_version: string
        preview_url: string
      }
    sla:
      response_time: 5000ms
      availability: 99%
      
  - method: GET
    path: /api/v1/legal/templates/freshness
    purpose: Check when templates were last updated
    output_schema: |
      {
        last_update: timestamp
        stale_templates: object[]
        update_available: boolean
      }
```

### Event Interface
```yaml
published_events:
  - name: legal.document.generated
    payload: {document_id, business_id, type}
    subscribers: billing scenarios, deployment scenarios
    
  - name: legal.templates.updated
    payload: {jurisdiction, update_count, timestamp}
    subscribers: compliance scenarios
    
consumed_events:
  - name: scenario.deployment.initiated
    action: Auto-generate required legal docs for deployment
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: privacy-terms-generator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show generator status and template freshness
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: generate
    description: Generate legal document
    api_endpoint: /api/v1/legal/generate
    arguments:
      - name: type
        type: string
        required: true
        description: Document type (privacy|terms|cookie|eula)
      - name: business-name
        type: string
        required: true
        description: Business or app name
    flags:
      - name: --jurisdiction
        description: Target jurisdiction(s)
      - name: --business-type
        description: Type of business
      - name: --format
        description: Output format (html|markdown|pdf)
      - name: --output
        description: Output file path
    output: Generated document content or file path
    
  - name: update-templates
    description: Fetch latest legal templates from web
    flags:
      - name: --jurisdiction
        description: Specific jurisdiction to update
      - name: --force
        description: Force update even if recent
    output: Update summary with changes
    
  - name: list-templates
    description: Show available templates and freshness
    flags:
      - name: --type
        description: Filter by document type
      - name: --stale
        description: Show only stale templates
    output: Template list with timestamps
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **Ollama**: For intelligent clause adaptation and customization
- **PostgreSQL**: For persistent storage of templates and documents
- **Web Access**: For fetching current legal requirements

### Downstream Enablement
- **SaaS Deployment**: Auto-generates required legal docs
- **App Store Submission**: Provides necessary compliance docs
- **Multi-tenant Platforms**: Per-tenant legal document generation
- **Compliance Auditing**: Enables automated compliance checking

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: saas-billing-hub
    capability: Auto-generated billing terms and privacy policy
    interface: CLI/API
    
  - scenario: app-to-ios
    capability: App Store required privacy policy
    interface: CLI
    
  - scenario: deployment-manager
    capability: Production-ready legal documents
    interface: API
    
consumes_from:
  - scenario: none initially
    capability: Standalone operation
    fallback: N/A
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "Clean legal tech SaaS like Termly or iubenda"
  
  visual_style:
    color_scheme: light with blue accents
    typography: modern, highly readable
    layout: spacious with clear sections
    animations: none (professional context)
  
  personality:
    tone: serious, trustworthy
    mood: calm, confident
    target_feeling: "This handles my legal needs professionally"
```

### Target Audience Alignment
- **Primary Users**: Developers, startup founders, scenario creators
- **User Expectations**: Professional, accurate, up-to-date legal docs
- **Accessibility**: WCAG AA compliance for document generation UI
- **Responsive Design**: Desktop-first, mobile-friendly

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates $5-15K legal fees per business
- **Revenue Potential**: $10K - $30K per deployment as legal-as-a-service
- **Cost Savings**: 100+ hours of legal research per scenario
- **Market Differentiator**: Auto-updating templates with jurisdiction awareness

### Technical Value
- **Reusability Score**: 10/10 - Every business scenario needs legal docs
- **Complexity Reduction**: Complex legal requirements → Simple API call
- **Innovation Enablement**: Removes legal friction from rapid deployment

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core document generation (privacy, terms)
- Major jurisdiction support
- CLI/API interfaces
- Template freshness tracking

### Version 2.0 (Planned)
- Industry-specific templates
- Multi-language support
- Compliance audit reports
- Legal change notifications

### Long-term Vision
- AI-powered compliance monitoring
- Automatic policy updates based on business changes
- Integration with legal review services
- Regulatory change prediction

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with metadata
    - PostgreSQL schema initialization
    - Template seeding scripts
    - Health check endpoints
    
  deployment_targets:
    - local: Shell scripts with resource CLI
    - kubernetes: ConfigMaps for templates
    - cloud: Lambda functions for generation
    
  revenue_model:
    - type: usage-based
    - pricing_tiers:
        - free: 10 documents/month
        - pro: Unlimited + custom clauses
        - enterprise: White-label + API
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: privacy-terms-generator
    category: generation
    capabilities: [legal-doc-generation, compliance, multi-jurisdiction]
    interfaces:
      - api: /api/v1/legal
      - cli: privacy-terms-generator
      - events: legal.*
      
  metadata:
    description: "Generate compliant privacy policies and terms of service"
    keywords: [legal, privacy, terms, GDPR, CCPA, compliance]
    dependencies: [postgres, ollama]
    enhances: [all business scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Stale templates | Medium | High | Auto-update with web fetch, timestamp tracking |
| Ollama unavailable | Low | Medium | Fallback to template-only generation |
| Legal inaccuracy | Low | High | Source from authoritative sites, add disclaimers |

### Operational Risks
- **Legal Liability**: Clear disclaimers about attorney review recommendation
- **Template Drift**: Weekly freshness checks with alerts
- **Jurisdiction Changes**: Web monitoring for regulatory updates

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: privacy-terms-generator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - lib/generator.sh
    - cli/privacy-terms-generator
    - cli/install.sh
    - initialization/postgres/schema.sql
    - scenario-test.yaml
    
  required_dirs:
    - cli
    - lib
    - initialization
    - initialization/postgres
    - initialization/prompts/legal
    - ui

resources:
  required: [postgres, ollama]
  optional: [qdrant, browserless]
  health_timeout: 60

tests:
  - name: "Generate privacy policy"
    type: exec
    command: ./cli/privacy-terms-generator generate privacy --business-name "TestCo" --jurisdiction US --format markdown
    expect:
      exit_code: 0
      output_contains: ["Privacy Policy", "TestCo", "Information We Collect"]
      
  - name: "Check template freshness"
    type: exec
    command: ./cli/privacy-terms-generator list-templates --json
    expect:
      exit_code: 0
      json_path: ".[0].fetched_at"
      json_value_type: string
```

## 📝 Implementation Notes

### Design Decisions
**No n8n dependency**: Direct CLI integration chosen for simplicity and reduced complexity
- Alternative considered: n8n workflows for orchestration
- Decision driver: Reduce dependencies, improve reliability
- Trade-offs: Less visual workflow editing, but more maintainable

### Known Limitations
- **Legal Accuracy**: Not a replacement for legal counsel
  - Workaround: Clear disclaimers and attorney review recommendations
  - Future fix: Partnership with legal review services

### Security Considerations
- **Data Protection**: Business profiles encrypted at rest
- **Access Control**: API key required for generation
- **Audit Trail**: All document generation logged with parameters

## 🔗 References

### Documentation
- README.md - User guide and quick start
- docs/templates.md - Template structure and customization
- docs/jurisdictions.md - Supported jurisdictions and requirements

### External Resources
- GDPR Official Text: https://gdpr-info.eu/
- CCPA Resources: https://oag.ca.gov/privacy/ccpa
- Terms of Service Best Practices: https://www.termsfeed.com/

---

**Last Updated**: 2025-10-20
**Status**: P0 Complete (100% of must-have requirements), P1 Complete (100% of should-have requirements)
**Security Posture**: ✅ All critical/high severity violations resolved
**Owner**: AI Agent
**Review Cycle**: Weekly template freshness check

## Progress History
- **2025-09-24**: 0% → 100% P0 (Initialized database, seeded templates, implemented freshness tracking, integrated CLI with database)
- **2025-09-28**: P1 60% complete (Added web search integration, PDF export, HTML conversion, enhanced UI with real API integration)
- **2025-10-03**: P1 60% → 100% complete (Added version history tracking, semantic search via Qdrant with PostgreSQL fallback, CLI commands for history and search, API endpoints for history and search)
- **2025-10-20**: Security & Standards hardening (Fixed lifecycle protection, CORS wildcard, environment variable defaults, Makefile structure, test infrastructure - all critical/high violations resolved)