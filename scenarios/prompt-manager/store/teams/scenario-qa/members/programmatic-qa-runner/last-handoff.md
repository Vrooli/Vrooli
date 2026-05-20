### Scenarios reviewed
Reviewed 2 queued scenarios with fresh GCT completeness runs:
- `web-console`: `production_ready`, score 99, calculated at `2026-05-20T22:01:27Z`; no backlog created.
- `vrooli-events`: `production_ready`, score 100, calculated at `2026-05-20T22:01:27Z`; no backlog created.

### Findings converted to backlog
None. Both scenarios had `has_issues=false`, `issue_count=0`, `total_penalty=0`, and no recommendations. Decision writes were forbidden by the active write contract.

### Dependencies wired
None. No QA gate backlog items were created. Existing `web-console` active fix `fix/web-console-vad-false-silence` remains unrelated to GCT readiness; `vrooli-events` has no active fixes.

### Skipped scenarios
Review queue returned only `web-console` and `vrooli-events`; 31 scenarios were excluded due to pending fixes.

### Bugs filed (via report-bug)
None this run.

### Knowledge entries written
- `qa-run/web-console`: `knw-1779314548512537835`
- `reviewed-scenario/web-console`: `knw-1779314548633059629`
- `qa-run/vrooli-events`: `knw-1779314548632971359`
- `reviewed-scenario/vrooli-events`: `knw-1779314548633061769`
- `dependency-wiring/2026-05-20-gct-completeness-queued-scenarios-4`: `knw-1779314556756720440`

Storage friction noted in dependency-wiring: generated instructions still show `knowledge-add --by=<agent-id>`, but the CLI now rejects `--by` and auto-attributes identity.