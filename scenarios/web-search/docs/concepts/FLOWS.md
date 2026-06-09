# Flows — Web Search

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, polling, or mutually
exclusive UI modes.

> **Scaffold status (2026-06-09):** The flows below are *inventoried*
> (Maturity Level 1) from `PRD.md` / requirements but not yet modeled or
> implemented. Formal `*.flow.json` + Quint artifacts (Levels 2–5) are
> implementation work, authored when each owning domain is built. The
> template `notes` attachment-upload flow is still present and will be
> removed with the example domain.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a
workflow model.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Maturity |
|---|---|---|---|---|---|
| Scope-aware federated query | federation / livesearch | A search-hub query routes to web-search. | Learnings always served; live web served only when gated-in; results blended. | Routing/escalation states; rate-limit gate. | L1 (inventory) |
| L3 research-and-reconcile run | research | User/agent starts a deep research run. | A cited brief is produced; findings are captured and the store is reconciled. | Long-running agentic run with ordered phases + budget ordering. | L1 (inventory) |
| Finding status lifecycle | findings | A finding is created, contradicted, or superseded. | Finding transitions across active/disputed/superseded with audit. | Durable state machine with illegal transitions. | L1 (inventory) |
| Live-web budget governor | livesearch | A live-web call is requested. | Call proceeds, is served from cache, or is gracefully rate-limited. | Token-bucket state per window. | L1 (inventory) |

## Flow Details

### Scope-aware federated query *(federation / livesearch)*

- Trigger: search-hub routes a query to web-search's provider(s).
- Inputs: query, limit, requested types/flags (`--type web`, `--all`),
  whether project results were empty/weak (fallback signal).
- Steps:
  1. `web-search.learnings` (SCOPE_PROJECT) always answers from the
     findings index (active + disputed, excluding superseded).
  2. `web-search.live` (SCOPE_EXTERNAL) answers **only if** gated in:
     explicit `--type web`/`--all`, OR fallback escalation when project
     results are empty/weak.
  3. If live is gated in, the budget governor + cache decide whether to
     hit SearXNG or serve cache / return "rate-limited".
  4. Results are returned to search-hub, each carrying provenance; live
     and learnings appear as distinct provider groups.
- Key invariant: **a default query never fires a live web call.** Live
  is reached only via explicit request or fallback escalation.
- Illegal: serving live results on a default (non-gated) query.

### L3 research-and-reconcile run *(research)*

- Trigger: `web-search research run "<question>"` (or API/agent).
- Statefulness: a long-running agent-manager run with ordered phases.
- Steps (states):
  1. `planning` — decompose the question into sub-questions.
  2. `gather` — **read existing relevant findings first** (bounded,
     semantically-near sweep), then loop: search (SearXNG) → fetch/extract
     (browserless, L2) → identify gaps → re-search, verifying across
     sources.
  3. `synthesize` — produce the cited brief (answers the user first,
     within budget).
  4. `distill` — LLM distillation emits structured findings
     (claim + citations + confidence). Auto-capture on by default for L3.
  5. `reconcile` — **bounded post-step**: write new findings, supersede
     outdated ones, flag contradictions the agent cannot confidently
     resolve. Confidence-gated: act above threshold, otherwise flag.
  6. `done` (or `failed`).
- Budget ordering invariant: the user's question is answered (state
  `synthesize` complete) before any curation (`reconcile`) consumes
  remaining budget; a messy store cannot starve the actual research.
- Failure modes: SearXNG/browserless unavailable (degrade to lower
  level), budget exhausted (return partial cited brief), agent-manager
  unavailable (L3 unavailable — only L0/L1/L2 offered).
- Illegal: capturing findings for L0/L1; reconciling before answering.

### Finding status lifecycle *(findings)*

- States: `active`, `disputed`, `superseded`.
- Transitions:
  - `active → disputed` — a contradiction is flagged (below confidence
    threshold to auto-resolve). Carries a `dispute_note` + both sources.
  - `disputed → active` — dispute resolved in this finding's favor
    (human command, targeted re-research, or new evidence).
  - `active → superseded` / `disputed → superseded` — a newer finding
    wins (confidence-gated); `superseded_by` is set; row archived in
    place.
- Illegal transitions:
  - `superseded → active`/`disputed` — archived findings are never
    revived; create a new finding instead.
  - **Silent resolution** of a dispute (must surface with a warning;
    resolution is an explicit, audited action).
  - Any hard delete (supersede/archive only).
- Enforcement: every transition writes an audit row (what/why/which
  brief); default search filter is `status != superseded`.

### Live-web budget governor *(livesearch)*

- States: `available`, `rate_limited`.
- Steps: on a live-web request, check cache first (TTL hit → serve, no
  token spent); else consume a token from the per-window bucket; if the
  bucket is empty → `rate_limited` (return graceful "try later", **no
  external call**); bucket refills on the window schedule.
- Key invariant: when `rate_limited`, SearXNG is never called.

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement (planned) |
|---|---|---|---|
| research / L3 run | planning, gather, synthesize, distill, reconcile, done, failed | reconcile before synthesize; capture at L0/L1; terminal-state escape | `*.flow.json` contract + generated Quint model + replay (when implemented) |
| findings / status | active, disputed, superseded | superseded→active/disputed; silent dispute resolution; hard delete | audit-log enforcement + status guard + `*.flow.json` (when implemented) |
| livesearch / governor | available, rate_limited | external call while rate_limited | token-bucket unit tests + flow model (when implemented) |

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

The four web-search flows above are at **Level 1**. They graduate to
Levels 2–5 as the owning domains are implemented (`research` and
`findings` flows are the highest-value to model formally, given their
ordered phases and illegal transitions).

## Production Shape

Three (Go) or four (UI) files per flow at the top of the feature folder,
plus one `generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.go               # hand: wrapper (package flow)
    flow_test.go                # hand: thin replay delegation (package flow)
    generated/
      model.qnt
      artifact.json
      runtime.go                # package generated
      replay.go
```

UI features that own client-side modes use:

```text
ui/src/features/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.ts               # hand: wrapper
    fixtures.ts                 # hand: replay fixtures
    flow.test.ts                # hand: thin replay delegation
    generated/
      model.qnt
      artifact.json
      runtime.ts
      replay.helper.ts
```

Every flow uses the same file names. The `flow/` directory IS the unit;
the contract no longer declares any output paths or module names.

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`. It should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, clocks, timers, HTTP
clients (SearXNG/browserless/agent-manager), or UI API modules.

The `*.flow.json` contract is the source of truth. Level 5 generated
Quint models, formal artifacts, and Go/TypeScript declarations are
checked-in source artifacts for reviewability, but they are refreshed
and checked by the `flow-verifier` scenario CLI; the scenario lifecycle
runs `make temporal-models` (which calls `flow-verifier verify check`)
before the normal test suite. A Quint file by itself is not accepted:
the model must typecheck, test, verify named invariants, emit
deterministic artifacts, and those artifacts must replay against the
production Go/TypeScript transition functions.

To scaffold a new flow:

```bash
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
```

The scaffold writes the hand-authored files and immediately runs
`generate`, so `check` is green from the moment it returns.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| All four flows above | Not yet modeled (L1). | Model `research` (L3 run) and `findings` (status) first when those domains are implemented — they have the most illegal-transition surface. |
| Curation (P2) | Background GC / telemetry flows not yet designed. | Inventory when curation domain is picked up. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
