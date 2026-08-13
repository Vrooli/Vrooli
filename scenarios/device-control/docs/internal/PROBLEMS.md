# Problems — Device Control

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

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

### 2026-08-10 — Redaction policy is undefined

**Symptom:** Nothing states what is redacted from a captured frame by
default, or who may view an unredacted capture. `DVC-P0-008` requires
redaction status to be *verified* before a capture leaves the producer,
but the policy that verification checks against does not exist.

**Root cause:** The capability was specified before the privacy policy
it depends on. A screen frame from a personal phone can contain
messages, authentication codes, tokens, financial detail, or health
data — none of which this scenario asked for and all of which it can
incidentally record.

**Workaround:** None needed yet; no strategy captures from a real device
today. Do not treat that as safety — it is only sequencing.

**Real fix:** Define the default redaction policy and the unredacted-view
authorization rule, then make `EvidenceSink` verify against it. A
default-permissive answer is how a personal device's screen ends up in a
shared evidence store.

**Owner:** unassigned.

**Refs:** `SECURITY.md` (Open Security Decisions 1, Security Gaps),
`../concepts/DATA.md` (Retention And Deletion — capture artifacts),
`DVC-P0-008`.

---

### 2026-08-10 — The lease invariant has an out-of-band bypass

**Symptom:** `SECURITY.md` states that no verb reaches a strategy
without both a bridge-authorized reach and a held lease. That holds for
verbs routed through this scenario. It does not hold for a consumer that
calls `vrooli-bridge` dispatch directly — such a caller can reach a host
node, and therefore its attached devices, without ever acquiring a
lease.

**Root cause:** Exclusivity is enforced here while reach is enforced in
bridge, so the lease governs only this scenario's own call path.
`scenario-to-desktop/api/bridgevalidation/client.go` already dispatches
directly to bridge today, so the bypass is an existing path rather than
a hypothetical one.

**Workaround:** None. The failure mode is the one `D-006` describes —
two consumers do not collide loudly, they interleave and quietly corrupt
each other's evidence.

**Real fix:** One of two, and it is a design decision rather than a bug
fix. Either move the lease check to the bridge dispatch gate for
device-scoped verbs — bridge already owns the allowlist, scope, and
audit gate, with bridge holding the lease *record* and this scenario
keeping lease *semantics* — or narrow the claim in `SECURITY.md` to
this scenario's own verbs and name separately what prevents out-of-band
device access. Resolving it may amend `D-008`.

**Owner:** unassigned — needs an owner decision before `DVC-P0-004`
is implemented.

**Refs:** `SECURITY.md` (Auth And Authorization — key invariant),
`DECISIONS.md` `D-006`, `D-008`, `DVC-P0-004`.

---

### 2026-08-10 — `ios-mirror` non-promotability has no mechanism

**Symptom:** `ios-mirror` is described as "explicitly non-promotable for
release evidence" in `OT-P1-003` prose and nowhere else. No field, no
capability, and no gate expresses it, so a conformance run on
`ios-mirror` would emit evidence structurally indistinguishable from
release-grade evidence.

**Root cause:** The property was stated as documentation rather than
built into the evidence shape. `D-009` already established the correct
pattern for exactly this class of problem — recording provenance is a
structural field on the result *because* a separate field "would
eventually be forgotten … while every test still passed." The same
argument applies here and was not carried across.

**Workaround:** None. `ios-mirror` is P1 and unimplemented, so the
window to fix this cheaply is now.

**Real fix:** Make it structural — an `evidence_class` (or equivalent)
on the strategy declaration, carried onto every `TargetVerdict` the
strategy produces, that a ramp's release gate reads. OCR reads pixels
rather than meaning, so it cannot distinguish a correctly rendered
screen from a convincing image of one, and it has no stable element
identity to assert against.

**Owner:** unassigned — decide before `DVC-P1-003` starts.

**Refs:** `OT-P1-003`, `DECISIONS.md` `D-009`,
`../reference/capabilities.md` (Expected strategy matrix).

---

### 2026-08-10 — No usefulness threshold for synthesized recordings

