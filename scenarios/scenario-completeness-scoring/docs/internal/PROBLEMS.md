# Problems — Scenario Completeness Scoring

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

### 2026-06-10 — Deferred: lifecycle ledger (greenfield marker, last-deployed, health history)

**Symptom:** Score output cannot say "this scenario was greenfield-regenerated on X" or show deployment/health history.

**Root cause:** Deliberately out of scope for the v1 rewrite (scenario-status-layer plan §12).

**Workaround:** None needed; the digest + freshness block covers "is this evidence current".

**Real fix:** A durable lifecycle event ledger owned by the platform (not this scenario), which SCS would then read as one more collector.

**Owner:** unassigned.

**Refs:** `vrooli plans show scenario-status-layer-…` §12.

### 2026-06-10 — Deferred: search-hub provider registration

**Symptom:** `search-hub query "how complete is X"` does not route to this scenario.

**Real fix:** Keep direct provider registration out of scope unless a consumer demonstrates a need beyond measures federation. The scenario now exposes score measures through measures-health; a separate search-hub timeout observed in that federation path is tracked in `topic:bug-inbox/unexpected-error/search-hub-measure-query-timeout`.

**Owner:** unassigned.

### 2026-06-10 — Digest scope is scenario-dir-only

**Symptom:** Edits to shared `packages/*` code do not flip dependent scenarios' phases to stale, even though their behavior may have changed.

**Root cause:** The treedigest spec (frozen, byte-identical with test-genie) hashes only the scenario working tree.

**Workaround:** Treat cross-package refactors as requiring a manual re-run of consumers' suites.

**Real fix:** A dependency-aware digest (v2 of the freshness contract, owned by freshness-go + test-genie together).

**Owner:** unassigned.

**Refs:** `packages/freshness-go/treedigest/treedigest.go`.

### 2026-06-10 — Per-finding detail only exists for runs recorded after today

**Symptom:** Maturity dimension counts show "(approximated from phase status)" for most scenarios.

**Root cause:** test-genie's per-run `phase-results/<phase>.json` files only started carrying the `findings` array on 2026-06-10 (additive change in `writePhasePointer`); older runs have status-only evidence.

**Workaround:** The rung derivation conservatively treats a failed phase as ≥1 error in that phase's dimension and labels the counts approximate.

**Real fix:** Self-heals as scenarios re-run their suites.

**Owner:** unassigned.

**Refs:** `scenarios/test-genie/api/internal/orchestrator/phases/phase_pointers.go`.

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
