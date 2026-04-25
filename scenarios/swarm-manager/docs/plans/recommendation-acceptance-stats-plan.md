# Swarm-Manager Recommendation-Acceptance Stats — Implementation Plan

## 1. Purpose

Workshop decisions in swarm-manager already carry both an agent-marked `recommended` option and the user's `selected` key per decision item. None of that signal currently feeds the stats engine: the only event emitted on round save is a `RoundNumber` count, so the Stats panel cannot answer "how often does the user agree with the agent's recommendation?" — the single most direct way to measure whether decision recommendations are improving over time.

This plan adds a `RecommendationAcceptance` metric (and `FreeformOverrideRate` companion) end-to-end: enriched event payload, engine aggregation with per-skill (per-kind) breakdown, response shape, UI card on the Agent tab, and a one-time backfill from existing round files on disk so historical decisions count.

## 2. Required Reading

Run at the start of any execution session:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement implementation-plan-authoring react-coherence test
```

Also read the prior repair plan that this work extends:

```bash
cat scenarios/swarm-manager/docs/plans/stats-feature-repair-plan.md
```

## 3. Problem Statement

Direct inspection of swarm-manager source confirms the gap:

| Observation | Evidence |
|---|---|
| The data model already supports recommendation tracking. | `api/internal/workshop/workshop.go:36-56` — `Item.Options[i].Recommended bool` (agent's pick) and `Item.Selected *string` (user's choice key). `Item.Freeform *string` captures the "Other" override. |
| Round files persist this on disk. | `api/internal/backlog/workshop_save.go:62-89` writes the full `workshop.Round` (with all items) to `<itemDir>/workshop/round-NNN.json`. |
| Round-completion events are emitted, but with empty signal. | `api/internal/backlog/workshop_save.go:91-93` calls `EmitWorkshopRoundCompleted(entityID, roundNumber)`. The payload (`api/internal/eventlog/types.go:213-216`) is just `{round_number: int}`. No per-item data, no recommended/selected/freeform. |
| Engine only computes "max round per entity". | `api/internal/stats/engine.go:350-356`, `metrics.go:199-220` — only `AvgWorkshopRounds` and `WorkshopRoundsSampleSize` are produced. There is no counter for items answered, recommendations chosen, or freeform overrides. |
| UI Agent tab has no recommendation metric. | `ui/src/surfaces/graph/components/StatsPanel.tsx:39-46` — tabs are dashboard/throughput/agent/timing/blocking/scope; the Agent tab consumes `AgentStats` from `types/stats.ts`, which does not expose recommendation acceptance fields. |

Result: there is no way for the user (or the system itself) to detect drift in recommendation quality, regressions after a prompt change, or which skills/kinds produce trustworthy recommendations.

## 4. Scope

**In scope**
- Extend the `decision.workshop_round_completed` event payload to carry per-item decision summary: items asked, items answered, items where user agreed with `Recommended`, items where user picked `Other`/freeform, plus the entity kind (`idea` / `research` / `fix` / `execute` / `chore`).
- Update the `EmitWorkshopRoundCompleted` emitter signature and its single caller in `workshop_save.go` to compute and pass that summary from the persisted `workshop.Round`.
- Extend the stats engine (`api/internal/stats/engine.go`) state and event handler to accumulate the new counters globally and per-kind.
- Extend `AgentStats` in `api/internal/stats/types.go` and `metrics.go` to expose `RecommendationAcceptanceRate`, `FreeformOverrideRate`, and per-kind breakdown, each with a sample-size denominator.
- Add a one-time backfill migration (sentinel-gated, following the pattern from the stats repair plan §6.4) that scans every `workshop/round-NNN.json` on disk and emits a synthetic enriched event per round that has not already been counted.
- Surface the new metrics on the Stats panel's **Agent** tab via `StatsMetricCard` + `InsufficientDataCard` for sub-threshold samples.
- Extend the swarm-manager scenario CLI's `stats` command (`cli/cmd_stats.go`) so the new metrics print in the human-readable output (CLI parity rule from `cli-steer`).
- Fix all lint / type / unit-test issues in modified files, including pre-existing ones.
- Restart the scenario after implementation (`vrooli scenario restart swarm-manager`) and verify health.

**Out of scope**
- Changing the workshop round file schema on disk (the data is already present; only the event payload changes).
- Trend sparklines or drill-down tables for recommendation acceptance over time. The static Agent-tab card and per-kind breakdown are V1; a sparkline mirroring the existing velocity sparkline can be added later but is not required for this plan.
- Cross-scenario aggregation (prompt-manager has its own `decision` workflow with separate semantics; explicitly not unified here).
- Exposing rationale-quality signals (e.g., did the chosen path lead to a successful execution downstream).

**Greenfield constraint:** No compatibility shims, no dual-write of old and new event payloads, no v1/v2 type split. The payload change is a single forward-only schema; the backfill replays disk truth. Old `{round_number}`-only events that already exist in the event DB are tolerated by the engine (treated as "round completed, no item data") but no compatibility code path is added — the engine simply skips empty per-item fields.

## 5. Current Technical Context

**API**
- Event types and payloads: `api/internal/eventlog/types.go` — `EventWorkshopRoundCompleted` at line 72; `WorkshopRoundPayload` at lines 213-216.
- Emitter: `api/internal/eventlog/emitter.go:198-202` — `EmitWorkshopRoundCompleted(entityID string, roundNumber int)`.
- Emitter interface declarations: `api/internal/backlog/handler.go:90` (`EmitWorkshopRoundCompleted(entityID string, roundNumber int)`). Search `rg -n "EmitWorkshopRoundCompleted" api/` to find every interface and call site before changing the signature.
- Caller: `api/internal/backlog/workshop_save.go:91-93`.
- Persisted round shape: `api/internal/workshop/workshop.go:24-56` (`Round`, `Item`, `Option`). `workshop.OtherKey` constant at line 60 marks freeform sentinel.
- Stats engine state: `api/internal/stats/engine.go:129` (`workshopRounds map[string]int`), 153 (init), 350-356 (handler).
- Stats metrics shaping: `api/internal/stats/metrics.go:199-220`.
- Stats response type: `api/internal/stats/types.go:95-109` (`AgentStats`).
- Migration sentinel pattern: `api/internal/eventlog/emitter.go` `EmitMigrationApplied`, plus repair-plan §6.4 prior art.

**UI**
- Stats panel: `ui/src/surfaces/graph/components/StatsPanel.tsx`. Agent tab section follows the `StatsMetricCard` + `InsufficientDataCard` pattern already used for `success_rate` etc.
- Types: `ui/src/types/stats.ts` — `AgentStats` interface mirrors the Go struct.
- Format helpers: `ui/src/lib/stats-format-utils.ts` — reuse `formatRate` for percentages.

**CLI**
- Stats output: `scenarios/swarm-manager/cli/cmd_stats.go:74` (`AvgWorkshopRounds` field), 247-248 (human print). Add new fields here; default is human output (per memory: "CLI default human output").

**Confirmed assumptions** (verified via `rg`):
- `EmitWorkshopRoundCompleted` is called from exactly **one** production site (`workshop_save.go:92`); test files use it via the same emitter. Signature change is contained.
- Per-kind data is reachable: `entityID` passed to the emitter is constructed as `string(kind)+"/"+name` at `workshop_save.go:92`, so `kind` is parseable from the event without reading round files.

## 6. Target End State

- Every workshop round save emits `decision.workshop_round_completed` with payload `{round_number, kind, items_total, items_answered, items_recommended_chosen, items_freeform_chosen, items_unanswered}`.
- The stats engine aggregates running totals globally and per-kind. New counters cover: total decision items asked, total answered, total where the user picked the option marked `Recommended`, total where the user picked `Other` (freeform).
- `AgentStats` in API responses includes:
  - `RecommendationAcceptanceRate float64` (= `items_recommended_chosen / items_answered`)
  - `RecommendationAcceptanceSampleSize int` (= `items_answered` denominator)
  - `FreeformOverrideRate float64` (= `items_freeform_chosen / items_answered`)
  - `RecommendationAcceptanceByKind map[string]{rate, sample_size}` (idea / research / fix / execute / chore)
  - `DecisionItemsTotal int`, `DecisionItemsAnswered int` for transparency.
- On startup, a one-time backfill walks `<workspace>/store/{idea,research,fix,execute,chore}/*/workshop/round-NNN.json`, computes the per-round summary, and emits enriched events for rounds whose `(entityID, roundNumber)` pair is not already represented by an enriched event. A sentinel via `EmitMigrationApplied("recommendation-acceptance-backfill", ...)` prevents re-runs.
- The Stats panel's **Agent** tab shows two new metric cards: "Recommendation Acceptance" (with denominator) and "Freeform Override Rate". Below threshold (`min_sample_meaningful`), `InsufficientDataCard` renders instead of a numeric value. A collapsible "By kind" subsection lists per-kind acceptance with sample sizes; a kind below threshold shows the insufficient-data state inline.
- The CLI `swarm-manager stats` command prints `Recommendation acceptance: 73.2% (n=42)` and `Freeform override: 4.8% (n=42)` in the Agent section.
- All Go tests pass (`go test ./...` in the API), all UI tests pass, lint is clean, the scenario restarts cleanly.

## 7. Implementation Strategy

### Phase 1 — Eventlog payload change

1. In `api/internal/eventlog/types.go`, replace `WorkshopRoundPayload` with:
   ```go
   type WorkshopRoundPayload struct {
       RoundNumber             int    `json:"round_number"`
       Kind                    string `json:"kind,omitempty"`
       ItemsTotal              int    `json:"items_total,omitempty"`
       ItemsAnswered           int    `json:"items_answered,omitempty"`
       ItemsRecommendedChosen  int    `json:"items_recommended_chosen,omitempty"`
       ItemsFreeformChosen     int    `json:"items_freeform_chosen,omitempty"`
   }
   ```
   `omitempty` lets pre-existing events (only `round_number` set) deserialize cleanly without compatibility branches.
2. In `api/internal/eventlog/emitter.go`, change `EmitWorkshopRoundCompleted` to accept the full payload value (or individual fields). Prefer passing the populated payload struct to keep the call site explicit.
3. Update every interface declaration (`rg -n "EmitWorkshopRoundCompleted" api/`) and any test fakes/mocks to match the new signature.

### Phase 2 — Caller computes summary from the round

In `api/internal/backlog/workshop_save.go`:

1. After the round is unmarshaled and validated, before writing the file (so the same in-memory struct can be summarized), call a new pure helper `workshop.SummarizeRound(round) workshop.RoundSummary` (add to `api/internal/workshop/workshop.go`) that returns:
   ```go
   type RoundSummary struct {
       ItemsTotal             int
       ItemsAnswered          int
       ItemsRecommendedChosen int
       ItemsFreeformChosen    int
   }
   ```
   Counting rules:
   - Iterate `round.Items`. Skip non-decision (`Type != "decision"`).
   - `ItemsTotal++` per decision item.
   - If `Selected == nil` → unanswered, no further counters.
   - If `Selected != nil`:
     - `ItemsAnswered++`.
     - If `*Selected == workshop.OtherKey` → `ItemsFreeformChosen++`.
     - Else, look up the option whose `Key == *Selected`; if that option's `Recommended == true` → `ItemsRecommendedChosen++`.
2. Pass the summary plus parsed `kind` into `EmitWorkshopRoundCompleted`.

### Phase 3 — Engine aggregation

In `api/internal/stats/engine.go`:

1. Add to engine state (next to `workshopRounds`):
   ```go
   decisionItemsTotal             int
   decisionItemsAnswered          int
   decisionItemsRecommendedChosen int
   decisionItemsFreeformChosen    int
   decisionByKind                 map[string]*decisionKindCounters
   ```
   where `decisionKindCounters` has the same four counters.
2. Initialize in the engine constructor (mirroring `workshopRounds` init).
3. In the `EventWorkshopRoundCompleted` handler (lines 350-356), after the existing round-number tracking, accumulate the new counters from the payload. Empty/zero fields (legacy events) contribute nothing.
4. Reset all new state in `Rebuild()` next to existing resets.

### Phase 4 — Metrics shaping & response type

1. Extend `api/internal/stats/types.go` `AgentStats` with the new fields listed in §6, plus a `RecommendationAcceptanceByKind map[string]KindRate` where `KindRate` is `{rate float64; sample_size int}`. Define `KindRate` adjacent to `AgentStats`.
2. In `api/internal/stats/metrics.go`, after the existing workshop-rounds block, compute:
   - `RecommendationAcceptanceRate = decisionItemsRecommendedChosen / decisionItemsAnswered` (zero-safe).
   - `FreeformOverrideRate = decisionItemsFreeformChosen / decisionItemsAnswered` (zero-safe).
   - Per-kind map populated from `decisionByKind`.
3. Populate the response struct.

### Phase 5 — Backfill migration

In `api/internal/eventlog/migrations` (or wherever the prior repair plan landed its backfill — re-use that location/pattern):

1. Migration name: `recommendation-acceptance-backfill`.
2. Gate via `EmitMigrationApplied` sentinel: check via `Repository.HasMigration(name)` (or the equivalent helper introduced by the repair plan). Skip if applied.
3. Walk `<workspace>/store/{idea,research,fix,execute,chore}/*/workshop/round-*.json`.
4. For each round file:
   - Load `workshop.Round`, compute `SummarizeRound`, derive `entityID = "<kind>/<name>"`.
   - Query the event DB for an existing enriched `decision.workshop_round_completed` event for `(entityID, roundNumber)` (enriched ⇔ payload has non-zero `ItemsTotal`).
   - If absent: emit a fresh enriched event with timestamp = round file mtime (use `Repository.InsertWithTimestamp` if it exists; else fall back to current time and accept the small drift — record this in the migration log).
5. After completing the walk, call `EmitMigrationApplied("recommendation-acceptance-backfill", ...)`.
6. Trigger via the same startup hook the repair plan uses.

### Phase 6 — UI

In `ui/src/types/stats.ts`:
1. Mirror the new Go fields onto `AgentStats`. Add `KindRate` interface.

In `ui/src/surfaces/graph/components/StatsPanel.tsx` (Agent tab block):
1. Add a `StatsMetricCard` for **Recommendation Acceptance** with `value = formatRate(stats.recommendation_acceptance_rate)` and `denominator = stats.recommendation_acceptance_sample_size`. Below `minSample(history)` → render `InsufficientDataCard` with reason `"Not enough decisions answered yet"`.
2. Add a sibling `StatsMetricCard` for **Freeform Override** using the same sample size.
3. Add a collapsible "By kind" section (default collapsed) that maps `recommendation_acceptance_by_kind` entries; per-kind below threshold → `InsufficientDataCard`.
4. Tooltip copy on the headline card: `"Of decisions where you picked one of the offered options, this is how often you picked the agent's recommendation. A high freeform-override rate means the option set itself is wrong."`

### Phase 7 — CLI parity

In `scenarios/swarm-manager/cli/cmd_stats.go`:
1. Add the new fields to the local response struct around line 74.
2. In the human-output Agent section (around line 247), print the headline rate + denominator, the freeform rate, and (when verbose flag is set or sample size ≥ threshold) per-kind lines.

### Phase 8 — Tests

See §9.

### Phase 9 — Validate & restart

1. `cd scenarios/swarm-manager/api && go build ./... && go test ./... -timeout 600s`
2. `cd scenarios/swarm-manager/cli && go build ./...`
3. `cd scenarios/swarm-manager/ui && npm run typecheck && npm test`
4. `gofumpt -w api/internal/eventlog api/internal/workshop api/internal/backlog api/internal/stats cli`
5. `golangci-lint run` from the api dir.
6. `vrooli scenario restart swarm-manager`
7. Open the UI, confirm Agent tab renders the new cards (likely showing `InsufficientDataCard` initially if backfill produced < threshold answered items; that is correct behavior).
8. `swarm-manager stats` from the CLI; confirm new lines present.

## 8. Contract Decisions

**Event payload — additive, no compatibility branch.** Old events with only `round_number` deserialize via `omitempty`; the engine treats zero `ItemsAnswered` as "no signal" and contributes nothing. No reader code branches on legacy vs new shape.

**Per-kind dimension uses the existing five `BoostN` kinds** (`idea`, `research`, `fix`, `execute`, `chore`) — `api/internal/workshop/workshop.go:77-83`. Unknown kinds are accumulated under a `"unknown"` bucket; this is logged (not silently dropped) so we notice if a new kind appears without map updates.

**"Answered" denominator excludes unanswered items.** A round saved with half its decisions still pending must not penalize acceptance rate — only items the user has actually selected count. This means the user can save a partial round during workshopping and the metric stays honest. Documented in tooltip and CLI help.

**"Other"/freeform counts as rejecting the recommendation.** Freeform answers contribute to `items_answered` (the recommendation-acceptance denominator) but never to `items_recommended_chosen` (the numerator), so a freeform pick lowers acceptance rate exactly like picking a non-recommended option. `FreeformOverrideRate` is tracked separately on the same `items_answered` denominator so the user can read the two side-by-side: "of all answered decisions, X% matched the recommendation" and "of all answered decisions, Y% were freeform". Rationale: from the user's perspective, "Other" is a rejection of the offered set — including the recommended option in that set — and the metric should reflect that.

**Backfill timestamps use round file mtime when the repository supports backdated inserts.** If not, current time is used and the migration log records the drift. Either way, the backfill is single-shot and idempotent via the sentinel.

**No new event type.** All data rides on the existing `decision.workshop_round_completed`. Adding a `decision.item_answered` per-item event was considered and rejected: it would 10–50x event volume for a metric that is fully derivable from the round summary, and would require a separate backfill walk.

## 9. Testing Plan

All tests must be added in the same change set and pass before the plan is considered done.

**Workshop summarizer** (`api/internal/workshop/workshop_test.go`, new test cases):
- `TestSummarizeRound_AllRecommended` — 3 decision items, user picks the recommended option in each → `{3, 3, 3, 0}`.
- `TestSummarizeRound_MixedAcceptance` — 4 items, 2 recommended chosen, 1 alternative chosen, 1 freeform → `{4, 4, 2, 1}`.
- `TestSummarizeRound_PartiallyAnswered` — 3 items, 1 unanswered → `{3, 2, ...}`.
- `TestSummarizeRound_IgnoresInfoItems` — info items don't count toward `ItemsTotal`.
- `TestSummarizeRound_NoRecommendedFlag` — item where no option is marked Recommended; selected option counts as answered but not recommended-chosen → contributes to denominator-minus-freeform but not numerator.

**Emitter** (`api/internal/eventlog/emitter_test.go`):
- `TestEmitWorkshopRoundCompleted_PayloadShape` — captures emitted event, asserts all six fields round-trip through JSON.

**Engine** (`api/internal/stats/engine_test.go`):
- `TestEngine_RecommendationAcceptanceAggregation` — feed a sequence of enriched events across multiple kinds; assert global rate, per-kind rates, sample sizes.
- `TestEngine_LegacyPayloadIgnored` — feed a payload with only `round_number` set; assert zero contribution to recommendation counters but `workshopRounds` still updated (regression guard for the additive contract).
- `TestEngine_FreeformCountsAsRejection` — round with all freeform answers → `items_answered` populated, `items_recommended_chosen=0`, acceptance rate = 0%, freeform override rate = 100%.

**Metrics** (`api/internal/stats/metrics_test.go`):
- `TestAgentStats_NewFieldsPopulated` — round-trip from engine state to `AgentStats`.

**Workshop save handler** (`api/internal/backlog/workshop_save_test.go`):
- `TestWorkshopSave_EmitsEnrichedEvent` — POST a round, capture emitted event from a fake emitter, assert payload matches the round.

**Backfill** (`api/internal/eventlog/migrations/.../recommendation_acceptance_backfill_test.go`):
- `TestBackfill_EmitsForUnseenRounds` — seed disk with 3 round files, no prior events; backfill produces 3 enriched events and a sentinel.
- `TestBackfill_Idempotent` — run twice; second run is a no-op (event count unchanged).
- `TestBackfill_SkipsAlreadyEnrichedRounds` — pre-seed an enriched event for round 1; backfill emits only for round 2 and 3.

**UI** (`ui/src/surfaces/graph/components/__tests__/StatsPanel.test.tsx` or co-located):
- `renders RecommendationAcceptance card when sample >= threshold`.
- `renders InsufficientDataCard when sample < threshold`.
- `per-kind subsection renders with mixed states`.

**CLI** (`cli/cmd_stats_test.go`):
- `TestStatsHumanOutput_IncludesRecommendation` — golden-style test on stdout containing the new lines.

## 10. Rollout / Validation Checklist

```bash
# 1. Build & test
cd scenarios/swarm-manager/api && go build ./... && go test ./... -timeout 600s
cd scenarios/swarm-manager/cli && go build ./... && go test ./...
cd scenarios/swarm-manager/ui && npm run typecheck && npm test

