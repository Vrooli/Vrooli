# Decisions — Device Control

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| ID | Date | Decision | Revisit Trigger |
|---|---|---|---|
| D-000 | 2026-08-10 | Use the generated `react-vite` scenario documentation contract. | Scenario adopts a different template or doc contract. |
| D-001 | 2026-08-10 | The strategy contract requires stable identity and a declaration; observation, input, media, properties, and other modalities are optional and probed. | A transport needs a mandatory interface beyond identity and declaration. |
| D-002 | 2026-08-10 | Capabilities are probed, never inferred from device kind. | Never — this is the scenario's core honesty property. |
| D-003 | 2026-08-10 | One flow envelope, several step vocabularies. | A vocabulary needs semantics the shared envelope cannot express. |
| D-004 | 2026-08-10 | Target resolution is a ladder, not a strategy choice. | A fourth rung appears, or rung ordering stops holding. |
| D-005 | 2026-08-10 | All inference goes through `ai-gateway`; the generated `locate.visual` role unblocks the vision rung without reversing the boundary. | A fourth inference transport or a change to the gateway's provider-neutral contract. |
| D-006 | 2026-08-10 | Leases are P0, not a later refinement. | Never while any single-session strategy exists. |
| D-007 | 2026-08-10 | The CLI is the contract, because the agent drives it. | Agent mode stops driving the CLI. |
| D-008 | 2026-08-13 | `vrooli-bridge` owns reach and the short-lived device-lease authorization record; this scenario owns lease semantics and operation. Device-scoped bridge dispatch requires a matching held lease record and is audited on refusal. | Bridge changes its fleet model, or a device class fits neither side. |
| D-009 | 2026-08-10 | Recording is a guaranteed capability with a synthesized fallback. | The ramps stop requiring video evidence. |
| D-010 | 2026-08-10 | A capability gap report is a successful response, not an error. | Never — making it an error would hide the scenario's most common honest answer. |
| D-011 | 2026-08-10 | Errors carry identifiers and reasons, never device content. | Never — this is a redaction bypass, not a style preference. |
| D-012 | 2026-08-10 | Every declared capability is exercisable by some flow construct. | Never — an unexercisable capability is an unfalsifiable claim. |
| D-013 | 2026-08-13 | Android generated-app conformance belongs to scenario-to-android; device-control retains only the provider-neutral device capability self-test. | The delivery ramp needs to vary its chapter set without changing the device strategy, while device-control must prove install, permissions, screenshot, and recording independently of a fixture. | Revisit when a device capability cannot be self-tested without an application supplied by a delivery ramp. |
| D-014 | 2026-09-06 | Keep the native desktop adapter in the lifecycle-managed Go helper and do not add Terminator as a runtime dependency. | A pinned Terminator release must expose a supported, embeddable adapter boundary with equivalent helper-owned identity, lease/session checks, and packaging evidence. |
| D-015 | 2026-09-17 | Treat Android TV and TV boxes as first-class Android targets of `android-adb`, with controller-owned network transport profiles and owner-asserted composition with Cast/Android TV Remote. Application install and launch stay behind the same lease, artifact-identity, capability-probe, audit, and postcondition gates as phone control. | A future Android transport or vendor API provides a stronger identity and application-control contract without weakening the common strategy boundary. |
| D-016 | 2026-11-09 | Cast, Android TV Remote and ADB-over-network are transports of one TV. A same-host candidate is confirmed explicitly (`device relation-confirm`, or `onboard-network --attach-to`); the non-ADB identity stays canonical; model disagreement refuses unless forced. | A transport offers a hardware claim shared with Cast, making confirmation automatic and safe. |
| D-017 | 2026-11-09 | One-call verbs (`screenshot`, `record`, `open`) own wake and evidence download; capture gaps and failures always carry `next_actions`; `device list` is compact by default. | A verb needs a side effect the owner has not authorized (for example Home), or callers need the full inventory by default. |
| D-018 | 2026-11-09 | Inventory reconnects an absent network ADB endpoint once, gated by a 60 s backoff per device; a display name resolves without inventory only when freshly observed and available. | A transport reports presence cheaply enough to poll, or name collisions make cached resolution unsafe. |
| D-019 | 2026-11-09 | Observe and open-app are programs bound to new typed `DeviceService` RPCs (`CaptureDevice`, `LaunchApp`), not canned flows. | The flow engine gains transport selection and wake as step semantics. |
| D-020 | 2026-11-09 | Owner-held resources with a fallible close (pairing handshake, native recording) share one explicit protocol: open, closing, uncertain; an uncertain close is never replayed. | A third owner needs a retry-from-uncertain policy. |
| D-021 | 2026-11-09 | Executable specifications of user journeys are Gherkin `.feature` files run by `internal/testutil/gherkin` (`TestFeatures`); unit-level Go tests remain for pure functions and fixtures too rich for plain steps. | A maintained Gherkin framework is approved as a dependency and replaces the in-repo runner. |
| D-022 | 2026-11-09 | The lifecycle provider declares `OPERATION_DURABLE` fence persistence, reports not-drained while owner quiescence is owed, and serves `/health` and `/api/v1/health` behind a closed fence. | The shared `api-core` lifecycle provider replaces the local one. |
| D-023 | 2026-11-09 | Hands-free unlock: when a swipe does not unlock a locked phone, `screen ready` (and so every one-call verb) uses the device's own active unlock profile; a profile whose last outcome was wrong, failed or unconfigured is never retried automatically. Lock-screen security settings are never changed. | A stronger unattended-unlock mechanism (for example a trusted-place or trusted-device policy the owner configures) replaces the PIN path. |
| D-024 | 2026-11-09 | "Try an app on a phone" is one verb (`device install --apk --open --screenshot`, RPC `InstallApp`) and one program (`device-control.check-app`); `check-app` also covers screenshot-only and the install-free TV path. The separate `observe` and `open-app` programs were folded into it. | Programs need recording or log capture, or a second program shape (then reinstate `observe`). |

