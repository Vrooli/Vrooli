### Domain worked this heartbeat
- agent

### Target picked
- `run-introspector` — direct handoff to me in `RUN_LESSONS.md` 2026-04-23 entry; concrete prose edit with baseline + measurement already drafted by the originating member.

### Disposition
- improve

### Evidence
- `RUN_LESSONS.md` 2026-04-23, run `60116710`: run-introspector spent a heartbeat investigating an `exit_code=429` run that was actually a clean investigation report.
- Root cause: `detectRateLimit` in `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go:1518-1559` substring-matches "rate limit" in the final assistant message regardless of `IsError`.
- Baseline: 2/22 (~9%) FAILED runs match (ids `60116710`, `e08357a4`).
- Current `run-introspector/HEARTBEAT.md` step 3 has no false-positive gate; tier-1 contaminated runs get picked first every time.
- Run-introspector health 0.72; never visited before by this member.

### Expected delta (if change proposed)
- Tier-1 false-positive investigations: ~9% of picks → ~0%.
- ~1 wasted heartbeat per ~11 runs recovered.
- Measurement: grep `RUN_LESSONS.md` 7 HB after merge — zero new tier-1 lessons opened on 429+completion-text runs; gate becomes a no-op once scenario-qa fixes the underlying matcher.

### Artifacts updated
- `AGENT_AUDIT.md`: row added for `run-introspector` (2026-04-24, health 0.72, rating 3, improve).
- `DEPRECATION_QUEUE.md`: unchanged (no pruning proposal).

### Decisions raised this heartbeat
- `dec-1777069916962818847` · `agent-improvement` · Edit `run-introspector/HEARTBEAT.md` step 3 to add a tier-1 false-positive verification gate for `exit_code=429` runs whose `error_msg` is substantive markdown report text.

### Knowledge entries written
- `agent-visited/run-introspector` (`knw-1777069883310471139`) — supersedes prior (none).
- `agent-audit-2026-04-24` (`knw-1777069892849442354`) — supersedes `agent-audit-2026-04-23`.