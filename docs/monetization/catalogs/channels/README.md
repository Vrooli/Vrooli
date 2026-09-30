# Channels

The **ways users discover Vrooli** — distinct from bundles (what we sell), tiers (how it's delivered), and revenue lines (how money flows). Each channel has its own audience, owner, telemetry, and discipline, and lives in its own file in this folder.

> Offer Desk is authoritative for the current channel records, lifecycle,
> ownership, and feed relationships. This document keeps the channel-axis
> definitions and cross-cutting discipline; it does not maintain a live index.

## Disambiguation: this is not docs/marketing/strategy/CHANNELS.md

[`docs/marketing/strategy/CHANNELS.md`](../../../marketing/strategy/CHANNELS.md) is **per-platform publishing rules** for the marketing-crew — what to post on X, blog, YouTube, TikTok, etc., and how. It is an operational doc owned by marketing.

This doc is **strategy-level discovery channels** — the full set of paths users take to find Vrooli (web SEO, app stores, skill registries, OSS discovery, in-product expansion, community content). Marketing's owned channels are a subset; the rest (app stores, skill registries) are not marketing-owned at all.

When working on per-platform publishing rules → marketing/CHANNELS.md.
When working on strategy-level channel discipline, telemetry, and activation triggers → this doc.

## Why channels are a first-class axis

A monetization model with three axes (bundles, tiers, revenue lines) hides where users come from. Two users on the same tier paying through the same revenue line can have arrived through wildly different channels with wildly different unit economics, audience profiles, and operational disciplines. Treating channels as undifferentiated "marketing" loses that resolution.

Channels and tiers couple but are different:

- **Tier** = deployment shape (mobile, self-hosted, cloud, hardware)
- **Channel** = discovery path (web, app stores, skill registries, in-product, etc.)

App stores couple tightly to Tier 1 (they ARE the deployment surface). Skill registries couple loosely to Tier 2/3 (where standalone install matters). Most other channels span all tiers.

## Channel status lifecycle

| Status | Meaning |
|---|---|
| `candidate` | Documented with hypothesis and activation trigger; not actively producing measurable signal. |
| `active` | Currently producing signal; tracked in telemetry. |
| `sunset` | Winding down — either consolidated into another channel or being abandoned. |
| `retired` | Wound down. Kept in the folder for history and future lessons. |

## Index

The files in this folder are the durable strategy and policy lenses for the
channel records. Offer Desk provides the current index and read-time state;
these files preserve the audience, discipline, and activation semantics.

New channels enter via `channel-activation` decisions when their activation
trigger fires. Retired channels stay in the folder as historical judgment.

## Cross-cutting principles

These apply across multiple channels and live here rather than being repeated in each file.

### 1. Earn curation, don't buy placement

On every channel where this is technically possible (web-seo, app-stores, skill-registries, community-content), the structural choice is organic + curated. Paid placement is a smell. On agent-audience channels it is actively negative — agents trained on the post-2025 web discount "Sponsored" / "Promoted" labels and prefer "Editor's Pick"-style endorsements.

### 2. Audience-aware optimization

Humans need narrative, visuals, and trust signals; agents need structured metadata, signatures, and scanner clearance. The same content optimized for both usually lands flat for both. Channels with mixed audiences (web-seo, in-product-expansion) need explicit dual-track production — a human-narrative track and a machine-structured track that share factual ground but not form.

### 3. Recommendation-blindness extends to channels that touch lifestyle

Any channel where Vrooli surfaces affiliate or own-product recommendations carries the same architectural rule as the [`consumer-products`](../revenue-lines/consumer-products.md) and [`affiliate-commerce`](../revenue-lines/affiliate-commerce.md) revenue lines: the agent producing the recommendation must not know what we sell or earn commission on. Currently this binds in-product-expansion when it operates inside a lifestyle-bundle context. See the individual revenue-line files for the full constraint set.

### 4. Channels and tiers couple but aren't the same axis

Document the coupling without conflating the axes. App-stores ↔ Tier 1; skill-registries ↔ Tier 2/3 (where standalone install is meaningful). Most other channels span all tiers.

### 5. Honesty flags travel with metrics

Every channel-attributed metric in a public claim carries the same honesty-flag discipline used elsewhere in the org: `pending-telemetry` is fine; unflagged speculation is a violation. This applies equally to "we got 10K stars from oss-discovery" and "skill installs converted at X%."

## What's NOT a channel

Mirror the [`revenue-lines`](../revenue-lines/README.md) "what's NOT a revenue line" section. These are explicitly off-table; proposals to enable them require an exception decision with strong justification, and `contrarian` rejects by default.

- **Paid search / social ads.** Fights principle 1 (earn curation, don't buy placement) and the brand positioning (local-first, user-controlled, OSS-aligned). Specific exemption process required if ever proposed.
- **Cold spam / scraped-list outreach.** TCPA/CAN-SPAM exposure aside, brand corrosion is permanent. Note that this also excludes cold-email-style outreach in support of services revenue lines — those have their own outreach discipline scoped to qualified prospects, not bulk-list contact.
- **Dark-pattern referral mechanisms.** Refer-a-friend is fine when value-aligned and disclosed; gamified referral with intrusive prompts, fake-urgency mechanics, or hidden compensation is not.
- **Astroturfed community presence.** Fake reviews, paid HN comments, ghost-written "user testimonials," persona-actor accounts impersonating real users with fabricated credentials — all in the same prohibition zone as the [`docs/marketing/strategy/patterns/ai-ugc-personas.md`](../../../marketing/strategy/patterns/ai-ugc-personas.md) AI-UGC stance.
- **Starbuying / fake stars.** Specifically called out for `oss-discovery` because the channel mechanics make the temptation real. Permanent credibility damage; never an acceptable shortcut.

## Active channel instrumentation (for when one activates)

Each active channel reports in the ledger:

- Inbound traffic (visits, installs, stars, registry views — channel-appropriate unit)
- Conversion rate to next funnel step (signup, install, trial, subscription)
- Trust signals where applicable (scanner pass rate, registry curation tier, review scores)
- Anti-pattern flags raised in the period (e.g., a post that drifted into paid-placement territory)
- Cross-channel reinforcement (did a web-seo session start from a community-content referrer? track the chain)

Money Ledger rolls admitted per-channel observations into the financial
position when those observations exist; missing telemetry remains absent.

## How channel files relate to revenue-line files

Most channels feed `subscription`. Some channels feed services revenue lines (e.g., direct outreach for `lead-generation` and `app-development`, currently captured inside those revenue-line files rather than as its own channel doc). When a channel feeds multiple revenue lines, the channel file says so explicitly and the revenue-line files cite the channel.

The design rule: if a channel produces measurable lift to one specific revenue line and not others, it stays in this folder with a clear "feeds X" note. If a channel and revenue line are so tightly coupled they cannot be analyzed separately, document them in the revenue-line file and skip the channel entry.
