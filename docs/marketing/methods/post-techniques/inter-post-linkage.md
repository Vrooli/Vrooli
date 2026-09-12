# Technique: Inter-Post Linkage

**Status:** Extracted from `STRATEGY.md`'s Dev-log narrative principles §6 on 2026-04-28 (walk #5 divergence #3, Action B). Originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

## Rule

Each post in a series connects to its predecessor.

**Mechanism:** after publishing, the operator pastes the post URL back into `content-desk publish-history records` (`post_url` field). The next post in the series cites that URL as `previous_post_url` (in the final reply, or as a visible link). Readers landing on the new post can find the prior chain.

For the first post in a series, `previous_post_url` is `null` and the conclusion invites readers to follow for future posts.

## Why it matters

A series whose posts don't reference their predecessors functionally isn't a series — each post stands alone, the audience has to guess at the connection, and the narrative arc the [essay-shape](essay-shape.md) discipline tries to build collapses across posts. Inter-post linkage is what turns N standalone posts into one ongoing story.

The mechanism is intentionally *post-publish* rather than *post-author*: the URL exists only after publish. Drafts cannot embed the actual URL of the prior post when authored; they cite the placeholder `previous_post_url` and the operator (or `social-media-scheduler` once shipped) populates the URL after the prior post lands.

## When it applies

- ✅ `dev-log` series — every post after the first.
- ✅ `scenario-spotlight` series — when a scenario gets multiple spotlights over time (different angles, different audiences, tier upgrades), they should link.
- ✅ `oss-framework` series (planned) — applies once authored.
- ➖ Standalone posts — `previous_post_url = null` is correct; no linkage required.
- ❌ Cross-series linkage — different rule. A scenario-spotlight does not have to cite a dev-log about the same scenario; if it does, that's a content choice, not this technique's mechanism.

## Cross-platform amplification

Inter-post linkage is not only intra-series-on-one-platform. The same content often lands on multiple surfaces — a thread on X, a longer-form version on a blog, a quote-tweet referencing the blog, a LinkedIn post linking to either or both. This is **cross-platform amplification**: the same idea, surfaced in different venues, with explicit linkage between them so a reader who lands on any one of them can find the others.

Empirical observation (walk #5, Post 4 — agent-dev-team video course quote-tweeting an underlying blog post): viral posts often *are* quote-tweets of an underlying blog or repository, where the X post is the hook surface and the linked surface is the body surface. The post itself is short and intriguing; the linked artifact carries the depth.

**When to use cross-platform amplification:**

- ✅ A scenario-spotlight has both a thread version (high-attention, low-depth, X) and a blog version (low-attention, high-depth) — the thread quote-amplifies or links the blog.
- ✅ An oss-framework post (planned type) drops a video tutorial as the body and uses an X post as the hook surface that links to it.
- ✅ A dev-log essay on a blog gets distilled into a thread that links back to the blog for readers who want the full context.
- ➖ Standalone short posts that don't have an underlying long-form artifact don't need this technique.
- ❌ Linking a thin restatement of the same content across platforms with no added value on each surface — that's spam, not amplification.

**Mechanism:** the Content Desk publish-history record's `post_url` is per-platform. Cross-platform amplification means the same `series_id` may have multiple entries (one per platform/surface) for the same nominal "post." A future schema extension may add a `surface` or `platform` field; today, separate entries with consistent `series_id` and a shared `draft_id` (or a paired `amplifies` reference) is the convention. The contrarian's `data_source=verifiable` check applies: every linked surface must be reachable and consistent.

## Mechanism details

The Content Desk publish-history record shape (per `content-desk publish-history records`):

```jsonl
{
  "draft_id": "<id>",
  "series_id": "<series-id>",
  "post_index_in_series": <integer>,
  "post_url": "<populated-after-publish>",
  "previous_post_url": "<from-prior-entry-in-series>"
}
```

`social-media-scheduler` (planned scenario, gated work item `dec-1777312920606447957`) is intended to own the URL paste-back roundtrip and the series-chaining lookup so it isn't operator-manual.

## Cross-references

- `post-techniques/essay-shape.md` — companion technique; the conclusion of an essay-shape post is the natural place to embed `previous_post_url` and the invitation to follow for the next post.
- `post-techniques/no-internal-numbering-externally.md` — `post_index_in_series` is the *only* numbering allowed externally; this technique uses it correctly.
- `content-desk publish-history records` — the persistence surface this technique reads/writes.
- `social-media-scheduler` scenario (planned) — future automation of the manual paste-back step.
