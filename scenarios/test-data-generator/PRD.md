# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
This scenario provides a universal test data generation service that creates deterministic or random mock data in multiple formats (JSON, YAML, CSV, XML, images, audio, video, PDFs). It eliminates repository bloat from committed test files and enables dynamic, on-demand test data generation for any scenario's testing needs.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Agents can now request exactly the test data they need without human intervention - specific file formats, sizes, schemas, or content patterns. This enables agents to autonomously write comprehensive tests, validate edge cases, and build more robust applications. The seed-based generation ensures reproducible testing across iterations.

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Automated test suite generators** - Agents can create comprehensive test suites with all necessary mock data
2. **Performance benchmarking tools** - Generate consistent datasets for load testing and optimization
3. **Data migration validators** - Create source data in various formats to test transformation pipelines
4. **AI training data generators** - Produce structured datasets for model training and evaluation
5. **Demo environment builders** - Instantly populate applications with realistic sample data

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Generate JSON with customizable schemas and faker data
  - [ ] Generate YAML, CSV, and XML formats
  - [ ] Generate placeholder images (PNG, JPG, WebP) with specified dimensions
  - [ ] Generate audio files (WAV, MP3) with sine waves or white noise
  - [ ] Generate video files (MP4, WebM) with animated patterns
  - [ ] Generate PDF documents with lorem ipsum content
  - [ ] Support deterministic generation via seed parameter
  - [ ] RESTful API with comprehensive endpoints
  - [ ] CLI with full API parity
  - [ ] Batch generation for multiple files
  
- **Should Have (P1)**
  - [ ] Custom schema validation for structured data
  - [ ] Template library for common data patterns
  - [ ] Streaming support for large file generation
  - [ ] Caching layer for frequently requested patterns
  - [ ] Text-to-speech for audio generation
  - [ ] DOCX document generation
  - [ ] Compression options (ZIP archives)
  
- **Nice to Have (P2)**
  - [ ] QR code and barcode generation
  - [ ] Complex video patterns (moving objects, transitions)
  - [ ] Database dump formats (SQL, MongoDB exports)
  - [ ] Binary file generation (custom protocols)
  - [ ] Integration with Faker.js locales for internationalization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for files under 10MB | API monitoring |
| Throughput | 100 files/second for small files | Load testing |
| Memory Usage | < 512MB for standard operations | System monitoring |
| Cache Hit Rate | > 80% for common patterns | Cache metrics |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [x] Documentation complete (README, API docs, CLI help)
- [x] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  # None - this is a foundational service with no hard dependencies
  
optional:
  - resource_name: redis
    purpose: Caching frequently requested patterns
    fallback: In-memory cache with size limits
    access_method: resource-redis CLI commands
    
  - resource_name: minio
    purpose: Store generated files for later retrieval
    fallback: Temporary filesystem storage
    access_method: resource-minio CLI commands
```

### Resource Integration Standards
```yaml
integration_priorities:
  2_resource_cli:
    - command: resource-redis set/get for caching
      purpose: Cache frequently generated patterns
    
  3_direct_api:
    - justification: File generation requires direct Node.js libraries
      libraries: faker.js, canvas, ffmpeg bindings
