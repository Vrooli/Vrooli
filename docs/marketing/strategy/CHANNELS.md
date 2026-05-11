# Channels

Per-platform publishing rules and per-account state. Publisher references this for variant production, scheduling, and polish. Advertisers reference it for platform-appropriate drafting. Researcher proposes drift corrections via `channel-strategy-update` decisions.

**Write rule:** operator-curated via accepted `channel-update` decisions (per-platform-rule changes) and `channel-strategy-update` decisions (channel-level strategy and prioritization changes). Publisher proposes drift corrections; does not edit directly.

## Secrets, credentials, and account handles do NOT live here

This file is committed to Git and consumed by every marketing-crew member every heartbeat. Anything sensitive lives elsewhere:

- **Account handles, usernames, OAuth tokens, API keys, posting credentials** — never in this file. Interim home: an untracked operator-only file or vault resource. Long-run home: the `social-media-scheduler` scenario's PostgreSQL state (already designed into its scenario PRD; OAuth-token storage is a P0 capability).
- **What goes here instead:** *count* of accounts per channel, the *purpose tags* of each account (brand vs persona-actor), strategy notes, format support, and (eventually) per-channel × per-bundle conversion metrics aggregated from the scheduler.
- When referring to an account in this file, use only the role/purpose label (e.g. "primary brand account", "persona-actor: homelab-builder"), never the platform handle.

## Channel × format matrix

The matrix names which post-type from [`../catalogs/post-types/`](../catalogs/post-types/) maps to which platform. P = primary surface, S = secondary (cross-posted variant), — = not used. Status reflects whether *we* publish that combination today, not whether the platform supports it.

| Channel | dev-log (text) | scenario-spotlight (text+asset) | oss-framework (text) — *planned* | use-case-tutorial — *planned* | single-image-ad | slideshow-listicle | slideshow-tips-then-plug | infographic | narrative-talking-head | day-in-life-ugc | problem-agitate-solve | demo-recording | comparison-reel | slideshow-voiceover |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| X / Twitter | **P** | **P** | P | S | S | S | S | S | — | — | — | S | S | — |
| Blog (self-hosted) | S | **P** (long-form) | **P** | **P** | — | — | — | S | — | — | — | S | — | — |
| Video (self-hosted YouTube) | — | S | S | **P** | — | — | — | — | S | S | S | **P** | S | S |
| LinkedIn — *placeholder* | S | S | S | S | S | — | — | S | — | — | — | S | — | — |
| TikTok — *placeholder* | — | — | — | S | — | — | **P** | — | **P** | **P** | **P** | — | **P** | **P** |
| Instagram Feed — *placeholder* | — | S | — | — | **P** | S | S | **P** | — | — | — | — | — | — |
| Instagram Reels — *placeholder* | — | S | — | S | — | — | S | — | **P** | **P** | **P** | S | **P** | **P** |
| Instagram Stories — *placeholder* | — | S | — | — | S | S | S | — | S | S | S | S | S | S |
| YouTube Shorts — *placeholder* | — | S | — | S | — | — | S | — | **P** | S | **P** | S | **P** | **P** |
| Threads — *placeholder* | S | S | S | S | — | — | — | — | — | — | — | — | — | — |
| Bluesky — *placeholder* | S | S | S | — | — | — | — | — | — | — | — | — | — | — |
| Reddit — *placeholder* | S | **P** (subreddit-fit) | **P** (r/MachineLearning, r/programming) | **P** | — | — | — | — | — | — | — | S | S | — |
| HackerNews — *placeholder* | S | S | **P** (Show HN) | S | — | — | — | — | — | — | — | — | — | — |
| ProductHunt — *placeholder* | — | **P** (launch day) | — | — | S | — | — | — | — | — | — | S | — | — |
| Mastodon — *placeholder* | S | S | S | — | — | — | — | — | — | — | — | — | — | — |

