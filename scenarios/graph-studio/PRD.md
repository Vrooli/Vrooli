# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Canonical Reference**: PRD Control Tower
> **Status**: Active

## 🎯 Overview

Graph-studio provides a unified, extensible platform for creating, validating, converting, and managing all forms of graph-based data structures and visualizations. It transforms disparate graph formats into a coherent intelligence layer that understands relationships, hierarchies, processes, and semantic connections across any domain.

**Purpose**: Add permanent capability for modeling any relationship-based problem using appropriate graph types (mind maps, BPMN, RDF, etc.) with automatic format conversion and validation.

**Primary Users**: Software engineers, business analysts, data scientists, knowledge workers, and AI agents building graph-based solutions.

**Deployment Surfaces**: CLI, API, UI dashboard, plugin system for extensibility.

**Intelligence Amplification**: Every graph created becomes searchable knowledge that enhances pattern recognition and problem-solving across all scenarios. Format translation enables agents to use the best representation for each task while maintaining interoperability.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Plugin architecture | Dynamic loading of graph type modules with base interface and isolation
- [x] OT-P0-002 | Core API for graph CRUD | Create, read, update, delete, validate operations via REST API
- [x] OT-P0-003 | Mind map plugin | FreeMind (.mm) format support with hierarchical visualization
- [x] OT-P0-004 | Process modeling plugin | BPMN 2.0 support for business workflow diagrams
- [x] OT-P0-005 | Graph conversion engine | Compatible format translations maintaining data integrity
- [x] OT-P0-006 | PostgreSQL storage | Graph metadata and relationships with JSONB data structures
- [x] OT-P0-007 | Dashboard UI | Visual gallery showing all available graph types with descriptions
- [x] OT-P0-008 | CLI interface | Programmatic graph operations (create, list, validate, convert, render)

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Mermaid diagram plugin | Lightweight visualizations for markdown-embedded diagrams
- [x] OT-P1-002 | Cytoscape.js plugin | Network/graph visualizations with force-directed layouts
- [x] OT-P1-003 | Graph search interface | Full-text search across name, description, and tags
- [ ] OT-P1-004 | UML plugin | Class, sequence, and activity diagram support
- [ ] OT-P1-005 | RDF/OWL plugin | Semantic web graphs with ontology support
- [ ] OT-P1-006 | GraphML/GEXF export | Gephi compatibility for network analysis
- [ ] OT-P1-007 | Real-time collaboration | WebSocket-based collaborative editing
- [ ] OT-P1-008 | Advanced query interface | SPARQL for RDF, GraphQL for other formats

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | XMind and CmapTools support | Additional mind mapping format compatibility
- [ ] OT-P2-002 | DMN plugin | Decision Model Notation for business rules
- [ ] OT-P2-003 | CMMN plugin | Case Management Model and Notation
- [ ] OT-P2-004 | SysML and ArchiMate | Systems engineering and enterprise architecture
- [ ] OT-P2-005 | D3.js custom builder | Custom visualization templates with interactive controls
- [ ] OT-P2-006 | AI-powered suggestions | Auto-completion and intelligent graph recommendations
- [ ] OT-P2-007 | Version control | Git-style diff visualization for graphs

## 🧱 Tech Direction Snapshot

**Preferred Stacks**:
- API: Go with Gin framework for performance and concurrency
- UI: React with Vite, TypeScript for type safety
- Storage: PostgreSQL with JSONB for flexible graph data structures
- Visualization: Plugin-based rendering (Cytoscape.js, Mermaid, D3.js)

**Data Storage Expectations**:
- Graph metadata and relationships in PostgreSQL
- Large graph files and assets in MinIO (S3-compatible)
- Optional caching layer via Redis for frequently accessed graphs

**Integration Strategy**:
- Direct API access for other scenarios
- CLI commands for automation and scripting
- Shared workflows for common operations
- Plugin SDK for custom graph type development

