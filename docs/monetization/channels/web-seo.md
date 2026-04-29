# Channel: Web SEO + Landing Pages

- **Status:** `active`
- **Audience:** humans (with structured-data side benefit for AI-crawler agents)
- **Owner:** marketing-crew (positioning, content cadence) + landing-page-business-suite (LPBS) (deliverables)
- **Feeds:** [`subscription`](../revenue-lines/subscription.md) primarily; secondary feed into services revenue lines via "talk to us" / "book a call" CTAs.
- **Coupling:** Spans all tiers. The most universal channel — every other channel routes back to landing pages eventually.

## Hypothesis

Long after the agentic-commerce shift completes, the open web remains the substrate humans use to evaluate whether to engage with a product. Strong landing pages plus organic search presence are the entry point for humans who don't already know what Vrooli is. Without this channel, every other channel needs to do the explanation job too — which dilutes their audience-specific advantages.

The structured-data side benefit (AEO — agentic engine optimization, llms.txt, schema.org) is real but secondary. Optimize primarily for the human reader; structured-data discipline is a hygiene baseline that pays off when agent crawlers cite Vrooli, not the primary objective.

## Operational discipline

- **LPBS produces, marketing-crew positions.** LPBS owns the landing-page production pipeline (templates, CMS, publish flow). Marketing-crew owns the messaging, audience targeting, and content cadence. Neither owns both halves alone.
- **One bundle, one landing page, one CTA hierarchy.** Each bundle has a primary landing page; each scenario inside the bundle has a sub-page; CTAs route to a single primary action per page (subscribe / install / try) with at most one secondary action.
- **Structured data hygiene.** Every landing page ships with schema.org Product/SoftwareApplication markup, llms.txt at the domain root, and Open Graph + Twitter Card metadata. These are the AEO baseline — non-optional but not the focus.
- **No SEO-spam patterns.** Keyword stuffing, doorway pages, spun content, content-farm patterns. The `seo-optimizer` member proposes optimizations; `contrarian` rejects optimizations that drift toward spam.
- **Honesty flags on metric claims.** Every metric in landing-page copy ("X% faster", "Y customers using") carries the same honesty-flag discipline used elsewhere. `pending-telemetry` is fine; unflagged claims are violations.

## Anti-patterns

- **Paid search ads.** Explicitly off-table per the channel cross-cutting principles. The funnel for organic search and paid search look superficially similar but the brand-positioning consequence is different. Don't conflate them; don't propose paid search as a "test" channel.
- **Doorway pages, gateway pages, content farms, AI-spun copy at scale.** Permanent SERP damage if Google catches them; even before that, contrarian rejects.
- **Hidden text, link schemes, exact-match-domain manipulation.** Black-hat SEO. Same prohibition.
- **Marketer-voice copy** that drifts from the builder-in-public voice canon in [`docs/marketing/STRATEGY.md`](../../marketing/STRATEGY.md). Voice violations are caught by contrarian during publishing review.

## Telemetry

- Organic search traffic (visits / week, by query category)
- Branded vs. unbranded search ratio (a leading indicator of brand strength)
- Landing-page → subscription conversion rate (per bundle)
- Landing-page → trial conversion rate (when trial flow exists)
- Time-on-page and scroll-depth (proxy for content quality)
- AI-crawler traffic share (via Cloudflare logs or equivalent — separate metric, not a primary KPI)
- Backlink count and domain quality

## Cross-channel relationships

- **OSS discovery** — strong reinforcement. GitHub stars and READMEs route inbound traffic to landing pages. Landing-page references back to GitHub repos close the loop.
- **Community content** — strong reinforcement. Blog posts on the self-hosted blog feed both organic search and community-content's social-amplification surface. Cross-references between blog posts and landing pages should be bidirectional.
- **Skill registries** — orthogonal. Different audience (agents vs humans). Some indirect lift if registry traffic discovers landing pages through skill metadata, but don't measure or optimize against this overlap.
- **App stores** — separate funnel. Once Tier 1 ships, expect bidirectional drift: ASO and SEO compete for the same intent ("download Vrooli"). Disambiguate via landing pages that route to app stores OR self-host depending on user intent.

## Phase posture

This channel is `active` and remains so. Phase work focuses on:

- Per-bundle landing-page completeness (currently focused on business bundle)
- Structured-data hygiene baseline (llms.txt, schema.org) on all production pages
- Conversion-rate optimization once telemetry is live and stable
- Long-form blog cadence as the SEO content engine

No sunset condition planned; this channel persists through every phase of the company.

## Notes

- Per-platform publishing rules for the blog itself live in [`docs/marketing/CHANNELS.md`](../../marketing/CHANNELS.md) under "Blog (self-hosted)." This channel doc is the strategy lens; that doc is the operational lens.
- The AEO / llms.txt discipline overlaps with [`skill-registries`](skill-registries.md) at the technical level (both reward structured data) but not at the audience level. Don't merge the channels.
