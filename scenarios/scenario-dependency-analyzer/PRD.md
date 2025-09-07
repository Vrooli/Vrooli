# Product Requirements Document (PRD) - Scenario Dependency Analyzer

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
This scenario adds **meta-intelligence for deployment optimization and capability planning** - the ability to automatically analyze, visualize, and optimize the dependency relationships between scenarios and resources. It enables Vrooli to understand its own architecture at a deep level and make intelligent decisions about deployments, resource allocation, and scenario development priorities.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability transforms Vrooli from a collection of individual scenarios into a **self-aware intelligence system** that:
- **Optimizes resource usage** by identifying redundancies and suggesting lightweight alternatives
- **Accelerates scenario development** by revealing missing prerequisite capabilities
- **Enables surgical deployments** where only necessary components are included
- **Provides strategic planning intelligence** for capability gap analysis
- **Creates compound learning** where each new scenario's dependencies become knowledge for future scenarios

### Recursive Value
**What new scenarios become possible after this exists?**
1. **deployment-optimizer** - Intelligent deployment system that auto-selects optimal resource configurations
2. **capability-planner** - Strategic planning for scenario development roadmaps
3. **resource-cost-analyzer** - Economic optimization of resource usage across deployments  
4. **scenario-merger** - Identifies opportunities to combine complementary scenarios
5. **architecture-health-monitor** - Continuous monitoring of system dependency health

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Automatically parse existing scenarios and extract all resource dependencies
  - [ ] Detect inter-scenario dependencies (CLI calls, API usage, shared workflows)
  - [ ] Store dependency metadata in standardized `dependencies.json` format
  - [ ] Provide visualization of dependency graphs with interactive UI
  - [ ] Integration with resource-claude-code for analyzing proposed scenarios
  - [ ] Integration with resource-qdrant for semantic similarity matching
  - [ ] CLI interface for programmatic access to dependency data
  - [ ] API endpoints for other scenarios to query dependency information
  
- **Should Have (P1)**
  - [ ] Optimization recommendations (e.g., ollama → openrouter for lightweight deployments)
  - [ ] Dependency impact analysis (what breaks if resource X is removed)
  - [ ] Historical tracking of dependency changes over time
  - [ ] Export dependency graphs to various formats (GraphViz, JSON, PNG)
  - [ ] Automated detection of circular dependencies
  - [ ] Resource cost estimation based on dependency depth
  
- **Nice to Have (P2)**
  - [ ] ML-powered prediction of likely dependencies for new scenarios
  - [ ] Automated refactoring suggestions to reduce dependency complexity
  - [ ] Integration with CI/CD to validate dependency changes

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Dependency Scan Time | < 30s for all scenarios | CLI timing |
| Visualization Load Time | < 3s for graphs with 100+ nodes | UI monitoring |
| API Response Time | < 500ms for dependency queries | Load testing |
| Memory Usage | < 1GB during full system analysis | Process monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Successfully analyzes all existing scenarios without errors
- [ ] Visualization accurately represents actual dependencies
- [ ] Other scenarios can programmatically query dependency data
- [ ] Performance targets met under full system load

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store dependency metadata, analysis results, and historical data
    integration_pattern: Direct SQL queries and migrations
    access_method: resource-postgres CLI commands
    
  - resource_name: claude-code  
    purpose: AI-powered analysis of scenario code and configurations
    integration_pattern: CLI wrapper for code analysis tasks
    access_method: resource-claude-code analyze
    
  - resource_name: qdrant
    purpose: Semantic similarity matching for proposed scenarios
    integration_pattern: Vector storage and similarity queries
    access_method: resource-qdrant CLI commands

optional:
  - resource_name: redis
    purpose: Cache analysis results and frequently accessed dependency data
    fallback: Direct database queries with slightly higher latency
    access_method: redis-cli via resource wrapper
