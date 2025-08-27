# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Intelligent, multi-source research orchestration with automated analysis, synthesis, and knowledge management. This scenario provides DeepSearch-quality research capabilities with complete data privacy, enabling agents and users to gather, verify, and synthesize information from 70+ search engines while maintaining local control of all data.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Provides validated, multi-source information gathering that any agent can leverage
- Creates a persistent knowledge base through vector embeddings that grows smarter over time
- Establishes research patterns and templates that agents can reuse and improve
- Enables fact-checking and verification workflows for other scenarios' outputs
- Offers scheduled intelligence gathering that keeps the entire system updated on relevant topics

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Market Intelligence Platform**: Automated competitive analysis using research pipelines
2. **Academic Paper Generator**: Leveraging research capabilities for literature reviews
3. **News Aggregation System**: Building on scheduled reports for personalized news digests
4. **Due Diligence Assistant**: Using deep research for investment/acquisition analysis
5. **Technical Documentation Hub**: Aggregating and synthesizing technical knowledge

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Multi-source search across 70+ engines via SearXNG
  - [x] AI-powered analysis and synthesis using Ollama
  - [x] Vector storage for semantic search in Qdrant
  - [x] PDF report generation via Unstructured-IO
  - [x] Dashboard interface through Windmill
  - [x] Scheduled report automation via n8n
  - [x] Chat interface with RAG capabilities
  
- **Should Have (P1)**
  - [x] Source quality ranking and verification
  - [x] Contradiction detection and highlighting
  - [x] Configurable research depth (quick/standard/deep)
  - [x] Report templates and presets
  - [x] JavaScript-heavy site extraction via Browserless
  
- **Nice to Have (P2)**
  - [ ] Multi-language research capabilities
  - [ ] Custom source prioritization
  - [ ] Research collaboration features

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for API requests | API monitoring |
| Research Completion | < 5min for standard depth | Pipeline monitoring |
| Source Verification | > 5 sources per claim | Quality audit |
| Embedding Generation | < 2s per document | Vector pipeline metrics |
| PDF Generation | < 30s per 10-page report | Export monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration tests pass with all 9 required resources
- [x] Research pipeline completes end-to-end in under 10 minutes
- [x] Dashboard loads and displays reports correctly
- [x] Chat interface provides contextual responses from report data

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: ollama
    purpose: AI analysis and embedding generation
    integration_pattern: Shared n8n workflow
    access_method: ollama.json and embedding-generator.json workflows
    
  - resource_name: searxng
    purpose: Privacy-respecting meta-search across 70+ engines
    integration_pattern: HTTP API for search queries
    access_method: Direct API calls (no CLI available)
    
  - resource_name: n8n
    purpose: Research pipeline orchestration and scheduling
    integration_pattern: Workflow triggers and webhook APIs
    access_method: resource-n8n CLI for workflow management
    
  - resource_name: windmill
    purpose: Dashboard UI and chat interface
    integration_pattern: Frontend application hosting
    access_method: resource-windmill CLI for app deployment
    
  - resource_name: postgres
    purpose: Report metadata, schedules, chat history
    integration_pattern: Direct SQL connections
    access_method: resource-postgres CLI for backups/management
    
  - resource_name: qdrant
    purpose: Vector embeddings for semantic search
    integration_pattern: REST API for vector operations
    access_method: Direct API (vector operations require API)
    
  - resource_name: unstructured-io
    purpose: Source processing and PDF generation
    integration_pattern: API for document processing
    access_method: Direct API calls for document processing
    
  - resource_name: browserless
    purpose: JavaScript-heavy site extraction
    integration_pattern: Headless browser API
    access_method: Direct API (browser automation requires API)
    
  - resource_name: minio
    purpose: PDF storage and source caching
    integration_pattern: S3-compatible object storage
    access_method: resource-minio CLI for bucket management
    
