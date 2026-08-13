# Monetization — Money Ledger

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

Money Ledger is **all four roles at once**, which is unusual and is the reason it
is worth building carefully:

| Role | Status | Detail |
|---|---|---|
| Internal capability | **committed** | It replaces the `financial-tracker` member's hand-executed runway formula and eleven-field snapshot schema. That is the reason it exists and the only role that needs no validation. |
| Direct product | **candidate** | The honest-numbers thesis below is a real product hypothesis with real comparables. Unvalidated. |
| SKU component | **candidate** | A financial scenario is the canonical example of many-to-many membership in `path:docs/monetization/catalogs/CATALOG.md`; it plausibly belongs to both the business and lifestyle bundles. |
| Service accelerator | **not-applicable** | Bookkeeping-as-a-service is a licensed, liability-bearing activity. Explicitly out of scope. |

The internal role does not depend on the product role being validated. If the
product hypothesis fails, the scenario is still correct and still earns its keep.

## The thesis

Every consumer and small-business finance product answers "what is my balance."
They differ in how they behave **when they cannot answer honestly**:

- An aggregator whose bank connection breaks shows a stale balance, or a zero, and
  usually does not distinguish those from a real zero.
- A spreadsheet has no concept of provenance at all — a typed number and an imported
  number look identical a month later.
- Double-entry accounting software has provenance but demands an accounting mental
  model most operators do not have and will not acquire.

Money Ledger's claim is narrow and testable: **every figure carries its basis, and a
figure that cannot be computed says so instead of showing a number.** `authoritative`,
`derived`, and `operator-asserted` are a first-class field on every event, and position
is a query that can name the adapter it is missing. That is the whole product.

This is a *trust* wedge, not a *features* wedge. Competing on feature count against
incumbents with years of integration work is not winnable and is not the plan.

### What the competitive scan changed about this thesis

Scanned 2026-08-13; full comparison in [`GO-TO-MARKET.md`](GO-TO-MARKET.md).

The thesis survives, but **one of its assumed supports does not.** "Local-first, no bank
credentials" was expected to carry weight; it does not. SenticMoney sells precisely that
posture to precisely this audience at $39/year, and Actual Budget and Firefly III give it
away. Privacy is now the price of entry to this segment.

What remains genuinely unoccupied is narrower and better: **every competitor is either
fully automatic or fully manual.** An aggregator-first product has no manual sources worth
distinguishing; a no-credential product has no automatic ones. Neither has a reason to
model how much a given figure can be trusted, so neither does. We admit both through one
contract, which is the only reason `basis` needs to exist at all.

So the wedge is not privacy and not honesty-in-the-abstract. It is **mixed provenance**:
the operator whose revenue arrives partly through an API and partly as cash is the one
person no existing product serves without making them lie to it.

Two honest caveats. A competitor could add a provenance field — it is not technically
hard; what is hard is retrofitting it through a product built on the assumption that every
figure came from the same kind of place. And if the mixed-source claim *also* fails to
differentiate under real demand testing, the correct move is to drop the direct-product
hypothesis rather than reposition a third time. The internal role does not depend on it.

## Customer / Buyer

- **Primary user:** an operator whose money moves through more than one place and who
  does not have a bookkeeper — a solo founder, freelancer, contractor, small landlord,
  or a household treating itself as a small business. The distinguishing trait is not
  income level; it is **having at least one revenue source with no API**, which is where
  every aggregator-first product degrades and this one does not.
- **Buyer = user.** No procurement, no seat expansion, no admin persona. This constrains
  price far more than it constrains features.
- **Pain:** the operator cannot answer "how long do I have" without half an hour of
  manual reconciliation, and does not trust the answer when they get it.
- **Existing alternatives:** a spreadsheet (most common), a personal-finance aggregator,
  small-business accounting software, or nothing. See `GO-TO-MARKET.md` for the
  comparison that positioning depends on.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | candidate | The strongest standalone case of any scenario in the monetization set. Local-first, no hosting cost, no per-user COGS beyond gateway usage if AI categorisation is enabled. |
| Bundle component | candidate | Plausibly in both `business` and `lifestyle`. Membership is proposed through Offer Desk once it owns the catalog, not asserted here. |
| Add-on | rejected | It is a base capability, not an extension of another SKU. Modelling it as an add-on would put it behind a parent bundle it does not depend on. |
| **Open money-event contract** | **candidate** | The contract published as a spec anyone may implement, with the ledger as its reference implementation. See below — this is the most durable asset here and the least like the others. |
| Service/consulting assist | not-applicable | See Role In Vrooli. |

