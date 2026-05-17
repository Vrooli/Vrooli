### Runs in window
- Errored: 0
- Retried: 0
- Slow: 1
- User-flagged: 0
- Successful: 39 complete + 1 prior investigation in needs_review

### Run picked this heartbeat
- Run ID: `b41e3f3c-31fd-4e43-be7a-93d657f52367`
- Agent: `programmatic-qa-runner`
- Triage tier: slow

### What happened
- The run completed successfully and created 8 Swarm Manager backlog items from 3 GCT readiness reviews, but ran 11m41s against a 600s resolved timeout.
- Investigation run `67eba1d1-b151-444e-9069-5544aa9385f6` found no task failure and flagged only timeout semantics as an anomaly.

### Implicated
- `scenario-qa/programmatic-qa-runner` heartbeat workflow: GCT readiness review to Swarm backlog creation, notes upload, dependency wiring, and knowledge writes.

### Proposed lesson
- Convert the stabilized GCT-readiness-to-backlog workflow into an Action candidate so programmatic QA stops repeating a long manual CLI sequence each heartbeat.
- Handoff to: skill-optimizer

### Action opportunity
- new-action-candidate
- Evidence: event scan found 55 tool calls, including 8 backlog creates, 8 notes uploads, dependency merge/update logic, verification, and 8 knowledge writes; `prompt-manager discover` returned broad skills only and no exact Action.

### Measurement plan
- After adoption, compare the next 5 `programmatic-qa-runner` heartbeats: expected fewer than 25 tool calls for the conversion path and no run exceeding configured timeout due only to backlog-conversion command volume.

### Decisions raised this heartbeat
- `dec-1778971728228105895` - run-lesson - treat programmatic-qa-runner GCT readiness-to-Swarm-backlog conversion as a new Action candidate.

### Knowledge entries written
- `knw-1778971703007132842` - `run-lesson-report/2026-05-16`
- `knw-1778971715484925564` - `friction-report/run-execution/2026-05-16/programmatic-qa-manual-backlog-conversion`