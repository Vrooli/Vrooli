# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Database schema visualization and intelligent query building - a permanent database intelligence layer that provides visual ER diagrams, natural language to SQL translation, and accumulated query knowledge that all scenarios can leverage to understand and interact with their data structures.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability transforms database interactions from requiring SQL expertise to natural language conversations. Every query becomes reusable knowledge stored in Qdrant, creating a growing library of proven query patterns. Agents can visually understand relationships between data entities, detect schema inconsistencies, and leverage accumulated query performance insights to optimize their own database operations.

### Recursive Value
**What new scenarios become possible after this exists?**
- **automated-migration-manager**: Can analyze schema differences and generate migration scripts
- **data-quality-monitor**: Leverages schema understanding to detect data anomalies
- **api-generator**: Automatically creates REST/GraphQL APIs from schema visualization
- **test-data-factory**: Uses schema knowledge to generate realistic test datasets
- **performance-optimizer**: Analyzes query patterns to suggest index improvements

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Connect to any Postgres instance and generate interactive ER diagrams
  - [ ] Natural language to SQL query builder with explanation
  - [ ] Query history storage in Qdrant for reusability
  - [ ] Support multiple database connections (main + scenario DBs)
  - [ ] Export schema as SQL DDL, Markdown documentation, or image
  - [ ] Real-time schema introspection and visualization
  
- **Should Have (P1)**
  - [ ] Schema diff detection between databases/scenarios
  - [ ] Multiple layout modes (compact, detailed, relationship-focused)
  - [ ] Query performance tracking and analysis
  - [ ] Collaborative annotations and documentation
  - [ ] Table/column search and filtering
  - [ ] Foreign key relationship navigation
  
- **Nice to Have (P2)**
  - [ ] AI-powered schema optimization recommendations
  - [ ] Embeddable diagram widgets for other scenarios
  - [ ] Migration script generation between schema versions
  - [ ] Support for MongoDB, Redis, and other databases
  - [ ] Query result visualization (charts, graphs)
  - [ ] Schema versioning and history tracking

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for schema load | API monitoring |
| Query Generation | < 2s for AI SQL generation | End-to-end timing |
| Diagram Render | < 1s for 100 tables | Frontend performance metrics |
| Concurrent Users | 50+ simultaneous connections | Load testing |
| Query Cache Hit Rate | > 60% for common patterns | Qdrant analytics |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with Postgres, Qdrant, and Ollama
- [ ] Schema visualization accurate for complex relationships
- [ ] Natural language queries produce valid SQL 95%+ of the time
- [ ] CLI and API provide identical functionality

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Primary database for schema introspection and query execution
    integration_pattern: CLI for schema inspection, API for query execution
    access_method: resource-postgres command for management, direct connection for queries
    
  - resource_name: qdrant
    purpose: Vector storage for query history and semantic search
    integration_pattern: API for embedding storage and retrieval
    access_method: Direct API calls for vector operations
    
  - resource_name: ollama
    purpose: Natural language to SQL translation and query explanation
    integration_pattern: Shared n8n workflow
    access_method: initialization/n8n/ollama.json workflow
    
optional:
  - resource_name: redis
    purpose: Query result caching for performance
    fallback: In-memory cache with shorter TTL
    access_method: resource-redis CLI commands
    
  - resource_name: browserless
    purpose: Export diagrams as high-quality images
    fallback: Canvas-based client-side export
    access_method: resource-browserless screenshot command
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: Natural language to SQL translation
    - workflow: embedding-generator.json
      location: initialization/n8n/
      purpose: Generate embeddings for query similarity
  
  2_resource_cli:
    - command: resource-postgres list-databases
      purpose: Discover available databases
    - command: resource-postgres execute-query
      purpose: Run generated SQL queries
  
  3_direct_api:
    - justification: Real-time schema introspection requires direct connection
      endpoint: postgres://localhost:5432/database
