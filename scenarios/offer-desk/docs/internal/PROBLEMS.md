# Problems — Offer Desk

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Entries are appended as constraints appear, so they are not rediscovered.

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

### Scaffold health gate is not yet run

The scenario now starts through `make start`, reports healthy, and has API,
CLI, and UI evidence. The remaining completion blocker is live experience
capture reconciliation, not scaffold boot.

### The import is the riskiest step, and it is ordered

`OT-P0-006` moves 22 markdown files into this scenario. The ordering rule is absolute and easy to get wrong under time pressure: **import, verify per-file counts, then delete sources.** The source files are the importer's only input. The team that owns them must be paused first, because a running team writes to the surfaces the importer reads.

### The source catalog is already 19% broken

33 of 174 internal links in `docs/monetization/` do not resolve. `MIG-002` requires those to import as findings rather than be discarded, so the drift stays visible after the move. Expect the first import run to produce a substantial finding list; that is the correct outcome, not a failure.

### The trigger language will be asked to grow

`GATE-002` deliberately admits only declared facts, comparison operators and boolean composition. The first real trigger that wants something richer will feel like a small exception. It is not — that is how a rules engine starts. Route it to `OT-P2-004` (scenario-sourced facts) rather than widening the expression language.

### The generated shell fails two of its own accessibility floors on mobile

Discovered 2026-08-13 during experience validation, and **reproduced identically in both scenarios generated from `react-vite` v1.6.5**, so this is a template defect rather than anything either scenario did:

- `floor-tap-target-size` — the theme control renders ~79×30px on mobile, under the 44px minimum.
- `floor-safe-area-tap-targets` — an interactive target on the dashboard overlaps the mobile unsafe bottom area.

This matters more than a normal placeholder complaint because `docs/START-HERE.md` lists the shell's *"fixed safe-area bottom navigation on mobile"* as durable infrastructure to preserve, not as illustrative content. The floor it is supposed to guarantee is the one failing.

It is invisible until a scenario's experience contract is real enough to be checked against a running UI, which is why a freshly generated scenario reports clean. Fix it in the template rather than per-scenario, or every future scenario inherits it.

**Reproduce:** `make start`, then `experience-manager spec validate <scenario> --json`.

**Filed:** against `template-manager` (`react-vite` v1.6.5), not against this scenario. Do not patch the shell here — a per-scenario fix hides the defect from every future generation.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None recorded._ | | | |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

### 2026-08-13 — requirement evidence is not yet promoted

The implementation and focused suites are green, but the registry remains
`planned` until a fresh comprehensive suite sync and the twelve Level 3
behavioural drills produce durable evidence. The affected Test Genie rerun is
The fresh full Test Genie run terminated with only the `ui-health` provider
failing and no findings payload. Direct experience validation is now clean
after BAS stabilized. The inherited Phase 1 log did not contain the required
start-of-plan protected-tree hashes, so the current end hashes are recorded in
`PROGRESS.md` without claiming a historical comparison.

### 2026-08-14 — final validation boundary and intentional deferrals

The requirement registry is now evidence-backed: `vrooli scenario requirements
validate offer-desk --json` passes at L3, with 18 requirements complete and 6
explicitly planned (including source retirement and later catalog capabilities).
The fresh comprehensive run `20260814-035300-bf1ee069` passed 20/21 phases. The
remaining failure is not an Offer Desk finding: the shared Test Genie `ui-health`
execution provider times out without returning a findings payload. Static-only
UI-health reports zero required findings, and direct experience validation reports
zero findings. The protected-tree end hashes are recorded in `PROGRESS.md`; no
historical start hash was available from Phase 1.
