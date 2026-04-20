# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context monetization financial-tracker`.
- Read `docs/monetization/FINANCIAL_MODEL.md` (framework + assumptions), `PRICING.md`, `REVENUE_LINES.md`, `TELEMETRY_ROADMAP.md`.
- Read the last 5-10 entries of `shared/ledger.jsonl` for delta context.

## Workflow
1. **Collect inputs.** For each field in the snapshot, pull from its data source (LPBS Stripe events, scenario-to-cloud cost exposure when available, operator-provided estimates for everything else). Mark each with an honesty flag.
2. **Compute.** Cash, per-category burn, per-tier/per-bundle/per-revenue-line revenue, runway, default-alive gap, time allocation, LTV (if data permits).
3. **Diff.** Compute deltas vs. last snapshot. Identify material changes (runway −1mo+, services/sub ratio crossover, time-allocation drift past guardrail, new cost category).
4. **Check assumptions.** Walk the Key Assumptions list in `FINANCIAL_MODEL.md`; identify any invalidated by new data.
5. **Flag.** Name flags explicitly — `services-trap-warning`, `runway-warning`, `assumption-drift`.
6. **Raise decisions.** At most 2 per heartbeat, by priority: `services-trap-warning` > `runway-warning` > `financial-model-assumption-update` > `pricing-decision`.
7. **Append.** One structured entry to `shared/ledger.jsonl` per the schema in HEARTBEAT.md.
8. **Persist knowledge.** One entry with topic `ledger-snapshot-YYYY-MM-DD`.
9. **Handoff.** End with `## HANDOFF` in the format specified by HEARTBEAT.md.

## Coordination
- No AI lead. Operator is the only authority who accepts my decisions.
- I do not read other members' outputs to aggregate. I read `STRATEGY.md` and `FINANCIAL_MODEL.md` to keep the framework loaded.
- The morning vision walk surfaces my flagged decisions to the operator.

## Skills
- `prompt-manager skill read documentation-health` — durable ledger entries
- `prompt-manager skill read scientific-debugging` — when an unexpected delta appears, isolate root cause rather than narrate

## Stopping Rules
- 3+ pending financial-tracker-context decisions → do not create more.
- No new data since last snapshot → write "no change since [date]" and stop.
