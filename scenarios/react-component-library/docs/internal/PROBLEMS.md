# Problems — React Component Library

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

### 2026-05-12 — iframe-bridge child wiring deferred (resolved for inspection)

**Symptom:** The live-preview iframe announced first paint via a hand-rolled `postMessage({ type: "preview-ready" })` instead of going through `@vrooli/iframe-bridge/child`.

**Root cause:** Slice 3 (req PR-001/002/003) scope-cut: getting real React execution inside the iframe was the gate, and bundling the bridge child into the per-component esbuild call adds a second resolver setup that wasn't needed for first-paint. Deferring to req-06 lets the bridge work land alongside the inspector UI that consumes it.

**Workaround:** No current user workaround is required. The harness now owns a narrow, data-only inspection wire contract alongside first-paint messages.

**Real fix:** Inspection is implemented, but adopting the full generic bridge remains optional future consolidation work rather than a blocker for the preview workspace.

**Owner:** unassigned (only if full bridge consolidation is later prioritized).

**Refs:** `docs/RESEARCH.md` (Preview harness bundling — Resolved contracts), `api/handlers/preview/static.go`, `requirements/06-element-selection-via-iframe-bridge/`.

### 2026-07-15 — setup JSON has no preview runtime model

**Symptom:** Indexed examples carry optional `setup`, which can look executable
in the component editor even though the harness never evaluates it.

**Root cause:** The examples-as-data index contract was deliberately broader
than the first preview renderer's data-only resolver.

**Workaround:** Use `props` and the documented `$` vocabulary for renderable
examples. Try props is intentionally limited to a shallow JSON-object override.

**Real fix:** Specify a safe, testable setup lifecycle and failure model in a
separately scoped requirement before adding any execution semantics.

**Owner:** unassigned.

**Refs:** `docs/concepts/DATA.md#preview-session-boundary`,
`docs/concepts/FLOWS.md#preview-workspace-experiment`,
`requirements/17-examples-as-data/module.json`.

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

## Work ladder

- Rung: W0 (goal/problem contract comparison — passed after registering the direct objective)
- Evidence: On 2026-08-14 the deterministic named-goal search returned `react-component-library-adoption-integrity` in addition to the adjacent `design-language-foundation` initiative. The new goal states the adoption-integrity capabilities directly: derived styling contracts, token-complete templates and adopters, blocking apply/reapply/reconverge, released-version immutability and source drift, atomic batch/lifecycle operations, and real browser validation on generated scenarios and the fleet. Those capabilities map to the PRD's P0 registry, live React preview, adoption workflow, and test/full-catalog coverage targets (OT-P0-001, OT-P0-003, OT-P0-006, OT-P0-008); the adjacent initiative is not used as the contract for this work.
- Constraint: The goal/PRD comparison now authorizes descending into W1–W3. Preserve the direct plan's acceptance evidence and do not treat the broader design-language initiative as a substitute for it.
- Measured: 2026-08-14.
