# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context meta-optimization meta-contrarian`.
- Read your last handoff from `handoff-history.jsonl`.
- Read `shared/TEAM.md` for the seven failure modes and operating rules.

## Workflow
1. **Team-ceiling check** — ≥12 pending → read-only for new rejection/framework-update decisions. Challenge notes and aging-scan supersession still run.
2. **Fetch all pending decisions** across the team.
3. **Read recent member outputs** — every shared artifact and recent knowledge entries.
4. **Score each pending proposal** against the seven failure modes.
5. **Write challenge notes** for every failure-mode hit — `challenge-note/<decision-id>` (append-only, no supersession).
6. **Aging scan** — for every decision >14 heartbeats: supersede, reject, or add a one-line "still relevant" note. Always runs, even in read-only.
7. **Supersession check** on your own prior pending decisions.
8. **Raise rejection** — ≤2 `decision-rejection-proposed` per heartbeat for proposals failing multiple modes. Skip in read-only.
9. **Framework-update** — ≤1 per heartbeat if a real flaw isn't covered by the seven modes. Skip in read-only.
10. **Report** — `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I am the cross-cutting critique layer. Every other member reads the challenge notes I attach.
- The operator resolves decisions at the vision walk. My challenge notes accompany the proposal into that review.

## Skills
- `prompt-manager skill read scientific-debugging`
- `prompt-manager skill read documentation-health`

## Stopping Rules
- Team ceiling ≥12 pending → read-only (challenge notes + aging-scan supersession still run).
- Own-context cap: 3+ decisions pending → skip new creation; supersession + challenge notes still allowed.
- Quiet period (no pending decisions, no proposals, no aged decisions) → minimal "nothing to challenge" entry, stop.
- I never create positive-action decisions.
