# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Workflow Health is the permanent Vrooli capability for validating, executing, indexing, and improving scenario-owned browser automation workflows under `bas/`.
- **Primary users/verticals**: Scenario authors, Test Genie maintainers, workflow operators, and agents that need to discover safe reusable browser workflows.
- **Deployment surfaces**: Go API, workflow-health CLI, operational React UI, Test Genie delegated provider, Search Hub leaves, deterministic fix actions, and lifecycle-managed automations.
- **Value promise**: Workflows become governed assets: validation cases prove requirements, agent flows are discoverable by intent, unsafe mutation is fail-closed, and Test Genie can delegate its workflow phase instead of owning native browser logic.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Workflow asset catalog | Scan `bas/cases`, `bas/flows`, `bas/actions`, seeds, registries, requirement links, selectors, routes, safety metadata, and dependency edges into stable asset IDs.
- [ ] OT-P0-002 | Scenario validation provider | Implement `ScenarioValidationService` for the canonical workflow phase with findings, metrics, maturity, native details, fix preview/apply, and optional execution.
- [ ] OT-P0-003 | Safe BAS execution | Execute validation cases through Browser Automation Studio while refusing mutating workflows unless routed isolation and safety metadata are proven.
- [ ] OT-P0-004 | Test Genie workflow migration | Let Test Genie delegate the canonical `workflow` phase to workflow-health and keep `playbooks` only as a documented deprecated alias.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Workflow search and action discovery | Index `bas/flows` as agent-runnable workflow capabilities, `bas/cases` as validation evidence, and `bas/actions` as hidden dependency fragments with safety-aware ranking.
- [ ] OT-P1-002 | Deterministic remediation | Preview and apply mechanical fixes for registry freshness, metadata normalization, docs stubs, reset metadata, and unambiguous requirement-label repair.
- [ ] OT-P1-003 | Operational workflow UI | Provide a dense production UI for inventory, maturity, search, runs, artifacts, findings, fixes, and settings.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Workflow maturity intelligence | Track an L0-L5 workflow maturity ladder across discoverability, traceability, safety, executability, and operational richness.
- [x] OT-P2-002 | Fleet workflow rollout evidence | Support representative scenario checks, search-hub verification, routed mutating fixtures, and baseline diffs so workflow-health can roll out without weakening existing gates.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: generated react-vite scenario with Go API, Go CLI, React/Vite/TypeScript UI, Connect RPC, maturity-go response construction, and structured JSON/proto parsing.
- Data + storage expectations: local SQLite for workflow-health run state, catalog snapshots, fix previews, artifact metadata, and search indexing metadata; BAS remains the source of browser execution artifacts.
- Integration strategy: workflow-health is the policy and intelligence layer above Browser Automation Studio; Test Genie consumes it only through `ScenarioValidationService`; Search Hub consumes typed workflow leaves.
- Non-goals / guardrails: do not fold workflow policy into Browser Automation Studio, do not implement workflow search in Test Genie, do not auto-run mutating search results without safety metadata and confirmation, and do not preserve `playbooks` as the canonical phase name.

## 🤝 Dependencies & Launch Plan
- Required resources: lifecycle-managed SQLite plus the normal Vrooli scenario runtime; routed test database evidence is required before destructive execution is enabled.
- Scenario dependencies: browser-automation-studio for execution, test-genie for phase orchestration, search-hub for discovery, storage-manager for routed isolation evidence, and business-health for PRD/requirements validation.
- Operational risks: BAS workflow JSON may have mixed legacy/current shapes; mutating workflows can damage state if routed isolation is bypassed; search ranking can misclassify tests as actions unless leaf types stay separate.
- Launch sequencing: scaffold contract first, then build catalog scanning, validation/maturity/fixes, safe execution, provider CLI/API, search indexing, UI, Test Genie migration, native phase retirement, and final hardening.

## 🎨 UX & Branding
- Look & feel: quiet operational dashboard using the default design kit, compact tables, segmented filters, status chips, diff previews, artifact timelines, and restrained visual density.
- Accessibility: keyboard-navigable inventory/search/run/fix workflows, WCAG AA contrast, stable focus states, responsive layouts with no overlapping controls or hidden critical state.
- Voice & messaging: precise validation language that names findings, safety verdicts, routed-isolation state, and remediation effects without marketing copy.
- Branding hooks: use Workflow Health as the first-viewport signal; keep generated PWA assets valid until final product branding replaces them.

## 📎 Appendix
- Plan source: `/home/matthalloran8/.vrooli/plans/workflow-health-scenario-and-workflow-phase.md`.
- Key references: `docs/reference/health-maturity-assessments.md`, `docs/reference/ai-search-routing.md`, `scenarios/test-genie/docs/phases/README.md`, and `scenarios/test-genie/docs/phases/playbooks/README.md`.