# 2. Lint & format
gofumpt -w scenarios/swarm-manager/api scenarios/swarm-manager/cli
( cd scenarios/swarm-manager/api && golangci-lint run )

# 3. Restart and verify
vrooli scenario restart swarm-manager
vrooli scenario logs swarm-manager | rg "recommendation-acceptance-backfill"
swarm-manager stats   # confirm new lines

# 4. UI smoke
#    - Open StatsPanel → Agent tab.
#    - Confirm "Recommendation Acceptance" card renders (insufficient-data state OK on fresh data).
#    - Expand "By kind" subsection.
#    - Save a workshop round; refresh stats (auto-refreshes ≤60s); confirm sample-size ticks up.

# 5. Event-DB sanity
sqlite3 ~/.local/share/vrooli/swarm-manager/events.db \
  "SELECT count(*) FROM events WHERE event_type='decision.workshop_round_completed' AND json_extract(metadata,'$.items_total')>0;"
sqlite3 ~/.local/share/vrooli/swarm-manager/events.db \
  "SELECT count(*) FROM events WHERE event_type='system.migration_applied' AND entity_id='recommendation-acceptance-backfill';"
# Expect: first ≥ number of enriched rounds; second = 1.
```

## 11. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Signature change to `EmitWorkshopRoundCompleted` ripples through fakes/mocks and breaks unrelated tests. | Pre-flight `rg -n "EmitWorkshopRoundCompleted"` to enumerate every interface, fake, and call before editing. Update them in one pass. |
| Backfill double-counts if "enriched event already present" check is wrong. | Idempotency test (§9) seeds an enriched event and asserts no new event for that `(entityID, roundNumber)`. Sentinel migration also gates the entire pass. |
| `omitempty` makes a legitimate-zero indistinguishable from "field absent" for `ItemsAnswered` (e.g., a round where every item is answered with freeform). | Treat both identically: zero recommended-chosen / zero answered-minus-freeform → contributes 0/0 to the rate, which the metric layer renders as `InsufficientDataCard`. Documented in the contract. |
| `kind` parsing from `entityID = "<kind>/<name>"` breaks if `name` ever contains a slash. | The emitter already passes `entityID` constructed via `string(kind)+"/"+name`; `kind` parsing splits on the *first* slash only. Add an explicit `Kind` field on the payload (the plan already does) so we never re-parse `entityID` for stats. |
| Backfill walks a large workshop tree synchronously and slows startup. | Run backfill in the existing migration goroutine the repair plan introduced; log progress per 100 rounds. If we don't have one yet, run it on a background goroutine after `Rebuild()` returns and accept that the first stats-panel open after upgrade may briefly show pre-backfill numbers (resolved on the next 60s refresh). |
| User opens UI before backfill completes and reports "metric is wrong". | Acceptable on first startup post-deploy; the 60s auto-refresh self-heals. Optional: add a `migration_in_progress` flag on the response and a UI banner. Defer unless the backfill walk routinely takes > 30s in practice. |
| Per-kind `"unknown"` bucket grows silently if a new kind is added later. | Engine logs at WARN whenever it accumulates into `"unknown"`. Test asserts log emission. |

## 12. Non-goals / Prohibited Patterns

- **No** dual-write of old `{round_number}`-only and new enriched payloads. Single forward-only schema; old events tolerated read-side via `omitempty`.
- **No** cross-scenario unification with `prompt-manager`'s `decision` workflow. Different domain semantics; documented as out-of-scope.
- **No** new `decision.item_answered` per-item event type. The round summary covers all signal needs at a fraction of the volume.
- **No** caching layer for recommendation stats. The watermark engine handles incremental aggregation; adding a cache duplicates that.
- **No** heuristic to infer "the user likely agreed" from rationale text or freeform content. Only explicit option-key equality counts toward acceptance.
- **No** migration shim that rewrites old events in place. Read-side tolerance + forward-only schema is the rule.
- **No** UI change beyond the Agent tab in V1. Sparkline / trend view is a follow-up plan.

## 13. Definition of Done

- All items in §10 pass.
- All tests in §9 are added and passing.
- `go build ./...`, `go test ./...`, UI typecheck, UI test, `golangci-lint run` all clean across modified packages.
- The Stats panel's Agent tab renders Recommendation Acceptance and Freeform Override cards. Below threshold → `InsufficientDataCard`. Above threshold → numeric value + denominator.
- The "By kind" subsection renders when at least one kind has any answered items, with per-kind insufficient-data fallback otherwise.
- `swarm-manager stats` CLI prints the new lines under the Agent section.
- The backfill sentinel exists in the event DB after first restart, and `decision.workshop_round_completed` events with non-zero `items_total` exist for every on-disk round file.
- `vrooli scenario restart swarm-manager` succeeds; `vrooli scenario status swarm-manager` reports healthy.
- The plan's greenfield constraint is satisfied: no compatibility branches, no dual-write, no v1/v2 type split anywhere in the diff.
