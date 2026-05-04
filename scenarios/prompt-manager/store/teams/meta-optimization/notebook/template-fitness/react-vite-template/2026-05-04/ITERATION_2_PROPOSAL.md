# Iteration 2 — Proposal

**Authored**: 2026-05-04, after iteration 1's Phase E completed.
**Author**: meta-optimization team / toolchain-validator (handing off to iteration-2 agent).

---

## Context inherited from iteration 1

- Iteration 1 added cli-core's declarative `ArgSchema` / `RunContext` / `Call[Req,Resp]` / `WrapAPIError`, plus in-template `ui/src/lib/api.ts::protoFetch` and relocated `ApiError`/`decodeApiError`. Notes domain rewritten on top.
- Hypothesis was **wrong** by original thresholds (≥40% drop on scen1, ≥35% on scen2 didn't land), **right** by recalibrated thresholds (5–15% drops on scen1/scen5, 10–20% on scen2 all landed).
- The baseline composition pinned the actual cost: **API layer is 59% of total `lines_added`** for the heavy scenarios (1 and 2). Iteration 1's substrate change was on the wrong layer for hitting the original numbers.

The lesson: cli-core helpers are net-positive (they ship the pattern, eliminate duplicated `apiError`/`decodeEnvelope` per CLI domain, halve UI per-method ribbon), but the dominant per-replica cost lives in the **API handler + service + repository layer**.

---

## Hypothesis for iteration 2

**Claim**: the dominant cost in adding domains and endpoints to scenarios is now **API-layer boilerplate** — specifically:

- Per-handler `protojson.Unmarshal` / `protojson.Marshal` ribbon in `api/internal/<domain>/handler.go`
- Per-handler `httpx.WriteError` envelope-write ribbon for translating typed sentinels (`ErrNoteNotFound`, `ErrInvalidNote`, `ErrDuplicateTitle`) into HTTP envelopes
- Per-method service interface + repository interface + sqlite struct mapping (some of this is genuine code, some is boilerplate)
- The `endpoints.go` declaration ribbon (one entry per endpoint, repeated structurally identical block)

**If we add to the template** (greenfield, regenerable, no cross-scenario substrate):
- An in-template `api/internal/handlerkit/` package owning `DecodeRequest[T]`, `WriteJSONProto`, `WriteEnvelope`, plus a typed-error → envelope translator that uses the canonical envelope writer.
- A small `endpoints.gen.go` codegen flow that derives the endpoint table from a single annotated source.

…**then** the per-replica cost (`lines_added`) for scenarios 1 and 2 will drop further:

| Scenario | Iter-1 actual | Iter-2 predicted | Predicted Δ |
|----------|--------------|------------------|-------------|
| 1 — Add endpoint | 339 | **265** | −22% |
| 2 — Add domain | 815 | **610** | −25% |
| 3 — Add field | 93 | 80 | −14% (handler-side ribbon collapse) |
| 4 — Add error path | 81 | **45** | −44% (typed-sentinel → envelope is the canonical translation) |
| 5 — Delete domain | 1343 | 1100 | −18% |
| 6 — Rename flag | 28 | 28 | flat |

These are **calibrated** predictions: they match the API-layer share of the baseline (~60%) times the fraction of API code that is genuinely boilerplate (~30–40% by inspection of the baseline implementations). Iteration 2's hypothesis grade will use these bands as the right thresholds.

---

## Suggested phase split for iteration 2

Mirror iteration 1's structure:

- **Phase A — harness adjustments**: minimal. The 6 scenarios stay frozen. `BASELINE.md` for iteration 2 is the post-iter-1 column already in `RESULTS.md`. Add an `Iter-2` column to `RESULTS.md`.
- **Phase B — re-baseline**: skip. Iteration 2's baseline IS iteration 1's post-iter measurement. Same recipes, same numbers.
- **Phase C — substrate change**: in-template `api/internal/handlerkit/` package with the helpers above. NO `api-core` cross-scenario package this iteration; keep architectural surface small until a third iteration confirms the pattern is durable.
- **Phase D — template refactor**: notes domain handler + service migrated onto handlerkit. Delete the duplicated `protojson.Unmarshal` ribbon from `handler.go`; collapse `WriteError` calls to envelope-aware helpers.
- **Phase E — re-measure**: same 6 scenarios, calibrated predictions.

Time budget similar to iteration 1: 1 day end-to-end if focused.

---

## Out of scope for iteration 2

(Continued deferrals from iteration 1.)

- **Tier-1 #3** — `module_test.go` route-vs-endpoint parity. Iteration 2 might happen to address this if the `endpoints.gen.go` codegen flow becomes the single source of truth, but that's a side-effect not a goal.
- **Tier-2 #4** — `cli_commands_seed.json` triple source. Still needs cli-core to expose a dump-commands binary; iteration 3+ work.
- **Tier-2 #5** — `Repository.Create` DTO. Could be folded into iteration 2 if the handlerkit pattern naturally requires a DTO, but don't force it.
- **Tier-3 #6** — `endpoints.go` description-style convention. Paint; iteration 4+.

---

## Risks specific to iteration 2

| Risk | Likelihood | Mitigation |
|---|---|---|
| `handlerkit` ends up coupling to specific proto types | medium | Use generics (Go 1.22+) for `DecodeRequest[T]`; only the envelope writer needs to know about the scenario-local `errorsv1.ErrorEnvelope`, and that can be wired via the same `cliapp.ErrorEnvelope`-style local struct. |
| Codegen for `endpoints.gen.go` regresses central-edits count | medium | Verify recipe: scen1 and scen2 should *not* get a higher `central_edits` post-iter-2 than post-iter-1. If they do, the codegen is leaky. |
| Service-layer changes are mistaken for handler-kit changes | low | Be explicit: handlerkit is *transport-only*. Service interface + repository implementation stay per-domain. |
| The 35–44% predictions miss again | low | Iteration 1's recalibration discipline means iteration 2 starts with calibrated predictions. If they miss, the next layer (proto codegen? service-layer DTOs?) is the next target — same iterative loop. |

---

## Stopping check

`STOPPING_RULE.md` says stop when (a) no scenario has more than 2 central-registry edits AND scenario 2 costs ≤200 lines, OR (b) two consecutive iterations achieve <20% improvement on the same metric.

Iteration 1 achieved 15% improvement on scen2; iteration 2 predicts 25%. Neither stop condition triggers. Iteration 3 checkpoint: if iteration 2 lands its predictions, the next iteration's threshold for "diminishing returns" is whether iteration 3 can meaningfully improve on what's left after API + CLI + UI substrates are in place.

---

## Handoff

Iteration 2's executing agent reads, in order:

1. `BASELINE.md` (Phase B numbers, frozen)
2. `RESULTS.md` (iter-1 post-substrate column)
3. `HYPOTHESIS.md` (the original + recalibration; the recalibration shape is the model for iteration 2's hypothesis)
4. This file (the iteration-2 proposal)
5. `docs/agent-system/REFERENCE_PATTERN_FITNESS.md` (the lens canon — iter-1's audit findings still drive iter-2's targeting)

Then they author `iteration-2-plan.md` mirroring iteration 1's plan structure. The proposal above gives them the hypothesis, scope, and predictions; the plan layers in skill-canon references, definition-of-done checklist, and end-to-end gate.
