# Progress — Vrooli Memory

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/vrooli-memory/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
Copied July 7 template history was removed from this scenario’s log.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/vrooli-memory/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

## 2026-09-04 — Outcome-linked learning

Implemented the operator-approved recommendations 1–3: provision the missing
Deployment Manager scope, link recalled advice to observed decisions/outcomes,
and measure learning effectiveness. Vrooli Memory owns the reusable typed
`LearningService.RecordAttempt` and `LearningService.MeasureLearning` operations;
Source Ledger remains the sole journal authority. No new database or program-owned
recall/capture loop was added.

Both usage skills record immutable task attempts through `vrooli-memory learning
record`. Records retain task/attempt identity, ordinal, task/attempt timestamps,
operation/context, outcome, owner evidence, and applied/rejected advice. The API
checks advice against the same scope and rejects advice created after the attempt
started. Replaying an identical attempt returns its entry; a conflicting payload
is refused. Failures, unavailable recall, no match, unknown advice, and test
provenance are explicit.

Both setpoint readers now project three learning rows from the typed sensor:
failure recurrence, effort to first verified success, and advice outcomes.
Readings remain per operation/context. Capped scans, malformed/legacy records,
empty denominators, unresolved tasks, and incomplete histories remain visible.
Targets stay null pending comparable baselines. These are evidence-linked caller
reports, not independently authenticated outcome claims or causal estimates.

Validation:
- Deployment Manager scope provisioned with four prefixed facets, 48-line wake
  budget, and two lines per entry. Journal receipt
  `84514c20-f7a3-40a1-8407-953d7a29d5e2` was returned by scoped semantic recall.
- Live typed fixture receipt `cf599e05-b8e0-449d-99ec-994e861d3d10` replayed with the same
  ID; conflicting payload and cross-scope advice were refused. Same-scope advice
  receipt `3f7705cb-f9fc-4917-b6bc-019362f0b126` was accepted. All fixtures use the
  separate `fixture-learning-outcomes` scope and declared test provenance.
- Domain, Connect handler, CLI, harness adapter, and module checks pass. Fifteen
  named learning tests include additional invalid-input subcases.
- Both setpoint contracts validate; two fresh runtime explanations and three
  declared fixture executions pass. Live readers report no eligible operator
  attempts rather than claiming a healthy zero.
- Desktop Test Genie `20260904-213256-6066e7da`: programs, skill-set, docs,
  business, and structure passed. Deployment Manager
  `20260904-213044-97c7f8cc`: the same five phases passed. Documentation maturity
  still carries the previously reported reference/snippet warnings.
- Vrooli Memory `20260904-213111-debd8bcb`: structure, proto, and skill-set pass;
  API and CLI commands pass. The overall suite remains failed on existing UI
  validation and an unattested completed operator-review requirement. Reports:
  `knw-1788557532619944628` and `knw-1788557543495423027`. No evidence floor or
  completion status was weakened. The old harness fake now explicitly refuses
  the existing CountEntries operation so the generated client interface compiles.

