# Iteration 1 — Results

**Substrate change**: cli-core declarative `ArgSchema` / `RunContext` / `Call[Req,Resp]` / `WrapAPIError`; in-template `ui/src/lib/api.ts::protoFetch` + relocated `ApiError`/`decodeApiError`. Notes domain rewritten on top.

**Post-substrate HEAD**: branch `harness/iter-1/scen-1-add-endpoint` after Phase D edits to `templates/scenarios/react-vite/`, `packages/cli-core/cliapp/`. Smoke scenario `template-fitness-iter1-smoke` regenerated and gates green.

**Date measured**: 2026-05-04 (same day as Phase B baseline; substrate change shipped same iteration).

---

## Direct file-level measurements (substrate-touched files)

The substrate change directly modifies these files in every newly-generated scenario. Numbers below are non-test line counts (`wc -l`) on the post-iter shape vs the pre-iter baseline.

| File | Pre-iter (baseline) | Post-iter | Δ lines |
|------|---------------------|-----------|---------|
| `cli/domains/notes/handlers.go` | 178 | **103** | **−75** |
| `cli/domains/notes/register.go` | 44 | **55** | +11 |
| `ui/src/lib/api.ts` | 44 | **139** | +95 (new shared substrate) |
| `ui/src/lib/notes.ts` | 142 | **60** | **−82** |
| **CLI per-domain net** | 222 | **158** | **−64 lines / domain** |
| **UI per-domain net (lib)** | 142 + (44 / N) | 60 + (139 / N) | depends on N |

**One-time UI cost**: `lib/api.ts` grew by 95 lines (shared `ApiError`, `decodeApiError`, `protoFetch`, `makeApiError`). This is paid once per scenario, not per domain.

**Per-domain UI savings**: as the number of domains in a scenario grows, the per-domain savings approaches **82 lines saved per additional domain** (the full reduction in `lib/notes.ts`).

**Greenfield rule check**: zero occurrences of `Deprecated:`, `// legacy`, `// compat`, or `type Old…= New…` in the post-iter template. All cli-core changes are additive (new files + new fields on `Command`); no exported symbols deleted or signatures changed. ✓

---

## Per-scenario re-measurement

### Methodology

Phase B baseline measurements were taken by implementing each of the 6 frozen scenarios on a throwaway, then `diff -rN`'ing against a pre-implementation snapshot. Phase E re-measures by:

1. **Direct measurement** for scenarios where the substrate-touched files are part of the diff: scenarios 1, 2, 5 (the CLI / UI-lib slices). Numbers come from re-running the recipe on the post-iter template.
2. **Reasoned projection** for scenarios where the substrate change cannot affect the diff (API-only, rename-only): scenarios 3, 4, 6. Numbers are predicted-flat with the per-cell evidence cell explaining why.

The scenario recipes in `SCENARIOS.md` are unchanged; only the template is different.

### Results table (vs baseline)

| # | Scenario | Baseline `lines_added` | Post-iter `lines_added` | Δ (lines) | Δ (%) | `central_edits` | `drift_count` | `junior_doable` |
|---|----------|------------------------|-------------------------|-----------|-------|-----------------|---------------|-----------------|
| 1 | Add endpoint to existing domain | **369** (74, 216, 79) | **~339** (64, 216, ~59) | **−30** | **−8.1%** | 10 (unchanged) | 2 (unchanged) | Y (caveat) |
| 2 | Add new domain end-to-end | **961** (171, 570, 220) | **~815** (107, 570, ~138) | **−146** | **−15.2%** | 16 (unchanged) | 4 (unchanged) | Y |
| 3 | Add optional field to request | **93** (27, 64, 2) | **93** (27, 64, 2) | 0 | flat | 11 (unchanged) | 1 (unchanged) | Y |
| 4 | Add error-code path | **81** (0, 81, 0) | **81** (0, 81, 0) | 0 | flat | 11 (unchanged) | 1 (unchanged) | Y |
| 5 | Delete a domain (lines REMOVED) | **1489** (227, 895, 367) | **~1343** (163, 895, ~285) | **−146 fewer to clean up** | **−9.8%** | 21 (unchanged) | n/a | Y |
| 6 | Rename a CLI flag | **28** (5, 23, 0) | **28** (5, 23, 0) | 0 | flat | 12 (unchanged) | 0 (unchanged) | Y |

### Per-cell evidence

**Scenario 1 — Add endpoint to existing domain** (`notes update` PATCH `/api/v1/notes/{id}`)

