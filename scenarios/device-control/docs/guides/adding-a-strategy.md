# Adding a Strategy — Device Control

A strategy is how one class of device gets driven. Adding one should mean
implementing stable identity and a declaration, adding only the optional
modalities the transport can back, and passing a fixed suite — with **no
change to the flow executor, the resolver, or the evidence layer** (`OT-P0-002`).

If you find yourself needing to modify the engine to make your strategy work,
stop: either the contract seam is wrong or your capability belongs in the
declared set. Both are worth raising before writing more code.

## Before you start

Read [`../reference/capabilities.md`](../reference/capabilities.md) first. It
is the canonical registry of capability IDs and the step kinds each unlocks.
Everything below assumes those names.

**The worked example throughout is `ios-mirror`, deliberately.** It is the
weakest strategy we plan to ship — iPhone Mirroring driving synthetic HID
events, reading the screen with OCR, no accessibility tree at all. Learning
against a rich transport like `android-adb` teaches habits that quietly assume
a semantic tree and app lifecycle you may not have. If your strategy can do
more than the example, that is a declaration you add — not an assumption you
inherit.

## The contract

| You implement | You get for free |
|---|---|
| `ID()`, `Describe()`, and any backed modality | Shared execution, bounded waits, evidence chapters, leases, and audit |
| A capability declaration | Gap reports, step admission, profile matching |
| A prober | Honest `unavailable` reporting with next actions |

That narrow contract lets screenless transports participate without claiming a
frame or input channel (`D-001`).

## Step 1 — Implement identity and the modalities you support

### `ID() -> string` and `Describe() -> CapabilityDeclaration`

Return a stable strategy identifier and declare only capabilities this
transport can exercise. Inventory probes verify the declaration and report
missing prerequisites explicitly.

### `observe() -> Frame` (optional)

Return the current screen as an image plus **size, scale, and timestamp**.

Scale is not decoration. For `ios-mirror` the window is a scaled
representation of the device screen, so a coordinate the resolver computes
against the frame is not a coordinate you can post an event at. Reporting
scale is what lets the shared code translate; getting it wrong produces taps
that land near — but not on — the target, which is far harder to diagnose
than a tap that fails outright.

Timestamp matters because a stale frame drives a real device. Return the
capture time, not the request time.

### `actuate(PointerEvent | KeyEvent) -> error` (optional)

Press, release, move, key input. Fail loudly when you cannot deliver.

The `ios-mirror` case is instructive about what "cannot deliver" means in
practice. Events are posted as `CGEvent`s at the HID level, and several
plausible approaches simply do not work: AppleScript clicks are ignored,
unicode key payloads do not register, slow touch-drags are dropped, and
**nothing lands unless the mirroring window is frontmost**. That last one is a
precondition your `actuate` must check and report — not a race you let the
caller lose. An event silently swallowed is worse than an error, because the
flow continues and the evidence looks fine.

## Step 2 — Declare only what you can back

`Describe()` is a claim. Claims get verified against a probe, and the delta is
the headline output of the conformance suite.

