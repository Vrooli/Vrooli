# Problems — Lighting Control

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

_None yet._

## Work ladder

- Rung W3 for the bounded Light Desk UI regression pass on 2026-11-11.
- Evidence came from the approved experience contract and focused page regressions. They identified cross-preset draft leakage, cached-policy mutation before persistence, missing failure handling for Off/enrollment, false save-failure feedback after local cleanup, and nested complementary landmarks in loaded pages.
- The affected local gates now pass with 149 UI tests, all configured coverage metrics above 85%, type-check and scoped ESLint. The durable refinement and proof limits are recorded under `ui-regression-owner-20261111` in `../concepts/EXPERIENCE.md`.
- Served desktop/phone comparison and Test Genie admission remain with the supervising delivery effort.
- W3 root-cause repair on 2026-11-14: live Combined policy fell back to gray because its 60-second max age was shorter than substrate check schedules (300–600 seconds). `vrooli-autoheal check list` confirmed the relevant cadence; `infrastructure-manager condition status --projection substrate --json` showed otherwise gradeable readings timestamped several minutes earlier. The default and saved policy are now 900 seconds; trust-saturated signals remain excluded and readings beyond the configured age still fall back.
- The OpenRGB resource also reported healthy while Light Desk's backend was unbound. Its managed source selection was absent from operator state, so ordinary resource start could not reconnect the SDK owner. Selecting the already-installed managed OpenRGB 1.1 source and re-enabling its exact existing lighting selection restored the SDK v6 binding. At that recovery, Combined selected green health at 25%; OpenRGB acknowledged the command and cached mode readback matched. Physical output remained unverified, and the runtime still reported a device-access launch blocker for background startup.
- Follow-up W3 reliability repair on 2026-11-14: a native OpenRGB restart quiesced Light Desk but returned without rebinding it, leaving recovery to the supervisor's 30-second retry. Start/restart now serialize with the durable lighting intent and synchronously bind the exact returned backend before success. OpenRGB stop now uses the same durable disable fence on Linux as on macOS/Windows, so reconcile cannot resurrect a stopped backend. If source metadata is missing, reconciliation restores it only from a matching qualified setup receipt while the resource remains explicitly enabled; host source recovery verifies the recorded version and digest before persisting. Controller discovery also reports unavailable when no devices respond, rather than leaving the backend status green. Focused regressions cover source recovery, binding order, stop semantics, and status consistency.
- W3 UI evidence repair on 2026-11-15: the dashboard omitted effective RGB, the zone card showed only saved static color, and controller-reported cached LEDs were not rendered. It now separates selected policy color, saved zone intent, last apply target, calculated static RGB after brightness scaling, and controller-wide OpenRGB cache values with source/time. The cache is never assigned to a zone or presented as optical proof. Focused UI tests, locale checks, lint, type-check and production build pass; the live GPU reports selected amber `#D99A24` at 25%, calculated static channels `54,39,9` (`#362709`), and four matching controller-cache LED values observed after the last apply. Policy brightness remains 25%; the prior write request was denied by the existing `lighting-control:write` capability, and physical emission is unmeasured.
- W3 effect-fallback clarity follow-up on 2026-11-15: repeated live controller reads over approximately ten seconds returned the same four cached RGB values `(54,39,9)` and mode `Direct`. Durable intent remains `Breathing` at 25%, while the acknowledged last apply is static color because this controller's native effect cannot honor the configured cap. The UI now explicitly says the saved effect is not pulsing when static/Off fallback is acknowledged, and says output may be unchanged when fallback is unconfirmed. The experience contract now matches the implemented static-then-Off fallback; physical color transitions remain unmeasured.
- W3 native-effect cap investigation on 2026-11-15: the GPU's captured OpenRGB v6 `Breathing` descriptor advertises per-LED color input but no brightness field. Before the safe fallback was active, live mode readback matched `Breathing` and its four cached input values matched `#362709`; this confirmed the selected mode and cached input only, not pulse phase or visible output. Adversarial review caught a cap-safety gap because the SDK protocol installs the per-LED buffer in a second packet after activating the effect. The controller now rejects this path below 100% and falls back to capped static output; native brightness and atomic mode-specific palettes remain supported. Partial effect writes now trigger static fallback, then Off if static is not acknowledged. The UI localizes mode readback, labels effect RGB as input rather than payload, and describes controller-cache timing without claiming causation. Uncached control tests, race tests, focused UI tests, type-check, lint, and production build pass. Managed restart is healthy; fresh SDK reads show backend READY, active GPU mode Static, and cached LED values `54,39,9` (`#362709`). Durable intent remains Breathing at 25%; last apply is acknowledged capped static color. Physical output remains unmeasured.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
