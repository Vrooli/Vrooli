# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
SEO Optimizer provides automated search engine optimization analysis and recommendations. It analyzes web pages for SEO best practices, performs keyword research, content optimization, and competitor analysis to help improve search rankings.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides SEO intelligence that other scenarios can leverage to improve their web presence. Any scenario that generates web content can use this to ensure it's optimized for discovery. The keyword research and competitive analysis become reusable knowledge for content generation scenarios.

### Recursive Value
**What new scenarios become possible after this exists?**
- Content marketing automation that generates SEO-optimized blog posts
- E-commerce product description optimizer using SEO insights
- Landing page generator with built-in SEO best practices
- Website migration assistant that preserves SEO equity
- Local business optimizer for Google My Business and local search

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] SEO audit of any webpage with scoring and recommendations
  - [ ] Integration with shared Ollama workflow for AI analysis
  - [ ] Persistent storage of audit results in PostgreSQL
  
- **Should Have (P1)**
  - [ ] Keyword research functionality
  - [ ] Content optimization suggestions
  - [ ] Competitor analysis features
  - [ ] Redis caching for improved performance
  
- **Nice to Have (P2)**
  - [ ] Rank tracking over time
  - [ ] Bulk URL auditing
  - [ ] SEO report generation

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 5000ms for single page audit | API monitoring |
| Throughput | 10 audits/minute | Load testing |
| Accuracy | > 85% for SEO recommendations | Validation against known good sites |
| Resource Usage | < 2GB memory, < 30% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: n8n
    purpose: Workflow orchestration for SEO analysis pipelines
    integration_pattern: Shared workflows and scenario-specific workflows
    access_method: Shared ollama.json workflow for AI analysis
    
  - resource_name: postgres
    purpose: Store SEO audits, keyword data, and historical rankings
    integration_pattern: Direct database connection
    access_method: SQL via Go database/sql
    
  - resource_name: ollama
    purpose: AI-powered SEO analysis and recommendations
    integration_pattern: Via shared n8n workflow
    access_method: initialization/n8n/ollama.json
    
  - resource_name: browserless
    purpose: Web page scraping and screenshot capture
    integration_pattern: CLI commands via n8n workflows
    access_method: vrooli resource browserless
    
optional:
  - resource_name: redis
    purpose: Cache SEO analysis results for faster repeat queries
    fallback: Direct database queries if unavailable
    access_method: Redis client library
    
  - resource_name: qdrant
    purpose: Store content embeddings for semantic SEO analysis
    fallback: Basic keyword matching without semantic understanding
    access_method: HTTP API
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: AI-powered SEO analysis and recommendations
  
  2_resource_cli:
    - command: resource-browserless screenshot
      purpose: Capture page screenshots for visual SEO assessment
    - command: resource-browserless scrape
      purpose: Extract page content for analysis
  
  3_direct_api:
    - justification: PostgreSQL requires direct connection for complex queries
      endpoint: postgres://localhost:5432/seo_optimizer
```

### Data Models
```yaml
primary_entities:
  - name: SEOAudit
    storage: postgres
    schema: |
      {
        id: UUID
        url: string
        timestamp: datetime
        overall_score: float
        title_analysis: json
        meta_analysis: json
        content_quality: json
        technical_seo: json
      }
    relationships: Can have multiple keyword analyses
    
  - name: KeywordAnalysis
    storage: postgres
    schema: |
      {
        id: UUID
        audit_id: UUID (FK)
        keyword: string
        volume: integer
        difficulty: float
        relevance_score: float
      }
    relationships: Belongs to SEOAudit
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/seo/audit
    purpose: Perform comprehensive SEO audit of a webpage
    input_schema: |
      {
        url: string (required)
        depth: integer (optional, default: 1)
        include_competitors: boolean (optional)
      }
    output_schema: |
      {
        audit_id: string
        url: string
        overall_score: float (0-100)
        recommendations: array
        issues: array
      }
    sla:
      response_time: 5000ms
      availability: 99%
      
  - method: GET
    path: /api/v1/seo/audit/{id}
    purpose: Retrieve stored SEO audit results
    output_schema: |
      {
        audit: SEOAudit object
        keywords: array of KeywordAnalysis
      }
```

### Event Interface
```yaml
published_events:
  - name: seo.audit.completed
    payload: { audit_id: string, url: string, score: float }
    subscribers: Content optimization scenarios
    
