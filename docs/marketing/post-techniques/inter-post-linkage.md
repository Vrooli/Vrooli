# Technique: Inter-Post Linkage

**Status:** Extracted from `STRATEGY.md`'s Dev-log narrative principles §6 on 2026-04-28 (walk #5 divergence #3, Action B). Originally added 2026-04-27 via accepted framework-update `dec-1777300532504756717`.

## Rule

Each post in a series connects to its predecessor.

**Mechanism:** after publishing, the operator pastes the post URL back into `marketing-crew/shared/publish-log.jsonl` (`post_url` field). The next post in the series cites that URL as `previous_post_url` (in the final reply, or as a visible link). Readers landing on the new post can find the prior chain.

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

## Mechanism details

The publish-log entry shape (per `marketing-crew/shared/publish-log.jsonl`):

```jsonl
{
  "draft_id": "<id>",
  "series_id": "<series-id>",
  "post_index_in_series": <integer>,
  "post_url": "<populated-after-publish>",
  "previous_post_url": "<from-prior-entry-in-series>"
}
```

`social-media-scheduler` (planned scenario, initiative-proposal `dec-1777312920606447957`) is intended to own the URL paste-back roundtrip and the series-chaining lookup so it isn't operator-manual.

## Cross-references

- `post-techniques/essay-shape.md` — companion technique; the conclusion of an essay-shape post is the natural place to embed `previous_post_url` and the invitation to follow for the next post.
- `post-techniques/no-internal-numbering-externally.md` — `post_index_in_series` is the *only* numbering allowed externally; this technique uses it correctly.
- `marketing-crew/shared/publish-log.jsonl` — the persistence surface this technique reads/writes.
- `social-media-scheduler` scenario (planned) — future automation of the manual paste-back step.
