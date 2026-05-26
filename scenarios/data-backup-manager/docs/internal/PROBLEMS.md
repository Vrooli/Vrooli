# Problems — Data Backup Manager

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

### 2026-05-26 — Inherited react-vite template test-compile defect (notes module)

**Symptom:** `api/handlers/notes/module_test.go` does not compile. It
calls `db.NewSQLite(t)` (which returns `*sql.DB`) and passes the result
into `notes.ModuleWithBlobStore(...)`, whose first parameter is
`*database.RoutedDB`. The types do not match, so the test package fails
to build.

**Root cause:** A defect in the upstream `react-vite` scenario template,
not in this scenario's design. The same mismatch is present in
`templates/scenarios/react-vite/`, so every scenario scaffolded from
that template inherits it. The seam (`ModuleWithBlobStore`) expects a
`*database.RoutedDB`; the test helper (`NewSQLite`) hands back a raw
`*sql.DB`.

**Workaround:** None needed in the short term — the `notes` domain is
example-only and slated for removal (see `PROGRESS.md`). Removing the
`notes` domain removes the failing test.

**Real fix:** Fix the template so `db.NewSQLite` returns (or is wrapped
into) a `*database.RoutedDB`, or so `ModuleWithBlobStore` accepts the
helper's type. Should be fixed at the template source
(`templates/scenarios/react-vite/`) so it stops propagating to new
scaffolds.

**Owner:** unassigned (template owner).

**Refs:** `api/handlers/notes/module_test.go:27,33,43,49`;
`api/handlers/notes/module.go:43`;
`api/internal/testutil/db/sqlite.go:49`; template mirror under
`templates/scenarios/react-vite/`.

### 2026-05-26 — Redis source backups are best-effort, not point-in-time

**Symptom:** A Redis source backup may not represent a single
transactionally-consistent instant. Keys written or deleted while the
snapshot is in progress can be partially captured.

**Root cause:** The Redis source kind captures by namespace prefix using
`SCAN` + `DUMP` over the live keyspace rather than a frozen snapshot.
`SCAN` is iterative and non-atomic by design, so the resulting artifact
is a near-consistent view, not a true point-in-time one.

**Workaround:** Accept the best-effort semantics for cache/ephemeral
state (the common Redis use). Where stronger consistency is needed,
prefer a quiesce hook (PRD OT-P1-001) around the source, or back up the
durable store that Redis fronts instead.

**Real fix:** Adopt a transactional snapshot path if/when one is
available for the target Redis deployment (e.g., an RDB/replica snapshot
the source CLI can hand off atomically). Until then this is an accepted
design limitation, not a bug.

**Owner:** unassigned.

**Refs:** `DECISIONS.md` (six source kinds; Redis best-effort);
`PRD.md` source-kind notes and OT-P1-001 (quiesce hooks).

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
