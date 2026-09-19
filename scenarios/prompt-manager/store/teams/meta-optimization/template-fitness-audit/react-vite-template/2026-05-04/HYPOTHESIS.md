# Iteration 1 — Hypothesis

**Authored**: 2026-05-04, before Phase B baseline measurement and before Phase C/D substrate work.

**Plan**: `/home/matthalloran8/.claude/plans/thanks-for-the-postgres-wise-hearth.md`.

## The claim

We believe the dominant cost in adding domains and endpoints to scenarios generated from the react-vite template is **boilerplate infrastructure that lives in domain folders but is logically substrate**:

- **CLI side**: per-handler `flag.NewFlagSet` + `protojson.Marshal` + `core.Request` + `protojson.Unmarshal` + `apiError` + `decodeEnvelope` ribbon. Each new domain copies ~50 lines of pure infrastructure into its handlers file. The substrate home is `cli-core` (already exists, already consumed via `replace` directive); the helpers don't exist yet.
- **UI side**: per-method `if (!res.ok) throw await decodeApiError(res); const json = await res.json(); return fromJson(...)` ribbon, with `ApiError` and `decodeApiError` themselves misfiled inside `lib/notes.ts` instead of in shared `lib/api.ts`. Each new domain duplicates or imports across domain boundaries. The substrate home is in-template `ui/src/lib/api.ts` (already exists, currently underused).

If we add to `cli-core`:
- A declarative `ArgSchema` + `RunContext` + parser path (mirroring `internal/cli/commandtree`'s shape).
- A generic `Call[Req, Resp proto.Message](app, method, path, req) (Resp, error)` helper.
- An envelope-aware `WrapAPIError` + `DecodeEnvelope`.

…and rewrite the template's notes domain on top, plus relocate `ApiError`/`decodeApiError` into `ui/src/lib/api.ts` with a shared `protoFetch<Req,Resp>` helper that `notes.ts` and `fetchHealth` both consume…

…then the per-replica cost (`lines_added`) for the "add an endpoint" and "add a domain" workflows will drop substantially.

## Predictions

| Scenario | Metric | Predicted movement | Threshold |
|----------|--------|--------------------|-----------|
| 1 — Add endpoint to existing domain | `lines_added` | down | **≥ 40%** drop vs baseline |
| 2 — Add new domain end-to-end | `lines_added` | down | **≥ 35%** drop vs baseline |
| 3 — Add optional field to request shape | `lines_added` | down | **≥ 10%** drop vs baseline |
| 4 — Add error-code path | `lines_added` | down | small drop, no specific threshold (cli-core's WrapAPIError is the only relevant change; most of this scenario's cost is application logic) |
| 5 — Delete a domain | `lines_added` (lines removed) | down (less cleanup tax) | **≥ 20%** drop vs baseline (less infra to delete because less infra was added) |
| 6 — Rename a CLI flag | `lines_added` | flat | no significant change predicted; rename target count is unchanged |

For all scenarios, `central_edits` and `drift_count` are predicted to be **flat or improved**, never worse. `junior_doable` is predicted to **stay yes** or move from yes-with-caveats to plain-yes.

## What "right" looks like

The hypothesis is **right** if at least 2 of the 3 numerically-thresholded predictions (Scenarios 1, 2, 3) land in their predicted ranges, and no scenario regresses on `central_edits`, `drift_count`, or `junior_doable`.

## What "partially right" looks like

Exactly 1 of the 3 thresholded predictions lands; or all 3 land but one of the meta numbers regresses; or the predicted scenarios move in the right direction but undershoot (e.g., Scenario 1 drops 25% instead of ≥40%).

## What "wrong" looks like

Scenario 1 drops by less than 20%, **or** any scenario regresses on `central_edits` / `drift_count` / `junior_doable`.

## What a wrong outcome teaches us

If the predictions don't land:

- **Per-replica cost barely moved on Scenarios 1 and 2** → cli-core's helpers were not the bottleneck. Iteration 2 should look at higher-level layers: service layer (repository → service → handler glue), output contract (the three `Render*Report` shapes), proto code (`protojson` usage at all), or the API handler layer (`api/internal/<domain>/handler.go` boilerplate). The substrate change shipped is still net positive (it codifies a pattern even if it doesn't dominate cost), but iteration 2 redirects to where the cost actually lives.
- **`central_edits` regressed** → the new declarative path leaks into more central registries than the old per-handler `flag.NewFlagSet` path. The substrate design is wrong; either roll back parts of it or restructure cli-core so the new path doesn't add registry coupling.
- **`junior_doable` flipped to no** → the new abstractions are too clever. The substrate is correct in shape but wrong in surface; iteration 2's first job is to add docs / examples / `REPLACING-NOTES.md` updates so a junior can find the new path from the existing materials.

A "wrong" outcome is **not** a failure of the iteration. It's the harness doing its job — preventing iteration 2 from inheriting a false premise.

## Recalibration (post-baseline, pre-substrate)

**Added**: 2026-05-04, after Phase B completed but before any Phase C work.

