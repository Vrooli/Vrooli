# Monetization — Device Control

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

An earlier draft of this document called Device Control "infrastructure, not
a product surface" and deferred the question. That was wrong, and the market
evidence below is why.

- **Direct product candidate:** yes, on the strength of the self-hosted
  control plane and the promotion loop described below.
- **Internal capability:** yes, and load bearing. It is what lets
  `scenario-to-ios` and `scenario-to-android` produce physical-device
  evidence, and those ramps are already on the monetization path.
- **Platform multiplier:** every future scenario gets device control for
  free. That compounding is worth more internally than the standalone SKU is
  worth externally, at least initially.

## Customer / Buyer

Teams that already own a shelf of test devices and cannot or will not send
their app binaries to a metered device cloud — regulated industries,
security-sensitive products, and apps under embargo before launch. The buyer
is whoever owns mobile release confidence; the pain is that the credible
self-hosted option is dead and the cloud option is priced per parallel.

## The comparable that matters

| Offering | Price | Shape |
|---|---|---|
| BrowserStack Automate | from $129/mo per parallel; ~$50–75k/yr at 100 parallel | Cloud device fleet |
| Sauce Labs | ~$80–120k/yr at comparable capacity | Cloud device fleet |
| AWS Device Farm | $250/device/mo unlimited, or $0.17/min metered | Cloud device fleet |
| **DeviceLab** | **$99/device/mo — using devices you already own** | **Control plane only** |
| OpenSTF / DeviceFarmer | free | **Abandoned since 2020; reported broken on Android 14/15** |

The DeviceLab line is the important one. It is almost exactly our shape —
software that drives hardware the customer already has — and it establishes
that **people pay for the control plane, not the device fleet.** The
OpenSTF line is the other half: the credible open self-hosted option is
dead, so teams wanting this today are choosing between an expensive metered
cloud and an unmaintained project.

## What we would actually be selling

Not a device cloud. Competing with a 30,000-device fleet on device breadth
is unwinnable and we should not pretend otherwise. The wedge is narrower and
more defensible:

1. **A self-hosted control plane for devices you already own.** No
   per-minute meter, no app binary leaving the owner's infrastructure. The
   buyers are teams for whom the cloud is either prohibited (regulated,
   security-sensitive, apps under embargo) or simply too expensive at scale.
2. **The promotion loop — "infer once, replay free."** Most AI-driven test
   tools re-infer on every run, which is what makes AI QA economically
   awkward: costs scale with executions rather than with authoring. Our
   agent run records every action as a flow step and exports a *deterministic*
   flow with the AI steps resolved away. The expensive inference happens once,
   at authoring time; every replay afterwards is free and repeatable. That
   directly attacks the cost structure the category currently has.
3. **Goal-level authoring.** "Log in, buy the thing, verify the receipt"
   instead of a selector script — with the resulting flow being maintainable
   afterwards precisely because it was promoted into something deterministic.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone self-hosted product | candidate | Per-device subscription, priced against the $99/device/mo comparable. Requires the security questions answered first. |
| Bundle component | likely | Ships as the device layer beneath the delivery ramps; strengthens their story rather than being sold separately. |
| Add-on | possible | Attach to a QA or deployment SKU once one exists. |
| Hosted service | rejected for now | The owner-scoped trust model makes multi-tenancy an architecture project, not a packaging choice. |

## Pricing Hypothesis

- **Model:** per-device subscription for a self-hosted control plane,
  anchored to the $99/device/mo comparable rather than to per-parallel or
  per-minute cloud pricing.
- **Cost drivers:** local runtime by default. The one variable cost is
  `ai-gateway` usage during authoring — and the promotion loop is
  specifically designed so that cost does not recur on replay.
- **Willingness-to-pay evidence:** none captured directly. The DeviceLab
  comparable is the only external signal so far.

## Honest obstacles

These are real, and none is a scheduling problem:

1. **The trust model is owner-scoped by construction.** `vrooli-bridge`
   pairing assumes one owner's fleet. Multi-tenant SaaS is not a
   configuration change; it is a different security architecture. Any
   external product starts as self-hosted, sold as software, not as a
   hosted service.
2. **The security surface is the sales obstacle.** A capability that can tap
   anything on a phone and read anything on its screen will not clear an
   enterprise review until the three open questions in
   [`../internal/SECURITY.md`](../internal/SECURITY.md) — redaction policy,
   unattended agent control, grant granularity — have real answers. Those
   are prerequisites to selling, not follow-ups.
3. **Hardware is the customer's problem, which is both the pitch and the
   friction.** "Use devices you already own" removes our capex and adds
   their ops burden. It sells well to teams that already maintain a device
   shelf and poorly to teams that wanted the shelf to be someone else's.
4. **Two different products live in one scenario.** Driving *our own
   generated apps* for deployment evidence is a contained, defensible
   capability. Driving *arbitrary third-party apps* on a personal phone is a
   far larger market and a far larger risk. They share an engine but not a
   security posture, and conflating them in a pitch would be a mistake.

## Validation signal before investing further

- Do the delivery ramps actually produce physical-device evidence that a
  reviewer trusts? If our own ramps do not depend on it, no external buyer
  will.
- Does the promotion loop hold up — does a promoted flow keep passing after
  the app under test changes, or does it need re-inference so often that the
  economic claim collapses? That is the single assumption the whole
  differentiator rests on, and it is cheap to test early.
- Does anyone ask for it unprompted once the ramps ship?

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — the open questions that gate external sale
- [Project-level monetization strategy](../../../../docs/monetization/README.md).
