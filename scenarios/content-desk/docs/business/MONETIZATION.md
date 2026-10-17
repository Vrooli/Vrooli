# Monetization — Content Desk

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

## Verdict — direct-product candidate

**Reversed 2026-07-28 (D-014).** An earlier revision of this document marked the
scenario `not-applicable` on three grounds. Two of them do not survive
inspection, and the third proves something narrower than it claimed.

| Prior reason | Why it fails |
|---|---|
| "No external buyer — value is coupled to Vrooli's marketing canon, post-type registry, and team structure." | The **seed data** is coupled; the engine is not. The post-type registry is a table any buyer seeds with their own types. The claim gate, the evidence model, and the ledger contain no marketing doctrine at all. This is the same finding as `vrooli-memory`: the vocabulary is per-tenant, the engine underneath does not change. |
| "Gates nothing a self-hoster could run, so there is no free-versus-paid boundary." | Correct, and it argues against **feature-gating** — not against monetization. `vrooli-memory` D-014 already resolved this shape: the tool is never gated, and revenue is indirect through ai-gateway inference fall-through. |
| "Touches no bundle; value accrues to the SKUs being marketed." | True of the bundle map today. It is a statement about current packaging, not about whether the capability has a buyer. |

What the capability actually is, stated without marketing vocabulary: **an
append-oriented publication ledger in which every factual assertion carries
re-runnable evidence, and publication is blocked while any assertion is
unverified.** That is not a marketing feature. It is claim substantiation with
an audit trail.

## Customer / Buyer

| Segment | Pain | Why this rather than a content tool |
|---|---|---|
| Developer-tools and infrastructure companies | Their marketing makes checkable claims about their own product, and their audience checks them. A wrong number is a credibility event. | Evidence can literally be a command run against the product. No other tool in the category has a concept of evidence at all. |
| Teams publishing AI-assisted content at volume | Generated copy asserts things confidently and wrongly. The failure is not tone, it is fact. | The gate is the missing control. Volume is exactly what makes manual fact-checking break down. |
| Agencies and contract marketers | Need to show a client what was claimed, on whose authority, and when. | The ledger is an audit trail by construction, not a reporting feature bolted on. |
| Regulated-adjacent marketing (health, finance, legal) | Claim substantiation is a legal obligation, not a preference. | Existing tooling is enterprise-priced and enterprise-shaped. See comparables. |

The differentiating question no competitor can answer: **"which of my published
posts contain a statement that is no longer true?"** That capability falls out
of the shared claim library plus scheduled re-verification, and it is the
headline. The ledger and workflow are table stakes; the gate is the product.

## Role In Vrooli

- Direct product: **candidate**, gated on the preconditions below.
- Internal capability: yes, and this is the dogfooding path that validates it.
- SKU/bundle candidate: routed to `path:docs/monetization/catalogs/CATALOG.md`; not decided here.
- Revenue line: indirect, via ai-gateway inference fall-through.

## Mechanism

The tool is **free and never feature-gated**, consistent with the fleet's BYOK
guarantee. Revenue arrives because the agentic path routes inference through
ai-gateway, which resolves local models first, then a user's own key, then a
paid hosted tier. A user who can run neither local inference nor their own key
falls through to the paid option.

```
UI action ─▶ agent-manager ─▶ opencode runner ─▶ ai-gateway ─▶ local │ BYOK │ hosted (paid)
```

**Agent invocation is what makes this mechanism produce meaningful volume.**
Claim extraction alone is bursty and small — a few calls per draft. Drafting,
revision, and evidence-hunting performed by an agent are an order of magnitude
larger, and they recur with every artifact slot. Without in-UI agent invocation
the fall-through is a rounding error; with it, inference load scales with
editorial output. This is the material difference from `vrooli-memory`, whose
demand came from recurring background compaction proportional to retained
corpus.

