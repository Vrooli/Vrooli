---
name: local-business-prospecting-toolkit
summary: Comprehensive Maps-driven B2B prospecting scenario — scrapes Google Maps for local businesses, AI-mines reviews for pain points, generates personalized cold outreach, optionally manages CRM with GPS-mapped sales zones. Reference architecture is the MapiLeads tool observed at walk #5.
source: social-media-alpha:walk-5-post-2-mapileads
sourced_at: 2026-04-28
status: staged
promoted_to: null
retired_reason: null
retired_at: null
---

# Local-Business Prospecting Toolkit

A scenario hypothesis surfaced during walk #5's social-media alpha extraction. The reference architecture is the MapiLeads tool described in walk #5's Post 2: a single integrated surface combining (a) Google Maps scraping with 30+ data fields per business, (b) AI review-mining for pain-point detection, (c) offer-cross-referencing for personalized cold outreach, (d) single-shot send (anti-bulk to land in main inbox), (e) CRM with GPS-mapped sales zones, route optimization, and voice-note transcription. Reportedly built by a single dev with Claude Code in ~2 weeks.

For Vrooli, this is **the productization target** of the `lead-generation` revenue line and the `property-services` add-on. It would be the underlying tool that we (a) operate as a service to local-service-business clients (dig-the-gold), (b) eventually sell as a subscription add-on (sell-the-shovel) when the business bundle has paying users to expand from. **Not capacity-available now** — gated on headliner-bundle-shipped per existing revenue-line discipline.

## Monetization framing

**Direct revenue paths:**

- **Lead-generation services line** (currently `candidate` in `monetization/revenue-lines/lead-generation.md`). Operator runs this scenario to deliver per-lead or per-close pricing to local service businesses. Three vertical playbooks already captured in `lead-generation.md`'s "Candidate vertical playbooks" section (website-building service, reputation management, cold-outreach automation) — this scenario is the underlying tool for all three.
- **`property-services` add-on** (currently `candidate` in `monetization/catalog/addons/property-services.md`). Once the business bundle has paying users and at least one vertical has been validated as a service-line pilot, the underlying scenario gets sold as a subscription add-on. The pitch becomes "here's a tool that does what we did for you; subscribe to run it yourself."

**Revisit trigger** (inherited from `lead-generation.md`): "Revisit when at least one property-services scenario is deployable as a thin tool AND one local-service prospect signs a pilot agreement."

