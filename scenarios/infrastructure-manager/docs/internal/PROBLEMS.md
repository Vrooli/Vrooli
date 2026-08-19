# Problems — Infrastructure Manager

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

### P-001 — Setpoint confidence is `SKETCH` because no obligation list exists

**Status:** open · blocks honest coverage reporting · **owner:** the `infra-health` team, not this scenario

The 14 authored target kinds in the sensor map are at *operation* granularity.
The **obligation** list they should derive from — what the team must be able to
do, from which the targets follow — has never been written. `TARGET_MODEL.md` § 4
is direct that conflating the two is the most common modelling error and that a
denominator built this way *"can never rise above `sketch` confidence."*

This is judgment that must be done by the team. It cannot be derived here, and
building the board first does not resolve it — it just means every ratio the
board reports carries `SKETCH` until it is done.

**Revisit trigger:** the team authors six to ten obligations derived from `I2`.

### P-002 — Every current-state reading is stale; the team loop is paused

**Status:** open · **owner:** the operator

The `infra-health` loop has been `paused-manual` since 2026-07-24. Every
current-state value in the plan of record — the 1,058/24h alarm flood, the
~55-running/~25-supervised diff — reflects the pause, not the platform. Building
an observer against them would bake a stale reading into the scenario's first
evidence.

**Revisit trigger:** resume the loop and run the resume-protocol scan before the
first vertical slice reports anything as `measured`.

### P-003 — Two of four trust rules cannot be computed until roadmap Gap 10 ships

**Status:** open · degrades safely · **owner:** `vrooli-autoheal`

Ghost and shelved verdicts need `vrooli-autoheal check reconcile` and
`check shelve`, neither of which exists. The scenario computes saturated and
unit-mismatch, and marks the rest `UNTRUSTED` rather than assuming `VALID`, so
the gap degrades conservatively — but declared blindness is wider than it will
be, and the shelving record is a hand-maintained artifact in the meantime.

Gap 10's priority signal has already fired twice and it is top of the team's own
sensor queue.

**Revisit trigger:** Gap 10 ships.

### P-004 — Generated from a template whose registry status is `quarantined`

**Status:** open · unresolved upstream · **owner:** `template-manager`

`template-manager registry list --kind scenario` reports both scenario templates
as `quarantined`, while Template Manager's own `PROGRESS.md` records react-vite
1.6.5 passing deep validation on 2026-07-12 with the registry active. Generation
succeeded and the scaffold validates, so this is a status discrepancy rather than
an observed defect — but it is unresolved and should not be discovered again by
the next agent.

**Revisit trigger:** confirm with Template Manager whether the flag is stale
before the first vertical slice. If the quarantine is real, re-evaluate the
template choice.

### P-005 — Three extension rules are prose with no mechanical enforcement

**Status:** open · **owner:** this scenario

Extension rules 3 (never persist a verdict), 4 (never cache the setpoint or a
derived set) and 6 (no actuation path) are the mitigations for the three
highest-impact risks in `SECURITY.md`, and all three are currently enforced only
by review. Each is the kind of invariant a well-meaning caching or convenience
change erodes silently.

**Revisit trigger:** first vertical slice. Add architecture tests asserting that
`targets` and `supervision` own no schema, and that no dependency client exposes
a mutating verb.

### P-006 — `secrets-manager` exposes no typed read surface

**Status:** open · blocks a target row · **owner:** `secrets-manager`

Secrets availability is load-bearing for scenario start and has no target row
anywhere in the team's plan of record. The scenario ships a `cli/` directory but
**no `cli/manifest.json`**, so it is unbindable and exposes no typed read this
board could consume.

**Revisit trigger:** `secrets-manager` ships a manifest and a health/status verb.
Do not compensate for the gap here with log scraping — that would breach the
typed-read boundary that keeps credentials out of the reading store.

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
