# Archived Device Control maturity evidence — lifecycle owner history

Moved verbatim from `LOG.md` when the active evidence log reached its size limit. The current lifecycle boundary and open conditions remain in E6 and `STATUS.md`.

## 2026-11-03 — fresh current identity and hostile review follow-up

- Device Control inventory at 03:32:06–09Z listed Moto G7 `android-6affcac615a281a2` / `ZY226TG2BM`, A03s `android-024665203bca17fa` / `R9TT608Q6MH`, SmartTV Cast `google-cast:3bc013548481e743aa769f04a9a9ba0b`, and desktop available. Moto was off/locked at Motorola Home; A03s state failed on USB/wireless; TV mute sources conflict and input/power are unavailable. Desktop screenshot is available, input is not admitted. No writes, wake, or consent. E2.
- Managed Device Control restart remains pre-stop refused (4 blockers, 9 work, 0 managed components) on old build `sha256:9b04c50800de7bb04a1c3c3f3f695b1951d12a931dd56cd86f148b880ce0c80b`.
- Two fresh hostile reviews found no material defect in the scoped Portal response-policy/API/test slice or E3 claims/E6 lifecycle-fence slice. They do not qualify the outstanding TV, full E3, A03s, or safe legacy-replacement requirements. E6.
- Rechecked E3 P2-off run `3f18e84f-93de-422b-bf67-ab8bf0077e39`: process-local flow inventory lists it, retained inventory/get omit it, and owner retain returns `live_run_not_found`. Current source requires result, flow and device indexes; this does not prove which index is missing in the old serving instance. No runtime storage changed; E3 pass/timing is not independently recoverable through the current API.

## 2026-11-02 — control-plane desktop-helper process exit trace

- Fresh read-only worker `/root/helper_exit_owner_trace` traced `service.json`'s `desktop-helper` sidecar through `internal/lifecycle/phases.go` and `teardown.go`: the control plane writes canonical `process.Record` identities, then owns signal/wait and can observe exit for those records during its own teardown. `internal/process.ReadScenarioRecords`/`LiveRecords` are private control-plane owner functions; there is no API-facing observation contract for Device Control.
- Device Control's helper lifecycle inventory intentionally keeps process exit unknown (`api/main.go`) and the existing regression proves receipt completion does not prove helper exit (`api/main_test.go`). The safe missing seam is a read-only supervisor-backed component observation keyed by scenario/component generation; unknown, corrupt or stale identity must remain blocking. Do not use PID-only probing or scenario-side process inspection.
- A fresh implementation worker `/root/lifecycle_component_exit_integration` is now assigned the whole control-plane/Device Control source integration and regressions. No lifecycle commands, tests, service operations, device actions or credentials were used by the investigator.
- Scope was expanded after the second source check: the canonical supervisor also has PID-only liveness and no retained OS-incarnation/wait receipt, while provider standing unconditionally adds an unknown owner and validates completeness before Drain. Worker is implementing the control-plane-owned pre-replacement stop/wait protocol, including generation binding; observer-only inventory would not meet the safety outcome. No live restart or service state mutation is authorized in this slice.

## 2026-11-02 — exact-generation helper stop/wait source integration

- **Changed:** Standing separates complete helper-generation enumeration from unresolved exit. Under a retained fence, Device Control closes ordinary admission and joins owner RPCs; control plane validates sidecar/launch identity and generation, pins, signals and waits the exact process, then submits matching completion. Emergency override cannot bypass exit proof. Successful wait publishes `exited_verified` before retiring the exact runtime ref. Tombstones never create proof. Recovery requires the same provider and fence.
- **Evidence:** two fresh post-fix hostile reviews found no material defect. Focused `internal/process` and `internal/lifecycle` regressions passed (including PID reuse, emergency, cancellation, failure ordering and exact-fence recovery); affected Device Control API packages passed; full process package passed; Proto `make verify-committed-gen` passed. Test Genie unit `20261102-231451-df2159fa` passed L2. Final docs run `20261102-231941-1170b2dd` passed L1 after removing absolute evidence paths; remaining content-quality advisories are existing debt. Requirements sync changed zero files/statuses; Business Health and requirements validation passed; goal-home lint passed at 27,297/30,000 bytes; scoped diff check passed. Memory work record `ce352657-735a-4f48-897a-6f5908dc7855` records the source outcome and limits.
- **Other limits:** shared dependency evidence and unrelated generated-output attribution remain open.
- **Remaining:** no replacement/deployment; live behavior is unverified and the old API/helper remain fenced. Lost coordinator context after signal leaves the fence closed; a durable supervisor proof flow needs owner design. Device Control/Portal still lacks live deployment, A03s and TV qualification, trusted HTTPS, matched performance/AI evidence, and consented audio.

## 2026-11-02 — lifecycle fixture repair

- Fixture persistence: seven regressions and lifecycle passed; refusal unchanged. `f0a9ce4f-b916-48e9-9446-bfef6b2a3e66`.
- Docs `20261102-235127-164f12e1` passed L1; validators clean; lint 28,196 bytes.



## Archived active-log receipt

The following prior concise lifecycle/device readback entry was moved verbatim to preserve the active log size bound.

## 2026-11-03 UTC — lifecycle/device readback

- Restart refused pre-stop; A03s state and TV readback remain limited. No writes/consent. Details in E2/E5/E6.


## Archived superseded run receipts

The following terminal documentation and Moto frame receipts remain referenced by STATUS and E2, but were moved out of the active log to retain its read cap.

## 2026-11-03 — current docs gate after live evidence/reviews

- Device Control docs Test Genie run `20261103-033231-dcfcab99` terminal PASS L1, 476 warnings/24 infos. Documentation contract, append-log integrity, links and manifest pass; content-quality findings remain. Goal-home lint passed at 28,356/30,000 bytes before this final status receipt update.


## 2026-11-03 — verified Moto readiness/capture

- Moto `android-6affcac615a281a2` ready/Home then sleep verified over USB; screenshot `bfb89cbd-fb7f-4043-9e90-990733b0779d`, 1080×2270, SHA-256 `4e296e2229ae7c7e5d32d21b51cbeb6ac9029eef21ae1841ecc834dae287a404`; no consent. E2.