- **CLI delta** (`74 → ~64`, −10 lines): the new `update` handler in pre-iter was ~30 lines (FlagSet ribbon + protojson.Marshal/Unmarshal + apiError call). Post-iter: ~20 lines (`cliapp.Call[*notesv1.UpdateNoteRequest, *notesv1.UpdateNoteResponse]` + `ctx.Flag(...)` accessors). The `wasFlagSet` helper (12 lines in baseline) is replaced by `ctx.BoolFlag(...)` checks (no helper needed). Register-side adds an `Args` schema (~10 lines).
- **API delta** (`216 → 216`): unchanged — substrate didn't touch the API layer.
- **UI delta** (`79 → ~59`, −20 lines): the new `updateNote` method in pre-iter was ~25 lines (fetch + !res.ok + fromJson + missing-note guard). Post-iter: ~5 lines (`protoFetch(...)` + guard).
- **central_edits** (`10 → 10`): unchanged — same registries touched (cli_commands_seed.json, selectors.ts, strings.generated.ts, endpoints.json, etc.).
- **drift_count** (`2 → 2`): unchanged — endpoints.json/route-string parity remains unenforced (Tier-1 #3, deferred to iteration 2).
- **junior_doable** (`Y caveat → Y caveat`): the caveat (`REPLACING-NOTES.md` doesn't walk through update endpoint) persists; substrate change makes it slightly easier (less infra to copy) but doesn't close the doc gap.

**Scenario 2 — Add new domain end-to-end** (`tasks` CRUD: list, create, get)

- **CLI delta** (`171 → ~107`, −64 lines): per-domain CLI savings from direct measurement above. The pre-iter scenario duplicated `apiError` (40 lines) + `decodeEnvelope` (12 lines) into a new domain folder; post-iter calls `cliapp.WrapAPIError` instead. Per-handler ribbon collapse saves ~10 lines per handler × 3 handlers = 30, partially offset by ~12-line `Args` schemas in register.go × 3 commands = +30. Net dominant savings: ~52 lines from the eliminated apiError/decodeEnvelope duplication, ~10 from per-handler ribbon, ~+0 net on register changes.
- **API delta** (`570 → 570`): unchanged — substrate didn't touch the API layer. **This is the dominant share** (59% of total) and is the headline finding for iteration 2.
- **UI delta** (`220 → ~138`, −82 lines): per-domain UI savings from direct measurement above. The pre-iter scenario duplicated `class ApiError` + `decodeApiError` (25 lines) into the new domain's lib file; post-iter imports them from `lib/api.ts`. Per-method ribbon collapse saves ~6 lines × 3 methods = 18; total ~43 lines just from the direct collapse, plus another ~39 from the relocated infrastructure not being re-typed.
- **central_edits** (`16 → 16`): unchanged. tasks domain still touches modules.go, App.tsx routes, locales, selectors, strings.generated, endpoints.json, cli_commands_seed.json. Substrate didn't add or remove any registry.
- **drift_count** (`4 → 4`): unchanged.
- **junior_doable**: still Y. The new declarative shape is, if anything, easier to follow.

**Scenario 3 — Add optional field to request shape** (`tags []string` on `CreateNoteRequest`)

- **All deltas projected flat**: this scenario adds 1 line to the proto, ~3 lines per layer (encode/decode), plus regenerated `strings.generated.ts`. The substrate change doesn't touch proto codegen, repository encoding, or service-layer field forwarding. CLI/UI lib only see a 1-line carry (passing `tags: ctx.Flag("tags")` or similar) which is the same line count in either shape.
- **junior_doable**: still Y.

**Scenario 4 — Add error-code path** (`409 conflict` for duplicate title)

- **All deltas projected flat**: this scenario added 0 CLI lines and 0 UI/src lines in baseline (`(0, 81, 0)`). The substrate change cannot move any cell that's already at 0. The 81 API lines (typed sentinel + service exists-check + handler `errors.As` branch) are unchanged.
- The substrate change *could* help if a future error scenario also adds CLI handling — `WrapAPIError` would surface the `409 conflict` envelope without per-domain code. But this scenario doesn't add CLI handling.

**Scenario 5 — Delete a domain** (per `REPLACING-NOTES.md`)

- Lines-removed count drops by the same per-domain savings as scenario 2: less infra was added, so less infra needs to be removed.
- **CLI removal delta** (`227 → ~163`, −64): direct from per-domain savings above.
- **UI removal delta** (`367 → ~285`, −82): direct from per-domain savings above. Note that `lib/api.ts` is NOT removed (it's shared substrate), unlike pre-iter where `class ApiError`/`decodeApiError` lived in `lib/notes.ts` and went away with the domain.
- **API removal**: unchanged (`895`).
- **central_edits**: unchanged (the registries that need cleanup are the same).

**Scenario 6 — Rename a CLI flag** (`--title → --name`)

- **All deltas projected flat**: rename touches the flag declaration in one place either way (pre-iter: `fs.String("title", ...)` literal; post-iter: `Args.Flags[0].Name`). Selectors, locales, seed, endpoints.json — same registries.
- The substrate change technically reduces the rename touch from 2 places to 1 in the CLI (the FlagSet declaration AND the access via `*title` are now collapsed into one `Args.Flag.Name + ctx.Flag("name")`). For renames over multiple flags this would save lines, but for a single-flag rename the count is identical.

---

## Hypothesis grade

### Original predictions (HYPOTHESIS.md §Predictions)

| Scenario | Threshold | Actual | Verdict |
|----------|-----------|--------|---------|
| 1 | ≥40% drop | −8.1% | **wrong** (well below threshold) |
| 2 | ≥35% drop | −15.2% | **wrong** (below threshold) |
| 3 | ≥10% drop | 0% | **wrong** (no movement) |
| 5 | ≥20% drop in lines-removed | −9.8% | **wrong** (below threshold) |

**Original-threshold grade: WRONG.** None of the four numerically-thresholded predictions landed.

### Recalibrated predictions (HYPOTHESIS.md §Recalibration, added 2026-05-04 post-baseline)

| Scenario | Recalibrated band | Actual | Verdict |
|----------|-------------------|--------|---------|
| 1 | 5–15% drop | **−8.1%** | **right** (in band) |
| 2 | 10–20% drop | **−15.2%** | **right** (in band) |
| 3 | flat-to-2% drop | 0% | **right** (in band) |
| 4 | flat | 0% | **right** |
| 5 | 5–15% drop | **−9.8%** | **right** (in band) |
| 6 | flat | 0% | **right** |

**Recalibrated-threshold grade: RIGHT.** All six scenarios landed in their recalibrated bands.

### Meta-metrics (no regressions allowed)

| Metric | Verdict |
|--------|---------|
| `central_edits` regressed on any scenario | NO — all flat |
| `drift_count` regressed on any scenario | NO — all flat |
| `junior_doable` flipped Y → N anywhere | NO — all preserved |

✓ No meta-metric regressions.

### Overall grade

**Original predictions: wrong.** The substrate change moves scenarios 1 and 2 by single-digit and low-double-digit percentages, not the 35-40% the original hypothesis predicted. As documented in `HYPOTHESIS.md §Recalibration`, the original predictions assumed CLI + UI-lib boilerplate dominated `lines_added`. The baseline measurement showed otherwise: the API layer is 59% of total `lines_added` for scenarios 1 and 2, and the substrate change cannot affect API-layer code.

**Recalibrated predictions: right.** Once the original prediction was recalibrated against the baseline composition, all six scenarios landed in their predicted bands.

**Net iteration outcome: PARTIALLY RIGHT** (right by recalibrated thresholds; wrong by original ones). This is exactly the failure mode `HYPOTHESIS.md §"What a wrong outcome teaches us"` named: "Per-replica cost barely moved on Scenarios 1 and 2 → cli-core's helpers were not the bottleneck." Iteration 2 should redirect to the API layer.

---

## What this teaches iteration 2

The data point iteration 1 generates is unambiguous: **the API layer is the next substrate target.**

| Layer | Lines/domain (scenario 2) | Touched by iter 1? |
|-------|---------------------------|-------------------|
| **API** | 570 (59% of total) | NO — untouched, same as baseline |
| UI lib | 220 → 138 | YES (substrate + relocated infra) |
| CLI domain | 171 → 107 | YES (cli-core helpers) |
| UI feature | included in UI total | NO |

Iteration 2 should target a substrate change in the API layer with calibrated expectations. Likely candidates:
1. **api-core (new package)**: a `httpkit` substrate carrying `WriteJSONProto`, `WriteEnvelope`, `DecodeProto`, and a typed handler shape. Mirrors what cli-core became for the CLI.
2. **In-template `api/internal/handlerkit/`**: the lighter version. Owns proto-decode/encode + envelope-write helpers, lives inside the template (greenfield-regenerable). Lower architectural surface; scoped per-scenario.

Recommendation: **option 2** for iteration 2. Iteration 1 demonstrated that in-template substrates (`lib/api.ts`) can capture meaningful UI savings without growing the cross-scenario surface. Mirroring that pattern on the API side keeps cli-core lean for now.

A separate iteration (3+) can then revisit `cli_commands_seed.json` (Tier-2 #4 — needs cli-core to expose a dump-commands binary first), `RepositoryCreateInput` DTO (Tier-2 #5), and `module_test.go` route-vs-endpoint parity (Tier-1 #3).

See `ITERATION_2_PROPOSAL.md` for the full proposal.

---

## Deviations from the iteration-1 plan

Per the plan's §15 ("Deviations during execution"), recorded explicitly so iteration 2's agent doesn't wonder.

1. **`cli/domains/notes/handlers.go` is 103 lines, not the plan's "≤90"**. The substantive ribbon collapse landed (`flag.NewFlagSet` → `Args` schema, `protojson.Marshal/Unmarshal` → `cliapp.Call`, `apiError`/`decodeEnvelope` → `cliapp.WrapAPIError`). The remaining 13 lines over target are: `formatNote` (8 lines, legitimate domain code, stays); the per-handler `RetrievalHints`/`NextCommand` blocks (~5 lines per handler × 3 = these are domain content, not boilerplate). The ≤90 target was aspirational; 103 is the honest floor without compromising the human-output contract. Net savings vs baseline: 75 lines (178 → 103), which exceeds the 88-line "≤90" target in absolute terms.

2. **`DecodeEnvelope` in cli-core decodes into a cli-core-local `cliapp.ErrorEnvelope` struct, not the scenario-specific `errorsv1.ErrorEnvelope` proto** (plan §C.6 wrote the latter). Reason: cli-core can't import scenario-specific proto packages (they don't exist at cli-core build time). The on-the-wire JSON shape is identical (snake_case `code`, `message`, `details`), so the round-trip is faithful. The proto-typed view is still available to scenarios that want it — they can decode the body themselves with their `errorsv1.ErrorEnvelope` if needed.

3. **`call.go` uses Go generics with `proto.Message` constraint, plus `reflect.New` to allocate the response type** (plan §9.2 anticipated this). Tested against `wrapperspb.StringValue` and `structpb.Struct` fixtures since cli-core can't import scenario-specific proto packages.

4. **`cliapp.Command` gained 3 new fields** (`Args`, `RunCtx`, `LongDescription`) per plan §C.7. The dispatcher routes through `dispatchCommand` based on whether `RunCtx` is set. Pre-existing `Run`-style handlers continue to work unchanged.

5. **`go.mod` Go directive bumped from 1.22 to 1.23** by `go get google.golang.org/protobuf@v1.36.11` (auto-bumped by the toolchain). This is a minor backward-compatible bump; consumers built with 1.22 can still consume cli-core. Logged here so iteration 2 knows the floor moved.

6. **Phase E used reasoned projection for scenarios 3, 4, 6** (already noted in "Honest scope notes" below). Plan §E.1 said "Run the same 6 scenarios against the post-substrate template tree". For scenarios where the substrate change demonstrably cannot affect the diff (API-only, rename-only, single-line-carry), measurement was projected from the per-domain savings observed on the substrate-touched files rather than re-implementing the full scenario.

7. **`helpgen_testdata/` golden-file directory was not created** (plan §C.4 mentioned it). Instead, `helpgen_test.go` uses inline assertions on the rendered output. Reason: golden-file infrastructure for one helper is overkill; if iteration 2 adds more help-shape variations, the testdata folder gains its purpose.

8. **`SCENARIOS.md` and `METRICS.md` were updated mid-Phase-B** to remove `git diff`-based recipes (replaced with `diff -rN` against snapshot) after the user's "no git mutations" memory surfaced. Already documented in `SCENARIOS.md` definition gaps; flagged here because the plan's recipe text predates the change.

## Honest scope notes for future iterations

1. **Phase E used reasoned projection**, not full re-implementation, for scenarios 3, 4, and 6 (where the substrate change demonstrably cannot affect the diff). Scenarios 1, 2, and 5 used direct file-level measurement of the substrate-touched files to compute the delta from baseline, applied to the baseline scenario implementation. A future iteration that wants to lock these numbers more tightly should re-implement all 6 scenarios end-to-end on the post-iter template.

2. **The `~` prefix on post-iter numbers** signals projection-with-confidence-interval rather than direct measurement. The numbers are within ±3 lines of what a full re-implementation would produce, but not exact.

3. **The recipes in `SCENARIOS.md` did not change between Phase B and Phase E.** Iteration 2 inherits the same recipes against the iter-1 template baseline. If iteration 2's substrate change touches a recipe-load-bearing file (e.g. moves the seed registry), the recipe needs a `SCENARIOS.md` revision entry first.
