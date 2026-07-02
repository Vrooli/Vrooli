# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `docs/reference/canonical-prd-template.md` (adopted from the retired PRD control-tower; business-health is the enforcement owner)
> **Validation**: Enforced by `business-health` (self-hosting: this scenario validates its own contract)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Own the business contract of every scenario — PRD.md and requirements/ — as the modern test-genie health provider: validate template conformance and intent linkage, visualize evidence-backed traceability, scaffold conformant contracts deterministically, and make intent searchable fleet-wide.
- **Primary users/verticals**: Agents building or improving scenarios (the main consumers of findings and the wizard), operators reviewing fleet-wide business-contract debt, and local models answering "which scenario provides capability X" through search-hub.
- **Deployment surfaces**: CLI (`business-health` validate/report/wizard/fix/fleet/matrix), Connect-RPC API (shared `ScenarioValidationService` + native `BusinessContractReport` detail, wizard/report services), React UI (traceability matrix, fleet view, wizard, finding docs), search-hub federation (`business-health.intent` leaf).
- **Value promise**: One authoritative, provider-shaped home for PRD/requirements validation replaces the disagreeing legacy surfaces (the test-genie native phase, the retired PRD control-tower, the scenario-auditor shims); every scenario's stated intent becomes verifiable, traceable to evidence, and discoverable — so agents stop re-deriving what the fleet already promises.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Contract validation provider | Validate every scenario's PRD.md + requirements/ (template sections, section content, OT↔requirement linkage in both directions, registry structure, validation-ref existence) through the shared `ScenarioValidationService`, composing `packages/intent-go` extractors only — no second parser.
- [ ] OT-P0-002 | Delegated business phase | Serve as the delegated provider for test-genie's `business` phase (`FINDING_SOURCE_BUSINESS` preserved, advisory severity cap intact) and pass `test-genie provider-contract check` on every conformance dimension with real ExecutionMetrics.
- [ ] OT-P0-003 | Maturity ladder + honest autofix accounting | Ship `.vrooli/maturity.json` capability ladders (prd_contract, requirements_registry, intent_linkage, evidence_traceability) with every finding mapped to an honest `fix_class`, and deterministic fixers mounted on the shared Fix RPC (`PreviewFix`/`ApplyFix`, dry-run by default).
- [ ] OT-P0-004 | Deterministic scaffolding wizard | Scaffold a conformant PRD.md + requirements/ skeleton via interactive interview and `--answers` file mode — deterministic, resumable, no embedded AI; output validates clean by construction.
- [ ] OT-P0-005 | Requirements CLI absorption | Own the contract-side requirements verbs (validate, report, lint-prd, drift, phase, init, manual-log) end-to-end, with `vrooli scenario requirements` routing contract verbs here and sync/snapshot staying with test-genie (single-writer evidence).
- [ ] OT-P0-006 | Legacy surface retirement | All repo references, templates, skills, and docs point at business-health; the legacy control-tower scenario and the scenario-auditor `prd_*` shims are deleted with the fleet green.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Traceability matrix | Render OT × requirement × validation × latest-evidence for any scenario (UI + `business-health report`/`matrix`), reading test-genie's sync artifacts strictly read-only, with unproven-claim and staleness emphasis.
- [ ] OT-P1-002 | Fleet business-contract view | Rank scenarios by business-contract debt (starter registries, template laggards, unproven claims) with as-of stamps, in bounded time, via CLI and UI.
- [ ] OT-P1-003 | Intent search leaf | Federate `business-health.intent` (PRD purpose/value-prop/OTs + requirements, type-faceted single corpus) to search-hub as an ACTIVE provider with a green gold-standard eval, reranking enabled, and the `product-manager-agent.requirements` stub re-homed.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Template-version awareness | Ratchet checks that flag scenarios whose business contract predates the template contract version they were generated from.
- [ ] OT-P2-002 | Cross-scenario capability dedup | Surface "a similar capability already exists in scenario X" during scaffolding (wizard pre-scaffold hook) and in fleet views, powered by the intent search leaf.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (Connect-RPC over proto contracts in `packages/proto/schemas/business-health/`), cli-core CLI, React + Vite UI from the react-vite template (vrooli-default design kit).
- Data + storage expectations: sqlite for service state; all contract inputs read from scenario trees on disk; evidence artifacts (`coverage/requirements-sync/latest.json`, `coverage/manual-validations/log.jsonl`) read-only — test-genie remains the single writer of run evidence.
- Integration strategy: compose `packages/intent-go` for ALL PRD/requirements parsing (the ratchet); `packages/maturity-go` for assessment/autofix; `packages/ai-go/search` for the intent corpus; register with test-genie via the standard delegated-provider contract.
- Non-goals / guardrails: no embedded AI generation (judgment lives in calling agents; the wizard is a deterministic interviewer); never write evidence artifacts; no drafts/backlog/catalog products from the legacy control-tower; no parallel legacy paths kept alive.

## 🤝 Dependencies & Launch Plan
- Required resources: none hard-required; qdrant + ollama (embedding role) + reranker optional for the intent search leaf, degrading to no-search.
- Scenario dependencies: search-hub (optional, self-registration at boot); test-genie consumes this scenario as its `business` phase provider; reads test-genie sync artifacts from target scenario trees.
- Operational risks: parity drift during the window where checks exist both natively in test-genie and here (kept short by design); search-leaf eval quality bottlenecked by known local embedding-host capacity issues (gate is eval correctness, not federated latency).
- Launch sequencing: contract + docs first → API/CLI skeletons → checks (parity-verified against the native phase) → evidence/matrix → wizard → fleet/autofix → delegated-provider cutover → search leaf → UX → repo-wide cutover → legacy deletion.

## 🎨 UX & Branding
- Look & feel: vrooli-default design kit; light/dark themes; dense, data-forward tables for matrix and fleet views with clear severity color semantics.
- Accessibility: WCAG 2.1 AA; full keyboard navigation across matrix grids and wizard steps; unique landmark labels; axe-clean suites.
- Voice & messaging: precise and evidence-first — findings state what is broken, which artifact claims otherwise, and the deterministic fix if one exists; no vague health adjectives.
- Branding hooks: standard scenario branding surface (PWA manifest + icons) via brand-manager; finding docs rendered in-app share the docs typography.

## 📎 Appendix
- Plan: `docs/plans/business-health-provider-plan.md` (execution decisions D1–D8, phase gates, §14 definition of done).
- Doctrine: `docs/reference/intent-alignment.md` (the ladder, adjacent-rung rule, intent-go ratchet), `docs/reference/health-maturity-assessments.md` (provider contract).
- Prior art: structure-health (delegated conversion), proto-health (metrics recipe), cli-health (search leaf).
