# Go To Market — Compute Manager

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

**Nothing is implemented.** The scenario was generated from the
`react-vite` template and contains template code only. There is no
launch, no channel and no offer today. Everything below is a plan.

**Read this section first if you read nothing else.** Compute Manager is
**operator infrastructure first and a sellable product second**. The
internal capability is worth building on its own merits and is funded by
that alone. The sale is a P2 roadmap item (`OT-P2-001`) that may never
happen, and it is not a reason to build anything in P0 or P1. Treating
the sale as the point would distort the design: it would make the
bring-your-own-key path feel like a leak to be plugged, when in fact
that path is permanently free by contract and is the majority use.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What must be true before anything is launched at all?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **Audience (primary, internal):** the Vrooli operator who needs a
  machine and wants it to arrive already trusted, already metered and
  already scheduled to die. This person supplies their own provider
  credential and pays their own provider bill.
- **Audience (primary, internal, non-human):** other scenarios that need
  capacity the standing fleet does not have. Validation bursting to an
  operating system no current node runs (`OT-P2-002`), and deployments
  targeting a host that does not exist yet. These consumers reach the
  scenario through its API and CLI, not through any marketing surface.
- **Audience (future, external):** a subscriber who wants managed
  capacity without holding a cloud account at all. This audience does not
  exist yet and no evidence says it does.
- **Positioning:** capacity with a memory and a deadline. The scenario is
  the one place that knows what a machine costs so far and when it dies,
  and it hands every machine it creates to `vrooli-bridge` so it arrives
  as a trusted node rather than as an IP address.
- **Main claim:** a machine you asked for exists, is enrolled, is
  metered, and destroys itself on schedule even if this scenario is
  offline.
- **Proof needed:** one real instance created, enrolled with no
  interactive step, metered, and destroyed by its own first-boot timer
  while the control plane is stopped. That single demonstration proves
  four of the seven P0 targets at once, and until it exists there is
  nothing to say to anybody.
- **What the positioning must never claim:** that connecting a machine
  you already own costs anything. It is free forever, and saying
  otherwise in any asset is a contract violation, not a wording problem.

### Positioning against self-hosting this scenario

The most likely competitor is not a vendor. It is the reader running this
scenario themselves against their own Hetzner key, which is a path the
monetization contract guarantees will always exist and will never be
degraded. That is not a leak to be plugged and it should be said out loud
in every asset, for the same reason `scenarios/treasury` says it: the
audience most likely to adopt this is the audience least willing to be
funnelled.

The honest split is:

| You should self-host this | You might pay for capacity |
|---|---|
| You already hold a cloud account, or are willing to open one. | You will not, or cannot, hold provider credentials at all. |
| You want the fleet view, the expiry backstop and the reconciler, all of which are free forever. | You want those and also do not want a second bill, a second console, or a second thing to remember to cancel. |
| You are comfortable being the party the provider holds liable for what runs on the machine. | You would rather that liability sat with somebody else. |

