# Problems — Proto Health

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

### 2026-06-11 - Generated notes slice is temporary reference code

**Symptom:** The scaffold still contains the template `notes` CRUD
domain, its SQLite schema, CLI commands, UI pages, and proto files.
Proto validation reports those relocated template contracts as
`proto.template_source` warnings while they remain marked
`@template react-vite/example`.

**Root cause:** `proto-health` was generated from the `react-vite`
template before its real `validation` and `protosurface` domains were
implemented.

**Workaround:** Treat notes as reference-only. Do not extend it for
product behavior. Keep the `@template` marker while it is reference
code. Copy patterns from it when implementing the first real domain,
then delete notes or remove the marker only when the contract is
intentionally adopted.

**Real fix:** Implement `validation` and `protosurface` with API, CLI,
UI, tests, and docs; remove notes code/protos/docs references after the
replacement pattern is green.

**Owner:** proto-health implementer.

**Refs:** `docs/concepts/DOMAINS.md`, `api/internal/notes/`,
`packages/proto/schemas/proto-health/v1/notes/`.

### 2026-06-12 - Implementation proof tier is deferred

**Symptom:** Implementation proof can degrade when `code-facts` or one
of its graph providers is not running.

**Root cause:** `proto-health` intentionally consumes source evidence
through `code-facts` instead of parsing Go or TypeScript directly. That
keeps proto policy separate from analyzer implementation, but the proof
tier now depends on code-facts lifecycle discovery and graph-provider
availability.

**Workaround:** Treat `proto.code_facts_unavailable` and unsupported
proof findings as degraded implementation evidence. Descriptor and
static contract findings remain authoritative.

**Real fix:** Keep code-facts healthy through scenario lifecycle, and
extend code-facts provider coverage when a surface returns unsupported
proof for a language or framework it should understand.

**Owner:** unassigned.

**Refs:** `api/internal/codefacts/`, `api/internal/validation/`,
`docs/internal/SEAMS.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Template reference | `notes` remains while real domains are not implemented. | Keeps extra product-looking surfaces in a meta scenario. | Remove after `validation` and `protosurface` replace its reference value. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
