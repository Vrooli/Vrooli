# Problems — Personal Planner

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

### 2026-09-18 — notes example pins an experience contract that does not resolve here

**Symptom:** `experience-manager spec validate` reports one error, scoped
to the removable `notes` example domain. The example page pins the library
component `experience-surface@1.0.0`, whose canonical experience contract
does not resolve in this environment.

**Root cause:** The generated `notes` worked example references a library
experience contract (`experience-surface@1.0.0`) that is not resolvable
here. The error is inherent to the shipped example, not to any real
product domain.

**Workaround:** None needed for docs-only work — the error is isolated to
the example domain and does not affect the real `health` domain or the
product design. Treat the single validation error as expected while the
example is still present.

**Real fix:** Remove the notes example. `template-manager detemplate
personal-planner` deletes the worked example (a code-phase step); the
validation error clears once it is gone.

**Owner:** unassigned (resolved during the example-domain-removal
code phase).

**Refs:** `experience-manager spec validate`; `template-manager detemplate`;
the "example-domain-removed" orientation gate below.

### 2026-09-18 — three orientation gates require code and are deferred in this docs-only pass

**Symptom:** The design-language, first-real-vertical-slice, and
example-domain-removed orientation gates are not satisfied.

**Root cause:** This was a documentation-only initialization. The three
gates require product code that has not been written:
- **design-language** — the home surface is still the generated
  placeholder in `ui/src/pages/DashboardPage.tsx`; the Observatory home
  surface is not built.
- **first-real-vertical-slice** — no real product domain has an
  end-to-end API→UI slice yet (only `health` and the `notes` example
  exist).
- **example-domain-removed** — the `notes` worked example is still
  present (see the entry above).

**Workaround:** None — these are intentionally deferred to the code phase.
They are recorded here so a future agent does not mistake the docs-only
state for regressions.

**Real fix:** Build the Observatory design-language home surface, land the
first real vertical slice, and run `template-manager detemplate
personal-planner` to remove the example domain.

**Owner:** unassigned (product code phase).

**Refs:** `ui/src/pages/DashboardPage.tsx`; `template-manager detemplate`;
[`PROGRESS.md`](PROGRESS.md) for the remaining-work summary.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ | Docs and the generated scaffold are consistent; no product code exists to drift from the intended capability map. | None. | n/a — add entries if code and the DOMAINS/DATA/FLOWS map diverge. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
