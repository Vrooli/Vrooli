# Integrations — Device Control

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
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `vrooli-bridge` | scenario | yes | `devices`, `sessions` | Node/attached-device registry, scopes, allowlisted dispatch, durable runs, audit | Device inventory reports every device `unreachable` with "bridge unavailable" as the reason. No device is assumed present. |
| `ai-gateway` | scenario | for `ai.*` steps and agent planning | `flows`, `agent` | Provider-neutral inference by intent, role, and constraints | AI steps and agent planning report `unavailable` naming the missing gateway capability. No direct provider fallback exists. |
| `agent-manager` | scenario | deferred | none in the delivered bounded loop | External agent orchestration may wrap the CLI; the current bounded agent run owns its loop, lease, chapters, abort, and promotion locally. | No runtime call is made, so its absence does not affect device-control agent runs. |
| `prompt-manager` | scenario | for agent mode | `agent` | The operator skill that teaches this scenario's CLI | Agent mode refuses to start rather than improvising without the skill. |
| `browser-automation-studio` | scenario | optional | `flows` | Named flow execution against an attached WebView | `bas.*` steps report `unavailable`; every other step kind still runs. Also gated on the strategy declaring `webview-attach` — a `bas.*` step on a strategy without it is `unsupported`, not `unavailable`. |
| `android-sdk` | resource | for `android-adb` | `strategies`, `conformance` | `adb`, platform-tools, emulator, system images, AVD lifecycle | The `android-adb` strategy reports `unavailable` naming the missing tool. Other strategies are unaffected. A physical conformance run fails closed unless the fixture APK and adb are both available. |
| Xcode | host capability | for iOS strategies | `strategies` | `xcodebuild`, `simctl`, `devicectl`, signing identities | Probed and instructed, never installed by us. Missing Xcode makes the iOS strategies `unavailable` with an install next-action. |

## Vrooli Resources

This scenario declares one planned Vrooli resource and one deliberate
non-resource. The distinction matters: a resource is something we can
acquire and version; a host capability is something we can only verify and
instruct.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `android-sdk` | active | The `android-adb` strategy needs adb and platform-tools for physical control; emulator, system images, and AVD lifecycle remain optional extensions. Installation must be an observable success or an explicit failure. | Revisit when mobile ramps require managed emulator images or AVD lifecycle. |
| Xcode | not-a-resource | We cannot install, license, or version Xcode. Modelling it as a resource would put a permanently failing acquisition step in the resource fleet. It is a **host capability**: probed by `strategies`, and reported `unavailable` with an install next-action when absent. | Never. Revisit only if Apple ships a redistributable toolchain. |
| SQLite | active | Default embedded store for flow definitions, run history, capability snapshots, leases, and audit. | Revisit only if run history outgrows single-writer access. |

## Scenario Dependencies

This scenario is deliberately not standalone. It sits in a four-layer stack
and owns exactly one layer; the boundaries below are the reason each
dependency exists.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `vrooli-bridge` | required | Owns the **reach plane**: which devices exist, whether they are online, and whether this caller may send them anything. A phone is modelled as an *attached device* — a fleet member that does not run the bridge agent and is reachable only through a host node. Putting that registry here instead would split the single answer to "what do I control." | Node and attached-device registry, per-node scopes, allowlisted dispatch verbs, durable runs, audit records. |
| `ai-gateway` | required for `ai.*` steps | Owns **all inference**. This scenario holds no provider client, model slug, or provider secret. The `flows` domain uses the generated Connect client and the `locate.visual` role. | Provider-neutral request with a caller-owned, downscaled frame; the resolver records the rung, confidence, submitted dimensions, and local device-coordinate conversion. |
| `agent-manager` | deferred | A future integration may own long-lived orchestration. The delivered agent mode keeps the bounded loop in device-control so typed state, direct actuation, chapters, and lease scope remain one transaction. | No direct dependency or provider SDK is used. |
| `prompt-manager` | required for agent mode | Owns the **operator skill** that teaches an agent this scenario's CLI. Keeping the skill there rather than here means the agent-facing instructions are versioned and discoverable with every other skill. | Skill read by slug; the skill's content is the CLI contract. |
| `browser-automation-studio` | optional | Owns **web-content automation**. Because generated apps wrap the same web bundle everywhere, a scenario's real UX flows are BAS's domain on every surface; this scenario drives only the native shell around them. | Named flow execution against an attached WebView; result merged into one evidence timeline. |
| `deployment-manager` | none (indirect) | Consumes device evidence only through the delivery ramps, never directly. Listing it as a dependency would invert the layering. | No direct call in either direction. |
| `device-sync-hub` | none (adjacent) | Moves files *between* trusted devices; this scenario *drives* them. Same fleet, opposite direction, no shared contract. | No direct call. |

