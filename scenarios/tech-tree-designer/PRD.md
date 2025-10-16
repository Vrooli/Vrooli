# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**  
`tech-tree-designer` is the authoritative system for authoring, maintaining, and operationalising Vrooli’s civilization-scale technology graph. It provides a Graphviz-powered editor, auto-detects which scenarios/resources fulfil each node, tracks maturity states, and runs dependency-aware analysis so planners and agents can decide what to build next. The scenario stores the living blueprint for how micro-capabilities evolve into digital twins and meta-simulations.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Supplies machine-readable dependency data so agents can validate prerequisites before launching work.  
- Surfaces maturity overlays and coverage gaps, enabling roadmap agents to focus on high-leverage nodes.  
- Exposes strategy APIs that downstream scenarios (ecosystem-manager, prd-control-tower, git-control-tower) can query to contextualise their actions.  
- Maintains historical progression so learning systems can correlate past investments with present capabilities.

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Autonomous Scenario Prioritiser** – continuously consumes designer recommendations to open Generate Scenario tasks.  
2. **Capability Coverage Auditor** – monitors sectors for stagnation, triggered when prerequisite nodes stall.  
3. **Digital Twin Forecaster** – predicts when composite twins reach viability based on node maturity.  
4. **Cross-Sector Fusion Generator** – proposes novel scenarios that combine underused capabilities.  
5. **Meta-Simulation Planner** – uses the graph to schedule large-scale simulation experiments.

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Import and export the master tech tree via Graphviz DOT (and JSON mirror) using Graph Studio as the editing surface.  
  - [ ] Semantic zoom + filtering UI (layer, domain, maturity, capability type) backed by the graph datastore.  
  - [ ] Automated mapping of existing scenarios/resources to graph nodes by scanning repo metadata (PRDs, service.json, directory structure, tags).  
  - [ ] Maturity tracking per node (`planned`, `building`, `live`, `scaled`) with overlays and history.  
  - [ ] Dependency/centrality engine that generates ranked build recommendations and exposes `/api/v1/tech-tree/recommendations`.  
  - [ ] Decision dashboard with reasoning, blockers, and downstream unlocks for every recommendation.  
  - [ ] REST + CLI parity for importing graphs, refreshing mappings, querying maturity, and retrieving recommendations.  
  - [ ] Test suite entirely powered by mocks/fixtures—**tests must never mutate the real tech tree files or hit actual git**.

- **Should Have (P1)**
  - [ ] Historical timeline with sparkline visualisations of maturity progression per node/domain.  
  - [ ] AI narrative generator (via resource-openrouter) that explains why a recommendation ranks highly.  
  - [ ] "What-if" sandbox that lets users mark hypothetical completions and preview unlocked capabilities.  
  - [ ] Exportable strategy briefs (Markdown/PDF) summarising priority paths for selected domains.  
  - [ ] Integration hook for ecosystem-manager to create Generate Scenario tasks directly from recommendations (with draft PRD snippets).  
  - [ ] Optional integration with scenario-auditor to surface PRD structure violations for mapped scenarios.

- **Nice to Have (P2)**
  - [ ] Cross-twin simulation harness projecting outcomes when multiple domains advance together.  
  - [ ] External signal ingestion (market intelligence, research feeds) enriching priority scores.  
  - [ ] Collaborative session mode with shared annotations and decision logs.  
  - [ ] Adaptive recommendation model that learns from user accept/reject feedback.

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Graph import (2k nodes / 5k edges) | < 3s | API benchmark |
| Recommendation generation | < 1s per request | Profiling dependency engine |
| Mapping refresh (incremental) | < 5s | Repo scan benchmark |
| UI overlay refresh | < 250ms | Front-end instrumentation |
| AI narrative latency | < 6s median | OpenRouter telemetry |

