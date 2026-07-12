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

### 2026-07-11 — react-vite deep validation has remaining provider-boundary debt

**Symptom:** The fresh comprehensive generated-scenario run executes all 20
Test Genie phases but fails dependencies, workflow, and proto.

**Root cause:** The generated scenario itself now passes documentation, unit,
business, and tidiness gates. The remaining failures are provider assumptions
that do not fully model a temporary generated scenario: dependency health reads
the shared repository root, workflow execution cannot discover temporary
runtime ports, and proto health cannot discover the generated Connect handler.

**Workaround:** Keep react-vite quarantined. Inspect the canonical ledger entry
`react-vite.test-genie.deep-validation.phase-results` and retained workspace
`/tmp/vrooli-template-deep-983430887` rather than historical summary rows.

**Real fix:** Repair the external workflow and dependency target-boundary
defects `knw-1783709862952116493` and `knw-1783710133998988533`, and make
proto implementation discovery honor the generated temporary scenario path.

**Owner:** The owning provider scenarios.

**Refs:** `validation-0f1b4951-ed84-458a-ab3b-cd588eddf306`; Test Genie run
`20260712-004751-0742f640`; Template Manager run
`validation-b157ccf9-3a64-4617-935b-0d685876479c`.

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

**Refs:** operator-local plan `template-manager-scenario-owning-the-template-domain.md`.

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
