# E4 — Semantic and AI-Powered Control

## Outcome
Make device control faster and more reliable by choosing available controls with evidence, not by guessing from screen images or blindly replaying actions.

## Work
- Trace AI control invocation through AI Gateway, prompt/action schema, decision model setup, returned action/probabilities, validation, history, retry and fallback. Reproduce the reported basic wrong choices and retain failed traces safely.
- Audit accessibility/semantic screen representation: completeness, stable identifiers, focus/selected/enabled state, bounds/action mapping, truncation, privacy and unsupported surfaces. Expose through shared Device Control tools and use it as model context where reliable.
- Compare decision-model vs LLM paths using equivalent TV/phone tasks, exact outcome assertions, action count, latency, cost/tokens, confidence calibration and fallback. Tune confusing/contradictory prompts and schemas using failure evidence.
- Promote successful repeated behavior to reusable flows only with preconditions, semantic target resolution and terminal outcome assertions; test variable initial focus/page state. Keep screenshots as fallback.

## Proof
Saved redacted traces and regression cases; measurable baseline/comparison; repeated task success on supported devices; scoped cost/latency evidence. No quality regression accepted in exchange for speed.

## Current implementation evidence (2026-11-02)

- Agent run records now measure end-to-end elapsed milliseconds, Gateway planner calls, executed action attempts/completions, and each Gateway response's provider/model and usage. Token/cost values preserve Gateway provenance; absent usage remains absent. Local test planners do not synthesize provider measurements.
- The Gateway planner still receives typed state and a complete semantic snapshot where available; it intentionally receives no screenshot pixels. This keeps screen content out of inference attachments, but leaves image-only surfaces and visual fallback unhandled. Current screenshots are not a substitute for image input to the planner.
- Regression coverage exercises action counts and Gateway-provided usage and provenance, including absent usage. `go test ./internal/flows ./internal/control` passed. Fresh Test Genie unit run `20261102-181835-f8957e35` passes at L2 with 34 advisory findings; Device Control docs run `20261102-181920-f5af9f8a` passes at L1.
- Remaining: compare model choice against deterministic semantic/flow route on matched assertion-bearing tasks; measure route latency, success rate and cost across repeated pairs; decide whether a privacy-qualified local-only visual input path improves outcomes without sending screen content outside its approved boundary.

## Follow-up — app lifecycle planner routing (2026-11-02)

- A live agent task on Moto G7 `android-6affcac615a281a2` asked it to open the installed Calculator package and verify foreground. Dry-run proposed `launch`. The live run made one planner call and one action attempt, then failed before actuation with `proposed=launch ... unsupported capability "media"`; the run reported no verified goal. This exposed that app-lifecycle planner output was being passed through `DirectActuation.Action`, which Device Control interprets as a media command.
- The deterministic owner CLI fallback launched `com.google.android.calculator` over wireless; Device Control observed the exact resumed activity and focused Calculator window. It is one successful fallback sample, not a repeated comparison cohort.
- Agent app-lifecycle steps now execute through Device Control's typed flow owner. Planner output must provide the application identity, and launch succeeds only when the exact foreground package is observed after the flow. Regressions cover typed launch routing, identity validation and foreground verification. Focused `go test ./internal/control ./internal/flows` passes.
- The phone was returned to Motorola Home, then a single delayed state read at 19:51 UTC confirmed screen-off/locked. No further physical action or owner-consent prompt was issued.
- Still open: rerun the live AI task on a current on/unlocked session, complete repeated matched agent/fallback pairs with exact assertions, latency/cost/action counts, and extend the comparison to TV where an identity-bearing observation route exists.

## Follow-up — planner action contract and Calculator comparison (2026-11-02)

