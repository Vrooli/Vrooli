# E2 — Android Lifecycle, Wireless, and Verified Artifact Boundaries

## Outcome
Make supported TV/phone lifecycle and connectivity operations consistent, identity-safe, observable, and repeatable; provide a trusted immutable artifact install route through the owning artifact producer.

## Work
- Clarify and unify wake/sleep, keyguard/auth readiness and Home/launcher navigation while preserving distinct state assertions and capability support.
- Implement/test wireless debugging enable/disable as reusable flow steps only on compatible Android versions. Track OS support and prompt requirements explicitly. Run repeated controlled reachability samples before correlating state with Wi-Fi loss.
- Reconcile USB and network ADB without stale endpoint identity promotion; report transport state and transient failure distinctly.
- Add artifact owner API/resolver as needed to return immutable ref, checksum, package/version metadata and bytes; install with Device Control lease and verify live installed package/version. Remove trusted-path dependence on caller arbitrary APK paths without breaking explicit developer workflows.

## Proof
API/CLI/flow unit regressions for support, unknown, timeout, identity mismatch and recovery; governed producer/consumer tests; live install only via verified artifact and authorized device. No raw ADB shortcut or claim of compatibility without device/OS evidence.

## Current evidence and open work (2026-11-02)
- Producer-side exact-ref resolution, retained APK/AAB rehashing and path-free metadata/APK endpoints are implemented in `scenario-to-android`; focused Go packages pass and Test Genie unit run `20261102-171444-db13b76f` is terminal PASS at L2 with advisories.
- The manifest and route expose artifact bytes through the existing plain JSON operator API without route authentication. The API binds loopback and rejects requests tagged by the network-facing UI proxy; runtime checks returned 403 from both UI-proxied routes and 404 for invalid direct-API refs, confirming the intended local-only exposure boundary.
- Device Control now has a bounded producer-ref client and `install-artifact` flow. It verifies the exact immutable ref, returned descriptor identity and APK digest/size, stages bytes without accepting a caller path, confirms and revalidates the active lease before install, then checks installed package/version state and writes structured audit proof. Redirects are disabled. Focused API/control/handler Go tests pass; Test Genie unit run `20261102-173240-8e0ece12` is terminal PASS at L2 with 34 low-coverage advisories. CLI domain tests pass and primitive evidence was refreshed through its owner test.
- Source review established that the endpoints have no route authentication. Runtime now confirms the listener is pinned to `127.0.0.1:17827` and the UI proxy rejects metadata/APK requests. Physical install was subsequently verified on the Moto G7; no projection consent was requested.

## Follow-up — bind APK identity before actuation (2026-11-02)

- Two fresh hostile reviews independently found that producer `artifact.json` identity fields were not bound to the APK bytes. Device Control could discover a package/version mismatch only after installation.
- Device Control now inspects the staged APK's compiled manifest through the Android SDK `aapt` executable resolved beside its `adb` SDK. Package name, version name and version code must exactly match the descriptor before the staged path can be returned to the lease-owned installer. Inspector failure or mismatch fails closed before ADB actuation.
- Regression coverage rejects a mismatched embedded package, version name or version code. A focused Test Genie Device Control unit run `20261102-182546-4bd8a8c9` passed terminal at L2 (34 advisory low-coverage findings). A real retained companion APK was independently inspected with build-tools `aapt dump badging`, which reported `com.vrooli.devicecontrol.capture`, version `1.0.0`, code `1`.
- The prior live install predates this pre-install manifest gate and therefore does not prove that the newly added gate was exercised by a physical install. Reinstalling the same APK is not required for this code proof; the owner-authored regression covers the pre-actuation rejection path.

## Follow-up — owner boundary hardening (2026-11-02)

- Changed scenario-to-android API startup to set `BindAddress: "127.0.0.1"`, preventing an inherited `API_BIND_ADDRESS` value from exposing its operator routes. The immutable artifact endpoints remain unauthenticated and are explicitly local-only. Owner API tests and a managed runtime listener check are pending.

## Follow-up — current identity and read-only inspection (2026-11-02 UTC)

