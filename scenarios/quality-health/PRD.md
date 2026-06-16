# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Quality Health is Vrooli's static-quality authority. It discovers scenario surfaces through Code Facts, evaluates language/framework quality contracts, runs bounded lint/type command checks, reports agent-readable findings, and offers safe config autofix previews so Test Genie can delegate one coherent `quality` phase instead of embedding lint/type policy internally.

Primary users/verticals: Vrooli agents, Test Genie, Ecosystem Manager maturity flows, scenario maintainers, and operators inspecting fleet quality. The primary vertical is Vrooli's engineering-quality loop.

Deployment surfaces: Go API, Go CLI, React/Vite UI, and Test Genie provider integration. The API and CLI are the canonical programmatic surfaces; the UI is an operator inspection console.

Value promise: Static-quality policy becomes discoverable, reusable, and stricter than the hidden Scenario Auditor -> Tidiness Manager type-safety chain it replaces. Agents get precise remediation guidance instead of scattered lint failures, and future scenarios can reuse the same language/framework contracts without coupling policy to templates.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Surface-Aware Quality Audit | `quality-health audit <scenario> --json` calls Code Facts, returns a degraded result when discovery is unavailable, and emits normalized findings for TypeScript React/Vite UI, Go API, Go CLI, scenario Makefile gates, and `.vrooli/testing.json` quality policy.
- [ ] OT-P0-002 | Rule-Parity Contract Registry | Quality Health preserves or strengthens the existing `TS_CONFIG_STRICT`, `ESLINT_SAFETY_RULES`, `TS_DANGEROUS_PATTERNS`, `ESLINT_TYPED_CONFIG`, `NODE_BUILD_TYPECHECK`, `TESTING_CONFIG_LINT_STRICT`, `GO_MOD_PRESENT_FOR_API_OR_CLI`, `GO_LINT_CONFIG_PRESENT`, `GO_LINT_REQUIRED_LINTERS`, and `MAKEFILE_QUALITY_GATES` semantics from Tidiness Manager / Scenario Auditor.
- [ ] OT-P0-003 | Protective Comment Enforcement | TypeScript and ESLint config audits treat safety-critical agent guidance comments as first-class contract evidence, so strict config values without the required guardrail text still fail.
- [ ] OT-P0-004 | Agent-Readable Findings And Maturity | Audit responses include stable finding IDs, evidence, expected/observed values, why-it-matters copy, remediation, next steps, and a deterministic quality maturity rung.
- [ ] OT-P0-005 | Safe Config Autofix Preview | `quality-health fix-config <scenario> --dry-run --json` previews deterministic config edits for supported rules without applying changes; `--apply` is required for mutation.
- [x] OT-P0-006 | Autofix Completeness And Honesty | Each rule declares a fix class, every audit output reports aggregate autofixability, and `autofix_available` is true only when a registered fixer can actually produce a safe preview.
- [x] OT-P0-007 | Suppression Governance | TypeScript and Go suppression patterns are visible contract findings unless they carry a non-empty written reason explaining the exception.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Test Genie Quality Phase Provider | Test Genie shells the `quality` phase to Quality Health, maps results to the standards dimension, and retires the native `lint` phase plus duplicated type-safety policy producers.
- [ ] OT-P1-002 | Operator Inspection UI | The UI shows scenario audit state, discovered surfaces, contract failures, command results, autofix previews, and remediation grouping with loading, empty, error, and degraded states.
- [ ] OT-P1-003 | CLI Discoverability And Explainability | CLI help exposes audit, contracts list, explain, and fix-config commands; `explain` returns the rationale and exact repair path for a stable finding ID.
- [ ] OT-P1-004 | Quality Health Self-Audit | Quality Health can audit itself and report meaningful degraded or failing findings without relying on template-specific assumptions.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Fleet Quality Dashboard | The UI can compare quality posture across multiple scenarios and group failures by remediation path.
- [ ] OT-P2-002 | Drift History And Trend Signals | Quality Health persists recent audit runs, summarizes drift, and highlights repeated weakening attempts or suppression growth.
- [ ] OT-P2-003 | Expanded Language Contracts | Python and additional Node frameworks receive contract packs once Code Facts exposes reliable surface metadata for them.

## 🧱 Tech Direction Snapshot
Preferred stacks / frameworks: generated React/Vite UI, Go API, Go CLI, proto/Connect contracts where possible, and SQLite only if run history or local cache is implemented. CLI business logic must remain thin over API behavior.

Data + storage expectations: v1 can be stateless for live audits. If run history lands, store audit metadata, surface snapshots, findings, command results, and autofix previews in local SQLite with deterministic retention.

Integration strategy: Code Facts is the source of surface, language, framework, package manager, and parse-unit discovery. Test Genie consumes Quality Health as a provider phase. Tidiness Manager keeps maintainability debt ownership; Scenario Auditor keeps standards that are not static-quality contracts.

Non-goals / guardrails: Do not own unit testing, coverage policy, generic maintainability debt, or template-specific policy. Do not hard-code `ui/`, `api/`, and `cli/` as the core model except as compatibility fallback evidence when Code Facts reports degraded discovery. Do not weaken existing validation while moving it. Do not auto-edit source suppressions in v1.

## 🤝 Dependencies & Launch Plan
Required resources: none for the live-audit v1 path. SQLite is optional for run history. No new external services are required.

Scenario dependencies: Code Facts for discovery, Test Genie for phase orchestration, Tidiness Manager and Scenario Auditor for cutover cleanup context, and Prompt Manager for the eventual steer skill. Maturity wiring uses `packages/maturity-go`.

Operational risks: Code Facts unavailability could create false confidence, mitigated by degraded audits that do not pass cleanly. Autofix could be destructive, mitigated by dry-run default and explicit `--apply`. Protective comments could be lost during migration, mitigated by fixture parity tests where strict values without comments fail. Duplicate producers could confuse agents, mitigated by hard-cutting old lint/type producers after Quality Health parity is proven.

Launch sequencing: (1) Generate scenario and lock docs/requirements from the Phase 0 handoff. (2) Implement API/CLI with Code Facts ingestion, contract registry, rule parity, command runners, findings, and autofix preview. (3) Build the inspection UI. (4) Cut Test Genie to `quality` and remove native lint/type producers. (5) Re-scope Tidiness Manager and Scenario Auditor. (6) Add steer skills and maturity wiring. (7) Run full validation and cleanup.

## 🎨 UX & Branding
Look & feel: Operational engineering console using the `vrooli-default` design kit. The first screen is the audit workbench, not a marketing landing page.

Accessibility: WCAG AA contrast, keyboard-operable filters/actions, ARIA labels for status controls, and stable `data-testid` selectors for all actionable and status-bearing elements.

Voice & messaging: Direct, diagnostic, agent-ready. Findings explain what failed, why it matters, and what to do next without hiding raw evidence.

Branding hooks: Use the generated PWA manifest and default icons until product branding is available. UI copy should use "Quality Health" as the product name and avoid template `notes` vocabulary.

## 📎 Appendix
- Phase 0 handoff: user plan store directory `quality-health-phase0`
- Rule-parity inventory: `quality-health-phase0/rule-parity-inventory.md` in the user plan store
- Contract decisions: `quality-health-phase0/contract-decisions.md` in the user plan store
