# Problems — Tunnel Manager

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

### 2026-06-18 — Product implementation not yet built (documentation-first)

**Symptom:** API/CLI/UI describe domains and endpoints that do not exist as code yet; only the template scaffold + fenced `notes` example are present.

**Root cause:** Intentional. Phase 1 is documentation-first (charter → requirements → domain map → docs). Implementation is Phase 2.

**Workaround:** Treat all reference docs as planned contracts. Requirement statuses are `planned`.

**Real fix:** Phase 2 — port domain logic from `/tmp/tunnel-manager-OLD-reference` into proto → API → CLI → UI, domain by domain (routes first), then `vrooli scenario detemplate tunnel-manager`.

**Owner:** unassigned. **Refs:** `docs/plans/tunnel-manager-regen-adoption-plan.md`, `PRD.md`.

### 2026-06-18 — `make test` reports fleet-reds from template/example content

**Symptom:** `make test` exits 1 with dependencies/unit/tidiness ERROR findings even though raw unit (21) and dependencies (14) phases pass and the scaffold is healthy.

**Root cause:** test-genie fleet analysis flags the template's own example/scaffold content — `notes` domain (`TEST_HELPER_FROM_PRODUCTION`, duplicated blocks, low coverage), formal-flow testutil cyclomatic complexity, the UI coverage gate (App.tsx/profiler 0%), and the pnpm `minimumReleaseAge` policy. None originate from this regen.

**Workaround:** Track, do not chase. Most clear at Gate 7 (`detemplate` removes `notes`) and when real UI/tests land.

**Real fix:** Phase 2 detemplate + real domain coverage; add the pnpm `minimumReleaseAge` policy when touching UI deps via SDA.

**Owner:** unassigned. **Refs:** `coverage/latest/findings.json`.

### 2026-06-18 — `prd-control-tower` generate/validate blocked

**Symptom:** `prd-control-tower prd generate` returns `ORPHANED_CRITICAL_TARGETS`; `prd validate` returns `blocked`.

**Root cause:** Known prd-control-tower issue (also hit by image-tools). Unrelated to PRD content.

**Workaround:** PRD authored directly to the canonical v2.0 template (the documented fallback); the orientation charter gate (placeholder-absence) passes. `requirements validate` works and returns healthy.

**Real fix:** Re-run `prd-control-tower` once the tool issue is resolved; until then the hand-authored PRD is authoritative.

**Owner:** unassigned. **Refs:** `PRD.md`.

### 2026-06-18 — Cloudflare hostname cap unconfirmed

**Symptom:** The exact maximum public-hostname count per tunnel is unknown (operator estimate ~100 via dashboard; docs don't state a hard cap).

**Root cause:** Cloudflare docs cover ingress config shape, not limits; the dashboard limit likely differs from API/config-managed limits.

**Workaround:** Tiered exposure (core + leased) is cap-robust regardless. Hostname-budget management is parked at OT-P2-001.

**Real fix:** Phase 3 — confirm the real cap against the live Cloudflare plan; promote OT-P2-001 to P0 if the cap is low.

**Owner:** unassigned. **Refs:** `PRD.md` (note under P2).

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
