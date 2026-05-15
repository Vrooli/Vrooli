### Domain worked this heartbeat
- agent

### Target picked
- `agent-29` - never visited, tied lowest active graph health at 0.39, and had concrete current-state evidence beyond health score.

### Disposition
- prune

### Evidence
- `agent-29` is active and tracked, but prompt files are template placeholders:
  - `SOUL.md`: “Who I am, how I communicate, and my boundaries.”
  - `AGENTS.md`: generic operating procedure text plus an example `e2e-testing` skill line.
  - `TOOLS.md`: “Tooling notes and preferences.”
- Markdown prompt surface totals 16 lines.
- Graph shows `incoming-edges=0.00`, one outbound `cli-read -> e2e-testing`, and low inbound discoverability.
- `rg` found no team roles or docs referencing `agent-29` outside its own files and generated indexes.

### Expected delta
- Remove one active placeholder from the agent graph.
- Measure by confirming `prompt-manager graph health --type agent` no longer lists `agent-29`, and `rg 'agent-29|Agent 29' scenarios/prompt-manager/store docs` only finds archival/deprecation references.

### Capability architecture
- weak
- Primary layer gap: identity
- Routing: team-agent-optimizer

### Artifacts updated
- `AGENT_AUDIT.md`: not edited because this heartbeat write surface allowed only knowledge, decisions, and handoff.
- `DEPRECATION_QUEUE.md`: unchanged for the same write-surface reason.

### Decisions raised this heartbeat
- `dec-1778797938232697845` - `agent-deprecation` - Deprecate active placeholder agent `agent-29` unless an owner attaches a real role contract.

### Knowledge entries written
- `agent-visited/agent-29` (`knw-1778797990410302903`)
- `agent-audit/2026-05-14` (`knw-1778797990480341515`)
- `friction-inbox/prompt-team-agent-storage/placeholder-agent-index-drift` (`knw-1778797990480345055`) - filed for curator routing; covers adjacent placeholder/stale-index drift: untracked `agent-30` and indexed-but-missing `agent-31`.