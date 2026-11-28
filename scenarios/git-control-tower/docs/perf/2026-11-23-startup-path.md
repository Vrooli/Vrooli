# Git Control Tower startup path

## Baseline

On 2026-11-23, a direct Connect JSON traversal of repository id `1` (the shared Vrooli checkout) returned 13,204 status category entries across 26 pages and 2,030,371 response bytes. The first 512-entry page took 1,174.7 ms; the complete sequential traversal took 1,355.4 ms. A separate repository-list request took 36.5 ms cold, then 2.8 ms and 1.5 ms warm.

The status timing varied with machine load and worktree contents. After restart, one traversal took 1,707.9 ms for the first page and 2,018.4 ms total (13,214 entries); the next took 960.4 ms and 1,203.8 ms (12,973 entries). These post-change samples are not a controlled before/after comparison. `git status --porcelain=v2 -z --branch` itself took 0.49–0.58 seconds over three local runs.

## Changes

- The workspace layout now resolves the selected repository path from the repository registry response, before the full status query finishes. Saved pane state is applied before paint, so the full status request no longer holds the entire workspace behind the layout skeleton.
- Group resolution now waits for the main status snapshot, keys its cache to that snapshot, and does not poll Git status every five seconds. Group data refreshes when the status snapshot or grouping rules change. This removes a duplicate status read competing with the initial status request and avoids up to twelve redundant group-status reads per minute per open client.
- Paged and unpaged status now share the same five-second, generation-safe status cache. A mutation invalidates every status variant and page snapshot; a status read already in flight cannot publish pre-mutation data afterward. Paged reads also use the existing bounded Git config cache instead of launching two author-config subprocesses on every assembly.
- Worktree identity's independent Git probes now run concurrently. They still resolve the same identity fields and preserve the existing best-effort behavior for HEAD, branch, and worktree-list metadata.
- Repository query transport calls now consume TanStack Query's abort signal for status pages, groups, history, diffs, metrics, sync state, branch data, file search, blame, and directory reads. Switching repos or superseding a query can stop obsolete requests instead of leaving their network/server work running.
- Shared invalidation recipes now cover branch workspace changes, commit effects, file-diff variants, manual refresh, registry changes, and credential changes. The source-distribution panel now uses the canonical key factory, including its default repository/scenario values.

After paged and unpaged status were moved onto the same five-second cache, a live first-page request returned 12,989 total entries in 973 ms; an identical request within the cache window returned the same snapshot in 43 ms. This is a warm-cache sample, not a controlled benchmark and not the cold full-list UI load. A separate 26-page traversal returned all 12,985 entries in 1,252 ms. Both are single-run smoke measurements, not a controlled before/after comparison. In a later run, the first request took 6,482 ms and its repeat took 28 ms; a standalone `git status` command took 560 ms. That cold-path variance remains unexplained and needs a finer runtime profile. The scenario was restarted healthy after these changes.

## Measurement limits

`performance-health audit run git-control-tower` could not capture a browser trace: Browser Automation Studio returned `workflow execution requires an authenticated account`. Linux `perf` also could not attach to the API because the host sets `perf_event_paranoid=4`. The direct timings above measure the API path, not first paint. The frontend changes were type-checked and production-built; the scenario restarted healthy. A BAS-authenticated production trace or an enabled in-process profile is still needed to quantify first paint, React commits, and the unexplained cold request.
