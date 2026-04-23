# Priority-Sort Pending-Questions Endpoint + CLI Wrapper

## Problem Statement

`GET /api/v1/backlog/pending-questions` (`scenarios/swarm-manager/api/internal/backlog/pending_questions.go`) iterates `store.LoadAll(nil)` in the map-iteration order produced by the filesystem store, so the grouped `items` array is effectively unordered. Downstream consumers (the UI's command post today, and the upcoming `workshop-decision-prep` heartbeat agent) must re-sort client-side, every caller duplicating the ranking logic. The ranking logic itself lives only in TypeScript (`scenarios/swarm-manager/ui/src/lib/dependency-sort.ts` + `backlog-sort.ts`): topological depth bucket, then effective priority (manual priority minus a capped boost of `0.5 × transitive incomplete dependents`, cap = 3), recency tiebreaker. No Go equivalent exists.

The `workshop-decision-prep` specialist picks the **top K** pending-decision briefs every 3-hour heartbeat. Without server-side ranking, it cannot pick "top K" — only "K arbitrary". The sync skill's UX (highest-impact decisions first within a ~5-minute session) collapses.

There is also no CLI surface for this endpoint. The sync skill's fallback and the prep specialist both need to invoke it from a shell context. Every other backlog resource has a CLI wrapper (`list`, `get`, `files`, …) but `pending-questions` does not.

## Goal / Success Criteria

1. `GET /api/v1/backlog/pending-questions` returns `items[]` sorted deterministically by the same ranking rule the UI applies: dependency-aware topological depth first, then effective priority (`manual - min(0.5 × transitive_incomplete_dependents, 3)`), then recency desc, then `(kind, name)` as a final tiebreaker.
2. The endpoint accepts `?source=workshop|review|all` (default `workshop`), `?limit=N` (0 = unlimited), and `?initiative=NAME` query params. Filtering happens server-side; `limit` caps the returned `items[]` length after sort.
3. A new CLI subcommand `swarm-manager backlog pending-questions [--source …] [--limit N] [--initiative NAME] [--json]` mirrors the shape of `backlog list` (human summary + structured JSON).
4. The UI continues to render correctly. Its own `sortBacklogItems` call over `PendingQuestionsItem[]` becomes a no-op because the server order is already canonical (verified by existing UI tests).
5. No DB/schema changes. No proto changes required (existing response shape is preserved; new sort is pure reordering plus the three new query params).

## Non-Goals

- Changing the pending-questions response **shape** (no new fields on `PendingQuestion` / `PendingQuestionsItem`). Rank scores stay server-internal.
- Adding a question-level answer endpoint (tracked by `idea/workshop-question-level-answer-endpoint`).
- Sorting the `questions[]` inside a single item by anything other than their current source-grouped emission order (workshop decisions first, then review items). Within-item ordering is not part of this ticket.
- Implementing the `workshop-decision-prep` specialist or `workshop-decision-sync` skill (separate initiative members).
- Touching `source=review` behavior beyond wiring the filter — no review-path changes, tests, or UX work here.

## Required Reading

**Kind-baseline:** (none; `execute` has no mandatory kind skill.)

**Discovered & embedded:**

- `prompt-manager skills read cli-steer` — thin-wrapper CLI conventions, `ParseInterspersed`, `cliutil.JSONFlag`, command registration pattern used by `cmd_backlog.go`.
- `prompt-manager skills read api-steer` — query-param conventions, handler signature rules, response stability guarantees (we must not break the existing JSON shape).
- `prompt-manager skills read seam-discovery-and-enforcement` — where to place the ported ranker so multiple packages (`backlog`, future `overview`) can consume it without a cycle.
- `prompt-manager skills read test` — table-driven Go tests, ordering-sensitive fixture design.

**Orchestration context:**

- `scenarios/swarm-manager/initiatives/workshop-decision-triage/orchestration-summary.md` — this item MUST land before `execute/workshop-decision-prep-agent`; the prep agent's top-K selection assumes this sort is in place.

**Reference files (read before implementing):**

- `scenarios/swarm-manager/api/internal/backlog/pending_questions.go` — current handler, to be extended (not rewritten).
- `scenarios/swarm-manager/api/internal/backlog/pending_questions_test.go` — existing test harness & fixture helpers.
- `scenarios/swarm-manager/ui/src/lib/dependency-sort.ts` — canonical ranking algorithm to port. Constants: `UNBLOCK_WEIGHT = 0.5`, `UNBLOCK_CAP = 3`, `SORT_RESOLVED_STATUSES = {"completed"}` (archived via `archivedAt`).
- `scenarios/swarm-manager/ui/src/lib/backlog-sort.ts` — composition (effective priority + recency tiebreaker via `buildCommandPostCompare`).
- `scenarios/swarm-manager/api/internal/depgraph/graph.go` — existing Go primitives (edges, dependents, unblocked/blocked); lacks depth + transitive-dependent counting.
- `scenarios/swarm-manager/cli/cmd_backlog.go` (`cmdBacklogList`, `cmdBacklogGet`) — shape to mirror for the new subcommand.

## Technical Context

**Ranker location.** The TS implementation in `dependency-sort.ts` is the single source of truth across the UI (backlog tab, command post). No Go port exists. "Reuse the existing ranker; do NOT duplicate" (per orchestration summary) cannot be taken literally across languages — the honest interpretation is **establish a canonical Go port in a shared package, and let every future Go consumer (backlog, overview, stats) depend on it**. The TS code stays until we're confident every consumer uses the server-side order; a later ticket can collapse the UI sort to a no-op assertion.

**Package placement options.** Discussed as decision D1. Candidates:

- `api/internal/depgraph` — already has dependency primitives; adding `ComputeDepthMap`, `ComputeUnblockingMap`, `ComputeEffectivePriority` there is the natural home. *Would require expanding acceptance_allow.*
- `api/internal/backlog/ranking.go` — keeps everything inside the current acceptance glob; downside is overview/stats can't import (backlog imports overview via bridge, so cycle risk).
- A new `api/internal/backlogrank` package — clean, cycle-free, greenfield.

**Existing response shape (unchanged):**

```go
type PendingQuestionsResponse struct {
    Items []PendingQuestionsItem `json:"items"`
}
type PendingQuestionsItem struct {
    Kind      BacklogKind       `json:"kind"`
    Name      string            `json:"name"`
    Questions []PendingQuestion `json:"questions"`
}
```

**Query params to add.**

| Param | Values | Default | Behavior |
|-------|--------|---------|----------|
| `source` | `workshop`, `review`, `all` | `workshop` | Filters which collector runs per item. `all` preserves current behavior (both collectors). |
| `limit` | non-negative int | `0` (unlimited) | Caps the sorted `items[]` length; applied AFTER sort/filter. |
| `initiative` | initiative name | empty | Restricts to items whose `Initiative == NAME`. Items with no initiative are excluded when set. |

**Dependency data source.** The ranker needs the **full** backlog item list (to compute correct depth / unblocking counts even for items with no pending questions) — not just the filtered subset. `store.LoadAll(nil)` already returns everything; we compute the depth & unblocking maps once over the full set, then filter to items-with-questions, then sort.

**UI no-op verification.** `ui/src/lib/command-post-utils.ts:22` currently calls `sortBacklogItems(…, buildCommandPostCompare(unblockingMap), allItems)`. After this change, the server's order already matches that compare, so the call becomes idempotent. We verify with an existing UI test that asserts sort output given a pre-sorted input (if none exists, we ADD one — but this is the only UI change, and it's test-only).

