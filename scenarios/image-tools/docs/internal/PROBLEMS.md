# Problems — Image Tools

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

### 2026-06-16 — Documentation foundation complete; implementation not started

**Symptom:** Orientation reports 5/8 gates. The three incomplete gates —
`scaffold-health` (`make test`), `dependency-decisions`
(`.vrooli/service.json:dependencies.resources` not present), and
`example-domain-removed` (the `notes` worked example still present) — are
all red.

**Root cause:** This pass deliberately delivered only the planning
foundation (PRD, requirements registry, docs). No product code has been
written, the `notes` template example is intentionally retained as the
worked vertical slice, and no resource dependencies have been wired into
`.vrooli/service.json` yet.

**Workaround:** None needed — this is the intended state at the end of the
charter/requirements/docs phase.

**Real fix:** During implementation: build the first real domain (start
with `ops` or the `models`/`jobs` spine), wire required dependencies into
`.vrooli/service.json` (ComfyUI + per-op model resources as `required:false`;
BYOK provider config), get `make test` green, then remove the `notes` example
(Gate 7). Host hardware facts for the AI tier come from the root `vrooli host
inventory --json` CLI (the shared `internal/hostinventory` collector) consumed
behind the `capabilities` seam — not a system-monitor dependency — see
[`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).

**Owner:** unassigned (next implementation session).

**Refs:** `PRD.md`, `requirements/`, [`DECISIONS.md`](DECISIONS.md),
[`../START-HERE.md`](../START-HERE.md) Gates 4/6/7.

### 2026-06-16 — prd-control-tower generation gotchas (tooling, not scenario)

**Symptom:** `prd-control-tower prd generate --publish` failed with
`ORPHANED_CRITICAL_TARGETS`; `prd-control-tower requirements generate`
exceeded the client HTTP timeout for a large (28-target) registry.

**Root cause:** `--publish` enforces that every P0 target already has a
linked requirement (stricter than the START-HERE flow implies), and the
requirements generator builds all modules in one synchronous LLM call
that can outrun the CLI's HTTP client timeout.

**Workaround:** Generate the PRD draft without `--publish`, read the
markdown from `.generation.generated_text` in the `--json` output, and
write `PRD.md` directly; author the requirements modules directly from the
PRD's OT ids (`prd_ref: "OT-Pn-NNN"`). Then validate with
`prd-control-tower prd validate` and `... requirements validate` (both
report healthy).

**Real fix:** N/A for this scenario — upstream prd-control-tower
ergonomics. Captured so the next scenario author doesn't re-discover it.

**Owner:** unassigned.

**Refs:** prd-control-tower CLI; `requirements/index.json`.

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
