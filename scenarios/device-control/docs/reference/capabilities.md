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

## The floor

Three operations, required of every strategy. A component that does not
implement all three is not a strategy.

| Operation | Signature intent | Guarantees |
|---|---|---|
| `observe()` | returns a `Frame` — image plus size, scale, timestamp | A frame stream exists on every strategy by construction. |
| `actuate()` | accepts pointer and key events | Press, release, move, key input. |
| `describe()` | returns a `CapabilityDeclaration` | What else this strategy claims. Claims are verified separately — see [`../internal/SEAMS.md`](../internal/SEAMS.md#why-describe-and-capabilityprober-are-two-seams). |

The floor is deliberately this small because it is the largest set
`ios-mirror` can satisfy (`D-001`). Every shared capability is built against
the floor, which is what makes it work on every strategy rather than only the
well-instrumented ones.

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

## Step kinds and what they require

The most important column is the last one. A **floor-guaranteed** step works
on every strategy, always, with no declaration required — those steps are the
portable core of the flow vocabulary.

### `device.*`

| Step | Requires | Floor-guaranteed |
|---|---|---|
| `device.observe` | — | ✅ |
| `device.tap` | — | ✅ |
| `device.swipe` | — | ✅ |
| `device.type` | — | ✅ |
| `device.key` | — | ✅ |
| `device.record` | — *(see exception below)* | ✅ |
| `device.install` | `app-lifecycle` | — |
| `device.launch` | `app-lifecycle` | — |
| `device.stop` | `app-lifecycle` | — |
| `device.uninstall` | `app-lifecycle` | — |
| `device.permission` | `permission-control` | — |
| `device.network` | `network-control` | — |
| `device.orientation` | `orientation` | — |
| `device.clipboard` | `clipboard` | — |
| `device.push` / `device.pull` | `file-transfer` | — |
| `device.logs` | `device-logs` | — |

### `ai.*`, `flow.*`, `wait.*`, `bas.*`

| Step | Requires | Notes |
|---|---|---|
| `ai.see` | floor + `ai-gateway` visual request kind | Blocked today (`D-005`). |
| `ai.extract` | floor + `ai-gateway` visual request kind | Blocked today. |
| `ai.verify` | floor + `ai-gateway` visual request kind | Blocked today. |
| `ai.decide` | `ai-gateway` text generation | Not blocked — takes prior step results, not a frame. |
| `flow.*` | — | Executor-level. Never touches a device. |
| `wait.*` (fixed duration) | — | Executor-level. |
| `wait.*` (until target appears) | at least one resolution rung | Inherits the rung's requirements. |
| `bas.*` | `webview-attach` | Delegates to `browser-automation-studio`. |

### The `device.record` exception

`device.record` is the **one step that requires no capability yet produces a
capability-grade artifact**, and it is a deliberate exception to "a step
requires its capability" (`D-009`, `OT-P0-012`).

| Strategy declares | Path taken | Evidence records |
|---|---|---|
| `native-recording` | Transport's own capture (`adb screenrecord`, `simctl recordVideo`) | `method: native`, plus effective frame rate |
| *not declared* | Synthesized from the `observe()` frame stream | `method: synthesized`, plus effective frame rate |

Both are legitimate evidence. They support different claims, so the method
and the effective frame rate are structural fields on the result, not
optional metadata — a reviewer must be able to distinguish a 2 fps
reconstruction from a 60 fps native capture without opening the file. See
the provenance rule in [`../internal/SEAMS.md`](../internal/SEAMS.md#the-provenance-rule).

This exception exists because the delivery ramps require video evidence on
exactly the device kinds where native capture is least likely — physical
iPhone via mirroring being the clearest case.

## Resolution rungs

The target resolution ladder (`D-004`) picks the highest rung the strategy
supports. Rung requirements are capability requirements:

| Rung | Requires | Cost | Deterministic |
|---|---|---|---|
| `semantic` | `semantic-tree` | Free, local | Yes |
| `visual-anchor` | floor + a captured reference in the anchor library | Free, local | Yes |
| `vision` | floor + `ai-gateway` visual request kind | Tokens + round trip | No |

Note that **`visual-anchor` and `vision` are both floor-only** — they operate
on frames. So every strategy has at least one deterministic rung available as
soon as an anchor exists, and `vision` is the universal fallback rather than
the universal requirement. The chosen rung and its confidence are always
recorded.

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
| `observer` | Floor only | Can watch and drive, cannot manage. Exploration and vision-driven control. |
| `driver` | Floor + `semantic-tree` | Deterministic targeting. Assertions become meaningful. |
| `manager` | Floor + `app-lifecycle` | The app is under our control — install, relaunch, upgrade, remove. |
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

| Capability | `android-adb` | `ios-simctl` | `ios-xcuitest` | `ios-mirror` | `host-desktop` |
|---|---|---|---|---|---|
| *floor* | ✅ | ✅ | ✅ | ✅ | ✅ |
| `semantic-tree` | ✅ | ✅ | ✅ | ❌ | ? |
| `app-lifecycle` | ✅ | ✅ | ✅ | ❌ | ✅ |
| `permission-control` | ✅ | ✅ | ⚠️ | ❌ | ➖ |
| `network-control` | ✅ | ⚠️ | ❌ | ❌ | ✅ |
| `orientation` | ✅ | ✅ | ✅ | ❌ | ➖ |
| `clipboard` | ✅ | ✅ | ? | ? | ✅ |
| `file-transfer` | ✅ | ✅ | ✅ | ❌ | ✅ |
| `native-recording` | ✅ | ✅ | ❌ | ❌ | ✅ |
| `device-logs` | ✅ | ✅ | ✅ | ❌ | ✅ |
| `webview-attach` | ✅ | ✅ | ✅ | ❌ | ✅ |
| **Expected profile** | `full` | `driver`+`manager` | `driver`+`manager` | `observer` | `manager` |

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
- **`ios-mirror` — floor only, by design.** Mirroring exposes pixels and
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
| Floor conformance | Pass or fail. The only hard gate. |
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