The baseline measurement made the original numerical thresholds (≥40% / ≥35% / ≥10%) mathematically infeasible to hit honestly. The original predictions assumed CLI + UI-lib boilerplate dominated `lines_added` for scenarios 1–3. The baseline shows the actual composition is API-layer-heavy:

| Scenario | Total `lines_added` | CLI share | UI/src share | API share |
|----------|---------------------|-----------|--------------|-----------|
| 1 — Add endpoint | 369 | 74 (20%) | 79 (21%) | 216 (59%) |
| 2 — Add domain | 961 | 171 (18%) | 220 (23%) | 570 (59%) |
| 3 — Add field | 93 | 27 (29%) | 2 (2%) | 64 (69%) |

Iteration 1's substrate change (cli-core helpers + in-template `protoFetch`) only addresses the CLI and UI-lib slices. Even a perfect substrate that eliminated 100% of CLI boilerplate and 100% of UI-lib boilerplate would cap savings at the CLI + UI/src share — and most of those slices are genuine domain code, not boilerplate, so realistic savings on the *boilerplate portion* are smaller still.

**Honest re-prediction** (what we now expect to see in Phase E):

| Scenario | Original threshold | Recalibrated expectation | Reasoning |
|----------|-------------------|--------------------------|-----------|
| 1 — Add endpoint | ≥40% drop | **5–15% drop** total | CLI is 20% of total; substrate eliminates ~half of that slice → ~10% absolute. Bounded above by CLI+UI-lib share. |
| 2 — Add domain | ≥35% drop | **10–20% drop** total | Same reasoning; bigger CLI footprint per new domain (full handler file + register.go) → cli-core surface eliminates more lines. |
| 3 — Add field | ≥10% drop | **flat-to-2% drop** | Field changes barely touch CLI/UI lib at all — substrate doesn't help here. Original prediction was wrong about which lever moves this scenario. |
| 4 — Add error path | small drop | **flat** | API-only scenario in baseline (0 CLI, 0 UI lib). cli-core's `WrapAPIError` could help if a future error scenario adds a CLI surface; this baseline has none. |
| 5 — Delete domain | ≥20% drop in lines removed | **5–15% drop** | Less infra to delete because less infra was added; same scaling as scenarios 1+2. |
| 6 — Rename flag | flat | **flat** | Unchanged. |

**The shipping decision**: even with recalibrated predictions, iteration 1 ships. Per plan §1.2, "shipping iteration 1 is success even if the final per-domain line counts are still high." The substrate change is worth the cost for:

1. **Pattern codification** — every future scenario gets the declarative shape; new agents copy from a clean template.
2. **Drift surface reduction** — `ApiError` lives in one file; envelope decode lives in cli-core; today's drift between `fetchHealth`'s plain `Error` and notes' typed `ApiError` disappears.
3. **The harness validates as a tool** — even a "small numerical movement" iteration teaches the harness's recipes how to be deterministic, which is load-bearing for iterations 2..N.

**What "right / partially right / wrong" mean now** (overriding §"What X looks like" above for the purposes of Phase E grading):

- **Right** under recalibration: scenarios 1 and 2 each drop by ≥5%, no scenario regresses on `central_edits` / `drift_count` / `junior_doable`.
- **Partially right**: only one of scenarios 1 or 2 hits the recalibrated band, but no regressions.
- **Wrong**: neither scenario 1 nor scenario 2 drops, **or** any scenario regresses on the meta numbers.

**What this teaches iteration 2**: the API layer (`api/internal/<domain>/handler.go` proto-marshal/unmarshal/error-envelope ribbon, plus the SQLite repository boilerplate) is where the real per-replica cost lives. Iteration 2 should target a substrate change there — likely a small `apicore` package mirroring cli-core's role, or an in-template `api/internal/handlerkit/` helper that owns proto-decode/encode + envelope-write. The baseline gives iteration 2 the numbers to predict against.

**Why we are NOT moving the goalposts** in the original §Predictions table: that table is the historical record of what was hypothesized before the baseline was measured. Per plan §2.1, "the hypothesis must be written before the change, not after." The recalibration above is **not** a substitution — it's the next layer of analysis enabled by the harness itself. Phase E will grade against **both** the original thresholds (which will likely fail, as documented) and the recalibrated bands (which we expect to pass). Both grades go in `RESULTS.md` for transparency.

## Out-of-scope predictions

- The hypothesis does **not** predict movement on Tier-1 #3 (route-vs-endpoint parity) since that finding is deferred to iteration 2+. Drift-surface count for Scenario 1 may not improve materially because the route-vs-endpoint pair stays as "hope"-enforced.
- The hypothesis does **not** predict movement on Tier-2 #4 (`cli_commands_seed.json` triple source) since that's deferred. Central-edits for Scenario 2 may still include a `cli_commands_seed.json` row.
- The hypothesis does **not** predict movement on Tier-2 #5 (`Repository.Create` DTO) since that's deferred. Contract location for Scenario 3 may still classify the "callers must leave id/timestamps zero" precondition as code-comment.
