# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`
`prompt-manager team decision-*`
`prompt-manager team knowledge-*`
`prompt-manager scenario status social-media-scheduler` — health of scheduling tool.
Filesystem reads on `docs/marketing/`, `docs/monetization/scenario-sku-map.json`, `shared/campaign-drafts.jsonl`, `shared/publish-log.jsonl`, `shared/coverage/*.json`, `shared/knowledge.jsonl`.
Filesystem writes on `shared/publish-log.jsonl` (append-only), `shared/coverage/<sku-id>.json` (rewrite per update), `docs/marketing/notebook/POSTING_WORKAROUNDS.md` (append-only).

## Primary Skills
- **social-media-scheduler** — the eventual automated release tool. Currently partially wired — expect workarounds.
- **seo-optimizer** — polish-time SEO checks for blog-length content.
- **campaign-content-studio** — per-platform variant generation.
- **documentation-health** — decisions stay concrete.

## Primary Surfaces
- **Read-canon:** `docs/marketing/CHANNELS.md` (platform rules), `STRATEGY.md` (voice), `CAMPAIGNS.md` (launch windows)
- **Read-monetization:** `docs/monetization/scenario-sku-map.json` (authoritative SKU list for coverage sweep)
- **Read-queue:** `prompt-manager team decision-list marketing-crew --context=content-publish-proposal --status=accepted`
- **Read-drafts:** `shared/campaign-drafts.jsonl`
- **Read-own:** `shared/publish-log.jsonl`, `shared/coverage/*.json`, `shared/knowledge.jsonl` (prior coverage snapshots)
- **Write-append:** `shared/publish-log.jsonl`
- **Write-per-update:** `shared/coverage/<sku-id>.json` (full-file rewrite per update)
- **Write-append (notebook):** `docs/marketing/notebook/POSTING_WORKAROUNDS.md` when scheduling tooling is unwired

## Publish-log entry schema

```
{
  "id": "pl-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "publisher",
  "source_decision_id": "<approved-content-publish-proposal-id>",
  "source_draft_id": "<campaign-drafts-jsonl-id>",
  "sku": "<sku-id-or-oss-platform>",
  "campaign_ref": "<campaign-slug-or-null>",
  "released": [
    {
      "channel": "x-twitter | blog | video | linkedin | ...",
      "platform_ref": "<url-or-post-id>",
      "released_at": "YYYY-MM-DDTHH:MM:SSZ",
      "method": "scheduler-automated | manual-workaround",
      "workaround_note": "<notebook-ref-if-manual-else-null>"
    }
  ],
  "variants_produced": <count>,
  "honesty_flags_preserved": true
}
```

## Coverage file schema (`shared/coverage/<sku-id>.json`)

```json
{
  "sku_id": "<sku-id-from-scenario-sku-map.json-or-oss-platform>",
  "display_name": "<human-friendly-name>",
  "last_touched": "YYYY-MM-DD",
  "status": "fresh | stale | missing",
  "stalenessPolicy": { "windowDays": 30 },
  "channels": {
    "x-twitter": { "last_posted": "YYYY-MM-DD-or-null", "artifact_ref": "publish-log.jsonl#<id>-or-null" },
    "blog": { "last_posted": "YYYY-MM-DD-or-null", "artifact_ref": "publish-log.jsonl#<id>-or-null" },
    "video": { "last_posted": "YYYY-MM-DD-or-null", "artifact_ref": "publish-log.jsonl#<id>-or-null" }
  },
  "next_review_date": "YYYY-MM-DD",
  "notes": "<operator-or-publisher-supplied-context>"
}
```

For `sku_id: "oss-platform"`, `display_name: "Vrooli (OSS narrative)"`. The channels map can include any platform advertisers target; add channels lazily as they're used.

## Analytical Approaches
- **Polish ≠ rewrite.** If the advertiser's voice is consistent and on-canon, preserve it. If the draft misrepresents a feature or breaks a platform rule, flag and revise; otherwise leave alone.
- **Variant-pack integrity.** Every variant in a pack traces back to the same approved proposal and same positioning claim. Diverging claims = bug.
- **Scheduling conflict check.** Before scheduling, query `publish-log.jsonl` + currently-scheduled queue for same-day collisions on the same channel. Space out when possible.
- **Coverage-sweep rigor.** Every SKU in `scenario-sku-map.json` gets a coverage file, even if `status: missing`. Missing files hide gaps from advertisers.

## Usage Rules
- Never auto-publish unapproved drafts.
- Never bypass `social-media-scheduler` without raising capability-gap + notebook note.
- Never rewrite advertiser voice — polish is structural only.
- Never drop honesty flags during polish.
- One release = one `publish-log.jsonl` entry + one `coverage/<sku-id>.json` update. Both or neither.
- Cap at 2 variant-pack proposals + 1 channel-update + 1 capability-gap per heartbeat.
