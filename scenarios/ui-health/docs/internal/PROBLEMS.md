# Problems — UI Health

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

_None yet._

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

## 2026-09-07 — Shared selector consolidation

W3 implementation review under the existing selector contract; no W0–W2 maturity promotion is claimed. Selector composition checks use declared UI paths, require library composition, and verify generated manifest source hashes. Focused composition and adoption doc cases pass. Broad run 20260907-213022-122303cc remains FAIL on UI discovery, provider wrapper and other rule cases. Direct full checks also observed missing standard_shell_ownership doc cases, reported as knw-1788818073139349494. Shared evidence and limitations: `packages/ui-selectors/README.md`.

## 2026-09-15 — API log grew to 75 MB on a month-old node

W3 implementation fix; no W0–W2 claim.

**Symptom:** On minimouse, `~/.vrooli/logs/scenarios/ui-health/vrooli.develop.ui-health.start-api.log` reached 75 MB after 32 days of uptime.

**Root cause:** Three writers, none of them bounded.
- Every five-minute aisearch sync logged one line per scenario when react-component-library was unavailable: about 40 identical failures per sync, 1,947 syncs.
- Every health probe was logged twice, by the local `internal/middleware` request logger and by api-core's access log.
- The lifecycle never bounded a live process's log.

**Real fix (done):**
- `aisearch.surfaceSource.LoadAll` writes one summary line per sync (`TestLoadAllSummarizesDiscoveryFailuresInOneLine`).
- The duplicate local request logger was removed.
- api-core logs a `/health` probe only when its status changes.
- The runtime supervisor bounds every scenario log (`process.BoundLogs`).

**Remaining:**
- react-component-library and qdrant are not running on minimouse, so surface search stays degraded there. The failure is now logged once per sync.
- 75 other scenarios generated from `templates/scenarios/react-vite` still carry the duplicate request logger. The template still emits it.

**Refs:** `api/internal/aisearch/surface_index.go`, `packages/api-core/server/server.go`, `internal/process/logbounds.go`.
