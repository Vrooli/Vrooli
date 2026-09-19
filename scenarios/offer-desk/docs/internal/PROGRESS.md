# Progress — Offer Desk

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/offer-desk/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/offer-desk/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Entries are appended when work lands, not while it is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-09-01 | codex | release-ladder-v2 | Archived |
| 2026-08-17 | codex | focused-validation-boundary | Archived |
| 2026-08-13 | codex | Implemented the offer graph, lifecycle gates, multi-clause evaluation, typed CLI, live board/catalog/gate console data reads, fixture-only catalog importer, mobile/first-run experience journeys, and generated contracts. | Archived |
| 2026-08-15 | codex | baseline | Archived |
| 2026-08-15 | codex | Operator catalog rehearsal against a 44-file copy completed: 44 files reported, 28 lifecycle/tier/membership records written, four variants, six membership edges, three requires edges, and eight sells_at edges. | Archived |
| 2026-08-15 | codex | done-with-boundary | Archived |
| 2026-08-15 | codex | done-with-boundary | Archived |
| 2026-08-16 | codex | done-with-boundary | Archived |
| 2026-08-15 | codex | baseline | Archived |
| 2026-08-16 | codex | phase-5-evidence | Archived |
| 2026-08-16 | — | codex | Archived |

### Fresh baseline phase results — 2026-08-16

| Phase | Money Ledger `20260816-175828-55146667` | Offer Desk `20260816-181248-26067354` |
|---|---|---|
| portability | passed | passed |
| structure | passed | passed |
| contracts | passed | passed |
| ui-health | passed | passed |
| api | passed | passed |
| architecture | passed | passed |
| dependencies | passed | passed |
| quality | passed | passed |
| docs | passed | passed |
| performance | passed | passed |
| unit | passed | failed — `COMPANION_REIMPLEMENTED` (`drillClock`) |
| storage | passed | passed |
| workflow | passed | passed |
| business | passed | passed |
| experience | passed | passed |
| tidiness | passed | passed |
| security | passed | passed |
| measures | passed | passed |
| proto | passed | passed |
| branding | passed | passed |
| templates | passed (advisory) | passed (advisory) |

### Fresh baseline backup coverage — 2026-08-16

| Owner/name | Kind | Locator |
|---|---|---|
| claude-code/file-history | filesystem | `~/.claude/file-history` |
| claude-code/history | filesystem | `~/.claude/history.jsonl` |
| claude-code/projects | filesystem | `~/.claude/projects` |
| codex/config | filesystem | `~/.codex/config.toml` |
| codex/history | filesystem | `~/.codex/history.jsonl` |
| codex/sessions | filesystem | `~/.codex/sessions` |
| codex/state | SQLite | `~/.codex/state_5.sqlite` |
| opencode/config | filesystem | `~/.config/opencode` |
| source-ledger/journal | SQLite | `<repo>/scenarios/source-ledger/data/source-ledger.db` |
| swarm-manager/data | filesystem | `~/.vrooli/data/vrooli/swarm-manager` |
| swarm-manager/domain-data | filesystem | `~/.vrooli/data/vrooli/swarm-manager` |
| vrooli/config | filesystem | `~/.vrooli/config` |
| vrooli/data | filesystem | `~/.vrooli/data` |
| vrooli/plans | filesystem | `~/.vrooli/plans` |
| vrooli/runtime-db | SQLite | `~/.vrooli/state/runtime.db` |
| vrooli/secrets | filesystem | `~/.vrooli/secrets.json` |
| vrooli/state | filesystem | `~/.vrooli/state` |
| vrooli-memory/journal | SQLite | `~/.vrooli/data/vrooli/vrooli-memory/vrooli-memory.db` |
| 2026-08-16 | codex | phase-3-done | Archived |
| 2026-08-16 | codex | phase-4-done | Archived |
| 2026-08-16 | codex | phase-5-gate | Archived |
| 2026-08-16 | codex | phase-6-done | Archived |
| 2026-08-16 | codex | phase-8-retirement | Archived |
| 2026-08-16 | codex | phase-8-validation | Archived |
| 2026-08-16 | codex | phase-11-done-with-boundary | Archived |
| 2026-08-17 | codex | phase-6-live-repair | Archived |
| 2026-08-17 | codex | phase-7-validation-pending | Archived |
| 2026-08-17 | codex | phase-10-operator-decision | Archived |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

| 2026-09-01 | codex | superseded-plan-reconciliation-20260901 | Archived |

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

| 2026-08-13 | codex | Full implementation and hardening pass: typed offer graph, audited lifecycle transitions, trigger/fact evaluation, operator-only promotion, fixture-only catalog importer, Money Ledger actuals/posture board, typed CLI projection, live console pages, generated proto/endpoint contracts, viewport-scoped first-run/mobile experience states, and governed dependency/storage/measure declarations. | Archived |
| 2026-08-13 | scenario initialization | Generated from `react-vite`. | Archived |
| 2026-08-13 | scenario initialization | note | Archived |
| 2026-08-17 | codex | repair-plan-final-boundary | Archived |

## Level 3 behavioral-drill evidence — 2026-08-14

