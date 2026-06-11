# Monetization — Ecosystem Manager

This document records how Ecosystem Manager creates value for Vrooli.
It is kept honest: this scenario is an internal capability, not a
directly sold product today, and this page says so plainly rather than
inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is Ecosystem Manager a direct product, an internal capability, a SKU
  component, or a service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists, if any?
- What signal would justify surfacing it as a paid offering?

## Role In Vrooli

- **Direct product:** not applicable today.
- **Internal capability:** **yes — this is the role.** Ecosystem
  Manager is the control plane that generates and improves scenarios
  and resources via autonomous auto-steer loops. It is the engine that
  produces *every other* monetizable scenario in the portfolio.
- **SKU/bundle candidate:** deferred — only as a higher-tier
  "autonomous software factory" control plane (see Packaging).
- **Revenue line:** indirect. Value is realized through the scenarios
  it produces, not through its own sales.

The honest framing: this is a **force multiplier on the whole
portfolio**, not a line item. The smarter its controller (the
closed-loop auto-steer model in
[`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md)), the
more leverage it applies to every target it touches — so investment
here compounds across all downstream products.

## Customer / Buyer

- **Primary user (internal):** Vrooli itself and the agents/operators
  who run scenario and resource generation/improvement.
- **Potential future buyer (external):** an enterprise running its own
  Vrooli installation in a higher deployment tier, who wants an
  in-house "autonomous software factory" — a control plane that
  generates and continuously improves their internal services.
- **Pain it addresses:** building and maintaining many internal
  services is slow and unbounded in labor; a closed-loop controller
  that drives targets toward measurable goals collapses that cost.
- **Existing alternatives:** hand-built internal platform/eng teams and
  one-off scaffolding tools — none of which run a measured, autonomous
  improvement loop.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | not applicable | It is infrastructure for the portfolio, not a standalone purchase. |
| Bundle component | deferred | Could ship inside an enterprise/self-host tier as the "factory" control plane. |
| Add-on | deferred | Only as a control-plane add-on to a higher Vrooli deployment tier. |
| Service/consulting assist | possible | Accelerates done-for-you delivery of bespoke internal scenarios. |

## Pricing Hypothesis

- **Model:** not applicable today (internal capability). Any future
  pricing would attach to a deployment tier, not to this scenario
  alone.
- **Comparable products:** internal developer platforms / "software
  factory" tooling — none captured as direct comps yet.
- **Willingness-to-pay evidence:** none captured.
- **Cost drivers:** agent runs dispatched through `agent-manager`
  (model/gateway usage), local storage, and operator time.

## Validation Plan

- **Demand signal needed (indirect):** measured uplift — does steering
  a target with a profile reliably move it toward its goal faster /
  more completely than an unsteered run? That throughput gain is the
  real product.
- **Demand signal needed (external):** an enterprise self-host customer
  asking for an in-house generation/improvement loop.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Revisit trigger:** a deployment tier needs a customer-facing
  "autonomous software factory," **or** internal metrics show the
  controller materially raising portfolio output.

## Current Status

**Internal capability / not directly monetized.** Ecosystem Manager is
the production engine for the rest of the portfolio; its value is
realized through the scenarios it generates and improves, not through
its own sales. Future direct monetization is plausible only as the
control-plane component of a higher Vrooli deployment tier and is
deferred until that tier and a customer exist.

## Cross-References

- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — the closed-loop controller; the smarter controller is what raises this scenario's leverage
- [`../START-HERE.md`](../START-HERE.md) — orientation
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — positioning and channels
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
