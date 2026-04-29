# Channel: Community Content

- **Status:** `candidate`
- **Audience:** humans (developers and early adopters primarily; lifestyle-bundle audiences once those scenarios mature)
- **Owner:** marketing-crew
- **Activation trigger:** *"Activate when marketing-crew has bandwidth to produce sustained cadence (not one-off launches), AND at least one Vrooli bundle has shippable headliner scenarios that benefit from launch-style content."*
- **Feeds:** [`subscription`](../revenue-lines/subscription.md) primarily; secondary feed into [`oss-discovery`](oss-discovery.md) via cross-amplification.
- **Coupling:** Spans all tiers. Long-form content (blog) couples loosely; short-form social and platform-launch content (HN, Reddit, ProductHunt) couples to whatever bundle is being launched at the moment.

## Hypothesis

Long-form content (blog), short-form social (X, LinkedIn, YouTube demos), and community presence (HN, Reddit, dev forums, Discord/Slack communities) collectively produce a brand-trust + audience-building signal that no other channel produces alone. This is what most companies call "marketing" — Vrooli treats it as one channel among several rather than the only channel because it's slow, expensive, and not always the highest-leverage path for a given goal.

For the developer audience the business bundle targets, builder-in-public dev-log content is structurally aligned with the audience's tastes and Vrooli's positioning. For the lifestyle-bundle audience, persona-actor short-form video is a different production discipline with different rules (see [`docs/marketing/strategies/ai-ugc-personas.md`](../../marketing/strategies/ai-ugc-personas.md)).

## Why this is `candidate` and not `active`

The channel is partially active in practice (some blog content exists, some social presence) but lacks the sustained cadence that distinguishes a real channel from sporadic activity. Single posts don't produce signal; consistent cadence over months does. The activation trigger is bandwidth + headliner readiness, not "post more" — sporadic effort produces no measurable lift.

## Operational discipline

- **Per-platform publishing rules live in marketing.** Detailed per-platform rules (X length limits, blog SEO discipline, TikTok format support, HN convention, ProductHunt launch mechanics) live in [`docs/marketing/CHANNELS.md`](../../marketing/CHANNELS.md). This file is the strategy lens; that file is the operational lens. Don't duplicate.
- **Builder-in-public voice.** [`docs/marketing/STRATEGY.md`](../../marketing/STRATEGY.md) defines the voice canon; corporate-marketer voice is rejected by `contrarian`. This applies on every platform.
- **Honesty flags travel.** `pending-telemetry`, `estimate`, `light-interpretation` flags carry through to social drafts. Publisher does not smooth them away during polish.
- **Variant integrity.** Every cross-platform variant traces back to the same approved proposal and same positioning claim.
- **AI-UGC discipline.** Persona-actor accounts on TikTok / Instagram Reels operate under [`docs/marketing/strategies/ai-ugc-personas.md`](../../marketing/strategies/ai-ugc-personas.md): native disclosure, no real-person impersonation, no fabricated credentials, no fake-customer-testimonial framings. Disclosure is non-negotiable.

## Anti-patterns

- **Cold spam / scraped-list outreach.** Explicitly off-table per the cross-cutting principles. This includes mass-email cold-outreach campaigns disguised as "newsletter" content.
- **AI-generated slop with no expert in the loop.** AI-assisted drafting is fine; AI-generated content with no human review and no domain expertise is rejected. The line is articulated in [`docs/marketing/strategies/ai-ugc-personas.md`](../../marketing/strategies/ai-ugc-personas.md): persona-actors with disclosure are allowed; fabricated credentials, real-person impersonation, and fake real-customer testimonials are banned.
- **Astroturfed community presence.** Fake reviews, paid HN comments, ghost-written user testimonials. Same prohibition as the cross-cutting list.
- **Paid social ads.** Off-table for the same reason as paid search — fights principle 1 and the brand positioning.
- **Launch-cycle thrash.** Launching on PH every month, relaunching on HN with renamed projects, crisis-marketing every two weeks. Cadence and consistency beat hype spikes.

## Telemetry

- Per-platform reach (impressions, views, plays — platform-appropriate)
- Engagement rate (replies, comments, shares, completion rate for video)
- Referrer traffic from each platform to landing pages and GitHub
- Platform-specific algorithmic surfacing (X reply-engagement, Reddit upvote ratio, YouTube watch-time)
- Sentiment in comments (qualitative; sampled, not exhaustive)
- Conversion-by-bundle attribution once `social-media-scheduler` returns engagement and click-through data (currently `pending-telemetry` per [`docs/marketing/CHANNELS.md`](../../marketing/CHANNELS.md))

## Cross-channel relationships

- **Web SEO** — strong reinforcement. Blog posts feed both channels simultaneously: organic search picks them up, social amplifies them. Cross-linking between blog and social drafts is structural, not optional.
- **OSS discovery** — strong reinforcement. HN Show HN posts, Reddit oss-framework essays, dev-log threads on X all route to GitHub and amplify the OSS-discovery channel.
- **App stores** — partial reinforcement. ProductHunt launches and HN posts drive app-store traffic spikes for Tier 1; otherwise mostly orthogonal.
- **Skill registries** — orthogonal. Different audience.
- **In-product expansion** — orthogonal.

## Phase posture

- **Pre-activation (current state):** `candidate`. Sporadic activity exists but no sustained cadence. Marketing-crew bandwidth is the binding constraint, not platform mechanics.
- **Activation:** when bandwidth + headliner readiness allows sustained cadence. Initial scope: one platform at a time (likely X + blog as the starting pair, since that's where the developer audience already is and the production cost is lowest).
- **Multi-platform expansion:** add platforms one at a time after the previous one has measurable signal. Same anti-thrash discipline as app stores.
- **Sunset:** unlikely; this channel persists indefinitely.

## Notes

- This channel deliberately does NOT include cold-outreach for services revenue lines (`lead-generation`, `app-development`). Outreach for services has its own discipline scoped to qualified prospects, captured inside the relevant revenue-line files.
- The marketing-crew operational ownership pattern is fully described in [`docs/marketing/`](../../marketing/). This channel doc is the high-altitude lens; do not duplicate per-post or per-platform rules here.