### The contract as the asset

The thesis above is about a *product*. The most defensible thing this scenario produces may not be the product at all: it is the **money-event contract** — a typed, versioned, public shape that any source can satisfy, carrying a date, a signed amount, an account, provenance, and a basis.

Two reasons to record it as a packaging option rather than leaving it implicit:

- **It inverts the integration-count problem.** The direct-product thesis explicitly refuses to compete on integration count (`GO-TO-MARKET.md`), because that axis is unwinnable against incumbents with years of adapter work. A published contract makes integration count something *other people* contribute to, which is the only version of that axis worth being on.
- **It is what the architecture already is.** `OT-P0-004` defines one inbound door and no privileged path; `INTEGRATIONS.md` states outright that the scenario "does not integrate with named systems — it defines one inbound shape and lets systems satisfy it." Publishing the shape costs a specification and a conformance test, not a new capability.

Two honest limits. A contract with one implementation is a file, not a standard — this stays `candidate` until at least one adapter exists that we did not write. And it is a *distribution* strategy, not a *revenue* strategy: it plausibly makes the OSS-discovery channel work, and it does not by itself make anyone pay. Do not let it become a reason to defer the demand testing the direct-product hypothesis still needs.

**Revisit trigger:** the manual, file, and commerce adapters are all green — proving the contract against three genuinely different source shapes — **and** an external party has asked how to feed the ledger from a source we do not support.

## Pricing Hypothesis

- **Model:** one-time purchase or low flat subscription, per operator. Not per-seat,
  not per-account, not percentage-of-assets.
- **Explicitly rejected — usage-based pricing on money volume.** Charging as a function
  of transactions or balances punishes the user for the product working, and it makes
  the vendor's incentive diverge from the honesty thesis. It is also the pricing model
  most likely to make an operator keep a shadow spreadsheet.
- **Cost drivers:** local runtime by default, so per-user COGS is near zero. The only
  variable line is gateway tokens if AI-assisted categorisation ships, which is why that
  feature must be optional and must degrade to rules rather than being load-bearing.

### Comparable pricing — observed, not validated

Observed 2026-08-13 from public marketing pages. Honesty flag: **`operator-asserted`**.
These are list prices read off websites, not captured comps — `market-validator` still
owns turning them into evidence through a `validation-inbox/*` entry, including whether
each product's audience actually overlaps ours.

| Product | List price | Shape |
|---|---|---|
| Actual Budget, Firefly III | free | self-hosted open source |
| SenticMoney | ~$39/yr | local-first, no bank login — the closest comparable |
| Copilot Money | ~$95/yr | aggregator-first, Apple-centric |
| Monarch Money | ~$100/yr core, ~$199/yr plus | aggregator-first, forecasting and business tracking in the upper tier |
| YNAB | ~$109/yr | method-led budgeting |

**What this bounds.** The segment prices between free and roughly $110/year, and the
nearest comparable sits at the bottom of that band. A subscription materially above
SenticMoney would need the mixed-source claim to be doing real work for the buyer, and
free self-hosted alternatives cap what the technical audience will pay at all.

**Still no price is set here**, and the decision recorded in `../internal/DECISIONS.md`
stands: a number in this document becomes canon by accident. This table narrows the range
for the eventual conversation; it does not conclude it.

## Validation Plan

- **Demand signal needed:** operators who describe the *honesty* failure specifically —
  "I don't trust the number", "it silently showed zero" — rather than asking for more
  integrations. Requests for more integrations are a signal for a different product.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** to be set from the project monetization taxonomy at promotion
  time. Deliberately unset here.
- **Revisit trigger:** `journal`, `ingest`, and `position` are green with the manual and
  file adapters shipped, **and** the ledger has held the operator's own real financial
  data through one full month including at least one adapter outage. Self-use before
  external claim is the whole point of the dogfooding argument.

## The self-referential asset, stated carefully

Vrooli runs its own monetization on this scenario. That is a genuine go-to-market asset
and it is also the most likely place to fool ourselves. Two disciplines:

1. **Vrooli is the first user, not the shape of the product.** Already binding in the
   PRD: no upstream is named in any P0 target, and the money-event contract is defined
   by the shape of an event rather than by any processor's API.
2. **A feature is not validated by our own use of it.** Dogfooding proves the thing
   runs; it does not prove anyone would pay. Keep those two claims separate in every
   downstream document.

## Current Status

`candidate` — internal capability is committed and needs no validation. The direct-product
and SKU-component roles are hypotheses with a stated revisit trigger and no price.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`.
