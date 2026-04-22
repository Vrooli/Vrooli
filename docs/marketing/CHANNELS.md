# Channels

Per-platform publishing rules. Publisher references this for variant production, scheduling, and polish. Advertisers reference it for platform-appropriate drafting.

**Write rule:** operator-curated via accepted `channel-update` decisions. Publisher proposes drift corrections; does not edit directly.

## X / Twitter

**Active:** yes
**Primary user:** oss-advertiser (dev logs), subscription-advertiser (launch threads).
**Rules:**
- Max 280 chars per tweet (hard platform limit).
- Thread length: 3-7 tweets. Hook in the first tweet — if the first tweet doesn't land, the thread doesn't matter.
- Sanitize paths, emails, credentials per `x-dev-log` guardrails. Non-negotiable.
- Threads cite sources (commit hashes, run ids, issue links) rather than vague claims.
- Every metric in a thread carries an honesty flag (`pending-telemetry` is fine; unflagged is a violation).
- No auto-posting. Always goes through an approved `content-publish-proposal`.

**Revisit:** when X platform rules change (API, length limits, format).

---

## Blog

**Active:** yes (self-hosted)
**Primary user:** subscription-advertiser (longer-form SKU narratives), oss-advertiser (architecture write-ups).
**Rules:**
- 500-2000 words. Technical depth welcome.
- Include code snippets when they carry the story.
- End with a concrete invitation (try it, read more, contribute, book a demo).
- Every metric carries an honesty flag.
- SEO checks via `seo-optimizer` at polish time.
- No auto-publish.

**Revisit:** when we migrate blog hosting or change the CMS.

---

## Video

**Active:** partial (production via notebook workaround; `video-studio` scenario not yet shipped).
**Primary user:** subscription-advertiser (SKU demos), oss-advertiser (architecture walkthroughs).
**Rules:**
- Production: currently manual; workarounds documented in `notebook/VIDEO_WORKAROUNDS.md`.
- Lengths: 30-60s for social clips, 2-5min for demos, 5-15min for architecture walks.
- Captions mandatory.
- Every metric in the description carries an honesty flag.
- No auto-publish.

**Revisit:** when `video-studio` scenario ships. At that point, expect this entry to tighten.

---

## LinkedIn

**Active:** placeholder.
**Primary user:** subscription-advertiser (small-team-lead audience), when activated.
**Rules:** not yet codified.

**Revisit:** when publisher releases first artifact to LinkedIn — at that point, the first `channel-update` decision populates the rules section with observed-successful patterns.

---

## Cross-channel rules

- **Variant integrity.** Every variant of an approved draft traces to the same proposal and same positioning claim. Diverging claims = bug.
- **Honesty flags preserved during polish.** Publisher never smooths away `pending-telemetry` or `estimate` flags.
- **Voice preserved during polish.** Publisher corrects typos and platform-rule violations; does not rewrite advertiser voice into house style.
- **Collision check before scheduling.** Publisher queries `publish-log.jsonl` + scheduled queue for same-day collisions on the same channel.