**Symptom:** `D-009` requires evidence to record the capture method and
effective frame rate, but nothing states the rate below which a
synthesized recording stops supporting a claim. A 1–2 fps reconstruction
satisfies "video evidence exists" while proving nothing about a
transition animation or a keyboard-avoidance defect.

**Root cause:** The provenance rule answers *which path produced this*
without answering *is the result good enough for the claim being made*.
`scenario-to-desktop`'s contract has an explicit counterpart — a visual
pass requires a "useful decoded MP4" — that was not carried over.

**Workaround:** None. Most likely to bite on `ios-mirror`, where
synthesis is the only path and the transport is inherently slow.

**Real fix:** Declare a minimum effective frame rate per claim class and
report a below-threshold capture as `degraded` rather than passing it
quietly. `keyboard_avoidance` and `background_resume` are the mobile
conformance chapters most sensitive to this.

**Owner:** unassigned.

**Refs:** `DECISIONS.md` `D-009`, `OT-P0-012`, `DVC-P0-012`,
`../reference/capabilities.md` (The `device.record` exception).

---

### 2026-08-10 — Experience floor errors are inherited from the template shell

**Symptom:** `experience-manager spec validate device-control` reports FAILED.
The only SEVERITY_ERROR findings are two floor claims, on every page that has a
live capture:

- `floor_tap_target_size` — the Theme control measures 29px tall at 390×844,
  under the 44px minimum.
- `floor_safe_area_tap_targets` — the bottom-nav Dashboard item sits flush at
  y=790 h=54 in an 844px viewport, overlapping the mobile unsafe bottom area.

**Root cause:** Not this scenario's code. `ui/src/layout/AppShell.tsx` and
`ui/src/layout/BottomNav.tsx` are byte-identical to the `react-vite` template,
and the nav delegates to the canonical `ui/src/components/ui/bottom-nav.tsx`,
which already applies `pb-safe` and `touch-target`. The safe-area finding looks
like `env(safe-area-inset-bottom)` resolving to 0 in the headless capture
browser rather than a CSS defect; the 29px Theme control is a genuine
tap-target defect. Both surfaces are shared, so every scenario generated from
this template inherits them.

**Workaround:** None needed for spec work — the warnings-only findings
(`state_missing`, `capture_unavailable`) are unaffected and the specs are
complete regardless. But the experience phase gate stays red until these are
fixed.

**Real fix:** Owned by the template and the component canon, not here. Patching
`ui/src/layout/*` locally would fix one scenario, leave every other generated
scenario broken, and register as template drift.

**Owner:** template-manager / react-component-library. Filed to scenario-qa as
`knw-1786386038390145690`
(`bug-inbox/code-defect/react-vite-template-shell-fails-experience-floor-tap`),
flagged `speculative-cause` because the safe-area half is a hypothesis about
the capture environment rather than a confirmed CSS defect.

**Refs:** `ui/src/components/ui/bottom-nav.tsx`,
`templates/scenarios/react-vite/ui/src/layout/`, `experience/pages/fleet.json`.

---

### 2026-08-10 — `ai-gateway` visual-understanding route (resolved)

**Symptom:** The `vision` resolution rung was blocked while `ai-gateway`
accepted only text-oriented inference requests.

**Resolution:** ai-gateway now accepts image attachments on the
provider-neutral inference contract and exposes the `locate.visual` role.
device-control calls that role through the generated Connect client and
normalizes the result into flow evidence.

**Unavailable behavior:** A missing route returns typed
`vision_route_unavailable` evidence. If a caller has an existing visual
anchor, the resolver may fall back to that lower rung. **Do not** add a
direct provider client — that is the coupling `browser-automation-studio`
already has in `playwright-driver/src/ai/vision-client/`, and taking it a
second time would make the gateway boundary fictional (`D-005`).

**Evidence:** Unit fixtures cover normalized responses, fallback, typed
unavailability, and caller-owned downscaling. The live flow proof is recorded
in the multimodal inference plan log.

**Owner:** ai-gateway owns the inference route; device-control owns the
caller contract and ladder behavior.

**Refs:** `DECISIONS.md` `D-005`, `../concepts/INTEGRATIONS.md`,
`SECURITY.md` (Security Gaps), `DVC-P0-007`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Resolved during the strategy-floor implementation

