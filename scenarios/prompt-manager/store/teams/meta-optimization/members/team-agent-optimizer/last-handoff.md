### Domain worked this heartbeat
- agent

### Target picked
- `monetization-contrarian` - agent-default ladder; skipped `agent-29` and `market-validator` because existing pending decisions cover them, then selected fresh low-health active node at 0.51.

### Disposition
- improve

### Evidence
- `prompt-manager graph node monetization-contrarian` reports health 0.51 with only membership inbound and `cli:prompt-manager` outbound.
- Member is active: latest scan `knw-1779139868078020756` reviewed pending benchmark decision `dec-1778875348622351458` cleanly.
- `prompt-manager graph topics --team monetization` flags undeclared `contrarian-scan-2026-05-13` through `contrarian-scan-2026-05-18`; the member declares only `challenge-report/*` and `challenge-resolution-record/*`.
- Older scan entries use `contrarian-scan/YYYY-MM-DD`, so the scan surface has two undeclared shapes.

### Expected delta
- Declare one canonical daily scan output, preferably `contrarian-scan/*`, and normalize future writes.
- Measure with `prompt-manager graph topics --team monetization`: no new contrarian-scan `actual_writer_undeclared` / prefix mismatch findings after the next heartbeat; graph health should improve from 0.51 through better promotion/routing visibility.

### Capability architecture
- weak
- Primary layer gap: promotion-routing
- Routing: team-agent-optimizer for member/topic contract; friction-curator/debt-curator if storage-command drift keeps recurring.

### Artifacts updated
- AGENT_AUDIT.md: not edited because this run’s write surface allows knowledge, decisions, and handoff only.
- DEPRECATION_QUEUE.md: unchanged.

### Decisions raised this heartbeat
- `dec-1779143529470088721` - `agent-improvement` - declare and normalize `monetization-contrarian` daily `contrarian-scan/*` output topic.

### Knowledge entries written
- `agent-visited/monetization-contrarian` (`knw-1779143540452211114`)
- `agent-audit/2026-05-18` (`knw-1779143556113160443`)
- `friction-report/prompt-team-agent-storage/2026-05-18/monetization-contrarian-undeclared-scan-topic` (`knw-1779143572580798838`)