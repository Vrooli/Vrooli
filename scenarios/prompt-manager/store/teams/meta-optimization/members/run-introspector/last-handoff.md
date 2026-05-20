### Runs in window
- Errored: 7
- Retried: 0 observed
- Slow: 2 over 600s, both also failed
- User-flagged: 0 observed
- Successful: 21 complete

### Run picked this heartbeat
- Run ID: `6c439ad4-0319-418c-9018-392f75fae29e`
- Agent: `programmatic-qa-runner`
- Triage tier: errored

### What happened
- The run failed after 1m35s with `SANDBOX_NO_EXIT_INFO`, then finalization failed because `workspace-sandbox` was unavailable and recovery was blocked by a concurrent workspace-sandbox start.
- Investigation also found a task blocker before the sandbox died: direct `vrooli scenario status/list` calls failed under `VROOLI_SANDBOX_MERGED` with `read runtime registry schema version: unable to open database file (14)`.

### Implicated
- `agent-manager` / `workspace-sandbox` lifecycle and finalization
- `internal/scenarioruntime` SQLite registry access under `VROOLI_SANDBOX_MERGED`
- `programmatic-qa-runner` GCT readiness flow
- Existing Action `scenario.status.show`

### Proposed lesson
- Route sandboxed scenario lifecycle reads through a host-safe status surface, preferably hardening/adopting `scenario.status.show`, instead of direct `vrooli scenario status ... --json` inside merged sandbox envs.
- Handoff to: skill-optimizer for existing Action adoption/hardening signal; director-swarm via capability-gap if host-safe lifecycle status needs platform work.

### Action opportunity
- existing-action-usage
- Evidence: `prompt-manager discover "scenario status git-control-tower VROOLI_SANDBOX_MERGED runtime registry" --type all` returned `action:scenario.status.show`, and `prompt-manager action validate scenario.status.show --json` reports it valid/runnable. The failed run used direct CLI calls twice instead.

### Measurement plan
- Over the next 7 run-introspector windows, grep errored/retried `programmatic-qa-runner` transcripts for `read runtime registry schema version: unable to open database file (14)` and direct `vrooli scenario status git-control-tower --json` under `VROOLI_SANDBOX_MERGED`; expected count is 0 after adoption/hardening.

### Decisions raised this heartbeat
- None. Owned run-introspector pending contexts are already at the 4-decision cap.

### Knowledge entries written
- `knw-1779231043374645876` - `run-lesson-report/2026-05-19`
- `knw-1779231043375943992` - `friction-report/run-execution/2026-05-19/sandboxed-scenario-status-registry-read-failure`