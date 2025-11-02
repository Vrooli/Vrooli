# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Text-tools provides a comprehensive, reusable text processing and analysis platform that all other scenarios can leverage for text manipulation, search, extraction, transformation, and intelligence operations. It serves as the central text processing hub, eliminating duplication across scenarios and providing consistent, high-quality text operations.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Text-tools amplifies agent intelligence by:
- Providing semantic search and similarity matching that helps agents understand conceptual relationships
- Offering diff and comparison tools that enable agents to learn from changes and iterations
- Supporting pattern extraction and entity recognition that helps agents identify important information
- Enabling text transformation pipelines that can be learned and optimized over time
- Creating a shared knowledge base of text processing patterns that all agents can access

### Recursive Value
**What new scenarios become possible after this exists?**
1. **code-review-assistant**: Can leverage diff, syntax highlighting, and semantic analysis
2. **document-version-control**: Can use diff, merge, and transformation capabilities
3. **contract-analyzer**: Can extract entities, compare versions, identify key terms
4. **content-moderator**: Can use sentiment analysis, entity extraction, pattern matching
5. **translation-hub**: Can build on text extraction and transformation infrastructure

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Diff and comparison tools (line-by-line, word-by-word, semantic) ✅ 2025-09-30
  - [x] Search and pattern matching (grep, regex, fuzzy search) ✅ 2025-09-30
  - [x] Text transformation (case, encoding, format conversion) ✅ 2025-09-30
  - [x] Text extraction from multiple formats (PDF, HTML, DOCX, images via OCR) ✅ 2025-09-30
  - [x] Basic statistics (word count, character count, language detection) ✅ 2025-09-30
  - [x] RESTful API for all core operations ✅ 2025-09-30
  - [x] CLI interface with full feature parity ✅ 2025-09-30
  
- **Should Have (P1)**
  - [x] Semantic search using embeddings (via Ollama) ✅ 2025-09-30
  - [x] Text summarization and condensation ✅ 2025-09-30
  - [x] Entity extraction (names, dates, locations, emails, URLs) ✅ 2025-09-30
  - [x] Sentiment and tone analysis ✅ 2025-09-30
  - [ ] Multi-language support and translation
  - [ ] Batch processing for multiple files
  - [x] Text sanitization (PII removal, HTML cleaning) ✅ 2025-09-30
  
- **Nice to Have (P2)**
  - [ ] Template engine with variables and conditionals
  - [ ] Advanced NLP operations (topic modeling, keyword extraction)
  - [ ] Text generation (test data, lorem ipsum, mockups)
  - [ ] Visual diff interface with side-by-side comparison
  - [ ] Pipeline builder for chaining operations
  - [ ] Version history and change tracking

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 100ms for basic ops, < 500ms for NLP | API monitoring |
| Throughput | 1000 operations/second | Load testing |
| Accuracy | > 95% for entity extraction | Validation suite |
| Resource Usage | < 2GB memory, < 30% CPU | System monitoring |
| File Size Support | Up to 100MB per file | Integration tests |

### Quality Gates
- [x] All P0 requirements implemented and tested ✅ 2025-09-30
- [x] Integration tests pass with Ollama, PostgreSQL, MinIO ✅ 2025-09-30
- [ ] Performance targets met under load
- [x] Documentation complete (README, API docs, CLI help) ✅ 2025-09-30
- [x] Scenario can be invoked by other agents via API/CLI ✅ 2025-09-30
- [ ] At least 3 other scenarios successfully integrated

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store text history, templates, and processing metadata
    integration_pattern: Database for persistence
    access_method: resource-postgres CLI commands
    
  - resource_name: minio
    purpose: Store large text files and processing results
    integration_pattern: Object storage for files
    access_method: resource-minio CLI commands
    
optional:
  - resource_name: ollama
    purpose: Enable semantic search, summarization, and NLP features
    fallback: Basic operations work without AI features
    access_method: initialization/n8n/ollama.json workflow
    
  - resource_name: redis
    purpose: Cache frequently accessed text and processing results
    fallback: Direct processing without cache
    access_method: resource-redis CLI commands
    
  - resource_name: qdrant
    purpose: Vector storage for semantic search
    fallback: Use PostgreSQL pgvector if available
    access_method: resource-qdrant CLI commands
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: Text summarization, translation, and NLP operations
    - workflow: embedding-generator.json
      location: initialization/n8n/
      purpose: Generate embeddings for semantic search
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-postgres query
      purpose: Store and retrieve text metadata
    - command: resource-minio upload/download
      purpose: Handle large text files
    - command: resource-redis get/set
      purpose: Cache processing results
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: Real-time streaming for large file processing
      endpoint: MinIO streaming API for progressive text processing