- Device Control inventory at `22:51:19Z` reverified Moto G7 stable ID `android-6affcac615a281a2`, serial `ZY226TG2BM`, Android 10, wireless endpoint `192.168.1.179:5555`; native Wireless Debugging is unsupported and legacy TCP 5555 is enabled. Live state at `22:51Z` reported screen off/locked and Chrome foreground. Owner-supported read-only inspect at `22:52:50Z` over wireless returned Home component `com.motorola.launcher3/com.android.launcher3.CustomizationPanelLauncher`, WebView `156.0.8078.47`, memory/storage/ABI facts; video encoder inventory and low-RAM property remain unavailable. No device write was issued.
- Galaxy A03s stable ID `android-024665203bca17fa`, serial `R9TT608Q6MH` remained listed available over wireless at `22:51Z`, but its state call failed because all wireless and USB state probes were unavailable. Owner-supported `inspect --transport wireless` also failed to establish any hardware or Home facts. Android version and native Wireless Debugging compatibility therefore remain unknown. No pairing or setting change was attempted.
- These reads refresh identity and current support only. They do not establish wireless reliability, Wi-Fi sleep causality, or A03s OS compatibility.

## Follow-up — proxy boundary and runtime gate (2026-11-02)

- Added explicit loopback binding to scenario-to-android API startup. Because the UI service binds network interfaces and proxies `/api`, its proxy injects `X-Vrooli-Scenario-UI-Proxy: scenario-to-android`; both immutable artifact handlers reject that marker. Device Control's direct API client uses the local producer endpoint and sends no marker. Producer handler tests and Test Genie unit run `20261102-174053-3232da30` pass.
- Managed start was attempted through `make start` and `vrooli scenario wait`, not by invoking the binary. It did not replace the serving process. The lifecycle dependency graph first exposed missing Go replacements in `vrooli-bridge`, `browser-automation-studio`, and `secrets-manager`; those owner module repairs were applied through Scenario Dependency Analyzer. The next attempt reached the declared `android-sdk` `must_start` gate and failed because resource state is disabled; `vrooli resource status android-sdk --json` also reports `healthy=false` because `avdmanager --version` fails. No resource enable/install action was taken.
- The artifact route has source and handler-test evidence but has not yet been validated through the updated running UI proxy. Existing scenario instance remains the prior build. No artifact was installed and no device was changed.


## Follow-up — resource health, runtime boundary and live install (2026-11-02)

- Fixed the Android SDK owner's incorrect `avdmanager --version` liveness command. `avdmanager list device` is supported and completed in 0.5 seconds. `vrooli resource validate android-sdk` passes; `vrooli resource status android-sdk --json` now reports healthy. Enabled it through `vrooli resource enable android-sdk` under the goal's infrastructure authorization. Scenario-to-Android `make start` completed healthy with API 17827/UI 24699.
- Runtime boundary proof: GET to both `http://127.0.0.1:24699/api/v1/android/artifacts/{metadata,apk}?immutable_ref=probe` returned 403 through the network-facing UI proxy. Direct API requests returned 404 for the intentionally invalid ref, proving they reached artifact resolution. `ss` showed API listening only on `127.0.0.1:17827`.
- `install-artifact` initially failed preflight because Android strategy `StepKinds` omitted the implemented Android-specific operation. Added the step only for `android-adb` declarations and regression coverage that non-Android lifecycle strategies do not advertise it. `go test ./strategy/... ./internal/control` passed. The initial failure happened before lease acquisition or device mutation.
- After a managed Device Control `make start`, fresh inventory verified Moto G7 Android 10 serial `ZY226TG2BM`, stable ID `android-6affcac615a281a2`, available with app-lifecycle. A confirmed flow installed immutable ref `android-debug:a4571098c6ec6ce4a12739faa287a2070367ee4761ee5304b3a88bd325852571`, package `com.vrooli.devicecontrol.capture`, version `1.0.0`. Flow `1d5c512e-26e6-46cb-bc58-2ee139df3eeb` passed with artifact/APK SHA-256, install accepted and post-install package/version verified. The app was not launched; no consent prompt occurred.
- Infrastructure repair required Scenario Dependency Analyzer reconciliation of the dirty Scenario Authenticator API module's missing in-repo replaces. Its prior source/version changes were preserved. First Device Control managed restart failed before stopping any process due this dependency; the second completed after repair.
- Remaining: run the fresh Test Genie unit phase for the `StepKinds` repair, capture a user-readable run receipt, repeat wireless recovery evidence, and keep permission/audibility/A-V-sync claims pending owner presence.

