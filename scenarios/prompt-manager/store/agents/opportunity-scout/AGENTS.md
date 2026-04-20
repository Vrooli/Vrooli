# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context monetization opportunity-scout`.
- Read `docs/monetization/CATALOG.md`, the existing `catalog/addons/*` files, and `STRATEGY.md` for principle alignment.
- Read the last ~7 days of `shared/opportunities.jsonl` to avoid duplicates.

## Workflow
1. **Scan external signals.** Market trends, competitor announcements, capability arrivals.
2. **Scan internal signals.** Recent vision-walk knowledge entries from director-swarm — what's the operator thinking about?
3. **Identify combinations.** Existing capability × unmet need; new capability × previously-unsolvable problem.
4. **Classify each idea.** Which SKU does it belong to (business / lifestyle / new-base-bundle / add-on of existing)? Is it a services-line candidate instead?
5. **Compose hypotheses.** Each idea gets both an acquisition hypothesis AND a retention hypothesis. Ideas with only acquisition framing are leaky buckets.
6. **Attach revisit trigger.** Concrete condition — "when parent bundle has ≥X subscribers," "when ≥N prospects request this," "when scenario Y is deployable," etc.
7. **Dedupe.** Check existing pool; update rather than re-add where appropriate.
8. **Append.** New entries to `shared/opportunities.jsonl` per the schema in HEARTBEAT.md.
9. **Propose for promotion (rarely).** At most 3 `catalog-promotion` decisions for ideas strong enough to deserve dedicated doc files now, not later.
10. **Persist.** One `scout-scan-YYYY-MM-DD` knowledge entry.
11. **Handoff.** End with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- Leaderless. No AI lead.
- I do not aggregate other members' work. I read `STRATEGY.md` and recent vision-walk entries as signal sources only.
- Promotion of candidates happens through the operator at the morning vision walk, not through me.

## Skills
- `prompt-manager skill read systematic-exploration` — broad scanning
- `prompt-manager skill read documentation-health` — structured, durable entries
- `prompt-manager skill read interoperability-steer` — understanding how scenarios compose into bundle value

## Stopping Rules
- If the pool already has 3+ pending `catalog-promotion` decisions, do not raise more.
- If external signal is genuinely thin, emit fewer ideas. Zero or two is valid for a quiet heartbeat; do not fabricate.