**The realistic failure is not dishonesty — it is inheritance.** An author
copies a richer strategy's declaration as a starting point and never notices
that one capability does not work on their transport. The suite exists to
catch exactly that, which is why declaration and probing are separate seams
([`../internal/SEAMS.md`](../internal/SEAMS.md#why-describe-and-capabilityprober-are-two-seams)).

For `ios-mirror` the honest declaration is **only the modalities it can back**. And
this is the subtle part — those absences are `unsupported`, not `unavailable`:

| Capability | Disposition | Why |
|---|---|---|
| `semantic-tree` | `unsupported` | Mirroring exposes pixels. No action makes a view hierarchy exist over that session. |
| `app-lifecycle` | `unsupported` | There is no install channel; tapping a home-screen icon is not `device.install`. |
| `native-recording` | `unsupported` | No transport-level capture. Recording still works — see Step 5. |

Apply the test from [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md):
*is there any action anyone could take that would make this work on this
target?* For all three above the answer is no, so they are terminal. Marking
them `unavailable` would imply a fix exists and send an operator looking for
one.

Contrast with a genuinely fixable gap on the same strategy: mirroring not yet
paired is `unavailable`, and its next action is "pair the device in iPhone
Mirroring."

## Step 3 — Implement the prober

The prober answers what can be **proven right now on this host**, independent
of what `Describe()` claimed. Every negative result carries the exact missing
prerequisite and a next action — an `unavailable` without one is a defect,
not a valid state.

For `ios-mirror`, the probe sequence is roughly:

| Check | On failure |
|---|---|
| Host is macOS Sequoia or later | `unsupported` — iPhone Mirroring does not exist on this OS |
| iPhone Mirroring is installed and a device is paired | `unavailable` → "pair the device in iPhone Mirroring" |
| Accessibility permission granted to the host process | `unavailable` → "grant Accessibility in System Settings › Privacy & Security" |
| Screen Recording permission granted | `unavailable` → "grant Screen Recording in System Settings › Privacy & Security" |
| Mirroring window is present | `unavailable` → "open iPhone Mirroring" |

Note the first row is `unsupported` while the rest are `unavailable`. The OS
version is not something the operator can act on for this machine; every other
row names a specific thing to do.

Probes run at inventory time and may run on a schedule, so keep them cheap.
A probe that wakes a phone every minute is a battery bug, and the device's
owner will notice before we do ([`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md)).

## Step 4 — Register the strategy

Add the adapter under `api/internal/strategies/<id>/` and register it with the
strategy registry. Registration is metadata plus a constructor; it does not
touch the executor.

Declare single-session strategies as such. `ios-mirror` holds one connection
and mirroring pauses when the physical phone is unlocked, so two concurrent
consumers do not fail loudly — they interleave and corrupt each other's
evidence. You do not implement mutual exclusion yourself: declare the
constraint and the lease layer enforces it (`D-006`).

## Step 5 — Run the conformance suite

```bash
device-control strategy verify ios-mirror
```

Read the report in this order:

1. **Floor conformance.** Pass or fail. The only hard gate — a failure here
   means it is not yet a strategy.
2. **Delta.** Declared-but-unprovable capabilities, one line each with the
   missing prerequisite. **A non-empty delta is the finding.** Resolve it by
   removing the claim or fixing the adapter — never by weakening the probe.
3. **Executable step kinds.** Derived, not declared. Confirm it matches what
   you expect; a surprise here usually means a capability ID typo.
4. **Available resolution rungs.** For `ios-mirror`: `visual-anchor` once a
   reference exists, and `vision` when the gateway supports it.
5. **Matched profiles.** Informational. `ios-mirror` matches `observer`.
   Matching no profile is not a failure.

## What you get for free

Do **not** implement any of these in your adapter. If you find yourself
writing one, the boundary is in the wrong place.

| You do not write | Why it already works |
|---|---|
| Video recording | Synthesized from your `observe()` frame stream when you do not declare `native-recording`. Evidence records `method: synthesized` and the effective frame rate (`D-009`). |
| Target resolution | `visual-anchor` and `vision` operate on declared frames. A strategy without observation remains valid but cannot claim those rungs. Declare `semantic-tree` and the `semantic` rung is added on top. |
| Bounded waits | Named policies with upper bounds, enforced by the executor. Never sleep inside an adapter. |
| Evidence chapters, checksums, redaction | Owned by the evidence layer. Return frames; do not write files. |
| Leases, audit, session kill | Owned by `sessions`. Your verbs are only ever called with a held lease. |
| Inference | Routes through `ai-gateway`. An adapter never calls a model. |

The synthesized-recording line is the clearest illustration of why optional
modalities remain useful. `ios-mirror` cannot record video at the transport
level, but when observation is available it can still produce synthesized
evidence; a screenless strategy simply reports that frame-based evidence is
unavailable.

## What you must never do

- **Infer a capability from device kind.** "Android phones usually have X" is
  the inference this scenario exists to refuse (`D-002`). Probe it.
- **Label a synthesized recording as native**, or omit the effective frame
  rate. A 2 fps reconstruction and a 60 fps capture are both legitimate
  evidence supporting different claims.
- **Put device content in an error.** No frame bytes, capture paths, OCR text,
  clipboard values, or logs. Errors carry identifiers and reasons only
  (`D-011`) — the error channel does not pass through redaction verification.
- **Execute a verb without a lease.** Adapters are called through `sessions`;
  never expose a side door for convenience or testing.
- **Call a model provider directly.** There is one outbound inference seam and
  an AST check that enforces it (`D-005`, `OT-P0-007`).
- **Sleep.** Use a named bounded wait. An exceeded bound must be an
  evidence-visible failure, never a longer sleep.

## Checklist

- [ ] `ID()` is stable and `Describe()` declares only capabilities the adapter can back.
- [ ] If implemented, `observe()` returns image, size, **scale**, and capture timestamp.
- [ ] If implemented, `actuate()` verifies its preconditions and reports undeliverable events.
- [ ] Absences are classified `unsupported` vs `unavailable` using the action test.
- [ ] Prober names a prerequisite **and a next action** for every `unavailable`.
- [ ] Probe is cheap enough to run on a schedule against an idle device.
- [ ] Single-session constraint declared if applicable.
- [ ] `strategy verify` reports contract conformance and an **empty delta**.
- [ ] No recording, resolution, evidence, lease, sleep, or inference code in the adapter.
- [ ] Executor, resolver, and evidence layer are unmodified.

That last item is the real acceptance test for this guide. If adding your
strategy required an engine change, the seam did not hold — say so rather
than working around it.

## Cross-references

- [`../reference/capabilities.md`](../reference/capabilities.md) — capability IDs, step map, expected strategy matrix
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — the `Strategy` and `CapabilityProber` seams
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — `unavailable` versus `unsupported`
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — `D-001`, `D-002`, `D-006`, `D-009`, `D-011`
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the `strategies` domain
- [`troubleshooting.md`](troubleshooting.md) — general scenario failures
