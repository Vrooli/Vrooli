# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Comprehensive image manipulation toolkit providing compression, resizing, upscaling, format conversion, and metadata management through both API and CLI interfaces. Enables any scenario to optimize visual assets for production use.

### Intelligence Amplification
**How does this capability make future agents smarter?**
Agents generating websites, documentation, or visual content can automatically optimize assets without manual intervention. This removes the technical barrier of image optimization, letting agents focus on creative/business logic while ensuring professional-quality output.

### Recursive Value
**What new scenarios become possible after this exists?**
- **Website generator scenarios** can auto-optimize all assets before deployment
- **Documentation builders** can standardize image formats and sizes
- **E-commerce platforms** can generate multiple product image variants
- **Social media managers** can auto-format images for different platforms
- **Performance analyzers** can identify and fix asset bottlenecks

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Image compression for JPEG, PNG, WebP, SVG formats
  - [ ] Format conversion between supported formats
  - [ ] Resizing with multiple resampling algorithms
  - [ ] Metadata stripping for privacy/size reduction
  - [ ] REST API with all operations
  - [ ] CLI with full API parity
  - [ ] Plugin architecture for format-specific operations
  - [ ] Live preview in UI with before/after comparison
  
- **Should Have (P1)**
  - [ ] AI upscaling using Real-ESRGAN or similar
  - [ ] Batch processing for multiple images
  - [ ] Preset profiles (web-optimized, email-safe, aggressive)
  - [ ] WebP and AVIF modern format support
  - [ ] Drag-and-drop UI for bulk operations
  - [ ] Size/quality optimization recommendations
  
- **Nice to Have (P2)**
  - [ ] Basic filters (blur, sharpen, brightness, contrast)
  - [ ] Smart cropping with object detection
  - [ ] Image format auto-detection and best format suggestion
  - [ ] Integration with CDN services
  - [ ] Watermarking capabilities

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 2000ms for images up to 10MB | API monitoring |
| Throughput | 50 images/minute | Load testing |
| Compression Ratio | > 60% size reduction average | Validation suite |
| Resource Usage | < 2GB memory, < 50% CPU | System monitoring |

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
  - resource_name: minio
    purpose: Store original and processed images
    integration_pattern: S3-compatible API for scalable storage
    access_method: resource-minio CLI and S3 SDK
    
optional:
  - resource_name: redis
    purpose: Cache processed images for faster retrieval
    fallback: Direct processing without cache
    access_method: resource-redis CLI
    
  - resource_name: ollama
    purpose: AI-powered image analysis and smart cropping
    fallback: Basic center cropping
    access_method: initialization/n8n/ollama.json workflow
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: Image content analysis for smart operations
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-minio put-object
      purpose: Store processed images
    - command: resource-redis set
      purpose: Cache results
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: Image processing libraries require direct integration
      endpoint: Native Go/Node image libraries

# Shared workflow guidelines:
shared_workflow_criteria:
  - Image analysis workflow could be shared for content moderation
  - Batch processing orchestration useful for other media scenarios
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: ProcessedImage
    storage: minio/redis
    schema: |
      {
        id: UUID
        original_url: string
        processed_url: string
        format: string
        dimensions: {width: int, height: int}
        size_bytes: int
        compression_ratio: float
        metadata: map
        created_at: timestamp
      }
    relationships: Can be referenced by any scenario needing optimized assets
    
  - name: ProcessingProfile
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        settings: {
          format: string
          quality: int
          max_width: int
          max_height: int
          strip_metadata: bool
        }
      }
    relationships: Reusable across multiple processing requests
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/image/compress
    purpose: Compress image with quality settings
    input_schema: |
      {
        image: binary or URL
        quality: int (1-100)
        format: string (optional)
      }
    output_schema: |
      {
        url: string
        original_size: int
        compressed_size: int
        savings_percent: float
      }
    sla:
      response_time: 2000ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/image/resize
    purpose: Resize image to specific dimensions
    input_schema: |
      {
        image: binary or URL
        width: int
        height: int
        maintain_aspect: bool
        algorithm: string (lanczos|bilinear|nearest)
      }
    output_schema: |
      {
        url: string
        dimensions: {width: int, height: int}
      }
      
  - method: POST
    path: /api/v1/image/convert
    purpose: Convert between formats
    input_schema: |
      {
        image: binary or URL
        target_format: string
        options: map (format-specific)
      }
    output_schema: |
      {
        url: string
        format: string
        size: int
      }
      
  - method: POST
    path: /api/v1/image/batch
    purpose: Process multiple images with same settings
    input_schema: |
      {
        images: array[binary or URL]
        operations: array[operation]
      }
    output_schema: |
      {
        results: array[processed_image]
        total_savings: int
      }
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: image.processing.completed
    payload: ProcessedImage object
    subscribers: Performance analyzers, CDN updaters
    
  - name: image.batch.completed
    payload: Batch processing summary
    subscribers: Website generators, documentation builders
    