```

### Resource Integration Standards
```yaml
# Priority order for resource access:
integration_priorities:
  1_resource_cli:        # FIRST: Use resource CLI commands
    - command: resource-postgres execute
      purpose: Database operations for dependency storage
    - command: resource-claude-code analyze  
      purpose: AI-powered scenario analysis
    - command: resource-qdrant search
      purpose: Semantic matching for similar scenarios
  
  2_direct_api:          # SECOND: Direct API when CLI insufficient
    - justification: Complex graph visualizations require direct D3.js integration
      endpoint: Custom visualization endpoints
```

### Data Models
```yaml
primary_entities:
  - name: ScenarioDependency
    storage: postgres
    schema: |
      {
        id: UUID,
        scenario_name: STRING,
        dependency_type: ENUM(resource, scenario, shared_workflow),
        dependency_name: STRING,
        required: BOOLEAN,
        purpose: STRING,
        access_method: STRING,
        discovered_at: TIMESTAMP,
        last_verified: TIMESTAMP
      }
    relationships: Many scenarios can depend on many resources/scenarios
    
  - name: DependencyGraph  
    storage: postgres
    schema: |
      {
        id: UUID,
        graph_type: ENUM(resource, scenario, combined),
        nodes: JSONB,
        edges: JSONB,
        metadata: JSONB,
        created_at: TIMESTAMP
      }
    relationships: Represents computed dependency graphs
    
  - name: OptimizationRecommendation
    storage: postgres  
    schema: |
      {
        id: UUID,
        scenario_name: STRING,
        recommendation_type: ENUM(resource_swap, dependency_reduction, merger_opportunity),
        current_state: JSONB,
        recommended_state: JSONB,
        estimated_impact: JSONB,
        confidence_score: FLOAT
      }
    relationships: Links to scenarios and provides optimization insights
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/scenarios/{name}/dependencies
    purpose: Get dependency information for a specific scenario
    input_schema: |
      {
        name: STRING (scenario name),
        include_transitive: BOOLEAN (optional, default false)
      }
    output_schema: |
      {
        scenario: STRING,
        resources: [ScenarioDependency],
        scenarios: [ScenarioDependency],
        shared_workflows: [ScenarioDependency],
        transitive_depth: INTEGER
      }
    sla:
      response_time: 200ms
      availability: 99.9%

  - method: POST  
    path: /api/v1/analyze/proposed
    purpose: Analyze dependencies for a proposed scenario description
    input_schema: |
      {
        name: STRING,
        description: STRING,
        requirements: [STRING],
        similar_scenarios: [STRING] (optional)
      }
    output_schema: |
      {
        predicted_resources: [ResourcePrediction],
        similar_patterns: [SimilarityMatch], 
        recommendations: [DependencyRecommendation],
        confidence_scores: {resource: FLOAT, scenario: FLOAT}
      }

  - method: GET
    path: /api/v1/graph/{type}
    purpose: Get dependency graph data for visualization
    input_schema: |
      {
        type: ENUM(resources, scenarios, combined),
        format: ENUM(json, graphviz, d3),
        filter: STRING (optional scenario name filter)
      }
    output_schema: |
      {
        nodes: [GraphNode],
        edges: [GraphEdge],  
        metadata: {total_nodes, total_edges, complexity_score}
      }
```

### Event Interface
```yaml
published_events:
  - name: dependency.analysis.completed
    payload: {scenario: STRING, dependencies_found: INTEGER, analysis_time: INTEGER}
    subscribers: [deployment-optimizer, capability-planner]
    
  - name: dependency.optimization.identified
    payload: {scenario: STRING, optimization_type: STRING, potential_savings: OBJECT}
    subscribers: [ecosystem-manager, resource-cost-analyzer]
    
