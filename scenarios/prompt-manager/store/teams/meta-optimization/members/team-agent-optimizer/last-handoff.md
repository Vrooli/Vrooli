### Domain worked this heartbeat
- agent

### Target picked
- `debt-curator` - agent-default ladder; skipped covered/recent low-health nodes, then selected next low-health active meta-optimization member with no prior visit entry.

### Disposition
- improve candidate; no decision raised due owned-context cap and team governance ceiling.

### Evidence
- `prompt-manager graph node debt-curator` reports health 0.56 with one membership inbound and only `cli:prompt-manager` / `cli:vrooli` outbound.
- Member is active and useful: latest handoff reviewed 139 storage entries, evaluated five promotion candidates, and wrote `debt-scan/2026-05-19` plus recurring workaround friction.
- `prompt-manager graph topics --team meta-optimization` flags five `debt-curator` `friction-report/recurring-workaround/*` writes as `actual_writer_undeclared` because outputs declare only `debt-scan/*`.
- Current debt-curator task requires `friction-report/recurring-workaround/<YYYY-MM-DD>/<slug>`, so this is promotion/routing drift rather than role failure.

### Expected delta
- When queue allows, clarify routing: either declare `friction-report/recurring-workaround/*` as a `debt-curator` output or route recurring workaround reports through `friction-curator` / `report-friction`.
- Measure with `prompt-manager graph topics --team meta-optimization`: no new `debt-curator` `actual_writer_undeclared` errors for recurring-workaround friction.

### Capability architecture
- weak
- Primary layer gap: promotion-routing
- Routing: team-agent-optimizer once owned-context pending count drops below 4; coordinates with recurring storage/friction output drift already seen for run-introspector and toolchain-validator.

### Artifacts updated
- AGENT_AUDIT.md: not edited because this run’s write surface allows knowledge, decisions, and handoff only.
- DEPRECATION_QUEUE.md: unchanged.

### Decisions raised this heartbeat
- None. Owned-context pending count is already 4: `dec-1778797938232697845`, `dec-1778884421236613535`, `dec-1779057112765291497`, `dec-1779143529470088721`. Total team pending count is 14, above the governance ceiling.

### Knowledge entries written
- `agent-visited/debt-curator` (`knw-1779316306089697111`)
- `agent-audit/2026-05-20` (`knw-1779316306197613136`)
- `friction-report/prompt-team-agent-storage/2026-05-20/debt-curator-undeclared-recurring-workaround-output` (`knw-1779316306198997338`)