### Quality Gates
- [ ] 100% of graph operations in tests use fixture DOT/JSON files; CI fails if real repo paths are touched.  
- [ ] All P0 endpoints covered by unit + integration tests with injectable adapters.  
- [ ] End-to-end smoke test: import fixture → auto-map sample scenarios → request recommendations.  
- [ ] Performance targets validated on synthetic large graph fixture.  
- [ ] Documentation complete (architecture, API, CLI, operator playbook).  
- [ ] Scenario-auditor PRD rule passes.

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: graph-studio
    purpose: Authoring and rendering of tech tree graphs
    integration_pattern: Shared file store + embedded React components
    access_method: Graphviz DOT exports/imports

  - resource_name: postgres
    purpose: Persist nodes, relations, mappings, maturity history, recommendations
    integration_pattern: Dedicated schema + migrations
    access_method: Direct DB driver

  - resource_name: openrouter
    purpose: AI narratives, what-if analyses, strategy briefs
    integration_pattern: resource-openrouter client with throttling
    access_method: CLI or HTTP wrapper

optional:
  - resource_name: scenario-auditor
    purpose: Validate mapped scenarios against governance rules
    fallback: Local validation pipeline
    access_method: REST API
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: tech-tree-broadcast.json
      location: initialization/automation/n8n/
      purpose: Notify other scenarios when priorities change

  2_resource_cli:
    - command: resource-openrouter infer --prompt <…>
      purpose: Generate AI recommendations and summaries

  3_direct_api:
    - justification: Graph analysis requires bespoke algorithms
      endpoint: Internal Go services operating on Postgres + DOT payloads
```

### Data Models
```yaml
primary_entities:
  - name: TechNode
    storage: postgres
    schema: |
      {
        id: UUID,
        graph_id: UUID,
        external_id: string,
        name: string,
        layer: enum(microservice, composite_app, digital_twin, simulation),
        domain: string,
        maturity: enum(planned, building, live, scaled),
        maturity_score: float,
        metadata: jsonb,
        last_updated_at: timestamp
      }
    relationships: Links to TechEdge, ScenarioMapping, MaturityEvent

  - name: TechEdge
    storage: postgres
    schema: |
      {
        id: UUID,
        graph_id: UUID,
        source_id: UUID,
        target_id: UUID,
        relation_type: enum(dependency, influence, data_flow),
        weight: float
      }

  - name: ScenarioMapping
    storage: postgres
    schema: |
      {
        id: UUID,
        tech_node_id: UUID,
        entity_type: enum(scenario, resource),
        entity_name: string,
        coverage_type: enum(implements, depends_on, enhances),
        signal_source: enum(prd_tag, service_json, directory, manual),
        confidence: float,
        maturity_signal: enum(not_started, in_progress, live),
        last_detected_at: timestamp
      }

  - name: Recommendation
    storage: postgres
    schema: |
      {
        id: UUID,
        tech_node_id: UUID,
        priority_score: float,
        impact_vector: {
          unlocked_nodes: int,
          centrality: float,
          maturity_gap: float
        },
        blockers: UUID[],
        rationale: text,
        created_at: timestamp
      }

  - name: MaturityEvent
    storage: postgres
    schema: |
      {
        id: UUID,
        tech_node_id: UUID,
        previous_state: string,
        new_state: string,
        source: enum(auto, manual, api),
        occurred_at: timestamp,
        notes: text
      }
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/tech-tree/import
    purpose: Import or refresh graph definitions
    input_schema: { graph: string (DOT), source: enum(designer, external), overwrite?: boolean }
    output_schema: { graph_id: UUID, node_count: number, edge_count: number }

  - method: GET
    path: /api/v1/tech-tree/nodes
    purpose: Retrieve nodes with filters and overlays
    input_schema: {
      domain?: string,
      layer?: string,
      maturity?: string,
      search?: string,
      includeEdges?: boolean
    }
    output_schema: { nodes: TechNode[], edges?: TechEdge[] }

  - method: POST
    path: /api/v1/tech-tree/mappings/sync
    purpose: Re-scan repository metadata to update scenario/resource mappings
    input_schema: { scope?: string[], force?: boolean }
    output_schema: { updated_nodes: int, signals_processed: int }

  - method: GET
    path: /api/v1/tech-tree/maturity-history
    purpose: Fetch maturity timeline for selected nodes
    input_schema: { nodeIds: UUID[], from?: string, to?: string }
    output_schema: { events: MaturityEvent[] }

  - method: POST
    path: /api/v1/tech-tree/recommendations
    purpose: Generate ranked build recommendations
    input_schema: {
      focusDomains?: string[],
      targetLayer?: string,
      maxResults?: int
    }
    output_schema: {
      recommendations: Recommendation[],
      summary: {
        bottlenecks: TechNode[],
        newlyUnlocked: TechNode[],
        rationale: string
      }
    }

  - method: POST
    path: /api/v1/tech-tree/what-if
    purpose: Evaluate hypothetical completions
    input_schema: { setLive: string[], setPlanned?: string[] }
    output_schema: { impactScore: float, unlockedNodes: TechNode[], affectedSimulations: string[] }
