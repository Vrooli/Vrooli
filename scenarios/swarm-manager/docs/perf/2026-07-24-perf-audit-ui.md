---
date: 2026-07-24
scenario: swarm-manager
interactions:
  - plan-board-load
  - plan-decision-drawer
traces:
  plan_board_flow: scenarios/swarm-manager/bas/flows/plan-board-load.json
  plan_decision_drawer_flow: scenarios/swarm-manager/bas/flows/plan-decision-drawer.json
status: in-progress
related_skill_run: performance
---

# Perf audit: Plan board and decision drawer render path

UI half of `swarm-manager-performance-hardening-known-hot-path-fixes`. The API
phases removed the server cost (`/next-actions/feed` 1352ms → 2ms, `/plan`
2543ms → 77ms); this audit examines what the client does with those responses
and how often it asks for them.

## Measurement status — read this before trusting any number below

**Component render timings were not captured.** Two `performance-health audit
run` attempts (`plan-decision-drawer`, 19:32Z and 19:36Z) both returned
`AUDIT_OUTCOME_UNAVAILABLE`. The reason string blames
`browser-automation-studio unreachable`, which is wrong: BAS ran both captures
to completion and wrote a 29.9MB trace each time. The real failure is a
readiness race — `audit run` restarts the target into profile mode and starts
driving the flow before the API is back up, so every in-page request logged
`net::ERR_CONNECTION_REFUSED` and the trace measures an empty board.
swarm-manager's own API access log corroborates: rotated at 15:36, new process
first logging at 15:37:14, after the 15:36:38 capture window.

Per the performance skill's own discipline, `unavailable` is not `skipped` and
must not be read as a pass. **No render timing is claimed here.** The defect is
filed against performance-health / browser-automation-studio, which this plan's
Boundaries section forbids changing.

The two perf flows authored for this audit
(`bas/flows/plan-board-load.json`, `bas/flows/plan-decision-drawer.json`) are
committed and correct; they will produce evidence as soon as the capture race
is fixed.

What *was* measured is stated as measured. Everything else is stated as
structural analysis of the code, or deferred.

## Per-component aggregation

| # | Component / behavior | Before | After | Root cause | Method | Outcome |
|---|---|---|---|---|---|---|
| 1 | Plan-lens invalidation fan-out | 1 refetch per mutation | 1 refetch per burst | `scheduleRefresh` used one 150ms debounce for every lens, so mutations more than 150ms apart each triggered a full board refetch (`useGraphWebSocket.ts:91`) | Automated test, fake timers | **Fixed** |
| 2 | Edge events refetching the Plan board | refetch | ignored | `affectsLens` returned true for `edge-add`/`edge-remove` on every lens, including a board that renders no edges (`useGraphWebSocket.ts:65`) | Automated test | **Fixed** |
| 3 | Ranked action list virtualization | n/a | n/a | Not applicable — see below | Code reading | **Deferred (not applicable)** |
| 4 | Plan board column render (288 Next / 153 Later cards) | not measured | not measured | `PlanColumn` and `WaveGroup` map every group and card with no windowing | Blocked by finding above | **Deferred (unmeasurable)** |
| 5 | 60s safety poll alongside socket invalidation | retained | retained | `usePlanData.ts:25` polls every 60s in addition to socket refresh | Code reading | **Deferred (deliberate)** |

### Finding 1 — invalidation fan-out (fixed)

The Plan board refetches a whole cross-entity projection; a graph snapshot
refetch is far cheaper. Both shared one 150ms debounce, so any two mutations
more than 150ms apart — which is most real operator bursts: a batch queue, a
multi-item decision, an agent run touching several items — cost one full board
refetch each, and each new one aborted the in-flight request mid-transfer.

Fixed by giving the plan lens its own 750ms coalescing window
(`invalidationDebounceFor`). Verified by test rather than by hand, so the
plan's manual check ("two rapid backlog mutations produce at most one /plan
refetch") is now a permanent assertion:

- three invalidations 200ms apart → exactly one `fetchGraph("plan", …)` call
- two invalidations 2s apart → two calls, so coalescing did not become
  swallowing
- edge events on the plan lens → zero calls
- edge events on the topology lens → still one call

### Finding 2 — edge events (fixed)

The Plan board is a column projection over items and gates. It renders no
edges, so a topology edge change cannot alter what the operator sees. Refetching
on those events was pure waste. The graph lenses still refresh on them.

### Finding 3 — ranked list virtualization (not applicable)

The plan anticipated that the drawer's 484-entry / 218KB feed would need
virtualization or pagination. It does not, and the reason is structural rather
than a matter of measurement: `DecisionDrawer.tsx:52` renders
`FeedDecisionStream` — a one-card-at-a-time decision stream — whenever the feed
has entries. `RankedActionList`, the component that maps every entry into a DOM
node, is only reached on the **empty** branch, where it renders nothing.

So the full-list render this finding was meant to address never happens at
production data scale. Adding virtualization would add machinery to a code path
that renders at most zero rows. Recorded rather than built.

(This also explains why the broken captures still satisfied the flow's
`ranked-next-action-feed` wait: with the API refused, the feed was empty, which
is exactly the branch that renders that element.)

### Finding 4 — board column render (unmeasurable right now)

The live board renders 288 Next cards and 153 Later cards across 111 wave
groups, all mapped without windowing. This is the one place where the payload
really does become DOM, and it is the strongest candidate for the next round.
It is left unfixed deliberately: the plan conditions this work on measured
render cost, that measurement is blocked by finding 1 of the API-side audit's
sibling defect, and adding virtualization to a board with grouped waves and
drag affordances on an unmeasured hunch is exactly the "ship on vibes" the
performance skill warns against.

### Finding 5 — 60s safety poll (deliberate)

`usePlanData` keeps a 60s silent poll beside socket invalidation. With `/plan`
now at 77ms and server-side invalidation newly covering agent-session proposal
writes (previously the one mutating service that announced nothing), the poll is
a cheap backstop for out-of-band changes. Removing it would trade a bounded,
now-cheap refresh for a class of silent staleness. Kept.

## Payload note

The feed response is 218KB for 484 entries and the board 191KB. Neither is now a
latency problem server-side, but both are parsed on every refresh. Finding 1
reduces how often that parse happens; trimming the payloads themselves is a
response-shape change the plan's Boundaries section places out of scope.
