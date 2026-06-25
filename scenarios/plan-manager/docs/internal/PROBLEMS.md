# Problems — Plan Manager

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

### 2026-06-25 — Documentation-first; no runtime code yet

**Symptom:** The scenario validates its PRD/requirements/docs but has no real domains; the fenced `notes` example domain is still present, so the `example-domain-removed` orientation step fails by design.

**Root cause:** Intentional — this session implemented Gates 1–3 + 5b (PRD, requirements, concept/business/ops docs) only. The first vertical slice (`plans`) and `vrooli scenario detemplate` are future work.

**Workaround:** N/A — expected state for a documentation-first handoff.

**Real fix:** Build the `plans` slice (Gate 6) beside the example, then `vrooli scenario detemplate plan-manager` (Gate 7).

**Owner:** unassigned (next implementation session).

**Refs:** `docs/START-HERE.md` Gates 6–7; `requirements/` modules.

### 2026-06-25 — Legacy `~/.vrooli/plans` coexistence is unspecified in code

**Symptom:** plan-manager will share the home store with the existing `vrooli plans` file store, but the adoption/coexistence path is documented (DATA.md) not built.

**Root cause:** Storage decision (scenario-owned logic over the durable home store) is new; the migration/coexistence step is deferred to the `plans` slice.

**Workaround:** Treat existing markdown plans as import sources; do not destructively migrate.

**Real fix:** Implement non-destructive adoption of existing `~/.vrooli/plans` records when building the `plans` domain.

**Owner:** unassigned.

**Refs:** `docs/concepts/DATA.md` (Migrations And Compatibility); `internal/app/plans` (existing store).

### 2026-06-25 — `prd-control-tower prd generate` (LLM path) returns HTTP 500

**Symptom:** `prd-control-tower prd generate plan-manager --publish` fails with `api error (500)`.

**Root cause:** Environmental (the generate LLM endpoint), not the scenario. Same behavior was seen for meta-optimization-manager.

**Workaround:** Hand-author `PRD.md` to the canonical template and validate with `prd-control-tower prd validate` (status healthy).

**Real fix:** None on this scenario's side; revisit if the generate endpoint is restored.

**Owner:** unassigned (prd-control-tower).

**Refs:** `PRD.md`; `prd-control-tower prd validate plan-manager --json`.

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
