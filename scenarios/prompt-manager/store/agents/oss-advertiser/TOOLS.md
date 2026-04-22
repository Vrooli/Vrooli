# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>` — primary: `x-dev-log`.
`prompt-manager team decision-*`
`prompt-manager team knowledge-*`
`prompt-manager scenario status <scenario>` — health check on x-dev-log data sources.
Filesystem reads on `docs/marketing/`, `shared/coverage/oss-platform.json`, `shared/campaign-drafts.jsonl`, `shared/knowledge.jsonl`.
Filesystem writes on `shared/campaign-drafts.jsonl` (append-only), `docs/marketing/notebook/*` (append-only; workarounds only).

## Primary Skills
- **x-dev-log** — mines git-control-tower, agent-manager, swarm-manager, app-issue-tracker for interestingness-scored story arcs. PRIMARY tool — invoke every heartbeat.
- **campaign-content-studio** — longer-form OSS narratives and architecture posts.
- **seo-optimizer** — blog/landing-copy discoverability.
- **video-studio** — feature demos and architecture walkthroughs (draft capability; notebook-workaround expected until the scenario ships).
- **documentation-health** — drafts stay concrete and readable.

## Data-source Scenarios (via x-dev-log skill)
- `git-control-tower` — commit history and audit entries
- `agent-manager` — agent runs, events, costs
- `swarm-manager` — backlog items and transitions
- `app-issue-tracker` — issues and investigations

If any scenario is unhealthy, the dev log source is incomplete — raise `capability-gap` rather than ship a partial thread.

## Primary Surfaces
- **Read-canon:** `docs/marketing/STRATEGY.md`, `AUDIENCES.md`, `CAMPAIGNS.md`, `BRAND.md`
- **Read-monetization-adjacent:** `docs/monetization/STRATEGY.md` (OSS positioning rules live here too)
- **Read-state:** `shared/coverage/oss-platform.json` (synthetic SKU for OSS narrative freshness)
- **Read-own:** `shared/campaign-drafts.jsonl`, `shared/knowledge.jsonl`
- **Write-append:** `shared/campaign-drafts.jsonl`
- **Write-append (notebook):** `docs/marketing/notebook/VIDEO_WORKAROUNDS.md`, `DEV_LOG_CRAFT.md`, etc. — workaround notes when capability-gap raised

## Draft entry schema for campaign-drafts.jsonl

Uses the same schema as subscription-advertiser. For OSS drafts, `sku` is always `"oss-platform"`. `audience` is an OSS-persona key from `AUDIENCES.md`. `positioning` states the OSS framing claim (collaboration, agents-as-builders, credibility). The artifact content embeds citations to the mined sources (commit hashes, run ids, issue refs) per `x-dev-log`'s output contract.

## Analytical Approaches
- **Mining score precedence.** Let `x-dev-log`'s interestingness scoring drive selection. Don't override on taste unless a draft would repeat last heartbeat's arc.
- **Story-arc grouping.** Bundle related items into a thread. 3-7 tweets per thread, 500-2000 words for a blog, per platform rules in `CHANNELS.md`.
- **Sanitize.** Strip paths, emails, keys per `x-dev-log` guardrails — non-negotiable.
- **Work-in-progress labeling.** If mined items are incomplete (PR open, not merged), label explicitly — don't claim shipped status.
- **Framing check.** Before raising publish-proposal, re-read `STRATEGY.md`'s OSS positioning rules. If the draft accidentally paywall-frames or leak-frames, revise.

## Usage Rules
- Never auto-publish.
- Never overclaim WIP as shipped.
- Never paywall-frame or leak-frame.
- Never ship a dev log when data sources are unhealthy — raise capability-gap instead.
- Silent workarounds violate operating rule 11 — raise `capability-gap` + notebook note together.
- Cap at 2 drafts + 1 coverage-gap + 1 capability-gap per heartbeat (4 new decisions max).
- No bare engagement numbers — every metric carries an honesty flag.