## Decision Details

### D-000 — Generated `react-vite` documentation contract

The scenario scaffold was generated from the template, so docs start with
stubs and maturity metadata in `docs/manifest.json`. Maturity values are
maintained to match actual content in both directions — a doc that
over-claims `active` is as much a defect as one that under-claims `stub`.

### D-001 — Identity and declaration are mandatory; modalities are optional

Every strategy supplies stable identity (`ID()`) and `Describe()`. Observation,
input actuation, media, properties, sensors, and other modalities are optional
interfaces backed by the declaration and an inventory-time probe.

*Why this size:* a screenless transport such as Google Cast can provide media
and property control without claiming a frame or input channel. Requiring
those modalities would exclude useful transports and make capability claims
less honest. Shared capabilities are composed from the interfaces each
transport actually implements.

*Rejected alternative:* require observation and input. It would exclude
screenless media transports and incorrectly imply that every useful device has
a visual surface.

### D-002 — Capabilities are probed, never inferred from device kind

A capability that cannot be proven at probe time is `unavailable` with a
named prerequisite and next action.

*Why:* the failure this prevents is a release gate believing a device
validated something it never ran. "Android phones usually have X" is not
evidence about *this* phone right now.

### D-003 — One flow envelope, several step vocabularies

`device.*`, `ai.*`, `flow.*`, `wait.*`, `bas.*` share one envelope, one
timeline, and one evidence record.

*Why not one unified action vocabulary:* device verbs and web-content verbs
are genuinely different domains — tapping a coordinate is not clicking a
selector. But a real automation crosses that boundary constantly (drive
native onboarding, then drive the web content inside the app). Sharing the
envelope makes them composable; merging the vocabularies would make both
worse.

*Consequence:* `browser-automation-studio` remains the authority for web
content on every surface, and this scenario never grows a competing
web-automation engine.

### D-004 — Target resolution is a ladder, not a strategy choice

`semantic` → `visual-anchor` → `vision`, highest supported rung wins, rung
and confidence always recorded.

*Why:* this is the mechanism that makes "shared capabilities work on any
strategy" true rather than aspirational. Vision is the universal fallback;
semantics are the fast, free, deterministic path. The flow author writes
intent once and does not choose — the strategy's declared capabilities
decide. Recording the rung is what lets a reviewer tell a proven result
from an inferred one.

