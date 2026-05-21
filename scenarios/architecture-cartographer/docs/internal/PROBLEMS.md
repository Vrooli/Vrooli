# Problems — Architecture Cartographer

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

### 2026-05-21 — go-code-graph and typescript-code-graph do not exist yet

**Symptom:** Cartographer cannot be implemented end-to-end because the two scenarios it depends on for source-code parsing have not been created.

**Root cause:** Layered scenario architecture (see [`DECISIONS.md`](DECISIONS.md), entry 2026-05-21) requires graph extraction to live in language-specific scenarios. Those scenarios are scheduled but not built.

**Workaround:** None at the implementation level. PRD and requirements can be authored without the dependencies existing; implementation must wait until at least `go-code-graph` ships.

**Real fix:** Build `go-code-graph` and `typescript-code-graph` per launch sequencing in `PRD.md`. Cartographer integration adapters then consume them via Connect-RPC.

**Owner:** Unassigned. Same team that builds the cartographer is the most likely candidate.

**Refs:** [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — Intentional Deviations; [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — Scenario Dependencies; PRD.md — Launch sequencing step 1.

### 2026-05-21 — `vrooli scenario requirements validate` and `lint-prd` are broken by an unrelated test-genie build error

**Symptom:** Running `vrooli scenario requirements validate architecture-cartographer` or `vrooli scenario requirements lint-prd architecture-cartographer --json` returns `go build failed in scenarios/test-genie/cli: exit status 1`.

**Root cause:** Unknown — test-genie CLI build is failing in the user's environment, blocking the wrapping `vrooli scenario` commands that shell out to it.

**Workaround:** Use `prd-control-tower prd validate architecture-cartographer --json` and `prd-control-tower requirements validate architecture-cartographer --json` directly — both ran clean during scenario initialization on 2026-05-21.

**Real fix:** Diagnose the test-genie CLI build failure (likely `go mod tidy` needed, per the error message) outside the architecture-cartographer scope.

**Owner:** Unassigned — global tooling issue, not cartographer-specific.

**Refs:** Initialization session 2026-05-21.

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
