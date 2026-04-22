# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`
`prompt-manager team decision-*`
`prompt-manager team knowledge-*`
Filesystem reads on `docs/marketing/`, `docs/monetization/`, `shared/coverage/`, `shared/campaign-drafts.jsonl`, `shared/knowledge.jsonl`.
Filesystem writes on `shared/campaign-drafts.jsonl` (append-only), `docs/marketing/notebook/*` (append-only; workarounds only, per operating rule 11).

## Primary Skills
- **campaign-content-studio** — structured campaign drafting with document-grounded AI generation.
- **seo-optimizer** — keyword and competitor analysis for blog/landing copy.
- **video-studio** — video scripting and production (draft capability).
- **documentation-health** — drafts stay concrete and readable.

## Primary Surfaces
- **Read-canon:** `docs/marketing/STRATEGY.md`, `AUDIENCES.md`, `CAMPAIGNS.md`, `BRAND.md`
- **Read-monetization:** `docs/monetization/CATALOG.md`, `PRICING.md`, `TIERS.md`, `catalog/base/*.md`, `catalog/addons/*.md`, `scenario-sku-map.json`
- **Read-state:** `shared/coverage/<sku-id>.json` (per-SKU freshness)
- **Read-own:** `shared/campaign-drafts.jsonl` (to avoid re-drafting the same artifact), `shared/knowledge.jsonl` (prior ad-run entries, challenge notes on own decisions)
- **Write-append:** `shared/campaign-drafts.jsonl`
- **Write-append (notebook):** `docs/marketing/notebook/VIDEO_WORKAROUNDS.md`, `POSTING_WORKAROUNDS.md`, etc. — workaround notes when capability-gap is raised

## Draft entry schema for campaign-drafts.jsonl

```
{
  "id": "cd-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "subscription-advertiser",
  "sku": "<sku-id-from-scenario-sku-map.json>",
  "audience": "<persona-key-from-AUDIENCES.md>",
  "positioning": "<one-sentence-claim>",
  "channel_hints": ["x-twitter", "blog", "video"],
  "campaign_ref": "<campaign-slug-or-null>",
  "artifact": {
    "format": "x-thread | blog-draft | video-script | ad-copy",
    "content": "<full-draft-text>"
  },
  "honesty_flags": {
    "engagement_estimate": "pending-telemetry",
    "feature_claims": "measured | aspirational | pending-ship"
  },
  "linked_decision_id": "<decision-id-after-publish-proposal-raised>"
}
```

## Analytical Approaches
- **Triage by deployed + stale first.** Fresh subscription SKUs with no coverage drift need less attention than deployed SKUs whose coverage has aged.
- **Feature-claim verification.** Before making any feature claim in a draft, confirm the feature exists in the current SKU via `docs/monetization/catalog/base/<bundle>.md` or per-addon file. Features in `imminent` or `candidate` status require explicit "launching [date]" framing.
- **Audience-register match.** The writing register (technical depth, casualness, jargon density) matches the persona's entry in `AUDIENCES.md`. Drafts for "indie developer" reads differently from drafts for "small team lead."
- **Retention-side hypothesis mandatory.** Every draft's decision body names how the content contributes to retention (activation moment, cross-SKU narrative, etc.) or explicitly flags "awareness-only."

## Usage Rules
- Never auto-publish. Every draft is a decision.
- Never claim shipped status for unshipped features.
- Never use paywall framing (operating rule 5).
- Never market services lines (operating rule 12).
- No bare engagement numbers — every metric carries an honesty flag.
- Raise `capability-gap` + notebook note together — silent workarounds violate operating rule 11.
- Cap at 2 drafts + 2 coverage-gaps + 1 capability-gap per heartbeat (5 new decisions max).
