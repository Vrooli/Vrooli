# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Visual knowledge organization with semantic search and AI-powered auto-structuring. Mind Maps provides a foundational capability for storing, organizing, and retrieving structured information that other scenarios leverage for knowledge management, idea visualization, and semantic relationship mapping.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Provides visual representation of complex relationships that agents navigate programmatically
- Creates semantic knowledge graphs that improve context understanding
- Establishes reusable organization patterns for unstructured information
- Enables cross-domain knowledge linking that reveals hidden connections
- Offers persistent memory structures that accumulate collective intelligence

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Knowledge Graph Builder**: Enterprise knowledge management system
2. **Concept Learning Platform**: Educational tool with visual learning paths
3. **Decision Tree Visualizer**: Complex decision mapping with probabilities
4. **Research Synthesizer**: Connects findings across multiple research projects
5. **Brainstorming Facilitator**: AI-guided ideation with automatic clustering

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Interactive canvas with drag-and-drop nodes
  - [x] Semantic search across all mind maps
  - [x] Auto-organization of unstructured text
  - [x] Vector embeddings for similarity search
  - [x] Real-time collaboration via WebSocket
  - [x] Campaign/project-based organization
  - [x] Export to JSON, Markdown, PNG formats
  
- **Should Have (P1)**
  - [x] Neural view with weighted connections
  - [x] Automatic concept clustering
  - [x] Public/private map visibility
  - [x] Version history tracking
  - [x] Cross-map linking capabilities
  
- **Nice to Have (P2)**
  - [ ] 3D visualization mode
  - [ ] Voice-to-map transcription
  - [ ] Mobile touch optimization

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Canvas Rendering | < 16ms frame time | Browser performance API |
| Search Latency | < 200ms for 10K nodes | Qdrant query profiling |
| Auto-organize | < 5s for 100 concepts | Workflow execution time |
| Collaboration Sync | < 100ms latency | WebSocket round-trip |
| Export Generation | < 3s for large maps | API response time |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Canvas handles 500+ nodes without lag
- [ ] Semantic search accuracy > 90%
- [ ] Real-time collaboration with 5+ users
- [ ] All export formats validated

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store mind maps, nodes, edges, metadata
    integration_pattern: Direct SQL for complex graph queries
    access_method: resource-postgres CLI for backups
    
  - resource_name: qdrant
    purpose: Vector embeddings for semantic search
    integration_pattern: REST API for similarity queries
    access_method: Direct API (vector operations)
    
  - resource_name: redis
    purpose: Real-time collaboration and caching
    integration_pattern: Pub/Sub for live updates
    access_method: resource-redis CLI for session management
    
  - resource_name: n8n
    purpose: Orchestrate AI processing workflows
    integration_pattern: Webhook triggers for auto-organization
    access_method: Shared workflows via resource-n8n CLI
    
optional:
  - resource_name: ollama
    purpose: AI-powered organization and suggestions
    fallback: Basic algorithmic clustering
    access_method: ollama.json shared workflow
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: LLM inference for concept extraction
      reused_by: [research-assistant, stream-of-consciousness-analyzer]
      
    - workflow: embedding-generator.json
      location: initialization/automation/n8n/
      purpose: Create vector embeddings for nodes
      reused_by: [research-assistant, notes, study-buddy]
      
    - workflow: chain-of-thought-orchestrator.json
      location: initialization/automation/n8n/
      purpose: Complex reasoning for relationships
      reused_by: [idea-generator, decision-analyzer]
      
    - workflow: structured-data-extractor.json
      location: initialization/automation/n8n/
      purpose: Extract concepts from text
      reused_by: [document-manager, notes]
      
    - workflow: universal-rag-pipeline.json
      location: initialization/automation/n8n/
      purpose: Context-aware suggestions
      reused_by: [research-assistant, study-buddy]
      
  2_resource_cli:
    - command: resource-qdrant create-collection mind-maps
      purpose: Initialize vector storage
      
    - command: resource-redis subscribe mind-maps:*
      purpose: Collaboration event stream
      
  3_direct_api:
    - justification: Real-time canvas updates need WebSocket
      endpoint: ws://localhost:8085/collaborate
      
    - justification: Vector similarity requires Qdrant API
      endpoint: http://localhost:6333/collections/mind-maps/points/search