**Bundle role:** depth (within the business bundle's property-services add-on; not a headliner). Standalone-app sale possible at a higher tier once productized.

**Dig-the-gold candidacy:** YES, deliberately. This scenario's design intent is operator-runs-as-service first, customer-runs-as-subscription second. The services-trap discipline in `monetization/REVENUE_LINES.md` applies: validation hypothesis + fixed-duration pilot + productization target + sunset-or-convert clause are required before activation.

**Legal surface:** highest among Vrooli scenarios. TCPA (US telemarketing — applies to SMS/calls, distinct rules for emails), CAN-SPAM (US email), GDPR (international), CASL (Canada), state-level B2B-vs-B2C distinctions on lead sales. Per `lead-generation.md`, explicit legal review is required before first paid engagement. The scenario itself must structurally enforce per-jurisdiction guardrails — not policy-as-prose.

## Marketing framing

*Tier 2 — not yet evaluated. Likely audience persona is "local-service-business operator" (HVAC / plumbing / roofing / pest-control / electrical / landscaping); secondary audience is "indie marketing-services consultant" who would resell our tool. Marketing-crew researcher should propose audience-update once the scenario is closer to ready.*

## Capability multipliers

**Upstream prerequisites (must exist before this scenario is buildable):**

- **BAS (Browser Automation Studio)** — for Maps scraping. Wrap-not-use governs: never inline Playwright; route through BAS scenario. ✅ Already shipped.
- **An AI inference resource** — local Ollama or hosted LLM access for review-mining + email-personalization. ✅ Already available.
- **A persistent storage layer** — Postgres + a schema for businesses, reviews, outreach-attempts, responses. ✅ Resource available.
- **An email-sending surface** — wrap-not-use applies; route through a wrapper scenario (mail-in-a-box-resource or similar). Status: partial; current `email-outreach-manager` scenario is archived and would need rebuilding under current scenario-templates.
- **An identity/permissioning surface** — when agents call this scenario, identity comes from agent-manager. Status: planned.

**Downstream consumers (scenarios that would benefit once this exists):**

- **`marketing-crew` team's outbound research** — could query this scenario for prospect-pool generation rather than relying on operator-surfaced manual research.
- **`bookmark-intelligence-hub` (BIH)** — when BIH ships, leads surfaced via prospecting could route through BIH's classification flow alongside operator's bookmark stream.
- **Future `social-media-scheduler` scenario** — if outreach has a social-media component (e.g., DM warming before email), the scheduler owns the publishing plumbing for that.

**Substrate vs. leaf:** mostly leaf — depends on substrates (BAS, storage, AI, email) without itself being a substrate for many downstream scenarios. That's fine; not every scenario needs to be substrate.

**Recursive-learning-loop story:** every prospecting run captures structured data (business profile, reviews, offer-fit-rationale, response). That data improves the AI's pain-point-detection and personalization quality over time. The longer it runs, the better the tool gets — classic compound-intelligence shape.

## Goal alignment

**Project goals served** (referencing `docs/strategy/`):

- **Headliner-pre-default-alive revenue lever** (per `monetization/REVENUE_LINES.md` Phase posture): services lines are expected to actively produce revenue in the window between core bundles shipping and subscriptions crossing default-alive. This scenario is one of the strongest near-term services-line candidates because of high capability reuse with the property-services add-on.
- **Customer acquisition for the business bundle** (per `monetization/STRATEGY.md`): service-clients are the most legible upgrade path to subscriptions ("we built you a tool; now subscribe to run it yourself"). This scenario operationalizes that path for a specific vertical.
- **Validation of property-services capabilities** (per the add-on's productization target): operating this scenario as a service surfaces real-world friction in geo-scanning, review-classification, outreach-deliverability — feedback that improves the productized add-on.

**Phase of deployment vision** (per CLAUDE.md): Phase 1 / Phase 2 — current local-stack deployment and emerging hosted/SaaS tier. Not Phase 3 (specialized hardware) territory.

**Honest counter-argument:** the sandboxing-trap-style coupling we documented in `EXECUTION-MODES.md` could apply here. This scenario integrates many substrates; building it will likely surface coupled-item friction inside the swarm-manager pipeline. May be a candidate for initiative-level operating mode (Plan B) if and when it's promoted to swarm-manager.

## Notes / open questions

- 2026-04-28: captured during walk #5 divergence #5 from social-media-alpha (Post 2 — MapiLeads). Hype-discounted: per `marketing/STRATEGY.md` Source-material discipline, the source's quantitative claims ("works in 221 countries", "30+ data fields", "single dev built it in 2 weeks") are upper-bound aspirational and require independent measurement before pricing or sizing decisions.
- 2026-04-28: gated on (a) headliner-bundle-shipped, (b) at least one property-services scenario deployable as a thin tool, (c) one local-service prospect signed for pilot. Per existing `lead-generation.md` revisit trigger.
- 2026-04-28: the archived `scenarios/swarm-manager/ideas/email-outreach-manager-archived` covers part of this scope (outreach + personalization) but not Maps-scraping or review-mining. When this is eventually promoted to swarm-manager, it should be a fresh scenario under current templates, not a revival of the archived one.

## Cross-references

- [`../../monetization/revenue-lines/lead-generation.md`](../../monetization/revenue-lines/lead-generation.md) — productization target's revenue line; reference architecture and candidate vertical playbooks live there.
- [`../../monetization/catalog/addons/property-services.md`](../../monetization/catalog/addons/property-services.md) — productization target's SKU.
- [`../../monetization/REVENUE_LINES.md`](../../monetization/REVENUE_LINES.md) — services-trap discipline, conversion-rate metric, 30%-time-budget cap.
- [`../../marketing/STRATEGY.md`](../../marketing/STRATEGY.md) — Source-material-discipline section; applies to all numbers borrowed from MapiLeads source.
- `scenarios/bas/` — substrate for Maps scraping (wrap-not-use).
- `scenarios/swarm-manager/ideas/email-outreach-manager-archived/` — partial-overlap archived scenario; not to be revived.
- `monetization` team knowledge under `monetization/opportunity/<slug>` — no paired opportunity-scout entry yet (this idea was captured operator-direct from walk #5 alpha; opportunity-scout may surface a paired entry on a future heartbeat).
