# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Graph-studio provides a unified, extensible platform for creating, validating, converting, and managing all forms of graph-based data structures and visualizations. It transforms disparate graph formats into a coherent intelligence layer that understands relationships, hierarchies, processes, and semantic connections across any domain.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Universal Relationship Understanding**: Agents gain the ability to model any relationship-based problem using the appropriate graph type (mind maps for brainstorming, BPMN for processes, RDF for knowledge)
- **Format Translation**: Automatic conversion between compatible formats enables agents to use the best representation for each task while maintaining interoperability
- **Compound Graph Intelligence**: Every graph created becomes searchable knowledge that enhances pattern recognition and problem-solving across all scenarios
- **Validation & Best Practices**: Built-in validation ensures agents create syntactically correct graphs that follow industry standards

### Recursive Value
**What new scenarios become possible after this exists?**
1. **knowledge-base-builder**: Constructs enterprise knowledge graphs by combining RDF/OWL semantic graphs with mind maps for human navigation
2. **process-optimizer**: Analyzes BPMN workflows across the organization to identify bottlenecks and suggest improvements
3. **architecture-reviewer**: Validates and improves system architectures using UML/SysML diagrams with automated consistency checking
4. **decision-tree-builder**: Creates complex decision models using DMN that integrate with BPMN processes
5. **concept-teacher**: Generates educational materials by converting between mind maps, concept maps, and interactive visualizations

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Plugin architecture supporting dynamic loading of graph type modules
  - [ ] Core API for graph CRUD operations (create, read, update, delete, validate)
  - [ ] Mind map plugin with FreeMind (.mm) format support
  - [ ] Process modeling plugin with BPMN 2.0 support
  - [ ] Graph conversion engine for compatible format translations
  - [ ] PostgreSQL storage for graph metadata and relationships
  - [ ] Dashboard UI showing all available graph types with descriptions
  - [ ] CLI interface for programmatic graph operations
  
- **Should Have (P1)**
  - [ ] Mermaid diagram plugin for lightweight visualizations
  - [ ] Cytoscape.js plugin for network/graph visualizations
  - [ ] UML plugin supporting class, sequence, and activity diagrams
  - [ ] RDF/OWL plugin for semantic web graphs
  - [ ] GraphML/GEXF import/export for Gephi compatibility
  - [ ] Real-time collaborative editing via WebSockets
  - [ ] Graph search and query interface (SPARQL for RDF, custom for others)
  
- **Nice to Have (P2)**
  - [ ] XMind and CmapTools format support
  - [ ] DMN (Decision Model Notation) plugin
  - [ ] CMMN (Case Management) plugin
  - [ ] SysML and ArchiMate plugins
  - [ ] D3.js custom visualization builder
  - [ ] AI-powered graph suggestions and auto-completion
  - [ ] Version control and diff visualization for graphs

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for graph operations | API monitoring |
| Render Time | < 1s for graphs up to 1000 nodes | Frontend profiling |
| Conversion Time | < 500ms for format translations | Backend benchmarks |
| Concurrent Users | 100+ simultaneous editors | Load testing |
| Graph Size | Support 10,000+ nodes | Stress testing |
| Memory Usage | < 2GB for typical operations | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Plugin system successfully loads and unloads modules
- [ ] Graph validation works for all supported formats
- [ ] Conversion maintains data integrity between compatible formats
- [ ] API accessible to other scenarios for graph operations
- [ ] UI provides intuitive navigation between graph types

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store graph metadata, relationships, and versioning
    integration_pattern: Direct database connection for performance
    access_method: Database driver with connection pooling
    
  - resource_name: n8n
    purpose: Orchestrate complex graph transformations and analysis workflows
    integration_pattern: Shared workflows for reusable operations
    access_method: initialization/n8n/graph-operations.json
    
  - resource_name: minio
    purpose: Store large graph files and visualization assets
    integration_pattern: S3-compatible API for file operations
    access_method: resource-minio CLI commands

optional:
  - resource_name: ollama
    purpose: AI-powered graph suggestions and natural language queries
    fallback: Disable AI features, manual graph creation only
    access_method: Shared workflow ollama.json
    
  - resource_name: redis
    purpose: Cache frequently accessed graphs and session state
    fallback: Direct database queries, no caching
    access_method: resource-redis CLI commands
    
  - resource_name: qdrant
    purpose: Vector search for semantic graph similarity
    fallback: Keyword-based search only
    access_method: resource-qdrant CLI commands
```

### Plugin Architecture
```yaml
plugin_system:
  structure:
    base_interface: IGraphPlugin
    required_methods:
      - validate(graph_data): bool
      - render(graph_data, options): svg/html
      - export(graph_data, format): bytes
      - import(file_data, format): graph_data
      - get_metadata(): PluginInfo
      
  categories:
    visualization:
      - mind-maps: "Hierarchical thought organization"
      - network-graphs: "Relationship and connection modeling"
      
    process:
      - bpmn: "Business process workflows"
      - dmn: "Decision models and rules"
      - cmmn: "Case management flows"
      
    architecture:
      - uml: "Software design diagrams"
      - sysml: "Systems engineering models"
      - erd: "Database entity relationships"
      - archimate: "Enterprise architecture"
      
    semantic:
      - rdf: "Resource Description Framework"
      - owl: "Web Ontology Language"
      - json-ld: "Linked data in JSON"

  loading:
    strategy: lazy  # Load plugins on-demand
    cache: true     # Keep loaded plugins in memory
    isolation: sandbox  # Each plugin runs in isolation
