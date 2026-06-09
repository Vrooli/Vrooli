# Problems — Web Search

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

### 2026-06-09 — Scenario is documentation-only; nothing is implemented

**Symptom:** `make test` / orientation `scaffold-health` fails; only the template `notes`/`health` example code exists.

**Root cause:** This pass intentionally formalized the charter, requirements, and docs only — no product code, proto, or wiring was written (per the request to formalize design before implementing).

**Workaround:** Treat `PRD.md`, `requirements/`, and `docs/concepts/*` as the build spec. Begin implementation at Gate 6 (`docs/START-HERE.md`), starting with the `livesearch` domain.

**Real fix:** Implement the domains (livesearch → findings → research → federation), make them green, remove the `notes` example (Gate 7).

**Owner:** unassigned.

**Refs:** `docs/internal/PROGRESS.md` (2026-06-09 entry), `docs/START-HERE.md` Gates 6–7.

### 2026-06-09 — Verify the SearXNG resource is healthy on this host before P0

**Symptom:** Unknown — to be checked. The live-web path (L0/L1/L2) depends entirely on the SearXNG resource.

**Root cause:** SearXNG (`resources/searxng/`) exists and appears maintained (image pinned, standards-compliant), but has not been touched in a while and has not been re-verified healthy on this host for this scenario.

**Workaround:** Before building L0, run `vrooli resource start searxng` and confirm a live JSON query against `${SEARXNG_URL}/search?q=...&format=json`. Confirm it is at current resource standards.

**Real fix:** Confirm SearXNG healthy/current; if drifted, bring it to standards (separate resource work, not this scenario).

**Owner:** unassigned.

**Refs:** `docs/concepts/INTEGRATIONS.md`, `resources/searxng/`.

### 2026-06-09 — Open design questions deferred to implementation/P2

**Symptom:** A few design choices were deliberately left open in the workshop.

**Root cause:** Lower-impact decisions that are cheap to defer.

**Workaround / open items:**
- Contradiction auto-resolution confidence threshold — concrete value TBD at implementation.
- Reconciliation scope is "semantically near the query" (bounded), with a full-store consistency sweep deferred to P2 GC (OT-P2-003).
- Usage-telemetry-driven curation (OT-P2-001) intentionally deferred; per-query reconcile + age-decay assumed sufficient initially.
- Findings export / cross-instance sharing not designed (see DATA.md import/export).

**Real fix:** Resolve each as its owning requirement is implemented.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md`, `PRD.md` P2 targets.

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
