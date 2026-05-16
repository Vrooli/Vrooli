### Runs in window
- Errored: 14
- Retried: 0
- Slow: 0
- User-flagged: 0
- Successful: 27 complete + 1 running at fetch time

### Run picked this heartbeat
- Run ID: `b969d9cb-c1f6-4850-bf8e-154467810a19`
- Agent: `subscription-advertiser`
- Triage tier: errored

### What happened
- The run failed before agent execution while creating the isolated workspace. Events show `SANDBOX_CREATE` connection refused to `localhost:15120`, marked `retryable=true` with `RECOVERY_ACTION_RETRY`, and no `session_id`.
- Investigation run `5ede7875-236f-4836-9790-74a31013366e` confirmed workspace-sandbox was reachable later, so this was transient/startup dependency availability, not agent prompt behavior.

### Implicated
- Agent-manager workspace setup / workspace-sandbox dependency path, specifically sandbox creation before agent execution.

### Proposed lesson
- Add bounded preflight/retry for retryable `SANDBOX_CREATE` connection-refused failures before finalizing user runs as failed.
- Handoff to: director-swarm via capability-gap

### Action opportunity
- new-action-candidate
- Evidence: this heartbeat again repeated deterministic `agent-manager run list/get/events/investigate` plus `jq` grouping for run-window classification; `prompt-manager discover` found no exact run-window summary Action, only related skills and `action:team.decisions.list`.

### Measurement plan
- For the next 7 run-introspector windows, count no-session `SANDBOX_CREATE` connection-refused user-run failures. Expected: 0 if workspace-sandbox comes up within retry window, otherwise failures include explicit exhausted-retry evidence.

### Decisions raised this heartbeat
- `dec-1778885362329320192` - capability-gap - add agent-manager workspace-sandbox create preflight/retry for retryable connection-refused failures.

### Knowledge entries written
- `knw-1778885334234212715` - `run-lesson-report/2026-05-15`
- `knw-1778885334234710814` - `friction-report/run-execution/2026-05-15/sandbox-create-transient-connection-refused`