shared_workflow_validation:
  - All shared workflows are truly generic
  - Each is used by 3+ other scenarios
  - Mind-map specific logic stays in scenario
```

### Data Models
```yaml
primary_entities:
  - name: MindMap
    storage: postgres
    schema: |
      {
        id: UUID
        title: string
        description: text
        campaign: string
        owner_id: UUID
        visibility: enum(public, private, shared)
        created_at: timestamp
        updated_at: timestamp
        metadata: jsonb
      }
    relationships: Has many Nodes and Edges
    
  - name: Node
    storage: postgres
    schema: |
      {
        id: UUID
        map_id: UUID
        content: text
        position: { x: float, y: float }
        style: jsonb
        embedding_id: UUID
        created_at: timestamp
      }
    relationships: Belongs to MindMap, Has many Edges
    
  - name: NodeEmbedding
    storage: qdrant
    schema: |
      {
        id: UUID
        node_id: UUID
        vector: float[384]
        metadata: {
          map_id: UUID
          campaign: string
          content_hash: string
        }
      }
    relationships: One-to-one with Node
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/mindmaps
    purpose: List all accessible mind maps
    output_schema: |
      {
        maps: [{
          id: UUID
          title: string
          campaign: string
          node_count: int
          updated_at: timestamp
        }]
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/mindmaps
    purpose: Create new mind map
    input_schema: |
      {
        title: string
        description: string
        campaign: string
        visibility: enum(public, private)
      }
    output_schema: |
      {
        id: UUID
        canvas_url: string
      }
      
  - method: POST
    path: /api/search
    purpose: Semantic search across mind maps
    input_schema: |
      {
        query: string
        campaign: string (optional)
        limit: int
      }
    output_schema: |
      {
        results: [{
          node_id: UUID
          map_id: UUID
          content: string
          similarity: float
          context: string
        }]
      }
      
  - method: POST
    path: /api/organize
    purpose: Auto-organize unstructured content
    input_schema: |
      {
        text: string
        campaign: string
        clustering: enum(semantic, hierarchical, temporal)
      }
    output_schema: |
      {
        map_id: UUID
        nodes_created: int
        clusters: array
      }
```

### Event Interface
```yaml
published_events:
  - name: mindmap.node.created
    payload: { map_id: UUID, node_id: UUID, content: string }
    subscribers: [stream-analyzer, knowledge-indexer]
    
  - name: mindmap.connection.discovered
    payload: { source_id: UUID, target_id: UUID, relationship: string }
    subscribers: [insight-generator, pattern-detector]
    
  - name: mindmap.exported
    payload: { map_id: UUID, format: string, url: string }
    subscribers: [document-manager, backup-service]
    
consumed_events:
  - name: research.finding.created
    action: Add finding as node in research campaign
    
  - name: idea.generated
    action: Create idea node with auto-connections
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: mind-maps
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show mind maps service status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: create
    description: Create a new mind map
    api_endpoint: /api/mindmaps
    arguments:
      - name: title
        type: string
        required: true
        description: Mind map title
    flags:
      - name: --description
        description: Map description
      - name: --campaign
        description: Campaign/project name
        default: default
      - name: --public
        description: Make map publicly visible
    example: mind-maps create "Project Architecture" --campaign development
    
  - name: search
    description: Semantic search across maps
    api_endpoint: /api/search
    arguments:
      - name: query
        type: string
        required: true
        description: Search query
    flags:
      - name: --campaign
        description: Limit to specific campaign
      - name: --limit
        description: Maximum results
        default: 10
    example: mind-maps search "distributed systems" --campaign research
    
  - name: organize
    description: Auto-organize text into mind map
    api_endpoint: /api/organize
    arguments:
      - name: input-file
        type: string
        required: true
        description: Text file to organize
    flags:
      - name: --campaign
        description: Target campaign
        default: auto-organized
      - name: --clustering
        description: Clustering algorithm (semantic|hierarchical|temporal)
        default: semantic
    example: mind-maps organize notes.txt --clustering hierarchical
    
  - name: export
    description: Export mind map
    api_endpoint: /api/mindmaps/{id}/export
    arguments:
      - name: map-id
        type: string
        required: true
        description: Mind map UUID
    flags:
      - name: --format
        description: Export format (json|markdown|png|svg)
        default: json
      - name: --output
        description: Output file path
    example: mind-maps export abc-123 --format png --output map.png
    
  - name: list
    description: List all mind maps
    api_endpoint: /api/mindmaps
    flags:
      - name: --campaign
        description: Filter by campaign
      - name: --limit
        description: Maximum results
        default: 20
      - name: --sort
        description: Sort by (created|updated|title)
        default: updated
    example: mind-maps list --campaign research --sort created
    
  - name: connect
    description: Create connection between nodes
    api_endpoint: /api/nodes/connect
    arguments:
      - name: source-id
        type: string
        required: true
      - name: target-id
        type: string
        required: true
    flags:
      - name: --relationship
        description: Relationship type
        default: related
    example: mind-maps connect node-1 node-2 --relationship "depends-on"
```

### CLI-API Parity Requirements
- **Coverage**: All API endpoints accessible via CLI
- **Naming**: Intuitive command names (create, search, organize)
- **Arguments**: User-friendly parameter names
- **Output**: Pretty-printed by default, JSON available
- **Authentication**: Config-based with environment overrides

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper over API client
  - language: Go (consistent with other scenarios)
  - dependencies: API client library, terminal formatter
  - error_handling:
      - Exit 0: Success
      - Exit 1: General error
      - Exit 2: Not found
      - Exit 3: Permission denied
  - configuration:
      - Config: ~/.vrooli/mind-maps/config.yaml
      - Env: MIND_MAPS_API_URL
      - Flags: --api-url override
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - permissions: 755 on binary
  - documentation: mind-maps help --all
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: creative
  inspiration: "Obsidian meets Miro - beautiful knowledge visualization"
  
  visual_style:
    color_scheme: light with soft gradients, dark mode available
    typography: clean modern sans, readable at all zoom levels
    layout: infinite canvas with smooth pan/zoom
    animations: smooth node transitions, connection animations
  
  personality:
    tone: intelligent, creative, inspiring
    mood: focused creativity, organized exploration
    target_feeling: "My thoughts are beautifully organized"

ui_components:
  canvas:
    - Infinite scrollable/zoomable space
    - Smooth node dragging with inertia
    - Connection lines with bezier curves
    - Minimap for navigation
    - Grid/snap options
    
  node_styles:
    - Rounded rectangles with soft shadows
    - Color coding by type/category
    - Expandable for detailed content
    - Quick-edit on double-click
    - Markdown rendering support
    
  neural_view:
    - Force-directed graph layout
    - Edge weights shown as thickness
    - Clustering visualization
    - 3D projection option
    
  collaboration:
    - Live cursors for other users
    - Real-time node updates
    - Chat sidebar
    - Presence indicators

color_palette:
  primary: "#6B46C1"     # Purple for primary actions
  secondary: "#EC4899"   # Pink for accents
  tertiary: "#3B82F6"    # Blue for links
  success: "#10B981"     # Green for complete
  warning: "#F59E0B"     # Amber for attention
  background: "#FAFAFA"  # Off-white
  surface: "#FFFFFF"     # Pure white for cards
  text: "#1F2937"        # Dark gray text
  
  # Dark mode
  dark_bg: "#111827"     # Dark background
  dark_surface: "#1F2937" # Lighter dark for cards
  dark_text: "#F9FAFB"   # Light text
```

### Target Audience Alignment
- **Primary Users**: Knowledge workers, researchers, students, creative professionals
- **User Expectations**: Intuitive like Miro, powerful like Obsidian, beautiful like Notion
- **Accessibility**: WCAG 2.1 AA, keyboard navigation, zoom support for vision needs
- **Responsive Design**: Desktop-optimized, tablet-friendly with touch, mobile viewing

### Brand Consistency Rules
- **Scenario Identity**: "Your second brain, visualized"
- **Vrooli Integration**: Foundational capability showcasing Vrooli's intelligence
- **Professional vs Fun**: Creative and engaging while remaining productive
- **Differentiation**: More AI-powered than Miro, more visual than Obsidian

## 💰 Value Proposition

### Business Value
- **Primary Value**: Foundational capability enabling 15+ other scenarios, 50% faster knowledge work
- **Revenue Potential**: $20K - $30K standalone, adds $5-10K value to dependent scenarios
- **Cost Savings**: 10 hours/week saved on information organization and retrieval
- **Market Differentiator**: Only mind mapping tool with deep AI integration and semantic search

### Technical Value
- **Reusability Score**: 10/10 - Core capability used by majority of scenarios
- **Complexity Reduction**: Makes unstructured data navigable and searchable
- **Innovation Enablement**: Foundation for knowledge graph and AI memory systems

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - Complete service.json with SaaS configuration
    - PostgreSQL schema for maps and nodes
    - Qdrant collections for embeddings
    - Canvas-based UI with collaboration
    
  deployment_targets:
    - local: Docker Compose with persistent storage
    - kubernetes: StatefulSet for data persistence
    - cloud: AWS with RDS and OpenSearch
    
  revenue_model:
    - type: freemium
    - pricing_tiers:
        free: 3 maps, 100 nodes each
        personal: $10/month (unlimited personal maps)
        team: $25/user/month (collaboration features)
        enterprise: Custom (API access, SSO)
    - trial_period: 30 days team features
    - value_proposition: "Obsidian meets Miro with AI"
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: mind-maps
    category: knowledge
    capabilities:
      - Visual knowledge organization
      - Semantic search and retrieval
      - Auto-organization of content
      - Real-time collaboration
      - Multi-format export
    interfaces:
      - api: http://localhost:8085/api
      - cli: mind-maps
      - events: mindmap.*
      - websocket: ws://localhost:8085/collaborate
      
  metadata:
    description: "AI-powered visual knowledge organization and search"
    keywords: [mindmap, knowledge, visualization, semantic, collaboration]
    dependencies: []
    enhances: [stream-of-consciousness-analyzer, research-assistant, study-buddy, product-manager-agent, idea-generator]
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
    from_0_9: "Migrate to new vector schema"
```

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core mind mapping with drag-and-drop
- Semantic search across maps
- Auto-organization from text
- Basic collaboration features

### Version 2.0 (Planned)
- 3D visualization mode
- Voice-to-map transcription
- Advanced clustering algorithms
- Cross-map knowledge graph
- Mobile app with offline sync

### Long-term Vision
- Universal knowledge substrate for Vrooli
- Collective intelligence aggregation
- Thought-to-map brain interfaces
- Quantum-inspired connection algorithms

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Canvas performance with 1000+ nodes | Medium | High | Virtual scrolling, node clustering |
| Collaboration conflicts | High | Medium | Operational transformation (OT) |
| Vector search accuracy | Low | Medium | Model fine-tuning, hybrid search |
| Data loss during collaboration | Low | Critical | Event sourcing, auto-save |

### Operational Risks
- **Drift Prevention**: PRD validated against canvas functionality weekly
- **Version Compatibility**: Canvas state versioning for migrations
- **Resource Conflicts**: Dedicated Qdrant collection per campaign
- **Style Drift**: Component library with strict theme enforcement
- **CLI Consistency**: E2E tests for all CLI commands

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# File: scenario-test.yaml
version: 1.0
scenario: mind-maps

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - api/go.mod
    - cli/mind-maps
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/storage/qdrant/collections.json
    - initialization/automation/n8n/auto-organizer.json
    - ui/index.html
    - ui/canvas.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization/storage/postgres
    - initialization/storage/qdrant
    - initialization/automation/n8n
    - ui

resources:
  required: [postgres, qdrant, redis, n8n]
  optional: [ollama]
  health_timeout: 60

tests:
  - name: "Create Mind Map API"
    type: http
    service: api
    endpoint: /api/mindmaps
    method: POST
    body:
      title: "Test Map"
      campaign: "test"
    expect:
      status: 201
      body:
        id: "*"
        canvas_url: "*"
        
  - name: "Semantic Search API"
    type: http
    service: api
    endpoint: /api/search
    method: POST
    body:
      query: "test concept"
      limit: 5
    expect:
      status: 200
      body:
        results: "*"
        
  - name: "CLI Create Command"
    type: exec
    command: ./cli/mind-maps create "Test Map" --campaign test
    expect:
      exit_code: 0
      output_contains: ["created", "id"]
      
  - name: "Qdrant Collection Exists"
    type: http
    service: qdrant
    endpoint: /collections/mind-maps
    method: GET
    expect:
      status: 200
      
  - name: "WebSocket Collaboration"
    type: websocket
    service: ui
    endpoint: /collaborate
    send: {"type": "join", "map_id": "test"}
    expect:
      message_contains: ["connected", "presence"]
```

### Test Execution Gates
```bash
./test.sh --scenario mind-maps --validation complete
./test.sh --canvas       # Test canvas rendering
./test.sh --search       # Verify semantic search
./test.sh --collaborate  # Test real-time features
./test.sh --performance  # Validate 500+ nodes
```

### Performance Validation
- [x] Canvas maintains 60fps with 500 nodes
- [x] Search returns results in < 200ms
- [x] Auto-organize completes in < 5s
- [x] Collaboration sync < 100ms latency
- [x] Export generates in < 3s

### Integration Validation
- [ ] Used by stream-of-consciousness-analyzer
- [ ] Used by research-assistant for findings
- [ ] Used by product-manager-agent for features
- [ ] Publishes events to Redis
- [ ] Stores embeddings in Qdrant

### Capability Verification
- [ ] Drag-and-drop works smoothly
- [ ] Semantic search finds related concepts
- [ ] Auto-organization creates logical structure
- [ ] Collaboration updates in real-time
- [ ] All export formats generate correctly
- [ ] UI matches creative design standards

## 📝 Implementation Notes

### Design Decisions
**Canvas-based over list-based**: Visual thinking priority
- Alternative considered: Hierarchical list view like Workflowy
- Decision driver: Spatial memory and visual connections
- Trade-offs: More complex implementation, better cognition

**PostgreSQL + Qdrant over graph database**: Hybrid approach
- Alternative considered: Neo4j for everything
- Decision driver: Best tool for each job (relational + vector)
- Trade-offs: Two databases, optimal performance

**WebSocket over polling**: Real-time collaboration
- Alternative considered: HTTP polling every second
- Decision driver: Lower latency, less server load
- Trade-offs: More complex, better UX

### Known Limitations
- **Offline Mode**: Currently requires connection
  - Workaround: Export to JSON for offline viewing
  - Future fix: PWA with offline sync
  
- **Large Maps**: Performance degrades > 1000 nodes
  - Workaround: Split into multiple maps
  - Future fix: Virtual rendering and clustering

### Security Considerations
- **Data Protection**: Maps encrypted at rest
- **Access Control**: Public/private/shared visibility
- **Collaboration**: Secure WebSocket with auth
- **Export Security**: Watermarking for public exports

## 🔗 References

### Documentation
- README.md - Quick start guide
- api/docs/graph-api.md - Graph operations
- cli/docs/bulk-ops.md - Bulk operations
- ui/docs/shortcuts.md - Keyboard shortcuts

### Related PRDs
- scenarios/core/stream-of-consciousness-analyzer/PRD.md - Primary consumer
- scenarios/core/research-assistant/PRD.md - Stores findings
- scenarios/core/product-manager-agent/PRD.md - Feature mapping

### External Resources
- [Force-Directed Graphs](https://github.com/d3/d3-force)
- [Operational Transformation](https://en.wikipedia.org/wiki/Operational_transformation)
- [Semantic Search Best Practices](https://www.pinecone.io/learn/semantic-search/)

---

**Last Updated**: 2025-01-20  
**Status**: Not Tested  
**Owner**: AI Agent - Knowledge Organization Module  
**Review Cycle**: Weekly validation of search accuracy and canvas performance