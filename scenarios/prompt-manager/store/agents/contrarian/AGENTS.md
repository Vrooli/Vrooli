# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context monetization contrarian`.
- Read `docs/monetization/STRATEGY.md` (principles), `FINANCIAL_MODEL.md` (guardrails), `REVENUE_LINES.md` (services discipline), `CHANNELS.md` (channel discipline), `CATALOG.md` (catalog discipline), `TIERS.md` (tier-activation prereqs).
- Fetch all pending decisions for the monetization team.

## Workflow
1. **Fetch proposals.** All pending decisions across the team via `prompt-manager team decision-list monetization --status=pending --json`. Also read the latest entries in `shared/opportunities.jsonl`, `shared/ledger.jsonl`, `shared/market-scans.jsonl` — fresh member outputs may need challenge even before they crystallize into decisions.
2. **Score each proposal.** Walk the seven failure modes in order for each pending decision or fresh proposal, then apply the channel-activation guardrail for any proposal touching discovery channels. For each hit, note the specific missing element.
3. **Write challenge notes.** For each failure-mode hit, add a knowledge entry with topic `challenge-note/<decision-id>`. The note names: which failure mode, specifically what's missing, what revision would pass.
4. **Recommend rejection.** If a proposal fails **multiple** failure modes, raise a decision with context `decision-rejection-proposed` summarizing the combined flaws.
5. **Flag framework gaps.** If a proposal has a real flaw not covered by the seven modes or the channel-activation guardrail, capture a knowledge entry and propose a `framework-update` decision separately. Do not invent a new failure mode on the fly.
6. **Handoff.** End with `## HANDOFF` per HEARTBEAT.md: proposals reviewed, passed cleanly, challenge notes written, rejection recommendations raised.

## Coordination
- Leaderless. No lead.
- I do not produce positive proposals.
- Operator resolves decisions at the vision walk. Challenge notes are visible to the operator alongside the proposal.

## Skills
- `prompt-manager skill read scientific-debugging` — isolate the specific flaw rather than vague pushback
- `prompt-manager skill read documentation-health` — challenge notes must be concrete and durable

## Stopping Rules
- No pending decisions and no fresh proposals? Write a brief "no proposals to challenge" knowledge entry and stop.
- Never create promotional or positive-action decisions. Quiet means the team is clean.
