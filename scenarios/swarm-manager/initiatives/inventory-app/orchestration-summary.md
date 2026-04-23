# Inventory App — Orchestration Summary

**Origin:** Morning Vision Walk, 2026-04-23. Elevated to a first-class initiative during Phase 7 when it became clear inventory is the substrate for consumer-product recommendations, not a feature of the routines app.

**Why this exists:** Vrooli has a clear pipeline for understanding the user's *digital* capabilities (scenarios installed, resources configured, agent configs). It has no ground truth for the user's *physical* capabilities — what tools, products, supplies, and assets they own. Without that ground truth, every physical-domain recommendation collapses to guessing or generic advice. With it, recommendations become completion of a known need.

The single most important architectural insight this scenario embodies:

> **Inventory is the substrate that makes natural product suggestion work. Without inventory data, product suggestion collapses to ads. With it, suggestions become completion of a known need.**

This principle is documented in the monetization revenue-line files (`consumer-products.md`, `affiliate-commerce.md`) as a hard gate — no scenario in the lifestyle bundle activates consumer-product offers or affiliate links until it has access to inventory-aware state.

## Vision framing (must be preserved in initiative docs and agent handoffs)

### Non-mandatory, grows over time

Users cannot and should not be required to inventory everything they own up front. The scenario is designed to accept partial data and become more useful as the inventory fills in. Short-term wins with even 10% coverage (a few tools, main cleaning supplies) justify the effort.

### Short-term uses

- **Routines gating.** The routines-app scenario reads inventory to surface which routines are available now vs. blocked on missing products.
- **Meal prep / grocery.** A nutrition / cooking scenario reads inventory to suggest meals from what's on hand and generate grocery lists for what's missing.
- **Cleaning / maintenance.** Routines reference required products; inventory tracks whether the user has them.
- **Product recommendation gating** (once revenue-line architecture exists). Never offer what the user already owns.

### Long-term uses

- Personalized emergency-kit / baseline-tools checklists (e.g., "every household should have X, Y, Z; you're missing W"); these are natural surfaces for consumer-product offers once revenue-line gating allows.
- Child-age-triggered safety lists (baby-proofing kits surfaced at the correct developmental moment).
- Warranty / purchase-date tracking (overlaps with a future home-management scenario).
- Robotic orchestration (distant). Humanoid robots performing physical tasks need ground truth about what's available in the home.

### Inventory ≠ asset catalog

The scope is practical: things relevant to tasks the user wants help with. Not "a list of every object in the home." Over-engineering toward completeness will kill adoption. Prioritize coverage of the *item categories actually referenced by other scenarios*.

## v0 scope

- React-vite scenario scaffold (UI + Go API + CLI, per Vrooli template standards)
- Data model supporting at least: consumables (cleaning products, groceries, parts), tools (owned once, tracked), assets (larger items with metadata — appliances, vehicles, electronics)
- CRUD via CLI + UI
- Semantic search
- Simple categorization / tagging
- Read API for other scenarios (initially the routines-app) to query "does the user have X?"
- Import surface: manual entry first, CSV import second. No OCR / photo-to-inventory in v0 (future).

v0 explicitly **does not** include: auto-detection via photos, auto-depletion tracking, integration with receipts, purchase-suggestion surfaces, consumer-product offers (those live in the routines-app / other scenarios, gated on the revenue-line architecture).

## v1+ directions (not committed)

- OCR-based inventory addition (take a photo of a shelf / pantry → structured entries)
- Auto-depletion tracking via routine completion (cleaning routine consumes 1/20 of Bar Keeper's Friend container)
- Warranty / purchase-date tracking
- Integration with a future receipt / purchase-tracking scenario
- Inventory-driven suggestion surfaces in partner scenarios (gated on revenue-line architecture)

## Compound value ties

- **Routines-app (separate initiative)** — primary consumer of inventory in v0.
- **Future meal-prep / nutrition scenario** — second consumer.
- **Future home-management scenario** — reuses asset tracking.
- **Lifestyle-bundle revenue-line gating** — inventory maturity unlocks consumer-product and affiliate monetization for the entire bundle.
- **Contact-book-plus (separate initiative)** — cross-reference for gift-purchase use cases (what have you given whom).

## Monetization implications (held separate)

Inventory is the **gate** for the lifestyle bundle's revenue-line activations, not a revenue-producing scenario itself. No monetization surfaces live inside this scenario. Its presence unlocks monetization in partner scenarios that legitimately recommend purchasable goods.

## Risks

- **Data-entry friction.** Biggest failure mode. Every barrier to adding an item reduces coverage. CLI-first entry, copy-paste-friendly, permissive on structure.
- **Over-modeling.** The temptation to build a "universal asset database" is strong. Resist. v0 models only what partner scenarios actually query.
- **Staleness.** Inventory that drifts out of sync with reality is worse than none. Needs a story (eventually) for refreshing / confirming / pruning.
