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

### 2026-08-13 — Galaxy A03s native screenrecord can capture a black display body

**Symptom:** `adb screenrecord` can return a valid H.264/MP4 file while the
decoded display body is uniformly near black. During the reproduced run,
`screencap` showed the visible launcher and the power probe reported awake, but
the recording body measured YAVG=16 and YMAX=16. A separate attempt recorded
zero frames when SurfaceFlinger reported the display power mode off.

**Root cause:** The Samsung Android 13 image's native screenrecord path is not
currently correlated with the actual display composition. `mWakefulness=Awake`
and the power-manager state are insufficient evidence that screenrecord can
capture visible pixels.

**Workaround:** Device Control now samples decoded recording content outside
the status/navigation bands and fails the recording before publishing an
evidence reference when the body is uniformly near black. Use `device state`
and `screencap` as separate diagnostics; do not treat a valid MP4 container as
proof of a visible recording.

**Real fix:** Repair or replace the Android native capture path on this image,
or add an explicit screenshot-backed fallback that is labelled synthesized and
subject to its claim-class threshold. The fallback must remain within the
Device Control evidence path and must not silently relabel a degraded capture
as native.

**Owner:** device-control strategy owner.

**Refs:** `api/internal/evidence/recording.go`,
`api/strategy/androidadb/androidadb.go`,
`DVC-P0-012`, live flow `8f60f7cb-57c2-4739-ae88-eb3074035fb4`.

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
