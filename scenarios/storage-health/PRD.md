# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Own all storage judgment in Vrooli — schema layout, migration hygiene, persistence-seam adoption, and (most importantly) test-isolation safety — as a language-aware, static analyzer that any scenario can be validated against, and that test-genie delegates a `storage` phase to.
- **Primary users / verticals**: test-genie (delegates the `storage` phase and consumes the L2 isolation verdict as a fail-closed precondition for destructive playbooks); the ecosystem-manager closed-loop controller (consumes `FINDING_SOURCE_STORAGE` findings on the `storage` dimension); platform/storage engineers answering "which scenarios can safely run destructive E2E?" and "which still use Postgres?"; the `storage-steer` skill, which gains its first producing dimension.
- **Deployment surfaces**: CLI (`storage-health validate scenario <name> --json`, `prove-isolation <name>`, `fleet --json`), Connect-RPC API (validation / isolation / fleet-inventory / migration-advisor services), and a React UI (dashboard, per-scenario validation view, fleet inventory, migration advisor, isolation scorecard).
- **Value promise**: Converts test-isolation from an unenforced, Go-only convention into a statically-proven, fail-closed guarantee — a scenario can no longer silently run mutating E2E playbooks against its real database — and turns "is this scenario's storage healthy?" from scattered scenario-auditor rules into one authoritative validator with agent-readable remediation.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Storage validation producer | A delegated `storage` validation producer that responds to `ScenarioValidationService.ValidateScenario`, classifies the target's storage surface via code-facts, and emits `FINDING_SOURCE_STORAGE` findings normalized to the maturity ladder
- [ ] OT-P0-002 | Static isolation proof + fail-closed gate | Statically prove test-isolation by checking the four routed-DB seams are wired; map an unproven result to the L2 safety rung so test-genie refuses destructive playbooks with a loud, instructive skip rather than mutating real data
- [ ] OT-P0-003 | Tier-1 schema-structure analyzers | Static Go analyzers for embedded-schema layout: centralized schema, non-idempotent DDL, `ALTER` in embedded schema, unwired `EnsureSchemas`, non-per-domain schema, non-empty system schema, and cross-domain hard foreign keys
- [ ] OT-P0-004 | Tier-1 isolation + shadow-safety analyzers | Analyzers for routed-seam adoption (`ROUTED_SEAMS_UNWIRED`), variant-aware namespace adoption (`STORAGE_NAMESPACE_HARDCODED`), and the non-Go fail-safe (`STORAGE_ISOLATION_UNVERIFIED`) that drives the fail-closed gate
- [ ] OT-P0-005 | Tier-2 persistence-hygiene analyzers | The persistence-hygiene checks migrated from scenario-auditor — raw `sql.Open`, routed-driver imports, captured `*sql.DB` handles, unclosed rows, direct SQL in handlers — plus the new SQLite single-connection deadlock check, with exempt-path parity
- [ ] OT-P0-006 | prove-isolation CLI | A `prove-isolation <scenario>` CLI verb that runs the isolation analyzers standalone and exits nonzero (with remediation) when isolation cannot be statically proven

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Autofix registry | Preview/apply autofixes for the deterministically-fixable findings (seam-wiring scaffold, namespace-helper rewrites, schema relocation, `defer rows.Close()` insertion), idempotent and fixture-backed
- [ ] OT-P1-002 | Fleet storage inventory & dashboard | Enumerate the fleet, classify each scenario's engines (SQLite / Postgres / Qdrant / Redis / file), namespace-helper adoption, isolation readiness, and storage stage; one query answers "show all Postgres scenarios"
- [ ] OT-P1-003 | Migration intelligence + Postgres→SQLite advisor + backup readiness | Schema-drift and migration-debt intelligence (stage-aware, informational), an engine-fitness Postgres→SQLite advisor, and detection of data-persisting scenarios with no registered backup target

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Production UI | A production-grade dashboard, per-scenario validation view (findings + remediation + autofix preview/apply), fleet inventory, migration advisor, and isolation-readiness scorecard
- [ ] OT-P2-002 | Non-Go storage analyzers | TS/Python storage analyzers once consumers exist, replacing the `STORAGE_ISOLATION_UNVERIFIED` fail-safe for those surfaces with real checks

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API on Connect-RPC over the `api-core` seams; Go CLI on `cli-core`; React + Vite + TypeScript + Tailwind UI on `vrooli-default` — cloning the `structure-health` / `security-health` health-scenario family.
- Data + storage expectations: SQLite, dogfooded on the very conventions this scenario enforces — per-domain embedded idempotent schema, the routed-DB seams (`database.Open → *RoutedDB`, `EnsureSchemas`, `TestModeMiddleware`, `devrouting.Register`), and variant-aware storage namespaces. No centralized schema, no committed `migrations/` folder for storage-health itself (it is greenfield).
- Integration strategy: language/domain detection via the `code-facts` client; backup-readiness via `data-backup-manager`; test-genie consumes the producer via the delegated-phase RPC (`ValidateScenario`) and the L2 isolation verdict. Static analysis only — AST/source inspection, never a runtime probe.
- Non-goals / guardrails: no filesystem/persistence-portability contract (deferred to a future cross-platform-readiness scenario, linked only); no TS/Python storage analyzers in v1 (covered by the `STORAGE_ISOLATION_UNVERIFIED` fail-safe); no restart-based fallback or behavioral write-probe (the isolation gate is static and fail-closed); no raw package-manager use (dependency work via SDA); no mass-update scripts.