optional:
  - resource_name: redis
    purpose: Caching for faster repeated searches
    fallback: Direct database queries
    access_method: resource-redis CLI for cache management
```

### Resource Integration Standards
```yaml
# Priority order for resource access:
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: Standardized text generation across all scenarios
      reused_by: [product-manager, study-buddy, idea-generator]
      
    - workflow: embedding-generator.json
      location: initialization/automation/n8n/
      purpose: Consistent vector embedding generation
      reused_by: [stream-of-consciousness-analyzer, notes]
      
    - workflow: research-orchestrator.json
      location: initialization/automation/n8n/
      purpose: Core research pipeline (scenario-specific)
      
  2_resource_cli:
    - command: resource-n8n list-workflows
      purpose: Verify workflow deployment
      
    - command: resource-postgres backup
      purpose: Scheduled database backups
      
    - command: resource-minio create-bucket research-reports
      purpose: Initialize storage buckets
      
  3_direct_api:
    - justification: SearXNG has no CLI, search requires direct API
      endpoint: http://localhost:9200/search
      
    - justification: Qdrant vector operations require API
      endpoint: http://localhost:6333/collections/research/points
      
    - justification: Real-time browser automation needs API
      endpoint: http://localhost:4110/content

# Shared workflow criteria:
shared_workflow_validation:
  - ollama.json is truly generic (any text generation)
  - embedding-generator.json is reusable (any document embedding)
  - research-orchestrator.json is scenario-specific (not shared)
```

### Data Models
```yaml
primary_entities:
  - name: ResearchReport
    storage: postgres
    schema: |
      {
        id: UUID
        title: string
        topic: string
        depth: enum(quick, standard, deep)
        status: enum(pending, processing, completed, failed)
        created_at: timestamp
        completed_at: timestamp
        markdown_content: text
        pdf_url: string
        metadata: jsonb
        embeddings_collection: string
      }
    relationships: Has many Sources, ChatConversations, ScheduledReports
    
  - name: ReportEmbedding
    storage: qdrant
    schema: |
      {
        id: UUID
        report_id: UUID
        chunk_text: string
        vector: float[384]
        metadata: {
          page_number: int
          section: string
          source_ids: UUID[]
        }
      }
    relationships: Belongs to ResearchReport
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/research/create
    purpose: Enables other scenarios to request research
    input_schema: |
      {
        topic: string (required)
        depth: enum(quick, standard, deep)
        length: int (1-10 pages)
        template: string (optional)
      }
    output_schema: |
      {
        report_id: UUID
        status: string
        estimated_completion: timestamp
        webhook_url: string
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/research/{report_id}
    purpose: Retrieve completed research reports
    output_schema: |
      {
        id: UUID
        title: string
        content: string (markdown)
        pdf_url: string
        sources: array
        metadata: object
      }
      
  - method: POST
    path: /api/v1/research/chat
    purpose: Conversational interface to research data
    input_schema: |
      {
        report_id: UUID (optional)
        message: string
        context_window: int
      }
    output_schema: |
      {
        response: string
        sources: array
        suggested_topics: array
      }
```

### Event Interface
```yaml
published_events:
  - name: research.report.completed
    payload: { report_id: UUID, topic: string, pdf_url: string }
    subscribers: [product-manager, study-buddy, content-generator]
    
  - name: research.insight.discovered
    payload: { type: string, content: string, confidence: float }
    subscribers: [stream-of-consciousness-analyzer, idea-generator]
    
consumed_events:
  - name: content.analysis.requested
    action: Trigger deep research on content topic
    
  - name: market.change.detected
    action: Schedule recurring market research reports
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: research-assistant
install_script: cli/install.sh

# Core commands that MUST be implemented:
required_commands:
  - name: status
    description: Show research pipeline status and resource health
    flags: [--json, --verbose]
    example: research-assistant status --json
    
  - name: help
    description: Display command help and usage examples
    flags: [--all, --command <name>]
    example: research-assistant help --command create
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]
    example: research-assistant version

