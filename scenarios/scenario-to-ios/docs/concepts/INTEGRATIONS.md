# Integrations — Scenario to iOS

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
| `vrooli-bridge` | scenario | **yes — hard** | targets, builds, journeys | Node registry, allowlisted dispatch, durable runs | Without a reachable macOS node this ramp can do nothing on a Linux host. Every native target reports `unavailable` with the reach reason. This is the only ramp for which bridge is load-bearing rather than an enhancement. |
| macOS node with Xcode | host capability | yes | builds, targets | Verified and instructed; never installed by us | Build reports `unavailable` naming the missing toolchain and its required version. |
| `device-control` | scenario | yes | journeys, targets | Connect-RPC: device inventory, lease acquire/release, device verbs; `ios-simctl`, `ios-xcuitest`, `ios-mirror` strategies | Journey cell reports `unavailable` naming device-control. Never drives the simulator or device directly. |
| `deployment-manager` | scenario | yes | releases | `common.v1.TargetVerdict` with `EvidenceRef` entries | Gate computes locally; verdict emission retries. Evidence is never discarded because the consumer is down. |
| `browser-automation-studio` | scenario | yes | journeys | Named BAS flow executed against the app WKWebView via the WebKit inspector protocol | Web-content chapters report `unavailable`; native shell chapters still run and report honestly. |
| `secrets-manager` | scenario | yes | builds, distribution | Certificates, provisioning profiles, and App Store Connect keys by reference | Build reports `unavailable` naming the signing rung. Never generates an ad-hoc identity. |
| `hello-mobile` | scenario (fixture) | yes for conformance | journeys | Generated, built, and driven as the ramp's own proof | Conformance proof cannot run; product scenarios are unaffected. |
| App Store Connect | third-party service | no | distribution, readiness | App Store Connect API; IPA upload, TestFlight, review submission | Channel reports `unavailable` with the blocking rung. Simulator-only channel is unaffected. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None acquirable | by design | The Apple toolchain cannot be installed, licensed, or versioned by us. Modelling Xcode as a resource would put a permanently failing acquisition step in the resource fleet. It is a **host capability**: verified and instructed. | Never — this is a property of Apple's licensing, not a gap. |
| macOS + Xcode 26 or later | host capability on a bridge node | Required for every build, simulator run, and physical-device session. Probed for version, simulator runtimes, and GUI session. | A node's Xcode is upgraded, or Apple raises the App Store Connect floor. |
| x86_64 simulator runtime variant | host capability, Intel hosts only | On Intel Macs the universal architecture variant must be fetched explicitly; it is not installed by default. | Probed per node; absence yields `unavailable` with the fetch command as the next action. |
| Logged-in GUI session | host capability on a macOS node | The iOS Simulator requires a graphical session. The bridge agent already installs as a LaunchAgent bootstrapped into the `gui/<uid>` domain and documents that SSH-only sessions have no GUI bootstrap. | Probed on the live node, never assumed from the agent's source. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `vrooli-bridge` | required — hard | No Apple toolchain runs on Linux, so remote execution is this ramp's normal path rather than a fallback. | Allowlisted verbs, per-node scopes, durable run identity preserved. |
| `device-control` | required | Every device verb belongs to that scenario, across all three iOS strategies. This ramp implements no device driving of its own. | Lease held for the full run duration; verbs refused without one. |
| `deployment-manager` | required | The evidence consumer and release-gate authority. | Reference-only `common.v1.TargetVerdict`. |
| `browser-automation-studio` | required | Owns web-content automation. A scenario's own flows replay inside the generated app through BAS, not through a mobile-specific rewrite. | Named flow, WKWebView attach, chapters merged onto the journey timeline. |
| `secrets-manager` | required | Holds signing identity so no key material exists in the repository. | Reference-only; material never returned into this scenario. |
| `hello-mobile` | required (fixture) | The ramp must be provable without depending on any product scenario's correctness. | Generated and driven like any other scenario. |

## Backdrop listing assets

This scenario owns journey capture and the App Store submission relationship.
For imagery, it supplies a captured screenshot and a Backdrop Studio surface id
to `ComposeDeviceFrame`; Backdrop Studio returns PNG bytes at the surface's exact
geometry plus the reserved device-frame region. This client never captures
screenshots and never reimplements treatment or surface rules. Publishing stays
owned by the existing listing pipeline.

| Input | Contract |
|---|---|
| `surface_id` | Backdrop Studio surface identifier, such as `app_store_6_7_screenshot`. |
| `screenshot_png` | Screenshot captured by this scenario's journey/device seam. |
| `arrangement` | `device_center`, `caption_above_device`, `caption_below_device`, or `caption_only`. |
| Output | Composed PNG, exact `width`/`height`, and an occlusion region. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Apple Developer Program | required for everything past the simulator | Gates signing, physical-device provisioning, TestFlight, and the App Store. Also gates `ios-xcuitest`, which cannot sign WebDriverAgent without it. | $99/yr enrollment; an owner action. The `ios-mirror` strategy is the deliberate escape hatch that needs none of it. |
| App Store Connect API | required for TestFlight and App Store channels | IPA upload, build management, and review submission. | API key held in secrets-manager; unavailable until the ladder rungs are satisfied. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| `vrooli-bridge` | No macOS node online | Every native iOS target reports `unavailable` carrying the reach reason. The Linux-host rows stay `unsupported`, which is a different and terminal state. | targets integration tests |
| macOS node toolchain | Xcode absent or below the SDK floor | Build fails closed naming the required version, rather than producing an artifact that will be rejected at upload. | builds assertion tests |
| macOS GUI session | No `gui/<uid>` bootstrap | Simulator targets report `unavailable` naming the session requirement. Never falls back to a headless attempt that would hang. | targets probe tests |
| `device-control` | Unreachable, or lease refused | Journey cell reports `unavailable` naming the missing capability. Never drives the simulator directly as a fallback. | journeys integration tests |
| `browser-automation-studio` | WKWebView attach fails | Web-content chapters report `unavailable`; native chapters still execute and the run reports partial honestly. | journeys integration tests |
| `secrets-manager` | Signing identity absent | Build reports `unavailable` naming the enrollment or certificate rung. Never generates an ad-hoc identity. | builds integration tests |
| App Store Connect API | Auth failure or upload rejection | TestFlight and App Store channels report `unavailable` with the reason; simulator-only is unaffected. | distribution integration tests |
| Build host expiry | Node can validate but no longer produce submittable builds | Reported as a distinct, explained state — not a generic failure and never a silent pass. | targets role-resolution tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
