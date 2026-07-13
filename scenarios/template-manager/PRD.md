# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Template Manager owns Vrooli's template domain: scenario generation, orientation, detemplating, template validation, drift, design kits, resource templates, factory documentation, and inherited-template debt.
- **Primary users/verticals**: Vrooli operators, scenario-building agents, test-genie providers, template maintainers, and future scenarios that need a stable programmatic template capability.
- **Deployment surfaces**: Proto+Connect API, thin manifest-driven CLI, React operations UI, test-genie phase provider, recurring monitor, measures federation, and search-hub indexed docs/debt records.
- **Value promise**: Standing up scenarios becomes measured, repeatable, and improvable because template quality has one accountable owner, persistent evidence, and an execution surface other scenarios can call.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Template registry and durable store | Catalog scenario templates, design kits, and resource templates with version and manifest metadata in a migration-owned SQLite store.
- [x] OT-P0-002 | Validation runs and debt ledger | Record shallow/deep validation, drift snapshots, version lag, and deduplicated defect debt entries through API and CLI.
- [ ] OT-P0-003 | Templates phase provider | Serve a static scenario-validation provider for test-genie's templates phase with the L0 generation-provenance autofix.
- [x] OT-P0-004 | Orientation guidance surface | Return the next incomplete orientation gate and its contract as structured API/CLI data for execution agents.

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Factory documentation service | Own, serve, and index factory docs for template maintenance, generation contracts, validation, drift, cleanup, and migration protocol.
- [ ] OT-P1-002 | Template standing dashboard and measures | Render fleet standing, validation history, debt trends, template registry state, and typed measures from one compute path.
- [x] OT-P1-003 | Recurring deep-validate monitor | Run scheduled deep validation for active scenario templates with serialized capacity-aware execution and persist scheduler-attributed results.
- [ ] OT-P1-004 | Scenario-template and design-kit cutover | Move scenario-template and design-kit lifecycle handling behind Template Manager API/CLI and delete the old vrooli CLI owners.

### 🟢 P2 – Future / expansion
- [x] OT-P2-001 | Resource-template cutover | Move resource-template handling into Template Manager and unify template substitution/text detection engines.
- [ ] OT-P2-002 | Orientation gate stewardship | Govern the orientation gate schema and strengthen template gate definitions so fresh scenarios expose honest work-order state.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API with Proto+Connect contracts, generated clients, cli-core manifest-driven CLI, React/Vite UI, and governed component primitives.
- Data + storage expectations: SQLite WAL under the api-core storage resolver, migration-first schema changes, and durable records for validation runs, debt entries, drift snapshots, registry rows, and monitor status.
- Integration strategy: Template Manager is the programmatic authority. The vrooli CLI surfaces are deleted during cutover rather than retained as wrappers, while test-genie calls the static validation provider and deep validation calls test-genie only from Template Manager-owned jobs.
- Non-goals / guardrails: Template content remains under `templates/`; the templates phase never triggers deep validation; a template remains quarantined until its current deep-validation evidence is clean; no offline CLI fallback or compatibility shim is added.

## 🤝 Dependencies & Launch Plan
- Required resources: embedded SQLite, Vrooli scenario lifecycle, test-genie, search-hub, measures-health, and vrooli-autoheal once critical monitoring is registered.
- Scenario dependencies: test-genie for provider conformance and deep validation runs; prompt-manager/search-hub for indexed factory docs and debt discovery; quality-health maturity/autofix patterns for deterministic fixes; vrooli-autoheal for critical availability.
- Operational risks: deep validation can be slow and must be serialized; hard cutover touches long-lived CLI/proto surfaces; inherited template defects must be logged as ledger debt rather than silently normalized; stale docs can cause agents to keep using removed command surfaces.
- Launch sequencing: scaffold and charter the scenario, add contracts/storage, ship additive validation/debt recording, register the templates phase, add docs/guidance/UI/measures/monitoring, then perform the scenario/design and resource hard cutovers.

## 🎨 UX & Branding
- Look & feel: Quiet operations dashboard optimized for scanning fleet standing, run history, debt status, and template registry data; no marketing shell or decorative hero.
- Accessibility: WCAG AA contrast, keyboard reachable tables/filters/actions, unique landmarks, responsive safe-area navigation, and axe-covered primary pages.
- Voice & messaging: Precise operator language that names template state, debt, remediation, and evidence without implying the user must read prose to understand next action.
- Branding hooks: Use Vrooli visual system defaults until a template-manager-specific mark is available; expose install/share metadata only when it conveys actual operator value.

## 📎 Appendix
- Implementation plan: operator-local plan `template-manager-scenario-owning-the-template-domain.md`.
- Vision context: `VISION.md`
- Template maintenance sources: `scenarios/template-manager/docs/factory/` and `templates/scenarios/react-vite/`.