```

### Data Models
```yaml
primary_entities:
  - name: GenerationRequest
    storage: In-memory/Redis
    schema: |
      {
        id: UUID,
        type: string, // json|yaml|csv|xml|image|audio|video|pdf
        format: string, // specific format variant
        options: object, // type-specific options
        seed: number?, // optional seed for determinism
        createdAt: timestamp
      }
    relationships: Links to cached results if seed matches
    
  - name: Template
    storage: Filesystem
    schema: |
      {
        name: string,
        type: string,
        description: string,
        schema: object,
        defaults: object
      }
    relationships: Used by generation requests
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/generate/json
    purpose: Generate JSON data with schema or faker patterns
    input_schema: |
      {
        schema?: object,  // JSON schema definition
        count?: number,   // Number of records
        seed?: number,    // Deterministic seed
        faker?: object    // Faker.js template
      }
    output_schema: |
      {
        data: array|object,
        metadata: {
          generated: number,
          seed: number,
          cacheable: boolean
        }
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/generate/image
    purpose: Generate placeholder images
    input_schema: |
      {
        width: number,
        height: number,
        format: "png"|"jpg"|"webp",
        type?: "solid"|"gradient"|"pattern"|"placeholder",
        seed?: number,
        text?: string  // Overlay text
      }
    output_schema: |
      {
        url: string,  // Download URL
        size: number, // File size in bytes
        dimensions: { width: number, height: number }
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/generate/batch
    purpose: Generate multiple files in one request
    input_schema: |
      {
        requests: array, // Array of generation requests
        archive?: boolean // Return as ZIP
      }
    output_schema: |
      {
        files: array,
        archive_url?: string
      }
    sla:
      response_time: 5000ms
      availability: 99.9%
```

### Event Interface
```yaml
published_events:
  - name: test-data.generation.completed
    payload: { type: string, format: string, size: number, cached: boolean }
    subscribers: Test runners, CI/CD pipelines
    
  - name: test-data.cache.hit
    payload: { pattern: string, hit_rate: number }
    subscribers: Performance monitors
    
consumed_events:
  - name: test.suite.started
    action: Pre-warm cache with common patterns
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: test-data-generator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show service status and statistics
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version
    flags: [--json]

custom_commands:
  - name: generate
    description: Generate test data of specified type
    api_endpoint: /api/v1/generate/{type}
    arguments:
      - name: type
        type: string
        required: true
        description: Data type to generate (json|yaml|csv|xml|image|audio|video|pdf)
    flags:
      - name: --output
        description: Output file path (default: stdout for text, temp file for binary)
      - name: --count
        description: Number of records for structured data
      - name: --seed
        description: Seed for deterministic generation
      - name: --schema
        description: Path to schema file for structured data
      - name: --width
        description: Width for images/video
      - name: --height  
        description: Height for images/video
      - name: --duration
        description: Duration for audio/video in seconds
    output: File content or path to generated file
    
  - name: batch
    description: Generate multiple files from specification
    api_endpoint: /api/v1/generate/batch
    arguments:
      - name: spec
        type: string
        required: true
        description: Path to batch specification file
    flags:
      - name: --output-dir
        description: Directory for generated files
      - name: --archive
        description: Create ZIP archive of results
    output: List of generated files
    
  - name: templates
    description: List or use predefined templates
    api_endpoint: /api/v1/templates
    flags:
      - name: --list
        description: List available templates
      - name: --use
        description: Generate using template name
    output: Template list or generated data
```

### CLI-API Parity Requirements
- **Coverage**: Every generation type available via API is accessible through CLI
- **Naming**: CLI uses kebab-case (test-data-generator generate --type json)
- **Arguments**: Direct mapping between CLI flags and API parameters
- **Output**: Supports stdout for text, file paths for binary, JSON with --json flag
- **Piping**: Text formats support Unix pipes for chaining commands

### Implementation Standards
```yaml
implementation_requirements:
  architecture: Express server with generation libraries
  language: Node.js for consistency with UI
  dependencies:
    - faker.js for realistic data
    - canvas for image generation
    - fluent-ffmpeg for audio/video
    - pdfkit for PDF generation
  error_handling:
    - 0: Success
    - 1: General error
    - 2: Invalid arguments
    - 3: Generation failed
  configuration:
    - Read from ~/.vrooli/test-data-generator/config.yaml
    - Environment variables: TEST_DATA_*
    - Command flags override all

installation:
  install_script: Creates symlink in ~/.vrooli/bin/
  path_update: Adds to PATH if needed
  permissions: 755 on executable
  documentation: Comprehensive --help output
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Node.js Runtime**: Required for server and generation libraries
- **FFmpeg**: Required for audio/video generation (installed by setup)

### Downstream Enablement
**What future capabilities does this unlock?**
- **Automated Testing**: All scenarios can write comprehensive tests without committing test files
- **Performance Benchmarking**: Consistent datasets enable reliable performance testing
- **Demo Environments**: Instant population with realistic data for demonstrations
- **AI Training**: Structured data generation for model training pipelines

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: ALL
    capability: On-demand test data generation
    interface: API/CLI
    
  - scenario: ci-cd-healer
    capability: Test fixture generation for pipeline validation
    interface: CLI
    
  - scenario: test-genie
    capability: Dynamic test data for learning test patterns
    interface: API
    
  - scenario: data-generator
    capability: Structured data templates for scraping
    interface: API

consumes_from:
  - scenario: None (foundational service)
    capability: N/A
    fallback: N/A
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: "PostMan's mock server UI + Mockaroo's data generator"
  
  visual_style:
    color_scheme: dark
    typography: technical
    layout: dashboard
    animations: subtle
  
  personality:
    tone: technical
    mood: focused
    target_feeling: "Powerful and efficient tool"

style_references:
  technical:
    - Swagger UI: "Clear API documentation layout"
    - JSONPlaceholder: "Simple, effective mock data service"
    - Postman Mock Servers: "Professional API mocking interface"
```

### Target Audience Alignment
- **Primary Users**: Developers, QA engineers, AI agents writing tests
- **User Expectations**: Fast, reliable, comprehensive format support
- **Accessibility**: WCAG 2.1 AA compliance
- **Responsive Design**: Desktop-first, mobile-friendly

### Brand Consistency Rules
- **Scenario Identity**: Technical powerhouse for test data
- **Vrooli Integration**: Foundational service that enhances all scenarios
- **Professional vs Fun**: Professional tool with comprehensive capabilities

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates test file repository bloat, enables dynamic testing
- **Revenue Potential**: $5K - $10K per deployment (as testing infrastructure)
- **Cost Savings**: 50% reduction in test maintenance time
- **Market Differentiator**: Integrated multi-format generation with deterministic seeds

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from test data
- **Complexity Reduction**: Testing becomes trivial with on-demand data
- **Innovation Enablement**: Unlocks comprehensive automated testing

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core format support (JSON, YAML, CSV, XML, images, audio, video, PDF)
- Deterministic generation via seeds
- RESTful API and CLI
- Basic templates

### Version 2.0 (Planned)
- Advanced schema validation
- Streaming for large datasets
- Redis caching integration
- Complex media generation
- Faker.js locale support

### Long-term Vision
- ML-powered realistic data generation
- Domain-specific generators (financial, medical, etc.)
- Integration with test frameworks
- Distributed generation for massive datasets

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with metadata
    - Complete initialization files
    - Startup and monitoring scripts
    - Health check endpoints
    
  deployment_targets:
    - local: Node.js process
    - docker: Containerized service
    - kubernetes: Helm chart ready
    
  revenue_model:
    - type: usage-based
    - pricing_tiers: 
      - free: 1000 requests/day
      - pro: Unlimited with caching
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: test-data-generator
    category: generation
    capabilities: 
      - Mock data generation
      - Multi-format support
      - Deterministic generation
      - Batch processing
    interfaces:
      - api: http://localhost:3001/api/v1
      - cli: test-data-generator
      
  metadata:
    description: Universal test data generator for all formats
    keywords: [test, mock, data, generation, faker, images, video, audio]
    dependencies: []
    enhances: [ALL scenarios requiring test data]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Memory exhaustion from large files | Medium | High | Streaming, size limits |
| FFmpeg unavailable | Low | Medium | Fallback to simple patterns |
| Cache overflow | Low | Low | LRU eviction policy |

### Operational Risks
- **Resource Conflicts**: Isolated Node.js process with resource limits
- **Security**: Input validation, no code execution, sandboxed generation
- **Rate Limiting**: Built-in throttling for resource protection

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: test-data-generator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/server.js
    - api/package.json
    - cli/test-data-generator
    - cli/install.sh
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - ui
    - tests
    - initialization

resources:
  required: []
  optional: [redis, minio]
  health_timeout: 30

tests:
  - name: "API health check"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Generate JSON data"
    type: http
    service: api
    endpoint: /api/v1/generate/json
    method: POST
    body:
      count: 10
      faker: { name: "firstName", email: "email" }
    expect:
      status: 200
      body:
        data: { _type: "array", _length: 10 }
        
  - name: "CLI generates CSV"
    type: exec
    command: ./cli/test-data-generator generate csv --count 5
    expect:
      exit_code: 0
      output_contains: ["Name,Email"]
      
  - name: "Batch generation works"
    type: http
    service: api
    endpoint: /api/v1/generate/batch
    method: POST
    body:
      requests: 
        - { type: "json", count: 5 }
        - { type: "csv", count: 5 }
    expect:
      status: 200
      body:
        files: { _type: "array", _length: 2 }
```

### Performance Validation
- [x] API responds under 500ms for standard requests
- [ ] Can generate 100 small files per second
- [ ] Memory stays under 512MB during normal operation
- [ ] No memory leaks over 24-hour test run

### Integration Validation
- [x] CLI executable with comprehensive --help
- [x] All API endpoints documented and functional
- [ ] Can be called from other scenarios
- [ ] Handles concurrent requests gracefully

### Capability Verification
- [x] Generates all P0 format types
- [x] Deterministic generation with seeds works
- [x] Batch generation functional
- [ ] Integrates with test frameworks
- [x] UI provides intuitive interface

## 📝 Implementation Notes

### Design Decisions
**Node.js over Go**: Chosen for rich ecosystem of generation libraries
- Alternative considered: Go for performance
- Decision driver: Library availability (faker.js, canvas, pdfkit)
- Trade-offs: Slightly lower performance for much faster development

**Standalone service over workflow integration**: Direct API for reliability
- Alternative considered: n8n workflows
- Decision driver: Foundational service needs maximum reliability
- Trade-offs: Less flexibility for more stability

### Known Limitations
- **Large files**: Generation limited to 100MB to prevent memory issues
  - Workaround: Use streaming or generate in chunks
  - Future fix: Implement full streaming in v2.0
  
- **Video complexity**: Currently limited to simple patterns
  - Workaround: Use repeated patterns or solid colors
  - Future fix: FFmpeg scene generation in v2.0

### Security Considerations
- **Input Validation**: All schemas validated before processing
- **Resource Limits**: CPU and memory limits enforced
- **No Code Execution**: Templates are data-only, no eval()
- **Sandboxing**: Generation runs in isolated context

## 🔗 References

### Documentation
- README.md - User guide and examples
- api/README.md - API specification
- cli/README.md - CLI documentation
- tests/README.md - Testing guide

### Related PRDs
- data-generator - Complementary web scraping capabilities
- test-genie - Consumes test data for learning patterns

### External Resources
- Faker.js documentation
- Canvas API reference
- FFmpeg encoding guides
- PDF generation best practices

---

**Last Updated**: 2024-01-09  
**Status**: Testing  
**Owner**: AI Agent  
**Review Cycle**: Monthly validation against implementation