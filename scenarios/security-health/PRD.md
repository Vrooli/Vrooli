# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Add a permanent, language/framework-aware capability that validates any scenario's security posture across the substrates it actually contains (secrets, Go SAST, Go vuln-DB, JS/TS dependency CVEs) and returns normalized findings — and a fleet-wide, continuously-reconciled, semantically-searchable index of every dependency annotated with known-vuln status.
- **Primary users / verticals**: The ecosystem-manager closed-loop controller (consumes security findings at maturity-ladder rung R1); test-genie (delegates a `security` phase to this scenario); platform/security engineers answering "which of our scenarios are exposed to CVE-X?".
- **Deployment surfaces**: CLI (`security-health validate scenario <name> --json`, `deps search`, `reindex`), Connect-RPC API (validation / search / reindex services), React UI (Posture / Dependencies / Secrets), and an embeddable per-scenario posture badge widget.
- **Value promise**: Closes the last open producer-less gate in the maturity ladder — the controller can no longer call a scenario "safe" while it leaks a key — and turns "who is exposed to this CVE?" from a manual lockfile grep into a single query.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Substrate-aware validation | Detect each scenario's stacks and run only the applicable scanners; unsupported substrates emit an INFO observation, never a failure
- [ ] OT-P0-002 | Normalized severity contract | Map every scanner's native severity to ERROR (critical/high) / WARNING (medium) / INFO (low/info) so downstream gating is consistent
- [ ] OT-P0-003 | Delegation-ready CLI | `validate scenario <name> --json` emits the exact findings shape test-genie parses and exits nonzero on any ERROR finding
- [ ] OT-P0-004 | test-genie security producer | A delegating `security` phase emits `FINDING_SOURCE_SECURITY`, feeding the ecosystem-manager `security` dimension that hard-gates ladder rung R1
- [ ] OT-P0-005 | Actionable remediation | Every finding carries a human-readable remediation string (rotate-and-purge for secrets, bump-to-patched for vulns)
- [ ] OT-P0-006 | Graceful scanner degradation | Absent scanner binaries register as INFO observations; the scenario stays fully functional on the host-present subset

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Fleet dependency intelligence | A continuously-reconciled SBOM corpus across every scenario's lockfiles, annotated with vuln status, in Qdrant
- [ ] OT-P1-002 | Semantic + structured dependency search | AI search (degrading to TEXT) plus `--ecosystem` / `--vulnerable-only` / `--name` filters answer exposure questions in one query
- [ ] OT-P1-003 | Security posture UI | Posture, Dependencies, and (redacted) Secrets pages with severity grouping, last-scan time, and per-finding remediation
- [ ] OT-P1-004 | Embeddable posture badge | A per-scenario severity-rollup widget honoring the ui-health `WidgetDeclaration` contract (slot `INLINE`)

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Python & JS/TS SAST | Pluggable `bandit` / `pip-audit` / `semgrep` runners once Python scenarios exist
- [ ] OT-P2-002 | Continuous CVE alerting | Notify when a newly-published CVE matches an already-indexed fleet dependency
- [ ] OT-P2-003 | Secret rotation workflows | Move beyond detect-and-advise toward guided rotation

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API on Connect-RPC, React + Vite + TypeScript + Tailwind UI, Go CLI on `cli-core` — cloning the `cli-health` / `ui-health` family verbatim.
- Data + storage expectations: SQLite (`${SCENARIO_DATA_DIR}/security-health.db`) for scan history and the structured dependency table; Qdrant (`security-health-deps`, dimensions resolved from `embedding.default`, cosine) + Ollama role-policy embeddings for the semantic dependency index, both optional with TEXT fallback.
- Integration strategy: shell out to scanner CLIs (gitleaks, gosec, govulncheck, pnpm audit, osv-scanner) behind a `Scanner` interface; test-genie consumes us via CLI `--json` (matching cli/ui-health), not HTTP.
- Non-goals / guardrails: no stack-specific scanner baked into test-genie; no broad auto-remediation or auto-PR; deterministic low-risk provider fixes are allowed when previewable and rule-scoped; no secret rotation in v1; no Python/semgrep scanners in v1; no committed `migrations/` folder; no `make breaking` on this branch.

## 🤝 Dependencies & Launch Plan
- Required resources: `qdrant` + `ollama` (both `required:false`, `startup_policy:"try_start"`; degrade to TEXT search when down).
- Host tools: `gitleaks` + `gosec` (present); `govulncheck` + `osv-scanner` (optional, install-gated — absent ⇒ INFO observation, not failure); `pnpm audit` (ships with pnpm).
- Scenario dependencies: `test-genie` (downstream consumer via the `security` phase); `ecosystem-manager` (consumes `FINDING_SOURCE_SECURITY` at R1).
- Operational risks: SAST false-positive flood spuriously gating R1 (mitigated: only critical/high ⇒ ERROR; over-firing rules get scoped, never globally disabled); network-bound scanner latency (mitigated: longer phase timeout + pre-warmed reconcile index); secret leakage in reports (mitigated: redaction-first, file:line only).
- Launch sequencing: proto `FINDING_SOURCE_SECURITY` → scenario scaffold → validation core → test-genie `security` phase + EM wiring → dependency intelligence → UI → live `balanced` vs `balanced-ladder` A/B proving R1 holds then advances.

## 🎨 UX & Branding
- Look & feel: an operational security console — severity-driven color semantics (critical/high = destructive red, medium = amber/warning, low/info = muted), dense but scannable, last-scan timestamps front and center.
- Accessibility: WCAG AA contrast, stable `data-testid` selectors, full keyboard operability, mobile-responsive layouts.
- Voice & messaging: terse, factual, non-alarmist; every finding is remediation-forward rather than blame-forward.
- Branding hooks: Lucide iconography, the `vrooli-default` design tokens; secrets are redacted by default everywhere they surface.

## 📎 Appendix
- Source plan: `docs/plans/security-health-scenario-and-test-genie-producer-plan.md`.
- Deferred-work origin: realizes "A3 — security producer" from `docs/plans/ecosystem-manager-maturity-ladder-and-anti-gaming-plan.md` §12.
