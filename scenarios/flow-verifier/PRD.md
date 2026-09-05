# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Promote the in-template tool `tools/temporal-model/` to a first-class scenario that owns formal-flow discovery, Quint-based verification, codegen, and persistent run history with a visual inventory UI.
- **Primary users/verticals**: Vrooli scenario authors and internal engineering agents. Used during scenario development to prove state machines correct before merge.
- **Deployment surfaces**: CLI (`flow-verifier`), HTTP API (in-scenario), UI (`Flow Studio` inventory). No SaaS plumbing in v1.
- **Value promise**: Catch state-machine bugs before they reach production by running Quint typecheck/test/verify against every `flow/flow.json` in the monorepo, with a visual inventory of last-run status across all scenarios.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Discoverable across repo | `flow-verifier flows list --root <repo>` enumerates every `flow/flow.json` in the monorepo.
- [ ] OT-P0-002 | Sub-30s cold verify | `flow-verifier verify check` finishes in ≤ 30 s end-to-end (cold) for ≤ 10 flows.
- [ ] OT-P0-003 | CLI parity | Exposes `flows list|validate|new|explain`, `verify run|check`, `runs list|show` with human-text default output.
- [ ] OT-P0-004 | Persistent run history | SQLite `verification_runs` table records every verification with status, hashes, counterexample, duration. Survives restart.
- [ ] OT-P0-005 | Flow Inventory UI | One UI screen renders every discovered flow with its last-run status; supports a "Verify all" action.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | State-graph visualizer | React Flow render of `model.qnt` for a selected flow.
- [ ] OT-P1-002 | Trace player | UI to step through Quint counterexample traces.
- [ ] OT-P1-003 | Counterexample diff | Side-by-side view of expected vs. actual transitions.
- [ ] OT-P1-004 | Verification timeline | Chart of pass/fail rate per flow over time.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Monetization plumbing | Tenant, billing, public landing.
- [ ] OT-P2-002 | Multi-tenant isolation | Per-tenant run history and quotas.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go for API and CLI (Cobra). React + Vite + Tailwind for UI. Standard react-vite template defaults.
- Data + storage expectations: SQLite via `modernc.org/sqlite` (pure-Go, no CGO). Single table `verification_runs`. Flows are filesystem-truth; the DB stores only the verification trail. SQLite (not postgres) chosen because run history is single-tenant, append-mostly, low-cardinality.
- Integration strategy: Cross-scenario invocation only via the `flow-verifier` CLI binary — never raw HTTP, never Go imports. Quint and Java are declared **hostTools**, not Vrooli resources.
- Non-goals / guardrails: No backwards-compat shims, no `--legacy` flags, no v5 schema acceptance. No `helpers/`, `common/`, `utils/`, `shared/` god-folders under `api/internal/`.

## 🤝 Dependencies & Launch Plan
- Required resources: None (no Vrooli resources required).
- Scenario dependencies: None at runtime. Uses `test-genie` for QA gating; `business-health` for PRD/requirement governance.
- Operational risks: Quint binary version drift between contract record and host; generated-file byte-identity drift when codegen modules move; SQLite file-path resolution across CLI invocation contexts.
- Launch sequencing: Internal-first. Cutover deletes `templates/scenarios/react-vite/tools/temporal-model/` in the same change that flow-verifier reaches CLI parity. Template `make temporal-models` retargets to `flow-verifier verify check`.

## 🎨 UX & Branding
- Shell + routing reference: [`docs/concepts/UI_ARCHITECTURE.md`](docs/concepts/UI_ARCHITECTURE.md) — full-width operational console (resizable sidebar / mobile drawer), light/dark/system themes, settings persisted via `/api/v1/settings`.
- Look & feel: Vrooli-default design kit, react-vite-tailwind adapter. Dense, list-first inventory page.
- Accessibility: WCAG-AA baseline. a11y tests required for every UI feature (matches template).
- Voice & messaging: Engineering-tool tone. Terse status labels (passed/failed/error).
- Branding hooks: UI brand name **Flow Studio**; scenario id `flow-verifier`.

## 📎 Appendix
- Source for the migrated code: `templates/scenarios/react-vite/tools/temporal-model/` (15 internal Go packages, 2 JSON schemas).
- Migration plan: tracked in the orchestrating session plan (Claude Code plan mode).
- Architectural anchors: screaming-architecture audit (`docs/scenario-qa/methods/audit/screaming-architecture-audit.md`); `feedback_scenarios_always_have_ui`; `feedback_skills_use_cli_never_api`.
