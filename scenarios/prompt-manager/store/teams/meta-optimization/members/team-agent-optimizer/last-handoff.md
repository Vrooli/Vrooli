### Domain worked this heartbeat
- agent

### Target picked
- `quality-auditor` — lowest health (0.56) in the 21-agent graph and the only agent returned by `graph skillless-agents`. Concrete fix available: AGENTS.md already names 11 skills in prose; peer `programmatic-qa-runner` in the same team has the full AGENTS/SOUL/TOOLS triad.

### Disposition
- improve

### Evidence
- Health 0.56 (next-lowest 0.60); `graph skillless-agents` returns `quality-auditor` as sole entry.
- Folder contains only `SOUL.md` + `AGENTS.md`; no `TOOLS.md`.
- `agent.json` `fileOrder` = `["SOUL.md", "AGENTS.md"]`, explicitly omitting TOOLS.md.
- AGENTS.md lines 26–38 name 11 steer skills (7 Tier-1 + 4 Tier-2) as the rotation, but none appear as outbound graph edges — only 2 `cli:*` edges total.
- Peer `programmatic-qa-runner` has 8 outbound edges including `bold-listed` skill edges, health 0.60.

### Expected delta (if change proposed)
- Remove `quality-auditor` from `graph skillless-agents` (1 → 0).
- Add ~11 outbound `bold-listed` / `cli-read` skill edges.
- Outgoing-edges factor 0.40 → ~1.00; predicted overall health 0.56 → 0.65–0.72.
- Measurement: re-run `graph node quality-auditor` and `graph skillless-agents` post-merge; 7 HB revisit.

### Artifacts updated
- `AGENT_AUDIT.md`: bootstrapped; added first row for `quality-auditor` with decision id.
- `DEPRECATION_QUEUE.md`: unchanged (no pruning proposal this HB).

### Decisions raised this heartbeat
- `dec-1776983541260124317` · `agent-improvement` · Add TOOLS.md to `quality-auditor` + update `agent.json` fileOrder so the 11 steer skills already in AGENTS.md become graph edges.

### Knowledge entries written
- `agent-visited/quality-auditor` (`knw-1776983516231938462`) — supersedes prior (none).
- `agent-audit-2026-04-23` (`knw-1776983522308520876`) — supersedes prior (none).