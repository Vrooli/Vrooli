### Inbox state
- `prompt-manager team knowledge-list scenario-qa --topic-prefix=bug-inbox/` returned `No knowledge entries found`.
- bug-inbox entries pending: 0

### Investigation in flight
- none

### Investigation closed this heartbeat
- `gct-review-poll-loses-api-base`
- Source inbox row `knw-1778774971484367104` was deleted after closure.
- Audit log written: `knw-1778788998030863951` on `bug-investigation-report/gct-review-poll-loses-api-base`.

### Technique applied
- `scientific-debugging`
- Phase 0 prior-art checks found no matching prior fix recurrence.

### Action taken
- `file-decision`
- Root cause identified: GCT `review run` starts successfully, then each poll re-resolves API base through `DetectPortFromVrooli`; a transient detector miss during polling returns empty base and aborts before status request.
- Verification: `go test ./domains/review` and `go test ./...` passed in `scenarios/git-control-tower/cli`.

### Backlog item / decision created
- Created pending `bug-resolution-proposal`: `dec-1778788973084288635`.
- Proposal: pin resolved API base for the lifetime of a review run, preserve/print job id for recovery, and add regression coverage for a resolver that succeeds on start and returns empty during polling.

### Capability-gap raised
- none

### Surface for technique graduation
- none; registered `scientific-debugging` fit the investigation.