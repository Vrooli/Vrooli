# Problems — Meta-Optimization Manager

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

> Note: fleet-level gaps that this scenario *measures* (e.g. MISSING Answer
> cells, the Validate experience-ladder, the SKETCH Guide denominator) are
> **domain data**, not this scenario's debt — they live in the space docs and
> the runtime gaps registry, not here.

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

### 2026-06-24 — `space --projection` verb does not exist on any owner yet

**Symptom:** `coverage` cannot read a real denominator for Answer/Validate/Guide; the readiness scoreboard has no machine source.

**Root cause:** The shared read contract (a `space --projection <p> --json` verb on search-hub/test-genie/prompt-manager) is a design dependency that has not been built. The denominators exist only as the markdown space docs today.

**Workaround:** None for live coverage; the space docs + this scenario's design stand, but `status` is unimplementable until the verb ships.

**Real fix:** Add the `space` verb (schema-validated JSON) to each owner as part of the implementation plan; `coverage` reads it.

**Owner:** unassigned (next: implementation plan).

**Refs:** `../concepts/ARCHITECTURE.md` (Contracts And Data Flow); the three `*/docs/spaces/*-space.md` docs.

### 2026-06-24 — convergence coordinated-edit walkthrough is unproven mechanization

**Symptom:** The add/delete coordinated-edit count (a reference-pattern-fitness sub-lens) has no existing programmatic engine.

**Root cause:** That sub-lens is a skill (judgment) today; mechanizing the *counting* is genuinely new work and may be fragile.

**Workaround:** Mark its findings lower-confidence until proven against a frozen template fixture.

**Real fix:** Build + validate the walkthrough counter; promote its confidence once it matches hand-audited counts.

**Owner:** unassigned.

**Refs:** `docs/agent-system/REFERENCE_PATTERN_FITNESS.md`; requirement MOM-CONVERGENCE-001.

### 2026-06-24 — example `notes` domain still present

**Symptom:** The template's fenced `EXAMPLE-DOMAIN:notes` files are still in the tree (orientation `example-domain-removed` fails).

**Root cause:** Documentation-first — the first real domain (`coverage`) is not yet built, so the example cannot be removed.

**Workaround:** Leave it; it keeps the scaffold green.

**Real fix:** After `coverage` is green, run `vrooli scenario detemplate meta-optimization-manager`.

**Owner:** unassigned (implementation phase).

**Refs:** `docs/START-HERE.md` Gate 7.

### 2026-06-24 — prd-control-tower LLM generation returned HTTP 500

**Symptom:** `prd-control-tower prd generate … --publish` failed with `api error (500)`.

**Root cause:** The generation LLM backend was unavailable (env); the deterministic validator (`prd validate`) was healthy.

**Workaround:** PRD hand-authored to the canonical template and validated deterministically (0 violations).

**Real fix:** None required for this scenario; if a regeneration is wanted later, retry once the LLM backend is healthy.

**Owner:** unassigned.

**Refs:** `DECISIONS.md` (2026-06-24 documentation-first row).

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
