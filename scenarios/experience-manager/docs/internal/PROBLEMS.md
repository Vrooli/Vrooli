# Problems — Experience Manager

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

### 2026-07-04 — BAS lacks accessibility-tree capture (blocks reconciliation)

**Symptom:** OT-P0-003 (structure reconciliation) cannot be implemented: BAS execution timelines carry screenshots but no accessibility-tree snapshot per step.

**Root cause:** Feature does not exist in BAS yet; ui-health's `uiruntime` only consumes screenshot + console/network observations.

**Workaround:** None needed yet — build order puts the spec contract, provider, and studio first.

**Real fix:** File the cross-scenario feature request to BAS (a11y-tree snapshot alongside each step screenshot) before reconciliation work starts; the `bas-screenshot-api-audit` backlog item already touches this capture surface.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md` (single-capture-engine decision), `scenarios/ui-health/api/internal/uiruntime/`.

### 2026-07-04 — Spec schema spike gate before broad rollout

**Symptom:** The claim vocabulary is designed but unproven against real pages; a too-abstract or too-rigid vocabulary would either block authoring or pigeonhole UX.

**Root cause:** Only this scenario's own `experience/` spec exists so far (authored 2026-07-04, schema-valid); the two external pages remain unexpressed.

**Workaround:** n/a — this is a deliberate gate, not a defect.

**Real fix:** Before promoting the phase beyond presence-keyed applicability, the schema must express three pages naturally at useful depth: business-health's Matrix (dense conventional), web-console's terminal/chat surface (hostile to descriptive schemas), and this scenario's own Studio (self-referential dogfood, OT-P0-005 — **done 2026-07-04**: `experience/pages/studio.json`, expressed naturally, no vocabulary fights). Vocabulary gets fixed before rollout if either remaining page fights it.

**Owner:** unassigned.

**Refs:** DECISIONS rows on claim schema, open-world semantics, maturity ladder.

### 2026-07-04 — requirements auto-sync disabled until real validation refs exist

**Symptom:** `requirements/index.json` has `auto_sync_enabled: false`, so requirement statuses will not update automatically from suite runs.

**Root cause:** Known platform bug (filed as `requirements-sync-refless-validations-fake-complete` during the business-health build): comprehensive runs flip refless test-typed validations to complete, fabricating completion for unbuilt work. All 13 seeded requirements currently carry refless stubs.

**Workaround:** Keep auto-sync off; statuses stay honestly `planned`.

**Real fix:** Re-enable once requirements carry real validation refs (`[REQ:...]`-tagged tests) as implementation lands.

**Owner:** unassigned.

**Refs:** business-health PROBLEMS.md documents the same revert loop.

### 2026-07-04 — Requirements modules are wizard-tier-shaped, not thematic

**Symptom:** `requirements/` has `01-must-ship` / `02-post-launch` / `03-future` (one module per OT tier) rather than the eight thematic modules sketched during design (spec-contract, provider-integration, reconciliation, authoring-studio, render-workshop, scaffolding-autofix, attestation-fleet, ux).

**Root cause:** The business-health wizard seeds the registry 1:1 from OT tiers — conformant by construction.

**Workaround:** None needed; the registry validates green and requirement↔OT linkage is complete.

**Real fix:** Optionally restructure into thematic modules during implementation planning, when requirements gain real validation refs anyway. Not worth hand-editing before then.

**Owner:** unassigned.

**Refs:** `requirements/index.json` imports.

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
