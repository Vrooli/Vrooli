# Meta-Orchestrator Summary

## Source

Surfaced while explaining the end-state operator experience for `execute/deployment-manager-lpbs-desktop-release-orchestration`. The UNIQUE constraint on `releases (profile_id, git_commit_hash, channel)` causes awkward retry behavior when an attempt fails mid-flight.

## Codebase Inspection Already Done

- `scenarios/deployment-manager/api/migrations/005_add_releases.sql:37` — `UNIQUE (profile_id, git_commit_hash, channel)` on the releases table.
- `scenarios/deployment-manager/api/releases/repo.go:68-116` — Insert opens a transaction, inserts the release row, then inserts one release_platforms row per target. A unique-violation here surfaces as a plain DB error with no retry logic.
- `scenarios/deployment-manager/api/releases/handlers.go:220-320` — Start calls Insert after acquiring the advisory lock. On unique violation today, the handler returns a 500 with "insert release: ...".
- `scenarios/deployment-manager/api/deployments/orchestrator_release.go` — When cloud-health, readiness, build, or publish fails, the orchestrator calls `o.markReleaseFailed(ds, releases.StatusFailed)` which updates `releases.status = 'failed'` but keeps the row in place.

## Decisions Made

- This is a real operational wart, not a theoretical edge case. Filing as P4 / fix / XS.
- Workshop decides between three implementation options (see Unresolved).

## Unresolved Questions Deferred To Workshop

- **Option A (delete-and-reinsert)**: simplest; under the advisory lock, if the existing row has status in (failed, verify_failed), DELETE it (CASCADE drops release_platforms) and INSERT the new one. Risk: loses failed-attempt history unless we add an audit/archive table.
- **Option B (update-in-place)**: reuse the row, bump updated_at, reset status=pending, re-insert release_platforms. Risk: callers who stored the old release_id now point at a row with different semantics. Can we guarantee nobody references a failed release_id externally? Probably yes since failed releases aren't surfaced.
- **Option C (partial unique index)**: change the constraint to `UNIQUE (profile_id, git_commit_hash, channel) WHERE status NOT IN ('failed', 'verify_failed')`. Allows multiple failed rows to coexist with one current. Cleanest but migration needs care on existing data.
- **History preservation**: whichever option is chosen, consider whether a `releases_history` side table is worth adding to retain forensic trail of failed attempts. Probably not critical — the step trace from the orchestrator is better diagnostic data than the frozen-in-time release row.

## Dependency Notes

Depends on `execute/deployment-manager-lpbs-desktop-release-orchestration` (the code that introduced the unique constraint). Independent of typed errors, preflight, and skill rewrite.

## Effort Assessment

XS — one repo method, one migration (if Option C), a couple of handler tests covering retry paths. Estimated 30-50 lines including tests.
