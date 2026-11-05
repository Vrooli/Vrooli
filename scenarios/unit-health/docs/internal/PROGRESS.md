# Progress — Unit Health

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/unit-health/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones; consult the archive for the full sequence.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/unit-health/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append concise milestones when work lands. Keep detailed execution receipts
with the owning plan.

## Progress Log

| 2026-09-09 | codex | Phase 12 closeout evidence | PRD/ledger/docs/manifest/skills synchronized; final direct validation passed at L3 with executed=passed and LOW_COVERAGE=0; calibration inventory is 42 implemented + 39 retired of 81 specified; focused Test Genie unit run `20260909-103153-95f21e1c` passed at L3; targeted docs rerun `20260909-095303-91db3096` passed with standing advisory findings; final board `prog_3648e801-8bb7-48ea-a457-876fb8c9f6ae` regenerated Today with unknown-share worst=0.0334 and skip-declaration=0.0. | Closeout evidence captured; fleet-adoption remains measured out of band, reviewed evidence remains operator-supplied, promotion remains requested |
| 2026-09-09 | codex | Phase 12 docs follow-up | Added the manifest-required `FLOWS.md` cross-reference and removed the self-referential manifest placeholder rule; latest docs run `20260909-105216-07cee767` passed with the local content contract clean. The remaining findings are the external Test Genie command-validator mismatch, partial CLI schema metadata, and advisory number lint. | Strict docs acceptance remains qualified; the actual `test-genie registry build` command exits 0 and regenerated the tracked registry |
| 2026-09-09 | codex | Phase 12 docs lint closeout | Marked authoritative documentation counts with `num[...]` source/threshold markers and reworded incidental counts; docs run `20260909-110303-c950fa1a` passed with content, links, manifest coverage, and number lint clean. Only the external Test Genie command-validator mismatch and informational CLI-schema partials remain. | Local docs quality debt resolved; external validator metadata remains qualified |
| 2026-09-09 | codex | Phase 12 final docs qualification | Docs run `20260909-110950-4eb02a27` passed with local content, links, manifest coverage, and number lint clean. The remaining command-validator and CLI-schema observations are filed against Test Genie and Knowledge Observatory; the historical prompt-manager observation was not reproduced. | Local docs quality is clean; strict acceptance permits the attributable external observations |
| 2026-09-09 | codex | Phase 11 done | Replaced the fleet’s external-only rows with bounded-roster composition through Test Genie validation-digest and Agent Manager friction-digest. Live board program `prog_541a6ce9-ff45-45ef-b9b4-db01db24517e` completed in 92005ms: applicable=1, covered=0, uncovered=[unit-health], child_failures=[], recurring_count=0; focused tests passed and Test Genie programs reached L2. | Complete |

| 2026-09-09 | codex | Phase 5 self-execution: added behavioral coverage for the adapter registry/workspace planner, Go and React/Vitest adapters, calibration/native reports, discovery, Connect module, executor hermetic paths, evidence/traceability/companion registries, and CLI handlers. API package coverage passed at 80.7% for executor and 88.4% for validation; CLI packages passed at 84.2% root, 89.4% calibrate, and 93.3% validate. The executed self-run `uh-20260909-074815-45f0eb8ca1dbc2a3ff6cabd384591532` reported zero `LOW_COVERAGE`, `evidence_stages.executed=passed`, and assertion-observation unknown=18 (unchanged). `.vrooli/testing.json` remained byte-identical (sha256 `f3807ba6ea15130d4b7515440f92d7a39131bb2804498a50ad1ed7ba950df39d`). | Complete |


