- A fresh physical Moto G7 agent run completed one `launch` over wireless but then proposed `tap`, which the serving agent path incorrectly routed through direct media actuation and failed. The trace has 2 planner calls, 2 action attempts, 1 completion; Gateway usage reported 2,454+4,677 input tokens and 33+32 output tokens, while cost was unavailable.
- The source now advertises only step kinds with implemented owner routes. Gesture/text choices are omitted because the planner receives no screen pixels or coordinates and those direct executor mappings are absent. Deterministic fallback still permits `observe`; Gateway planner does not. Risk classification precedes capability validation so risky unsupported plans retain confirmation requirements. `TestAgentPlannerOnlyReceivesExecutableStepKinds` and related regressions pass; full `go test ./internal/control` passes and Test Genie unit `20261102-205207-ef244368` passes L2.
- A deterministic USB flow on the same device passed `launch` plus exact `assert-foreground` for `com.google.android.calculator` at 20:54:30–20:54:31Z. The deterministic flow ran after the agent had already launched Calculator and left it foreground, so this is a sequential fallback sample with a different starting state, not a matched comparison or repeated benchmark.
- Managed restart refused because the lifecycle provider reported incomplete inventory; the currently serving process remains healthy on its old build. Therefore the source fix has not been live verified. The phone was restored to Home/off/locked and no consent prompt was opened. See `LOG.md` for exact run IDs, owner audit receipts, unit artifacts, and the Memory work record.
- Remaining: safe runtime replacement, a post-rebuild live trial, repeated matched outcome/action/latency/cost pairs, and broader semantic/TV coverage.
- Final malformed-plan ordering regression also passes: contradictory `goal_met` payloads are rejected before confirmation classification, while valid unsupported risky requests remain confirmation-gated. Test Genie unit `20261102-205713-9d39b18b` passes L2; exact artifact hashes are in `LOG.md`. The final claims review corrected the chronology to a pre-source-change agent attempt and a sequential, non-matched fallback sample.

## Follow-up — live Chrome playback semantics read (2026-11-02 UTC)

- On Moto G7 `android-6affcac615a281a2`, a read-only Device Control semantic snapshot at `21:49:47.953340473Z` over wireless returned 30 elements with `truncated=false` while Chrome displayed the paused Sintel player. The accessible tree exposed enabled `play`, `unmute`, `enter full screen`, and seekbar controls, alongside Chrome toolbar elements.
- This verifies semantic availability for this one Android Chrome media surface and state. It is not an agent plan or completed semantic action; no task assertion, model latency/cost, confidence, or deterministic comparison was measured. The screen was not sent to AI Gateway, no audio consent was requested, and no private browser text was copied into this record.
- Next: use a stable assertion-bearing task with the Gateway route and deterministic semantic/flow route from equivalent initial states; first resolve the serving planner's `ai_gateway_client_not_configured` state through its owner and safe deployment boundary. Preserve the local-only semantic privacy policy.
- Memory work record: `8275fddb-7366-4878-a783-048147ba811d`.

## Follow-up — Gateway wiring and live-boundary diagnosis (2026-11-02 UTC)

- Read-only source tracing confirms the current API composition root constructs `Gateway` with `discovery.ResolveScenarioURLDefault` and installs `GatewayPlanner` on the control service. The `ai_gateway_client_not_configured` result is returned only when that Gateway or its URL resolver is nil. This source fact does not establish what the currently serving process executed.
- The live Device Control process still reports build identity `sha256:9b04c50800de7bb04a1c3c3f3f695b1951d12a931dd56cd86f148b880ce0c80b`, the pre-change build recorded in earlier E4 evidence. Lifecycle replacement was refused because the owner inventory remains incomplete. Therefore the old live failure cannot diagnose current source wiring, and current source wiring cannot be claimed as deployed.
- Do not inspect or print AI Gateway CLI credentials to investigate this boundary. Next E4 evidence requires a lifecycle-owner-approved current build and then a live assertion-bearing run. Repeated matched Gateway/fallback pairs remain pending; no device state was changed in this diagnosis.
- Memory work record: `b76531ee-8908-40c4-8c2c-8549f1fc2fc0`.