# Scenario-specific commands (mirror API endpoints):
custom_commands:
  - name: create
    description: Create a new research report
    api_endpoint: /api/v1/research/create
    arguments:
      - name: topic
        type: string
        required: true
        description: Research topic or question
    flags:
      - name: --depth
        description: Research depth (quick|standard|deep)
        default: standard
      - name: --length
        description: Target report length in pages (1-10)
        default: 5
      - name: --template
        description: Report template to use
    output: JSON with report_id and status
    example: research-assistant create "AI safety research" --depth deep --length 10
    
  - name: get
    description: Retrieve a completed research report
    api_endpoint: /api/v1/research/{report_id}
    arguments:
      - name: report-id
        type: string
        required: true
        description: UUID of the research report
    flags:
      - name: --format
        description: Output format (markdown|pdf|json)
        default: markdown
      - name: --save
        description: Save to file instead of stdout
    output: Report content in requested format
    example: research-assistant get abc-123 --format pdf --save report.pdf
    
  - name: chat
    description: Interactive chat about research data
    api_endpoint: /api/v1/research/chat
    arguments:
      - name: message
        type: string
        required: true
        description: Question or message
    flags:
      - name: --report-id
        description: Specific report context
      - name: --context-window
        description: Number of context chunks (1-10)
        default: 5
    output: Conversational response with sources
    example: research-assistant chat "What are the main findings?" --report-id abc-123
    
  - name: list
    description: List all research reports
    api_endpoint: /api/v1/research/list
    flags:
      - name: --limit
        description: Number of reports to show
        default: 10
      - name: --status
        description: Filter by status (pending|completed|failed)
    output: Table of reports with IDs, titles, and status
    example: research-assistant list --status completed --limit 5
    
  - name: schedule
    description: Schedule recurring research reports
    api_endpoint: /api/v1/research/schedule
    arguments:
      - name: topic
        type: string
        required: true
        description: Research topic for recurring reports
    flags:
      - name: --frequency
        description: Schedule frequency (daily|weekly|monthly)
        default: weekly
      - name: --time
        description: Time to run (HH:MM format)
        default: "09:00"
    output: Schedule confirmation with ID
    example: research-assistant schedule "AI news" --frequency daily --time 08:00
