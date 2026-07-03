# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Own Vrooli scenario API readiness as a modern Test Genie health provider: validate API lifecycle wiring, live `/health` behavior, HTTP response semantics, and API-runtime hygiene from first principles, replacing the API-shaped residue currently trapped in scenario-auditor.
- **Primary users/verticals**: Coding agents fixing scenarios, operators reviewing fleet API readiness, Test Genie delegating provider phases, and future health providers that need a reference for API-surface validation boundaries.
- **Deployment surfaces**: Connect-RPC API (`ScenarioValidationService` plus provider-native detail), CLI (`api-health validate scenario`, `api-health probe health`, `api-health fix preview/apply`), React UI (capability summary, findings, probe evidence, fix preview), and Test Genie delegated phase integration.
- **Value promise**: API correctness moves out of a broad legacy auditor into a focused provider with a maturity ladder, execution metrics, honest fixability declarations, and low-ambiguity runtime probes, so agents can improve API readiness without weakening quality or security boundaries.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Provider contract foundation | Expose `scenario-validation/v1.ScenarioValidationService.ValidateScenario` with real `ExecutionMetrics`, provider identity, native API Health detail, and a `.vrooli/maturity.json` capability ladder for API readiness.
- [ ] OT-P0-002 | API lifecycle contract | Validate that an API scenario is lifecycle-compatible: service metadata declares API health paths, `api/main.go` uses `api-core/preflight` before business logic, and long-running HTTP servers use `api-core/server` for graceful shutdown.
- [ ] OT-P0-003 | Health endpoint contract | Validate the standard API health surface from static evidence and a safe live probe: `/health` exists, is lifecycle-reachable, returns bounded JSON with `status`, `service`, `timestamp`, and `readiness`, and reports degraded/unhealthy states honestly.
- [ ] OT-P0-004 | HTTP semantics contract | Validate low-ambiguity HTTP response semantics for first-party API code: status-code constants, no implicit success on error responses, correct content types for obvious response encoders/downloads/streams, and versioned feature endpoints while exempting operational probes.
- [ ] OT-P0-005 | Runtime hygiene contract | Validate API-runtime footguns that are not generic lint: outbound HTTP timeouts, response-body close discipline, request-context propagation for outbound calls, cancellable long-lived goroutines, and structured logging in API service paths.
- [ ] OT-P0-006 | Honest autofix coverage | Declare every finding's `fix_class` and `fixer_status`; implement deterministic preview/apply fixes only where the repair is local and unambiguous, such as service health-path normalization, missing health endpoint descriptors, simple raw status constant replacements, and simple missing JSON content-type headers.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Scenario-auditor API migration | Migrate the relevant scenario-auditor API expectations by preserving intent, not copying flawed rule logic, and publish a parity-or-better report explaining kept, redesigned, deferred, and rejected checks.
- [ ] OT-P1-002 | Endpoint inventory reconciliation | Reconcile registered handlers, endpoint descriptors, `.vrooli/endpoints.json`, CLI manifest mappings, and REST exception tags so API surface drift is visible without duplicating proto-health or cli-health ownership.
- [ ] OT-P1-003 | Operator workbench | Provide a UI and human CLI report that groups findings by capability, shows live probe evidence separately from static findings, and previews deterministic fixes before writes.
- [ ] OT-P1-004 | Test Genie phase cutover | Replace the remaining scenario-auditor `standards` API responsibility with an API Health delegated phase or subphase without changing unrelated standards ownership.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | API fleet readiness view | Rank scenarios by API-readiness debt, stale live-probe evidence, missing health-schema adoption, and fixable finding gaps with as-of stamps.
- [ ] OT-P2-002 | Contract-aware probe packs | Allow scenarios to declare additional representative safe GET probes for public API endpoints, with opt-in budgets and response-shape expectations.
- [ ] OT-P2-003 | Cross-provider signal sharing | Publish normalized API-surface facts that quality-health, security-health, performance-health, and architecture-cartographer can read without re-scanning the same files.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API and CLI generated from the react-vite template; Connect-RPC for provider APIs; React + Vite UI with the `vrooli-default` design kit; `packages/maturity-go` for assessments/autofix accounting; `packages/api-core/metrics` for execution metrics.
- Data + storage expectations: SQLite for local provider state only if probe history or fix sessions need persistence; validation inputs are target scenario trees, lifecycle metadata, endpoint descriptors, and optional live HTTP probe responses.
- Integration strategy: delegate through Test Genie's standard health-provider runner; use lifecycle discovery for live probes; reuse `api-core/health` as the health response schema reference; cross-reference quality-health, security-health, cli-health, proto-health, storage-health, and performance-health instead of duplicating their ownership.
- Non-goals / guardrails: no broad static-quality policy, no security-header/CORS ownership, no proto breaking-change validation, no load testing, no product-specific endpoint assertions, no auto-fixes that require API design judgment, and no direct binary execution outside Vrooli lifecycle.

## 🤝 Dependencies & Launch Plan
- Required resources: none.
- Scenario dependencies: test-genie consumes this provider; optional discovery reads from code-facts when available; quality-health/security-health/proto-health/cli-health remain neighboring authorities; scenario-auditor is a migration source only, not a long-term dependency.
- Operational risks: old scenario-auditor rules are heuristic-heavy and may encode false positives; the first release must document redesigned validation logic and keep ambiguous checks advisory or deferred until semantics are proven.
- Launch sequencing: scaffold + contract docs → provider maturity spec → target resolver and static lifecycle checks → live health probe → HTTP semantics checks → runtime hygiene checks → deterministic fix registry → UI/CLI workbench → Test Genie delegated phase cutover → scenario-auditor API rule retirement.

## 🎨 UX & Branding
- Look & feel: Vrooli-default, operational and evidence-dense; compact capability summaries, finding tables, probe timelines, and fix previews rather than marketing panels.
- Accessibility: WCAG 2.1 AA; keyboard-navigable tables and fix previews; clear status text in addition to color; screen-reader labels for capability and severity badges.
- Voice & messaging: direct, diagnostic, and boundary-aware: every finding says what platform contract failed, what evidence proved it, which provider owns adjacent concerns, and whether an auto-fix exists.
- Branding hooks: standard generated PWA and scenario branding surfaces; no custom brand beyond the API Health name until brand-manager supplies assets.

## 📎 Appendix
- Source migration inventory: `scenarios/scenario-auditor/api/rules/api/`.
- Provider contract doctrine: `docs/reference/health-maturity-assessments.md`.
- API health schema reference: `packages/api-core/health/health.go`.
- Adjacent provider boundaries: `scenarios/quality-health/`, `scenarios/security-health/`, `scenarios/cli-health/`, `scenarios/proto-health/`, `scenarios/performance-health/`, `scenarios/storage-health/`.
