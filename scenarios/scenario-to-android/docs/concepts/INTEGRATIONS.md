# Integrations — Scenario to Android

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `packages/delivery-ramp-go` | shared Go module | yes | targets, builds, journeys, releases, distribution | Exported `Prober`, `Builder`, `Driver`, `Distributor` interfaces | Compile-time. A ramp that reaches into spine internals is a design error, not a runtime failure. |
| `device-control` | scenario | yes | journeys, targets | Connect-RPC: device inventory, lease acquire/release, device verbs | Journey cell reports `unavailable` naming device-control as the missing capability. Never falls back to driving the device directly. |
| `vrooli-bridge` | scenario | yes | targets, journeys | Node registry, allowlisted dispatch, durable runs | Remote targets report `unavailable` with the reach reason. Local emulator targets remain usable. |
| `deployment-manager` | scenario | yes | releases | `common.v1.TargetVerdict` with `EvidenceRef` entries | Gate computes locally; verdict emission retries. Evidence is never discarded because the consumer is down. |
| `browser-automation-studio` | scenario | yes | journeys | Named BAS flow executed against the app WebView via CDP over `adb forward` | Web-content chapters report `unavailable`; native shell chapters still run and report honestly. |
| `secrets-manager` | scenario | yes | builds | Signing identity by reference; key material never returned into this scenario | Build reports `unavailable` naming the signing rung. Never generates an ad-hoc local key as a fallback. |
| `android-sdk` | Vrooli resource | yes | targets, builds | SDK, platform-tools, emulator binary, system images | Targets report `unavailable` with the acquisition next action. |
| `hello-mobile` | scenario (fixture) | yes for conformance | journeys | Generated, built, and driven as the ramp's own proof | Conformance proof cannot run; product scenarios are unaffected. |
| Google Play | third-party service | no | distribution, readiness | Play Developer API; AAB upload, track management | Channel reports `unavailable` with the blocking rung. Other channels are unaffected. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `android-sdk` | required, to be created | The Android SDK, platform-tools, emulator, and system images are fully scriptable, so they earn real governed-resource treatment rather than being acquired ad hoc by this scenario. | Resource exists and is declared in `.vrooli/service.json`. |
| JDK 17 | host prerequisite | Required by the Android Gradle Plugin. Present on the development host; probed rather than assumed. | A node lacks it, or AGP raises its floor. |
| `ffmpeg` | host prerequisite | Used by the spine for recording verification. Present on the development host. | A node lacks it. |
| `/dev/kvm` | host capability | Hardware acceleration is the difference between a ~30 second emulator boot and an unusable 3–5 FPS. Video evidence without it is worthless. | Probed per host; absence yields `unavailable`, never a slow pass. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `device-control` | required | Every device verb — install, launch, input, permission, screenshot, recording — belongs to that scenario. This ramp implements no device driving of its own. | Lease held for the full run duration; verbs refused without one. |
| `vrooli-bridge` | required | Owns device and node identity, pairing, trust, scopes, and durable dispatch. Remote targets are reached only through it. | Allowlisted verbs, per-node scopes, durable run identity preserved. |
| `deployment-manager` | required | The evidence consumer and release-gate authority. | Reference-only `common.v1.TargetVerdict`. |
| `browser-automation-studio` | required | Owns web-content automation. A scenario's own flows replay inside the generated app through BAS, not through a mobile-specific rewrite. | Named flow, WebView attach, chapters merged onto the journey timeline. |
| `secrets-manager` | required | Holds signing identity so no key material exists in the repository. | Reference-only; material never returned into this scenario. |
| `hello-mobile` | required (fixture) | The ramp must be provable without depending on any product scenario's correctness. | Generated and driven like any other scenario. |

## Backdrop listing assets

This scenario owns journey capture and the Play submission relationship. For
imagery, it supplies a captured screenshot and a Backdrop Studio surface id to
`ComposeDeviceFrame`; Backdrop Studio returns PNG bytes at the surface's exact
geometry plus the reserved device-frame region. This client never captures
screenshots and never reimplements treatment or surface rules. Publishing stays
owned by the existing listing pipeline.

| Input | Contract |
|---|---|
| `surface_id` | Backdrop Studio surface identifier, such as `play_phone_screenshot`. |
| `screenshot_png` | Screenshot captured by this scenario's journey/device seam. |
| `arrangement` | `device_center`, `caption_above_device`, `caption_below_device`, or `caption_only`. |
| Output | Composed PNG, exact `width`/`height`, and an occlusion region. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Google Play (Developer API) | required for the Play channel | AAB upload, track management, and listing. Gated behind Play Console registration and developer verification. | Service-account credential held in secrets-manager; unavailable until the ladder rungs are satisfied. |
| Google developer verification | required for certified-device install | From 30 September 2026 in Brazil, Indonesia, Singapore, and Thailand, and globally through 2027, apps on certified devices must come from a verified developer — including sideloaded ones. | Identity verification is an owner action; the readiness ladder tracks its state. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `device-control` | Unreachable, or lease refused | Journey cell reports `unavailable` naming the missing capability and next action. Never drives the device directly as a fallback. | journeys integration tests |
| `vrooli-bridge` | Node offline, dispatch rejected | Remote target reports `unavailable` carrying the reach reason; the target is never silently omitted from inventory. | targets integration tests |
| `browser-automation-studio` | WebView attach fails | Web-content chapters report `unavailable`; native chapters still execute and the run reports partial honestly. | journeys integration tests |
| `secrets-manager` | Signing identity absent | Build reports `unavailable` naming the signing rung. Never generates an ad-hoc key. | builds integration tests |
| `android-sdk` | Toolchain absent or unusable | Targets report `unavailable` with the acquisition next action. | targets probe tests |
| `/dev/kvm` | Absent or not writable | Emulator target reports `unavailable`. An unaccelerated run is never substituted, because its video evidence would be unusable. | targets probe tests |
| Google Play API | Auth failure or upload rejection | Play channel reports `unavailable` with the reason; sideload and ADB channels are unaffected. | distribution integration tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