```

### CLI-API Parity Requirements
- **Coverage**: All 5 API endpoints have corresponding CLI commands
- **Naming**: Commands use kebab-case (create, get, list vs /api/v1/research/create)
- **Arguments**: CLI args map directly to API parameters
- **Output**: Supports human-readable tables and JSON (--json flag)
- **Authentication**: Uses API key from ~/.vrooli/research-assistant/config.yaml

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin Go wrapper over api/lib/ functions
  - language: Go (consistent with API implementation)
  - dependencies: Reuses API client library from api/client/
  - error_handling: 
      - Exit 0: Success
      - Exit 1: General error
      - Exit 2: Resource unavailable
      - Exit 3: Invalid arguments
  - configuration:
      - Config file: ~/.vrooli/research-assistant/config.yaml
      - Env override: RESEARCH_API_URL, RESEARCH_API_KEY
      - Flag override: --api-url, --api-key
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/research-assistant
  - path_update: Adds ~/.vrooli/bin to PATH if missing
  - permissions: Sets 755 on CLI binary
  - documentation: Full help via research-assistant help --all
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Ollama**: Must have qwen2.5:32b and nomic-embed-text models loaded
- **n8n Shared Workflows**: ollama.json and embedding-generator.json must be available
- **Resource Connectivity**: All 9 required resources must be healthy and accessible

### Downstream Enablement
**What future capabilities does this unlock?**
- **Knowledge Graph Construction**: Research reports become nodes in knowledge networks
- **Automated Content Generation**: Research feeds into blog posts, documentation, reports
- **Decision Support Systems**: Research data enables AI-powered recommendations
- **Competitive Intelligence**: Automated market monitoring and analysis

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: product-manager
    capability: Market research and competitive analysis
    interface: API - /api/v1/research/create
    
  - scenario: study-buddy
    capability: Academic research and paper summaries
    interface: API + Shared Qdrant collections
    
  - scenario: stream-of-consciousness-analyzer
    capability: Research context for note organization
    interface: Event - research.report.completed
    
consumes_from:
  - scenario: agent-metareasoning-manager
    capability: Research task prioritization
    fallback: Default FIFO queue processing
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "Perplexity Pro meets Notion - clean information density"
  
  visual_style:
    color_scheme: light with dark mode support
    typography: modern sans-serif, optimized for reading
    layout: dashboard with collapsible panels
    animations: subtle transitions, no distractions
  
  personality:
    tone: professional yet approachable
    mood: focused and efficient
    target_feeling: "I have a powerful research assistant at my fingertips"

ui_components:
  dashboard:
    - Report gallery with card-based layout
    - Search bar prominently positioned
    - Quick action buttons for common tasks
    - Schedule overview sidebar
    
  chat_interface:
    - Clean message bubbles with source citations
    - Collapsible context panel
    - Suggested follow-up questions
    
  report_viewer:
    - Distraction-free reading mode
    - Table of contents sidebar
    - Inline source previews on hover
    - Export options toolbar

color_palette:
  primary: "#2563EB"  # Professional blue
  secondary: "#10B981"  # Success green
  accent: "#F59E0B"  # Highlight amber
  background: "#FFFFFF" / "#1F2937" (dark mode)
  text: "#111827" / "#F9FAFB" (dark mode)
```

### Target Audience Alignment
- **Primary Users**: Knowledge workers, researchers, analysts, product managers
- **User Expectations**: Clean, professional interface similar to enterprise SaaS tools
- **Accessibility**: WCAG 2.1 AA compliant, keyboard navigation, screen reader support
- **Responsive Design**: Desktop-first, tablet-optimized, mobile-readable

### Brand Consistency Rules
- **Scenario Identity**: "The professional's research companion"
- **Vrooli Integration**: Maintains Vrooli's efficiency-first philosophy
- **Professional vs Fun**: Professional design - this is a business tool worth $30-50K
- **Differentiation**: More polished than Perplexity, more focused than ChatGPT

## 💰 Value Proposition

### Business Value
- **Primary Value**: Enterprise-grade research platform competing with Perplexity Pro ($200/mo), Elicit ($125/mo), Consensus ($200/mo)
- **Revenue Potential**: $30K - $50K per deployment
- **Cost Savings**: 80% reduction in manual research time, 100% data privacy vs cloud solutions
- **Market Differentiator**: Only solution offering complete on-premise deployment with 70+ search sources

### Technical Value
- **Reusability Score**: 9/10 - Core research capability used by 15+ other scenarios
- **Complexity Reduction**: Turns 20+ manual research steps into single API call
- **Innovation Enablement**: Enables autonomous knowledge acquisition for entire Vrooli system

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core research pipeline with 70+ search engines
- Dashboard and chat interfaces
- Scheduled reports with templates
- PDF export and vector storage

### Version 2.0 (Planned)
- Multi-language research capabilities
- Custom source prioritization and weighting
- Collaborative research workspaces
- Advanced contradiction resolution
- Research quality scoring algorithms

