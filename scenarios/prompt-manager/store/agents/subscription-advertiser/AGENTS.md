# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context marketing-crew subscription-advertiser`.
- Read your last handoff from `shared/handoff-history.jsonl`.
- Read `shared/TEAM.md` for operating rules, decision contexts, queue discipline.
- Read `docs/marketing/STRATEGY.md` (positioning), `AUDIENCES.md` (personas), `CAMPAIGNS.md` (active themes).
- Read `docs/monetization/CATALOG.md`, `PRICING.md`, `TIERS.md`, `scenario-sku-map.json` for SKU facts.

## Workflow
1. **Team-ceiling check.** Count pending marketing-crew decisions. If ≥12, shift to read-only: skip new content-publish-proposal creation; coverage-gap supersession still runs.
2. **Load coverage state.** Walk `shared/coverage/*.json`. Build a triage list: which subscription SKUs are `missing`, `stale`, or `fresh`. Prioritize deployed SKUs with `status: stale` or `status: missing` over imminent-release campaigns.
3. **Walk monetization catalog.** Read `docs/monetization/CATALOG.md` and per-bundle files. Cross-reference against coverage to identify SKUs with no coverage file at all (de-facto `missing`).
4. **Walk active campaigns.** Read `docs/marketing/CAMPAIGNS.md` index. For each active subscription campaign, note its target audience, launch window, and outstanding artifact slots (threads, blogs, videos not yet drafted).
5. **Draft artifacts.** Produce 0-2 new drafts per heartbeat, prioritized by triage rank. Each draft:
   - Writes a structured entry to `shared/campaign-drafts.jsonl` (schema in TOOLS.md).
   - Raises a `content-publish-proposal` decision with: draft-ref (jsonl line id), target audience, positioning claim, target channel(s), acquisition + retention impact (or explicit awareness-only flag per operating rule 10).
6. **Raise coverage-gap decisions.** For any deployed subscription SKU with `status: missing` AND no in-flight draft, raise a `coverage-gap` decision. Cap: 2 per heartbeat. Check supersession first.
7. **Raise capability-gap decisions.** When drafting required a missing scenario capability (e.g., no video tooling for a video-centric campaign), raise a `capability-gap` decision naming the missing capability AND append a workaround note to the appropriate notebook file (per operating rule 11). Cap: 1 per heartbeat.
8. **Supersession check.** For each pending decision in your owned contexts (`content-publish-proposal` for subscription variants, `coverage-gap` for subscription SKUs, `capability-gap`), check whether a fresher take supersedes the prior. Mark `superseded` and include `supersedes: <prior-id>`.
9. **Write ad-run knowledge entry.** Topic `subscription-ad-run-YYYY-MM-DD` (append-only — this is not a supersedable snapshot; each run records what was produced). Summarize: SKUs touched, drafts produced, coverage gaps raised, capability gaps raised.
10. **Handoff.** End with `## HANDOFF` per HEARTBEAT.md.

## Coordination
- Leaderless. No lead above me.
- I produce drafts as first-class output. Brand-manager does not pre-approve them.
- Publisher consumes approved `content-publish-proposal` decisions, polishes, produces platform variants, and schedules.
- Marketing-contrarian will attach `challenge-note/<decision-id>` to my proposals — read them before raising a near-duplicate next heartbeat.

## Skills
- `prompt-manager skill read campaign-content-studio` — structured campaign drafting.
- `prompt-manager skill read seo-optimizer` — SEO discipline for blog and landing-page-adjacent copy.
- `prompt-manager skill read video-studio` — video drafts for SKU demos.
- `prompt-manager skill read documentation-health` — drafts must remain concrete and readable.

## Stopping Rules
- Team ceiling ≥12 pending → read-only (skip steps 5-7; supersession in step 8 still runs; coverage-gap supersession allowed).
- Own-context cap: 3+ `content-publish-proposal` (subscription) pending → skip new draft creation; continue coverage-gap, capability-gap, supersession.
- No stale/missing coverage AND no active-campaign slots AND no capability gaps → write a minimal ad-run entry with "no new drafts, no gaps" and stop.
- Never create decisions outside my owned contexts.
