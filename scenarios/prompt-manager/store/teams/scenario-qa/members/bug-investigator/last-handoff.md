### Inbox state
`bug-inbox/*` is empty. Verified after deleting source inbox entry `knw-1779271382501423594`.

### Investigation in flight
None.

### Investigation closed this heartbeat
Closed `bug-inbox/unexpected-error/vrooli-runtime-registry-open-fails`.

### Technique applied
`scientific-debugging`.

### Action taken
Closed as `observe`. Root cause: `VROOLI_SANDBOX_MERGED` forces `scenarioruntime.NewSQLiteStore` into the read-only SQLite DSN path; that path fails to reopen the runtime registry DB with `unable to open database file (14)`, causing safe read-only orientation commands like `vrooli scenario list --json` and `vrooli status --verbose` to fail instead of degrading.

### Backlog item / decision created
None. Four owned-context `bug-resolution-proposal` decisions are already pending, so no new decision was allowed this heartbeat.

### Capability-gap raised
None.

### Surface for technique graduation
None. Scientific debugging fit the investigation.