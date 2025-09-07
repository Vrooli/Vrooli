# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Data-structurer adds the fundamental capability to transform any unstructured data (text, PDFs, images, documents) into clean, schema-defined structured data stored in PostgreSQL. This creates a universal data ingestion and normalization pipeline that other scenarios can leverage to handle messy real-world data inputs.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability eliminates the "data preparation bottleneck" that limits most AI applications. Future agents no longer need to build custom parsers for each data format - they can simply define their desired schema and feed any unstructured input through data-structurer. This transforms agents from "format-specific" to "truly general-purpose" intelligence systems.

### Recursive Value
**What new scenarios become possible after this exists?**
- **email-outreach-manager** can ingest prospect lists from any source (web scraping, CSV files, business cards, LinkedIn profiles)
- **competitor-change-monitor** can structure competitor data from websites, press releases, SEC filings, and social media
- **resume-screening-assistant** can process resumes in any format (PDF, DOC, text, LinkedIn profiles, handwritten notes)
- **research-assistant** can structure research data from academic papers, web articles, interview transcripts, and survey responses
- **financial-calculators-hub** can ingest financial data from bank statements, invoices, receipts, and spreadsheets

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Accept unstructured inputs (text, PDF, images, DOCX, HTML) and convert to structured JSON/YAML
  - [ ] CRUD operations for schema management (create, read, update, delete data schemas)
  - [ ] Processing pipeline: unstructured-io → ollama interpretation → postgres storage
  - [ ] REST API for data submission and retrieval with structured responses
  - [ ] CLI interface mirroring all API functionality
  - [ ] PostgreSQL storage with dynamic table creation based on schemas
  
- **Should Have (P1)**
  - [ ] Batch processing capability for multiple files
  - [ ] Data validation and error correction using Ollama feedback loops
  - [ ] Export functionality (JSON, CSV, YAML formats)
  - [ ] Schema templates for common data types (contacts, products, events, documents)
  - [ ] Integration with Qdrant for semantic search on structured data
  - [ ] Confidence scoring for extracted data quality
  
- **Nice to Have (P2)**
  - [ ] Real-time processing via websocket API
  - [ ] Schema inference from example data
  - [ ] Data enrichment using external APIs
  - [ ] Visual schema builder UI component
  - [ ] Automated data cleaning and deduplication
  - [ ] Multi-language support for international documents

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Processing Time | < 5s for single document, < 30s for batch | API timing logs |
| Throughput | 50 documents/minute concurrent processing | Load testing |
| Accuracy | > 95% field extraction accuracy for structured documents | Validation suite |
| Resource Usage | < 4GB memory, < 80% CPU during processing | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with postgres, ollama, and unstructured-io
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Storage of structured data with dynamic table creation
    integration_pattern: CLI via resource-postgres
    access_method: resource-postgres exec, resource-postgres query
    
  - resource_name: ollama
    purpose: Content interpretation and schema mapping intelligence
    integration_pattern: Shared workflow (ollama.json)
    access_method: initialization/n8n/ollama.json workflow
    
  - resource_name: unstructured-io
    purpose: Extract raw content from PDFs, images, documents
    integration_pattern: CLI via resource-unstructured-io
    access_method: resource-unstructured-io process
    
optional:
  - resource_name: qdrant
    purpose: Semantic search and similarity matching of structured data
    fallback: Basic text search via PostgreSQL full-text search
    access_method: resource-qdrant insert, resource-qdrant search
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: Reliable Ollama inference for schema mapping and data interpretation
  
  2_resource_cli:
    - command: resource-unstructured-io process --input [file] --output json
      purpose: Extract raw content from any document format
    - command: resource-postgres exec --query [sql]
      purpose: Dynamic table creation and data storage operations
  
  3_direct_api:
    - justification: N/A - All resources accessible via CLI or shared workflows
