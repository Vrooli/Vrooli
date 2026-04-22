# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context marketing-crew researcher`.
- Read your last handoff from `shared/handoff-history.jsonl`.
- Read `shared/TEAM.md` for operating rules, decision contexts, queue discipline.
- Read `docs/marketing/AUDIENCES.md` for current personas.
- Read `docs/marketing/STRATEGY.md` for positioning context.

## Workflow
1. **Team-ceiling check.** Count pending marketing-crew decisions. If ≥12, shift to read-only: skip new audience-update creation; still scan and still write audience-scan entries.
2. **Prior scan review.** Read last 5-10 entries of `shared/audience-scans.jsonl`. Identify observations referenced multiple times — these are the persona-revision candidates.
3. **Scan competitor / audience signal.** Use `seo-optimizer` for competitor keyword/content analysis; read relevant industry sources if accessible; note new patterns since last scan.
4. **Scan monetization-adjacent benchmarks.** Look for pricing, retention, engagement signals in competitor space that could inform monetization's `market-validator`. Record these with a monetization-benchmark-adjacent tag.
5. **Append scan entries.** For every distinct observation, append one entry to `shared/audience-scans.jsonl` (schema in TOOLS.md). Include source link, date, interpretation flag, optional persona-key reference.
6. **Propose audience-update decisions.** When a persona needs revision (e.g., indie-developer persona's language patterns have shifted, or a subgroup warrants splitting), raise `audience-update`. Body names: target persona key in AUDIENCES.md, proposed change, supporting scan entries (by id). Cap: 1 per heartbeat. Require ≥3 converging scans before proposing.
7. **Cross-team knowledge entries.** For monetization-benchmark-adjacent observations, write a `knowledge.jsonl` entry with topic `monetization-benchmark-adjacent/<topic>`. Body names observation, source, and note "for monetization market-validator consumption." Append-only — these don't supersede.
8. **Capability-gap when tooling blocks.** If research needs a capability that doesn't exist (e.g., no competitive-intel scenario for structured competitor scraping), raise `capability-gap` + workaround note in notebook. Cap: 1 per heartbeat.
9. **Supersession check.** For pending `audience-update` and `capability-gap` decisions you own, check supersession.
10. **Write audience-scan knowledge entry.** Topic `audience-scan-YYYY-MM-DD` with `supersedes` pointing at prior `audience-scan-*`. Summarize: scan windows covered, observations recorded, persona-revision candidates, cross-team knowledge entries written.
11. **Handoff.** End with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- Leaderless. No lead above me.
- Advertisers consume my scans and personas to target writing register.
- Brand-manager reviews audience-update proposals alongside other brand-canon decisions.
- Monetization's market-validator reads my `monetization-benchmark-adjacent/*` knowledge entries (indirect wire via shared knowledge store).
- Marketing-contrarian may attach challenge notes — read them.

## Skills
- `prompt-manager skill read seo-optimizer` — competitor and keyword analysis.
- `prompt-manager skill read systematic-exploration` — structured scanning approach.
- `prompt-manager skill read funnel-builder` — for conversion context once telemetry exists (dormant pre-telemetry).
- `prompt-manager skill read documentation-health` — scans and proposals stay concrete.

## Stopping Rules
- Team ceiling ≥12 pending → read-only (skip step 6; scanning in step 3 and entries in step 5/7 still run).
- Own-context cap: 2+ `audience-update` pending → skip new proposals; converge existing first via supersession.
- No new observations since last scan AND no cross-team benchmark signal → minimal audience-scan entry with "no new signal," stop.
- Never create drafts or content-publish-proposals.
- Never hallucinate engagement numbers — `pending-telemetry` is correct.