## 🤝 Dependencies & Launch Plan
- Required resources: none beyond the host toolchain; the scenario persists to its own SQLite store. Qdrant/Redis are not required.
- Scenario dependencies: `code-facts` (API-surface language + `FACT_FAMILY_FILE_DOMAIN` domain ownership), `data-backup-manager` (backup/restore-readiness detection), `test-genie` (downstream consumer via the `storage` phase and the fail-closed isolation gate), `ecosystem-manager` (consumes `FINDING_SOURCE_STORAGE` on the `storage` dimension).
- Operational risks: the `storage_stage` greenfield heuristic mislabeling a deployed scenario (mitigated: advisory-only, operator-overridable via the `maturity` field, never hard-fails migration choices); fail-closed blocking legitimate read-only playbooks (mitigated: the gate only blocks destructive playbooks and honors the read-only / `allow_empty_test_pool` declaration); non-Go fail-safe over-flagging (mitigated: `STORAGE_ISOLATION_UNVERIFIED` is advisory-visible, not an error rung); dogfooding risk — storage-health must pass its own L2/L3 (mitigated: built on the canonical seams, self-validated).
- Launch sequencing: proto `FINDING_SOURCE_STORAGE` + `storage` dimension + `maturity` field → scenario scaffold + documentation-first PRD/requirements/maturity ladder → validation producer skeleton (code-facts) → Tier-1 schema analyzers → Tier-1 isolation/shadow-safety analyzers → Tier-2 hygiene analyzers (parity vs auditor) → autofix registry → fleet inventory + migration advisor + backup readiness → UX → test-genie `storage` phase + fail-closed gate + delete the restart fallback → remove the five scenario-auditor rules → rewrite `storage-steer` → relocate `routed-test-db.md` → full validation + regression anchor.

## 🎨 UX & Branding
- Look & feel: an operational storage console — ladder-driven status semantics (L2 isolation failures = destructive red, hygiene warnings = amber, advisory = muted), dense but scannable, last-validation timestamps and isolation readiness front and center.
- Accessibility: WCAG AA contrast, stable `data-testid` selectors, full keyboard operability, mobile-responsive layouts.
- Voice & messaging: terse, factual, remediation-forward; every finding names the exact seam to wire or autofix to run, and the isolation gate's skip message explains why it skipped, why it matters (real-data risk), and how to fix it.
- Branding hooks: Lucide iconography, the `vrooli-default` design tokens; isolation status is the primary visual anchor.

## 📎 Appendix
- Source plan: `docs/plans/storage-health-scenario-and-test-genie-producer-plan.md`.
- Relocated contract: the durable test-isolation contract lives at `docs/concepts/test-isolation-contract.md` (migrated from `docs/agent-system/routed-test-db.md`).
- Maturity ladder + finding catalog: `.vrooli/maturity.json` (L0–L4, `dimension: storage`).