It is also what makes the product complete for someone outside Vrooli. An
external buyer has no marketing team. Without agent invocation they receive a
form and a gate, which is useful but thin.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Desktop application (Tier 2) | **target** | `scenario-to-desktop` is the existing Electron ramp. Free; inference falls through. Distribution and updates ride the existing landing-page download hosting. |
| Bundle component | candidate | Business bundle. Route through `path:docs/monetization/catalogs/CATALOG.md`; do not decide here. |
| Hosted team tier | P2 expansion | Collaboration is a later monetization opportunity, not a requirement. See below. |
| Add-on | not-applicable | It does not extend another SKU. |
| Service/consulting assist | possible | Claim-substantiation setup is plausible done-for-you work. Not pursued. |

### A solo user is a complete user

An earlier revision of this document claimed multi-user was on the critical path,
because an editorial gate needs a producer and a separate approver. That was
wrong, and the correction matters for scoping.

**The separation the gate depends on is producer versus approver, and an agent
drafting with a human approving already satisfies it.** The human is not marking
their own homework; they are reviewing the agent's. Where the human writes the
draft themselves, the mechanical claim checks still run and still bind, because
a re-runnable check does not care who authored the assertion. One person running
this alone gets the whole product.

That makes the shape identical to `vrooli-memory`: single machine, no tenancy,
nothing gated. Multi-user collaboration stays a genuine later opportunity —
hosting and synchronisation are legitimate paid value that a self-hoster cannot
run locally, so charging for them would not violate the BYOK guarantee — but it
is expansion, not a precondition. See D-015 and D-017.

## Pricing Hypothesis

- Model: free tool; indirect revenue through hosted inference fall-through. Possible hosted team tier, undecided.
- Comparables: **Veeva Vault PromoMats** and comparable MLR-review suites serve claim substantiation for regulated marketing at enterprise price points, which establishes willingness to pay for exactly this control. General content and scheduling tools (Buffer, Hootsuite, Later, and the current crop of AI content workspaces) are the crowded side of the market and have **no concept of verified evidence**. The gap between those two poles is where this sits — nothing lightweight, developer-shaped, and AI-native occupies it.
- Willingness-to-pay evidence: none captured directly. The enterprise category is inference, not measurement.
- Cost drivers: local runtime by default; inference through ai-gateway; hosting and sandboxed check execution if a team tier is ever built.

## Preconditions

None of these are met today. Each is a gate, not a task list.

1. **Internal dogfooding proves the gate earns its cost.** The kill signal below.
2. **Doctrine portability.** A buyer must be able to seed their own post types, failure modes, and audiences without inheriting Vrooli's. The engine already permits this; nothing has proven it.
3. **Agent invocation and authoring from the UI**, or the product is a review form and the revenue mechanism is a rounding error.
4. **Desktop packaging through `scenario-to-desktop`** (D-017), which is the delivery vehicle rather than an optimisation.

Two things that were previously listed here are no longer preconditions.
Tenancy is not one, per the note above. Evidence execution (D-016) resolves
toward the local runner because Tier 2 desktop is the target — the sandboxing
problem only exists in a hosted product that is explicitly not the plan.

## Validation Plan

- Demand signal needed: the marketing team routes real work through the desk, and at least one claim is caught by re-verification that a human review would have missed.
- Channel: see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: define from the project-level monetization taxonomy at review time.
- Revisit trigger: the P0 loop has published posts, and the contamination report has produced a true positive.

### Kill signal

**If internal use shows the claim gate is routinely bypassed, or that the claims
it catches are trivial, the product case collapses regardless of market
interest.** The gate is the whole differentiator; a gate people work around is a
workflow tax. This signal is internal, cheap, and available long before any
external move — which is the point of stating it.

## Current Status

`hypothesis` — verdict reversed 2026-07-28, preconditions unmet, no external
work scheduled. Nothing in the P0 build depends on this document being right.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-014, D-015, D-016
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`
- Free / metered / gated contract: `path:docs/concepts/PAID_FEATURES.md`