### Inbound consumers

These scenarios depend on this one. They are listed because the
direction is load-bearing: this scenario must never acquire a dependency
on them, and a reverse import is an architecture defect rather than a
convenience.

| Scenario | Direction | Reason | Contract |
|---|---|---|---|
| `scenario-to-android` | inbound only | The ramp's `Driver` adapter is a thin translator that turns ramp intent — "install this artifact, run this journey chapter" — into device verbs. It serves emulator and physical Android identically through the `android-adb` strategy. | Calls this scenario's verb surface under a held lease. This scenario never learns what an artifact or a release is. |
| `scenario-to-ios` | inbound only | Same translator role. Selects among `ios-simctl`, `ios-xcuitest`, and `ios-mirror` by what the target can prove, which is why capability probing rather than device kind decides. | As above. See the non-promotability gap in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) before treating `ios-mirror` output as release evidence. |
| `scenario-to-desktop` | inbound, optional | May consume the `host-desktop` strategy once it exists, reusing headless display and input tooling through the same flow vocabulary. Its existing Electron journey path is independent of this scenario. | Optional; no coupling today. |
| `packages/delivery-ramp-go` | contract only | The shared ramp spine defines the `Driver` adapter these ramps implement. No code dependency in either direction — the boundary is described, not imported. | See [`../internal/SEAMS.md`](../internal/SEAMS.md#strategy-is-not-a-ramp-driver). |

Mobile conformance chapters (`install_cold_start`,
`process_death_restore`, `update_migration`, and the rest) are
**ramp-owned journeys executed through this scenario's verbs**. The
chapter set, its assertions, and the release gate belong to the ramp;
this scenario supplies `app-lifecycle`, input, and capture, and knows
nothing about why they were called. That is also why `manager` is the
profile the ramps actually need — a `driver`-only strategy can validate
a running app but cannot run a conformance journey.

The device-control API exposes the provider-neutral chapter contract at
`GET /api/v1/conformance/android` and executes a caller-supplied
`hello-mobile` fixture at `POST /api/v1/conformance/android/run`. The result
contains chapter dispositions, producer-owned evidence references, and a
`common.v1.TargetVerdict` marked `DEVICE_KIND_PHYSICAL`; it never embeds APK
bytes or local artifact paths. The shared contract descriptor is
`fixtures/hello-mobile.contract.json`; its buildable source project is owned by
`scenario-to-android/fixtures/hello-mobile`. The generated debug APK is a
validation artifact and is intentionally not committed. Physical execution
still requires building that artifact and installing it on the attached phone.

### Working vision integration

The `flows` domain posts a caller-owned frame to
`/api/v1/flows/resolve-target`. The resolver decodes and downsizes the frame
before creating an `ai-gateway` `InferenceService.Run` request for role
`locate.visual`. The generated Connect client is the only inference transport
in this scenario; there is no provider SDK, model slug, provider URL, or
provider secret here.

The gateway returns canonical normalized bounds. Device-control converts them
to the original capture dimensions locally, so downscaling cannot change the
device coordinate space. Evidence emits `attempt_vision`, then `resolved` or
`unresolved`, and records the selected rung and confidence without recording
frame bytes or screen text. If confidence is below the caller threshold, a
caller-supplied visual anchor may resolve the target at the lower rung and
the evidence includes `fallback`. If no route exists, the response is the
typed `vision_route_unavailable` disposition; no direct provider fallback is
attempted.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | This scenario reaches no third-party service directly. Everything external is mediated: inference through `ai-gateway`, device reach through `vrooli-bridge`. That is deliberate — a scenario that can read a personal device's screen must not also hold its own outbound credentials. | Revisit only if a device transport requires a vendor cloud (for example a manufacturer's remote-device service), which would be a security decision before it is an integration one. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