consumed_events:
  - name: asset.uploaded
    action: Auto-optimize if matches criteria
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: image-tools
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
  - name: compress
    description: Compress an image file
    api_endpoint: /api/v1/image/compress
    arguments:
      - name: input
        type: string
        required: true
        description: Path to input image or URL
    flags:
      - name: --quality
        description: Compression quality (1-100)
      - name: --output
        description: Output file path
    output: Compressed image path and statistics
    
  - name: resize
    description: Resize an image
    api_endpoint: /api/v1/image/resize
    arguments:
      - name: input
        type: string
        required: true
        description: Input image path or URL
    flags:
      - name: --width
        description: Target width in pixels
      - name: --height
        description: Target height in pixels
      - name: --maintain-aspect
        description: Keep aspect ratio
    
  - name: convert
    description: Convert image format
    api_endpoint: /api/v1/image/convert
    arguments:
      - name: input
        type: string
        required: true
      - name: format
        type: string
        required: true
        description: Target format (jpg|png|webp|svg)
        
  - name: batch
    description: Process multiple images
    api_endpoint: /api/v1/image/batch
    arguments:
      - name: input-dir
        type: string
        required: true
    flags:
      - name: --operations
        description: JSON array of operations to apply
      - name: --output-dir
        description: Directory for processed images
```

### CLI-API Parity Requirements
- **Coverage**: Every API endpoint has corresponding CLI command
- **Naming**: CLI uses intuitive verb-noun pattern
- **Arguments**: Direct mapping to API parameters
- **Output**: Human-readable by default, JSON with --json flag
- **Authentication**: Uses API tokens from environment/config

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper over API client
  - language: Go for consistency
  - dependencies: Minimal - just API client
  - error_handling: Clear error messages with suggestions
  - configuration: 
      - ~/.vrooli/image-tools/config.yaml
      - Environment variables (VROOLI_IMAGE_*)
      - Command flags highest priority
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - path_update: Adds to PATH if needed
  - permissions: 755 executable
  - documentation: Comprehensive --help
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Minio**: Object storage for images
- **Basic API infrastructure**: HTTP server, routing

### Downstream Enablement
**What future capabilities does this unlock?**
- **Website generators**: Auto-optimized assets
- **Documentation builders**: Standardized image handling
- **E-commerce platforms**: Product image variants
- **Social media tools**: Platform-specific formatting

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: website-generator
    capability: Automatic asset optimization
    interface: API/CLI
    
  - scenario: documentation-builder
    capability: Image standardization
    interface: API
    
  - scenario: social-media-manager
    capability: Platform-specific image formatting
    interface: API/Events
    
consumes_from:
  - scenario: storage-manager
    capability: Centralized file storage
    fallback: Local filesystem
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: creative-technical
  inspiration: Retro photo lab meets modern dev tools
  
  visual_style:
    color_scheme: dark with red accent lighting
    typography: monospace primary, vintage labels
    layout: darkroom aesthetic with film strips
    animations: developing photo effects, spinning dials
  
  personality:
    tone: professional but playful
    mood: nostalgic creativity
    target_feeling: "developing memories in a digital darkroom"

# Specific design elements:
design_elements:
  - Film strip borders for image galleries
  - Red "darkroom" lighting accents
  - Vintage toggle switches and dials for controls
  - "Developing..." animation during processing
  - Before/after split view with draggable divider
  - Histogram displays with retro CRT glow
  - File size "weight scale" visualization
```

### Target Audience Alignment
- **Primary Users**: Developers, designers, content creators
- **User Expectations**: Professional tools with personality
- **Accessibility**: WCAG AA compliance, keyboard navigation
- **Responsive Design**: Desktop-first, tablet-friendly

### Brand Consistency Rules
- **Scenario Identity**: Photo lab aesthetic unique in Vrooli
- **Vrooli Integration**: Maintains quality bar while adding character
- **Professional vs Fun**: Technical capability with creative flair

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates manual image optimization labor
- **Revenue Potential**: $15K - $30K per enterprise deployment
- **Cost Savings**: 10+ hours/week for content teams
- **Market Differentiator**: Plugin architecture for extensibility

### Technical Value
- **Reusability Score**: 9/10 - Nearly every visual scenario needs this
- **Complexity Reduction**: Complex image operations become one-liners
- **Innovation Enablement**: Removes image handling as a blocker

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core compression, resize, convert, metadata operations
- Plugin architecture for formats
- REST API and CLI
- Retro photo lab UI

