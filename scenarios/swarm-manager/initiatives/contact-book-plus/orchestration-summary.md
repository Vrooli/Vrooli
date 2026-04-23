# Contact Book Plus — Orchestration Summary

**Origin:** Morning Vision Walk, 2026-04-23. This initiative is a **revival and evolution** of the archived contact-book scenario (see `scenarios/swarm-manager/ideas/contact-book-archived/`), aligned to the lifestyle-bundle vision articulated in today's walk. The archived attempt's direction — social intelligence engine, relationship graph, cross-scenario memory — is close to what today's conversation converged on; reuse the archived PRD as a starting point, evolve rather than reinvent.

**Why this exists:** A contact book in Vrooli should not be "names, addresses, and phone numbers." It is a structured memory about the *people* in the operator's life — interests, relationship context, dietary preferences, personality notes, shared history, gift ideas, birthdays, and anything else that helps an agent act as if it actually knew the person. Every lifestyle-bundle scenario that touches interpersonal context (wedding planning, gift purchases, event recommendations, meal planning with guests) depends on this substrate.

## Vision framing (must be preserved in initiative docs and agent handoffs)

### Beyond contact info — a relationship substrate

The scope extends past CRUD over names and addresses:

- **Knowledge per person** — freeform notes, structured facts (dietary restrictions, interests, allergies, important dates), relationship strength signals.
- **Relationship graph** — how people in the book relate to each other (spouse, coworker, childhood friend, parent-of-X). Powers downstream features like seating arrangements, group-event planning, and gift-for-group suggestions.
- **Cross-scenario memory** — facts mined from user interactions with other scenarios become available here (nutrition preferences surfaced in meal-planning, communication style from email drafting, etc.).
- **Consent and privacy by design** — time-bounded facts, per-relationship visibility rules, user control over what is stored and for how long. This is not optional; the archived PRD already models this correctly and should be preserved.

### What this substrate unlocks (patterns, not committed scenarios)

- Gift-purchase surface — legitimately a monetizable moment (user explicitly wants to buy something for a specific person); the only user-initiated purchase-intent surface in the bundle.
- Birthday / anniversary reminders with gift suggestions tuned to the person's interests.
- Event recommendations — "X is in town next week, they're into Y, here are things you could do together."
- Wedding planning — seating arrangements, dietary matrices for the food plan, invitation list with relationship-weight priorities.
- Email / message drafting — auto-personalization based on relationship context.
- Relationship-maintenance nudges — "haven't talked to X in 6 months; they mentioned Y was on their mind last time."

These are capabilities that become possible *because* contact-book-plus exists. They do not belong in this scenario; they belong in scenarios that consume this one's API.

### Agent use, not just UI use

Like inventory, the primary consumers are other scenarios and agents, not the end user browsing the UI. Design the API around "does the user know someone who X?" / "what do we know about person Y?" queries, not around list/detail views.

## Relationship to the archived scenario

The archived contact-book PRD (`scenarios/swarm-manager/ideas/contact-book-archived/PRD.md`) is the starting point. Key reuse candidates:

- Data model (rich contacts, time-bounded facts, consent management) — already aligned.
- PostgreSQL schema — likely reusable.
- The "Intelligence Amplification" framing — carries directly.
- Recursive-value scenarios enumerated in the PRD (wedding planner, personal digital twin, email assistant, birthday reminders, social orchestration) — match today's walk's direction.

What needs re-evaluation:

- Whether any actual code from the archived attempt is reusable, or whether to restart from the PRD and the new react-vite template standard.
- Whether the archived status markers (some P0s showing ✅ 2025-09-24) correspond to code that still exists and builds, or are stale.
- Integration points with scenarios that didn't exist when the original attempt ran (inventory, routines-app, future calendar / phone-agent).

The first research sub-item on this initiative is an audit of what's salvageable.

## v0 scope (to be refined after archived-audit research)

Tentative:

- React-vite scenario scaffold (UI + Go API + CLI)
- Core CRUD on contacts with rich metadata (name, handles, relationship-type, freeform notes)
- Structured facts with optional time-bounds and visibility
- Basic relationship graph edges (person → person with typed relationship)
- Read API for partner scenarios
- Semantic search over contact notes and facts
- Import: manual entry + vCard / CSV

Explicitly not in v0: auto-mining from emails / messages, consent negotiation flows, relationship-strength scoring, birthday-reminder automation, gift-suggestion surfacing.

## Compound value ties

- **Inventory-app** — cross-reference for gift history ("we gave X this last year, don't repeat").
- **Routines-app** — relationship-aware routine customization (different cleaning cadence when hosting guests).
- **Future calendar scenario** — birthday / anniversary surfacing.
- **Future meal-prep scenario** — dietary matrices for events.
- **Future wedding-planning scenario** — seating, invitations, dietary planning (highest compound-value payoff).
- **Lifestyle-bundle revenue-line gating** — gift-purchase surface is the bundle's cleanest consumer-product / affiliate monetization moment; unlocked by this substrate.

## Monetization implications (held separate)

As with inventory, this scenario is a **substrate**, not a revenue-producer. Gift-purchase and event-recommendation surfaces that monetize live in downstream scenarios, gated on the revenue-line architecture.

## Risks

- **Privacy / consent failures** — this is the most sensitive data in any lifestyle scenario. Getting consent, visibility, and retention wrong is worse than not building it. The archived PRD's discipline here should be preserved and audited, not rewritten.
- **Scope creep toward "CRM."** Vrooli is not a CRM. The personal / relational axis is explicitly the focus; business contacts may overlap with a business-bundle scenario but that's future work.
- **Staleness.** Notes and facts drift. Eventually needs a refresh / re-confirm loop.
- **Auto-mining trap.** Automatic extraction from emails / messages is tempting but high-risk without strict user control; defer to v2+ with explicit consent workflows.
