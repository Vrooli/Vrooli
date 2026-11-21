# Campaigns

Index of currently-active and recently-closed marketing campaigns. Brand-manager proposes new campaigns via `campaign-launch-proposal` decisions; the operator accepts; the approved work item triggers an entry in this file.

**Write rule:** operator-curated via accepted `campaign-launch-proposal` decisions. Brand-manager proposes; does not edit directly.

## Active campaigns

Each entry below is the campaign **plan** approved by the operator: theme, audience, channels, hypotheses, and the artifact-slot budget. It is not the live status. Current campaign and draft status, the next action per draft, and coverage are computed by the instrument (`content-desk.board-read`); read them there. The plan-of-record holds the intention; a status line here is the declaration at approval time, not a running counter.

### Aquila launch 2026-09-17

**Slug:** `aquila-launch-2026-09-17`
**Status:** active
**Theme:** A truthful first-marketed look at Aquila, the browser-based terminal workspace with durable, reconnect-safe sessions — enough real evidence to prompt an operator to try it on their own instance.
**Audience(s):** Primary: terminal-centric engineers and technical operators (Offer Desk `developer`, status `IDEA` — evidence pending). Secondary: operators who need terminal access from mobile or a second device. No persona is a validated customer segment until an owner supplies evidence.
**Scenarios:** `web-console`
**Launch window:** 2026-09-17
**Duration:** open-ended through the September 17 target
**Channels:** x-twitter (`launch-thread`, `single-image-ad`), blog (`long-form-essay`, `technical-tutorial`), youtube (`demo-recording`, `short-form-video`)
**Acquisition hypothesis:** Reach terminal-centric engineers where they already read, then prove the product is real with unedited demo evidence.
**Retention hypothesis:** Durable session continuity and multi-pane visibility give returning-operator value; measured through Channel Manager receipts and metrics.
**Linked SKUs:** `web-console` (Offer Desk deliverable `82881311`, `TRIGGER_MET`, release rank 1)
**Source decision:** operator execution authorization 2026-09-11; campaign registered directly in Content Desk (`campaign-launch-proposal` route not live — see effort `recovery.jsonl`)
**Outstanding artifact slots:**
- [x] x-twitter: launch-thread (draft `7713e7a5`)
- [x] blog: technical-tutorial (draft `8d80f973`)
- [x] blog: long-form-essay (draft `c3fe4048`)
- [ ] x-twitter: single-image-ad (post type `v0` — needs paired skill)
- [ ] youtube: demo-recording (post type `v0` — needs paired skill)
- [ ] youtube: short-form-video (post type `v0` — needs paired skill)
**Close note:** open

When a `campaign-launch-proposal` is accepted, append an entry in the shape below.

## Campaign entry shape

```markdown
### <Campaign Name>

**Slug:** `<kebab-case-slug-matches-campaign_ref-in-draft-metadata>`
**Status:** active | closed
**Theme:** <one-sentence hypothesis>
**Audience(s):** <persona-key or keys from AUDIENCES.md>
**Scenarios:** <one or more scenario names whose launch assets this campaign supports>
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
- Operator updates the campaign's outstanding artifact slots as the producer drafts them and they are released (this can be low-overhead — a tick-box pass during vision walks). Content Desk now stores the scenario dimension and derives slot state with `content-desk campaigns launch-assets --scenario <name>`.