# Shared workflow guidelines:
shared_workflow_criteria:
  - Text processing workflows will be made generic and reusable
  - Place in initialization/n8n/ for cross-scenario use
  - Document all scenarios that depend on text-tools
  - Provide fallback for offline text processing
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: TextDocument
    storage: postgres + minio
    schema: |
      {
        id: UUID
        name: string
        content_hash: string
        format: enum(plain, markdown, html, pdf, docx)
        size_bytes: integer
        language: string
        created_at: timestamp
        modified_at: timestamp
        minio_path: string  # For large files
        metadata: jsonb
      }
    relationships: Links to TextOperation history
    
  - name: TextOperation
    storage: postgres
    schema: |
      {
        id: UUID
        document_id: UUID
        operation_type: enum(diff, search, transform, extract, analyze)
        parameters: jsonb
        result_summary: jsonb
        result_path: string  # MinIO path for large results
        duration_ms: integer
        created_at: timestamp
      }
    relationships: Many operations per document
    
  - name: TextTemplate
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        content: text
        variables: jsonb
        description: text
        usage_count: integer
        created_at: timestamp
      }
    relationships: Used by multiple scenarios
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/text/diff
    purpose: Compare two texts and return differences
    input_schema: |
      {
        text1: string | {url: string} | {document_id: UUID}
        text2: string | {url: string} | {document_id: UUID}
        options: {
          type: "line" | "word" | "character" | "semantic"
          ignore_whitespace: boolean
          ignore_case: boolean
        }
      }
    output_schema: |
      {
        changes: [{
          type: "add" | "remove" | "modify"
          line_start: number
          line_end: number
          content: string
        }]
        similarity_score: float
        summary: string
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/text/search
    purpose: Search for patterns in text
    input_schema: |
      {
        text: string | {document_id: UUID}
        pattern: string  # regex or plain text
        options: {
          regex: boolean
          case_sensitive: boolean
          whole_word: boolean
          fuzzy: boolean
          semantic: boolean  # requires Ollama
        }
      }
    output_schema: |
      {
        matches: [{
          line: number
          column: number
          length: number
          context: string
          score: float  # for fuzzy/semantic
        }]
        total_matches: number
      }
      
  - method: POST
    path: /api/v1/text/transform
    purpose: Transform text format or encoding
    input_schema: |
      {
        text: string
        transformations: [{
          type: "case" | "encode" | "format" | "sanitize"
          parameters: object
        }]
      }
    output_schema: |
      {
        result: string
        transformations_applied: array
        warnings: array
      }
      
  - method: POST
    path: /api/v1/text/extract
    purpose: Extract text from various formats
    input_schema: |
      {
        source: {url: string} | {file: base64} | {document_id: UUID}
        format: "pdf" | "html" | "docx" | "image" | "auto"
        options: {
          ocr: boolean
          preserve_formatting: boolean
          extract_metadata: boolean
        }
      }
    output_schema: |
      {
        text: string
        metadata: {
          format: string
          pages: number
          author: string
          created: timestamp
          language: string
        }
        warnings: array
      }
      
  - method: POST
    path: /api/v1/text/analyze
    purpose: Perform NLP analysis on text
    input_schema: |
      {
        text: string
        analyses: ["entities", "sentiment", "summary", "keywords", "language"]
        options: {
          summary_length: number
          entity_types: array
        }
      }
    output_schema: |
      {
        entities: [{type: string, value: string, confidence: float}]
        sentiment: {score: float, label: string}
        summary: string
        keywords: [{word: string, score: float}]
        language: {code: string, confidence: float}
      }
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: text.document.created
    payload: {document_id: UUID, format: string, size: number}
    subscribers: [document-manager, search-indexer]
    
  - name: text.analysis.completed
    payload: {document_id: UUID, analysis_type: string, results: object}
    subscribers: [knowledge-observatory, research-assistant]
    
  - name: text.diff.detected
    payload: {doc1_id: UUID, doc2_id: UUID, change_count: number}
    subscribers: [version-control, change-monitor]
    
consumed_events:
  - name: document.uploaded
    action: Automatically extract and index text content
    
  - name: translation.requested
    action: Process translation via Ollama integration
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: text-tools
install_script: cli/install.sh

# Core commands that MUST be implemented:
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

# Scenario-specific commands:
custom_commands:
  - name: diff
    description: Compare two text files or strings
    api_endpoint: /api/v1/text/diff
    arguments:
      - name: file1
        type: string
        required: true
        description: First file path or '-' for stdin
      - name: file2
        type: string
        required: true
        description: Second file path or '-' for stdin
    flags:
      - name: --type
        description: Diff algorithm (line, word, char, semantic)
      - name: --ignore-whitespace
        description: Ignore whitespace differences
      - name: --output
        description: Output format (unified, side-by-side, json)
    output: Diff results in specified format
    
  - name: search
    description: Search for patterns in text
    api_endpoint: /api/v1/text/search
    arguments:
      - name: pattern
        type: string
        required: true
        description: Search pattern (regex or plain text)
      - name: files
        type: string
        required: false
        description: File paths to search (default stdin)
    flags:
      - name: --regex
        description: Use regex pattern matching
      - name: --fuzzy
        description: Enable fuzzy matching
      - name: --semantic
        description: Use semantic search (requires Ollama)
    
  - name: transform
    description: Transform text format or encoding
    api_endpoint: /api/v1/text/transform
    arguments:
      - name: input
        type: string
        required: true
        description: Input file or '-' for stdin
    flags:
      - name: --upper
        description: Convert to uppercase
      - name: --lower
        description: Convert to lowercase
      - name: --base64
        description: Encode/decode base64
      - name: --format
        description: Format as json, xml, yaml
      
  - name: extract
    description: Extract text from various formats
    api_endpoint: /api/v1/text/extract
    arguments:
      - name: source
        type: string
        required: true
        description: Source file path or URL
    flags:
      - name: --ocr
        description: Use OCR for images
      - name: --format
        description: Source format (auto-detect if not specified)
      
  - name: analyze
    description: Perform NLP analysis on text
    api_endpoint: /api/v1/text/analyze
    arguments:
      - name: input
        type: string
        required: true
        description: Input file or '-' for stdin
    flags:
      - name: --entities
        description: Extract named entities
      - name: --sentiment
        description: Analyze sentiment
      - name: --summary
        description: Generate summary
      - name: --keywords
        description: Extract keywords
```

### CLI-API Parity Requirements
- **Coverage**: Every API endpoint has a corresponding CLI command
- **Naming**: CLI uses intuitive verb-noun patterns
- **Arguments**: Direct mapping to API parameters
- **Output**: Human-readable by default, --json for machines
- **Piping**: Support Unix pipes for text processing chains

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **PostgreSQL**: Database for metadata and small text storage
- **MinIO**: Object storage for large files
- **Ollama (optional)**: AI-powered text analysis and generation
- **initialization/n8n/ollama.json**: Shared workflow for AI operations

### Downstream Enablement
**What future capabilities does this unlock?**
- **document-version-control**: Full document versioning with diff visualization
- **code-review-assistant**: Automated code review with semantic understanding
- **contract-analyzer**: Legal document analysis and comparison
- **content-moderator**: Automated content moderation and filtering
- **translation-hub**: Multi-language translation service

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: notes
    capability: Diff and version tracking for notes
    interface: API/CLI
    
  - scenario: research-assistant
    capability: Text extraction and summarization
    interface: API/Events
    
  - scenario: code-smell
    capability: Pattern matching and code analysis
    interface: CLI
    
  - scenario: document-manager
    capability: Full text extraction and indexing
    interface: API/Events
    
consumes_from:
  - scenario: none  # This is a foundational service
    capability: n/a
    fallback: n/a
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Modern code editors and diff tools (VS Code, GitHub)
  
  visual_style:
    color_scheme: dark  # Developer-friendly dark theme
    typography: monospace for code/text, system font for UI
    layout: dense  # Maximum information density
    animations: subtle  # Only for state transitions
  
  personality:
    tone: technical
    mood: focused
    target_feeling: Powerful and precise

# UI Components:
ui_elements:
  - Split-pane diff viewer with syntax highlighting
  - Real-time search with highlighting
  - Pipeline builder with drag-and-drop
  - Text statistics dashboard
  - Format converter with preview
```

### Target Audience Alignment
- **Primary Users**: Developers, data analysts, content creators
- **User Expectations**: Fast, accurate, command-line friendly
- **Accessibility**: WCAG AA compliance, keyboard navigation
- **Responsive Design**: Desktop-first, mobile-readable

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates need for multiple text processing tools
- **Revenue Potential**: $10K - $30K per enterprise deployment
- **Cost Savings**: 70% reduction in text processing tool licenses
- **Market Differentiator**: Unified text processing with AI enhancement

### Technical Value
- **Reusability Score**: 10/10 - Every scenario needs text processing
- **Complexity Reduction**: Single API for all text operations
- **Innovation Enablement**: Foundation for document intelligence platform

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core text operations (diff, search, transform, extract)
- Basic API and CLI
- PostgreSQL and MinIO integration

### Version 2.0 (Planned)
- Full Ollama integration for AI features
- Visual pipeline builder
- Real-time collaboration on text operations
- Plugin system for custom transformations

### Long-term Vision
- Become the "ImageMagick of text" for Vrooli
- Self-learning optimization of processing pipelines
- Cross-language semantic understanding

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with resource dependencies
    - PostgreSQL schema initialization
    - N8n workflow registration
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose
    - kubernetes: Helm chart
    - cloud: Serverless functions for API
    
  revenue_model:
    - type: usage-based
    - pricing_tiers:
        - free: 1000 operations/month
        - pro: 100,000 operations/month
        - enterprise: unlimited
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: text-tools
    category: foundation
    capabilities: [diff, search, transform, extract, analyze]
    interfaces:
      - api: http://localhost:${TEXT_TOOLS_PORT}/api/v1
      - cli: text-tools
      - events: text.*
      
  metadata:
    description: Comprehensive text processing and analysis platform
    keywords: [text, diff, search, NLP, extraction, transformation]
    dependencies: []
    enhances: [all text-processing scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Large file processing OOM | Medium | High | Stream processing, MinIO offload |
| Ollama unavailability | Medium | Medium | Graceful degradation to basic ops |
| OCR accuracy issues | Low | Medium | Multiple OCR engines, confidence scores |

### Operational Risks
- **Resource Conflicts**: Managed through service.json priorities
- **API Rate Limiting**: Redis caching and request throttling
- **Data Privacy**: PII detection and optional redaction

## ✅ Validation Criteria

### Declarative Test Specification

- **Phased testing**: `test/run-tests.sh` orchestrates structure, dependencies, unit, integration, business, and performance phases through the shared `testing::runner` shell library (runtime management, caching, presets, and artifact handling).
- **CLI coverage**: `test/cli/run-cli-tests.sh` executes the BATS suite (`cli/text-tools.bats`) via the runner’s `cli` test type, skipping safely when prerequisites such as `bats` or the CLI binary are absent.
- **Lifecycle integration**: `.vrooli/service.json` maps the test lifecycle to `test/run-tests.sh`; locally run `./test/run-tests.sh comprehensive` or lighter presets like `./test/run-tests.sh quick` for fast feedback.

## 📝 Implementation Notes

### Design Decisions
**Plugin Architecture**: Modular plugin system for extensibility
- Alternative considered: Monolithic implementation
- Decision driver: Need for custom transformations per scenario
- Trade-offs: Slight complexity for massive flexibility

**Streaming Processing**: Stream large files instead of loading to memory
- Alternative considered: Load entire files
- Decision driver: Support for 100MB+ files
- Trade-offs: More complex implementation for better scalability

### Known Limitations
- **OCR Accuracy**: ~90% accuracy on handwritten text
  - Workaround: Confidence scores and manual review option
  - Future fix: Multiple OCR engine voting system

### Security Considerations
- **Data Protection**: Optional encryption at rest in MinIO
- **Access Control**: API key authentication
- **Audit Trail**: All operations logged with user context

## 🔗 References

### Documentation
- README.md - Quick start guide
- docs/api.md - Complete API reference
- docs/cli.md - CLI usage examples
- docs/plugins.md - Plugin development guide

### Related PRDs
- scenarios/image-tools/PRD.md - Sister service for images
- scenarios/document-manager/PRD.md - Consumes text-tools

---

**Last Updated**: 2025-09-30
**Status**: Implementation Complete (P0: 100%, P1: 71%)
**Owner**: AI Agent
**Review Cycle**: Weekly validation against implementation

## Progress History
- **2025-09-30**: Fixed compilation errors, transform endpoint case conversion, validated all P0 and most P1 features working
