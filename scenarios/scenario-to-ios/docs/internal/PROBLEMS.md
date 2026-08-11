# Problems — Scenario to iOS

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear. The entries below were recorded during
scenario authoring on 2026-08-10, before any code existed — they are
constraints discovered while designing, not defects introduced by it.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-10 — The only available macOS node has a dated release capability

**Symptom:** `minimouse` is `darwin/amd64` — Intel. It can validate iOS indefinitely, but its ability to produce **submittable** builds expires.

**Root cause:** macOS 26 Tahoe was the last Intel-supporting release, and Xcode 27 dropped Intel entirely. Apple's current App Store Connect floor — Xcode 26 with the iOS 26 SDK, mandatory since 28 April 2026 — is still satisfiable on this node. When Apple raises the floor again, projected around April 2027 from their annual cadence, it will not be.

**Workaround:** Structural, and already reflected in the design: the build host is modelled as a **bridge-node role** rather than a named machine, and validation capability is tracked separately from release capability. A host that can validate but not produce submittable builds reports that as a distinct state.

**Real fix:** Register an Apple Silicon node before the floor moves. Because the host is a role, that is a node registration rather than a rewrite.

**Owner:** repository owner (hardware acquisition).

**Refs:** `S2I-P0-004`, `docs/concepts/ARCHITECTURE.md`.

### 2026-08-10 — Xcode state on `minimouse` is unknown and not exposed by any CLI

**Symptom:** Every P0 target depends on a macOS node with Xcode 26 or later, correct simulator runtimes, and a logged-in GUI session. None of that has been verified on the live node, and no `vrooli-bridge` subcommand reports it.

**Root cause:** Node capability reporting covers OS, architecture, and scopes but not installed developer toolchains. The bridge agent *does* install on macOS as a LaunchAgent bootstrapped into the `gui/<uid>` domain and explicitly documents that SSH-only sessions have no GUI bootstrap — so the prerequisite is understood in code, but its live state is unobserved.

**Workaround:** The `targets` domain probes the node directly rather than trusting a bridge capability field, and reports each failure with its own distinct reason.

**Real fix:** Verify on the live node, then decide whether toolchain probing belongs in this ramp permanently or should be promoted into bridge node capabilities.

**Owner:** unassigned.

**Refs:** `S2I-P0-003`, `scenarios/vrooli-bridge/agent/internal/service/service_install.go`.

### 2026-08-10 — No Apple Developer Program enrollment exists

**Symptom:** Signing, physical-device provisioning, TestFlight, and the App Store are all unsatisfiable. `ios-xcuitest` is also blocked, because WebDriverAgent cannot be signed without an enrollment.

**Root cause:** Not yet done. Enrollment is $99/yr and requires an Apple ID, 2FA, and identity verification; organization enrollment additionally needs a D-U-N-S number. None of it can be automated.

**Workaround:** Two real paths remain open. Simulator conformance needs no enrollment at all. For physical hardware, the `ios-mirror` strategy needs no developer account — which is exactly why it is worth having, despite being pixel-grade.

**Real fix:** Owner enrolls. The readiness ladder reports every dependent rung with enrollment as the next action until then.

**Owner:** repository owner.

**Refs:** `S2I-P0-009`, `S2I-P1-001`, `S2I-P1-002`.

### 2026-08-10 — Mirror-derived evidence must never gate a release, and that rule needs enforcement

**Symptom:** The `ios-mirror` strategy is the first usable physical-iPhone path, but it reads pixels through OCR rather than semantics. It cannot distinguish a correctly-rendered screen from a convincing image of one, and it has no stable element identity to assert against.

**Root cause:** A structural property of the approach — iPhone Mirroring plus synthetic HID events plus OCR — not a defect to be fixed.

**Workaround:** The `releases` domain records promotability per contributing cell, and mirror-derived evidence is marked non-promotable.

**Real fix:** None; this is permanent. The requirement is that the non-promotion rule is *enforced in the gate*, not merely documented — a forbidden-transition case in the release flow contract.

**Owner:** this scenario, at implementation time.

**Refs:** `S2I-P0-010`, `S2I-P1-002`, `docs/concepts/FLOWS.md`.

### 2026-08-10 — Capture redaction policy is a cross-scenario gap, and it is worse here

**Symptom:** `S2I-P0-010` requires verified redaction before a capture is referenced in a verdict. The policy that states what is redacted, and who may view an unredacted capture, does not exist.

**Root cause:** `device-control` owns the redaction policy and records the same gap. The exposure is larger for this ramp than for Android: the `ios-mirror` strategy captures the mirrored **screen**, which includes content outside the app under test — notifications, messages, and 2FA codes that arrive mid-run.

**Workaround:** Simulator captures carry no personal data, so simulator-only conformance is safe to build now.

**Real fix:** The `device-control` redaction policy lands and explicitly addresses whole-screen capture. **No physical-device journey may ship before that.**

**Owner:** `device-control`, with this ramp as a consumer.

**Refs:** `scenarios/device-control/docs/internal/PROBLEMS.md`, `docs/concepts/DATA.md`.

### 2026-08-10 — The `hello-mobile` conformance fixture does not exist

**Symptom:** `S2I-P0-012` requires the ramp to be provable end to end — and, because the build runs on a remote node, it is simultaneously the transport-symmetry proof. There is no mobile equivalent of `hello-desktop`.

**Root cause:** Not yet built. It is shared with `scenario-to-android`, so neither ramp owns it and it can fall between them.

**Workaround:** None. Until it exists, conformance can only be exercised against a product scenario, which conflates two failure sources.

**Real fix:** Create `hello-mobile` as a minimal self-contained fixture, mirroring `hello-desktop`'s shape.

**Owner:** unassigned — needs an explicit owner precisely because it is shared.

**Refs:** `scenarios/hello-desktop/`, `S2I-P0-012`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Whole scenario | The domain map in `docs/concepts/DOMAINS.md` describes seven product domains; none of them exist in `api/internal/` yet. The scaffold still carries the template's `notes` example domain. | Expected for a scenario authored documentation-first. It is drift only if code lands in a shape that contradicts the map. | Implement the domains as mapped, then run `template-manager detemplate scenario-to-ios`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