### Long-term Vision
- Autonomous research agent that proactively gathers intelligence
- Self-improving research strategies based on outcome analysis
- Integration with external knowledge bases and APIs
- Becomes the "memory and learning" layer for all Vrooli agents

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - Complete service.json with business model
    - All initialization files for 9 resources
    - Deployment scripts (startup.sh, monitor.sh)
    - Health endpoints on API and all resources
    
  deployment_targets:
    - local: Docker Compose with resource orchestration
    - kubernetes: Helm chart with resource operators
    - cloud: AWS ECS/Fargate templates
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        starter: $500/month (10 users, 100 reports)
        professional: $2000/month (50 users, 1000 reports)  
        enterprise: $5000/month (unlimited users/reports)
    - trial_period: 14 days full access
    - value_proposition: "Replace Perplexity Pro + Elicit + Consensus"
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: research-assistant
    category: research
    capabilities:
      - Multi-source research orchestration
      - Vector-based semantic search
      - Scheduled report generation
      - PDF export with citations
      - Conversational exploration
    interfaces:
      - api: http://localhost:8080/api/v1/research
      - cli: research-assistant
      - events: research.*
      
  metadata:
    description: "DeepSearch-quality research with complete data privacy"
    keywords: [research, analysis, synthesis, RAG, reports, citations]
    dependencies: []
    enhances: [product-manager, study-buddy, content-generator]
```

### Version Management  
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  api_version: v1
  
  breaking_changes: []
  
  deprecations: []
  
  upgrade_path:
    from_0_9: "Run migration script for new vector schema"
    from_1_0: "Direct upgrade, no changes needed"
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| SearXNG rate limiting | Medium | High | Implement request queuing and backoff |
| Ollama model performance | Low | Medium | Model caching and fallback to smaller models |
| Vector database scaling | Low | High | Implement collection partitioning |
| PDF generation failures | Medium | Low | Retry logic with HTML fallback |

### Operational Risks
- **Drift Prevention**: PRD as single source of truth, validated by scenario-test.yaml every deployment
- **Version Compatibility**: Semantic versioning with API v1 stability guarantee
- **Resource Conflicts**: Service.json priorities manage resource allocation (research > other scenarios)
- **Style Drift**: Windmill UI components validated against style guide in PRD
- **CLI Consistency**: Automated tests ensure 100% CLI-API parity on every commit

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# File: scenario-test.yaml
version: 1.0
scenario: research-assistant

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - IMPLEMENTATION_PLAN.md
    - api/main.go
    - api/go.mod
    - cli/research-assistant
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/storage/postgres/seed.sql
    - initialization/automation/n8n/research-orchestrator.json
    - initialization/automation/n8n/scheduled-reports.json
    - initialization/automation/n8n/chat-rag-workflow.json
    - initialization/automation/windmill/dashboard-app.json
    - initialization/storage/qdrant/collections.json
    - initialization/storage/minio/buckets.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization/storage/postgres
    - initialization/storage/qdrant
    - initialization/storage/minio
    - initialization/automation/n8n
    - initialization/automation/windmill
    - initialization/configuration
    - initialization/search/searxng

resources:
  required: [ollama, qdrant, postgres, searxng, n8n, windmill, unstructured-io, browserless, minio]
  optional: [redis]
  health_timeout: 60

tests:
  # Resource health checks:
  - name: "Ollama AI Models Ready"
    type: http
    service: ollama
    endpoint: /api/tags
    method: GET
    expect:
      status: 200
      body_contains: ["qwen2.5:32b", "nomic-embed-text"]
      
  - name: "SearXNG Search Engine Active"
    type: http
    service: searxng
    endpoint: /search?q=test
    method: GET
    expect:
      status: 200
      
  - name: "Qdrant Vector Database Ready"
    type: http
    service: qdrant
    endpoint: /collections
    method: GET
    expect:
      status: 200
      
  # API endpoint tests:
  - name: "Research Creation Endpoint"
    type: http
    service: api
    endpoint: /api/v1/research/create
    method: POST
    body:
      topic: "test research"
      depth: "quick"
    expect:
      status: 201
      body:
        report_id: "*"
        status: "pending"
        
  # CLI command tests:
  - name: "CLI Status Command"
    type: exec
    command: ./cli/research-assistant status --json
    expect:
      exit_code: 0
      output_contains: ["healthy", "resources"]
      
  - name: "CLI Help Command"
    type: exec
    command: ./cli/research-assistant help
    expect:
      exit_code: 0
      output_contains: ["create", "get", "chat", "list"]
      
  # Database tests:
  - name: "Database Schema Initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'"
    expect:
      rows:
        - count: 7  # reports, sources, schedules, chat_history, etc.
        
  # Workflow tests:
  - name: "Research Orchestrator Workflow Active"
    type: n8n
    workflow: research-orchestrator
    expect:
      active: true
      node_count: 12  # Expected nodes in workflow
      
  - name: "Chat RAG Workflow Ready"
    type: n8n  
    workflow: chat-rag-workflow
    expect:
      active: true
      has_webhook: true
```