The third row is the one that is usually left out, and it is the row that
actually distinguishes the two products. Under Hetzner's §7.1 the account
holder "remains the sole contractual partner" and "continues to be solely
and fully liable". A self-hoster carries that liability for their own
workloads, which is normally fine. Buying capacity moves it to Vrooli,
which is a real thing being sold and also, from Vrooli's side, the largest
risk in the whole paid path. See
[The single-supplier risk](#the-single-supplier-risk) and
[`../internal/SECURITY.md`](../internal/SECURITY.md).

There is no honest argument that the paid path is technically better. It is
the same code. The only thing being sold is not having an account.

### The paid path is the missing half of Tier 3

The paid path reads as a narrow P2 add-on for a market this document
admits may not exist. That framing is too small, and
`path:../../docs/monetization/strategy/TIERS.md` is where the larger one is
written down.

Tier 3, `hosted_cloud`, is "a managed, per-account Vrooli instance on our
infrastructure". TIERS.md calls it "probably the largest long-term revenue
surface" because it captures users who would otherwise churn on self-host
setup friction. Its revisit trigger requires three things, and the second
is that a scenario "can reliably provision a full Vrooli instance per
account on a VPS or container platform".

**That is this scenario.** The trigger names `scenario-to-cloud`, which
deploys onto a host but does not acquire one. Acquiring, metering and
retiring the host has no owner other than Compute Manager. The offer desk
graph says the same thing from the other direction: the only two edges
into the `compute_minutes` stream come from `vrooli-bridge` and
`scenario-to-cloud`, and there is no `compute-manager` node in the catalog
at all.

This changes the reason for the ordering without changing the ordering.
Tier 3's first trigger condition is that Tier 2 is `active` and shipped,
which is not this scenario's to satisfy, so the paid path still waits.
But it waits as a prerequisite for a named strategic surface rather than as
a speculative add-on, and TIERS.md's own note that Tier 3 "most changes the
company's operational posture, from software shop to infrastructure
operator" is the clearest statement anywhere of what promoting this offer
would actually commit to.

### What the provider terms permit

The choice of first adapter was made on contract terms before technical
merit, and the terms are the binding constraint on the entire paid path.
Seven providers were surveyed. Full clause text, document versions and
retrieval dates are in
[`../reference/provider-survey.md`](../reference/provider-survey.md);
the decision that follows from them is in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

| Provider | May we let a customer use capacity we bought? | Governing clause | Programme? | Per-end-customer acceptance? | Revocable? |
|---|---|---|---|---|---|
| **Hetzner** | **Yes, under the standard terms** | T&C §7.1 | No | No | Only by terminating the contract |
| DigitalOcean | Only inside the Partner Program | Partner Terms §1.1, §1.3 | Yes | **Yes, each End Customer individually** | Yes, on programme termination |
| Linode / Akamai | Prohibited by default; granted on consent to the Reseller Policy | MSA §4.4; Reseller Policy §2, §5 | Yes | Written contract required with every end user | **Yes, "revocable by Linode at any time"** |
| Scaleway | **No, for Instances.** Other product lines are silent | Instance Specific Conditions | n/a | n/a | n/a |
| AWS (Lightsail surveyed) | No | Customer Agreement §6.4(c) | Solution Provider or Distribution, neither self-serve | n/a | n/a |
| Fly.io | No | ToS §1.4(d) | No programme exists | n/a | n/a |
| Vultr | Not without written permission | AUP; ToS §5(b)(ii) | Signed Partner Agreement; programme acceptance alone is not sufficient | n/a | n/a |

**Hetzner is the only one of the seven whose standard terms permit
granting third parties use rights**, with no partner programme to join, no
per-customer acceptance step, and no separate revocable policy. §7.1 reads:
"The Customer is entitled to grant third parties a contractual term of use
to any services the Customer orders from Hetzner." Three of the seven
permit resale in some form; four forbid it or gate it behind written
permission.

Two rows deserve more than a table cell.

**The DigitalOcean conflict, stated plainly.** DigitalOcean is named in
[`../../PRD.md`](../../PRD.md) as the intended second adapter, chosen for
per-second billing and geographic diversification. But DigitalOcean's
Partner Terms §1.3 makes the partner "responsible for ensuring that each
End Customer agrees to the End Customer Terms in a manner that is legally
binding upon the End Customer", and reserves DigitalOcean's right to refuse
service to an End Customer until that acceptance is confirmed. **That is
mutually exclusive with the stated paid audience**, which is "a subscriber
who wants managed capacity without holding a cloud account at all". A
subscriber who must individually accept DigitalOcean's terms has a
relationship with DigitalOcean, which is the exact thing they were paying
not to have.

The conflict has three possible resolutions and someone has to pick one
before the second adapter is written:

1. **DigitalOcean is a second adapter for the free path only.** An operator
   using their own DigitalOcean credential is unaffected by the Partner
   Terms, because they are not reselling. This is the cheapest resolution
   and it keeps `OT-P1-004` intact, because a second provider behind the
   same interface is still delivered. The router simply must not select
   DigitalOcean for a Vrooli-owned account.
2. **The paid path accepts a click-through.** The customer accepts
   DigitalOcean's End Customer Terms during purchase. This is a smaller
   change than it sounds, and it is a large change to the product claim.
3. **A different second provider.** The survey offers Scaleway Elastic
   Metal, where the Instance-only resale prohibition does not reach. That
   is a different product shape and a different price point.

Resolution 1 is the recommendation, because it separates the technical goal
of `OT-P1-004` (a second provider changes no caller) from the commercial
question (which providers may serve a paid request), and those two are not
the same question. The routing pool for Vrooli-owned accounts is a
narrower set than the adapter set, and that distinction should be encoded
rather than assumed.

**Linode permits resale, but the polarity is easy to get backwards.**
Akamai's MSA §4.4 is headed *Resale Prohibited* and prohibits it by
default. Permission comes from a separate Reseller Policy that grants "a
non-exclusive, revocable right to resell", "revocable by Linode at any
time". That policy also carries a channel prohibition, §5, verbatim:
"Reseller is strictly prohibited from marketing, soliciting, or selling any
Eligible Service to any current customer of Linode, provided that Reseller
is not prohibited from responding to any inbound inquiry." Inbound is
fine; outbound to anyone who is already a Linode customer is not. **That is
a constraint on marketing, not on engineering**, and it is the kind of
constraint that gets breached by a well-meaning campaign rather than by a
deliberate decision. Both Linode documents also predate the Akamai rebrand
and have not been revised since 2023 and 2019 respectively.

### The single-supplier risk

**This is the largest commercial risk in the paid path, and it is larger
than the demand risk.**

The only provider whose standard terms permit the model is Hetzner. Of the
other two that permit resale at all, one requires per-customer acceptance
that contradicts the product claim, and the other grants permission that is
revocable at any time. There is no contractual second source. If Hetzner's
§7.1 changes, or if the account is locked under §8.3 or §5.2, the paid path
stops and there is nowhere to move it to.

That supplier has just repriced. Effective 15 June 2026, announced
27 May 2026, Hetzner raised prices for new orders and rescales across all
cloud plans and all dedicated servers at all locations. Derived from
Hetzner's own published per-plan prices, because **Hetzner published no
percentages at all**:

- The EU CX and CAX shared lines rose **30 to 38 percent**.
- The CPX shared line rose **143.9 to 175.4 percent** in the EU, **166.8 to
  209.0 percent** in the USA, and **50.8 to 93.9 percent** in Singapore.
- The CCX dedicated-vCPU line rose **113.4 to 173.1 percent** in the EU,
  **107.1 to 157.4 percent** in the USA, and **65.3 to 110.7 percent** in
  Singapore.

The convenient summary "shared rose by about a third" is false: CPX is a
shared line and it rose by up to six times that. §1.3 of the same T&C
reserves Hetzner's right to change terms, System Policies and prices on
notice. Nineteen days of notice was what the last change came with.

Two things follow for the launch:

- **No multi-year or fixed price may be published.** Any price carries a
  supplier-change clause and a revision mechanism. A worked example of what
  happens without one is in [`MONETIZATION.md`](MONETIZATION.md).
- **The single-supplier dependency is an explicit operator acceptance**,
  not an assumption. It belongs on the launch-gating checklist below as a
  decision somebody signed, because the alternative is discovering it
  during an outage.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal capability rollout (operator, direct) | The operator stops opening a provider console because the CLI and the inventory dashboard are strictly better for capacity they will destroy soon. | The CLI with full headless parity, the inventory surface (`OT-P1-005`), and the runbook procedures the requirements already reference. | The operator provisions and destroys through this scenario rather than the console, and the reconciler finds no orphans they created by hand. |
| Composition by other scenarios | Validation and deployment scenarios call for capacity rather than assuming a standing fleet. | A stable provider-agnostic API, the enrollment path via bridge, and `../concepts/INTEGRATIONS.md`. | The cross-operating-system gate accepts a node that did not exist when the gate started (`OT-P2-002`). |
| Dogfood on Vrooli's own bursty work | Short-lived capacity for test and build work is the highest-frequency internal demand and the fastest way to surface hourly-rounding cost reality. | The fake-provider spine plus one real Hetzner adapter, and daily cost reconciliation (`OT-P1-003`). | Rounding waste computed over at least 30 completed instances, and meter-to-statement divergence below 2 percent of the provider-billed total over a full billing period. |
| Self-hosting and homelab communities | The audience that already refuses hosted vendors is the audience that most wants a fleet view with an expiry backstop, and the whole free path costs them nothing. | A one-command local setup against their own provider key, and the unattended lifecycle demonstration. | Installs that proceed past setup to a first instance created and destroyed on its own timer. |
| Operator-run paid offer (future, P2) | A subscriber who will not hold cloud credentials pays a margin for capacity through the existing subscription and credit rails. | Everything on the launch-gating checklist below. | An identified buyer who has the free path available and still asks to be billed. Deferred until the internal capability is proven. |

### The promotion rule

The paid offer is promoted to active by an **operator, never an agent**.
This is not a process nicety. Promoting an offer commits Vrooli to paying
a real hourly provider bill on behalf of a customer, under a single
supplier whose prices moved by up to 209 percent in June 2026 with
nineteen days of notice, and whose terms permit the model only because
they were read per service rather than per provider. It also accepts,
on the customer's behalf, that one other customer's abuse can lock the
shared account and stop every instance. That is a commercial commitment
with an unbounded downside, and pricing, bundle membership and the
decision to sell at all are operator-curated canon in any case.

An agent may prepare the offer, gather the evidence and say plainly that
the prerequisites are met. It stops there. The same rule covers the
catalog: creating a `compute-manager` deliverable node and rewiring the
`unlocks` edge into `compute_minutes` are catalog mutations, and the
catalog is operator-curated.

## Launch Motion

This is an internal capability rollout. The commercial launch is the last
step and is conditional. **The free path ships first**, because it is the
only claim that is already settled and because holding it behind
commercial work would be exactly backwards.

1. **Ship adopt-a-machine-you-own (`OT-P1-001`) and say the free claim out
   loud.** Adoption records no instance, no intent and no reservation. It
   is free forever by contract, it is the majority use, and it is the one
   message in the table below whose status is `settled` rather than
   `planned`. It should not wait behind anything, and least of all behind
   anything commercial.
2. **Build the four failure modes against a fake provider**, because none
   of them needs a real API key: reserve, provision and settle; intent
   written before any provider call; bidirectional reconciliation; and
   double-enforced expiry. If these do not hold, nothing built on top of
   them is trustworthy.
3. **Ship the operator inventory surface (`OT-P1-005`).** The free path is
   only as good as the fleet view, and the fleet view is what replaces the
   provider console.
4. **Wire the Hetzner adapter** once step 2 holds. Chosen first because its
   standard terms permit granting third parties use rights with no partner
   programme and no per-customer acceptance, it bills outbound traffic
   only, and it leaves inbound UDP unencumbered.
5. **Land unattended enrollment** once `vrooli-bridge` publishes its
   onboarding public key. That endpoint does not exist yet and is an
   upstream prerequisite, not integration work.
6. **Run daily cost reconciliation against a real provider statement**
   until metered usage and the provider bill agree to within 2 percent of
   the provider-billed total over a full billing period. Until they do, no
   price can be defended. Note that no provider publishes a
   billing-latency SLA, so reconciliation must tolerate unbounded lateness
   and correct retroactively rather than assume a settlement window.
7. **Publish the acceptable-use policy and write the abuse runbook.**
   Both are launch assets, not paperwork. Hetzner's T&C §8.3 prohibits
   crypto mining ("mining, farming and plotting"), §5.2 prohibits (d)DoS
   sourcing and open relays, and both reserve the right to lock the
   account rather than the server, §5.2 explicitly "without prior notice".
   A customer mining on capacity Vrooli bought is Vrooli's breach. The AUP
   must be at least as strict as the strictest provider in the routing
   pool, and the runbook must exist before it is needed, because a runbook
   written during an incident is written too late.
8. **Only then:** declare the meter, seed the tier limits, gather the
   evidence, and hand an operator the decision about whether to promote an
   offer at all.

### Launch-gating checklist

Every line is a hard gate on the paid launch. None is a gate on steps 1
through 6, which are the free path and ship on their own merits.

| # | Gate | Evidence that it is met |
|---|---|---|
| 1 | **Acceptable-use policy published** | A public AUP naming crypto mining, (d)DoS sourcing and open relays explicitly, at least as strict as the strictest provider in the routing pool. |
| 2 | **Provider terms re-read per service, not per provider** | [`../reference/provider-survey.md`](../reference/provider-survey.md) refreshed with a current retrieval date for every provider in the routing pool, including the per-service annexes. Scaleway is the worked example of why: its general terms are silent on resale and its Instance annex forbids it. |
| 3 | **Tier limits seeded** | Rows exist in `subscription_tier_limits` for `compute_minutes` across all five `plan_tier` values, and an integration test shows a reservation over the limit returning the insufficient-credits sentinel. Without rows, every reservation succeeds and the meter is decorative. |
| 4 | **Meter declared** | `.vrooli/monetization.json` exists with `compute_minutes` as class `A`, and `enforcement_paths` pointing at real reservation code rather than `api/main.go`. |
| 5 | **Inventory regenerated** | `go run ./cmd/meter-inventory` has been run in `packages/monetization-go` and the drift test passes. |
| 6 | **Offer desk node and edge created** | A `compute-manager` deliverable node exists and the `unlocks` edge into `compute_minutes` originates from it. Operator work; today the catalog has no node for this scenario at all. |
| 7 | **Offer promoted by a human** | A catalog record moved to `active` by an operator. Agents never self-promote. |
| 8 | **Abuse runbook exists** | A written procedure covering intake, who may destroy an instance without consensus, what the customer is told, and what is preserved for the provider. |
| 9 | **Single-supplier risk accepted** | A recorded operator decision accepting that one supplier permits the model, that its prices moved by up to 209 percent with nineteen days of notice, and that a shared-account lock stops every customer at once. |

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Connect a machine you already own, free, forever." | All audiences | The monetization contract forbids gating what a self-hoster could run with their own keys; adoption records no instance, no intent and no reservation. | settled, unbuilt |
| "Capacity that destroys itself, even when Vrooli is down." | Operator, cost-sensitive adopters | Expiry enforced twice (`OT-P0-004`), including a first-boot timer that needs no control plane. | planned |
| "There is no pause button, and that is the feature." | Operator, cost-sensitive adopters | A stopped instance still bills at the full rate on most providers. Destroy is the only stop (`OT-P0-007`), asserted structurally. **The specific "five of the seven surveyed" figure is unsourced**; see the unsourced-claims table in [`../reference/provider-survey.md`](../reference/provider-survey.md). Do not use the number in an asset until it is sourced. | planned, partly unsourced |
| "Every machine is accounted for in both directions." | Operator, security-minded adopters | Bidirectional reconciliation (`OT-P0-003`), which reports divergence and never resolves it silently. | planned |
| "It arrives as a trusted node, with no password on any wire." | Operator, security-minded adopters | Unattended enrollment through the bridge onboarding contract (`OT-P0-005`); the instance trusts the bridge key from first boot. | planned, blocked upstream |
| "You will know what it cost before the bill arrives." | Operator, future subscriber | Metering from transitions this scenario caused, plus daily reconciliation against the provider statement (`OT-P1-003`). | planned |
| "Run it yourself against your own key. That path is not the degraded one." | Self-hosters | The free/metered split in [`MONETIZATION.md`](MONETIZATION.md); the paid path is the same code with a different credential. | settled, unbuilt |
| Any claim about price, tiers or included capacity | Future subscriber | None. Pricing is operator canon and no offer exists. | forbidden until an operator promotes an offer |

**Messages deliberately not used.** Nothing that implies capacity is
suspended rather than destroyed when credit runs out. Because destroy is
the only stop, exhausting credit mid-instance is a customer-visible
data-loss event, and any asset describing the paid path must say so rather
than let a customer infer the platform-as-a-service behaviour that every
comparator has. Nothing comparing prices against Fly.io, Render, Railway,
Heroku, Northflank or DigitalOcean App Platform, because no price
comparison has been retrieved. And nothing marketing capacity to an
existing Linode customer, if Linode ever enters the routing pool, because
the Reseller Policy §5 forbids it.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| The unattended lifecycle demonstration | Internal rollout | One real instance created, enrolled with no interactive step, metered, and destroyed by its own timer with this scenario stopped. | Proves the capability is real. Nothing else may be claimed publicly before this passes. |
| Console abandonment | Internal rollout | The operator's next five capacity needs are met through the CLI or dashboard rather than the provider console, and the reconciler reports zero hand-created orphans over that period. | Confirms the internal capability is better than the alternative it replaces. |
| Rounding cost measurement | Dogfood | Rounding waste, defined as provider-billed time minus customer-billed time over provider-billed time, computed over at least 30 completed instances. | Decides whether warm pooling (`OT-P2-003`) is worth building, or whether a minimum billable unit alone is sufficient. |
| Meter versus statement agreement | Dogfood | Meter-to-statement divergence below 2 percent of the provider-billed total across a full billing period, on the provider serving the majority of instance-hours. Two percent is roughly one hour in fifty, which is the granularity Hetzner's own rounding imposes; replace it with a measured figure once one exists. | Gate on any pricing conversation. A price defended by a meter that disagrees with the supplier is a guess. |
| Free-path share | Internal and self-hosting | Fraction of instances created against an operator's own credential rather than a Vrooli-owned account, measured after the paid path ships. | The honest denominator. If it stays near 100 percent, the paid path has no customer and the scenario stays internal infrastructure. |
| Paid-capacity interest | Future external | At least one identified buyer who has the free bring-your-own-key path available, understands it, and still asks to be billed by Vrooli. | The only signal that justifies preparing an offer for operator promotion. Absent it, the scenario stays internal infrastructure and that is an acceptable end state. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md): role, packaging, the free/metered/gated split, the manifest and seed specs, and the prerequisites for selling capacity
- [`../reference/provider-survey.md`](../reference/provider-survey.md): the clause text, document versions and retrieval dates behind the terms table above, plus the rounding and price-change facts
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md): why Hetzner is first, why there is no pause, and the per-service annex method rule
- [`../internal/SECURITY.md`](../internal/SECURITY.md): account termination as a fleet-wide availability failure, and the missing per-tenant isolation the paid path depends on
- [`../../PRD.md`](../../PRD.md): product outcomes and launch sequencing, including `OT-P1-004` and the DigitalOcean second-adapter intent
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md): bridge, business suite, treasury and offer desk seams
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md): the timing budgets that turn into cost
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md): validation signals and telemetry
- Paid-features contract: `path:../../docs/concepts/PAID_FEATURES.md`
- Delivery tiers and the Tier 3 revisit trigger: `path:../../docs/monetization/strategy/TIERS.md`
