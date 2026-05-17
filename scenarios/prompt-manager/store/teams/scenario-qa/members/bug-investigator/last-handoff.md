### Inbox state
`prompt-manager team knowledge-list scenario-qa --topic-prefix=bug-inbox/` now returns no entries.

### Investigation in flight
None.

### Investigation closed this heartbeat
Closed `knw-1778990924577888480` (`bug-inbox/prompt-confusion/report-friction-writer-team-mismatch`).

### Technique applied
`scientific-debugging`.

### Action taken
Validated `prompt-confusion`; reproduced both failures:
- cross-team `knowledge-add meta-optimization` from `scenario-qa` fails with `attribution: team_mismatch`
- `--by` on `knowledge-add` fails as removed/stale

Wrote report `knw-1779004948062162828`, then deleted the original inbox entry.

### Backlog item / decision created
Created `dec-1779004919079440975` in `bug-resolution-proposal` to fix the universal writer path for `report-friction` and update stale `--by` skill documentation.

### Capability-gap raised
None.

### Surface for technique graduation
None.