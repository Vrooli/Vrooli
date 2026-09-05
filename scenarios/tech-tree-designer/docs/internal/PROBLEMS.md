# Problems — Tech Tree Designer

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

### 2026-06-14 - AI strategic analysis deferred

**Symptom:** The old implementation included AI-flavored strategic analysis concepts, but the regenerated scenario has no `ollama` or `openrouter` dependency.

**Root cause:** The deterministic graph and contract-first planning surface must ship before analysis layers are useful.

**Workaround:** Keep AI analysis out of service dependencies and UI/API scope.

**Real fix:** Implement follow-up `tech-tree-designer-ai-strategic-analysis` after graph, planning, ontology, and UI are stable.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P2-002.

### 2026-06-14 - SDA GraphSource deferred

**Status:** Resolved on 2026-06-15. TTD now consumes `scenario-dependency-analyzer` `DescribeInterfaceGraph` through `SDASource`.

**Root cause:** Earlier graph work landed before SDA's Connect graph contract existed.

**Outcome:** The old proto-health source was deleted; SDA is now the canonical live graph source for proto and Go import evidence.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P2-001.

### 2026-06-14 - Scenario scaffold generation from plans deferred

**Symptom:** Planned scenarios can be materialized as proto schemas, but this phase will not scaffold a full scenario from a planned node.

**Root cause:** The generator integration is a separate governance and lifecycle concern from proto materialization.

**Workaround:** `plan materialize` writes validated proto text only; agents run `vrooli scenario generate` separately when implementation starts.

**Real fix:** Design and implement the TTD-to-generator seam after planned proto validation is proven.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P0-002.

### 2026-06-14 - Premature experimental proto import guard belongs in proto-health

**Symptom:** After a planned proto is materialized, other scenarios could import an experimental cross-scenario proto before it is ready.

**Root cause:** Import policy enforcement belongs in proto-health or shared proto governance, not in TTD's planning UI.

**Workaround:** Keep materialization explicit and validation-gated.

**Real fix:** Add a proto-health guard that detects or rejects premature imports of another scenario's experimental cross-scenario proto.

**Owner:** unassigned.

**Refs:** `PRD.md` OT-P0-002.

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