The original redaction-policy, iOS-mirror non-promotability, and synthesized
recording-threshold findings are resolved by `docs/concepts/REDACTION.md`, the
producer-side `internal/evidence` verifier, the structural `evidence_class`
and `promotable` strategy fields, and the 5 FPS usefulness threshold.

## Work ladder

- Rung: W0
- Evidence: the deterministic `swarm-manager goals list --json` name/title/description search found no goal naming `device-control`; the active work is recorded in Plan Manager instead, so the goal-to-PRD comparison cannot be performed.
- Blocker: W0 is unverifiable until an approved swarm-manager goal names this scenario, or the contract workflow explicitly records the plan as the governing goal artifact.
- Measured: 2026-08-12

## Browser E2E validation notes

- 2026-08-12: BAS direct execution successfully completed the dashboard identity journey before the later transport drop and the onboarding/re-probe journey (`1475435b-13f5-4122-afed-9ae9c7b3e735`). The managed browser DOM showed the real Galaxy A03s stable ID, serial, model, Android version, USB transport, and capability inventory.
- 2026-08-12: The flow validation/run journey completed all six nodes after adding the missing flows-page root selector (`6934c56c-ecd1-4685-aa00-dfc39b36bfdd`). The journey now asserts that no operator-visible error is present after dispatch.
- BAS emits a 404 for its optional `__vrooli_recording_event__` callback during mutating browser steps. The device-control API calls themselves returned the expected 200 responses when the phone was attached; the callback warning is owned by the BAS capture environment.
- Standalone BAS CLI runs require explicit substitution of `@scenario/self` and `@selector/...` tokens. The authored cases retain those portable tokens for registry/workflow-health execution; direct CLI validation uses the selector manifest as the resolver.

## Android physical conformance status

- 2026-08-12: The scenario now exposes a provider-neutral, twelve-chapter Android physical conformance plan and emits a `common.v1.TargetVerdict` with `DEVICE_KIND_PHYSICAL`, evidence references, and a serial-bearing detail string. The API and CLI fail closed with `unavailable` when the fixture APK is absent.
- `fixtures/hello-mobile.contract.json` records the shared fixture contract, and `scenario-to-android/fixtures/hello-mobile` now contains the buildable platform-only source project. The generated debug APK is intentionally ignored rather than committed; a build on 2026-08-12 produced the declared package `com.vrooli.hello.mobile`, but that artifact has not yet been installed on a physical device.
- The conformance gate remains open until the fixture and two physical Android devices are available for the required chapter execution.
- Targeted validation is green: API/CLI Go suites, UI 36-file/145-test suite, type-check, lint, managed build/restart, JSON/endpoint generation, the hello-mobile APK build/package inspection, and requirement structure validation. Comprehensive Test Genie evidence is currently degraded: run `20260812-183608-51052a99` found the initial undeclared CLI commands, run `20260812-184226-7d007c1d` found and exposed manifest schema errors, and run `20260812-185005-fbd1e28e` aborted because its legacy runner predates canonical terminal snapshots. These are recorded as infrastructure/contract findings, not treated as passing baseline evidence.

### 2026-08-12 — Cross-platform honesty and wireless seams implemented

The cross-platform strategy declarations now distinguish unsupported host OSes
from unavailable prerequisites, and expose supported host platforms in the
strategy API and CLI. USB onboarding uses native inspection commands on Linux,
macOS, and Windows; bounded sampling reports link flaps and cable guidance.
Android ADB promotion preserves the stable device identity while switching to a
verified wireless endpoint. Lease listings and terminal session responses omit
bearer tokens, and the physical smoke fixture requires a changed frame.

The dashboard now exposes onboarding from the zero-device state. The Windows
host-desktop path remains explicitly unsupported until a real capture/input
implementation is verified; PowerShell presence is not treated as capability
proof.

**`network-control` is unavailable on the Galaxy A03s.** `adb shell svc wifi`
fails on this device image, which blocks the `offline_transition` conformance
chapter from running on the only physical device currently available.

**Owner:** unassigned.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
