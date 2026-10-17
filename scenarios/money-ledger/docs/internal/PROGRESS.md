# Progress — Money Ledger

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/money-ledger/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/money-ledger/docs/internal/PROGRESS.md`.
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
| 2026-08-17 | codex | focused-validation-boundary | Archived |
| 2026-08-13 | codex | Implemented the journal, ingestion, position, goals, statements, typed CLI, live-data console pages, fixture-only operator-input importer, mobile/first-run experience journeys, and generated contracts. | Archived |
| 2026-08-15 | codex | baseline | Archived |
| 2026-08-15 | codex | Capability-gap implementation: operator-mode import is reachable through RPC and CLI with dry-run default; thirteen operator paths are classified into eight monetary fields, four measures, and one refused derived rate. | Archived |
| 2026-08-15 | codex | done-with-boundary | Archived |
| 2026-08-15 | codex | done-with-boundary | Archived |
| 2026-08-15 | codex | baseline | Archived |
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

`data-backup-manager coverage report --json` measured `registered_count=18`,
`recommended_count=91`, `sensitive_count=30`, `planned_count=18`,
`backed_up_count=18`, and `verified_count=17`. Registered targets were:

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
| 2026-08-16 | codex | phase-5-gate | Archived |
| 2026-08-16 | codex | phase-7-done | Archived |
| 2026-08-16 | codex | phase-11-done | Archived |

### Operator request — 2026-08-16, now entered through Money Ledger `/adapters`

The real file is intentionally unanswered. Runway remains `UNKNOWN` until the
operator supplies the fields below; no null is interpreted as zero.

| Path | What it means | Where to gather it |
|---|---|---|
| `cash` | Cash and readily available operator funds | `HOW_TO_GATHER_INPUTS.md#cash-on-hand` |
| `monthlyBurn.aiApi` | Monthly model, gateway, STT/TTS, and embedding cost | `HOW_TO_GATHER_INPUTS.md#monthly-ai--api-cost` |
| `monthlyBurn.infrastructure` | Monthly VPS, storage, CDN, DNS, and backup cost | `HOW_TO_GATHER_INPUTS.md#monthly-infrastructure-cost` |
| `monthlyBurn.saas` | Monthly Stripe, email, analytics, and business SaaS cost | `HOW_TO_GATHER_INPUTS.md#monthly-third-party-saas-cost` |
| `monthlyBurn.tooling` | Monthly development, CI, monitoring, and tooling cost | `HOW_TO_GATHER_INPUTS.md#monthly-tooling-cost` |
| `timeAllocation.product` | Share of the last seven days spent building product | `HOW_TO_GATHER_INPUTS.md#time-allocation` |
| `timeAllocation.services` | Share of the last seven days spent on paid services | `HOW_TO_GATHER_INPUTS.md#time-allocation` |
| `timeAllocation.ops` | Share of the last seven days spent on recurring operations | `HOW_TO_GATHER_INPUTS.md#time-allocation` |
| `servicesRevenue.leadGen` | Monthly revenue from lead-generation services | `HOW_TO_GATHER_INPUTS.md#services-revenue` |
| `servicesRevenue.doneForYou` | Monthly revenue from done-for-you services | `HOW_TO_GATHER_INPUTS.md#services-revenue` |
| `servicesRevenue.consulting` | Monthly revenue from consulting services | `HOW_TO_GATHER_INPUTS.md#services-revenue` |
| `servicesTime.hoursThisWindow` | Hours spent on active services in the current window | `HOW_TO_GATHER_INPUTS.md#services-time` |
| `subscriptions.mrr` | Subscription MRR from telemetry; not a manual operator number | `HOW_TO_GATHER_INPUTS.md#subscription-mrr` (console displays this as refused derived rate) |


## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

| 2026-08-13 | codex | Full implementation and hardening pass: books, append-only journal/reversals/transfers, typed ingestion and fixture-only operator-input import, derived position/goals/statements, typed CLI, live console pages, generated proto/endpoint contracts, viewport-scoped first-run/mobile experience states, and governed dependency/storage/measure declarations. | Archived |
| 2026-08-13 | scenario initialization | Generated from `react-vite`. | Archived |
| 2026-08-13 | scenario initialization | note | Archived |
| 2026-08-17 | codex | repair-plan-final-boundary | Archived |

## Level 3 behavioral-drill evidence — 2026-08-14