**Non-Goals**:
- Not a replacement for specialized tools like Visio or Lucidchart
- Not providing real-time 3D rendering (use existing tools for that)
- Not managing version control beyond basic versioning (use git for complex workflows)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- postgres: Database initialization required before first run
- minio: S3-compatible storage for large files and assets

**Optional Local Resources**:
- ollama: AI-powered graph suggestions and natural language queries (fallback: manual creation)
- redis: Caching layer for performance (fallback: direct database queries)
- qdrant: Vector search for semantic similarity (fallback: keyword search only)

**Launch Sequencing**:
1. Ensure postgres resource is running and initialized
2. Apply database schema via initialization/postgres/schema.sql
3. Start API service with environment configuration
4. Seed initial plugin registry
5. Launch UI dashboard for visual access

**Risks**:
- Plugin isolation may add complexity vs. monolithic approach
- Large graph rendering (10,000+ nodes) requires pagination and progressive loading
- Format conversion may lose format-specific features (preserve originals for round-trip)

## 🎨 UX & Branding

**Visual Palette**: Clean, professional workspace with creative flexibility. Light mode default with dark mode toggle. Nature-inspired soft gradients for mind maps, corporate blues/grays for BPMN, customizable palettes for network graphs.

**Typography Tone**: Modern, readable fonts. Clear hierarchy with generous whitespace. Technical precision without overwhelming users.

**Accessibility Commitments**: WCAG 2.1 Level AA compliance. High contrast mode support. Keyboard navigation for all operations. Screen reader compatible labels and ARIA attributes.

**Voice/Personality**: Friendly but professional. Approachable expert who empowers users to visualize any idea. Focused productivity feel like "Notion + Miro hybrid."

**Target Feeling**: Users should feel empowered to model complex relationships visually without needing multiple specialized tools. The interface should get out of the way and let the graphs shine.

**Dashboard Design Principles**:
- Plugin gallery with visual cards showing example graphs
- Quick actions with smart type suggestions based on description
- Recent graphs with thumbnail previews and timestamps
- Conversion matrix showing which formats can convert to others
- Natural language search across all graphs and types

## 📎 Appendix

### Cross-Scenario Interactions

**Provides to**:
- research-assistant: Visualize research connections as knowledge graphs via API
- product-manager: Create and manage product roadmap diagrams via CLI
- system-monitor: Visualize system dependencies and data flows with real-time updates

**Consumes from**:
- stream-of-consciousness-analyzer: Convert unstructured thoughts to mind maps via text transformation API

### Evolution Path

**Version 1.0 (Current)**: Core plugin architecture, essential graph types (mind maps, BPMN, network graphs), basic conversion capabilities, PostgreSQL storage.

**Version 2.0 (Planned)**: Real-time collaboration, AI-powered auto-completion, advanced query languages (SPARQL, GraphQL), 3D graph visualizations.

**Long-term Vision**: Become the "Figma for structured thinking" supporting 50+ graph formats with ML-based pattern recognition and automatic graph generation from any data source.

### Known Limitations

- **Large Graphs**: Rendering 10,000+ nodes requires pagination (workaround: progressive loading and viewport culling; future: WebGL-based engine)
- **Format Fidelity**: Some conversions may lose format-specific features (workaround: preserve original for round-trip; future: richer intermediate representation)

### Security Considerations

- Data protection for sensitive business processes
- Per-graph permissions with organization support
- Audit trail for all modifications with user attribution
- Rate limiting and request size limits to prevent DoS

### External Resources

- [BPMN 2.0 Specification](https://www.omg.org/spec/BPMN/2.0/)
- [W3C RDF Primer](https://www.w3.org/TR/rdf-primer/)
- [Cytoscape.js Documentation](https://js.cytoscape.org/)
- [FreeMind File Format](https://freemind.sourceforge.io/wiki/index.php/File_format)