| 2026-08-21 | codex | v2 continuation: changed weighted CPU/memory admission to FIFO waiter reservation with cancellation-safe removal, preventing resource-heavy or late requests from barging ahead of queued work. | Archived |
| 2026-08-21 | codex | v2 continuation: wired the bounded evidence store into the production validation module through the platform storage resolver, and included each workspace's effective runner profile in cache keys. | Archived |
| 2026-08-21 | codex | v2 continuation: added explicit workspace-scope identities to evidence keys so identical source contents from different selected workspaces cannot cross-reuse cached responses. | Archived |
| 2026-08-21 | codex | v2 continuation: adapter-owned launcher resolution now preserves absolute package-manager/Python/Cargo/PowerShell paths (including spaces and platform extensions), emits missing-runtime diagnostics before execution, and Jest declares normalized Istanbul/LCOV coverage artifacts. | Archived |
| 2026-08-21 | codex | v2 continuation: moved coverage artifact parsing and detailed native projection checks behind adapter contracts; added the analyzer registry and framework-neutral validation boundary guard; removed Web Console's multi-process test launcher and package-script worker flags from the slow-suite migrations; added native API e2e CI execution and targeted cache speedup evidence. | Archived |

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-09-09 | claude | done | Scenario skill set brought to the contract: usage skill rewritten to a rung-labelled decision tree with the learning spine; improve skill rewritten to the control-loop sections with a generated setpoint board; `unit-health.setpoint-read` program authored with fixtures and tests; phantom `code-facts/tests/list` binding removed from `test-quality-sample`; API next-step messages now name the CLI flag `--execution`; maturity reference rewritten to the capability spec; problem-ledger entries recorded; starter-domain residue removed from the docs tree. |

## Phase 8 Handoff (next agent — start here)

**State:** Phases 1–7 are DONE. The unit-health engine is complete (API/CLI/UI),
and **Test Genie has been hard-cut over**: one delegated `unit` phase backed by
`unit-health validate scenario <name> --execution --json`, the native unit
runner + separate `coverage` phase are deleted, maturity-go dimension maps +
anti-drift fixture updated, and the cutover is live-proven (see the Phase 7
log row above). What remains is **Phase 8 (skills)** and **Phase 9 (full
validation, doc-prose cleanup, baselines, final record)**.

