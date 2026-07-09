# Problems — Template Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear.

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

### 2026-07-09 — comprehensive maturity gates remain red

**Symptom:** Focused Template Manager phase validations pass, but the default
`vrooli scenario test template-manager` comprehensive run still reports broader
maturity-provider failures.

**Root cause:** The current plan slices intentionally shipped domain behavior
before final hardening. Remaining findings include structure/profile evidence,
architecture ownership, docs snippets, duplication, security advisories, and
other provider-level maturity gaps.

**Workaround:** Use the focused phase validations recorded in the plan evidence
for Phase 1 through Phase 7 while continuing the remaining plan phases.

**Real fix:** Complete Phase 8 through Phase 11, then run and close the full
comprehensive suite and baseline diffs.

**Owner:** Template Manager implementation agents.

**Refs:** `/home/matthalloran8/.vrooli/plans/template-manager-scenario-owning-the-template-domain.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| comprehensive suite | Focused plan slices are green while broader maturity gates remain unresolved. | Template Manager cannot claim final DoD until the default suite is green. | Finish hard cutover and final hardening phases, then update this entry with the closing run id. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
