# Channel: App Stores

- **Status:** `candidate`
- **Audience:** humans
- **Owner:** TBD on activation. Likely a future "app-store-ops" function inside marketing-crew or a dedicated lifecycle role; in the interim, monetization and the scenario teams that own bundle apps share responsibility.
- **Activation trigger:** *"Activate when Tier 1 (bundle apps) ships its first paid bundle to at least one platform store (Apple App Store, Google Play, Microsoft Store)."*
- **Feeds:** [`subscription`](../revenue-lines/subscription.md) — directly, since Tier 1 deployments are sold via app-store subscriptions.
- **Coupling:** Tight. App stores ARE the deployment surface for Tier 1, not just a discovery channel — they're the only channel where the discovery surface and the delivery vehicle are the same thing.

## Hypothesis

For the bundle-app delivery shape (Tier 1), app stores are the primary discovery surface for non-developer humans. For audiences that don't read tech blogs, follow GitHub trending, or use AI agents — most of the lifestyle-bundle TAM, large parts of the business-bundle TAM — the app store is *the* discovery channel. Without strong ASO presence and good store-listing fundamentals, Tier 1 reaches only the audience already aware of Vrooli through other channels.

## Why this is `candidate` and not `active`

Tier 1 hasn't shipped. The activation trigger is concrete: first paid bundle live in at least one store. Until that day, this channel is purely speculative — store listings of unreleased products produce no signal.

## Operational discipline (for when active)

- **One bundle, one listing per store.** Each bundle gets a single primary listing on each store; sub-app listings only if a single app from the bundle has standalone appeal worth its own listing.
- **ASO is not SEO.** Different ranking signals (downloads, retention, ratings, keyword fields), different content shape (icon + screenshots + 80-char description), different review dynamics. Treat ASO as its own discipline; do not assume SEO playbooks transfer.
- **Review responsiveness.** Every review on every store gets a response within 72h. Negative reviews get a substantive response (not a templated "thanks for the feedback").
- **Honest metric claims in listings.** Same honesty-flag discipline as everywhere else. "Used by X" requires X to be a real number; aspirational copy carries `pending-telemetry`.
- **One launch per bundle per store.** Don't relaunch the same product to game ranking signals.

## Anti-patterns

- **Gaming reviews.** Paid reviews, fake reviews, review-swap rings, incentivized reviews. Permanent ban risk; permanent credibility damage. Same prohibition as the cross-cutting "astroturfed community presence" rule.
- **Keyword stuffing in listings.** Apple and Google penalize hard.
- **In-app upsell that violates platform rules.** Each store has its own rules about subscriptions, in-app purchases, and external billing. Violations get listings pulled.
- **Cross-store cannibalization via copycat listings.** One bundle, one listing per store; do not maintain "lite" + "pro" parallel listings unless they're genuinely different products.

## Telemetry (when active)

- Store-listing impressions × time
- Listing → install conversion (per store)
- Install → trial / first-paid-action conversion
- Rating × time (1-star alerts, mean drift)
- Review velocity and sentiment
- ASO keyword rank for primary terms
- Crash-free rate (table-stakes for store algorithms)
- Refund rate (high refund rates affect rankings)

## Cross-channel relationships

- **Web SEO** — partial substitution at the funnel level. Users searching "Vrooli iOS" might land on either a landing page or the App Store directly; the landing page should route to app stores when the intent is install-on-device. Disambiguate explicitly.
- **Community content** — reinforces. Launches on ProductHunt and HN drive app-store traffic spikes.
- **Skill registries** — orthogonal. Different audience, different deployment shape (Tier 1 vs Tier 2/3).
- **OSS discovery** — partial conflict. App-store users typically don't care that Vrooli is open source; OSS-discovery audience typically prefers self-hosting over installing from a store. The two channels reach different audiences and the messaging shouldn't blur.

## Phase posture

- **Pre-Tier-1 (current state):** `candidate`. No work happens on this channel until Tier 1 ships. Documentation-level placeholder only.
- **Tier-1 launch (activation):** moves to `active`. Initial cadence: one bundle, one store, learn the platform mechanics before expanding.
- **Multi-store expansion:** add platforms one at a time after the previous one has measurable signal. Don't launch on three stores simultaneously.
- **Sunset:** unlikely while Tier 1 exists. If Tier 1 itself sunsets, this channel sunsets with it.

## Notes

- The platform-revenue-share dynamics (15-30% to the store) affect Tier 1 unit economics directly. See [`TIERS.md`](../TIERS.md) and [`FINANCIAL_MODEL.md`](../FINANCIAL_MODEL.md) for cost-structure context. Listed here as a structural fact, not a channel-side concern.
- ToS regime matters more than for any other channel. Each store has different rules about subscriptions, parental controls, AI content, and in-app purchases that Vrooli's bundles must conform to. Pre-launch checklist required before activation.
