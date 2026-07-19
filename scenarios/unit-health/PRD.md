# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Unit Health is Vrooli's test-execution and test-maturity authority. It discovers scenario test surfaces through Code Facts, plans canonical test commands per workspace, executes them under bounded safety limits, parses coverage, evaluates test architecture and test quality, diagnoses flake/runtime risk, and reports an agent-readable local maturity assessment so Test Genie can delegate one coherent `unit` phase instead of embedding language-specific test runners and a separate coverage parser internally.


Deployment surfaces: Go API, Go CLI, React/Vite UI, and Test Genie provider integration. The API and CLI are the canonical programmatic surfaces; the UI is an operator inspection console.

Value promise: Test execution, coverage, test architecture, and test maturity become discoverable, bounded, and reusable. Agents get one command — `unit-health validate scenario <name>` — that explains current test maturity, the next level, exactly what blocks it, what would run and why, what failed or hung, and which skills repair each gap. Test Genie stops owning four hard-coded language runners and a separate coverage phase; future scenarios reuse the same test contracts without coupling test policy to templates.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Surface-Aware Test Plan | `unit-health validate scenario <name> --json` calls Code Facts for surfaces and parse units, returns a degraded result when discovery is unavailable, and produces a per-workspace test plan for Go modules, React/Vite TypeScript UIs, and (degraded) Python packages without filesystem-only assumptions.
- [ ] OT-P0-002 | Bounded Diagnostic Execution | Planned tests run under per-workspace timeout, no-output watchdog, process-group cleanup, bounded concurrency, and memory-aware worker caps; every command returns a structured result classified as pass, test-failure, missing-dependency, misconfiguration, timeout/hang, or system-failure.
- [ ] OT-P0-003 | Coverage Ownership | Unit Health parses Go cover profiles, LCOV, and Vitest coverage summaries, computes per-surface/per-file coverage, and emits low/missing coverage findings — coverage is part of the unit phase, not a separate Test Genie phase.
- [ ] OT-P0-004 | Agent-Readable Maturity Assessment | The response includes stable finding codes, evidence, remediation, recommended skills, current and next Unit Health local maturity (L0–L5), the blocking findings for the next level, and global semantic impact grouping — and every emitted finding code maps in `.vrooli/maturity.json`. The ladder is honest about enforcement: **L0–L3 are enforced gates** (ERROR findings block local maturity) while **L4–L5 are advisory tiers** (measured, `global_impact: advisory`, never gating); the `TestMaturityLadderGateAdvisorySplit` anti-drift test holds the split.
- [ ] OT-P0-005 | Canonical Framework Contracts | Unit Health detects and requires canonical frameworks (Go `go test`, React/Vite `vitest`, Python `pytest` preferred, Bash `bats`) and reports Jest, missing test scripts, missing coverage config, package-manager mismatch, missing bats tests, and absent bats runner as degraded/error findings — Vitest is canonical for Vrooli React/Vite UIs and bats for shell. Node surfaces get a cross-platform lockfile-frozen dependency install (pnpm/yarn/npm) before vitest, classified as a dependency gap on failure.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Test Genie Unit Phase Provider | Test Genie shells the `unit` phase to `unit-health validate scenario <name> --json`, maps results through shared maturity metadata into the `tests` and `coverage` dimensions, and retires the internal unit runner plus the separate `coverage` phase.
- [ ] OT-P1-002 | Test Architecture And Quality Findings | Unit Health reports co-location, shared test utilities, no-helper-from-production, injectable-seam, assertion-strength, render-only, skipped/only, snapshot-overuse, and missing edge-case findings mapped to local maturity and global impact.
- [ ] OT-P1-003 | Operator Inspection UI | The UI shows scenario test maturity, next-level blockers, the test plan table, execution results, a coverage dashboard, architecture/quality findings, artifact/log links, and global impact grouping with loading, empty, degraded, running, and failed states.
- [ ] OT-P1-004 | CLI Discoverability And Human Default | CLI help exposes `validate scenario` with human output as the default agent/operator workflow and `--json` only for programmatic consumers; output never requires `--json` to be useful.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Flake And Runtime Diagnostics | Unit Health persists recent run timing/status to scenario-local SQLite (api-core/database, deterministic retention), flags runtime growth from a rolling baseline (current vs median of recent passing runs), detects flake from cross-run pass/fail variance, and surfaces likely-hang culprits from no-output timeout plus last log lines. Static "flaky" source markers survive only as a weak supplementary signal.
- [ ] OT-P2-002 | Requirement-Linked Tests | Unit Health reports untagged requirement validation where requirements expect test references, feeding the L5 rung.
- [ ] OT-P2-003 | Expanded Language Contracts | Python and additional Node frameworks receive first-class contracts once Code Facts exposes reliable parse units for them.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: generated React/Vite UI, Go API, Go CLI, proto/Connect contracts, and SQLite only if run-timing history or local cache is implemented. CLI business logic stays thin over API behavior.
- Data + storage expectations: v1 can be stateless for live validation. If run history lands (flake/runtime diagnostics), store run metadata, command results, and coverage snapshots in local SQLite with deterministic retention.
- Integration strategy: Code Facts is the source of surface/parse-unit discovery for Go and TypeScript. Test Genie consumes Unit Health as its `unit` phase provider. `packages/maturity-go` validates the `.vrooli/maturity.json` ladder and turns finding impacts into global signals. Shell syntax validation stays with Quality Health/static quality.
- Non-goals / guardrails: Do not own shell syntax validation, smoke/playbooks/E2E (BAS keeps those), or final global scenario maturity. Do not duplicate Code Facts language/surface discovery for Go/TypeScript. Do not make Jest the canonical React/Vite path. Do not keep legacy internal unit-runner wrappers or a separate `coverage` phase after cutover. Do not require `--json` for the human workflow.

