### Runs in window
- Errored: 10
- Retried: 0
- Slow: 0
- User-flagged: 0
- Successful: 20 complete + 0 needs-review + 1 running at fetch time

### Run picked this heartbeat
- Run ID: `1d5be21f-20c1-4066-9c02-eebf68a95b05`
- Agent: `run-introspector`
- Triage tier: errored

### What happened
- The run failed before agent execution: workspace creation hit `SANDBOX_CREATE` with `localhost:15120` connection refused, no `session_id`, and summary error `Unable to create isolated workspace`.
- Window cluster: 9 identical no-session sandbox-create failures from 2026-05-13T22:45:00Z through 2026-05-14T00:30:00Z.

### Implicated
- `run-introspector` `HEARTBEAT.md` tier-1 triage gate.
- Underlying runtime surface: agent-manager/workspace-sandbox availability, not agent prompt or skill behavior.

### Proposed lesson
- Add `sandbox-create-unavailable` to the consolidated tier-1 environmental-failure exclusions block, distinct from `sandbox-no-exit-info`.
- Handoff to: team-agent-optimizer

### Action opportunity
- new-action-candidate
- Evidence: this heartbeat repeated deterministic `agent-manager run list/get/events` + `jq` grouping for run-window classification; `prompt-manager discover` found no exact Action for run-window failed-run summarization. Secondary handoff: skill-optimizer should consider a run-window-summary Action.

### Measurement plan
- After implementation, future run-lesson reports should not pick `Unable to create isolated workspace` or `SANDBOX_CREATE` no-session failures as tier-1 investigations. Check over the next 7 heartbeats; expected count is 0.

### Decisions raised this heartbeat
- `dec-1778798933304271622` - run-lesson - add sandbox-create-unavailable to run-introspector environmental exclusions.

### Knowledge entries written
- `knw-1778798964555672652` - `run-lesson-report/2026-05-14`
- `knw-1778798979572442723` - `friction-report/run-execution/2026-05-14/sandbox-create-connection-refused`