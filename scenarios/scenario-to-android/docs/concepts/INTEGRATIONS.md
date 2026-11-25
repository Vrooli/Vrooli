# Integrations — Scenario to Android

## Purpose Of This Document

This guide records the Android ramp's declared dependencies, guarded device
contracts, and other integration relationships.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `packages/delivery-ramp-go` | shared Go module | yes | targets, builds, journeys, releases, distribution | Exported `Prober`, `Builder`, `Driver`, `Distributor` interfaces | Compile-time. A ramp that reaches into spine internals is a design error, not a runtime failure. |

Shared resources, scenario peers, and third-party services are listed in their
respective sections below.

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `android-sdk` | required, governed | The resource owns SDK acquisition, binary health checks, API 36+ system-image selection, KVM probing, and AVD lifecycle. | Android SDK packaging or Google Play's target-API floor changes. |
| JDK 17 + Gradle 8.10.2 | governed `android-sdk` toolchain | Installed and exposed by the `android-sdk` resource; the builder consumes the resource-owned wrappers and does not acquire build-tool bytes itself. | The resource cannot download or verify the pinned archive, or AGP raises its floor. |
| `ffmpeg` | host prerequisite | Used by the spine for recording verification. Present on the development host. | A node lacks it. |
| `/dev/kvm` | host capability | Hardware acceleration is the difference between a ~30 second emulator boot and an unusable 3–5 FPS. Video evidence without it is worthless. | Probed per host; absence yields `unavailable`, never a slow pass. |

## Scenario Dependencies

The service manifest declares nine scenario entries. Only
`scenario-authenticator` explicitly sets `enabled: true`; the other entries
omit the `enabled` field, and none sets it to false.

| Scenario | Explicit enabled field | Required | Startup policy | Declared purpose | Degraded behavior |
|---|---|---:|---|---|---|
| `scenario-authenticator` | yes | yes | `must_start` | Authenticates protected scenario-to-android APIs and validates Deployment Manager service tokens. | Protected artifact, target-catalog, and matrix operations fail closed when the authenticator is unavailable. |
| `device-control` | omitted | yes | `must_start` | Every device verb — install, launch, input, permission control, screenshot, screen recording — under a held lease. This ramp implements no device driving of its own. | Journey cells report unavailable naming device-control as the missing capability. The ramp never falls back to driving adb directly, because that would bypass leases and audit. |
| `vrooli-bridge` | omitted | yes | `must_start` | Device and node reach plane: fleet registry, per-node scopes, allowlisted dispatch, and durable runs for remote Android targets. | Remote targets report unavailable carrying the reach reason and are never silently omitted from inventory. The local emulator target remains usable. |
| `deployment-manager` | omitted | yes | `must_start` | Consumes this ramp's reference-only common.v1.TargetVerdict and owns the cross-ramp release gate. | The local gate still computes and evidence is retained; verdict emission retries. Evidence is never discarded because the consumer is down. |
| `browser-automation-studio` | omitted | yes | `must_start` | Executes a scenario's own named web-content flows against the generated app's WebView, so an author writes a flow once and it replays on Android unchanged. | Web-content chapters report unavailable; native shell chapters still execute and the run reports partial honestly rather than passing. |
| `backdrop-studio` | omitted | yes | `must_start` | Provides the typed backdrop composition contract used by the Android journey adapter for scenario-owned web-content flows. | Journey web-content chapters report unavailable with the missing backdrop capability; the ramp never silently replaces the typed client. |
| `secrets-manager` | omitted | yes | `must_start` | Holds the upload signing key by reference so no key material exists in this repository, in a generated project, or in evidence. | Builds requiring a signed artifact report unavailable naming the signing rung. The ramp never generates an ad-hoc local key as a fallback. |
| `hello-mobile` | omitted | no | `try_start` | Conformance fixture. The ramp must be provable end to end without depending on any product scenario's correctness. | The ramp's own conformance proof cannot run. Product scenario deliveries are unaffected. |
| `scenario-to-desktop` | omitted | no | `try_start` | Compile-time shared target-verdict domain contract used by the release matrix; the Android ramp does not call the desktop scenario at runtime. | The local release matrix cannot compile until the shared desktop-domain package is available; no desktop process is required at runtime. |

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