```

### Event Interface
```yaml
published_events:
  - name: tech_tree.node.updated
    payload: { nodeId: UUID, maturity: string, confidence: float }
    subscribers: prd-control-tower, git-control-tower

  - name: tech_tree.recommendation.created
    payload: { recommendationId: UUID, nodeId: UUID, score: float }
    subscribers: ecosystem-manager

consumed_events:
  - name: scenario.lifecycle.changed
    action: Update mapped nodes’ maturity signals
  - name: resource.lifecycle.changed
    action: Refresh relevant scenario mappings
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: tech-tree-designer
install_script: cli/install.sh

required_commands:
  - name: import
    description: Import DOT/JSON tech tree definitions
    flags: [--file <path>, --source <designer|external>, --overwrite]

  - name: status
    description: Summarise coverage and maturity by domain/layer
    flags: [--domain, --layer, --json]

  - name: recommend
    description: Output ranked build recommendations
    flags: [--domain, --layer, --limit, --json, --explain]

custom_commands:
  - name: sync-mappings
    description: Re-scan repo metadata using adapters (fixtures during tests)
    api_endpoint: /api/v1/tech-tree/mappings/sync
    flags:
      - name: --scope
        description: Comma-separated scenario/resource names
      - name: --force
        description: Ignore cache
    output: Human-readable + JSON when --json provided

  - name: what-if
    description: Run hypothetical completion analysis
    api_endpoint: /api/v1/tech-tree/what-if
    flags:
      - name: --set-live
        description: Node IDs to mark as live
      - name: --set-planned
        description: Node IDs to mark as planned
      - name: --json
        description: JSON output
```

### CLI-API Parity Requirements
- Every API endpoint has a CLI command or flag combination invoking it.  
- JSON output from CLI mirrors API schemas exactly.  
- Exit codes: 0 success, 1 validation error, 2 backend failure, 3 adapter misconfiguration.

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Go service with shared core modules for API, CLI, and background jobs
  - dependencies: Reuse graph parsing logic between API and CLI; inject interfaces for repo scanning and AI
  - error_handling: Structured errors with actionable remediation messages
  - configuration:
      - tech tree storage path configurable via ~/.vrooli/tech-tree-designer/config.yaml
      - repository root override via env or flag
      - AI provider configuration inherited from resource-openrouter

installation:
  - Symlink CLI into ~/.vrooli/bin/
  - Register API + web front-end with service.json lifecycle hooks
  - Ensure graph-studio export directory mounted/available
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **graph-studio**: Supplies authoring environment and DOT exports; Designer watches export directory for changes.  
- **scenario-registry (or metadata service)**: Provides canonical scenario/resource listings and lifecycle state.  
- **repo filesystem**: Needed for mapping; adapters must support dry-run with fixtures for tests.  
- **openrouter**: Enables AI narratives and summarisation.

### Downstream Enablement
- **ecosystem-manager**: Consumes recommendations to launch Generate Scenario tasks.  
- **prd-control-tower**: Uses node mappings to highlight documentation debt.  
- **git-control-tower**: Annotates commits with affected tech nodes.  
- **Simulation suites**: Read aggregated maturity to decide when digital twins are simulation-ready.

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: ecosystem-manager
    capability: Ranked backlog with dependency context
    interface: API/Event

  - scenario: prd-control-tower
    capability: Map PRD status to tech nodes
    interface: API

  - scenario: git-control-tower
    capability: Tag commit summaries with strategic impact
    interface: Event stream

consumes_from:
  - scenario: tech-tree-authoring (graph-studio)
    capability: Visual editor + DOT export
    fallback: Manual file upload via UI

  - scenario: scenario-auditor
    capability: Compliance status for mapped entities
    fallback: Inline validation
```

