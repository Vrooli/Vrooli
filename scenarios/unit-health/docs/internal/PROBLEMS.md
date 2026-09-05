# Problems — Unit Health

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

_None yet._

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| `docs/concepts/DOMAINS.md`, `docs/concepts/DATA.md`, `docs/concepts/FLOWS.md`, `docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`, `docs/internal/TESTING.md`, `docs/START-HERE.md`, `docs/QUICKSTART.md`, `docs/manifest.json`, and a handful of lighter mentions (`OBSERVABILITY.md`, `SECURITY.md`, `DECISIONS.md`, `INTEGRATIONS.md`, `MONETIZATION.md`, `DEPLOYMENT.md`, `configuration.md`, `ui-manifest.md`, `ERROR-HANDLING.md`) still describe the `react-vite` template's `notes` CRUD starter domain — fictional `api/internal/notes/` paths, a `NotesService` Connect surface, `notes`/`attachment` tables, a `BlobStore`, and `notes` CLI commands that **do not exist**. Unit Health's real domains are `validation` (analyzer engine) + `health`, with `runhistory` persistence. | docs/architecture drift — these reference docs actively mislead a reader/agent with non-existent file paths, endpoints, tables, and commands. No code/test depends on them. | Rewrite each doc to the real `validation`/`health`/`runhistory`/`discovery`/`executor` contract (mirror the 2026-06-17 SEAMS.md/ARCHITECTURE.md rewrite). Substantial prose pass across ~18 files; best done as its own focused docs sweep. | unassigned — broader than the 2026-06-16 hardening pass scope |

_2026-06-17: `SEAMS.md` and `ARCHITECTURE.md` (the two files the hardening pass explicitly deferred) were fully rewritten from the `notes` starter domain to the real `validation` + `health` + `runhistory` + `discovery`/`executor` seams. While doing so it surfaced that the same `notes` starter residue pervades the rest of the docs tree (row above) — a larger cleanup than those two files, tracked here rather than silently half-done._

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
