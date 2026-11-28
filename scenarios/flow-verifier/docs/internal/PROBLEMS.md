# Problems — Flow Verifier

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

### Template defects surfaced during Phase A scaffold (2026-05-12) — resolved 2026-11-24

**Disposition:** The scaffold-fix entry is closed for this scenario. Phase G removed the local `tools/temporal-model/` bridge; current temporal codegen and flow tests live under `api/internal/flows/kinds/temporal/` and generated replay helpers under `api/internal/`.

**Owner evidence:** Current generation supplies the module path and generic `Transition` alias; generated replay files use the current schema/runtime loaders and the generated-directory import exception. Focused temporal-codegen, pipeline, testutil, and flow tests pass. The separate scenario-auditor `prd-linkage` and comprehensive readiness concerns below remain open; this disposition is not a full scenario-readiness claim.

**Owner:** Flow Verifier maintainers.

**Refs:** `api/internal/flows/kinds/temporal/`, `api/internal/pipeline/pipeline.go`, `api/internal/testutil/no_prod_import_test.go`, `api/internal/runs/flow/`; focused test evidence is in the PT5 run receipt.

---

### `scenario-auditor` `prd-linkage` standards on initial PRD/requirements (2026-05-12)

**Problem:** Even after the PRD and requirements registry were rewritten with PRD-specific content, scenario-auditor's `prd-linkage` rule reports HIGH+ violations: `P1 target missing requirements` for each of the 4 P1 targets in `PRD.md`, and `Requirement … references missing PRD content` for several requirements. This blocks `make test` and therefore `make orient` Gate 0 (scaffold-health) when test is re-run.

**Workaround:** Direct `make test` (without orient) currently passes the unit/lint phases; the scaffold-health gate intermittently passes vs fails depending on whether the auditor runs. Requirement `prd_ref` values were normalized from `PRD.md#OT-P0-NNN` to `OT-P0-NNN` (matching test-genie format) to clear part of the warnings.

**Real fix:** Add one requirement module per P1 target (state-graph visualizer, trace player, counterexample diff, verification timeline) referencing `OT-P1-001..004`. Defer until Phase H or as part of post-MVP scoping.

**Owner:** continuation of the flow-verifier extraction plan.

**Refs:** `requirements/0[1-4]-*/module.json`, `PRD.md`.

---

### Deferred UI screens (carry-over backlog)

**Problem:** OT-P1-001 (state-graph visualizer), OT-P1-002 (trace player), OT-P1-003 (counterexample diff), OT-P1-004 (verification timeline) are explicitly deferred from the MVP plan.

**Workaround:** None needed; documented as P1 in PRD.

**Real fix:** Schedule follow-up plans after Phase F lands.

**Owner:** unassigned.

**Refs:** `PRD.md` P1 section; `docs/concepts/DOMAINS.md` Deferred Domains table.

## Architecture Drift

Use this section for deferred findings from `architecture-lenses`.
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