## 🎨 Style and Branding Requirements
```yaml
style_profile:
  category: technical visionary
  inspiration: Graph Studio canvas + NASA mission control overlays

  visual_style:
    color_scheme: deep dark with neon highlights
    typography: Inter + JetBrains Mono
    layout: canvas-centric with dockable insight panels
    animations: subtle zoom/pan easing, node pulse on maturity change

  personality:
    tone: strategic, data-driven
    mood: focused mission control
    target_feeling: clarity about civilization-scale progress
```

### Target Audience Alignment
- **Primary Users**: Architects, roadmap agents, leadership reviewing strategic progress.  
- **User Expectations**: High-density insights, fast filtering, trustworthy analytics.  
- **Accessibility**: WCAG AA compliance, keyboard-first navigation for graph interactions.  
- **Responsive Design**: Desktop-first, condensed layout for tablets during reviews.

## 💰 Value Proposition

### Business Value
- **Primary Value**: Converts ad-hoc scenario ideas into a continuously optimised civilization build plan.  
- **Revenue Potential**: $50K – $200K per deployment (strategic planning suite).  
- **Cost Savings**: Prevents low-impact work, aligns investments with maximal downstream unlocks.  
- **Market Differentiator**: No other platform offers an evolving tech tree that feeds digital twin strategy.

### Technical Value
- **Reusability Score**: 10/10 – every scenario and resource decision references this graph.  
- **Complexity Reduction**: Encodes dependencies so teams stop guessing prerequisites.  
- **Innovation Enablement**: Highlights cross-domain opportunities that manual planning misses.

## 🧬 Evolution Path

### Version 1.0 (Current)
- Full DOT import/export, semantic zoom UI, auto-mapping engine, recommendation API, maturity overlays, CLI parity, mocked tests.

### Version 2.0 (Planned)
- Historical analytics, AI narratives, what-if sandbox, ecosystem-manager integration, strategy brief exporter.

