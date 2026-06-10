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

### 2026-06-09 — Scenario is documentation-only; nothing is implemented — RESOLVED

**Status:** RESOLVED 2026-06-09. This entry described the post-workshop state; it was
never updated after the two-agent build landed the full implementation. All P0 + P1
domains (livesearch, findings, research, federation) are built, green, and live-validated,
and the `notes` example was removed. Verified by 3 code audits + live runtime. Kept here
(struck through) as the record of the doc-drift that misled subsequent agents.

**Original symptom:** `make test` / orientation `scaffold-health` fails; only the template
`notes`/`health` example code exists. — No longer true.

### 2026-06-09 — Verify the SearXNG resource is healthy on this host before P0 — RESOLVED

**Status:** RESOLVED 2026-06-09. SearXNG confirmed healthy on this host: live `web-search
search` returns real hits and the L1/L2/L3 paths exercise it end-to-end. No drift observed.

**Original concern:** The live-web path (L0/L1/L2) depends entirely on the SearXNG resource,
which had not been re-verified on this host. — Now verified.

**Refs:** `docs/concepts/INTEGRATIONS.md`, `resources/searxng/`.

### 2026-06-09 — Open design questions deferred to implementation/P2

**Symptom:** A few design choices were deliberately left open in the workshop.

**Root cause:** Lower-impact decisions that are cheap to defer.

**Workaround / open items:**
- Contradiction auto-resolution confidence threshold — RESOLVED: `HighConfidenceThreshold = 0.75` (`api/internal/research/l3_agent.go`), the SSOT for the supersede-vs-flag gate.
- Reconciliation scope is "semantically near the query" (bounded) — RESOLVED at the API layer: `GatherRelatedFindings(ctx, query, max=20)` enforces the bound (OT-P1-003 hardening). Full-store consistency sweep shipped as OT-P2-003 GC.
- Usage-telemetry-driven curation (OT-P2-001) — RESOLVED: effectiveness ledger (`finding_usage` side table + async surfacing capture) shipped.
- Findings export / cross-instance sharing — STILL DEFERRED (see DATA.md import/export); out of scope for the completion plan.

**Real fix:** Resolve each as its owning requirement is implemented. Only findings
export / cross-instance sharing remains open.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md`, `PRD.md` P2 targets.

### 2026-06-09 — `scenario test` standards phase RED on pre-existing scanner false-positives

**Symptom:** `vrooli scenario test web-search` (and orient `scaffold-health`) fails on
the `standards` phase: 44 findings, highest=critical, exceeds `fail_on=high`.

**Root cause:** The HIGH+ findings are scanner-auditor false-positives / established
conventions on idiomatic code, NOT defects:
- 6× "Database Rows Not Closed" (`api/internal/findings/sqlite.go`) — the rows ARE
  closed: every flagged query passes its `*sql.Rows` to `hydrate` → `scanFindingRows`,
  which has `defer rows.Close()`. The scanner only looks for a literal `defer
  rows.Close()` at the call site.
- 1× "HTTP Client Without Timeout" (`api/internal/httpc/doer.go:34`) — that line is the
  `var _ Doer = (*http.Client)(nil)` compile-time guard, not a client construction;
  the real client is built in `main.go` with a timeout. (Known template false-positive,
  see memory `feedback_react_vite_template_defects`.)
- 1× "Hardcoded Password" (`cli/domains/findings/handlers.go:18`) — flags
  `const defaultWindowToken = "this_week"`, a time-window token, not a credential.
- 7× "Hardcoded Localhost" — the documented degraded-default URLs for SearXNG / Ollama /
  browserless (the established degraded-behavior convention).

**Evidence it is NOT from the completion work:** `git-control-tower baseline diff
--scenario web-search --name web-search-completion` reports the standards failure as
`preexisting (inherited from baseline)` — it was already RED in the baseline captured
before any completion-plan code landed. `tests` (unit/integration/smoke) stay green and
unchanged. The completion changes are independently clean under `golangci-lint`,
`gofumpt`, `tsc`, and `eslint`.

**Workaround:** Treat the standards phase as a known false-positive campaign; gate on the
real `tests` phase (green) plus the four standard linters (green).

**Real fix:** A scanner-auditor heuristics upgrade (see-through-indirection for
rows.Close; distinguish compile-guards from constructions; stop flagging window tokens as
passwords; recognize degraded-default localhost). Tracked via `report-bug`; this is a
fleet-wide template/scanner campaign, not web-search-specific.

**Owner:** unassigned (scanner-auditor / template owners).

**Refs:** memory `feedback_react_vite_template_defects`; GCT baseline `web-search-completion`.

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