## Approach / Strategy

Port the TS ranker to Go in a single shared package, extend the handler to accept query params and apply the sort, then layer the CLI wrapper on top. Phases run strictly sequentially; phase 2 depends on phase 1's exported API.

## Phased Plan

### Phase 1 — Port the ranker to Go

- Create `{{RANKER_PKG}}` (see D1) with the following exports, matching the TS semantics bit-for-bit:
  - `const UnblockWeight = 0.5`
  - `const UnblockCap = 3.0`
  - `var SortResolvedStatuses = map[string]struct{}{"completed": {}}` (archived via `ArchivedAt != ""`)
  - `type RankItem interface { Kind() string; Name() string; Status() string; DependsOn() []string; ArchivedAt() string; Priority() int; UpdatedAt() time.Time }` (plus a concrete adapter from `backlog.BacklogItem`).
  - `func ComputeDepthMap(items []RankItem) map[string]int`
  - `func ComputeUnblockingMap(items []RankItem) map[string]int`
  - `func EffectivePriority(manual int, transitiveDeps int) float64`
  - `func CommandPostLess(a, b RankItem, depth, unblock map[string]int) bool` — combined depth → effective priority → recency → `(kind, name)` ordering.
- Unit tests (`ranking_test.go`) mirror the TS test cases in `ui/src/lib/dependency-sort.test.ts` and `ui/src/lib/backlog-sort.test.ts`: depth propagation with completed deps, cycle stabilization, dangling refs, unblocking cap, recency tiebreaker, kind/name tiebreaker.