| Drill | Artifact and result | Boundary still open |
|---|---|---|
| The unreachable status becomes reachable | Expected-behavior test `TestSchedulerPromotesSatisfiedCandidateWithoutManualEvaluate` passed with a fake ticker and exactly one evaluation. The live CLI path also reached `TRIGGER_MET` after `gates-evaluate --json`, with the evaluation naming `drill_revenue` and its observation age. | The live CLI transcript used an explicit evaluate call; a minute-cadence scheduler transition has not yet been captured as a separate live artifact. |
| Unknown is not false | `TestLifecycleReachesTriggerMetAndUnknownIsNotFalse` and `TestStaleFactIsUnknownAndLeavesCandidateInPlace` passed; they assert UNKNOWN, explanation/gap, and unchanged candidate state. | No separate live CLI run for a missing fact and a stale fact has been captured. |
| A refusal teaches | Live `catalog-transition --status CANDIDATE --json` refused with `candidate_requires_trigger` and remediation. UI proof is `renders a refusal as an explicit error with its remediation [REQ:UI-001] [REQ:UI-002]`; API proof is `TestCandidateRequiresTriggerAndPromotionIsOperatorOnly`. | The three-surface artifact is split across a live CLI response plus API/UI tests, not one live API/CLI/UI recording. |
| An agent cannot promote | Live `gates-promote --role agent --json` returned an operator-only proposal and `catalog-list --json` left the node non-active; backend proof is `TestCandidateRequiresTriggerAndPromotionIsOperatorOnly`. | — |
| The board degrades legibly | Live `board-show --json` returned a catalog row while preserving `money-ledger` and `money-ledger.actuals` availability reasons; backend proof is `TestBoardReportsLedgerUnavailableWithoutInventingActuals`. | This used an unconfigured ledger client rather than stopping the already-running Money Ledger process. |
| The pair's headline claim | After operator promotion, live `board-show --json` returned status `ACTIVE`, rank reason `active and earning nothing`, and source attribution for unavailable actuals. | No screenshot/screen recording was captured. |
| The importer is honest | `TestImportTreeReportsBrokenReferencesWithoutCopyingNarrative` passed with two files read/written, one finding, and narrative excluded; `MIG-001`/`MIG-002` refs resolve. | Manual source-retirement review remains intentionally planned; no source was deleted. |

The Money Ledger progress log records the paired manual-entry, correction,
idempotency, position-degradation, and pending-operator artifacts. The full Offer
Desk run `20260814-031220-2a348d91` is server-owned; the Test Genie UI-health provider
boundary is not represented as a product finding. Requirement statuses remain
planned until the authoritative full runs sync them.

| 2026-08-14 | codex | done-with-boundary | Archived |
| 2026-08-15 | codex | done-with-boundary | Archived |
| 2026-08-16 | codex | phase-2-done | Archived |
| 2026-08-17 | claude | instrument-repair | Archived |
| 2026-08-18 | claude | validation-with-boundary | Archived |

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-08-17 — focused-validation-boundary**: Closed the remaining Offer UI coverage and observer workflow defects.


- **2026-08-13 — in-progress**: UI 34 files/139 tests green; production build green; experience validation still reports capture/binding drift from the running shell and is not yet a completion claim.

- **2026-08-15 — done-with-boundary**: The sole remaining phase failure is experience capture reconciliation (18 unresolved bindings, 20 unjoined captures, 16 unproven claims); no source selectors were changed to hide it.

- **2026-08-16 — done-with-boundary**: Offer run admission was blocked by Test Genie's unrelated invalid `landing-page-business-suite` maturity descriptor.

- **2026-08-16 — phase-3-done**: The remaining `MIGRATION_DEBT` advisory is recorded as future schema-change hardening because this phase introduced no schema migration.

- **2026-08-16 — phase-6-done**: It materialized 28 new nodes (15 offers, 8 variants, 12 channels, 14 revenue lines, 12 deliverables total live counts), eight new `sells_at` edges (17 live), six membership records, and three `requires` relationships; the live catalog now has 40 durable unresolved-reference findings, all non-blocking and visible, with no narrative prose copied.

- **2026-08-17 — phase-6-live-repair**: The live database already contained an audited `business` merge (`5248f61d-e553-49da-ac7f-b73b48dcd3d8` into `127625ce-d99f-4465-80a3-245d3aea8fe7`), so 27 remaining earlier-generation survivors were merged one at a time through `catalog-merge`; four remaining drills were transitioned to `RETIRED`, preserving the already-retired fifth.

- **2026-08-17 — phase-7-validation-pending**: Test Genie run `20260817-071311-ea3469c0` exposed and then led to fixing verifier row-close analysis; post-fix run `20260817-073257-125a9786` remains active in provider readiness without progress, so the phase validation ticket is intentionally pending rather than marked passed.

- **2026-08-13 — partial**: Requirement statuses remain planned pending fresh suite sync and recorded Level 3 drills.

- **2026-08-17 — instrument-repair**: Docs: runbook reconciliation/mapping/degradation sections, CLI and API reference entries, five DECISIONS rows, and a maturity relabel for six docs whose content had outgrown `deferred`.

- **2026-08-18 — validation-with-boundary**: The sole remaining failure is `experience`, which is a capture-provider boundary rather than product debt: two `capture_bindings_unjoined` errors on component `dirty-state-guard` state `prompt-open`, while direct `experience-manager spec validate offer-desk --json` reports PASSED at L3 with zero findings against the same running UI, and the phase passed in both prior runs with no intervening UI change (this repair touched only API, CLI, and docs).