consumed_events:
  - name: scenario.created
    action: Automatically analyze dependencies for newly created scenarios
  - name: scenario.updated  
    action: Re-analyze dependencies when scenarios are modified
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: scenario-dependency-analyzer
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
  - name: analyze
    description: Analyze dependencies for specific scenario or all scenarios
    api_endpoint: /api/v1/analyze
    arguments:
      - name: scenario
        type: string
        required: false
        description: Specific scenario to analyze (or 'all' for full system)
    flags:
      - name: --output
        description: Output format (json, table, graph)
      - name: --include-transitive  
        description: Include transitive dependencies
    output: Dependency analysis results

  - name: graph
    description: Generate and display dependency graphs
    api_endpoint: /api/v1/graph  
    arguments:
      - name: type
        type: string
        required: true
        description: Graph type (resources, scenarios, combined)
    flags:
      - name: --format
        description: Output format (json, png, svg, html)
      - name: --filter
        description: Filter by scenario name pattern
      - name: --output-file
        description: Save graph to file
    output: Dependency graph visualization

  - name: optimize
    description: Get optimization recommendations for scenarios
    api_endpoint: /api/v1/optimize
    arguments:
      - name: scenario
        type: string  
        required: false
        description: Scenario to optimize (or 'all' for system-wide)
    flags:
      - name: --type
        description: Optimization type (resource, deployment, cost)
      - name: --apply
        description: Apply safe optimizations automatically
    output: Optimization recommendations and results

  - name: propose
    description: Analyze dependencies for proposed scenario
    api_endpoint: /api/v1/analyze/proposed
    arguments:
      - name: description
        type: string
        required: true
        description: Scenario description or requirements file
    flags:
      - name: --similar
        description: Number of similar scenarios to find
      - name: --resources-only
        description: Only predict resource dependencies
    output: Predicted dependencies and recommendations
```

## 🔄 Integration Requirements

### Upstream Dependencies  
**What capabilities must exist before this can function?**
- **PostgreSQL Resource**: Required for storing dependency metadata and analysis results
- **resource-claude-code**: Essential for AI-powered analysis of scenario code and configurations  
- **resource-qdrant**: Critical for semantic similarity matching of proposed scenarios
- **Scenario File System**: All existing scenarios must have discoverable `.vrooli/service.json` files

### Downstream Enablement
**What future capabilities does this unlock?**
- **Intelligent Deployment System**: Enables surgical deployments with minimal resource footprint
- **Strategic Capability Planning**: Identifies capability gaps and development priorities
- **Resource Cost Optimization**: Enables economic analysis of different resource configurations
- **Automated Architecture Health**: Continuous monitoring of system dependency health
- **Scenario Development Acceleration**: Faster development through dependency insight

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: deployment-optimizer
    capability: Dependency analysis for minimal deployment configurations
    interface: API + CLI
    
  - scenario: capability-planner  
    capability: Capability gap analysis and development roadmaps
    interface: API + Events
    
  - scenario: ecosystem-manager
    capability: Dependency predictions for generated scenarios
    interface: API
    
consumes_from:
  - scenario: ecosystem-manager
    capability: Notifications when scenarios are updated
    fallback: Periodic full system scans
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: NASA mission control meets network topology visualization
  
  visual_style:
    color_scheme: dark
    typography: technical
    layout: dashboard
    animations: subtle

  personality:
    tone: technical
    mood: focused  
    target_feeling: Intelligence and control over complex systems

# Style references:
style_references:
  technical:
    - system-monitor: "Matrix-style green terminal aesthetic"
    - agent-dashboard: "NASA mission control vibes"
  
# Custom elements for this scenario:
custom_elements:
  - Interactive dependency graphs with zoom/pan capabilities
  - Real-time dependency health indicators
  - Color-coded optimization opportunities
  - Hierarchical tree views for resource relationships
```

### Target Audience Alignment
- **Primary Users**: DevOps engineers, system architects, AI researchers
- **User Expectations**: Sophisticated technical interface with powerful analytical capabilities
- **Accessibility**: WCAG AA compliance, high contrast mode support
- **Responsive Design**: Desktop-first with tablet support for graph viewing

## 💰 Value Proposition

### Business Value
- **Primary Value**: Reduces deployment costs and complexity through intelligent dependency optimization
- **Revenue Potential**: $15K - $75K per deployment (higher for complex enterprise deployments)
- **Cost Savings**: 40-70% reduction in resource usage through optimization recommendations
- **Market Differentiator**: First AI system with true architectural self-awareness