### Phase 2 — Extend the pending-questions handler

- In `pending_questions.go`:
  - Parse `source`, `limit`, `initiative` from `r.URL.Query()`. Validate `source ∈ {workshop, review, all}` (400 on invalid), `limit` non-negative int (400 on invalid), `initiative` accepted as-is.
  - Compute `depth`, `unblock` maps once from the full `items` list (using the new ranker package).
  - Change the per-item collection loop so it only calls `collectWorkshopQuestions` if `source != review`, and `collectReviewQuestions` if `source != workshop`.
  - Apply `initiative` filter before the question-emit check.
  - After building `[]PendingQuestionsItem`, sort it with `CommandPostLess` (wrapped to look up the underlying `BacklogItem` for each entry — carry a parallel `map[itemKey]BacklogItem` populated during the loop).
  - Apply `limit` trim after sort.
- Extend `pending_questions_test.go`:
  - `TestPendingQuestions_SortedByPriority` — 3 items with priorities 5/3/1 → returned order 1, 3, 5.
  - `TestPendingQuestions_DependencyDepthBeatsPriority` — high-priority dependent still sorts below its low-priority dependency.
  - `TestPendingQuestions_UnblockingBoost` — item with 10 transitive dependents beats same-priority peer.
  - `TestPendingQuestions_SourceFilter` — `source=workshop` excludes review rows and vice versa.
  - `TestPendingQuestions_LimitTrim` — `limit=2` trims 5 items to 2, preserving sort order.
  - `TestPendingQuestions_InitiativeFilter` — items outside the given initiative excluded.
  - `TestPendingQuestions_InvalidQueryParams` — 400 for bad `source`, negative `limit`.

### Phase 3 — CLI wrapper

- Add `cmdBacklogPendingQuestions` to `cmd_backlog.go`:
  - Flags: `--source` (default `workshop`), `--limit` (int, default 0), `--initiative` (string), `--json` (via `cliutil.JSONFlag`).
  - Use `cliutil.ParseInterspersed`.
  - Build `url.Values`, call `a.core.Get("/backlog/pending-questions", query)`.
  - Structured path: if `--json`, passthrough via `printJSONIfRequested`.
  - Human path: summary line (`Found N items, M pending questions total`), per-item block with `[kind] name` header, one line per question showing `id | topic/title | options summary`.
  - `Retrieval Hints` footer with `backlog get --kind … --name …` for the first item.
- Register in `app.go` (find `BacklogList: a.cmdBacklogList,` and add `BacklogPendingQuestions: a.cmdBacklogPendingQuestions,` alongside; add the corresponding `Name: "pending-questions"` command entry).
- CLI tests in `cli/app_test.go` mirror the existing `cmdBacklogList` test pattern: argument validation (invalid `--source`), JSON flag pass-through, round-trip decode against a stub API response.

### Phase 4 — UI verification & doc touch-up

- Run existing UI test suite (`scenarios/swarm-manager/ui`) — `buildCommandPostCompare` applied to server-sorted input must be a no-op. If any test fails, the Go port diverges from the TS algorithm and must be fixed in Phase 1.
- Update `docs/concepts/ARCHITECTURE.md#priority-ranking` (if present) with a one-line note that the canonical implementation is now in `{{RANKER_PKG}}` and the TS version mirrors it.

## Testing Strategy

| Layer | Test | Location |
|-------|------|----------|
| Ranker unit | Depth/unblocking/effective-priority semantics | `{{RANKER_PKG}}/ranking_test.go` |
| Handler unit | Sort order, source/limit/initiative filters, 400 on bad input | `api/internal/backlog/pending_questions_test.go` |
| CLI unit | Flag parsing, JSON passthrough, error wrap | `cli/app_test.go` (mirroring `cmdBacklogList` tests) |
| Integration | End-to-end: populate fixture backlog, hit real API, verify order matches oracle computed in the test | `api/internal/backlog/pending_questions_test.go` (existing `setupTestHandler` + `createTestItem` infra) |
| UI regression | Existing `command-post-utils.test.ts` assertions stay green with pre-sorted input | `ui/src/lib/command-post-utils.test.ts` (no new tests; regression guard only) |