Skill divergence review: no applicable advice means no_match; unreachable recall
means unavailable; retrieved-but-unused hits are not advice uses. An operation
without owner evidence cannot be recorded as verified_success. A fixture cannot
supply an operator baseline. Missing/capped measurements remain unreliable;
comparable data without a baseline remains unbanded. Capture failure never changes
the actual operation outcome. These branches have one conservative interpretation.


Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This is the durable handoff for the shipped unified-memory implementation.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-08 | codex | Phase 17 UI boundary: vrooli-memory is now a harness-state surface (health, projection, import, capture, maintenance) while Source Ledger owns corpus operations. | Archived |
| 2026-08-08 | codex | Phase 18 federation ownership moved fully to source-ledger. | Archived |
| 2026-08-08 | codex | Phase 16: completed the authority cutover. | Archived |
| 2026-08-06 | codex | Phases 9–10: carried explicit scope through journal, recall, forest, facets, rules, proto, API, and CLI surfaces with `agent-memory` defaulting at the handler boundary; added isolation/default/purity coverage and per-scope policy resolution with a cache. | Archived |
| 2026-08-06 | codex | Phase 8: installed and verified the exact canonical prompt block in Claude Code, Codex, and OpenCode convention files; added the Antigravity projection target; made declared empty stores return honest zero-source dry-run observations; and validated every adapter. | Archived |
| 2026-08-06 | codex | Phase 7: added Go-native harness capture hook parsing and non-blocking hook execution, plus reversible install/remove commands for Claude Code and Grok that preserve unrelated native hook entries. | Archived |
| 2026-08-06 | codex | Phase 6: added an in-process maintenance loop with an immediate startup tick, configurable `VROOLI_MEMORY_MAINTENANCE_INTERVAL` (default 6h, zero disables scheduling), clock-owned tickers, durable run/outcome tables, `maintenance-status` CLI/API reporting, and health exposure of the latest run. | Archived |
| 2026-08-06 | codex | Phase 5: added recall-ranked standing-rule pin candidates with persistent recall frequency/recency statistics, recorded an operator-confirmed three-entry pinned set, and made the chosen quarterly (90-day) review interval explicit in code and configuration. | Archived |
| 2026-08-06 | codex | Phase 4: added `rules measure-distribution`, which reports enabled deterministic-rule coverage separately from the classifier tail and fails its 45% gate only on the tail. | Archived |
| 2026-08-06 | codex | Phase 2: journal durability is now enforced at runtime. | Archived |
| 2026-08-06 | codex | Phase 3: frontier reads now use the latest facet assignment, load all derived embedding spaces, and share the compaction candidate eligibility path (policy, pin, recency, root, and vector guards). | Archived |
| 2026-08-06 | codex | Phase 1: made harness import one-directional. | Archived |
| 2026-08-05 | codex | Resolved the phase-4 acceptance contradiction through a validated Plan Manager candidate: deterministic provenance-rule coverage and classifier-tail distribution are now measured separately, so the required `swarm-manager` rule remains intact without treating its expected source skew as classifier imbalance. | Archived |
| 2026-08-05 | codex | Re-measured the policy classifier with a repeatable live fixture: 432/432 provenance-held-out work records correct (100.00%) and 558/558 fall-through triples unanimous (100.00%) through the normal gateway role, after one bounded corpus example per facet and structural work-record guidance. | Archived |
| 2026-08-04 | codex | Final lifecycle validation is green: Test Genie run `20260804-195525-2414d11f` passed all 20/20 phases, business-health is PASSED, and the managed scenario is healthy on API_PORT 17441/UI_PORT 21680. | Archived |
| 2026-08-04 | codex | Completed the P0/P1 policy and operator surface slice: managed wake-block splicing preserves native bytes and rejects malformed markers; scope-aware import identity and data-driven facet vocabulary are live; declarative rules require current-corpus dry runs and carry rule provenance; resident budgets drive wake; pin review renewal, lapse, budget trade-offs, and merge proposals are append-safe; vectors use compact binary storage with brownfield migration; the operator UI/CLI expose frontier, correction, pins, proposals, and rules. | Archived |
| 2026-07-27 | codex | Hardened Claude Code import into a durable, observable, single-flight job. | Archived |
| 2026-07-28 | codex | Completed the harness capability measurement gate. | Archived |
| 2026-07-28 | codex | Completed facets-domain invariants: the seeded, table-owned vocabulary remains exactly six rows after repeated live restarts; journal writes reject an explicit unknown facet with the typed error; only episode entries are compaction candidates and pins exempt them; re-facet retains assignment history; supersession preserves the original journal entry. | Archived |
| 2026-07-28 | codex | Completed cross-depth recall and wake-budget contract hardening. | Archived |
| 2026-07-29 | codex | Phase 1: centralized embeddings, classification, and summarization behind the ai-gateway inference seam with injected fakes. | Archived |
| 2026-07-29 | codex | Phase 2: measured harness stores, context limits, projection behavior, and available capture surfaces. | Archived |
| 2026-07-29 | codex | Phase 3: regenerated and validated journal, facet, forest, recall, federation, and harness contracts. | Archived |
| 2026-07-29 | codex | Phases 4–8: completed append-only journal, closed facet policy, full-fidelity recall, pin-first wake, and derived forest behavior. | Archived |
| 2026-07-29 | codex | Phases 9–11: imported the existing corpus idempotently, recorded calibration outcomes, and validated compaction invariants and temporal models. | Archived |
| 2026-07-29 | codex | Phases 12–15: federated memory through search-hub, projected/captured harness memory, and absorbed swarm work records with provenance. | Archived |
| 2026-07-29 | codex | Phase 16: delivered the operator surface for journal, frontier, facets, and pin curation. | Archived |
| 2026-07-29 | — | Phase 17: mechanically removed the template notes domain; API/CLI/UI focused suites and both temporal models pass. | Archived |
| 2026-08-05 | claude | Correctness, durability, and policy-seam pass. | Archived |
| 2026-08-05 | claude | Scale pass. | Archived |
| 2026-08-07 | claude | Phase 11 (corpus identity and canopy repair). | Archived |

## Entry Template

| 2026-07-27 | design workshop | Scenario generated from `react-vite`. | Archived |

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

- **2026-08-06 — done**: Managed run `304166cd-2b78-46e3-a5a1-561c0cdb9176` completed 2026-08-06T20:10:26Z: all five projection timestamps advanced, six adapters returned honest outcomes, the large swarm-manager import timed out rather than blocking the remaining projections, health stayed healthy, and the entry corpus was unchanged.

- **2026-08-06 — done**: Live candidates, recall attribution, pins/reviews, proposals, and wake were verified against the managed database: 3 active pins, 2 review rows, 0 unresolved proposals, and all 3 pinned entries were the first wake hits.

- **2026-08-04 — partial**: The plan remains an honest partial handoff because W0 has no governing swarm-manager goal artifact and requirements evidence sync currently reports 5/24 complete; remaining P0/P1 validation refs and integration/manual evidence must be earned before normal plan completion.

- **2026-08-04 — done**: Focused Go/CLI/UI suites pass; lifecycle restart is healthy after fixing a single-connection migration deadlock; live Claude import dry-run validated 428 sources and the idempotent run is processing under run `3e851da1-0492-4edb-9f90-f87a93e44670`; comprehensive Test Genie run `20260804-190847-b4136ee9` is server-owned and pending terminal evidence.


- **2026-07-27 — partial**: Focused API and CLI tests passed; full Test Genie evidence remains pending on its separate stuck run.
