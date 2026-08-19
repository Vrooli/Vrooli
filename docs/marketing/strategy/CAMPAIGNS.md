# Campaigns

Index of currently-active and recently-closed marketing campaigns. Brand-manager proposes new campaigns via `campaign-launch-proposal` decisions; the operator accepts; the approved work item triggers an entry in this file.

**Write rule:** operator-curated via accepted `campaign-launch-proposal` decisions. Brand-manager proposes; does not edit directly.

## Active campaigns

_No active campaigns._

When a `campaign-launch-proposal` is accepted, append an entry in the shape below.

## Campaign entry shape

```markdown
### <Campaign Name>

**Slug:** `<kebab-case-slug-matches-campaign_ref-in-draft-metadata>`
**Status:** active | closed
**Theme:** <one-sentence hypothesis>
**Audience(s):** <persona-key or keys from AUDIENCES.md>
**Launch window:** <YYYY-MM-DD or ongoing>
**Duration:** <weeks or open-ended>
**Channels:** <x-twitter, blog, video, etc.>
**Acquisition hypothesis:** <what this does for new-user acquisition — OR "awareness-only: true">
**Retention hypothesis:** <what this does for activation/retention — OR "awareness-only: true">
**Linked SKUs:** <sku-ids from `offer-desk offers catalog-list --json`, or "oss-platform">
**Source decision:** <campaign-launch-proposal decision id>
**Outstanding artifact slots:**
- [ ] <channel>: <format> (e.g., `x-twitter: launch thread`)
- [ ] <channel>: <format>
**Close note:** <when closed — summary of what was produced and any postmortem reference topic>
```

## Closed campaigns

_No closed campaigns yet._

Closed campaigns stay in this file as historical record. Long-term, these may move into a separate archive file when the active set gets large enough to warrant it.

## Operator discipline

- Campaign creation: operator accepts `campaign-launch-proposal` → operator adds entry here → commit cites decision id.
- Campaign close: `brand-manager` raises `campaign-launch-proposal` with status=close plus a post-mortem reference. The close note and final artifact go through `content-desk`'s operator approval gate; the former `content-publish-proposal` work type is retired.
- Operator updates the campaign's outstanding artifact slots as the producer drafts them and they are released (this can be low-overhead — a tick-box pass during vision walks). Once `content-desk` owns campaign records, slot state is derived rather than ticked.