**Phase 8 — skill updates (per plan §7 Phase 8), all in `scenarios/prompt-manager/store/skills/`:**
1. `test` skill — make it the primary skill for `unit-health` / the Test Genie
   `unit` phase; tell agents to run `unit-health validate scenario {{TARGET}}`
   (human, no `--json`) for the workflow; keep behavior/coverage/assertion
   guidance; remove any implication that the skill is the canonical *scorer*
   (unit-health's local maturity is the scorer now).
2. `unit-testing-architecture-steer` — reframe as a finding-specific remediation
   skill for architecture/testability findings; add `targetDimensions` (tests)
   if missing; add `programmaticHome: unit-health:unit`; point current-maturity
   discovery at the unit-health CLI output.
3. `e2e-testing` — touch only if unit-health findings can recommend it for gaps
   that truly belong to BAS/playbooks.
4. Update skill validation metadata/histories; verify
   `prompt-manager skill validate test unit-testing-architecture-steer e2e-testing`
   and `prompt-manager discover "fix unit-health missing testutil coverage low scenario" --complexity moderate`
   return the right skills.

**Phase 9 — final validation & cleanup:** run unit-health against itself +
test-genie + code-facts + a UI-heavy scenario; run the new `unit` phase; run
`maturity-go` tests; sweep the remaining stale doc prose still naming `unit`/
`coverage` as the *old* active phases (RESEARCH.md, QUICKSTART.md, GLOSSARY.md
`test-genie coverage` CLI mention, concepts/strategy.md, etc. — Phase 7 only
fixed the catalog-defining docs); resolve the test-genie + code-facts baseline
diffs (`git-control-tower baseline diff status --scenario test-genie --name unit-health-hard-cutover`);
verify the §6a file allowlist; write the final shipped record.

**Test Genie cutover layout (as built — for reference):**
- `scenarios/test-genie/api/internal/orchestrator/phases/phase_unit.go` — the
  delegated runner (seam `runUnitValidate`; `parseUnitOutput`/`translateUnitReport`/
  `unitCoverageFindings`). Mirror of `phase_quality.go`.
- `phase_unit_test.go` — delegated-phase tests via the `runUnitValidate` seam.
- `catalog.go` / `types.go` — `unit` is now a `FINDING_SOURCE_COVERAGE` producer;
  no `coverage` phase/const.
- `packages/maturity-go/dimensions/{dimensions.json,testdata/testgenie_audit_fixture.json}`
  — `coverage` PhaseMap key removed; `unit` finding now carries source 8 in the fixture.

**unit-health engine layout (as built):**
- `api/internal/discovery/discovery.go` — Code Facts client + filesystem fallback.
- `api/internal/executor/{executor,watch,procattr_*}.go` — bounded runner.
- `api/internal/validation/plan.go` — `buildPlan` + all finding codes + `codeSeverity`.
- `api/internal/validation/analyze.go` — shared bounded walk + Phase-5 code consts + test-file predicates.
- `api/internal/validation/coverage.go` — Go profile + Vitest summary + LCOV parsers; threshold from `.vrooli/testing.json`.
- `api/internal/validation/architecture.go` — Go/TS static test-architecture checks.
- `api/internal/validation/quality.go` — Go (go/ast) + TS (regex) test-quality checks + requirement tagging.
- `api/internal/validation/diagnostics.go` — flake markers + runtime-pressure + hang diagnostics.
- `api/internal/validation/service.go` — `Validate` orchestrates discover → plan →
  static analyzers → (optional) execute + coverage → diagnostics → status → maturity.
- `api/internal/validation/maturity_spec_test.go` — anti-drift: codeSeverity ≡ maturity.json (both directions).
- `api/handlers/validation/handler.go` — flat `Response`→proto; builds the shared assessment.
- `cli/domains/validate/handlers.go` — human renderer (workspaces, plan, execution, coverage roll-up, diagnostics) + `--json`.

**Phase 6 steps (per plan §7 Phase 6) — operator UI:** The UI scaffold from
Phase 2 is `ui/src/features/validation/ScenarioValidationWorkbench` with an API
client at `ui/src/api/validation.ts`. Build it into a dense operational
inspection surface: scenario selector, local-maturity summary + next-level
blockers, test-plan table, execution results, coverage dashboard,
architecture/quality findings, diagnostics, global-impact grouping, recommended
skills. Cover states: loading, no-scenario, Code-Facts-degraded, running,
complete, failed, no-tests. Use existing design tokens; type the client from the
generated proto. Add component tests for the core views. Validate with
`cd ui && pnpm install --ignore-workspace && pnpm lint && pnpm test && pnpm build`.

**KNOWN UI gotcha (carry into Phase 6):** the react-vite template's
`vite.config.ts` enforces an 85% global branch-coverage gate; the current
scaffold sits at ~84.46% branches, so `pnpm test:coverage` exits 1 (Unit Health
honestly reports this as TEST_EXECUTION_FAILURE when validating itself). Building
out real UI + tests in Phase 6 should clear it; do not lower the gate to hide it.

**Status vocabulary:** `passed` / `failed` (any error finding) / `degraded`.
CLI exits nonzero when `counts.errors > 0`. Keep stable for Test Genie (Phase 7).

**Restart gotcha:** the scenario compiles from source on `vrooli scenario
restart unit-health`; the CLI auto-rebuilds per-invocation but the **API server
binary does not** — restart the scenario after API changes before live testing.

**Known pre-existing scaffolding debt (NOT introduced by Phase 2; shared with
sibling quality-health):**
- Full `vrooli scenario test unit-health` STANDARDS phase is red (~32 high) —
  the fleet-wide freshly-generated-template standards campaign, not this work.
- `dependencies` phase: governance-review warnings for the scenario's deps
  (need recording in approved-dependency memory) — template baseline.
- `proto` phase WARNINGS only (errors=0 now): `errors`/`health` protos still
  carry `@template react-vite/example` (sibling does too); `errors` domain has
  no handler dir. Accepted scaffold state.
- `docs/manifest.json` nav still lists the old `notes` CRUD domain in a couple
  of entries (lines ~884/924/1416); the `bas/cases/routed-database/` case still
  references `@selector/notes.list`. These are doc/UI-test scaffolding — clean
  up when the UI is built out (Phase 6) and during doc passes; smoke/playbooks
  currently pass regardless.

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-08-21 — active**: Hosted execution remains pending.

- **2026-08-21 — active**: Focused race-enabled repetitions pass; native runner results remain pending.

- **2026-08-21 — active**: This is runnable on Linux, macOS, and Windows CI and directly exercises the historical descendant-orphan defect (`knw-1787267826185908192`); native runner results remain pending.

- **2026-06-16 — done**: **DEFERRED pre-existing doc-rot (NOT caused by this work):** `docs/{GLOSSARY,reference/cli-commands,guides/test-generation,guides/vault-testing}.md` document a `test-genie coverage <scenario>` CLI verb that does **not exist** in the current binary (`test-genie coverage` → "Unknown command") — stale before this cutover; the docs phase validates links/markdown not CLI existence, so it doesn't gate.

- **2026-06-16 — done**: Remaining doc prose mentioning unit/coverage as active phases (RESEARCH/QUICKSTART/GLOSSARY etc.) deferred to the Phase 9 cleanup sweep.