## Follow-up — live pre-install manifest proof (2026-11-02)

- After the manifest-verification change, `make start` rebuilt/restarted Device Control through its managed lifecycle. Fresh inventory/state again identified Moto G7 stable ID `android-6affcac615a281a2`, serial `ZY226TG2BM`; the device was available and app-lifecycle was available. Screen was off/locked; the operation adapter selected USB.
- Reinstalled the exact immutable capture artifact through a confirmed Device Control `install-artifact` flow. Audit `62f101f5-66d9-4667-8fdb-4aa026659cc7` at 18:32 UTC records the artifact and APK SHA-256, package `com.vrooli.devicecontrol.capture`, version `1.0.0`/code `1`, install acceptance and post-install verification. The flow passed. The app was not launched, and no projection/audio consent prompt was requested.

## Follow-up — repeated Moto wireless reconnect and A03s capability read (2026-11-02 UTC)

- Fresh Device Control inventory at `22:11:07Z` verified Moto G7 `android-6affcac615a281a2`, serial `ZY226TG2BM`, over USB and wireless endpoint `192.168.1.179:5555`. A USB state read reported screen off/locked, Chrome foreground, Android 10 legacy TCP enabled on 5555, and native wireless debugging unsupported.
- Three owner-mediated `device reconnect` operations for that exact stable ID succeeded with serial verification at `22:11:34.168679074Z`, `22:12:00.295346357Z`, and `22:12:24.656252211Z`. The intermediate and final reads continued to report off/locked; final inventory at `22:12:33.8256248Z` found both USB and wireless available on the same identity and endpoint. No phone Wi-Fi outage or screen transition was induced. This is one short, repeated recovery window, not a reliability cohort or evidence for/against sleep causality.
- A fresh list at `22:12:58.136960975Z` verified Galaxy A03s `android-024665203bca17fa` / `R9TT608Q6MH` as available on USB and wireless, but Device Control state failed on both transports and USB hardware/Home inspection returned `failed_precondition`. Current OS/version and native Wireless Debugging support are unknown; no pairing or enable flow was attempted.
- Memory work records: `e0907635-c1fc-4901-b07b-9d0ad0202477` (Moto reconnect window), `06c4afd9-a588-4aa8-9646-e23f97fe46aa` (corrected A03s capability record; the preceding note contains malformed shell examples and should not be used).
- Follow-up diagnosis found prior shipped owner work `rec-65a7aa6ebb386b1b` for endpoint-bound Android inventory/reconnect/state, keyguard parsing, effective transport selection, and held-lease restoration. Current live Device Control still reports build identity `sha256:9b04c50800de7bb04a1c3c3f3f695b1951d12a931dd56cd86f148b880ce0c80b`; the source tree has changes in `service.go` and `androidadb.go`. The A03s is absent from retained stale-device diagnostics, while `device describe` shows a USB capability observation from 21:21:14Z, older than the 22:12:58Z list. This suggests a runtime/source freshness gap but does not establish its cause. Work record `da22b5dc-3914-406b-871d-e0742acca7fc`; no restart attempted because E6 still has the desktop owner/helper unknown and replacement must fail closed.
- Next: pursue owner-supported A03s state discovery only when its transport probes recover; use a compatible device for a native Wireless Debugging flow when exact OS and identity proof are available. Continue reliability sampling without inducing unmeasured Wi-Fi outages or changing screens to manufacture causal evidence.

## Follow-up — fresh A03s inventory and state probe (2026-11-02 UTC)

- Device Control inventory at `22:23:10.491861301Z` now reports Galaxy A03s `android-024665203bca17fa`, serial `R9TT608Q6MH`, Android 13, with wireless available. Its wireless profile at `22:23:03.702184221Z` still reports screenshot, semantic tree, app lifecycle, and wireless-debugging capability unavailable; USB capability data is older (`21:21:14.647076Z`).
- Fresh owner `device state` still failed across wireless and USB. Exact-transport USB and wireless inspections each returned `failed_precondition` because no hardware or Home facts could be established. No pairing or setting change was attempted. Android 13 makes native Wireless Debugging a candidate to investigate, but the owner profile does not verify capability or current setting; the phone owner is away.
- Memory work record `ef446f0e-5957-4a0e-808a-6c31bbeee1d3`. Next: diagnose the owner probe/profile freshness gap through the owning Device Control source and runtime path. Pairing still requires an owner-present, consented operation after the capability and setting preconditions are verified.


