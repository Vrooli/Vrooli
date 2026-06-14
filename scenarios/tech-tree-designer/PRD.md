# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Make Vrooli's scenario ecosystem plan-able by rendering the actual cross-scenario interface graph and letting agents design future scenarios as proto contracts before implementation.
- **Primary users/verticals**: Vrooli engineering agents, scenario planners, ecosystem maintainers, and operators reviewing the capability roadmap.
- **Deployment surfaces**: Connect API, Go CLI, React UI, generated proto contracts, and future agent/tool consumers.
- **Value promise**: Reduce scenario drift and rework by planning around real interfaces first, then materializing proto contracts only when they validate.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Live interface graph | Render scenario nodes and proto-import dependency edges from `proto-health` behind a fakeable `GraphSource` seam.
- [ ] OT-P0-002 | Contract-first planning | Store planned scenarios as real `.proto` text, validate them against live schemas, and expose CRUD/validate/materialize flows through API and CLI.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Roadmap overlay | Attach sector, tier, and milestone metadata to live and planned graph nodes without making tiers a competing backbone.
- [ ] OT-P1-002 | Production planning UI | Provide a D3 graph surface, planned-proto editor, validation findings, and roadmap overview with loading, error, and empty states.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Rich graph source | Add a `scenario-dependency-analyzer` GraphSource when its `DescribeInterfaceGraph` RPC is available.
- [ ] OT-P2-002 | Strategic analysis | Add optional AI-assisted bottleneck, node idea, and timeline analysis after the deterministic graph/planning surface is stable.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: `react-vite` template, Go API/CLI, Connect-RPC, generated proto clients, SQLite, D3 for graph visualization.
- Data + storage expectations: SQLite for planned scenarios, planned proto files, roadmap metadata, optional graph cache, and milestone state.
- Integration strategy: Consume `proto-health` `DescribeScenariosProtos` first; keep `GraphSource` shaped for `scenario-dependency-analyzer` later.
- Non-goals / guardrails: No migration from the old Gin/Postgres implementation, no compatibility shims, no old heuristic scenario catalog, no AI analysis in this phase, no SDA client until SDA ships the intended graph RPC.

## 🤝 Dependencies & Launch Plan
- Required resources: SQLite only.
- Scenario dependencies: `proto-health` is required for the live graph; planned-only mode should degrade cleanly if unavailable.
- Operational risks: `materialize` writes into `packages/proto/schemas/<slug>/` and must refuse on validation failure; generated proto churn must stay inside the plan allowlist.
- Launch sequencing: regenerate scaffold, define graph/planning/roadmap protos, implement graph source/query/export, implement planned proto storage/validation/materialization, add roadmap overlay, then ship UI.

## 🎨 UX & Branding
- Look & feel: Dense engineering planning surface, not a marketing page; restrained controls, readable graph labels, stable panels, and theme-aware status colors.
- Accessibility: Keyboard reachable graph controls, text alternatives for graph summaries/exports, WCAG-conscious contrast, and explicit loading/error/empty states.
- Voice & messaging: Precise operator language; findings should tell agents what failed and what to change.
- Branding hooks: Use existing Vrooli design tokens and lucide icons; graph visuals distinguish live vs planned, transport world, stability, sector, and tier.

## Appendix
- Implementation plan: `tech-tree-designer-regeneration-scenario-centric-interface-graph-contract-first-planning.md` in the user-scoped Vrooli plans directory.
- `proto-health` source surface: `packages/proto/schemas/proto-health/v1/shared/surface.proto`
- Future SDA plan: `scenario-dependency-analyzer-actual-interface-graph-and-import-drift.md` in the user-scoped Vrooli plans directory.