**Reading the matrix.** A cell answers: "if I have an approved draft of post-type X, do I publish it to channel Y, and is that channel a primary surface for that type or a secondary cross-post?" Cells with stub post-types (image/* and video/*) are forecasts; they tighten as those post-types are authored and as researcher's format scans confirm what actually converts on each platform.

**Why the placeholder marker.** A channel marked *placeholder* means the rules section below names it but we have not yet released artifacts to it. Activation requires a `channel-strategy-update` decision that supplies (a) account count, (b) purpose tags, (c) primary-user member, (d) initial strategy. Until then, draft variants for that channel are speculative.

---

## X / Twitter

**Active:** yes
**Priority:** P0 — primary distribution surface for OSS narrative.
**Accounts:** 1 primary brand account. (Persona-actor accounts: 0; persona accounts on X are a future option once `ai-ugc-personas.md` rules and disclosure plumbing are in place.)
**Primary lane/member:** OSS lane (`oss-advertiser`) for dev logs; subscription lane (`subscription-advertiser`) for launch threads.
**Format support:** dev-log threads, scenario-spotlight threads with attached video, oss-framework threads, occasional cross-posts of image content.
**Strategy notes:**
- Hook in the first tweet; the rest of the thread doesn't matter if the first tweet doesn't land.
- Builder-in-public voice from `STRATEGY.md` is non-negotiable; corporate-marketer voice triggers contrarian rejection.
- Algorithm rewards reply-engagement on the first tweet; hooks designed for reply ("hot take", "concrete number nobody else has", "asking for help") outperform pure-broadcast hooks.

**Rules:**
- Max 280 chars per tweet (hard platform limit).
- Thread length: 3-7 tweets typical; longer is fine when essay-shape requires it.
- Sanitize paths, emails, credentials per `x-dev-log` guardrails. Non-negotiable.
- Threads cite sources (commit hashes, run ids, issue links) rather than vague claims.
- Every metric in a thread carries an honesty flag (`pending-telemetry` is fine; unflagged is a violation).
- No auto-posting. Always goes through an approved `content-publish-proposal`.

**Revisit:** when X platform rules change (API, length limits, format), or when a `channel-strategy-update` decision proposes activating persona-actor accounts.

---

## Blog (self-hosted)

**Active:** yes
**Priority:** P0 — primary surface for long-form scenario-spotlights and oss-framework essays.
**Accounts:** 1 self-hosted blog (no platform accounts).
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for long-form SKU narratives; OSS lane (`oss-advertiser`) for architecture write-ups and OSS-framework essays.
**Format support:** long-form essay shape, embedded code, embedded screenshots, occasional embedded video.
**Strategy notes:**
- SEO-friendly headlines (researcher proposes via `seo-optimizer`); no SEO-spam patterns.
- Long-form is where dev-log series get their permanent home; X threads link back to blog for the full version.

**Rules:**
- 500-2000 words. Technical depth welcome.
- Include code snippets when they carry the story.
- End with a concrete invitation (try it, read more, contribute, book a demo).
- Every metric carries an honesty flag.
- SEO checks via `seo-optimizer` at polish time.
- No auto-publish.

**Revisit:** when we migrate blog hosting or change the CMS.

---

## Video (self-hosted YouTube)

**Active:** partial (manual production; `video-studio` scenario not yet shipped).
**Priority:** P1 — long-form demos and architecture walks live here; short-form video lives on TikTok/Reels/Shorts (separate channels below).
**Accounts:** 1 brand YouTube channel (placeholder until activated).
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for SKU demos; OSS lane (`oss-advertiser`) for architecture walkthroughs.
**Format support:** demo-recording (long-form), occasional comparison-reel and use-case-tutorial.

**Rules:**
- Production: currently manual; recurring workarounds are captured as typed marketing-craft observations or `capability-gap` decisions.
- Lengths: 30-60s for social clips, 2-5min for demos, 5-15min for architecture walks.
- Captions mandatory.
- Every metric in the description carries an honesty flag.
- No auto-publish.

**Revisit:** when `video-studio` scenario ships. At that point, expect this entry to tighten.

---

## LinkedIn

**Active:** placeholder.
**Priority:** P2 — small-team-lead audience surface; activated when the subscription lane has a draft sized for the platform.
**Accounts:** TBD on activation.
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for the small-team-lead audience, when activated.
**Format support:** TBD on activation; expected primaries are scenario-spotlight (text+asset) and oss-framework cross-posts.
**Rules:** not yet codified.

**Revisit:** when publisher releases first artifact to LinkedIn — at that point, the first `channel-update` decision populates the rules section with observed-successful patterns.

---

## TikTok

**Active:** placeholder.
**Priority:** P1 — primary surface for short-form video and AI-UGC formats targeting the lifestyle-bundle audience.
**Accounts:** TBD on activation. Expected pattern: 1 primary brand account + N persona-actor accounts (per `patterns/ai-ugc-personas.md`).
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for lifestyle bundle and persona-actor content; publisher for cross-platform variants.
**Format support:** narrative-talking-head, day-in-life-ugc, problem-agitate-solve, comparison-reel, slideshow-tips-then-plug, slideshow-voiceover. (See matrix above.)
**Strategy notes:**
- AI-UGC content requires native AI-generated label per TikTok 2026 rules; see `patterns/ai-ugc-personas.md` for disclosure protocol.
- TikTok algorithm rewards completion-rate and rewatch on short videos; first 1.5 seconds are the entire hook budget.
- Persona-actor accounts post in their own voice and niche, not in brand voice; cross-account cannibalization risk is real and managed via slate.json (per persona).
- No real-person impersonation; no professional-credential claims by personas; strict per `patterns/ai-ugc-personas.md`.

**Rules:** not yet codified beyond the AI-UGC discipline above.

**Revisit:** when first persona-actor account is activated (requires `channel-strategy-update` decision establishing accounts, slate, and disclosure protocol).

---

## Instagram (Feed / Reels / Stories)

**Active:** placeholder.
**Priority:** P1 — visual-first surface; lifestyle bundle is the primary target audience.
**Accounts:** TBD on activation. Expected pattern similar to TikTok: 1 primary brand + N persona-actor accounts.
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for lifestyle bundle; publisher for cross-posts.
**Format support per surface:**
- **Feed:** single-image-ad (P), infographic (P), slideshow-listicle (S), scenario-spotlight cross-posts (S).
- **Reels:** narrative-talking-head (P), day-in-life-ugc (P), problem-agitate-solve (P), comparison-reel (P), slideshow-voiceover (P).
- **Stories:** ephemeral cross-posts of any of the above plus behind-the-scenes content.
**Strategy notes:**
- Same AI-UGC disclosure rules as TikTok apply.
- Reels and TikTok share most short-form video formats; produce once, variant per platform aspect ratio (9:16 vertical for both, but caption/cover styling differs).

**Rules:** not yet codified.

**Revisit:** when first artifact ships to Instagram (likely paired with TikTok activation).

---

## YouTube Shorts

**Active:** placeholder.
**Priority:** P1 — short-form video surface tied to the same brand YouTube channel as long-form demos.
**Accounts:** uses the brand YouTube channel; persona-actor channels possible later.
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for lifestyle and dev-tool bundles.
**Format support:** narrative-talking-head, problem-agitate-solve, comparison-reel, slideshow-voiceover, occasional day-in-life-ugc and demo-recording cut-downs.
**Strategy notes:**
- Shorts feed into the brand channel's algorithmic surface; consistent posting helps long-form discovery.
- Same AI-UGC disclosure rules as TikTok apply (YouTube also requires AI-generated content disclosure as of 2026).

**Rules:** not yet codified.

**Revisit:** when YouTube channel activates with regular cadence.

---

## Threads

**Active:** placeholder.
**Priority:** P2 — X cross-post surface; minimal incremental cost once X drafts exist.
**Accounts:** 1 primary brand account (matched to X).
**Primary lane/member:** Publisher for cross-posts of existing X threads.
**Format support:** dev-log, scenario-spotlight, oss-framework, use-case-tutorial — all as text-only or text+image cross-posts of X originals.
**Strategy notes:**
- Threads is mostly an audience-overlap surface for X; treat as zero-cost cross-post until evidence emerges that it deserves separate drafting.

**Rules:** not yet codified.

**Revisit:** when researcher's format scans show Threads-specific formats outperforming X cross-posts.

---

## Bluesky

**Active:** placeholder.
**Priority:** P2 — X-alternative; relevant for the OSS-contributor audience that has migrated.
**Accounts:** 1 primary brand account.
**Primary lane/member:** Publisher for cross-posts; OSS lane (`oss-advertiser`) for occasional Bluesky-native dev logs.
**Format support:** same as X — dev-log, scenario-spotlight, oss-framework cross-posts.
**Strategy notes:**
- Smaller audience than X but heavier OSS-developer concentration; dev-log content fits naturally.
- Federation/feed dynamics differ from X; reply-engagement matters less.

**Rules:** not yet codified.

**Revisit:** when bluesky audience > 10% of X audience or when researcher flags a Bluesky-specific format outperforming.

---

## Reddit

**Active:** placeholder.
**Priority:** P1 — high-conversion surface for use-case-tutorial and oss-framework content in the right subreddit; also dangerous (off-topic posts get downvoted hard).
**Accounts:** 1 brand account (rare posting), N personal-operator accounts (organic participation, not brand-marketing).
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for subreddit-targeted scenario spotlights; OSS lane (`oss-advertiser`) for OSS-framework essays in r/MachineLearning, r/programming, r/selfhosted.
**Format support:** scenario-spotlight (subreddit-fit), oss-framework, use-case-tutorial. No image/video originals; embedded media OK.
**Strategy notes:**
- Subreddit fit is everything. Each post-channel decision specifies exact target subreddit and verifies its rules against draft.
- Reddit voice is anti-marketer; recommendation-framing applies heavily.
- Operator-account participation (replying genuinely in threads) is more valuable than brand-account posting.

**Rules:** not yet codified per subreddit; first `channel-strategy-update` should propose a subreddit allowlist.

**Revisit:** before first scheduled Reddit post.

---

## HackerNews

**Active:** placeholder.
**Priority:** P1 — Show HN is a primary surface for oss-framework and major scenario-spotlight launches.
**Accounts:** operator-personal account (HN convention; brand accounts get downvoted).
**Primary lane/member:** OSS lane (`oss-advertiser`) for Show HN posts; subscription lane (`subscription-advertiser`) for occasional scenario launches.
**Format support:** oss-framework (Show HN), scenario-spotlight (rare, only if launch-worthy), use-case-tutorial (rare, only if novel).
**Strategy notes:**
- Show HN convention: title is "Show HN: <Project> — <one-line>"; first comment is the operator giving context (the post body is typically just the link).
- Karma cliff is real; off-topic posts tank.
- Best windows: Tuesday-Thursday morning Pacific.

**Rules:** not yet codified.

**Revisit:** before first Show HN.

---

## ProductHunt

**Active:** placeholder.
**Priority:** P1 — primary launch surface for major scenario or bundle drops.
**Accounts:** brand account + maker accounts (operator + named contributors).
**Primary lane/member:** Subscription lane (`subscription-advertiser`) for launch-day scenario spotlights; brand-manager for campaign coordination.
**Format support:** scenario-spotlight (launch-day variant), with assets (single-image-ad on the launch page, demo-recording embedded).
**Strategy notes:**
- Launch day is the only day that matters; rank decay after 24h is steep.
- Pre-launch hunter recruitment matters more than the launch-day post itself.
- One launch per scenario or bundle; do not relaunch the same product on PH.

**Rules:** not yet codified.

**Revisit:** before first PH launch.

---

## Mastodon

**Active:** placeholder.
**Priority:** P3 — federated X-alternative; small audience, OSS-aligned.
**Accounts:** 1 primary brand account on a relevant instance.
**Primary lane/member:** Publisher for cross-posts of X originals.
**Format support:** dev-log, scenario-spotlight, oss-framework as zero-cost cross-posts.
**Strategy notes:**
- Treat as zero-cost cross-post until evidence emerges that it deserves dedicated drafting.

**Rules:** not yet codified.

**Revisit:** when researcher flags a Mastodon-specific format or instance worth targeting.

---

## Conversion-by-bundle metrics (placeholder)

This table will populate as the social-media-scheduler scenario returns engagement and click-through data per channel. Until then, `pending-telemetry` is the correct value for every cell.

| Channel | dev-tool bundle | lifestyle bundle | OSS-platform | business bundle |
|---|---|---|---|---|
| X / Twitter | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| Blog | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| Video (YouTube) | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| TikTok | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| Instagram | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| YouTube Shorts | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| Reddit | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| HackerNews | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |
| ProductHunt | pending-telemetry | pending-telemetry | pending-telemetry | pending-telemetry |

Decision context for populating: `channel-strategy-update` (researcher proposes after reading scheduler metrics). The researcher must apply the same `interpretation_flag` discipline used for `audience-scans.jsonl` — single-snapshot data is `light-interpretation`, ≥3 converging snapshots over comparable periods are required before promoting to a stable claim.

---

## Cross-channel rules

- **Variant integrity.** Every variant of an approved draft traces to the same proposal and same positioning claim. Diverging claims = bug.
- **Honesty flags preserved during polish.** Publisher never smooths away `pending-telemetry` or `estimate` flags.
- **Voice preserved during polish.** Publisher corrects typos and platform-rule violations; does not rewrite advertiser voice into house style.
- **Collision check before scheduling.** Publisher queries `publish-log.jsonl` + scheduled queue for same-day collisions on the same channel.
- **AI-UGC disclosure preserved across channels.** Persona-actor content carries the same disclosure on every cross-post; honesty-flag schema enforces.
- **Persona-actor account discipline.** Persona accounts post in persona voice on their own niche; brand-voice content does not appear on persona accounts and persona content does not appear on the brand account. See `patterns/ai-ugc-personas.md`.