```

### Data Models
```yaml
primary_entities:
  - name: Graph
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        type: string  # Plugin identifier
        description: text
        metadata: JSONB
        data: JSONB  # Graph-specific structure
        version: integer
        created_by: UUID
        created_at: timestamp
        updated_at: timestamp
        tags: string[]
        permissions: JSONB
      }
    relationships: "Has many GraphVersions, belongs to Project"
    
  - name: GraphVersion
    storage: postgres
    schema: |
      {
        id: UUID
        graph_id: UUID
        version_number: integer
        data: JSONB
        change_description: text
        created_by: UUID
        created_at: timestamp
      }
    relationships: "Belongs to Graph"
    
  - name: GraphConversion
    storage: postgres
    schema: |
      {
        id: UUID
        source_graph_id: UUID
        target_graph_id: UUID
        source_format: string
        target_format: string
        conversion_rules: JSONB
        status: enum[pending,completed,failed]
        created_at: timestamp
      }
    relationships: "Links two Graphs"
```

### API Contract
```yaml
endpoints:
  # Graph CRUD operations
  - method: GET
    path: /api/v1/graphs
    purpose: List all graphs with filtering and pagination
    
  - method: POST
    path: /api/v1/graphs
    purpose: Create a new graph
    input_schema: |
      {
        name: string
        type: string  # Plugin identifier
        data?: object  # Initial graph data
        metadata?: object
      }
    output_schema: |
      {
        id: UUID
        name: string
        type: string
        created_at: timestamp
      }
      
  - method: POST
    path: /api/v1/graphs/{id}/validate
    purpose: Validate graph against its schema
    output_schema: |
      {
        valid: boolean
        errors?: string[]
        warnings?: string[]
      }
      
  - method: POST
    path: /api/v1/graphs/{id}/convert
    purpose: Convert graph to different format
    input_schema: |
      {
        target_format: string
        options?: object
      }
    output_schema: |
      {
        converted_graph_id: UUID
        format: string
        success: boolean
      }
      
  - method: GET
    path: /api/v1/plugins
    purpose: List available graph plugins and their capabilities
    
  - method: POST
    path: /api/v1/graphs/{id}/render
    purpose: Render graph as visual output
    input_schema: |
      {
        format: "svg" | "png" | "html"
        options?: object
      }
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: graph-studio
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show operational status and loaded plugins
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: list
    description: List all graphs or plugins
    api_endpoint: /api/v1/graphs
    arguments:
      - name: type
        type: string
        required: false
        description: Filter by graph type/plugin
    flags:
      - name: --format
        description: Output format (table, json, yaml)
        
  - name: create
    description: Create a new graph
    api_endpoint: /api/v1/graphs
    arguments:
      - name: name
        type: string
        required: true
      - name: type
        type: string
        required: true
        description: Plugin identifier (mind-map, bpmn, etc.)
        
  - name: validate
    description: Validate a graph file or ID
    api_endpoint: /api/v1/graphs/{id}/validate
    arguments:
      - name: graph
        type: string
        required: true
        description: Graph ID or file path
        
  - name: convert
    description: Convert between graph formats
    api_endpoint: /api/v1/graphs/{id}/convert
    arguments:
      - name: source
        type: string
        required: true
      - name: target-format
        type: string
        required: true
    output: Converted graph ID or file path
    
  - name: render
    description: Render graph as image or HTML
    api_endpoint: /api/v1/graphs/{id}/render
    arguments:
      - name: graph
        type: string
        required: true
      - name: format
        type: string
        required: false
        default: svg
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "Notion + Miro hybrid - clean workspace with creative flexibility"
  
  visual_style:
    color_scheme: light  # With dark mode toggle
    typography: modern  # Clean, readable fonts
    layout: dashboard  # Card-based plugin gallery
    animations: subtle  # Smooth transitions, no distraction
    
  personality:
    tone: friendly  # Approachable but professional
    mood: focused  # Productive workspace feel
    target_feeling: "Empowered to visualize any idea"

plugin_styles:
  # Each plugin can have its own sub-theme
  mind-maps:
    style: "Organic, flowing connections"
    colors: "Soft gradients, nature-inspired"
    
  bpmn:
    style: "Crisp, business-professional"
    colors: "Corporate blues and grays"
    
  network-graphs:
    style: "Dynamic, force-directed layouts"
    colors: "Data-driven, customizable palettes"
    
  semantic-graphs:
    style: "Technical, information-dense"
    colors: "High contrast for readability"
