# Hosted-Cloud-Tier (Tier 3) Foundation — Orchestration Summary

## Strategic Rationale

Vrooli's catalog (`docs/monetization/catalog/base/business.md`) currently positions web-console and git-control-tower as co-headliners — both deployable today, both with standalone appeal. The web-console pitch as written is "lets a solo operator manage their own infrastructure without learning cloud-provider UIs." That is true and the audience is real, but it's a narrow audience: people who already self-host servers. The much larger latent market is the audience who would value Vrooli but does *not* run their own infrastructure — and for whom the self-host onboarding friction is the blocker, not a feature.

Tier 3 (Hosted cloud) per `docs/monetization/PRICING.md` is the offering that opens that market: managed Vrooli instances on a VPS, connected via Cloudflare tunnel, accessible from any device including phone. This is the offer that makes web-console and GCT compelling as bundle headliners for non-self-hosters.

This initiative does not displace web-console-readiness or change the catalog's headliner status. Both pitches coexist:
- **Tier 1 / Tier 2 (self-host):** existing pitch, existing audience, existing path. Catalog stays accurate.
- **Tier 3 (hosted cloud):** new pitch, larger audience, this initiative provides the foundation.

Origin: 2026-04-25 vision walk. Operator surfaced the question of whether web-console-readiness should depend on a hosted-tier prerequisite. After investigation, the catalog framing was confirmed accurate for the existing audience — so option C (coexistence, no depends_on edge) was selected over option A (hard reorder) and option B (no Tier 3 work). The portfolio decision filed alongside this initiative captures that framing for director-swarm record.

## Cross-Item Decisions

- **Distinct from `remote-deployment-ops`.** That initiative is about internal deployment ergonomics (Vrooli the company shipping its own releases without needing a laptop). This initiative is about Vrooli the bundle being run as a managed service for paying customers. The names are similar by surface, the concerns are not by depth. Items do not migrate between the two.

- **No depends_on edge from web-console-readiness.** Per option C from the 2026-04-25 walk, web-console keeps its current headliner positioning for self-hosters. Tier 3 is a tier expansion, not a prerequisite. If telemetry later shows the Tier 1 audience is structurally too small to sustain the headliner pitch, that is a future portfolio-decision, not encoded today.

- **Pricing is deferred to monetization team.** The pricing-trough decision (`dec-1777061056395576280`) was deferred 2026-04-25 to 2026-05-09 pending Stripe→PRICING.md sync. Tier 3 pricing depends on that sync completing first. This initiative does not set or recommend a Tier 3 number; it provides the substrate so a number can be set.

- **LPBS subscription surface is reused, not rebuilt.** LPBS already has Stripe integration with plan/price store, payment settings handlers, account linking, etc. Tier 3 plugs into that — this initiative does not own subscription billing. The seam is "Stripe customer subscribed to Tier 3 plan → provision a hosted instance for that customer."

- **Minimum-viable surface is the load-bearing scope question.** Whether Tier 3 ships with web-console only, full business-bundle, or full Vrooli changes everything: provisioning cost, onboarding complexity, operational footprint, support burden. Workshop must answer this before vendor research locks in any concrete numbers.

- **Self-hosted instance migration is in scope, not separate.** A customer that subscribes to Tier 3 and later moves back to Tier 2 (self-host) should be able to export their full state. Conversely, a Tier 2 customer who moves up to Tier 3 should be able to import. This is a Tier 2↔Tier 3 fluidity question and should not be deferred to a separate initiative; it shapes the storage and identity decisions made here.

## Sequencing Notes

The intended phase order (workshop will refine — this is not prescriptive):

1. **Phase 1 — Research blockers.** VPS provider comparison, Cloudflare tunnel UX patterns, auth/identity model, billing integration seam with LPBS, minimum-viable Vrooli surface scoping. All research items, all priority 1. None of these can be skipped without Phase 2 inheriting unresolved decisions.

2. **Phase 2 — Architectural decisions.** Vendor pick, identity model pick, MVS pick, Cloudflare topology pick. One decision per Phase 1 research item. These are operator decisions, surfaced as portfolio decisions on the team that owns each domain.

3. **Phase 3 — Prototype.** Single-customer end-to-end: Stripe checkout → instance provisioned → tunnel established → operator connects → web-console loads. No production hardening yet. Goal is to expose unknown unknowns in the integration seams.

4. **Phase 4 — Production hardening.** SLA framework, support tooling, operational monitoring, multi-customer scaling, customer-data retention policy, offboarding flow.

5. **Phase 5 — Pricing + launch.** Pricing decision is unblocked once the Stripe→PRICING.md sync (separate work) completes and Tier 3 unit economics from Phase 4 monitoring are real numbers. Public launch follows.

## Open Workshop Questions

- Single-tenant per-customer-VPS vs multi-tenant on shared infrastructure? Operationally cheaper but security/data-isolation harder. Decision affects every downstream choice.
- Customer-owned domains supported from day 1, or vrooli.app/<slug> subdomain only? Customer-owned is friction the customer pays once; subdomain is friction Vrooli pays per-customer-cumulative.
- Self-hosting customer migration tooling — bidirectional from day 1, or one-direction (Tier 2 → Tier 3) first? The "customer wants their data back when they cancel" path is the harder one and probably the one that needs to ship at launch for trust.
- How does the hosted-instance update lifecycle work? Do customers run the latest Vrooli version automatically, or pinned-with-update-windows? Latest is operationally simpler; pinned is what some customers will demand.
- LPBS subscription tier upgrade UX — does Tier 1 → Tier 3 trigger automatic instance provisioning, or does it require a separate "claim your hosted instance" step?

## Pattern Note

This is the third "-foundation"-suffixed initiative shaped at this grain (`ai-image-generation-foundation`, `design-language-foundation`, `hosted-cloud-tier-foundation`). All three follow the pattern: research-heavy Phase 1, architectural-decisions Phase 2, prototype Phase 3, hardening Phase 4, launch Phase 5. After two more instances of this pattern, the shape probably promotes into a reusable foundation-initiative skill or template. Until then, the structure is a breadcrumb, not a contract.
