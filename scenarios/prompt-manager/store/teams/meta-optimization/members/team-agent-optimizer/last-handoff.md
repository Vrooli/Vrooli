### Domain worked this heartbeat
- agent

### Target picked
- `run-introspector` - agent-default ladder; skipped covered/recent low-health nodes, then selected next low-health active node at 0.51 with no prior visit entry.

### Disposition
- improve candidate; no decision raised due owned-context cap.

### Evidence
- `prompt-manager graph node run-introspector` reports health 0.51 with only membership inbound and `cli:prompt-manager` outbound.
- Member is active and useful: latest run lesson `run-lesson-report/2026-05-18` supported capability-gap `dec-1779144543464355217`.
- Generated required memory includes `friction-report/run-execution/*`, and recent heartbeats wrote that topic.
- `prompt-manager graph topics --team meta-optimization` flags five `run-introspector` `friction-report/run-execution/*` writes as `actual_writer_undeclared` because outputs declare only `run-lesson-report/*`.

### Expected delta
- When queue allows, clarify routing: either declare `friction-report/run-execution/*` as a `run-introspector` output or route these through `friction-inbox/*` / `report-friction` so `friction-curator` remains the declared writer.
- Measure with `prompt-manager graph topics --team meta-optimization`: no new `run-introspector` `actual_writer_undeclared` errors for run-execution friction.

### Capability architecture
- weak
- Primary layer gap: promotion-routing
- Routing: team-agent-optimizer once owned-context pending count drops below 4; coordinates with existing storage/friction capability-gap `dec-1779144543464355217`.

### Artifacts updated
- AGENT_AUDIT.md: not edited because this run’s write surface allows knowledge, decisions, and handoff only.
- DEPRECATION_QUEUE.md: unchanged.

### Decisions raised this heartbeat
- None. Owned-context pending count is already 4: `dec-1778797938232697845`, `dec-1778884421236613535`, `dec-1779057112765291497`, `dec-1779143529470088721`.

### Knowledge entries written
- `agent-visited/run-introspector` (`knw-1779229899895926455`)
- `agent-audit/2026-05-19` (`knw-1779229915880820101`)
- `friction-report/prompt-team-agent-storage/2026-05-19/run-introspector-undeclared-run-execution-friction-output` (`knw-1779229928986456831`)