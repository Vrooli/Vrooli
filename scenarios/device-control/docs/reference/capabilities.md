# Capabilities — Device Control

The canonical registry of device capabilities: what each one is, which flow
step kinds it unlocks, and what each strategy is expected to provide.

`device-control strategy verify <id>` reports against this registry, and the
pre-execution capability gap report (`OT-P0-005`) is computed from the step
table below. Both surfaces read the same rows, so a capability that is not
listed here cannot be required by a step or reported by a probe.

## Host desktop readiness

The legacy `host-desktop` adapter probes capture by executing the capture path
and decoding the complete PNG. Command discovery and PNG headers alone do not
prove readiness. A probe has a five-second command deadline, a 32 MiB output
limit, and a 64-million-pixel decode limit. Inventory retains dimensions and the
probe timestamp, not captured pixels. Failed capture reports unavailable; the
legacy command interface cannot reliably classify permission denial separately
from other failures.

Input readiness remains unavailable until user-session admission is verified.
Discovery never sends a click to test permission. These checks do not establish
authenticated helper identity, OS-session isolation, native semantic control, or
live Windows/Wayland support rows; those remain separate helper and acceptance
evidence.

## Desktop helper transport

The experimental canonical `DesktopHelperService` defines Open, Observe, Act, and Stop
for a local user-session helper. Act has typed pointer, key, text, assertion,
and semantic invocation variants;
no shell command or arbitrary execution endpoint is present. Generated Go,
TypeScript, and Python messages live under the device-control desktop namespace.

`internal/sessions.DesktopUnixHelper` serves an existing Unix listener and uses
kernel peer credentials plus signed desktop grants on every request. The
lifecycle owner must authenticate the OS session, activate the helper generation,
and provision the socket directory and permissions before serving. The helper
runs lease maintenance and revokes control when its lifetime ends. This transport
is not mounted on Device Control's public API and does not yet ship as a managed
helper executable. Native backends and Windows transport remain required. Observe requires its own grant permission and
returns bounded PNG pixels with display identity, geometry revision, dimensions,
and capture time. Observation-only leases cannot actuate. The controller checks
revocation again after capture, and pixel bytes are not stored in the command
receipt journal.

## Native X11 backend

`internal/native/x11` connects to a resolved local display using an explicit
X cookie and an exact logind session ID through `NewForSession`. It captures root pixels and
checks the root dimensions plus RandR configuration timestamp before accepting
pointer input. It uses checked XTest requests, tracks pressed buttons, and
releases those buttons during cleanup. Layout validation and input execute
within a brief X server grab; connection deadlines bound the operation.

The current implementation supports one X screen with a 32-bit little-endian
TrueColor image format and up to 16 million pixels. It provides pointer input;
keyboard, Unicode text, semantic access, and richer monitor metadata remain
unfinished. The production constructor checks fresh control-plane session facts against
the kernel X-socket peer PID/UID and current helper UID before capture/input.
The injected session-check constructor is package-private for conformance tests;
X-server access alone does not establish session authority. The isolated Xvfb test
proves native pixels and pointer state, not a physical desktop support row.

## Native Wayland portal backend

`internal/native/wayland` is selected explicitly as `gnome` or `kde` in the
helper bootstrap for a Wayland user session. It uses the XDG Screenshot portal
for bounded PNG capture and the RemoteDesktop portal for keyboard, pointer, and
wheel notifications. The helper records and rechecks the user-session D-Bus
identity, stops the portal session on shutdown, and never falls back to an
XWayland `DISPLAY`. Absolute pointer and wheel actions are refused when the
portal does not return a usable stream. Semantic access remains optional and is
composed through an authenticated AT-SPI socket when provisioned. Protocol tests
cover both named backends; live GNOME and KDE support rows still require
compositor and permission evidence.

## Native macOS and Windows backend

`internal/native/host` binds non-Linux operations to the current interactive
user session. Windows uses bounded PowerShell capture/input and UI Automation
references. macOS uses bounded `screencapture` and System Events Accessibility
scripts for application/window discovery, exact opaque element resolution, and
checked text mutation. Every session-bound observe and input path rechecks the
user session; revoked or empty macOS capture returns permission evidence rather
than reusing an old image. These seams have deterministic and cross-build
coverage, while live Windows/macOS permission, artifact, and ScreenCaptureKit
acceptance remain required before those support rows can be claimed.