### Long-term Vision
- Adaptive recommendation models, external signal ingestion, collaborative planning, and direct hooks into simulation orchestration.

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json enumerates dependencies (graph-studio, postgres, openrouter)
    - initialization includes graph storage directory setup and DB migrations
    - health checks verify latest graph sync + mapping freshness

  deployment_targets:
    - local: Docker Compose (API + web + Postgres + Graph Studio mount)
    - kubernetes: Helm chart with PVC for graph files
    - cloud: Future managed deployment leveraging hosted Postgres

  revenue_model:
    - type: subscription
    - pricing_tiers:
      - internal use: bundled with Vrooli platform
      - enterprise: strategic planning package (future)
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: tech-tree-designer
    category: planning
    capabilities: [tech_graph_authoring, maturity_tracking, build_recommendations]
    interfaces:
      - api: /api/v1/tech-tree/*
      - cli: tech-tree-designer
      - events: tech_tree.*

  metadata:
    description: Master authoring + planning environment for Vrooli’s civilization tech graph
    keywords: [tech tree, strategy, roadmap, digital twin, recommendation]
    dependencies: [graph-studio, postgres, openrouter]
    enhances: [ecosystem-manager, prd-control-tower, git-control-tower]
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
|------|-------------|--------|------------|
| DOT import drift vs Graph Studio schema | Medium | High | Version-tag DOT exports, strict validation, automated migration scripts |
| False-positive scenario mappings | Medium | Medium | Multi-signal matching + manual override workflow |
| Large graph rendering performance | Low | High | Progressive loading, WebGL option, canvas chunking |
| AI reasoning hallucinations | Medium | Medium | Provide raw scoring context, require human confirmation before applying |
| Fixture/test drift | Low | High | Enforce fixture-only tests, CI guard rejecting real path access |

### Operational Risks
- **Stale data**: Watchers + scheduled rescan keep mappings fresh; UI warns when data older than threshold.  
- **Decision overload**: Limit recommendation lists, provide score explanations, allow manual curation.  
- **Integration churn**: Publish schema + API contracts, maintain backward compatibility.  
- **Security**: Restrict AI prompts to non-sensitive meta-data; redact scenario secrets.

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: tech-tree-designer

structure:
  required_files:
    - PRD.md
    - api/main.go
    - api/graph/importer.go
    - api/graph/importer_mock_test.go
    - api/recommendations/engine.go
    - api/recommendations/engine_test.go
    - api/mappings/repo_scanner.go
    - api/mappings/repo_scanner_mock_test.go
    - web/src/App.tsx
    - cli/tech-tree-designer/main.go
    - scenario-test.yaml
    - tests/fixtures/graphs/sample_tree.dot
    - tests/fixtures/mappings/sample_repo.json

resources:
  required: [graph-studio, postgres, openrouter]
  optional: [scenario-auditor]
  health_timeout: 60

# NOTE: All tests must rely on fixtures/mocks; CI fails if real repo paths are accessed.
tests:
  - name: "Graph import consumes fixtures"
    type: unit
    command: go test ./api/graph -run TestImportFromFixture
    expect: { exit_code: 0 }

  - name: "Recommendation engine ranks nodes deterministically"
    type: unit
    command: go test ./api/recommendations -run TestRankingsMockGraph
    expect: { output_contains: ["Top recommendation:"] }

  - name: "Repository scanner resolves mappings via fake repo"
    type: unit
    command: go test ./api/mappings -run TestScannerMock
    expect: { exit_code: 0 }

  - name: "REST endpoint returns recommendations"
    type: http
    service: api
    endpoint: /api/v1/tech-tree/recommendations
    method: POST
    body: { maxResults: 3 }
    expect:
      status: 200
      body:
        recommendations: []
```

### Test Execution Gates
```bash
make test          # Unit + integration (fixture-backed)
make lint          # Static analysis + fixture enforcement checks
make e2e           # UI smoke tests using mocked API
```

### Performance Validation
- [ ] Import benchmark on 2k-node fixture meets SLA.  
- [ ] Recommendation endpoint under load (100 parallel requests) stays <1s.  
- [ ] UI semantic zoom handles 1k-node viewport smoothly (Playwright/Lighthouse run).

### Integration Validation
- [ ] CLI commands mirror API behaviour against mock backend.  
- [ ] Events published to test bus for maturity + recommendation updates.  
- [ ] Ecosystem-manager task creation tested against stub service.

### Capability Verification
- [ ] Fixture scenarios/resources map to expected tech nodes.  
- [ ] Recommendations respect dependency ordering (no blocked nodes).  
- [ ] Digital twin overlays aggregate underlying maturity accurately.

## 📝 Implementation Notes

### Design Decisions
**Graph representation**: Normalise DOT into Postgres (nodes + edges) for analytics while keeping DOT as source-of-truth for editing.  
- Alternative: operate directly on DOT files.  
- Decision driver: enables efficient queries, versioning, and AI overlays.  
- Trade-off: need sync layer between DOT and DB.

**Mapping engine**: Combine PRD tags, service.json metadata, and directory heuristics to infer coverage.  
- Alternative: manual mapping.  
- Decision driver: ensures graph stays current automatically.  
- Trade-off: requires heuristics + manual override UI.

**Recommendation scoring**: Weighted sum of dependency satisfaction, centrality, maturity gap, and strategic weighting (configurable).  
- Alternative: simple dependency count.  
- Decision driver: capture real systemic impact.  
- Trade-off: requires tuning + transparency tooling.

### Known Limitations
- **Cross-repo support**: MVP assumes single repo; future work must federate multiple repositories.  
- **Binary graph assets**: Large background images or assets not yet supported in editor.  
- **Manual override UI**: Initial release focuses on auto-mapping; richer override workflows deferred to P1.

### Security Considerations
- **Data exposure**: Internal-only scenario; still sanitise AI prompts by stripping secrets.  
- **Access control**: No auth MVP; future enterprise deployments may require SSO.  
- **Audit trail**: Log imports, mapping changes, recommendation generation with user context for accountability.

## 🔗 References

### Documentation
- README.md – scenario overview and operator instructions.  
- docs/api.md – endpoint details and sample payloads.  
- docs/architecture.md – system diagrams and data flow.  
- docs/ai.md – prompt templates and guardrails.

### Related PRDs
- scenarios/ecosystem-manager/PRD.md  
- scenarios/prd-control-tower/PRD.md  
- scenarios/git-control-tower/PRD.md  
- resources/openrouter/PRD.md  
- resources/postgres/PRD.md

### External Resources
- Graphviz DOT specification  
- NetworkX/graph analytics references  
- OpenRouter API documentation

---

**Last Updated**: 2024-06-XX  
**Status**: Draft  
**Owner**: Strategic Planning Agent  
**Review Cycle**: Revalidate after major graph schema or mapping rule changes
