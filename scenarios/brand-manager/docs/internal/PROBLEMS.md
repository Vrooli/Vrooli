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

### 2026-09-15 — First live explore produced unusable candidates (fixed)

**Symptom:** `candidates explore` for Vega returned muddy SD 1.5 images that ignored the prompts,
then (re-run with the cloud model) flat geometric SVGs the operator called "extremely primitive".
Every CLI call ended in `unavailable: unexpected EOF` after ~2 minutes, and only 6 of 10
generations were stored.

**Root cause:** (1) explore passed the caller's empty quality policy and `allow_byok=false`
instead of the brand image defaults, so image-tools picked local SD 1.5; (2) the default role
`image.generate.logo` maps to `recraft/recraft-v4.1-vector`, and the "mark only, flat vector line
art" prompt stripped the rendering the approved Aquila concept had; (3) generations ran serially
inside one RPC, past api-core's 30 s write deadline and the CLI's 120 s timeout, on the request
context, so results after the disconnect were dropped.

**Workaround:** none needed after the fix.

**Real fix (landed 2026-09-15):** brand image defaults for explore and refine; illustration roles
and a finished-icon prompt; `style_reference_brand`; parallel generation (4) on a detached context
with a 15-minute budget; write deadline cleared for explore/refine/pick; a 20-minute CLI client for
those RPCs; candidates `--json` emits the typed response; import is idempotent per asset. image-tools
now ranks the gateway first when a role is named and BYOK is permitted, and `--explain` forwards
quality, fallback and role. See `DECISIONS.md` 2026-09-15.

**Owner:** brand-manager.

**Refs:** `api/internal/candidates/service.go`, `handlers/candidates/module.go`,
`scenarios/image-tools/api/internal/models/selector.go`.

### 2026-09-15 — Duplicate Aquila candidate records

**Symptom:** Aquila lists the round-1 constellation concept twice, and a rejected record shares
the picked vector's asset id.

**Root cause:** the plan executor imported the same assets twice before import was idempotent.

**Workaround:** the duplicates are REJECTED and harmless; the PICKED candidate is correct.

**Real fix:** a `candidates delete` (or merge) verb so history can be cleaned without editing the
database. Import is now idempotent, so no new duplicates appear.

**Owner:** unassigned.

**Refs:** brand `2e797e31-c051-494d-bddd-aedce30cb4c4`; candidates `f40f0a6f…`/`2df85f26…`,
`2bcb1db3…`/`c124f09c…`.

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