The Go binding is `github.com/jezek/xgb` v1.3.0, approved and installed through
Scenario Dependency Analyzer for Device Control only. API references:
[XGB](https://pkg.go.dev/github.com/jezek/xgb@v1.3.0) and
[XTest](https://www.x.org/releases/X11R7.5/doc/man/man3/XTestFakeKeyEvent.3.html).

## How to read this document

**The probe is the source of truth. This document is the expectation.**

The strategy matrix near the end records what each adapter is *expected* to
provide, based on what its underlying transport can do. It is reference
material for planning, never an input to a runtime decision. A capability is
`available` only when probing proves it on that device at that moment
(`D-002`). Where this document and a probe disagree, the probe is right and
this document is stale.

That distinction is not pedantry. "Android phones usually have X" is exactly
the inference this scenario exists to refuse.

## The mandatory contract

Every strategy supplies stable identity and a declaration. Modalities are
optional interfaces, and each declared modality is verified by probing.

| Operation | Signature intent | Guarantees |
|---|---|---|
| `ID()` | returns a stable strategy identifier | Inventory can correlate transports without guessing from names or addresses. |
| `Describe()` | returns a `CapabilityDeclaration` | What this strategy claims; claims are verified separately — see [`../internal/SEAMS.md`](../internal/SEAMS.md#why-describe-and-capabilityprober-are-two-seams). |
| Optional `observe()` / `actuate()` | returns frames or accepts input events | Only the corresponding declared capabilities and step kinds are available. |

The contract is deliberately this small because screenless transports such as
Google Cast can provide media and property control without claiming a frame or
input channel (`D-001`).

## Optional capabilities

The canonical list. Each has a stable ID used in declarations, probe results,
and gap reports.

| ID | Capability | What it means |
|---|---|---|
| `semantic-tree` | Semantic tree | An accessibility or view hierarchy with addressable elements. |
| `app-lifecycle` | App lifecycle | Install, launch, stop, uninstall a named application. |
| `permission-control` | Permission control | Grant, revoke, or respond to OS permission prompts. |
| `network-control` | Network control | Toggle or condition the device's connectivity. |
| `orientation` | Orientation | Set or read device orientation. |
| `clipboard` | Clipboard | Read or write the device clipboard. |
| `file-transfer` | File transfer | Move files to or from the device. |
| `native-recording` | Native video recording | The transport provides its own video capture. |
| `device-logs` | Device logs | Read the device's system or application log stream. |
| `webview-attach` | WebView attach | Attach a debugger to an application WebView. Required by `bas.*` delegation. |
| `multi-touch` | Multi-touch pointer streams | Two simultaneous normalized pointer streams are available for gestures such as pinch. |
| `property` | Property control | Read or write typed, declared device properties. |
| `sensor` | Sensor readings | Read typed observations that may change over time. |
| `media` | Media transport | Play, pause, stop, next, previous, or set absolute volume. |
| `pairing` | Interactive pairing | Complete a secret-bearing transport pairing exchange without serializing the secret. |

## App lifecycle control

The REST and CLI app surface accepts one package-identified operation at a time:
`launch`, `focus`, `close`, `minimize`, `restore`, `stop`, `uninstall`,
`clear-data`, `grant-permission`, `revoke-permission`, or `package-state`. The request must hold a live device
lease. Mutating operations also require an explicit `confirmed`/`--confirmed`
flag. Package names are validated as fully-qualified identities; executable
paths and arguments are refused until a strategy-owned allowlist exists.

The control plane requires both an available `app-lifecycle` declaration and
the typed `strategy.AppLifecycle` implementation. A declaration without an
executable adapter returns a typed unavailable/unsupported result. Every
attempt receives a command ID and an audit record; lifecycle attempts do not
claim evidence-backed completion without a separate verification step.

## Step kinds and what they require

The mandatory floor is identity plus declaration. Every device step below is
capability-gated; a screenless or read-only transport is valid and simply has
fewer executable steps.

### `device.*`

| Step | Requires |
|---|---|
| `device.observe` | `screenshot` |
| `device.tap` | `input` |
| `device.swipe` | `input` |
| `device.long-press` | `input` |
| `device.double-tap` | `input` |
| `device.drag` | `input` |
| `device.fling` | `input` |
| `device.pinch` | `input` + `multi-touch` |
| `device.scroll-to` | `semantic-tree` or a resolver rung |
| `device.type` | `input` |
| `device.key` | `input` |
| `device.record` | `screenshot` or `native-recording` |
| `device.install` | `app-lifecycle` |
| `device.launch` | `app-lifecycle` |
| `device.share` | `app-lifecycle` |
| `device.stop` | `app-lifecycle` |
| `device.uninstall` | `app-lifecycle` |
| `device.permission` | `permission-control` |
| `device.network` | `network-control` |
| `device.orientation` | `orientation` |
| `device.clipboard` | `clipboard` |
| `device.push` / `device.pull` | `file-transfer` |
| `device.logs` | `device-logs` |
| `device.property-get` / `device.property-set` | `property` |
| `device.sensor-read` | `sensor` |
| `device.media-play` / `device.media-pause` / `device.media-stop` | `media` |
| `device.media-next` / `device.media-previous` / `device.media-volume` | `media` |

### `ai.*`, `flow.*`, `wait.*`, `bas.*`

| Step | Requires | Notes |
|---|---|---|
| `ai.see` | `screenshot` + `ai-gateway` visual request kind | Requires a frame-bearing transport. |
| `ai.extract` | `screenshot` + `ai-gateway` visual request kind | Requires a frame-bearing transport. |
| `ai.verify` | `screenshot` + `ai-gateway` visual request kind | Requires a frame-bearing transport. |
| `ai.decide` | `ai-gateway` text generation | Not blocked — takes prior step results, not a frame. |
| `flow.*` | — | Executor-level. Never touches a device. |
| `wait.*` (fixed duration) | — | Executor-level. |
| `wait.*` (until target appears) | at least one resolution rung | Inherits the rung's requirements. |
| `bas.*` | `webview-attach` | Delegates to `browser-automation-studio`. |

### The `device.record` recording paths

`device.record` requires a frame or native-recording capability. It is not
available to a screenless transport such as Google Cast.

| Strategy declares | Path taken | Evidence records |
|---|---|---|
| `native-recording` | Transport's own capture (`adb screenrecord`, `simctl recordVideo`) | `method: native`, plus effective frame rate |
| `screenshot` without `native-recording` | Synthesized from the `observe()` frame stream | `method: synthesized`, plus effective frame rate |

Both are legitimate evidence. They support different claims, so the method
and the effective frame rate are structural fields on the result, not
optional metadata — a reviewer must be able to distinguish a 2 fps
reconstruction from a 60 fps native capture without opening the file. See
the provenance rule in [`../internal/SEAMS.md`](../internal/SEAMS.md#the-provenance-rule).

This fallback exists because frame-bearing devices may lack native capture;
the declaration still makes the frame prerequisite explicit.

## Resolution rungs

The target resolution ladder (`D-004`) picks the highest rung the strategy
supports. Rung requirements are capability requirements:

| Rung | Requires | Cost | Deterministic |
|---|---|---|---|
| `semantic` | `semantic-tree` | Free, local | Yes |
| `visual-anchor` | `screenshot` + a captured reference in the anchor library | Free, local | Yes |
| `vision` | `screenshot` + `ai-gateway` visual request kind | Tokens + round trip | No |

Note that **`visual-anchor` and `vision` both require a frame**. A screenless
transport can still use typed state and declared non-visual step kinds; it is
not forced to claim a visual rung. The chosen rung and its confidence are
always recorded.

## Profiles, not tiers

`OT-P0-002` says `strategy verify` reports "which capability tiers it
satisfies." **Implemented as unordered profiles rather than a linear tier
ladder, because the capabilities do not nest.**

The evidence that they do not: `ios-xcuitest` is expected to have
`semantic-tree` (high value, hard to obtain) but *not* `native-recording`
(low value, easy on other transports). A linear ladder would have to place
one below the other, and would then under-report a strategy that has eight of
nine capabilities but is missing a low-numbered one. Any ordering we picked
would be an artifact of the ordering, not of the device.

So a strategy is defined by **its capability set**. Profiles are convenient
labels for recognizable combinations, and carry no authority:

| Profile | Means | Typical use |
|---|---|---|
| `observer` | `screenshot` + `input` | Can watch and drive, but cannot manage applications. |
| `driver` | `screenshot` + `input` + `semantic-tree` | Deterministic targeting. |
| `manager` | `app-lifecycle` plus the modalities it declares | The app is under our control — install, relaunch, upgrade, remove. |
| `full` | All ten optional capabilities | Reference class. |

A strategy may match several profiles, or none. **Matching no profile is not
a failure** — it means the capability set is unusual, and the set is what
`strategy verify` reports. A profile is a summary of probe results and never
a claim in its own right.

`manager` is called out separately from `driver` because it is the profile
the delivery ramps actually need: the mobile conformance chapters
(`install_cold_start`, `process_death_restore`, `update_migration`) are all
app-lifecycle work. A `driver`-only strategy can validate a running app but
cannot run a conformance journey.

## Expected strategy matrix

**Expectation only. The probe decides.** `?` marks a genuine unknown that
implementation will resolve — recorded as unknown rather than guessed.

| Capability | `android-adb` | `android-tv-remote` | `google-cast` | `ios-simctl` | `ios-xcuitest` | `ios-mirror` | `host-desktop` |
|---|---|---|---|---|---|---|---|
| ID + declaration | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `screenshot` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ |
| `input` | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ |
| `media` | ✅ | ✅ | ✅ | ? | ? | ❌ | ? |
| `property` | ✅ | ❌ | ✅ | ? | ? | ❌ | ? |
| `sensor` | ✅ | ❌ | ✅ | ? | ? | ❌ | ? |
| `pairing` | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `push` observation | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `semantic-tree` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ? |
| `app-lifecycle` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ? |
| `permission-control` | ✅ | ❌ | ❌ | ✅ | ⚠️ | ❌ | ➖ |
| `network-control` | ✅ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ✅ |
| `orientation` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ➖ |
| `clipboard` | ✅ | ❌ | ❌ | ✅ | ? | ? | ✅ |
| `file-transfer` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ |
| `native-recording` | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ |
| `device-logs` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ |
| `webview-attach` | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | ✅ |
| **Expected profile** | `full` | `remote` | `cast-state` | `driver`+`manager` | `driver`+`manager` | `observer` | `manager` |

Legend: ✅ expected · ❌ expected absent · ⚠️ partial · ➖ not meaningful · ? unknown

Notes on the non-obvious cells:

- **`ios-xcuitest` / `permission-control` — partial.** WebDriverAgent can
  respond to a permission prompt when it appears, but cannot pre-grant or
  revoke the way `adb pm grant` or `simctl privacy` can. A flow that needs a
  known starting permission state must reinstall.
- **`ios-xcuitest` / `native-recording` — absent.** `simctl recordVideo` is
  simulator-only. Physical iOS has no equivalent, which is the single
  strongest justification for the synthesized fallback.
- **`ios-simctl` / `network-control` — partial.** No `simctl` equivalent;
  conditioning happens on the host, so it affects the whole simulator rather
  than one app.
- **`ios-mirror` — visual/input only, by design.** Mirroring exposes pixels and
  synthetic HID. Everything else is `unsupported` rather than `unavailable`:
  no action makes a semantic tree exist over a mirroring session
  (`../internal/ERROR-HANDLING.md`).
- **`host-desktop` / `semantic-tree` — unknown.** Depends on whether the
  platform accessibility API (AT-SPI on Linux) is reachable and populated for
  the target application. Genuinely varies per app; the probe must answer it
  per target rather than per strategy.

## What `strategy verify` reports

| Field | Content |
|---|---|
| Identity/declaration conformance | Pass or fail. The only mandatory gate. |
| Declared capabilities | What `describe()` claimed. |
| Probed capabilities | What verification proved. |
| Delta | Declared-but-unprovable, reported per capability with the missing prerequisite. |
| Executable step kinds | Derived from the step tables above. |
| Available resolution rungs | Derived from the rung table. |
| Matched profiles | Zero or more. Informational. |

A non-empty delta is the finding the suite exists to produce. The realistic
cause is not malice: an adapter author copies a richer strategy's declaration
and does not notice that one capability never worked on their transport.

## Registry consistency

**Every capability in this registry is exercisable by some flow construct, and
every construct's requirement names a capability in this registry.** That
round-trip is the property this document exists to hold (`D-012`), and it is
checkable: a capability nothing exercises can be declared and probed but never
proven, and a construct requiring an unlisted capability cannot appear in a
gap report.

Exercisable covers three mechanisms — a step **requires** it
(`device.install` → `app-lifecycle`), a resolution **rung** requires it
(`semantic` → `semantic-tree`), or a step's execution **path** selects on it
(`device.record` → `native-recording`). Checking only the first would flag
`semantic-tree` and `native-recording` as orphans, and they are not.

| Capability | Exercised by | Mechanism |
|---|---|---|
| `semantic-tree` | `semantic` rung | rung |
| `app-lifecycle` | `device.install` / `launch` / `stop` / `uninstall` | step |
| `permission-control` | `device.permission` | step |
| `network-control` | `device.network` | step |
| `orientation` | `device.orientation` | step |
| `clipboard` | `device.clipboard` | step |
| `file-transfer` | `device.push` / `device.pull` | step |
| `native-recording` | `device.record` | path |
| `device-logs` | `device.logs` | step |
| `webview-attach` | `bas.*` | step |

Two mismatches surfaced while first assembling the registry. Both are now
closed, and the registry, the step vocabulary, and `OT-P0-001` agree on the
same ten optional capabilities.

1. **Five capabilities had no step verb.** ✅ **Resolved 2026-08-10.**
   `network-control`, `orientation`, `clipboard`, `file-transfer`, and
   `device-logs` were declarable but unreachable. `device.network`,
   `device.orientation`, `device.clipboard`, `device.push` / `device.pull`,
   and `device.logs` joined the vocabulary in
   [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md); see `D-012` for the
   rule this established.

2. **`bas.*` required a capability the PRD did not list.** ✅ **Resolved
   2026-08-10.** Attaching to an application WebView requires
   debuggable-WebView access, which was absent from `OT-P0-001`'s original
   nine. `webview-attach` was added to the PRD and to `DVC-P0-001`, making
   ten. Without it, `bas.*` steps would have had no declarable prerequisite
   and would have failed at execution instead of appearing in the
   pre-execution gap report — precisely the failure `OT-P0-005` exists to
   prevent.

## Cross-References

- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the `strategies` and `flows` domains
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — the `Strategy`, `CapabilityProber`, `Recorder`, and `TargetResolver` seams
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — `unavailable` versus `unsupported` for a missing capability
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — `D-001`, `D-002`, `D-004`, `D-009`
- [`../guides/adding-a-strategy.md`](../guides/adding-a-strategy.md) — implementing against this registry, and reading a `strategy verify` report

The session authority is the read-only control-plane command
`vrooli host desktop-session --session-id <id> --peer-pid <pid> --json`.
It checks logind user/type/activity/lock state and exact process cgroup membership,
with a process start-time check against PID reuse during inspection. Missing or
ambiguous facts fail matching. These observations are not a grant, and the helper
continues to enforce signed lease authority separately. No unlock or host repair
is performed. The helper bootstrap must still supply the authenticated session
ID and X cookie; managed startup is not implemented by this probe.

## Lifecycle-managed helper bootstrap

The `desktop-helper` sidecar builds `api/cmd/desktop-helper` and is enabled only
when `DEVICE_CONTROL_DESKTOP_HELPER_CONFIG` is set. Use the ordinary scenario
lifecycle to launch it. Its process guard rejects direct unmanaged execution.
Unavailable or locked bootstrap retries without claiming capture/input readiness.

The private JSON config has these fields: `version` (1), `surface` (shared
SurfaceRef for the local Bridge host), `session_id`, `display` (number),
`xauthority_file`, `public_key` (base64 Ed25519 public key), `grant_status_file`,
and `state_directory`. All paths are absolute. Config, Xauthority and grant status
must be regular files owned by the helper user with no group/other permissions;
symlinks and oversized files are refused. State lives in a private directory.
The helper selects only the exact local display's MIT cookie and rejects
conflicting matching records.

The admission owner provisions the public-key pin and publishes grant status as
`active` grant IDs, `observed_at`, and `expires_at`; status spans at most one
second and stale/missing status denies authorization. The helper contains no
private signing key. The publisher uses atomic private files, refreshes every
250 milliseconds, excludes competing writers, and clears grants on exit. The
owner snapshot checks existing leases and caps status expiry at lease expiry.
Owner provisioning uses the shared credential authority for the private signing
seed and writes only the public pin to the helper configuration. Reprovisioning
retains the key; a lost credential with an existing pin refuses automatic
replacement. Lifecycle wiring is still unfinished.
An exclusive lock serializes helper bootstrap. A stale socket is removed only
under that lock after a refused connection; an active listener is not displaced.
Private `registration.json` identifies the new helper generation, initial epoch,
session, surface and local socket for the admission owner. It is not a public
surface descriptor or proof of capture/input permission.

The helper persists the accepted grant ID with its lease, never the bearer token.
Lease maintenance checks that ID against current owner revocation state and
checks native login/lock state even when the client is idle. Revocation, missing
status, expiry, or failed session verification revokes the lease and attempts to
release held input. Failed release remains durable for cleanup retry. The Xvfb
integration test verifies button release after grant revocation without a further
helper request; physical lock/revocation acceptance remains required.

Local desktop admission now has an internal Unix-peer owner that signs grants
only after acquiring the existing device lease. Ordinary device/flow sessions
and desktop grants share that exclusion. Actor identity comes from the kernel
peer, and kill/release/expiry revoke the grant. Public SessionService actor
strings do not authorize this path. Protected helper registration is resolved
against current durable epoch and generation. The grant-status publisher is
implemented and tested, as is credential-backed configuration provisioning.
The local owner IPC/proxy is implemented; configured X11 observation launch is now verified.

The API starts optional desktop owner provisioning/publication when
`DEVICE_CONTROL_DESKTOP_OWNER_CONFIG` names a private JSON file. Its fields are
`device_id`, `helper_config_path`, and `helper` (the helper configuration above,
with `public_key` omitted). Set the lifecycle sidecar's
`DEVICE_CONTROL_DESKTOP_HELPER_CONFIG` to that `helper_config_path`. Configuration
failures remain optional availability failures and retry every five seconds.
API shutdown cancels publication before closing the database. The owner mounts
`DesktopOwnerService` only on private `desktop-owner.sock` beside the bootstrap
file. Every request checks kernel peer identity. Open acquires the existing
device lease, publishes its grant before contacting the helper, and keeps the
helper token server-side. Observe/Act require exact session references; Stop
releases helper input and owner exclusion. Failed Open rolls back exclusion.
Normal lifecycle startup with both configuration variables absent is verified;
configured X11 helper launch and observation are now verified.

On the live swarminator X11 session, the managed owner/helper completed
observation-only Open/Observe/Stop and captured a valid 1920×1080 PNG. Stop
left no active grant or helper lease. This establishes live managed capture,
not physical input or full platform acceptance. GDM empty-display Xauthority
records are accepted as cookie wildcards; the destination remains the explicit
Unix display and kernel-peer/logind session binding. Conflicting cookies fail.

The installed CLI exposes `device-control desktop open|observe|act|stop`.
Each command requires `--socket /absolute/path/desktop-owner.sock` and
`--request request.json`; the request uses the canonical DesktopOwnerService
message for that operation. Open names `surface`, `ttl_seconds` (1–600), and
`control` (false for observation only). Observe/Stop name the returned `session`;
Act additionally names `command_id`, `geometry_revision`, and typed `action`.
Observe requires `--output capture.png`, creates a new0600 file, and omits pixels
from `--json` output. Existing output files are never overwritten. The installed
CLI completed live1920×1080 X11 observation and Stop through the configured
owner. CLI native pointer acceptance is now verified against a disposable X11 fixture.

Physical X11 pointer acceptance through the installed CLI verified movement,
one native press/release pair for a duplicated click command, identical durable
receipts, pointer restoration and successful Stop. The fixture was verified
under the pointer before clicking and destroyed afterward. No active grant,
helper lease or pending cleanup remained. Keyboard/Unicode/semantic and broader
platform acceptance remain incomplete.

DesktopOwnerService.Describe returns the protected desktop surface/OS-session
identity over authenticated local IPC. It performs no capture or lease
acquisition. Catalog capability facts remain unknown until owner admission.
Portal can consume this projection with server-side
`PORTAL_DESKTOP_OWNER_SOCKET`; the socket and helper authority are not exposed
in the surface descriptor. Live Portal catalog discovery is verified; opening
and routing a desktop session from Portal remains unfinished.

For a future browser-mediated caller, protected owner bootstrap can pin
`operator_subject`. A forwarded `Authorization: Bearer` token must verify
through the shared owneridentity validator, match that subject, and carry
`device-control:read`; control Open/Act also require `device-control:write`.
The lease actor is the verified account and helper expiry cannot exceed token
expiry. Invalid forwarded credentials never fall back to local Unix authority.
Omitting the bearer cannot reuse an account-bound session. The bare local CLI
path remains kernel-peer authenticated. Portal browser session routing and
live account enrollment are still unfinished.

Account-mediated desktop operations use the distinct DesktopAccountService
namespace. It requires a bearer credential even on the local Unix transport,
then applies the configured operator pin and scope checks. An older owner
that only exposes DesktopOwnerService cannot accept these requests. Portal
DesktopSessionService forwards through this account namespace with no-store
responses and bounded requests; local CLI commands retain OwnerService.
Portal login/session UI and live authorized browser validation remain pending.
