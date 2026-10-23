# E1 return journey handoff

## Result

This worker did not operate an answer. The current source supports the queue/detail/answer/readback/return behavior, but the only registered disposable-recipient browser case is a read-only observer. Its fixture has no auth material, and there is no current E1 lease. G2 remains unmet. No benchmark gate is accepted. The safe bounded fallback now has an exact owner refusal observed in a local handler integration test; it does not close G2.

## Current source and served build

- Source revision: `52097258290d1bff0191319242d4948e214af9e3`; Test Genie tree digest: `td:5b66775c4b85317c9219095a5b11fd6cb1c88773da9dbe0a2796fff063044a5b`.
- Served Notification Hub build: `sha256:2e49c6cb66e64181488c61a607843773d90a4c49e2807354973464e92b52cb4f`, healthy after managed start operation `startop-be4f5273d792ee9f05ef9a9a47927513`.
- Exact inspected source hashes and the prewrite comparison are in `source-build-census.json` and `inventory.json`; original snapshot manifest is `prewrite-original/manifest.json`.

## G1 — current governed UI-health

- Test Genie run `20261020-173747-5ea8b428` completed `failed`, one phase (`ui-health`), 4 errors, 32 warnings, 9 infos. Its exact target revision, tree digest and phase result are retained in `ui-health.json`.
- Owner artifacts: command log `artifact_2a07635e56858777fb00c3a03c749f48`; findings report `artifact_4d28af4950fad0c6e9334e63c702e15c`; phase result `artifact_63d666ee85494bcaa851fa26c39882f6`. No evidence-manifest artifact was cataloged.
- Findings include four `visual_offscreen_interactive` errors on `a[data-testid="layout-shell-skip"]` with rect `(8,-56 117x32)`, two `visual_native_screen_unverified` capture limitations, and two `visual_safe_area_not_exercised` limitations on `/settings` and `/updates`. The runtime rendered, but this does not pass focus or visual acceptance.

## G2 — exact auth refusal and return condition

The focused handler receipt is `auth-refusal-check.json`. Its first local attempt caught a test ordering issue (the fixture recipient had already been cleared); after moving the empty-recipient case after the refusal postcondition, the focused test passed. Both prewrite checkpoints and hashes are retained in the inventory.

The fixture subject is `e1-fixture@notification-hub.invalid`; the seeded ask is `ask-e1-disposable`. BAS case `bas/cases/e3/disposable-recipient-isolated.json` is explicitly `execution_mode: observer`; its exact prior bytes/hash and receipt show only `/updates` navigation and a text assertion. It submits no identity and no answer. The Answer owner requires a verified owner identity; its exact missing-identity response is Connect `Unauthenticated`, `verified owner identity is required` (`api/handlers/conversations/module.go`). The exact response was observed through the existing ConversationsService handler in a safe `httptest` Connect integration test backed by `databasetest.NewSQLite(t)`. The test asserts that the unauthenticated request returns this refusal and the ask remains pending with no answer. This is not the managed app or routed testpool journey; no managed E1 Answer request was attempted.

The accepted E3 lease `2050fb04-3884-46d5-a71c-8689c92550eb` and its counters are historical observer evidence, not an active E1 lease. A future G2 run needs a legitimate existing auth binding for the disposable subject and a new exact active lease with `routed_isolation=true`, `TestPoolRequests > 0`, zero primary requests, and both file-root write counters. Do not use a real owner identity, issue a grant, create credentials, or replay the E3 lease.

## Source, state, and test mapping

- `AsksPage.tsx`: bounded open-ask list, URL filter, deep link, stable row identity, saved main scroll, and focus restoration after return. `AsksPage.test.tsx` covers the route/filter/return behavior.
- `AskPage.tsx` and `askDrafts.ts`: exact options remain drafts until deliberate submit; mutation retries are disabled; one immutable attempt is read back after uncertain outcome; safe note/answer draft is keyed by subject and ask; recorded result and explicit Back preserve route state. `AskPage.test.tsx` covers answer, conflict, expiry/session and readback states.
- `api/handlers/conversations/module.go` + `api/internal/hub/asks.go`: verified owner at Answer boundary, recipient-scoped atomic answer, authoritative ask readback. `api/internal/hub/asks_test.go` protects service answer/default/late-answer invariants.
- Applicable source/design mapping: `NOTIFICA-P1-009`/`OT-P1-009` in `evidence/commissioning-source-map.md` maps to NH14.2–NH14.10 and NH17.1–NH17.6, with NH-AC-20–25 and 37. Queue/detail/result states map to selected NH01 (Needs you), NH02 (prepare answer), NH03 (recorded result), and NH14 (submission/uncertainty) images. NH13.1/NH13.3 cover sign-in and session expiry/draft boundary. Selected originals are enumerated and hashed in `selected-image-manifest.json`; this worker made no visual comparison.
- Adaptations currently visible in source: `/` is the Needs you queue while `/asks` remains a deep-link alias; answer draft is in-memory per subject/request; selection and navigation do not call Answer; current route return behavior restores the selected row and scroll. These are source observations, not operated-build proof.

## Consolidation and inventory

No deletion was safe in the inspected paths. `AsksQueue` and `AskDetail` own different queue/answer states; `askDrafts` is the only subject/request draft store; answer outcome/readback is a distinct mutation boundary; the E3 observer case protects a different route/counter seam. Removing any would drop unique protection or the user journey. No product runtime source changed. One existing API handler test gained four assertions/check lines (net test delta +5 lines after formatting/order preservation); the serving build is unchanged. The exact path comparison records the new evidence, slice-log append, generated files and unknown attribution.

Test Genie performed a governed Proto freshness/materialization preflight; the shared `packages/proto` output was already dirty in the checkout. Its current generated status has 45 paths, but no exact prewrite byte snapshot exists for those out-of-scope paths, so per-path attribution is unknown. They were not restored or hand-edited. The UI selector/string outputs captured in this worker baseline did not change. Test Genie rebuilt its CLI during its owner wait; those CLI writes are outside the admitted Notification Hub product scope.

## Remaining and unverified

- G2 operated answer, authoritative runtime readback, and queue-return focus/scroll continuity: unmet pending existing auth owner fixture identity and a fresh lease.
- G1 is a current completed observation but failed; focus finding and visual capture limitations remain open.
- Selected-build desktop/phone light/dark visual comparison, independent visual acceptance, and separate parent code/architecture review are unrun.
- No managed E1 answer case bytes, active E1 lease, answer counters, or post-answer managed DB state exist because no fixture-authenticated mutating case was authorized or run. The isolated handler test confirms the exact no-identity refusal and unchanged local ask state.
- E2 and the complete selected redesign remain outside this slice and open. Original benchmark clock remains unchanged.