No manual verification required — full automated coverage. Coverage for the sibling `chore/workshop-decision-sync-tests` item is the end-to-end harness across all three execute items; this item owns its unit + integration tests only.

## Risk Matrix

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Go port diverges from TS algorithm (off-by-one, different tiebreaker) | Medium | High — UI & CLI disagree on "top K" | Port TS test cases verbatim; Phase 4 runs UI tests against server-sorted input as a cross-check. |
| New ranker package introduces import cycle (`backlog ↔ overview`) | Medium | Medium | Place ranker in standalone package (D1) so it has no internal dependencies; decided before phase 1. |
| `--initiative` semantics surprise callers (exclude-no-initiative vs include-all) | Low | Low | Documented explicitly here; covered by `TestPendingQuestions_InitiativeFilter`. |
| Limit applied before sort would silently drop high-priority items | High if implementation is naive | High | Phase 2 test `TestPendingQuestions_LimitTrim` locks order-then-trim invariant. |
| UI has implicit reliance on **unsorted** order (unlikely but possible) | Low | Medium | `command-post-utils.test.ts` regression guard catches any assumption break. |
| Existing pending-questions clients cache by identity and expect stable order across unrelated backlog mutations | Low | Low | Sort is deterministic given the same dataset; unrelated mutations already invalidate cache elsewhere. |

## Cross-Initiative Implications

- `execute/workshop-decision-prep-agent` (next sibling) **blocks on this** per the orchestration summary. Once this lands, the prep agent can call `backlog pending-questions --source workshop --limit K` and trust the order.
- `execute/workshop-decision-sync-skill` (grandchild) uses the CLI wrapper's human output for default (non-JSON) interactions; keep the human formatter concise.
- `chore/workshop-decision-sync-tests` adds end-to-end coverage across the chain; this item's unit tests are strictly a subset of that future surface — don't pre-bake the e2e tests here.
- `idea/workshop-question-level-answer-endpoint` is intentionally deferred; if it lands later, the sort order in this endpoint is what the future endpoint will also inherit, so any stability concerns belong here.

No other initiatives touched. The ranker package itself is net-new and has no current consumers besides the handler we're modifying.

## Acceptance Criteria

1. `curl -s localhost:PORT/api/v1/backlog/pending-questions?source=workshop&limit=5` returns at most 5 items, sorted by the documented rule.
2. `curl -s localhost:PORT/api/v1/backlog/pending-questions?source=invalid` returns HTTP 400.
3. `swarm-manager backlog pending-questions --json` produces the same JSON as the raw HTTP endpoint.
4. `swarm-manager backlog pending-questions` (human mode) prints a summary section + per-item blocks + hint footer.
5. All new & existing tests pass (`go test ./...` under `scenarios/swarm-manager/api`, `scenarios/swarm-manager/cli`).
6. UI tests stay green (`cd scenarios/swarm-manager/ui && npm test -- --run`).
7. `acceptance_allow` holds: changes confined to `scenarios/swarm-manager/cli/**` and `scenarios/swarm-manager/api/internal/backlog/**` — unless D1 resolves to a package outside `backlog/`, in which case update `acceptance_allow` before starting phase 1.

## Verification Steps

Automated only — the user has asked not to let the active Claude Code session restart the scenario it runs inside.

1. `cd scenarios/swarm-manager/api && go test ./internal/backlog/... ./internal/depgraph/... -timeout 120s`
2. `cd scenarios/swarm-manager/cli && go build ./... && go test ./... -timeout 60s`
3. `cd scenarios/swarm-manager/ui && npm test -- --run src/lib/command-post-utils.test.ts src/lib/backlog-sort.test.ts`
4. `cd scenarios/swarm-manager && go vet ./... && gofumpt -l api cli` (no output expected).

After landing, the **user** runs `vrooli scenario restart swarm-manager` out-of-band — do not restart from inside this session.

## Post-Landing Follow-Ups

- Track (do not implement in this ticket): collapse the UI's `sortBacklogItems` call to an assert-only no-op once a release cycle has confirmed server parity.
- If overview / stats pages want ranking-aware ordering later, they import the same ranker package — no new work beyond the import.
- If the sync skill surfaces a concrete need for sorting **within** a single item's `questions[]`, escalate to `idea/workshop-question-level-answer-endpoint`.

<!-- Readiness gaps remaining: D1 package-placement decision; D2 "--source=all" ordering (interleave vs grouped); D3 scope of UI test guard; D4 whether to expose rank diagnostics on the JSON response. -->
