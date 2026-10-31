# Problems — Structure Health

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

### SH-PROB-001 — `TestFleetManifestsValidateCanonicalSchema` fails on non-scenario directories
- **Status**: open
- **Severity**: medium (one failing package test; no runtime effect)
- **Affects**: `internal/packs/configpack/manifestschema`
- **Symptom**: The test requires every directory under `scenarios/` to declare `.vrooli/service.json`, and three do not: `rcl-fixture-positive-1787641498371379912`, `scenarios`, and `template-validation-react-vite-deep`. It reports "validated 121 of 124 scenario directories".
- **Impact**: `go test ./...` is red in this scenario, which masks new failures in the same package.
- **Root cause**: The same distinction fixed in the deployability vocabulary on 2026-09-01 — a *directory* under `scenarios/` is not necessarily a *scenario*. The test enumerates directories where it means scenarios.
- **Blocked on**: An operator decision about the three directories, which is a repo-layout question rather than a test bug: give them manifests, move them out of `scenarios/`, or teach the enumeration to skip a declared fixture set. Deliberately not guessed.


_None yet._

### 2026-08-06 — Work-ladder owner unavailable

**Symptom:** The exact W0 named-mention search returned no swarm-manager goal for `structure-health`.

**Root cause:** This implementation is governed by an active external plan rather than a registered swarm-manager goal, so the W0 contract comparison cannot be verified through the goal registry.

**Workaround:** Treat the active implementation plan as the governing request and preserve this evidence until an owning goal is registered.

**Real fix:** Register an owning swarm-manager goal and rerun the W0 contract comparison.

**Owner:** Unassigned.

**Refs:** `swarm-manager goals list --json` exact named-mention search; `scenario-work-ladder` W0 gate.

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
