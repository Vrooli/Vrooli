# Capabilities — Device Control

The canonical registry of device capabilities: what each one is, which flow
step kinds it unlocks, and what each strategy is expected to provide.

`device-control strategy verify <id>` reports against this registry, and the
pre-execution capability gap report (`OT-P0-005`) is computed from the step
table below. Both surfaces read the same rows, so a capability that is not
listed here cannot be required by a step or reported by a probe.

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