```

### Data Models
```yaml
primary_entities:
  - name: SchemaSnapshot
    storage: postgres
    schema: |
      {
        id: UUID
        database_name: string
        timestamp: datetime
        tables: JSON
        relationships: JSON
        indexes: JSON
        version: string
      }
    relationships: Contains multiple QueryHistory records
    
  - name: QueryHistory
    storage: qdrant
    schema: |
      {
        id: UUID
        natural_language: string
        generated_sql: string
        execution_time: float
        result_count: integer
        embedding: vector[768]
        database_context: string
        user_feedback: string
      }
    relationships: Associated with SchemaSnapshot
    
  - name: VisualizationLayout
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        database_name: string
        layout_data: JSON
        view_type: enum[compact|detailed|relationship]
        created_by: string
        shared: boolean
      }
    relationships: References SchemaSnapshot
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/schema/connect
    purpose: Connect to a database and retrieve schema
    input_schema: |
      {
        connection_string: string
        database_name: string
      }
    output_schema: |
      {
        tables: Table[]
        relationships: Relationship[]
        diagram_data: object
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/query/generate
    purpose: Convert natural language to SQL
    input_schema: |
      {
        natural_language: string
        database_context: string
        include_explanation: boolean
      }
    output_schema: |
      {
        sql: string
        explanation: string
        confidence: float
        similar_queries: Query[]
      }
    sla:
      response_time: 2000ms
      availability: 99.5%
      
  - method: POST
    path: /api/v1/query/execute
    purpose: Execute SQL and return results
    input_schema: |
      {
        sql: string
        database_name: string
        limit: integer
      }
    output_schema: |
      {
        columns: Column[]
        rows: any[][]
        execution_time: float
        row_count: integer
      }
    sla:
      response_time: 5000ms
      availability: 99.9%
```

### Event Interface
```yaml
published_events:
  - name: schema.analysis.completed
    payload: {database_name: string, table_count: integer, issues: Issue[]}
    subscribers: [test-genie, app-debugger, migration-manager]
    
  - name: query.pattern.discovered
    payload: {pattern: string, frequency: integer, performance: object}
    subscribers: [performance-optimizer, test-data-factory]
    
consumed_events:
  - name: scenario.database.created
    action: Automatically connect and visualize new database
    
  - name: performance.issue.detected
    action: Analyze related queries and suggest optimizations
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: db-schema-explorer
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
  - name: connect
    description: Connect to a database and visualize schema
    api_endpoint: /api/v1/schema/connect
    arguments:
      - name: database
        type: string
        required: true
        description: Database name or connection string
    flags:
      - name: --output
        description: Output format (terminal|json|image)
      - name: --save-layout
        description: Save visualization layout with name
    output: ER diagram visualization or JSON schema
    
  - name: query
    description: Generate and execute SQL from natural language
    api_endpoint: /api/v1/query/generate
    arguments:
      - name: prompt
        type: string
        required: true
        description: Natural language query description
    flags:
      - name: --database
        description: Target database (defaults to current)
      - name: --explain
        description: Include query explanation
      - name: --execute
        description: Run the generated query
    output: Generated SQL and optionally results
    
  - name: diff
    description: Compare schemas between databases
    api_endpoint: /api/v1/schema/diff
    arguments:
      - name: source
        type: string
        required: true
        description: Source database
      - name: target
        type: string
        required: true
        description: Target database
    flags:
      - name: --format
        description: Output format (summary|detailed|sql)
    output: Schema differences report
    
  - name: export
    description: Export schema documentation
    api_endpoint: /api/v1/schema/export
    arguments:
      - name: database
        type: string
        required: true
        description: Database to export
    flags:
      - name: --format
        description: Export format (sql|markdown|dbml|image)
      - name: --output
        description: Output file path
    output: Exported schema in specified format
```

### CLI-API Parity Requirements
- **Coverage**: Every API endpoint has corresponding CLI command
- **Naming**: CLI uses intuitive verbs (connect, query, diff, export)
- **Arguments**: Direct mapping to API parameters
- **Output**: Human-readable by default, --json for programmatic use
- **Authentication**: Inherits from VROOLI_DB_* environment variables

## 🔄 Integration Requirements

### Upstream Dependencies
- **postgres resource**: Required for database connectivity and schema introspection
- **ollama resource**: Powers natural language understanding for query generation
- **qdrant resource**: Enables semantic search over query history
- **n8n workflows**: Provides reliable resource orchestration

### Downstream Enablement
- **automated-migration-manager**: Uses schema diff capabilities for migration generation
- **test-genie**: Leverages schema understanding for test data generation
- **api-generator**: Builds on schema visualization for API scaffolding
- **performance-optimizer**: Analyzes query patterns for optimization

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: app-debugger
    capability: Visual database state inspection during debugging
    interface: API - /api/v1/schema/snapshot
    
  - scenario: test-genie
    capability: Schema-aware test data generation
    interface: CLI - db-schema-explorer export --format json
    
  - scenario: product-manager
    capability: Database complexity metrics for planning
    interface: Event - schema.analysis.completed
    
consumes_from:
  - scenario: prompt-manager
    capability: Optimized prompts for SQL generation
    fallback: Use default prompts if unavailable
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Modern database tools like ChartDB with Vrooli personality
  
  visual_style:
    color_scheme: dark  # Developer-friendly dark theme
    typography: technical  # Monospace for SQL, clean sans-serif for UI
    layout: dashboard  # Multi-panel with diagram, query builder, results
    animations: subtle  # Smooth transitions, no distraction
  
  personality:
    tone: technical  # Professional but approachable
    mood: focused  # Productivity-oriented
    target_feeling: "In control of complex data"

ui_components:
  - Interactive ER diagram with zoom/pan/search
  - Split-pane query editor with syntax highlighting
  - Tabbed result viewer with export options
  - Floating AI assistant for natural language input
  - History sidebar with semantic search
```

### Target Audience Alignment
- **Primary Users**: Developers, data analysts, QA engineers
- **User Expectations**: Professional tool with powerful features
- **Accessibility**: WCAG 2.1 AA compliance, keyboard navigation
- **Responsive Design**: Desktop-first, tablet-compatible

## 💰 Value Proposition

### Business Value
- **Primary Value**: Reduces database exploration time by 80%
- **Revenue Potential**: $15K - $30K per enterprise deployment
- **Cost Savings**: 10+ hours/week saved on database tasks
- **Market Differentiator**: AI-powered query generation with learning

### Technical Value
- **Reusability Score**: 9/10 - Every data-driven scenario benefits
- **Complexity Reduction**: SQL expertise no longer required
- **Innovation Enablement**: Unlocks automated schema management

## 🧬 Evolution Path

### Version 1.0 (Current)
- Postgres schema visualization
- Natural language to SQL
- Query history with Qdrant
- Basic export capabilities

### Version 2.0 (Planned)
- Multi-database support (MongoDB, Redis)
- Collaborative features
- Advanced performance analytics
- Schema version control

### Long-term Vision
- Universal data intelligence layer for Vrooli
- Self-optimizing based on usage patterns
- Cross-scenario data lineage tracking
- Automated data governance

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with resource configuration
    - Schema initialization scripts
    - n8n workflow definitions
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose with persistent volumes
    - kubernetes: StatefulSet for data persistence
    - cloud: RDS integration for AWS/GCP/Azure
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        - starter: $99/month (5 databases)
        - professional: $299/month (unlimited databases)
        - enterprise: Custom pricing with support
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: db-schema-explorer
    category: analysis
    capabilities:
      - Database schema visualization
      - Natural language to SQL
      - Query pattern learning
      - Schema comparison
    interfaces:
      - api: http://localhost:${API_PORT}/api/v1
      - cli: db-schema-explorer
      - events: schema.*, query.*
      
  metadata:
    description: Visual database exploration with AI-powered queries
    keywords: [database, schema, ER diagram, SQL, query builder, postgres]
    dependencies: [postgres, qdrant, ollama]
    enhances: [app-debugger, test-genie, migration-manager]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Large schema performance | Medium | High | Implement virtual scrolling and lazy loading |
| SQL injection | Low | Critical | Parameterized queries, read-only connections |
| Resource exhaustion | Medium | Medium | Query timeouts and result limits |

### Operational Risks
- **Schema Drift**: Regular snapshot comparison with alerts
- **Query Performance**: Automatic slow query detection
- **Data Privacy**: Connection encryption, audit logging

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: db-schema-explorer

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/db-schema-explorer
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/schema-inspector.json
    - test/run-tests.sh
    
  required_dirs:
    - api
    - cli
    - ui
    - initialization/automation/n8n
    - initialization/storage/postgres
    - test/phases

resources:
  required: [postgres, qdrant, ollama, n8n]
  optional: [redis, browserless]
  health_timeout: 60

tests:
  - name: "Postgres connection works"
    type: sql
    service: postgres
    query: "SELECT version()"
    expect:
      rows_returned: 1
      
  - name: "API generates SQL from natural language"
    type: http
    service: api
    endpoint: /api/v1/query/generate
    method: POST
    body:
      natural_language: "show all tables"
      database_context: "public"
    expect:
      status: 200
      body:
        sql: ~"SELECT.*FROM.*information_schema.tables"
        
  - name: "CLI exports schema"
    type: exec
    command: ./cli/db-schema-explorer export main --format json
    expect:
      exit_code: 0
      output_contains: ["tables", "relationships"]
```

### Performance Validation
- [ ] Schema loads in < 500ms for 100 tables
- [ ] Query generation completes in < 2s
- [ ] No memory leaks after 1000 queries
- [ ] Supports 50 concurrent users

## 📝 Implementation Notes

### Design Decisions
**React + ReactFlow for diagrams**: Chosen for interactivity and customization
- Alternative considered: D3.js
- Decision driver: Better developer experience and component reusability
- Trade-offs: Larger bundle size for better UX

**Qdrant for query history**: Vector similarity enables semantic search
- Alternative considered: PostgreSQL with pg_vector
- Decision driver: Better performance and native vector operations
- Trade-offs: Additional resource dependency

### Known Limitations
- **Initial version Postgres-only**: Multi-database support in v2
  - Workaround: Use separate instances for different DB types
  - Future fix: Abstract database adapter interface

### Security Considerations
- **Data Protection**: Read-only connections by default
- **Access Control**: Role-based permissions per database
- **Audit Trail**: All queries logged with user context

## 🔗 References

### Documentation
- README.md - Quick start guide
- docs/api.md - Complete API reference
- docs/cli.md - CLI command documentation
- docs/query-patterns.md - Common query examples

### Related PRDs
- scenarios/app-debugger/PRD.md
- scenarios/test-genie/PRD.md
- scenarios/migration-manager/PRD.md (future)

### External Resources
- [ChartDB GitHub](https://github.com/chartdb/chartdb)
- [Azimutt Documentation](https://azimutt.app/docs)
- [PostgreSQL Information Schema](https://www.postgresql.org/docs/current/information-schema.html)

---

**Last Updated**: 2025-01-06  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Weekly validation against implementation
