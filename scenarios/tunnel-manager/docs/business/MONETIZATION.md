# Monetization — Tunnel Manager

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

Tunnel Manager is **foundational internal infrastructure**, not a
standalone revenue product. Direct monetization is **not applicable**.

It is an **interface enabler** (per
[`docs/concepts/ECOSYSTEM.md`](../../../../docs/concepts/ECOSYSTEM.md),
where `tunnel-manager` is listed as an interface enabler): it makes
remote access reliable and programmatic. Its commercial value is
**indirect** — it underpins other scenarios that *are* monetizable and
unlocks future remote-access tiers.

- Direct product: **not applicable** — foundational infra, not sold on its own.
- Internal capability: **yes** — the external-access control plane (exposure broker + self-healing tunnel manager).
- SKU/bundle candidate: indirect — enables remote-access features of monetizable scenarios; not a SKU itself.
- Revenue line: deferred / not-applicable (see enabler rationale below).

## Enabler Rationale (indirect value)

- Makes published Vrooli scenarios reliably reachable from the public
  internet without manual tunnel babysitting — a prerequisite for any
  scenario that ships as SaaS or an enterprise remote-access install.
- Guarantees core scenarios stay exposed (CORE tier) and lets others
  lease reachability on demand (LEASED tier) — the networking
  substrate the self-improvement loop and future multi-server tiers
  depend on.
- Programmatic, budget-aware exposure replaces the operator's manual
  Cloudflare dashboard step, lowering the operational cost of every
  scenario that needs remote access.

## Customer / Buyer

- Primary user: Vrooli operators, infrastructure agents, and other
  scenarios that need to be (or need another scenario to be) publicly
  reachable. See [`GO-TO-MARKET.md`](GO-TO-MARKET.md) (internal adoption).
- Buyer: none — there is no external buyer for the tunnel control plane
  itself. Buyers exist for the *scenarios it enables*.
- Pain it removes: manual, error-prone Cloudflare hostname management
  and tunnel downtime.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | not-applicable | Foundational infra; ships as core Vrooli, not as a sold product. |
| Bundle component | indirect | Implicitly part of any future SaaS/enterprise tier that offers remote access. |
| Add-on | not-applicable | Not an add-on to another SKU. |
| Service/consulting assist | deferred | Could support done-for-you remote-access setup; not a near-term motion. |

## Pricing Hypothesis

- Model: **not-applicable** — no direct price; value accrues to enabled scenarios and tiers.
- Comparable products: hosted tunnel/ingress services (Cloudflare Tunnel, ngrok) — but those are inputs, not what Vrooli would sell.
- Willingness-to-pay evidence: n/a (internal infra).
- Cost drivers: local runtime + SQLite (no external DB); Cloudflare tunnel usage is a host-level cost, not per-scenario.

## Validation Plan

- Demand signal: internal — measured by adoption (scenarios using the
  exposure broker instead of the manual dashboard workflow) rather than
  external sales.
- Channel: internal adoption — see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: core-tier exposure seeded and leasing in use by
  operators and other scenarios; app-monitor new-tab integration live.
- Revisit trigger: only if a future deployment tier (SaaS/enterprise
  remote access) packages remote access as a paid capability — then the
  *tier* is monetized, with Tunnel Manager as its enabler.

## Current Status

`not-applicable (direct) / active enabler (indirect)` — foundational
internal infrastructure. Direct monetization is deferred and not the
goal; the scenario's business value is realized through the monetizable
scenarios and tiers it makes possible.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