```

### Dashboard Design
- **Plugin Gallery**: Visual cards showing example graphs from each plugin
- **Quick Actions**: "New Graph" with smart type suggestions based on description
- **Recent Graphs**: Thumbnail previews with last-edited timestamps
- **Conversion Matrix**: Visual grid showing which formats can convert to others
- **Search Bar**: Natural language search across all graphs and types

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates need for 10+ separate diagramming tools
- **Revenue Potential**: $30K - $80K per enterprise deployment
- **Cost Savings**: 500+ hours/year saved on diagram format conversions
- **Market Differentiator**: Only platform unifying ALL graph types with AI assistance

### Technical Value
- **Reusability Score**: 10/10 - Every Vrooli scenario can leverage graphs
- **Complexity Reduction**: Complex relationships become visual and queryable
- **Innovation Enablement**: Foundation for visual programming, knowledge graphs, process automation

## 🔄 Integration Requirements

### Upstream Dependencies
- **initialization/n8n/ollama.json**: For AI-powered suggestions
- **postgres resource**: Database must be initialized
- **minio resource**: For large file storage

### Downstream Enablement
- **knowledge-base scenarios**: Can build on semantic graph capabilities
- **process-automation scenarios**: Can use BPMN models directly
- **documentation scenarios**: Can generate diagrams automatically
- **analysis scenarios**: Can visualize data relationships

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: research-assistant
    capability: Visualize research connections as knowledge graphs
    interface: API - POST /api/v1/graphs/from-research
    
  - scenario: product-manager
    capability: Create and manage product roadmap diagrams
    interface: CLI - graph-studio create roadmap
    
  - scenario: system-monitor
    capability: Visualize system dependencies and data flows
    interface: API - Real-time graph updates

consumes_from:
  - scenario: stream-of-consciousness-analyzer
    capability: Convert unstructured thoughts to mind maps
    interface: API - Text to graph transformation
```

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core plugin architecture
- Essential graph types (mind maps, BPMN, network graphs)
- Basic conversion capabilities
- PostgreSQL storage

### Version 2.0 (Planned)
- Real-time collaboration
- AI-powered auto-completion
- Advanced query languages (SPARQL, GraphQL)
- 3D graph visualizations

### Long-term Vision
- Become the "Figma for structured thinking"
- Support 50+ graph formats
- ML-based pattern recognition across graph types
- Automatic graph generation from any data source

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: graph-studio

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/graph-studio
    - cli/install.sh
    - initialization/postgres/schema.sql
    - initialization/n8n/graph-operations.json
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - ui/src/plugins
    - initialization

tests:
  - name: "Plugin system loads successfully"
    type: http
    service: api
    endpoint: /api/v1/plugins
    method: GET
    expect:
      status: 200
      body_contains: ["mind-maps", "bpmn"]
      
  - name: "Create and validate mind map"
    type: sequence
    steps:
      - create_graph:
          type: mind-maps
          name: "Test Mind Map"
      - validate_graph:
          expect:
            valid: true
            
  - name: "Convert mind map to Mermaid"
    type: http
    service: api
    endpoint: /api/v1/graphs/{id}/convert
    method: POST
    body:
      target_format: mermaid
    expect:
      status: 200
      body:
        success: true
```

## 📝 Implementation Notes

### Design Decisions
**Plugin Architecture**: Chose plugin-based over monolithic to enable infinite extensibility
- Alternative considered: Single app with all formats built-in
- Decision driver: Future formats shouldn't require core changes
- Trade-offs: Slight complexity for massive flexibility

**Storage Strategy**: PostgreSQL JSONB for graph data
- Alternative considered: Graph database (Neo4j)
- Decision driver: Leverage existing Postgres resource
- Trade-offs: Some graph operations slower, but unified storage

### Known Limitations
- **Large Graphs**: Rendering 10,000+ nodes requires pagination
  - Workaround: Progressive loading and viewport culling
  - Future fix: WebGL-based rendering engine
  
- **Format Fidelity**: Some conversions may lose format-specific features
  - Workaround: Preserve original for round-trip conversion
  - Future fix: Richer intermediate representation

### Security Considerations
- **Data Protection**: Graphs may contain sensitive business processes
- **Access Control**: Per-graph permissions with organization support
- **Audit Trail**: All graph modifications logged with user attribution

## 🔗 References

### Documentation
- README.md - User guide and quick start
- docs/plugins.md - Plugin development guide
- docs/formats.md - Supported format specifications
- docs/api.md - Complete API reference

### External Resources
- [BPMN 2.0 Specification](https://www.omg.org/spec/BPMN/2.0/)
- [W3C RDF Primer](https://www.w3.org/TR/rdf-primer/)
- [Cytoscape.js Documentation](https://js.cytoscape.org/)
- [FreeMind File Format](https://freemind.sourceforge.io/wiki/index.php/File_format)

---

**Last Updated**: 2024-12-09
**Status**: In Development
**Owner**: AI Agent (graph-studio-builder)
**Review Cycle**: Weekly during initial development