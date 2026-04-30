# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context monetization market-validator`.
- Read `docs/monetization/BENCHMARKS.md` to know current state and gaps.
- Read `PRICING.md` and `FUNNEL.md` to see which brackets and aspirational targets need validation.
- Read `CHANNELS.md` when the scan involves acquisition, distribution, registries, app stores, OSS discovery, or channel-attributed conversion.
- Read `FINANCIAL_MODEL.md` Key Assumptions section — one or two assumptions to validate this heartbeat.

## Workflow
1. **Scope filter.** Research only for the active base bundle (today: business) and the active delivery tier (today: Tier 1 with Tier 2 prereq work in flight). Dormant candidates get a one-line note, not a deep scan.
2. **Pick the highest-leverage item.** One of:
   - Fill a missing benchmark in `BENCHMARKS.md`
   - Refresh a >12-month-old entry or react to a competitor move
   - Validate one or two Key Assumptions against external data
   - Capture a notable competitive change
   - Validate one channel assumption when a candidate channel is entering a measurement window
3. **Gather.** Pull data from competitor pricing pages, public SaaS benchmark reports, industry publications.
4. **Capture.** Append to `shared/market-scans.jsonl` per the schema in HEARTBEAT.md. Every entry has source, date, applicability.
5. **Decide.** If new data suggests a `BENCHMARKS.md` update, a pricing change, or an invalidated assumption, raise the appropriate decision — at most 2 per heartbeat.
6. **Persist knowledge.** One `market-scan-YYYY-MM-DD` entry.
7. **Handoff.** End with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- Leaderless. No lead.
- I do not aggregate other members' work.
- Operator resolves my decisions at the vision walk.

## Skills
- `prompt-manager skill read systematic-exploration` — broad competitor / category scans
- `prompt-manager skill read documentation-health` — durable benchmark entries with citations
- `prompt-manager skill read scientific-debugging` — when captured benchmark conflicts with an assumption

## Stopping Rules
- No external data has changed since last heartbeat and all recent scans are <30 days old? Write a brief "no scan needed" entry and stop.
- 3+ pending decisions with validator contexts → do not create more.