### D-005 — All inference goes through `ai-gateway`

The `flows` domain sends visual understanding through the generated
`InferenceService` Connect client using role `locate.visual`. The caller
owns frame downscaling and the conversion from canonical normalized bounds
back to device coordinates. If the route is unavailable, the flow reports a
typed unavailable disposition or uses an existing visual anchor; it never
reaches for a provider client.

*Why preserve the boundary:* the alternative is the shortcut
`browser-automation-studio` already took — a direct OpenRouter/Anthropic/
Ollama client in `playwright-driver/src/ai/vision-client/` — which is
exactly the coupling `ai-gateway`'s conformance phase exists to flag. Taking
it twice would make the gateway boundary fictional. The device-control
caller therefore names an intent and role, never a provider, model, URL, or
credential.

### D-006 — Leases are P0, not a later refinement

*Why:* several strategies are physically single-session. Without a lease,
two consumers do not collide loudly — they interleave and quietly corrupt
each other's evidence. That is the hardest class of failure to diagnose
after the fact, and retrofitting exclusivity into an executor that assumed
it was free is expensive.

### D-007 — The CLI is the contract, because the agent drives it

CLI completeness is a P0 target rather than a convenience.

*Why:* the prompt-manager skill drives devices through this scenario's CLI,
while the bounded agent loop uses the same lease-owned control surface. Any
capability reachable only from the API or UI is invisible to the agent, so
CLI parity is a functional requirement, not ergonomics.

### D-008 — `vrooli-bridge` owns reach; this scenario owns operation

A phone is modelled in bridge as an *attached device*: a fleet member that
does not run the bridge agent and is reachable only through a host node.

*Why the split:* bridge's value is being the single answer to "what do I
control and may I." Putting a second device registry here would split that.
But if bridge also held ADB, WebDriverAgent, and mirroring logic it would
become a mobile toolkit, and every new device platform would bloat the trust
plane. Reach and operation are different rates of change.

The reach gate also owns the short-lived authorization record needed to close
the out-of-band dispatch path. Device-control remains authoritative for lease
lifecycle semantics: it acquires, renews, expires, kills, and releases the
lease, then supplies the bridge gate with the device-scoped token and expiry.
Bridge never treats a non-empty token as proof by itself; it refuses a
device-scoped verb unless its own held record matches the device and token, and
it appends the refusal to the dispatch audit trail. This keeps the trust-plane
check at the point where reach is granted without moving device operation into
bridge.

### D-009 — Recording is capability-gated with a synthesized fallback

Native when the strategy declares it; synthesized from the `observe()` frame
stream otherwise. The evidence always records which path produced the video
and its effective frame rate.

Recording is optional because a screenless transport cannot honestly produce
frames. A frame-bearing strategy may use native recording or synthesize a
labeled recording from its observation stream; Google Cast declares the
screen and recording capabilities unavailable.

*Why the fallback is sound:* the strategy declaration makes the frame stream
explicit, and encoding it into video is a local, deterministic operation.

*What it must never do:* label a synthesized capture as native, or omit the
effective frame rate. A 2 fps reconstruction and a 60 fps native capture are
both legitimate evidence, but they support different claims, and a reviewer
must be able to tell them apart without opening the file.

### D-010 — A capability gap report is a successful response, not an error

Asking "can this flow run on this device?" and getting back "no, and here is
exactly which capabilities are missing" is a **successful** call carrying a
structured gap report. Only *dispatching* a flow despite a known gap is an
error.

*Why:* "this device cannot do X" is the single most common honest answer this
scenario gives. Modelling it as an error would make the normal case
indistinguishable from a broken request, and would force every caller to
parse error strings to recover structured data the API already holds.

*Rejected alternative:* raise on any unsatisfiable request. It reads as
stricter and is actually weaker — callers end up catching broadly and losing
the per-capability detail that makes the gap report useful.

### D-011 — Errors carry identifiers and reasons, never device content

