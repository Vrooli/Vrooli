# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Make Vrooli's scenario ecosystem plan-able by rendering the actual cross-scenario interface graph and letting agents design future scenarios as proto contracts before implementation.
- **Primary users/verticals**: Vrooli engineering agents, scenario planners, ecosystem maintainers, and operators reviewing capability coverage.
- **Deployment surfaces**: Connect API, Go CLI, React UI, generated proto contracts, and future agent/tool consumers.
- **Value promise**: Reduce scenario drift and rework by planning around real interfaces first, then materializing proto contracts only when they validate.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Live interface graph | Render scenario nodes and proto/Go import dependency edges from SDA's `DescribeInterfaceGraph` behind a fakeable `GraphSource` seam.
- [ ] OT-P0-002 | Contract-first planning | Store planned scenarios as real `.proto` text, validate them against live schemas, and expose CRUD/validate/materialize flows through API and CLI.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Capability ontology | Model the top-down capability tree, fulfillment links, coverage analytics, focus ranking, and overlay projection beside the scenario graph.
- [ ] OT-P1-002 | Production planning UI | Provide an interactive graph surface, planned-proto editor, validation findings, and ontology coverage overview with loading, error, and empty states.

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Rich graph source | Consume `scenario-dependency-analyzer` `DescribeInterfaceGraph` as the live graph source.
- [ ] OT-P2-002 | Strategic analysis | Add optional AI-assisted bottleneck, node idea, and timeline analysis after the deterministic graph/planning surface is stable.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: `react-vite` template, Go API/CLI, Connect-RPC, generated proto clients, SQLite, D3 for graph visualization.
- Data + storage expectations: SQLite for planned scenarios, planned proto files, ontology metadata, fulfillment links, optional graph cache, and coverage exclusions.
- Integration strategy: Consume SDA `DescribeInterfaceGraph`; SDA owns fleet graph interpretation while TTD owns graph queries, planning overlays, and ontology joins.
- Non-goals / guardrails: No migration from the old Gin/Postgres implementation, no compatibility shims, no old heuristic scenario catalog, no AI analysis in the shipped scope, no duplicate proto/import scanning inside TTD.

## 🤝 Dependencies & Launch Plan
- Required resources: SQLite only.
- Scenario dependencies: `scenario-dependency-analyzer` is required for the live graph; planned-only mode should degrade cleanly if unavailable.
- Operational risks: `materialize` writes into `packages/proto/schemas/<slug>/` and must refuse on validation failure; generated proto churn must stay inside the plan allowlist.
- Launch sequencing: regenerate scaffold, define graph/planning/ontology protos, implement graph source/query/export, implement planned proto storage/validation/materialization, add ontology coverage, then ship UI.

## 🎨 UX & Branding
- Look & feel: Dense engineering planning surface, not a marketing page; restrained controls, readable graph labels, stable panels, and theme-aware status colors.
- Accessibility: Keyboard reachable graph controls, text alternatives for graph summaries/exports, WCAG-conscious contrast, and explicit loading/error/empty states.
- Voice & messaging: Precise operator language; findings should tell agents what failed and what to change.
- Branding hooks: Use existing Vrooli design tokens and lucide icons; graph visuals distinguish live vs planned, transport world, stability, sector, and tier.

## Appendix
- Implementation plan: `tech-tree-designer-regeneration-scenario-centric-interface-graph-contract-first-planning.md` in the user-scoped Vrooli plans directory.
- SDA source surface: `packages/proto/schemas/scenario-dependency-analyzer/v1/graph/graph.proto`
