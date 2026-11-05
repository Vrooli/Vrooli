# Monetization

System Monitor is the second marketed deliverable in the business bundle,
sold as **Vega** (operator decision, 2026-09-17). Its commercial boundary is
simple: the product is the app plus the agent-facing skills it deploys with —
there is no metered resource of its own.

- Monitoring, storage triage, and investigations run locally; nothing in the
  scenario charges per use.
- AI-driven anomaly investigations use the buyer's configured model and
  provider (local Ollama works without an account). Any routed inference is
  owned by the scenario that routes it (`ai-gateway`), not by System Monitor.
- The agent skills that ship with deployment are part of the deliverable, not
  a separate SKU. They are why this app opens the **agentic deployment ramp**:
  the buyer's agents gain instruments (health checks, triage, storage sweeps)
  the moment the app is installed.

Positioning inside the bundle: Vega complements Aquila (`web-console`) for the
same buyer. Aquila is where you run your agents; Vega is how you see what that
work does to the machine — CPU, memory, GPU, storage, thermals — with
plain-language investigations and reports.

The durable evidence contract is:

1. `.vrooli/monetization.json` declares bundle membership
   (`bundle_key: business_suite`, `app_key: system-monitor`) with no meters
   and no entitlement requirement.
2. The landing-page-business-suite delivery catalog row (`download_seed.json`)
   and the public presentation (`/apps/vega`) are branded from this scenario's
   brand-manager output via `sync-bundle-catalog.mjs`; the catalog is the only
   place install artifacts are declared.
3. Capability claims on the public page stay at `preview`/`coming-soon` until
   a release owner records a qualified attestation; `available` claims are
   rejected by the presentation validator without one.

Go-to-market posture: reach developers and agent operators through the bundle
page (`/`) and Vega's own route (`/apps/vega`); the catalog source of truth
for lifecycle is Offer Desk (see
`docs/monetization/catalogs/skus/base/business.md`).