## Follow-up — exact A03s reconnect attempt and endpoint trace (2026-11-02 UTC)

- A fresh inventory at `22:28:22.780859116Z` still identifies A03s `android-024665203bca17fa` / `R9TT608Q6MH` as Android 13, but its current endpoint is empty. The separate Moto profile remains `android-6affcac615a281a2` / `ZY226TG2BM` at `192.168.1.179:5555`.
- Exact-ID owner reconnect for A03s returned `wireless_reconnect_failed`, connection refused at `192.168.1.179:34483`. Source trace shows reconnect can restore a saved per-device endpoint even when fresh inventory has no endpoint; the adapter verifies discovered candidates against the onboarded serial before accepting them. This refusal is consistent with stale A03s endpoint state, but does not establish the cause or a cross-device endpoint defect. Do not infer collision or substitute the Moto endpoint.
- State/inspect still fail, pairing was not attempted, and phone owner is away. No further reconnect or device mutation is planned without a better owner-supported recovery path. Memory work record `dbae07e1-c0f5-4151-bc21-9292e793204e`.

## Follow-up — Bridge inventory versus local state route (2026-11-02 UTC)

- At 23:23Z, fresh `device describe` reports A03s `android-024665203bca17fa` / `R9TT608Q6MH` with host node `697b6224-6283-4a31-90e2-73724e424c05`, `available`, and reason “wireless ADB endpoint reachable and identity verified”; its endpoint field is empty. Both USB and wireless state reads fail, and read-only inspect cannot establish hardware/Home facts. This is a cross-surface discrepancy, not proof the phone itself is reachable through a local ADB endpoint.
- Source trace: Bridge attached inventory merges trust/reachability into Device Control's physical-device record (`control/inventory.go`), while `ReadDeviceState` iterates only locally registered/endpoint-bound strategies (`control/service.go`). There is no attached-device state-reader route in this contract. Bridge reachability can therefore coexist with unavailable local state. The final promoted-device inventory pass also rewrites an available health reason to the local wireless ADB wording; this wording should not be treated as endpoint evidence for a Bridge-only attachment.
- No source change was made because adding remote state access requires an owner-defined Bridge contract and authorization boundary; an inventory fact alone does not establish that such reads are supported. Live API build identity remains the prior serving build, so this observation describes current behavior but does not certify the current source snapshot. Do not reconnect, pair, or mutate A03s while the phone owner is away.

## Follow-up — current Moto readiness and owned screenshot (2026-11-02 UTC)

- Fresh inventory at 23:26Z verified Moto G7 stable ID `android-6affcac615a281a2`, serial `ZY226TG2BM`, available over USB and wireless (`192.168.1.179:5555`). State was screen off/locked, Android 10, native Wireless Debugging unsupported, legacy TCP enabled on 5555.
- Device Control `device screen --operation ready --home` using configured profile `auth-1a7ba66d-4776-441a-bffd-18ff8eb93b4e` returned `verified` over USB. The immediate read showed screen on/unlocked and Motorola launcher foreground; exact-USB inspect then confirmed Home component `com.motorola.launcher3/com.android.launcher3.CustomizationPanelLauncher`. No app launch was requested.
- Held-lease `device observe` flow `81c709dd-5ba4-4d83-8cd4-7d654e745ac8` passed over USB. Image evidence `8b9997ce-cb3b-425d-9d28-0467f8efbab0`: 3,262,811 bytes, SHA-256 `584548e51d5387008e07aef57c101e5f9032a2b6bd9c33a93d2e817c310b2d83`; producer marks content unclassified and unverified. Lease `132a8c37-e206-443e-a820-e8e209f744ff` was released. This is a fresh readiness/capture receipt, not a visual frame-QA result or paired performance sample. No projection/audio consent was opened.
