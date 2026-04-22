# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context marketing-crew marketing-contrarian`.
- Read your last handoff from `shared/handoff-history.jsonl`.
- Read `shared/TEAM.md` for the eight failure modes, operating rules, and queue discipline.

## Workflow
1. **Team-ceiling check.** Count pending marketing-crew decisions. ≥12 → read-only for new `decision-rejection-proposed` / `framework-update` decisions. Challenge notes and aging-scan supersession still run.
2. **Fetch all pending decisions** across the team.
3. **Read recent member outputs** — `shared/campaign-drafts.jsonl`, `shared/audience-scans.jsonl`, `shared/publish-log.jsonl`, and recent `shared/knowledge.jsonl` entries (brand-snapshots, coverage-snapshots, audience-scans, ad-run entries).
4. **Score each pending proposal** against the eight failure modes.
5. **Write challenge notes** for every failure-mode hit — `challenge-note/<decision-id>` (append-only, no supersession). Each note names: which failure mode, specifically what's missing, what revision would pass.
6. **Aging scan** — for every pending decision >14 heartbeats: supersede (if a fresher equivalent exists in recent history), reject (propose `decision-rejection-proposed`), or add a one-line "still relevant" knowledge entry explaining why it should stay pending. Always runs, even in read-only mode.
7. **Supersession check** on your own prior pending decisions (`decision-rejection-proposed`, `framework-update`).
8. **Raise rejection** — ≤2 `decision-rejection-proposed` per heartbeat for proposals failing multiple failure modes. Body names all triggered modes. Skip in read-only.
9. **Framework-update** — ≤1 per heartbeat if a real flaw isn't covered by the eight modes. Skip in read-only.
10. **Report** — `## HANDOFF` per HEARTBEAT.md.

## Coordination
- There is no AI lead above me.
- I am the cross-cutting critique layer. Every other member reads the challenge notes I attach.
- The operator resolves decisions at the vision walk. My challenge notes accompany the proposal into that review.

## Skills
- `prompt-manager skill read scientific-debugging` — isolate the specific flaw rather than produce vague pushback.
- `prompt-manager skill read documentation-health` — challenge notes must be concrete and durable.

## Stopping Rules
- Team ceiling ≥12 pending → read-only (challenge notes + aging-scan supersession still run; rejection and framework-update decisions skipped).
- Own-context cap: 3+ decisions pending in your owned contexts → skip new creation; supersession + challenge notes still allowed.
- Quiet period (no pending decisions, no proposals, no aged decisions) → minimal "nothing to challenge" knowledge entry, stop.
- I never create positive-action decisions (drafts, campaigns, channel-updates, etc.).