No error message, log line, or CLI output may contain frame bytes, capture
paths, text read from a device screen, clipboard values, device logs, or
credentials.

*Why:* errors do not pass through `EvidenceSink`, so they never receive
redaction verification. That makes the error channel the one path by which
device content could leave unchecked — a redaction bypass rather than a
formatting preference.

*Rejected alternative:* echo what the resolver searched for and what it saw.
This is the *natural* implementation and the reason the rule needs writing
down: `could not find target "password" in frame; visible text was "Enter
code 481920"` is a genuinely helpful debugging message and a credential leak
in the same sentence. Naming the target identifier and the per-rung outcome
is strictly more useful for debugging and carries nothing off the device.

### D-012 — Every declared capability is exercisable by some flow construct

No capability may be declarable unless something in the flow vocabulary
behaves differently because of it, and no flow construct may require a
capability the registry does not list.

"Exercisable" rather than "has a verb", because there are three legitimate
mechanisms and only the first is a step requirement:

| Mechanism | Example |
|---|---|
| A step **requires** it | `device.install` requires `app-lifecycle` |
| A resolution **rung** requires it | the `semantic` rung requires `semantic-tree` |
| A step's execution **path** selects on it | `device.record` uses `native-recording` when declared, synthesis otherwise |

The first draft of this decision said "reachable by at least one step verb",
and verifying it immediately flagged `semantic-tree` and `native-recording` as
violations — both of which are perfectly legitimate. The rule was too narrow,
not the registry. Recorded here because the narrow version is the obvious one
to reach for again.

*Context:* assembling [`../reference/capabilities.md`](../reference/capabilities.md)
surfaced five capabilities — `network-control`, `orientation`, `clipboard`,
`file-transfer`, `device-logs` — that a strategy could declare and probe but
that no step could ever use. `device.network`, `device.orientation`,
`device.clipboard`, `device.push` / `device.pull`, and `device.logs` were added
to close the gap.

*Why the rule rather than just the fix:* an unreachable capability is an
**unfalsifiable claim**. It can be declared, it can be probed, it can appear
`available` in a report — and nothing can ever exercise it to prove the
declaration was true. That is the same class of confident-but-empty statement
`D-002` refuses when it insists capabilities are probed rather than inferred,
arriving by a different route. The registry is only trustworthy if every row
in it is exercisable.

*Rejected alternative:* drop the five capabilities as aspirational. Cheaper,
and wrong for two of them — `file-transfer` and `device-logs` are load-bearing
for the delivery ramps, which need to pull artifacts and diagnostics off a
device after a conformance journey.

*Consequence:* adding a capability now requires adding the construct that
exercises it in the same change. The reverse also holds — a proposed construct
whose capability is not in the registry is a signal that the registry is
incomplete, which is how `webview-attach` was found.

### D-014 — Native helper adapter selection

The bounded Terminator evaluation used the repository's pinned `main` commit
`73a381c0c1c33eda55f2c0ecb1d918bf5ec7561a` (MIT). Its upstream support table
and README describe Windows-only automation delivered through its own Rust,
Node, CLI, and MCP packages; macOS and Linux are explicitly unsupported. No
Vrooli helper IPC, lease/session binding, or artifact packaging contract is
provided for embedding. Device Control already has a Go helper with opaque
UI Automation references, signed local admission, and platform-specific
backend selection, so introducing Terminator would add a second authority and
would not satisfy the required cross-platform adapter boundary. Re-evaluate
only when a release provides an embeddable adapter and equivalent conformance
evidence.

### D-015 — Network ADB onboarding and default-transport routing (2026-09-17)

Android TV and TV boxes are first-class ADB targets that are typically reachable
only by network address (classic TCP 5555 or Wireless Debugging/TLS), never over
USB. `OnboardNetworkADB` adopts such an endpoint directly: it validates the
endpoint, connects, confirms authorization, and derives the durable identity from
the hardware serial so the network address is never treated as the device
identity. The endpoint-bound adapter is persisted as the device's transport
strategy, matching the existing wireless-promotion machinery.