### Version 2.0 (Planned)
- AI upscaling integration
- Smart cropping with object detection
- Advanced filters and effects
- CDN integration

### Long-term Vision
- Full creative suite capabilities
- Video frame extraction and optimization
- Real-time collaborative editing
- ML-powered format recommendations

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with image service config
    - Plugin system for format handlers
    - API and CLI implementations
    - Health check endpoints
    
  deployment_targets:
    - local: Docker with image processing libraries
    - kubernetes: StatefulSet for plugin management
    - cloud: Lambda for serverless processing
    
  revenue_model:
    - type: usage-based
    - pricing_tiers: 
        - free: 1000 images/month
        - pro: 50000 images/month
        - enterprise: unlimited
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: image-tools
    category: utilities
    capabilities: [compress, resize, upscale, convert, metadata]
    interfaces:
      - api: http://localhost:PORT/api/v1/image
      - cli: image-tools
      - events: image.*
      
  metadata:
    description: Complete image manipulation toolkit
    keywords: [image, compress, resize, optimize, convert]
    dependencies: [minio]
    enhances: [website-generator, documentation-builder]
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
| Large file processing OOM | Medium | High | Streaming processing, size limits |
| Format compatibility issues | Low | Medium | Comprehensive format testing |
| Processing bottlenecks | Medium | Medium | Queue system, horizontal scaling |

### Operational Risks
- **Drift Prevention**: Plugin interface versioning
- **Version Compatibility**: Backward compatible API
- **Resource Conflicts**: Isolated processing environments
- **Style Drift**: Component library enforcement
- **CLI Consistency**: Automated API-CLI parity tests

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: image-tools

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/image-tools
    - cli/install.sh
    - plugins/jpeg/handler.go
    - plugins/png/handler.go
    - ui/index.html
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - plugins
    - ui
    - initialization

resources:
  required: [minio]
  optional: [redis, ollama]
  health_timeout: 60

tests:
  - name: "API compression endpoint"
    type: http
    service: api
    endpoint: /api/v1/image/compress
    method: POST
    body:
      image: "test-image.jpg"
      quality: 80
    expect:
      status: 200
      body:
        savings_percent: ">0"
        
  - name: "CLI compress command"
    type: exec
    command: ./cli/image-tools compress test.jpg --quality 80
    expect:
      exit_code: 0
      output_contains: ["Compressed", "saved"]
      
  - name: "Plugin architecture loads"
    type: http
    service: api
    endpoint: /api/v1/plugins
    method: GET
    expect:
      status: 200
      body:
        plugins: ["jpeg", "png", "webp", "svg"]
```

### Test Execution Gates
```bash
./test.sh --scenario image-tools --validation complete
```

### Performance Validation
- [ ] Compression under 2s for 10MB images
- [ ] Batch processing maintains throughput
- [ ] Memory usage stable under load
- [ ] Cache hit ratio > 80% for repeat requests

### Integration Validation
- [ ] Discoverable via registry
- [ ] All endpoints documented
- [ ] CLI commands match API
- [ ] Events published correctly
- [ ] Plugin system extensible

### Capability Verification
- [ ] All major formats supported
- [ ] Quality/size tradeoffs acceptable
- [ ] Batch operations performant
- [ ] UI provides live preview
- [ ] Retro aesthetic implemented

## 📝 Implementation Notes

### Design Decisions
**Plugin Architecture**: Chose plugin pattern for format handlers
- Alternative considered: Monolithic processor
- Decision driver: Extensibility and maintenance
- Trade-offs: Slight complexity for major flexibility

**API-First Design**: Process server-side not client-side
- Alternative considered: WASM client processing
- Decision driver: Consistency and power
- Trade-offs: Network overhead for privacy and capability

### Known Limitations
- **Large files**: 100MB limit initially
  - Workaround: Pre-split large images
  - Future fix: Streaming processing in v2
  
- **Format support**: Limited to mainstream formats
  - Workaround: Convert to supported format first
  - Future fix: Plugin marketplace

### Security Considerations
- **Data Protection**: Images deleted after processing
- **Access Control**: API key authentication
- **Audit Trail**: All operations logged with user/timestamp

## 🔗 References

### Documentation
- README.md - User guide
- docs/api.md - API specification
- docs/plugins.md - Plugin development guide
- docs/architecture.md - Technical design

### Related PRDs
- storage-manager - File storage foundation
- website-generator - Primary consumer

### External Resources
- ImageMagick documentation
- mozjpeg optimization research
- Real-ESRGAN upscaling papers

---

**Last Updated**: 2025-01-09  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Weekly validation against implementation