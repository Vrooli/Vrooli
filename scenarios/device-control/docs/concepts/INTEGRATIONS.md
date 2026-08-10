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
| SQLite | embedded storage | yes | API, persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `vrooli-bridge` | scenario | yes | `devices`, `sessions` | Node/attached-device registry, scopes, allowlisted dispatch, durable runs, audit | Device inventory reports every device `unreachable` with "bridge unavailable" as the reason. No device is assumed present. |
| `ai-gateway` | scenario | for `ai.*` steps | `flows` | Provider-neutral inference by intent, role, and constraints | `ai.*` steps report `unavailable` naming the missing gateway capability. Flows with no `ai.*` step are unaffected. Never falls back to a direct provider call. |
| `agent-manager` | scenario | for agent mode | `agent` | Run spawn, bounds, transcript, terminal state | `device-control agent run` reports `unavailable`. Flow execution is unaffected. |
| `prompt-manager` | scenario | for agent mode | `agent` | The operator skill that teaches this scenario's CLI | Agent mode refuses to start rather than improvising without the skill. |
| `browser-automation-studio` | scenario | optional | `flows` | Named flow execution against an attached WebView | `bas.*` steps report `unavailable`; every other step kind still runs. Also gated on the strategy declaring `webview-attach` — a `bas.*` step on a strategy without it is `unsupported`, not `unavailable`. |
| `android-sdk` | resource | for `android-adb` | `strategies` | adb, platform-tools, emulator, system images, AVD lifecycle | The `android-adb` strategy reports `unavailable` naming the missing tool. Other strategies are unaffected. |
| Xcode | host capability | for iOS strategies | `strategies` | `xcodebuild`, `simctl`, `devicectl`, signing identities | Probed and instructed, never installed by us. Missing Xcode makes the iOS strategies `unavailable` with an install next-action. |

## Vrooli Resources

This scenario declares one planned Vrooli resource and one deliberate
non-resource. The distinction matters: a resource is something we can
acquire and version; a host capability is something we can only verify and
instruct.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| `android-sdk` | planned | The `android-adb` strategy needs adb, platform-tools, the emulator, system images, and AVD lifecycle. This is fully scriptable, so it earns real resource treatment rather than being a host prerequisite. | Create the resource before `OT-P0-011` (`android-adb` strategy) starts. |
| Xcode | not-a-resource | We cannot install, license, or version Xcode. Modelling it as a resource would put a permanently failing acquisition step in the resource fleet. It is a **host capability**: probed by `strategies`, and reported `unavailable` with an install next-action when absent. | Never. Revisit only if Apple ships a redistributable toolchain. |
| SQLite | active | Default embedded store for flow definitions, run history, capability snapshots, leases, and audit. | Revisit only if run history outgrows single-writer access. |

## Scenario Dependencies

This scenario is deliberately not standalone. It sits in a four-layer stack
and owns exactly one layer; the boundaries below are the reason each
dependency exists.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `vrooli-bridge` | required | Owns the **reach plane**: which devices exist, whether they are online, and whether this caller may send them anything. A phone is modelled as an *attached device* — a fleet member that does not run the bridge agent and is reachable only through a host node. Putting that registry here instead would split the single answer to "what do I control." | Node and attached-device registry, per-node scopes, allowlisted dispatch verbs, durable runs, audit records. |
| `ai-gateway` | required for `ai.*` steps | Owns **all inference**. This scenario holds no provider client, model slug, or provider secret. See the blocking gap below. | Provider-neutral request by intent, role, profile, and constraints; route evidence returned with the result. |
| `agent-manager` | required for agent mode | Owns the **agent runtime**. This scenario supplies the goal, the bounds, and the device lease; it does not implement a reasoning loop. | Run spawn with bounds, transcript, terminal state, abort. |
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

### Blocking gap — `ai-gateway` has no visual-understanding request kind

`ai-gateway`'s request kinds are `TEXT_GENERATION`, `TEXT_EMBEDDING`,
`STRUCTURED_EXTRACTION`, `IMAGE_GENERATION`, and `VIDEO_GENERATION`. Every
one of them takes text *in*; `IMAGE_GENERATION` produces an image rather
than interpreting one. There is currently **no way to ask the gateway to
look at a screenshot**.

This blocks the `ai.*` step kinds and the `vision` rung of the target
resolution ladder. It does not block the rest of the scenario: the floor,
strategies, sessions, flows, and both deterministic resolution rungs are
independent of it.

The tempting shortcut is the one `browser-automation-studio` already took —
its vision agent calls OpenRouter, Anthropic, and Ollama directly from
`playwright-driver/src/ai/vision-client/`, bypassing the gateway entirely.
That is precisely the direct-provider coupling `ai-gateway`'s own
conformance phase exists to flag. **This scenario must not repeat it.** The
gateway capability is a declared prerequisite, and until it exists the
`ai.*` steps report `unavailable` naming exactly that.

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