## 🤝 Dependencies & Launch Plan
- Required resources: none for the live-validation v1 path. SQLite is optional for run-timing history.
- Scenario dependencies: Code Facts for discovery, Test Genie for phase orchestration, Quality Health for shell/static-quality ownership context, and Prompt Manager for the `test` / `unit-testing-architecture-steer` steer skills. Maturity wiring uses `packages/maturity-go`.
- Operational risks: Code Facts unavailability could create false confidence, mitigated by degraded plans that never pass cleanly. Test execution could hang or OOM, mitigated by a bounded runner (timeouts, no-output watchdog, worker caps, process-group cleanup) as a v1 requirement. React test framework fragmentation, mitigated by making Vitest canonical and reporting Jest as degraded. Duplicate producers could confuse agents, mitigated by hard-cutting the internal unit runner and `coverage` phase after parity is proven.
- Launch sequencing: (1) Generate scenario and lock docs/requirements/maturity from the Phase 0 handoff. (2) Implement proto/API/CLI foundations. (3) Code Facts intake + test plan builder. (4) Bounded execution engine. (5) Coverage/architecture/quality/diagnostics analyzers + maturity assessor. (6) Inspection UI. (7) Cut Test Genie to one delegated `unit` phase and remove the internal runner + `coverage` phase. (8) Update skills. (9) Full validation and cleanup.

## 🎨 UX & Branding
- Look & feel: Operational engineering console using the `vrooli-default` design kit. The first screen is the test-maturity workbench, not a marketing landing page.
- Accessibility: WCAG AA contrast, keyboard-operable filters/actions, ARIA labels for status controls, and stable `data-testid` selectors for all actionable and status-bearing elements.
- Voice & messaging: Direct, diagnostic, agent-ready. Findings explain what failed, why it matters, and what to do next without hiding raw command output.
- Branding hooks: `vrooli-default` design tokens; diagnostic iconography for pass/warn/error/hang states; no external logos.

## 📎 Appendix
- Source plan: `~/.vrooli/plans/unit-health-scenario-and-test-genie-unit-phase-cutover.md`
- Phase 0 handoff: `~/.vrooli/plans/unit-health-PHASE0-handoff.md`
- Sibling reference scenario: `scenarios/quality-health` (static-quality cutover).
- Local maturity ladder: `.vrooli/maturity.json` (L0–L5, provider=`unit-health`, phase=`unit`).