### Technical Value
- **Reusability Score**: 10/10 - Every scenario benefits from dependency analysis
- **Complexity Reduction**: Transforms complex deployment decisions into automated recommendations  
- **Innovation Enablement**: Unlocks new categories of intelligent deployment and planning scenarios

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core dependency analysis and visualization
- Basic resource and scenario relationship mapping
- CLI and API interfaces for programmatic access
- Integration with claude-code and qdrant

### Version 2.0 (Planned)
- ML-powered dependency prediction for proposed scenarios
- Automated optimization application with rollback capability
- Real-time dependency health monitoring and alerting
- Advanced cost modeling and ROI analysis

### Long-term Vision  
- Evolve into Vrooli's central nervous system for architectural intelligence
- Enable fully autonomous deployment optimization across cloud providers
- Become the foundation for self-improving AI system architecture

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: scenario-dependency-analyzer

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/scenario-dependency-analyzer
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - ui/index.html
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli  
    - initialization
    - initialization/storage
    - ui

resources:
  required: [postgres, claude-code, qdrant]
  optional: [redis]
  health_timeout: 120

tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200

  - name: "Dependency analysis completes successfully"
    type: http
    service: api
    endpoint: /api/v1/analyze/chart-generator
    method: GET
    expect:
      status: 200
      body:
        resources: []
        scenarios: []

  - name: "CLI analyze command works"
    type: exec
    command: ./cli/scenario-dependency-analyzer analyze chart-generator --output json
    expect:
      exit_code: 0
      output_contains: ["resources", "dependencies"]

  - name: "Dependency graph visualization loads"
    type: http
    service: api  
    endpoint: /api/v1/graph/combined
    method: GET
    expect:
      status: 200
      body:
        nodes: []
        edges: []

  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('scenario_dependencies', 'dependency_graphs')"
    expect:
      rows:
        - count: 2
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Large graph visualization performance | High | Medium | Implement progressive loading and zoom-based detail levels |
| Circular dependency detection complexity | Medium | High | Use established graph algorithms with cycle detection |
| AI analysis accuracy for new scenarios | Medium | Medium | Combine claude-code analysis with qdrant similarity matching |

### Operational Risks
- **Analysis Staleness**: Implement event-driven re-analysis when scenarios change
- **Resource Conflicts**: Careful resource allocation and conflict detection in service.json  
- **Performance at Scale**: Optimize database queries and implement result caching

## 📝 Implementation Notes

### Design Decisions
**Graph Storage Format**: Store graphs as JSONB in PostgreSQL rather than specialized graph database
- Alternative considered: Neo4j or ArangoDB for native graph operations
- Decision driver: Consistency with existing Vrooli PostgreSQL usage and simpler deployment
- Trade-offs: Slightly more complex queries but better resource utilization

**AI Integration Strategy**: Use resource-claude-code for analysis rather than direct LLM integration  
- Alternative considered: Direct OpenAI/Claude API calls
- Decision driver: Consistency with Vrooli resource abstraction pattern
- Trade-offs: Additional abstraction layer but better long-term flexibility

### Security Considerations
- **Data Protection**: Dependency metadata contains no sensitive information
- **Access Control**: Read-only access to scenario configurations for analysis
- **Audit Trail**: All dependency changes and optimizations are logged with timestamps

## 🔗 References

### Documentation
- README.md - User-facing overview and quick start
- docs/api.md - Complete API specification  
- docs/cli.md - CLI command reference
- docs/graph-formats.md - Graph export format specifications

### Related PRDs
- deployment-optimizer (planned) - Will consume dependency analysis
- capability-planner (planned) - Strategic planning using dependency insights

### External Resources  
- D3.js documentation for graph visualization
- GraphViz specification for graph export formats
- PostgreSQL JSONB documentation for graph storage

---

**Last Updated**: 2025-09-05  
**Status**: Draft  
**Owner**: Claude Code AI Agent  
**Review Cycle**: Every iteration during development, then quarterly