```

### Data Models
```yaml
primary_entities:
  - name: Schema
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        description: string,
        schema_definition: JSONB,
        created_at: timestamp,
        updated_at: timestamp,
        version: integer
      }
    relationships: One-to-many with ProcessedData
    
  - name: ProcessedData
    storage: postgres
    schema: |
      {
        id: UUID,
        schema_id: UUID (foreign key),
        source_file: string,
        raw_content: text,
        structured_data: JSONB,
        confidence_score: float,
        processing_status: enum(pending, processing, completed, failed),
        created_at: timestamp,
        metadata: JSONB
      }
    relationships: Belongs-to Schema
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/schemas
    purpose: Create new data schema definition
    input_schema: |
      {
        name: string,
        description: string,
        schema_definition: object,
        example_data: object (optional)
      }
    output_schema: |
      {
        id: UUID,
        name: string,
        status: "created"
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/process
    purpose: Process unstructured data into structured format
    input_schema: |
      {
        schema_id: UUID,
        input_type: enum(file, text, url),
        input_data: string|base64,
        batch_mode: boolean (optional)
      }
    output_schema: |
      {
        processing_id: UUID,
        status: "processing|completed",
        structured_data: object,
        confidence_score: float,
        errors: array (optional)
      }
    sla:
      response_time: 5000ms
      availability: 99.5%
      
  - method: GET
    path: /api/v1/data/{schema_id}
    purpose: Retrieve all structured data for a schema
    output_schema: |
      {
        schema: object,
        data: array,
        total_count: integer,
        pagination: object
      }
    sla:
      response_time: 500ms
      availability: 99.9%
```

### Event Interface
```yaml
published_events:
  - name: data-structurer.processing.started
    payload: {processing_id: UUID, schema_id: UUID, input_size: integer}
    subscribers: [monitoring systems, dependent scenarios]
    
  - name: data-structurer.processing.completed
    payload: {processing_id: UUID, structured_data: object, confidence_score: float}
    subscribers: [email-outreach-manager, research-assistant, other data consumers]
    
consumed_events:
  - name: schema.updated
    action: Revalidate existing data against updated schema
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: data-structurer
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
  - name: create-schema
    description: Create a new data schema
    api_endpoint: /api/v1/schemas
    arguments:
      - name: name
        type: string
        required: true
        description: Schema name
      - name: definition
        type: string
        required: true
        description: Path to JSON/YAML schema file
    flags:
      - name: --description
        description: Human-readable schema description
    output: Schema ID and confirmation
    
  - name: process
    description: Process unstructured data
    api_endpoint: /api/v1/process
    arguments:
      - name: schema-id
        type: string
        required: true
        description: UUID of target schema
      - name: input
        type: string
        required: true
        description: Path to input file or text
    flags:
      - name: --batch
        description: Process multiple files
      - name: --output-format
        description: json|csv|yaml output format
    output: Structured data result
    
  - name: list-schemas
    description: List all available schemas
    api_endpoint: /api/v1/schemas
    flags:
      - name: --detailed
        description: Include schema definitions
    output: Schema list with metadata
    
  - name: get-data
    description: Retrieve structured data
    api_endpoint: /api/v1/data/{schema_id}
    arguments:
      - name: schema-id
        type: string
        required: true
        description: Schema UUID
    flags:
      - name: --format
        description: Output format (json|csv|table)
      - name: --limit
        description: Number of records to return
    output: Structured data in specified format
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **PostgreSQL Resource**: Database storage for schemas and structured data
- **Ollama Resource**: AI inference for content interpretation and schema mapping
- **Unstructured-io Resource**: Document content extraction from various formats
- **Shared Ollama Workflow**: Reliable AI inference pattern via initialization/n8n/ollama.json

### Downstream Enablement
**What future capabilities does this unlock?**
- **Universal Data Ingestion**: All scenarios can handle any input format
- **Schema-Driven Development**: New scenarios can focus on business logic instead of data parsing
- **Cross-Scenario Data Sharing**: Structured data becomes a common language between scenarios
- **Compound Intelligence**: Each scenario's data processing becomes available to all others

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: email-outreach-manager
    capability: Prospect list processing from any format
    interface: API/CLI
  - scenario: research-assistant
    capability: Document structuring and knowledge extraction
    interface: API/CLI
  - scenario: competitor-change-monitor
    capability: Competitor data normalization from various sources
    interface: API/CLI
    
consumes_from:
  - scenario: N/A (foundational capability)
    capability: N/A
    fallback: N/A
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: "Professional data tool with clear visual hierarchy"
  
  visual_style:
    color_scheme: light
    typography: modern
    layout: dashboard
    animations: subtle
  
  personality:
    tone: technical
    mood: focused
    target_feeling: "Confidence in data accuracy and reliability"

style_references:
  technical:
    - system-monitor: "Clean dashboard with status indicators"
    - agent-dashboard: "Data-focused interface with clear metrics"
```

### Target Audience Alignment
- **Primary Users**: AI agents and technical users needing data transformation
- **User Expectations**: Fast, reliable, accurate data conversion with clear error reporting
- **Accessibility**: WCAG AA compliance for screen readers and high contrast
- **Responsive Design**: Desktop-first for complex schema management, mobile for status monitoring

### Brand Consistency Rules
- **Scenario Identity**: Professional data processing tool with emphasis on accuracy
- **Vrooli Integration**: Consistent with technical scenarios in the ecosystem
- **Professional vs Fun**: Professional design - this is infrastructure other scenarios depend on

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates 70-80% of development time for data-dependent scenarios
- **Revenue Potential**: $15K - $40K per deployment (saves months of custom parsing development)
- **Cost Savings**: Reduces scenario development time from weeks to days
- **Market Differentiator**: First universal data structuring pipeline with AI interpretation

### Technical Value
- **Reusability Score**: 10/10 - Every data-dependent scenario benefits
- **Complexity Reduction**: Complex document parsing becomes single API call
- **Innovation Enablement**: Scenarios can focus on business logic instead of data wrangling

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core CRUD operations for schemas
- Single document processing via unstructured-io + ollama
- PostgreSQL storage with basic CLI/API

### Version 2.0 (Planned)
- Batch processing and queue management
- Qdrant integration for semantic search
- Schema templates and inference
- Performance optimizations

### Long-term Vision
- Real-time streaming data processing
- Multi-modal AI for complex document understanding
- Autonomous schema evolution based on data patterns
- Integration hub for external data APIs

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - .vrooli/service.json with postgres, ollama, unstructured-io dependencies
    - Complete initialization files for database schema
    - Startup scripts for API and worker processes
    - Health check endpoints for all components
    
  deployment_targets:
    - local: Docker Compose with all resource dependencies
    - kubernetes: Helm chart with PVC for postgres data
    - cloud: AWS/GCP templates with managed database options
    
  revenue_model:
    - type: subscription
    - pricing_tiers: Based on data processing volume (documents per month)
    - trial_period: 14 days with 100 document limit
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: data-structurer
    category: automation
    capabilities: [data-transformation, schema-management, document-processing, ai-interpretation]
    interfaces:
      - api: /api/v1/
      - cli: data-structurer
      - events: data-structurer.*
      
  metadata:
    description: Transform unstructured data into schema-defined structured data
    keywords: [data, schema, transformation, ai, documents, extraction]
    dependencies: [postgres, ollama, unstructured-io]
    enhances: [email-outreach-manager, research-assistant, competitor-change-monitor]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Ollama inference failure | Medium | High | Fallback to simpler extraction rules, retry logic |
| Large document processing timeout | High | Medium | Chunking strategy, async processing queue |
| Schema migration complexity | Low | High | Version control for schemas, migration scripts |
| PostgreSQL table sprawl | Medium | Medium | Table naming conventions, cleanup procedures |

### Operational Risks
- **Data Quality Drift**: Continuous validation against known-good examples
- **Schema Evolution**: Backward compatibility requirements and migration paths
- **Resource Scaling**: Auto-scaling based on processing queue depth
- **Privacy/Security**: Data sanitization and audit logging for sensitive content

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: data-structurer

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/data-structurer
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/n8n/data-processing.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - tests

resources:
  required: [postgres, ollama, unstructured-io]
  optional: [qdrant]
  health_timeout: 90

tests:
  - name: "PostgreSQL is accessible and schema exists"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Ollama is responding to inference requests"
    type: http
    service: ollama
    endpoint: /api/generate
    method: POST
    body:
      model: llama2
      prompt: "test"
    expect:
      status: 200
      
  - name: "Schema creation API works"
    type: http
    service: api
    endpoint: /api/v1/schemas
    method: POST
    body:
      name: "test_schema"
      description: "Test schema for validation"
      schema_definition: {"name": "string", "age": "integer"}
    expect:
      status: 201
      body:
        name: "test_schema"
        
  - name: "Document processing API works"
    type: http
    service: api
    endpoint: /api/v1/process
    method: POST
    body:
      schema_id: "test_uuid"
      input_type: "text"
      input_data: "John Doe is 25 years old"
    expect:
      status: 200
      body:
        status: "completed"
        
  - name: "CLI status command works"
    type: exec
    command: ./cli/data-structurer status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
```

### Performance Validation
- [ ] Process single PDF document < 5 seconds
- [ ] Handle 50 concurrent requests without degradation
- [ ] Memory usage stays under 4GB during batch processing
- [ ] Database queries return results < 500ms for 10k records

### Integration Validation
- [ ] All API endpoints return proper HTTP status codes
- [ ] CLI commands mirror API functionality exactly
- [ ] Shared ollama.json workflow processes requests successfully
- [ ] PostgreSQL schema creation and table management works
- [ ] Error handling provides actionable feedback

### Capability Verification
- [ ] Successfully extracts data from PDF, DOCX, and image files
- [ ] Schema-based validation catches malformed data
- [ ] Structured data is queryable via standard SQL
- [ ] Other scenarios can integrate via documented API
- [ ] Confidence scoring accurately reflects extraction quality

## 📝 Implementation Notes

### Design Decisions
**Processing Architecture**: Chose async queue-based processing over real-time
- Alternative considered: Synchronous processing for immediate results
- Decision driver: Large documents require significant processing time, blocking API calls harm user experience
- Trade-offs: Added complexity of job queuing for better scalability and user experience

**Schema Storage**: PostgreSQL JSONB for schema definitions instead of separate schema service
- Alternative considered: Dedicated schema registry service
- Decision driver: Simpler deployment, leverages existing PostgreSQL resource
- Trade-offs: Less specialized tooling for advanced schema capabilities

### Known Limitations
- **Large File Processing**: Files > 50MB may timeout or consume excessive memory
  - Workaround: Chunking strategy for large documents, processing in segments
  - Future fix: Streaming processing pipeline in v2.0
- **Complex Layout Documents**: Advanced formatting (tables, charts) may not extract perfectly
  - Workaround: Manual review mode for high-stakes documents
  - Future fix: Multi-modal AI integration for visual understanding

### Security Considerations
- **Data Protection**: All processed data encrypted at rest in PostgreSQL
- **Access Control**: API key authentication required for all endpoints
- **Audit Trail**: All schema changes and processing operations logged with timestamps and user identification

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start guide
- docs/api.md - Complete API specification with examples
- docs/cli.md - CLI command reference and usage patterns
- docs/architecture.md - Technical deep-dive on processing pipeline

### Related PRDs
- Email-outreach-manager PRD - Primary consumer of prospect structuring capability
- Research-assistant PRD - Document processing integration patterns
- System-monitor PRD - Reference for technical dashboard style

### External Resources
- [Unstructured.io Documentation](https://unstructured-io.github.io/unstructured/) - Document processing capabilities
- [Ollama API Reference](https://github.com/ollama/ollama/blob/main/docs/api.md) - AI inference integration
- [PostgreSQL JSONB Documentation](https://www.postgresql.org/docs/current/datatype-json.html) - Schema storage approach

---

**Last Updated**: 2025-09-06  
**Status**: Draft  
**Owner**: Claude Code AI Agent  
**Review Cycle**: Weekly validation against implementation progress