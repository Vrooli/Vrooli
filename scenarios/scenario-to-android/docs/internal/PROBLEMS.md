# Problems — Scenario to Android

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

### 2026-08-10 — The `android-sdk` resource does not exist

**Symptom:** `S2A-P0-006` depends on a governed `android-sdk` resource for SDK, platform-tools, emulator, and system images. No such resource exists in the repository, so `.vrooli/service.json` declares `dependencies.resources` as empty rather than naming a resource that lifecycle resolution would fail to find.

**Root cause:** The resource was identified during design but has not been created. Probed on this host on 2026-08-10: `adb`, `emulator`, and `sdkmanager` are all absent; JDK 17.0.19 and `ffmpeg` are present; `/dev/kvm` exists and is writable by the owner's user.

**Workaround:** The `targets` domain probes for the toolchain and reports `unavailable` with the acquisition next action. Nothing silently assumes a working SDK.

**Real fix:** Create the `android-sdk` resource and declare it in `.vrooli/service.json`, replacing the `resources_rationale` note.

**Owner:** unassigned.

**Refs:** `docs/concepts/INTEGRATIONS.md`, `.vrooli/service.json`, `S2A-P0-006`.

### 2026-08-10 — The `hello-mobile` conformance fixture does not exist

**Symptom:** `S2A-P0-012` requires the ramp to be provable end to end without depending on any product scenario's correctness. `hello-desktop` fills that role for the desktop ramp; there is no mobile equivalent.

**Root cause:** Not yet built. It is shared with `scenario-to-ios`, so neither ramp owns it and it can fall between them.

**Workaround:** None. Until it exists, conformance can only be exercised against a product scenario, which conflates two failure sources.

**Real fix:** Create `hello-mobile` as a minimal self-contained fixture, mirroring `hello-desktop`'s shape and bundle metadata.

**Owner:** unassigned — needs an explicit owner precisely because it is shared.

**Refs:** `scenarios/hello-desktop/`, `S2A-P0-012`.

### 2026-08-10 — Capture redaction policy is a cross-scenario gap

**Symptom:** `S2A-P0-010` requires verified redaction before a capture is referenced in a verdict. The policy that states what is redacted, and who may view an unredacted capture, does not exist.

**Root cause:** `device-control` owns the redaction policy and records the same gap in its own `docs/internal/PROBLEMS.md`. The capability was specified before the privacy policy it depends on.

**Workaround:** Emulator captures carry no personal data, so emulator-only conformance is safe to build now.

**Real fix:** The `device-control` redaction policy lands; this ramp then verifies redaction status before referencing a capture. **No physical-device journey may ship before that.**

**Owner:** `device-control`, with this ramp as a consumer.

**Refs:** `scenarios/device-control/docs/internal/PROBLEMS.md`, `docs/concepts/DATA.md`.

### 2026-08-10 — No Google account state exists, and the verification deadline is close

**Symptom:** Every `distribution` channel except ADB internal, and every `readiness` rung, is currently unsatisfiable. No Play Console registration exists and no developer verification has been performed.

**Root cause:** Not yet done; both are owner actions requiring identity and payment, and neither can be automated.

**Workaround:** The readiness ladder reports each rung honestly with its next action. ADB internal distribution remains available and needs no account.

**Real fix:** Owner completes Play Console registration and developer verification. Note the dates: Play requires target API 36 for new apps and updates from **31 August 2026** (extension to 1 November 2026), and developer verification begins enforcement on certified devices in Brazil, Indonesia, Singapore, and Thailand on **30 September 2026**, going global through 2027 — and it gates sideloading, not only Play.

**Owner:** repository owner.

**Refs:** `S2A-P0-009`, `docs/concepts/INTEGRATIONS.md`.

### 2026-08-10 — Wireless ADB flakiness would surface as failed conformance chapters

**Symptom:** Anticipated, not yet observed. Wireless debugging pairing expires on device reboot and the link is less reliable than USB, so a transport fault presents as a failed journey chapter rather than as a transport error.

**Root cause:** The failure surfaces at the wrong layer — a dropped `adb` connection mid-journey is indistinguishable from an app that stopped responding.

**Workaround:** Support both transports but default release-relevant runs to USB, so release evidence cannot fail for network reasons.

**Real fix:** Classify transport loss as a distinct, non-product failure class in evidence, so a reviewer can tell a flaky link from a broken app.

**Owner:** unassigned.

**Refs:** `S2A-P1-001`, `docs/concepts/FLOWS.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Whole scenario | The domain map in `docs/concepts/DOMAINS.md` describes seven product domains; none of them exist in `api/internal/` yet. The scaffold still carries the template's `notes` example domain. | Expected for a scenario authored documentation-first. It is drift only if code lands in a shape that contradicts the map. | Implement the domains as mapped, then run `template-manager detemplate scenario-to-android`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