### Test Execution Gates
```bash
# Full validation suite:
./test.sh --scenario research-assistant --validation complete

# Component tests:
./test.sh --structure    # Verify all files/dirs exist
./test.sh --resources    # Check all 9 resources healthy
./test.sh --integration  # Run API/CLI integration tests
./test.sh --performance  # Validate < 5min research completion
```

### Performance Validation
- [x] API response times < 500ms (95th percentile)
- [x] Research completion < 5 minutes for standard depth
- [x] PDF generation < 30s for 10-page report
- [x] Memory usage < 2GB under normal load
- [x] No memory leaks in 24-hour stress test

### Integration Validation  
- [x] Discoverable via resource registry
- [x] All 5 API endpoints functional with OpenAPI docs
- [x] All 6 CLI commands work with --help documentation
- [x] Shared workflows (ollama.json, embedding-generator.json) registered
- [x] Events published to Redis event bus

### Capability Verification
- [x] Generates multi-source research reports
- [x] Maintains source attribution with citations
- [x] Detects and highlights contradictions
- [x] Produces professional PDF outputs
- [x] Chat interface provides contextual responses
- [x] Style matches professional SaaS expectations

## 📝 Implementation Notes

### Design Decisions
**RAG Architecture**: Chose Qdrant over pgvector for superior performance
- Alternative considered: PostgreSQL with pgvector extension
- Decision driver: 10x faster similarity search at scale
- Trade-offs: Additional resource dependency for specialized performance

**UI Framework**: Selected Windmill over custom React app
- Alternative considered: Custom React dashboard
- Decision driver: Rapid development with built-in components
- Trade-offs: Less customization for faster time-to-market

**Search Engine**: SearXNG over individual API integrations
- Alternative considered: Direct API calls to Google, Bing, etc.
- Decision driver: Privacy preservation and 70+ sources in one
- Trade-offs: Slightly higher latency for complete privacy

### Known Limitations
- **Language Support**: Currently English-only
  - Workaround: Translate queries before searching
  - Future fix: Multi-language models in v2.0
  
- **Real-time Data**: No streaming data sources
  - Workaround: Scheduled reports for regular updates
  - Future fix: WebSocket integration for live feeds

### Security Considerations
- **Data Protection**: All research data stored locally, no external API keys required
- **Access Control**: Role-based access via Windmill authentication
- **Audit Trail**: Complete logging of all research requests and access

## 🔗 References

### Documentation
- README.md - User-facing overview
- IMPLEMENTATION_PLAN.md - Detailed implementation guide
- api/README.md - API specification
- cli/README.md - CLI documentation

### Related PRDs
- scenarios/product-manager/PRD.md - Consumes research capability
- scenarios/study-buddy/PRD.md - Extends research for academic use
- scenarios/agent-metareasoning-manager/PRD.md - Provides task prioritization

### External Resources
- [SearXNG Documentation](https://docs.searxng.org/)
- [Qdrant Vector Database Guide](https://qdrant.tech/documentation/)
- [n8n Workflow Automation](https://docs.n8n.io/)

---

**Last Updated**: 2025-01-20  
**Status**: Testing  
**Owner**: AI Agent - Research Intelligence Module  
**Review Cycle**: Monthly validation against test suite