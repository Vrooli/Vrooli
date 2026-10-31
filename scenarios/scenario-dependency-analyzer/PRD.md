# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: `scenario-dependency-analyzer` is Vrooli's authority for scenario and resource dependency intelligence. It analyzes declared scenario configuration, computes actual cross-scenario interface usage from proto/import facts, exposes evidence-tagged graphs, and reports declared-vs-actual dependency drift.

Target users: Vrooli agents, test-genie, tech-tree-designer, deployment planners, and engineers inspecting scenario architecture.

Deployment surfaces: Tier 1 local stack for v1. Go API and CLI provide programmatic access; the React UI provides direct graph inspection for operators.

Value proposition: Vrooli needs a reliable way to understand its own live dependency graph. SDA turns scattered declarations, proto surfaces, and import facts into one reusable engineering capability: graph inspection, deployment planning, and drift-proof dependency declarations.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Declared dependency extraction | Automatically parse existing scenarios and extract resource and scenario dependencies from `service.json` and supported metadata.
- [x] OT-P0-002 | Legacy inter-scenario signal coverage | Detect retained non-import scenario usage signals such as supported Vrooli command invocations and shared workflows until AST facts replace them.
- [x] OT-P0-003 | Dependency metadata storage | Store history-bearing dependency metadata and analysis runs in the standardized dependency schema.
- [x] OT-P0-004 | Interactive dependency graph UI | Provide a visualization of dependency graphs with loading, empty, error, and drift states.
- [x] OT-P0-005 | Semantic scenario matching | Integrate with Qdrant and embedding resources for proposed-scenario similarity matching, with deterministic fallback heuristics.
- [x] OT-P0-006 | CLI graph access | Provide CLI commands for graph, drift, optimization, and export workflows.
- [x] OT-P0-007 | API graph access | Provide API endpoints other scenarios can query for dependency and graph information.
- [x] OT-P0-008 | Actual interface graph | Compute actual cross-scenario interface edges from `proto-health` proto surfaces and `code-facts` import facts.
- [x] OT-P0-009 | Dependency drift reporting | Report declared-vs-actual scenario dependency drift with asymmetric severity.
- [x] OT-P0-010 | Connect graph seam | Expose `DescribeInterfaceGraph` as a Connect RPC for downstream planning scenarios.
- [x] OT-P0-011 | SQLite storage cutover | Run on SQLite with domain-owned schemas instead of Postgres.
- [x] OT-P0-012 | Test Genie dependency producer | Provide the single read-only `scenario-dependency-analyzer health <scenario> --json` producer consumed by Test Genie's dependencies phase.
- [x] OT-P0-013 | Dependency surface readiness | Discover dependency surfaces through Code Facts evidence and validate runtimes, commands, modules, package managers, lockfiles, and local install state without mutating files.
- [x] OT-P0-014 | Runtime dependency health | Report required resources and scenario dependencies from `.vrooli/service.json` with degraded runtime status evidence.
- [x] OT-P0-015 | Approved dependency governance | Maintain reviewable approved-dependency governance memory for package choices, ranges, constraints, and non-allowlist guidance.
- [x] OT-P0-016 | Release-age policy validation | Validate pnpm `minimumReleaseAge` policy and governed release-age exceptions for dependency supply-chain safety.
- [x] OT-P0-017 | Security evidence boundary | Consume Security Health dependency-index and vulnerability evidence only at the correct boundary: index context in dependency health, vulnerability evidence in governance, and security gating in Security Health.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Optimization recommendations | Recommend resource swaps and dependency reductions for lightweight deployment profiles.
- [ ] OT-P1-002 | Dependency impact analysis | Explain what scenarios or resources would be affected if a dependency is removed.
- [ ] OT-P1-003 | Dependency history | Track dependency changes over time for trend and regression analysis.
- [x] OT-P1-004 | Graph export formats | Export dependency graphs as JSON, DOT/GraphViz, and image-ready data.
- [ ] OT-P1-005 | Cycle detection | Detect and report circular scenario or resource dependency chains.
- [ ] OT-P1-006 | Resource cost estimation | Estimate resource cost and deployment weight from dependency depth and resource classes.
- [x] OT-P1-007 | Governance operator workflow | Provide CLI and UI workflows for dependency governance triage, fleet validation, dry-run mutation, security-gap review, and remediation preview.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Predictive dependency planning | Predict likely dependencies for new scenario proposals from similar historical scenarios.
- [ ] OT-P2-002 | Refactoring suggestions | Suggest refactors that reduce dependency complexity or improve deployment modularity.
- [ ] OT-P2-003 | CI dependency policy | Integrate with CI and quality loops to validate dependency changes before merge.

## 🧱 Tech Direction Snapshot
Preferred stacks: Go API, Go CLI, React + Vite UI, SQLite through `packages/api-core/database`, and generated protobuf/Connect contracts for new reusable surfaces.

Preferred storage: SQLite stores history-bearing analysis records, recommendations, and rebuildable derived interface graph cache entries. Qdrant remains available for semantic similarity matching.

Integration strategy: `proto-health` owns proto surface facts. `code-facts` owns language-level import extraction. SDA owns cross-scenario interpretation: mapping import/proto paths to scenario slugs, unifying evidence into graph edges, and comparing actual usage with `service.json`.

Non-goals: No source-file scanning for import evidence inside SDA. No raw fact persistence or source-content cache. No full Gin-to-template migration in this plan. No planned-vs-live future-state modeling; tech-tree-designer owns that layer.

## 🤝 Dependencies & Launch Plan
Required resources: SQLite for local scenario storage, Qdrant for semantic matching, Ollama embeddings via the `embedding.default` role, and claude-code for analysis workflows.

Scenario dependencies: `proto-health` for batch protobuf surface facts and `code-facts` for batch import facts. Downstream consumers include `test-genie` and `tech-tree-designer` planning flows.

Operational risks: Declared-without-import-evidence can be a false positive while runtime URL and CLI shell-out AST facts are deferred. The mitigation is asymmetric severity: undeclared actual imports are warnings, while declared-only dependencies are informational until the AST-facts follow-up lands.

Launch sequencing: Documentation and decisions → proto-health batch facts → code-facts batch imports → SQLite cutover → actual graph domain → drift reporting and detector cleanup → Connect graph seam → test-genie and maturity integration → full validation and standards cleanup.

## 🎨 UX & Branding
User experience: Operational-console graph inspection for engineers and agents. The UI prioritizes dense graph scanning, clear edge evidence, drift filtering, and fast recovery from loading/error states over marketing presentation.

Visual design: Existing Vrooli operational styling with restrained graph controls, readable node labels, and status affordances for actual/declaration drift. Graph views should be useful on desktop first and remain navigable in constrained viewports.

Accessibility: Interactive controls require keyboard and focus-visible behavior. Graph data must have accessible alternate representation through lists or panels so users are not forced to rely on canvas/SVG spatial interpretation alone.