consumed_events:
  - name: content.published
    action: Trigger automatic SEO audit of new content
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: seo-optimizer
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
  - name: audit
    description: Perform SEO audit on a URL
    api_endpoint: /api/v1/seo/audit
    arguments:
      - name: url
        type: string
        required: true
        description: URL to audit
    flags:
      - name: --depth
        description: Crawl depth for multi-page audit
      - name: --json
        description: Output results as JSON
    output: SEO audit results with score and recommendations
    
  - name: keywords
    description: Perform keyword research
    api_endpoint: /api/v1/seo/keywords
    arguments:
      - name: topic
        type: string
        required: true
        description: Topic or seed keyword
    output: Keyword suggestions with volume and difficulty
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Ollama**: Provides AI analysis capabilities via shared workflow
- **N8n**: Orchestrates complex SEO analysis workflows
- **Browserless**: Enables web scraping and screenshot capture
- **PostgreSQL**: Stores audit results and historical data

### Downstream Enablement
**What future capabilities does this unlock?**
- **Content Generator**: Can create SEO-optimized content
- **E-commerce Optimizer**: Product descriptions with SEO best practices
- **Marketing Automation**: SEO-aware campaign creation

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: content-generator
    capability: SEO scoring and optimization suggestions
    interface: API/CLI
    
  - scenario: e-commerce-manager
    capability: Product page SEO analysis
    interface: API
    
consumes_from:
  - scenario: competitor-monitor
    capability: Competitor website changes
    fallback: Manual competitor URL entry
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Ahrefs/SEMrush professional SEO tools
  
  visual_style:
    color_scheme: light with blue/green accents
    typography: modern, clean, data-focused
    layout: dashboard with clear metrics
    animations: subtle, professional transitions
  
  personality:
    tone: professional, authoritative
    mood: focused, analytical
    target_feeling: confidence in SEO improvements

style_references:
  professional: 
    - product-manager: "Modern SaaS dashboard aesthetic"
    - research-assistant: "Clean, professional, information-dense"
```

### Target Audience Alignment
- **Primary Users**: Digital marketers, content creators, website owners
- **User Expectations**: Professional tool with actionable insights
- **Accessibility**: WCAG 2.1 AA compliance
- **Responsive Design**: Desktop-first, tablet-supported

## 💰 Value Proposition

### Business Value
- **Primary Value**: Improved search rankings leading to increased organic traffic
- **Revenue Potential**: $5K - $15K per deployment for agencies
- **Cost Savings**: Replaces $300/month SEO tool subscriptions
- **Market Differentiator**: AI-powered insights with local deployment

### Technical Value
- **Reusability Score**: High - any web-facing scenario can leverage
- **Complexity Reduction**: Makes SEO accessible to non-experts
- **Innovation Enablement**: Foundation for automated content optimization

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core SEO audit functionality
- Basic keyword research
- N8n workflow integration
- PostgreSQL storage

### Version 2.0 (Planned)
- Rank tracking over time
- Bulk URL auditing
- Advanced competitor analysis
- PDF report generation

### Long-term Vision
- ML-based SEO prediction models
- Automated fix implementation
- Integration with CMS platforms
- Multi-language SEO support

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: seo-optimizer

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/seo-optimizer
    - initialization/postgres/schema.sql
    - initialization/n8n/seo-audit.json
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/n8n
    - initialization/postgres
    - ui

resources:
  required: [n8n, postgres, ollama, browserless]
  optional: [redis, qdrant]
  health_timeout: 60

tests:
  - name: "API health check"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "SEO audit endpoint"
    type: http
    service: api
    endpoint: /api/v1/seo/audit
    method: POST
    body:
      url: "https://example.com"
    expect:
      status: 201
      body:
        audit_id: string
        overall_score: number
```

## 📝 Implementation Notes

### Design Decisions
**Workflow Architecture**: Chose n8n workflows over direct API calls
- Alternative considered: Direct Ollama API integration
- Decision driver: Reusability and visual debugging
- Trade-offs: Slight latency for better maintainability

### Known Limitations
- **Screenshot capture**: Limited to public websites
  - Workaround: Manual HTML upload option
  - Future fix: Authentication support in v2.0

### Security Considerations
- **Data Protection**: Audit results contain no PII
- **Access Control**: API key authentication for external access
- **Audit Trail**: All audits logged with timestamp and requester

## 🔗 References

### Documentation
- README.md - User-facing overview
- docs/api.md - API specification
- docs/cli.md - CLI documentation

### Related PRDs
- content-generator PRD (downstream consumer)
- competitor-monitor PRD (upstream provider)

### External Resources
- Google SEO Starter Guide
- Schema.org structured data specifications
- Core Web Vitals documentation

---

**Last Updated**: 2025-08-20  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Monthly validation against implementation