The following artifacts were produced against the running local services or by the
scenario's expected-behavior tests. A live CLI/API/UI claim is marked only where the
surface was actually exercised; fixture tests are not presented as end-to-end proof.

| Drill | Artifact and result | Boundary still open |
|---|---|---|
| Manual entry is first-class | `money-ledger ingest event-ingest --json` wrote `drill-sale-1` with `BASIS_OPERATOR_ASSERTED`; `ledger journal-list --json` returned the same typed row. | No upstream adapter was needed for this manual-path proof. |
| A correction cannot be an edit | `money-ledger ledger journal-reverse --json` created `9acbd76a-d330-4791-8871-bca9e7962dc1` with `reversal_of=1aa3be2a-43c8-40b7-8981-c8baf7ba6138`; `journal-list --json` returned both entries. Backend proof is also `TestStoreIngestIsIdempotentAndReversalIsAppendOnly`. | A dedicated live browser scan for edit/delete affordances remains absent; the UI state suite covers the authored journal states. |
| Ingestion is idempotent | Repeating the same `event-ingest --json` returned `duplicate=true` and `skipped_duplicates=1`; backend proof is the same named test. | — |
| Position degrades legibly | `money-ledger ingest adapter-run --adapter-id drill-manual --json` returned `status=failed`, a named reason, and `last_success_at`; `position-show --json` returned `partial=true` with no synthesized zero. Backend proof is `TestFailedAdapterIsVisibleAndNeverWritesZero`; UI fixture proof is `names an unavailable adapter and its impact [REQ:POS-004]`. | The UI artifact is fixture-based, not a browser read of this exact live outage. |
| Pending operator input is absent, not zero | `TestOperatorInputsImportPreservesPendingAsAbsent` passed; `TestOperatorInputsFixtureImportCarriesSourceProvenance` passed. | The live fixture import is not exposed as a safe operator CLI workflow. |

The remaining joint drills are recorded by Offer Desk because their primary claim is
the offer lifecycle or board; the paired evidence is in that scenario's progress log.
The full suite run `20260814-031020-841b7ec6` was still server-owned at the time of
this entry and its UI-health execution phase remains an infrastructure boundary.

| 2026-08-14 | codex | done-with-boundary | Archived |
| 2026-08-15 | codex | done-with-boundary | Archived |
| 2026-08-16 | codex | Production-ready console write/read slice: added typed book/account/event/reversal/transfer methods, first-class manual entry with duplicate reporting, append-only audit hydration, inter-book transfer, goal declaration and verdict rendering, statement period selection/export, and adapter register/run/import controls. | Archived |
| 2026-08-16 | codex | evidence | Archived |
| 2026-08-16 | codex | done-with-boundary | Archived |
| 2026-08-16 | codex | evidence | Archived |
| 2026-08-17 | codex | phase-8-contract | Archived |
| 2026-08-17 | codex | phase-9-entry-path | Archived |
| 2026-08-17 | codex | phase-8-production-book | Archived |
| 2026-08-16 | codex | phase-2-done | Archived |
| 2026-08-17 | claude | operator-entry-repair | Archived |
| 2026-08-18 | claude | validation | Archived |

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-08-17 — focused-validation-boundary**: Closed the remaining Money UI/experience defects: semantic chart naming and table coverage, explicit operator-input region/table roles, and truthful manual tiering for mutation and repeated chart-instance claims.


- **2026-08-13 — in-progress**: UI 34 files/139 tests green; production build green; experience validation still reports capture/binding drift from the running shell and is not yet a completion claim.

- **2026-08-16 — phase-3-done**: The remaining `MIGRATION_DEBT` advisory is recorded as future schema-change hardening because this phase introduced no schema migration.

- **2026-08-13 — partial**: Requirement statuses remain planned pending fresh suite sync and recorded Level 3 drills.

- **2026-08-17 — repair-plan-final-boundary**: Fresh comprehensive run `20260817-155535-f8e5ae8a` reached 19/21 before the stale CLI evidence was regenerated; final run `20260817-160840-882cc6c6` was aborted after remaining non-terminal with no phase output.

- **2026-08-16 — evidence**: The repository has no runnable alternative environment, so the comparative side-by-side half remains explicitly pending rather than fabricated.

- **2026-08-17 — phase-8-contract**: The remaining Phase 8 operator boundary is the production book name and currency; no production book or live normalization has been performed without those values.

- **2026-08-17 — operator-entry-repair**: Twelve fields remain deliberately absent as `pending-operator`.
