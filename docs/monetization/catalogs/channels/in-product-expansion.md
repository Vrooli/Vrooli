# Channel: In-Product Expansion

> Offer Desk is authoritative for this channel's current status, owner, and
> feed relationships. This document keeps the structural hypothesis and
> operating judgment rather than a live channel snapshot.

- **Audience:** both — humans encounter suggestions in UI, agents handle them structurally
- **Owner:** structural — emerges from template-manager design rather than from a dedicated marketing function. Each scenario team is the operational owner of the in-product moments where their scenario suggests other bundle apps.
- **Feeds:** [`subscription`](../revenue-lines/subscription.md) — drives cross-app activation, which is the strongest retention signal Vrooli has (per [STRATEGY.md §5: breadth of adoption = retention](../STRATEGY.md)).
- **Coupling:** Spans all tiers. The mechanics differ slightly per tier (in-app suggestion in Tier 1; runtime-level cross-scenario invocation in Tier 2/3) but the channel is the same.

## Hypothesis

Vrooli has a structurally different surface from most SaaS: agents already running in a user's workflow can organically suggest other bundle apps when relevant work appears. This makes in-bundle expansion (the drive toward breadth-of-adoption, the strongest retention signal in multi-product SaaS) a native capability of the platform rather than a marketing function.

This is the operational home for [STRATEGY.md §2 ("agents are the expansion engine")](../STRATEGY.md). The principle is structural; documenting it as a channel gives it telemetry, discipline, and a place to land instead of floating in strategy without a doc.

## Activation criteria

The channel is meaningful when scenarios can invoke relevant capabilities and
the suggestion remains reversible, dismissible, and recommendation-blind.
Instrumentation maturity is a separate concern from the lifecycle record.

## Operational discipline

- **Default to agent-driven expansion over marketing-driven.** Per [STRATEGY.md §2](../STRATEGY.md), in-bundle expansion is a structural property of scenarios. Only fall back to marketing-driven (lifecycle email, in-app notification banners) when an agent surface cannot reach the relevant moment.
- **Suggestions trigger on relevance, not on cadence.** An agent suggesting another bundle app should do so because the user's current task evidently benefits from it, not because a marketing rule says "suggest X every 7 days."
- **Suggestions are reversible and dismissible.** Users can decline; the agent learns. Suggestions are not modal blockers.
- **Recommendation-blindness applies in lifestyle-bundle contexts.** When in-product expansion operates inside a lifestyle-bundle scenario, the agent producing the suggestion must not know which suggestions earn affiliate commission or which are own-products. Same rule as `consumer-products` and `affiliate-commerce`. See those revenue-line files for the full constraint set.
- **Cross-bundle suggestions are higher-friction.** Within a bundle, suggestions are low-friction. Cross-bundle (e.g., business-bundle scenario suggesting a lifestyle-bundle app) requires an explicit user-context cue and stronger evidence of relevance.

## Anti-patterns

- **Nag-style suggestion.** Repeated suggestions for the same app, dismissals not respected, suggestions inserted into unrelated work. Kills trust; corrupts every other suggestion the agent makes.
- **Modal blockers.** A suggestion that interrupts ongoing work is a marketing tactic, not an agent suggestion. Agent suggestions are inline, optional, and dismissible.
- **Recommendation-blindness violations.** An agent that knows which suggestion earns commission, in any context where Vrooli surfaces affiliate or own-product recommendations, corrupts the entire authority layer of the lifestyle-bundle. Architectural separation is required, not a policy.
- **Marketing-disguised-as-agent.** Hard-coded suggestion rules ("every 5th task, suggest the lifestyle bundle") dressed up as agent reasoning. Either it's an agent suggestion based on actual relevance or it's a marketing surface — don't blur.
- **Suggestion patterns that bypass user-control settings.** If a user has dismissed all suggestions for app X, suggestions for X stay dismissed. No "well, this time it's really relevant" overrides.

## Telemetry

- Cross-app activation rate (users running ≥2 bundle apps / users running ≥1)
- Suggestion → adoption conversion (how often does a suggestion lead to the suggested app being used)
- Suggestion frequency per user-session (proxy for nag-risk)
- Dismissal rate and dismissal-respected rate (the second is a quality signal)
- Time-to-second-app (after first scenario adoption, how long until the user runs a second one)
- Cross-bundle vs. in-bundle suggestion ratio
- Recommendation-blindness audit results (separate compliance metric, not a conversion KPI)

## Cross-channel relationships

- **Web SEO, app-stores, oss-discovery, community-content, skill-registries** — all orthogonal. Those are *external* discovery channels (user → product); this is *internal* expansion (product → product within an account). Different funnel, different mechanics, different telemetry.
- **Substitution effect with paid lifecycle marketing.** This channel is what most SaaS companies pay heavily for via email drips, in-app nudges, and webinar funnels. Vrooli's structural advantage is that the agent surface reaches users where the work happens, without a marketing budget. When this channel is healthy, paid lifecycle marketing produces less marginal lift; when it's not, paid lifecycle becomes a fallback (see operational discipline above).

## Lifecycle interpretation

Offer Desk records the channel lifecycle. Maturity requires cross-app
activation, suggestion conversion, dismissal-respected rate, and routine
recommendation-blindness audits; it is not inferred from the existence of a
single invocation path.

## Notes

- The lifestyle-bundle implementation of this channel needs particular care because it's where the recommendation-blindness rule binds hardest. Lifestyle-bundle template-manager design must build the "agent can't see commission structure" separation into code and data flow, not policy. See [`consumer-products`](../revenue-lines/consumer-products.md) and [`affiliate-commerce`](../revenue-lines/affiliate-commerce.md) for the full constraint architecture.
- This channel's strength is also its constraint: it cannot reach users who haven't yet adopted a single bundle app. Acquisition through other channels remains necessary; in-product-expansion is the *retention and expansion* engine, not the acquisition engine.
- Per [STRATEGY.md §6 (activation is the leading indicator of retention)](../STRATEGY.md), the work that makes first-app activation strong is what enables this channel to function at all. Activation work and in-product-expansion work reinforce each other.
