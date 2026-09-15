# Problems — Brand Manager

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

### 2026-06-27 — Rebuild in progress: authoring domains not yet ported

**Symptom:** A comprehensive `vrooli scenario test brand-manager` shows several non-green phases
(docs/business/measures/dependencies/tidiness/unit) and the authoring domains (brands,
assignments, assets, generation, apply, discovery, design) are not yet implemented on the new
Connect stack. Requirements are all `status: planned`.

**Root cause:** The scenario was regenerated from `react-vite` (2026-06-27) and is mid-port. The
transport-agnostic algorithms (aigen/contrast/repository/apply/discovery/DESIGN-export) live in
`/tmp/brand-manager-OLD-reference` and are being lifted into `api/internal/<domain>/` domain by
domain. Until each domain lands, its requirements stay `planned` and its phase coverage is thin.

**Workaround:** Treat the post-regen test-genie baseline (`brand-manager-postregen`) as the anchor;
the remaining reds are expected scaffold-incompleteness, not regressions. The example `notes`
domain reds (e.g. `notes attach` undeclared, `notes/flow/generated/replay.go` test-helper import)
clear once `vrooli scenario detemplate brand-manager` runs.

**Real fix:** Complete the domain port (Phase 2) and the validation phase (Phase 3); flip
requirement statuses from `planned` to `complete`/`implemented` as each lands.

**Owner:** unassigned (rebuild executor).

**Refs:** `~/.vrooli/plans/brand-manager-regenerate-validation-as-test-genie-phase.md`;
`/tmp/brand-manager-OLD-reference` (port source); `docs/internal/DECISIONS.md`.

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
