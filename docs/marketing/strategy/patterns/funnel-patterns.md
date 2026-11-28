# Strategy: Funnel Patterns

**Status:** v0 (skeleton — populated as the producer captures conversion paths).

The conversion paths from a marketing post to a Vrooli scenario, tier, or sign-up. A funnel pattern names: where the post lands, what the next step is (link-in-bio, attached link, profile click), what surface the next step lands on (landing page, demo URL, sign-up page, scenario surface), and the activation event that closes the funnel.

This is **strategic canon** for how funnels are designed. Per-campaign funnel state lives in [`../CAMPAIGNS.md`](../CAMPAIGNS.md). Skill execution (when telemetry exists) lives in `funnel-builder` skill.

## Why this exists

Many otherwise-good marketing artifacts fail because the conversion path is broken: the post is great but the link goes to a generic landing page that doesn't address the audience the post was aimed at, or the link goes to a sign-up screen that demands more commitment than the post earned. Funnel patterns codify "for *this* post type with *this* audience and *this* CTA, the next surface should look *like this*."

## How entries get here

- The producer captures funnel observations in the `team:marketing-crew` Source Ledger scope (topic `marketing-craft-observation/*` for funnel-pattern observations from external content).
- The producer captures funnel observations from our own publish history once telemetry exists — what worked, what didn't.
- Operator approves promotion via `brand-guideline-update`; entries land here.

Until telemetry exists, entries are `aspirational` or `light-interpretation` — patterns we *expect* will work based on external research, not patterns we've *measured* working at Vrooli.

## Schema for each entry

```
- **Pattern name:** [short slug]
  - **Source post type(s):** [post-type slugs the funnel starts from]
  - **Source audience(s):** [persona keys]
  - **Source channel(s):** [channels the post lives on]
  - **Conversion-rung target:** click-through | trial | sign-up | upgrade
  - **Step 1: from post to next surface:** [profile click → bio link / attached link / inline link / Reddit comment]
  - **Step 2: bio-link or landing surface:** [what's there; how it speaks to this audience]
  - **Step 3: activation:** [what the audience does on the surface that closes the funnel]
  - **Tier alignment:** [which tier the activation grants; per `docs/monetization/strategy/TIERS.md`]
  - **Honesty:** measured | aspirational | light-interpretation | heavy-interpretation
  - **Source / supporting evidence:** [scan ids, content-desk publish-history records, external references]
  - **Captured:** YYYY-MM-DD
```

## Pattern slots (populate as entries arrive)

### Profile → bio-link → landing-page (TikTok / Reels / Shorts)

*Persona-actor account on short-video surface. Bio link rotates per active campaign; landing page speaks to the lifestyle-bundle audience.*

(No specific patterns documented yet. Aspirational shape — populate after first persona-actor activation.)

### Tweet → attached link → product page (X)

*Direct-link surface; the tweet's CTA points at a specific scenario page or demo URL. Click-through rate depends on hook-body alignment with the link's destination.*

(No specific patterns documented yet.)

### Show HN → operator first-comment → scenario repo (HackerNews)

*The HN convention: title is a one-line, body is the link, operator gives context in the first comment. Activation event: GitHub star, contributor sign-up, or self-host.*

(No specific patterns documented yet.)

### Blog post → inline CTA → trial sign-up

*Long-form content with a CTA mid-body and one at the close. The mid-body CTA hits readers who decided early; the close CTA hits readers who consumed the full post.*

(No specific patterns documented yet.)

### ProductHunt launch → upvote → website → trial sign-up

*Launch-day funnel. Compressed timeline (24h to peak rank, 7d to long-tail). Trial sign-up is the activation; conversion to paid is downstream.*

(No specific patterns documented yet.)

## Cross-cutting funnel principles

These are stable; they apply across all patterns once any are documented:

- **One ask per post.** If the post asks for click-through, sign-up, *and* save in one CTA, the conversion-rung is blurry and conversion suffers. Per scenario-spotlight's `conversion-rung-blur` failure mode.
- **Audience-CTA-tier alignment.** The CTA's surface must match the tier the audience can realistically reach. Sending a free-tier-eligible audience to a sign-up screen demanding payment information is friction that breaks the funnel.
- **Bio-link freshness.** When a persona-actor account's link-in-bio rotates per campaign, only one campaign's funnel is active per persona at a time. Multiple-campaign concurrency on a single persona dilutes attribution.
- **Honesty in funnel handoff.** The next surface must not promise more than the post promised. If the post says "here's how I plan my chores" and the next surface tries to upsell to an enterprise tier, the funnel reads as bait-and-switch.

## Anti-patterns (funnels the contrarian rejects)

- **Tier-misalignment.** Demo'd or implied feature is gated behind a tier the funnel doesn't grant. (Mode 1 specialization.)
- **Conversion-rung-blur.** Multiple rungs in one CTA. (See `scenario-spotlight.md`'s `conversion-rung blur`.)
- **Activation-without-fulfillment.** Funnel closes by collecting an email or a sign-up, but the user isn't given any actual value at that point — they bounce.
- **Phantom-funnel.** Pattern entry asserts a flow that hasn't been measured *or* observed in research; reads as wishful thinking.

## Cross-references

- [`../CAMPAIGNS.md`](../CAMPAIGNS.md) — per-campaign funnel state.
- [`../CHANNELS.md`](../CHANNELS.md) — per-channel funnel-relevant rules and the bundle-conversion table that funnels feed.
- [`../../../monetization/strategy/TIERS.md`](../../../monetization/strategy/TIERS.md), [`../../../monetization/catalogs/CATALOG.md`](../../../monetization/catalogs/CATALOG.md) — tier alignment authority.
- `funnel-builder` skill (in `prompt-manager` core skills) — skill consumes these patterns once telemetry is wired.
