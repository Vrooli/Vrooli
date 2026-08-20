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

The 14 authored target kinds are at *operation* granularity. The **obligation**
list they should derive from — what the team must be able to do, from which the
targets follow — has never been written. `TARGET_MODEL.md` § 4 is direct that
conflating the two is the most common modelling error and that a denominator
built this way *"can never rise above `sketch` confidence."*

The 2026-08-19 projection model narrows this substantially without closing it.
The ten projections and their cell grids **are** the structural half of the
missing obligation list: they name what the platform must be able to demonstrate,
per layer, rather than which command happens to emit a number. What remains is
the team's judgment call that the ten projections are the right ten — and each
owner authoring the cells inside its own space doc. Until both exist, every
denominator reports `SKETCH`, which is stated on every ratio rather than rounded
away.

**Revisit trigger:** the team confirms the projection set against `I2`, and each
control layer authors its `docs/spaces/<projection>-space.md`.

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

**Status:** closed 2026-08-20 · **owner:** `vrooli-autoheal`

Ghost and shelved verdicts need a two-direction `check reconcile` and a
shelve-with-expiry verb on `vrooli-autoheal`, neither of which exists. The
scenario computes saturated and unit-mismatch, and marks the rest `UNTRUSTED`
rather than assuming `VALID`, so the gap degrades conservatively — but declared
blindness is wider than it will be, and the shelving record is a hand-maintained
artifact in the meantime.

**Resolution:** the typed autoheal checks surface now exposes both-direction
reconcile, saturation, and mandatory-expiry shelves; the condition source
consumes those reads and assigns the closed trust vocabulary conservatively.

**Evidence:** `scenarios/vrooli-autoheal/api/internal/handlers/typed_connect.go`
and the autoheal checks/reconcile tests.

### P-004 — Generated from a template whose registry status is `quarantined`

**Status:** open · escalated upstream · **owner:** `template-manager`

`template-manager registry list --kind scenario --json` was re-run on 2026-08-20
and reports `react-vite` 1.6.5 as `quarantined` (and also reports the other
scenario template as quarantined). Template Manager's own `PROGRESS.md` records
react-vite 1.6.5 passing deep validation on 2026-07-12, so the registry state is
still inconsistent with the last known validation evidence. Generation succeeded
and the scaffold validates, but this run cannot independently establish that the
quarantine is stale or safe to override.

The finding is escalated rather than closed: the plan's provenance gate remains
open until Template Manager records a definitive resolution. Product work may
continue only against the already-generated scenario while this provenance
question is tracked.

**Revisit trigger:** Template Manager publishes whether the flag is stale or
genuine. If genuine, re-evaluate the template choice before claiming the
scenario's provenance is healthy.

### P-005 — Three extension rules are prose with no mechanical enforcement

**Status:** open · **owner:** this scenario

Extension rules 3 (never persist a verdict), 4 (never cache the setpoint or a
derived set) and 6 (no actuation path) are the mitigations for the three
highest-impact risks in `SECURITY.md`, and all three are currently enforced only
by review. Each is the kind of invariant a well-meaning caching or convenience
change erodes silently.

**Revisit trigger:** first vertical slice. Add architecture tests asserting that
`coverage` owns no schema, that no setpoint write path exists, and that no dependency client exposes
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

### P-007 — `vrooli-autoheal` has no typed read surface at all

**Status:** closed 2026-08-20 · **owner:** this scenario's plan

`vrooli-autoheal` backs three of ten projections (`supervision`, `availability`,
`recovery`), owns the check registry, and answers two of four trust rules.

The typed surface now preserves the existing checks, actions, incidents,
healing, and Measures domains rather than introducing a minimal shim.

**Evidence:** `packages/proto/schemas/vrooli-autoheal/v1/`,
`scenarios/vrooli-autoheal/api/internal/handlers/typed_connect.go`, and
`scenarios/vrooli-autoheal/cli/manifest.json`.

**Refs:** `scenarios/vrooli-autoheal/api/main.go`, `api/internal/{checks,healing,incidents}/`

### P-008 — No control layer has authored a reliability space doc

**Status:** substantially resolved 2026-08-20 · **owner:** each control layer

The coverage denominator is designed to be read from each owner's
`docs/spaces/<projection>-space.md` through its `space --projection <p> --json`
verb — the same contract `search-hub`, `test-genie`, `prompt-manager` and
`program-runtime` already implement for `meta-optimization-manager`. The six
scenario owners and the control-plane exception now author the required spaces.

The denominator still reports `SKETCH` until the obligation list and operator
confidence are approved; that is an intentional confidence state, not a
missing transport. The scenario must not assert the cell sets itself: that
would be a roster in all but name and would go stale on owner change.

**Evidence:** owner `docs/spaces/` documents and the shared flat `space` CLI
projection contract.

### P-009 — `vrooli capacity` cannot be read typed, by construction

**Status:** open · accepted, fenced · **owner:** the control plane

The capacity projection's sensor is `vrooli capacity reconcile|recommend`, and
`vrooli capacity` lives in the repo-root `internal/capacity` package. Go's
`internal/` visibility rule means a scenario — a separate module — cannot import
it, and `api-core/discovery` resolves scenario ports, not control-plane
subcommands. So this one source is read as a bounded CLI subprocess.

This is accepted rather than fixed, but it is **fenced to one named source**. A
second CLI read is a design smell and should be challenged in review, because
the failure mode it reintroduces — a hung owner stalling the board — is the one
`meta-optimization-manager` migrated away from CLI reads to escape.

**Real fix:** the control plane exposes a typed local surface for capacity
reads. Until then, the subprocess carries the same 10s deadline and
`UNAVAILABLE` degradation as every typed source.

**Refs:** `internal/capacity/`, `packages/api-core/discovery/resolve.go`

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