This exposed a latent routing assumption. `strategyForFlow` previously defaulted
an unselected transport for any wireless device to USB ("wireless must be
explicit"), which assumed every wireless device had a USB origin. A
network-onboarded TV has no USB base, so that default routed to the ambient base
adapter — precisely the "stale or ambient endpoint" DVC-P0-011 says to refuse.
The corrected contract: an unselected transport resolves to the device's current
governed transport; a wireless device (USB-promoted phone or network-onboarded
TV) resolves to its verified endpoint-bound strategy when one is registered, and
falls back to USB only when no wireless strategy exists. Explicit transport
selection is still honored and unknown transports are still refused. See
`TestFlowTransportResolvesWirelessToEndpointBoundStrategy`.

## Superseded Decisions

## 2026-09-18 — Desktop ownership and first-release shape

Device Control owns desktop sessions and semantics; Bridge owns trusted
interactive reach/signaling/revocation; Web Console owns the desktop pane.
The first implementation uses a Pion Go companion, native ScreenCaptureKit,
Quartz/CGEvent input, VP8, one selected display, raw plus semantic input, and
text clipboard. Direct WebRTC is preferred with TURN fallback. A separate
remote-desktop scenario is not introduced for this release.

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`SEAMS.md`](SEAMS.md) — the seams these decisions imply
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history

### D-016 — One TV, several transports (2026-11-09)

*Evidence.* A Cast TV, its Android TV Remote endpoint and its ADB endpoint appear as separate identities; the
agent session of 2026-11-07 minted a fourth ephemeral one and could not take a screenshot. The relation
candidate (same normalized host, confidence 0.72) already existed and nothing surfaced it.

*Decision.* The owner confirms the relation. The merge asserts the ADB hardware serial (`adb-serial`), keeps the
identity the owner already knows canonical, retains every transport profile across restart, and routes the ADB profile
by serial plus network endpoint (`strategyForFlow` no longer requires the identity's primary transport to be
`wireless`). Confirmation refuses when both sides report a model and the names disagree unless forced.

*Alternatives.* Auto-merge at 0.72 confidence: rejected, same-host is not same-device (bridges, containers, NAT).
One identity per host: rejected, it loses the per-transport capability honesty of D-002. A new "ADB rung" as a
separate identity kind: rejected, it recreates the second identity the owner cannot name.
*Switch cost.* Auto-confirmation would be a one-function change in `ConfirmRelation`'s caller; the merge, routing and
persistence are already transport-agnostic.

*Why the identity "vanished" (2026-11-07).* It did not: it was retained but reported `device not present on host
node local`, so `device list` hid it. Network ADB endpoints never reconnected after the host's adb server dropped
them. D-018 fixes the cause; the hidden-row presentation is now explicit through `device list --all`.

### D-017 — One-call verbs and next actions (2026-11-09)

`screenshot`, `record` and `open` wake the screen first (`--no-wake` opts out; Home is opt-in because it can change the
foreground app), download evidence to a file, and exit non-zero with `next_actions` on a gap or failure. The service-level
`Capture` keeps wake opt-in (`wake`, `home`) so flow callers see no new side effect.
*Alternatives.* Default wake inside `Capture`: rejected, an invisible actuation for programmatic callers. A single
`observe` verb with `--kind`: rejected, a TV owner searches for "screenshot" and "record".
*Switch cost.* Defaults live in `cli/domains/control/verbs.go` and the program inputs; changing one is a flag default.

### D-018 — Reconnect and name resolution (2026-11-09)

`startPromotedDiscoveries` reconnects an endpoint-bound adapter whose endpoint is absent, behind `reconnectGate` (one
attempt in flight, 60 s after a failure). Name resolution used to run a full inventory (about 7 s) per call; a unique,
available, recently observed identity (2 minutes) now resolves directly, and anything else still inventories.
*Alternatives.* Periodic background reconnect: rejected, it pings a sleeping TV forever. Longer freshness window:
rejected, stale `available` rows would route to a dropped endpoint.
*Switch cost.* Two constants (`reconnectBackoff`, `freshNameWindow`).

### D-019 — Programs bind to typed RPCs (2026-11-09)

