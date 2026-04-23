# Revenue Line: Affiliate / Commerce (partner-produced)

- **Status:** `candidate`
- **Revisit trigger:** *"Revisit when at least one lifestyle-bundle scenario is deployable with inventory-aware product-recommendation surfaces AND we have Amazon Associates (or equivalent) account approval AND the recommendation-blindness post-processing layer exists in code."*
- **Revenue model:** commission on referred purchases via partner programs (Amazon Associates as the default first partner; others added case-by-case)
- **Productization target:** any scenario that legitimately recommends purchasable goods; concentrated in the lifestyle bundle
- **Legal surface:** FTC affiliate disclosure (mandatory, inline), platform-specific affiliate terms (e.g., Amazon's operating agreement is restrictive and changes frequently), tax treatment of commission income. Non-trivial; requires ongoing compliance attention, not one-time legal review.

## Hypothesis

Many lifestyle-bundle scenarios legitimately surface "you need product X" as an output of their work — cleaning routines require specific products, maintenance tasks require parts, inventory gaps imply purchases. When the user acts on that recommendation and buys via a partner like Amazon, a commission flows to Vrooli. Done right, this earns incremental revenue from recommendations the agent was going to make anyway — without changing the recommendation itself.

Done wrong — with recommendations drifting toward whatever earns the best commission — it destroys the authority layer that makes the lifestyle bundle valuable. This line is permissible *only* if the architectural constraint preventing that drift is actually enforced.

## Why this is a distinct revenue line (not a subset of consumer products)

- **Product ownership is different.** Consumer products: we produce / sell it; inventory, returns, support are ours. Affiliate: partner owns the full product lifecycle; we earn a cut for referring.
- **Infrastructure is different.** Consumer products need fulfillment / partner integration. Affiliate needs a link-rewrite post-processor and disclosure UI.
- **Legal surface is different.** Consumer products: general consumer-goods compliance. Affiliate: FTC disclosure rules are strict, platform terms are specific and changing.
- **Trust risk is different.** Consumer products can be quality-graded by us. Affiliate product quality is the partner's responsibility — a bad Amazon recommendation damages our trust even though we didn't produce it.

## Mandatory architectural constraints

These are not "best practices." They are hard rules. Violating them destroys the authority layer and the revenue line becomes untenable.

### 1. Recommendation-blindness — the agent does not know what earns commission

The agent producing a product recommendation must not know (a) which SKUs have affiliate relationships, (b) what commission rates are, (c) whether a specific recommendation earns us anything. The agent optimizes purely for what is actually best for the user.

### 2. Post-processing-only link rewrite

A separate, auditable post-processing layer checks whether the recommended product has an available affiliate link and rewrites the URL if so. This layer runs after the recommendation is complete. It can never reject, reorder, or filter recommendations based on commission availability.

### 3. Always-available non-affiliate option

If the user has opted out, or per-recommendation, a non-affiliate link must be reachable without penalty. If the affiliate price is not competitive, the non-affiliate cheaper option wins on ranking — recommendation-blindness means the post-processor may even expose "here's a cheaper non-affiliate option" when price data warrants.

### 4. Inline FTC disclosure

Every recommendation that results in an affiliate-linked destination must carry an inline disclosure — near the link, in the same visual unit, in plain language the user reads before clicking. Not a site-wide footnote.

### 5. Truthful opt-out

The user must be able to disable affiliate linking globally and per-scenario. The opt-out must describe what's actually happening ("this controls whether we earn a commission when you buy through us"), not a euphemism. When opted out, the scenario behaves identically except the post-processor is a no-op.

## Mandatory UX discipline

- **Disclosure is inline, not hidden.** A footer link to "affiliate disclosure" does not satisfy this rule. Each individual recommendation surface discloses at the point of the recommendation.
- **Price and merit are surfaced first.** Affiliate link is a URL rewrite, not a visual upgrade. The user should not be able to tell from appearance which links are affiliate — they tell from the inline disclosure.
- **No affiliate-aware copy.** Recommendations are written the same way whether or not the target has affiliate — never "check out this amazing deal on Amazon," which would suggest the agent is affiliate-aware.
- **Opt-out persistence.** Respected across sessions, respected across scenarios within the bundle, never silently re-enabled by updates.

## Bundle applicability

- **Lifestyle bundle: high fit.** Scenarios legitimately recommend purchasable goods (cleaning supplies, tools, parts, baby-proofing items, nutrition products). The inventory / routines substrate generates natural recommendation moments.
- **Business bundle: narrow fit.** Fewer natural recommendation moments for physical goods. Possible sub-cases: dev merchandise recommendations, hardware accessories, books. Most business-bundle work stays subscription.

## Gating: architecture must exist before activation

**Rule:** This line cannot activate until the recommendation-blindness post-processor exists in code and has been audited. The architectural constraint is the product. Without it, there is no responsible way to run this revenue line.

## Pattern examples (not commitments)

- **Cleaning routines → product purchase.** Routine specifies "use [Bar Keeper's Friend]." The agent recommended the product based on what works. The post-processor rewrites the link to affiliate if available.
- **Inventory gap filling.** Scenario determines the user lacks tools for a task. Agent recommends the right tool for the job. Post-processor adds affiliate where available.
- **Replacement parts for maintenance.** Routine specifies "replace your furnace filter size MERV 11." Agent sourced the recommendation from specs. Post-processor handles the link.

Every pattern above depends on the recommendation happening without affiliate awareness. Every pattern collapses if that constraint is violated.

## Instrumentation

Tracked separately from both subscription and consumer-product revenue. `financial-tracker` reports:

- Affiliate clicks per scenario per month
- Conversion rate (click → qualifying purchase)
- Commission revenue per scenario, per partner
- Opt-out rate (fraction of users who disable affiliate)
- Recommendation-integrity signal: ratio of affiliate-available recommendations to total recommendations, stable across time (sudden jump means agent may have become affiliate-aware)

## Activation discipline

On promotion to `active`, this line must have all five:

1. **Architecture audited** — post-processing layer is in code, a test exists that proves the recommendation agent cannot access affiliate metadata.
2. **Disclosure UI approved** — inline disclosure treatment reviewed by someone outside the implementing agent.
3. **Opt-out end-to-end tested** — per-scenario and global opt-out demonstrably zero-out commission flow.
4. **Partner agreement in force** — Amazon Associates (or chosen starter) approved, terms reviewed.
5. **Cross-line monitoring active** — `financial-tracker` reports the recommendation-integrity signal to the morning vision walk.

See [REVENUE_LINES.md](../REVENUE_LINES.md) for the broader revenue-line discipline.

## Notes

- Amazon Associates is typically the first partner because catalog breadth is matched by almost no other program. Onboarding is straightforward; terms are strict (never share commission, never use in email templates, etc.).
- As this line matures, additional partners (specialty retailers in lifestyle domains) may be added case-by-case — each new partner inherits the architectural constraints.
- Affiliate and [consumer products](consumer-products.md) are complementary: consumer products are our own SKUs where we control quality and margin; affiliate covers the long tail of recommendations where owning product doesn't make sense.
