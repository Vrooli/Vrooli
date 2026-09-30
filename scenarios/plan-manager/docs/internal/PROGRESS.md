# Progress — Plan Manager

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/plan-manager/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-25 | Matthew Halloran (agent) | Generated scenario from `react-vite` + `vrooli-default`. | Archived |
| 2026-06-25 | agent | Full product implementation. | Archived |
| 2026-06-25 | Codex | Began `plan-manager-full-impl-validation` Phase 0. | Archived |
| 2026-06-25 | Codex | Landed the first Phase 1 validation slice: production wiring now uses a code-facts-backed `ReferenceResolver` adapter at `handlers/validation`, calling the real `CodeFactsService.DescribeCodeFacts` Connect surface for CODE/DOC path references and preserving `FileResolver` as the honest floor when code-facts is down or cannot express a reference kind. | Archived |
| 2026-06-25 | Codex | Continued Phase 1 validation integration: derived git-control-tower oracle commands now use the verified `baseline diff --scenario <name> --name <baseline>` shape and do not fabricate runnable oracle commands when a baseline name is missing. | Archived |
| 2026-06-25 | Codex | Began Phase 2 autofill repair: command-backed authoring seams now use live substrate command shapes (`git-control-tower baseline show --scenario --name --json`, `code-facts facts describe <target> --include surfaces,parse_units --json`) instead of the dead no-arg GCT show / `code-facts search` paths. | Archived |
| 2026-06-25 | Codex | Landed a Phase 3 silent-data-loss slice: authoring `Finalize` now returns typed `ErrAuthoredMarkup` when non-empty references/phases markup cannot be parsed into structured references/phases, and the plans markdown parser rejects malformed machine-readable reference markers and malformed `### Phase N - Title` headings. | Archived |
| 2026-06-25 | Codex | Landed a Phase 4 execution semantics slice: `GetNext` now advances past the current pointer to the next later non-done phase when one exists, while `GetStatus` continues to expose the resume point as the earliest unfinished phase. | Archived |
| 2026-06-25 | Codex | Continued Phase 4 and landed Phase 5 migration. | Archived |
| 2026-06-25 | Codex | Continued Phase 6 architecture and Phase 7 debris cleanup. | Archived |
| 2026-06-26 | Codex | Continued Phase 8 and a small Phase 9 UI-health slice. | Archived |
| 2026-06-26 | Codex | Implemented phase-native Plan Manager authoring. | Archived |
| 2026-06-27 | Codex | Began `plan-manager-hardening-readiness` with the continue-loop and phase-context gate slice. | Archived |
| 2026-06-27 | Codex | Continued `plan-manager-hardening-readiness` Phase 3 typed-anchor hardening. | Archived |
| 2026-06-27 | Codex | Continued `plan-manager-hardening-readiness` Phase 4 required-reading cutover. | Archived |
| 2026-06-27 | Codex | Continued `plan-manager-hardening-readiness` Phase 5 execution-runner hardening. | Archived |
| 2026-06-27 | Codex | Continued `plan-manager-hardening-readiness` Phase 6 CLI/API/seam polish. | Archived |
| 2026-06-27 | Codex | Completed the remaining `plan-manager-hardening-readiness` implementation/validation slice. | Archived |
| 2026-06-27 | agent | Wizard + log hardening pass. | Archived |
| 2026-07-01 | Codex | Added an execution feedback checkpoint so smaller agents are explicitly guided to review phase feedback before marking a phase done. | Archived |
| 2026-09-02 | Claude | Corpus-integrity and retention pass. | Archived |

## 2026-09-05 — Plan-family and skill/program follow-through

Implemented revision-aware runnable admission, projected future waves, durable Agent Manager family execution, a typed child execution skill, and the bounded execution board. Ordinary execution progress preserves graph review; admission rechecks current dependencies, claims and concurrency. Focused family/execution tests and owner-executed program fixtures pass. Agent Manager integration tests prove two independent children followed by a dependent child, restart recovery, owner acceptance and cancellation claim retention. This is deterministic integration evidence, not a real autonomous initiative trial.

Current acceptance remains incomplete: Test Genie shared admission rejected fresh suite requests; Agent Manager UI branch coverage is below its existing gate; the four execution telemetry rows in PROBLEMS.md remain unavailable. No empirical supervision improvement or policy promotion is claimed. The active plan log carries the exact evidence and remaining obligations.

## 2026-09-06 — Advisory completion follow-up

Ordinary completion now relies on explicit outcome assessments while preserving
strict evidence gates for plans that opt into certification. The execution
captures its completion policy at start and persists it with the execution-local
scope state, so concurrent edits to the shared plan cannot silently change the
meaning of an active run. Legacy executions without the marker retain the
live-plan fallback. Updated domain/proto/validation comments and added a
regression test covering an advisory run after the authored plan is changed to
certification.

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

- **2026-06-25 — done**: UX console + coverage/quality-health + live `vrooli scenario test` are the remaining gates.

- **2026-06-26 — done**: Full `vrooli scenario test plan-manager` run `20260626-030706-557ad8db` still failed before these fixes in unit/structure/ui-health/standards/architecture/tidiness; the unit failure was the now-fixed coverage gate, while remaining reds include lifecycle Makefile structure, env validation, architecture classification, duplicated-code tidiness, and standards timeout. Targeted `test-genie execute plan-manager ui-health --json` did not return output and was terminated after remaining silent; no server-owned run was aborted. Phase 10 MoM velocity emit remains blocked because `meta-optimization-manager` exposes trials list/run/history/read RPCs only, with no public ingest/velocity write contract to call without coupling to its internals.

- **2026-06-26 — done**: Full `vrooli scenario test plan-manager --json` run `20260626-210808-8360d98c` reached 12/17 phases passing; remaining failures are ui-health bridge/blank screenshot, standards timeout, Lighthouse performance 0.73 < 0.75, and duplicated-code tidiness.

- **2026-06-27 — done**: Full scenario Test Genie and consumer inversion remain deferred to later plan phases.

- **2026-06-27 — done**: Completed the remaining `plan-manager-hardening-readiness` implementation/validation slice.

- **2026-06-27 — done**: Downstream forwarding is owned internally via seams (`BugReporter`→scenario-qa, `RecordWriter`→swarm-manager) with documented pending-stub defaults (mirroring `VelocitySink`/MoM; the live command/API adapters are a deferred drop-in follow-up); a failed/unavailable forward is never fatal — the entry stays `pending`/`sync_failed` and is retried via `log sync`.
