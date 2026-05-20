### Scenarios reviewed
Reviewed 2 queued scenarios with fresh GCT completeness runs:
- `web-console`: `production_ready`, score 99, calculated at `2026-05-20T16:01:17Z`; no backlog created.
- `vrooli-events`: `production_ready`, score 100, calculated at `2026-05-20T16:01:17Z`; no backlog created.

### Findings converted to backlog
None. Both scenarios had `has_issues=false`, `issue_count=0`, `total_penalty=0`, and no recommendations. Decision writes were forbidden by the active write contract.

### Dependencies wired
None. No QA gate backlog items were created. Existing `web-console` active fix `fix/web-console-vad-false-silence` remains unrelated to GCT readiness; `vrooli-events` has no active fixes.

### Skipped scenarios
Review queue returned only `web-console` and `vrooli-events`; 31 scenarios were excluded due to pending fixes.

### Bugs filed (via report-bug)
None this run.

### Knowledge entries written
- `qa-run/web-console`: `knw-1779292910079199116`
- `reviewed-scenario/web-console`: `knw-1779292915957875727`
- `qa-run/vrooli-events`: `knw-1779292924136735602`
- `reviewed-scenario/vrooli-events`: `knw-1779292932942946452`
- `dependency-wiring/2026-05-20-gct-completeness-queued-scenarios-3`: `knw-1779292939708638877`