Governed program bindings derive from `cli/manifest.json` commands with `connect-rpc` bindings. A canned flow through
`flow/run` cannot select a frame-bearing transport or wake the device, so `CaptureDevice` and `LaunchApp` were added
to `DeviceService` (single repeated response field each, so the binding has an unambiguous primary field).
*Switch cost.* Proto regeneration through the governed owner command; consumers are the two programs and the CLI manifest.

### D-020 — Close protocol (2026-11-09)

`closeState` replaces the `closing`/`uncertain` and `Stopping`/`Uncertain` boolean pairs. A completion arriving at an
uncertain pairing session is now refused (previously it could proceed with the flags in a contradictory state).

### D-021 — Gherkin specifications (2026-11-09)

`api/internal/control/features/*.feature` hold the journeys: one TV many transports, open an app, close protocol,
pairing close, reconnect gating. The runner supports Feature, Background, Scenario, Scenario Outline with Examples,
and regex steps, in about 200 lines with its own tests. *Alternative:* godog, rejected for now because a dependency
needs analyzer approval and the runner covers current needs.

### D-022 — Lifecycle provider contract (2026-11-09)

`vrooli scenario restart device-control` failed in three separate ways, found by running it:
(1) the standing carried no `fence_persistence`, so the control plane could not prove a durable fence across the
replacement; (2) an idle provider with quiescence still owed reported `drained` and `quiescence_pending` together,
which the control plane rejects; (3) `/health` returned 503 behind the closed durable fence, but the control plane waits
for a healthy replacement *before* resuming the fence, so every restart stranded the scenario with admission closed.
All three are fixed in `api/handlers/lifecycle/provider.go` with tests; a restart now completes in about 7 s.
A stranded fence was recovered by resuming it with its exact operation id, token and revision (the same Resume the
control plane issues), then verified with two further ordinary restarts.
*Alternative.* Move to the shared `packages/api-core/lifecycle` provider, which already serves health under a closed
gate: deferred, because this provider's durable (SQLite) fence and component quiescence hooks are device-control specific.

### D-023 — Hands-free unlock (2026-11-09)

*Finding.* The moto had an active, provisioned PIN profile (credential provider available, last outcome
`already_unlocked`), yet `screenshot`, `open` and `observe` reported "locked". `screen ready` only used a profile when
`--auth-profile` was passed, so no one-call verb ever unlocked. The Galaxy A03s's only active profile last failed with
`wrong_credential` and its credential version metadata is absent, so it needs one re-provision.
*Decision.* Choose (a): a profile-backed unlock inside the ready path, after a swipe attempt, selecting the newest active
profile whose last outcome is not wrong/failed/unconfigured (`auth.UnlockCandidate`). Wrong-PIN retries would burn the phone's
lockout counter, so such a profile is surfaced as a re-provision step instead. The onboarding ladder (`connect --kind android`)
has an `unattended-unlock` rung with the exact setup commands, and every locked-screen gap prints them. The PIN is only ever
piped to `auth provision`.
*Alternatives.* (b) keep-awake plus a swipe-only lock screen: rejected as the default because it weakens the phone's security
and the operator asked that lock settings stay untouched; it remains the owner's own choice. (c) both: the profile path already
covers a swipe-only screen through the dismissal step that runs first.
*Switch cost.* One selector function and one call site in `Service.Screen`.

### D-024 — One program for try-an-app (2026-11-09)

*Install* was only reachable through the verified-artifact flow step; a local APK had no one-call path. `InstallApp` reads the package identity from the APK
itself (aapt), installs through the transport that declares app lifecycle, proves it with package-state and the APK's version code, then reuses `LaunchApp` and
`Capture`. A failed install never reaches open or screenshot.
*Why programs were merged.* Adding install and check-app pushed the skill out of the top five for phone queries (recall 15/16 fell to 10/16): five device-control
leaves filled the list. Folding `observe` and `open-app` into `check-app` restored 15/16. Recall budgets leaf count, not only wording.
*TV.* The TV path of `check-app` sends no install: `app` given, open then screenshot only.
*Switch cost.* Reinstating a program is a JSON and Python file pair over existing bindings.
