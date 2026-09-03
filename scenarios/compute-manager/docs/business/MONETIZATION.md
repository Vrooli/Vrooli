# Monetization — Compute Manager

This document records how the scenario could create revenue or support a
monetizable Vrooli capability. Keep it honest: `not-applicable` is better
than inventing a commercial story.

**Nothing in this scenario is implemented.** It was generated from the
`react-vite` template and contains template code only. Everything below
describes an intended contract and a hypothesis, not behaviour that
exists. No instance has ever been created, no credit has ever been
reserved, and no offer has ever been sold.

**Pricing, bundle membership and whether to monetize at all are
operator-curated canon.** This document states a hypothesis and the
evidence behind it; it does not set strategy. Read, do not write:
`path:docs/monetization/README.md`,
`path:docs/monetization/strategy/STRATEGY.md`, and the catalog. Wiring a
paid feature follows `path:docs/concepts/PAID_FEATURES.md`.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What has to be built, seeded and declared before a single credit can be
  charged, and where does each of those things live?
- What validation signal would justify more investment?

A note on vocabulary. The words `deferred`, `not-applicable` and
`hypothesis` are used below as document-local status labels. They are
**not** Offer Desk catalog states. The catalog lifecycle is `idea`,
`candidate`, `trigger-met`, `proposed`, `active`, `shipped`, `retired`,
per `path:docs/monetization/catalogs/CATALOG.md`. Nothing in this document
sets or reports a catalog state.

## Role In Vrooli

Compute Manager is **operator infrastructure first and a sellable product
second**, and the ordering is not a hedge. The capability it supplies to
the rest of the system (capacity that arrives already trusted, already
metered, and already scheduled to die) is worth building whether or not
anyone outside Vrooli ever buys a minute of it.

- **Direct product: roadmap scope.** `OT-P2-001` is the only monetized
  target and it is explicitly P2. Selling capacity requires an operator
  decision that has not been taken.
- **Internal capability: primary role.** Every scenario that needs a
  machine that does not exist yet composes this one instead of learning a
  provider API. Validation bursting past the standing fleet
  (`OT-P2-002`), deployments targeting a host that does not exist yet,
  and the operator buying their own capacity without opening a provider
  console are all the same capability.
- **SKU/bundle candidate: business bundle, proposed only.** Bundle
  membership is operator canon and is not decided here. It pairs
  naturally with `vrooli-bridge` (which turns the capacity into a trusted
  node) and `landing-page-business-suite` (which holds the wallet).
- **Revenue line: metered usage under the `compute_minutes` limit key**,
  Class A, cost-bearing. See
  [Selling Provisioned Capacity](#selling-provisioned-capacity).

### The delivery tier this actually serves

The framing above treats the paid path as a narrow P2 add-on, and that
framing understates it. `path:docs/monetization/strategy/TIERS.md`
describes Tier 3, `hosted_cloud`, as "a managed, per-account Vrooli
instance on our infrastructure", calls it "probably the largest long-term
revenue surface", and gives it a revisit trigger whose second condition is
that a scenario "can reliably provision a full Vrooli instance per account
on a VPS or container platform".

**That condition is this scenario.** The trigger names `scenario-to-cloud`
as its subject, and `scenario-to-cloud` deploys onto a host; it does not
acquire one. The acquire-and-meter half of Tier 3 has no owner other than
this scenario, and the offer desk graph agrees: the only two edges pointing
at `compute_minutes` come from `vrooli-bridge` and `scenario-to-cloud`.

Two consequences follow, and they pull in opposite directions:

- The paid path is not a speculative side quest. It is the missing half of
  a delivery tier that project canon already names as the largest
  long-term revenue surface, which is a materially better reason to build
  it than "a subscriber might want it".
- It is still gated behind Tier 2 being `active` and shipped, which is the
  trigger's first condition and is not this scenario's to satisfy. The
  ordering does not change; the reason for the ordering does.

Tier 3's own cost-of-goods note is the sharpest available statement of the
risk: "this is the tier where unit economics matter most". Everything under
[Pricing Hypothesis](#pricing-hypothesis) is that sentence made specific.

## Customer / Buyer

- **Primary user (today, unpaid):** the Vrooli operator who needs a
  machine, wants it to become a trusted node without a first-touch
  ritual, and wants one place that knows what the machine has cost so far
  and when it dies. This user supplies their own provider credential and
  pays their own provider bill. They are never charged by Vrooli.
- **Primary user (later, paid):** a subscriber who wants capacity without
  holding a cloud account at all, and is willing to pay a margin over the
  provider rate for that convenience.
- **Buyer:** in both cases the same person as the user. This is a
  self-serve purchase through the existing subscription and credit rails,
  not an enterprise sale, and `OT-P2-001` requires it to stay that way
  rather than growing a second payment path.
- **Pain, stated concretely:** the failure mode is not "provisioning is
  hard", it is "provisioning is easy and forgetting is easier". An
  instance created while the API response was lost bills forever and
  appears in no inventory. A provider spend alert sends mail and stops
  nothing. And the first honest external signal that something is wrong
  arrives late by an amount nobody can bound: **no provider in scope
  publishes a billing-data latency SLA at all.** AWS documents an
  approximately 24-hour Cost Explorer refresh and then explicitly states
  that some data may arrive later than that; DigitalOcean, Vultr, Hetzner,
  Linode, Scaleway and Fly.io document nothing. See
  [`../reference/provider-survey.md`](../reference/provider-survey.md).
  The product claim is cost visibility and enforced mortality, not a nicer
  create button.
- **Existing alternatives:** the provider console (free, no memory, no
  fleet view, no expiry), infrastructure-as-code tooling (excellent at
  declaring capacity, indifferent to what it costs per hour and to
  whether it should still exist), and hosted platform-as-a-service
  vendors (which solve this by owning the machine and charging a large
  markup). None of them make the machine a trusted node in an existing
  fleet, which is the part that only composes inside Vrooli.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | not-applicable | Capacity without a fleet to join is a worse provider console. The value comes from composing with `vrooli-bridge` and the wallet, so it does not ship alone. |
| Bundle component | candidate | The most likely path. Proposed for the business bundle; membership is operator canon and is proposed here only, never decided here. |
| Add-on | candidate | "Managed capacity" as an add-on to a deployment or validation tier, where Vrooli supplies the machine the subscriber does not own. |
| Tier 3 substrate | candidate | The acquire-and-meter half of `hosted_cloud`. See [The delivery tier this actually serves](#the-delivery-tier-this-actually-serves). |
| Service/consulting assist | deferred | Standing up bounded, self-destructing fleets for a client is plausible done-for-you work, but it follows product validation rather than preceding it. |

### The free / metered / gated split

The governing rule is the one in `path:docs/concepts/PAID_FEATURES.md`:
if a self-hoster could run it with their own keys, gating it means the
framing is wrong. That rule decides almost everything here, because
almost all of this scenario is exactly that.

| Capability | Class | Reason |
|---|---|---|
| Adopting a machine the operator already owns (`OT-P1-001`) | **free, permanently** | Vrooli pays nothing and creates nothing. Adoption records no instance, no intent and no reservation, which is asserted by the intended unit test behind `COMPUTEM-P1-001`. Metering a machine the user already bought would be charging for a network call. |
| The whole lifecycle spine against the operator's own provider credential | **free, permanently** | Create, describe, list, destroy, expiry, reconciliation and the inventory surface all run locally against a key the operator supplies. Vrooli bears no marginal cost, so there is nothing to meter. This is the bring-your-own-key path and it must stay a first-class path, not a degraded one. |
| Bidirectional reconciliation and the unaccounted-instance report | **free, permanently** | This is the scenario's integrity feature. An operator who cannot see an orphan cannot trust the tool at all, and an integrity feature behind a paywall is a worse product for everyone including the people who pay. |
| Double-enforced expiry, including the instance-side timer | **free, permanently** | Same reasoning. The backstop that drains the fleet when the control plane is down is a safety property, not a premium one. |
| Capacity Vrooli provisions on a Vrooli-owned provider account | **metered, Class A** | Vrooli pays a real hourly bill. This is the one thing here with a genuine marginal cost, so it is the one thing metered. |
| Off-box retention of cost and reconciliation history beyond the local instance | **deferred, would be metered** | Storage cost, and only relevant to an operator who wants durability off the box. Not scoped, listed so the boundary is explicit. |

**Nothing is gated.** There is no plan-tier flag anywhere in the P0 and
P1 scope, and that is deliberate rather than unfinished. Every candidate
for a gate here (a fleet ceiling, a provider choice, a longer maximum
lifetime) is either a safety limit that should apply to everybody or a
limit that already falls out of the wallet balance. A per-tenant ceiling
computed from our own meter (`OT-P1-002`) is a spend bound, not an
entitlement.

That claim has a machine-readable consequence which is easy to get wrong.
`.vrooli/schemas/monetization.schema.json` declares
`"requires_entitlement": { "type": "boolean", "default": true }`. A manifest
that omits the field therefore asserts the opposite of the paragraph above:
it says the scenario as a whole requires an entitlement. **The field must be
written explicitly as `false`**, following
`scenarios/web-console/.vrooli/monetization.json`, which is the one manifest
in the fleet that has made the same choice. Every other declared scenario
(`ai-gateway`, `agent-manager`, `audio-tools`, `switchboard`,
`vrooli-bridge`, `browser-automation-studio`) sets it `true` because each of
them does gate something. This one does not, so the honest value is `false`
and the empty `features` array is the corroborating evidence.

## Selling Provisioned Capacity

This section is the referenced target of the `COMPUTEM-P2-001` business
validation. It is the intended mechanism, and none of it is built.

### The meter, and what the catalog actually says

The limit key is `compute_minutes`. Three facts about it matter.

**It exists as a stream node in the live offer desk catalog, with status
`idea`.** Not `active`. The node carries `status = 1`, which is `IDEA` in
`packages/proto/schemas/offer-desk/v1/offers/offers.proto:13`. Sibling
documents in this scenario describe it as active; they are wrong and this
row is the checkable one.

**Its two edges come from elsewhere, and neither is this scenario.**
`vrooli-bridge --unlocks--> compute_minutes` and
`scenario-to-cloud --enables--> compute_minutes`. **There is no
`compute-manager` node in offer desk at all**, of any kind. The catalog
records the sellable shape but attributes it to two neighbours, one of
which merely enrolls the machine and the other of which deploys onto it.

**It is an undeclared meter.** It does not appear in
`packages/monetization-go/meter-inventory.json`, which lists exactly
`ai_credits`, `voice_minutes` and `workflow_executions`. In the repository
the key appears only in this scenario's own documents and in one
`landing-page-business-suite` limit-key test fixture
(`api/usage_concurrent_test.go:478`). The catalog row above is live
database state, not repository state, which is why both statements are
true at once.

Run `offer-desk offers meters` to see this. The command binds to
`CatalogService.GetMeterInventory`
(`scenarios/offer-desk/cli/manifest.json`), and its response carries two
distinct diagnostics that are easy to conflate:

| Diagnostic | Where it is computed | What it means | Does declaring a manifest here clear it? |
|---|---|---|---|
| `undeclared_streams` | `scenarios/offer-desk/api/handlers/offers/meter_inventory.go:85` | A STREAM node whose name is in no committed meter declaration. Node status is not consulted. | **Yes.** Declaring `compute_minutes` in this scenario puts the key in the committed inventory, and the name stops being unknown. |
| `deliverable_meter_gaps` | `meter_inventory.go:101` | A DELIVERABLE that `unlocks` a STREAM the deliverable's own scenario does not declare. | **No.** |

The gap loop scans **only** `unlocks` edges (`meter_inventory.go:93-95`),
skipping `enables` and every other edge kind, and it keys the declaration
lookup on the *source deliverable's* name. Today the only `unlocks` edge
into `compute_minutes` starts at `vrooli-bridge`, so the reported gap is
`vrooli-bridge -> compute_minutes` and it persists no matter what this
scenario declares, because `vrooli-bridge` does not declare
`compute_minutes` (`scenarios/vrooli-bridge/.vrooli/monetization.json`
declares only `workflow_executions`).

**Clearing the gap is operator catalog work, not code.** It needs a
`compute-manager` DELIVERABLE node created and the `unlocks` edge rewired
to originate from it. Both are catalog mutations, and the catalog is
operator-curated. An agent may prepare the request and say the
prerequisites are met; it stops there. Until then, expect a green
`undeclared_streams` and a red `deliverable_meter_gaps`, and do not read
the second as a defect in this scenario.

### The manifest to declare

Declaring the meter means writing `.vrooli/monetization.json` in this
scenario. `path:.vrooli/schemas/monetization.schema.json` sets
`"additionalProperties": false` at the top level and requires `version`,
`bundle_key`, `app_key`, `features` and `meters`. `version` is
`{ "const": 2 }`. The intended shape, once enforcement code exists:

```json
{
  "$schema": "../../../.vrooli/schemas/monetization.schema.json",
  "version": 2,
  "bundle_key": "business_suite",
  "app_key": "compute-manager",
  "requires_entitlement": false,
  "features": [],
  "meters": [
    {
      "limit_key": "compute_minutes",
      "class": "A",
      "byok": true,
      "enforcement_paths": ["api/internal/meter/reserve.go"]
    }
  ]
}
```

Four field decisions, each with its reason:

- **`requires_entitlement: false`.** Argued above. The schema default is
  `true`, so omitting the field would silently assert a gate this document
  says does not exist.
- **`byok: true`.** This is the machine-readable form of the free-forever
  path. `path:docs/concepts/PAID_FEATURES.md` states the rule as "BYOK
  must remain a valid path: a metered feature falls back to the user's own
  provider key with no credit charge", which is exactly the first four rows
  of the free/metered table. The field surfaces in the generated inventory
  (`ai_credits` already carries `"byok": true` there), so declaring it puts
  the free path on the record rather than only in prose.
- **`features: []`.** Nothing is gated, so there is nothing to list. An
  empty array is required by the schema, not optional.
- **`enforcement_paths`** must point at the code that performs the reserve,
  not at `api/main.go`. The path shown is a placeholder for a file that
  does not exist yet; the schema requires `minItems: 1`, so the manifest
  cannot be written truthfully before the code is.

**`bundle_key` is canon and is not this scenario's to choose.** Every
declared scenario in the fleet uses `business_suite`, but offer desk holds
no `belongs_to` edge for `compute-manager`, because it holds no node for it
at all. The value must come from the catalog strategist rather than from
precedent, and this is the same blocker `switchboard` records in its own
manifest section.

**Declaring the file is not the last step.** The committed inventory at
`packages/monetization-go/meter-inventory.json` is generated, and a drift
test fails the build when it is stale:
`packages/monetization-go/meter_inventory_drift_test.go:25-30` reads the
committed file, compares it byte for byte against a fresh
`BuildMeterInventory`, and fails with "committed meter inventory is stale;
run go run ./cmd/meter-inventory". So the sequence is: write the manifest,
then run `go run ./cmd/meter-inventory` from `packages/monetization-go`,
then commit both files.

The file's presence is also the applicability trigger for the
`monetization-conformance` test-genie phase. Declaring nothing is not a way
to pass that phase; the phase also becomes applicable when a scenario is
detected to touch LPBS, so deleting the manifest produces a failing run
rather than a skipped one.

### Seeding `subscription_tier_limits`

**This is the largest single gap between "the meter is declared" and "the
meter does anything", and it is invisible from the manifest.**

`ReserveCredits` enforces a limit only when every one of four conditions
holds (`scenarios/landing-page-business-suite/api/internal/commerce/reservation_service.go:281-294`):

```go
if tier != "" && s.limitsSvc != nil {
    limit, err := s.limitsSvc.GetLimit(ctx, tier, limitKey, nil)
    ...
    if limit != nil && limit.LimitValue >= 0 {
        if effectiveUsage+amount > limit.LimitValue {
            return "", fmt.Errorf("%w: ...", s.insufficientCredits, ...)
        }
    }
}
```

A non-empty tier, a wired limits service, a row that exists, and a limit
value that is not negative. **With no row for `compute_minutes`,
`GetLimit` returns nil, the whole block is skipped, every reservation
succeeds, and the meter is decorative.** It will still write reservation
rows, still settle, still produce usage records, and still never refuse
anything. That failure is silent and it looks exactly like a working
meter.

Note also that an empty `tier` string short-circuits the check before the
row is ever consulted. A caller that forgets to resolve the tier gets
unbounded reservations even when the rows are correctly seeded, which is a
second reason the enforcement-boundary test behind `COMPUTEM-P0-006` has to
assert refusal rather than assert that a call was made.

**The rows to seed.** Follow the pattern at
`scenarios/landing-page-business-suite/api/startup_seed.go:139-143`, which
is the five `ai_credits` rows:

```go
{"free",     "cost_based", "ai_credits",          0, stringPointer(bundleKey)},
{"solo",     "cost_based", "ai_credits",  500000000, stringPointer(bundleKey)},
{"pro",      "cost_based", "ai_credits", 2000000000, stringPointer(bundleKey)},
{"studio",   "cost_based", "ai_credits",10000000000, stringPointer(bundleKey)},
{"business", "cost_based", "ai_credits",         -1, stringPointer(bundleKey)},
```

The five values in the first column are the `plan_tier` axis:
`free`, `solo`, `pro`, `studio`, `business`, per the vocabulary table at
`docs/monetization/strategy/TIERS.md:48`. The column they are written into
is named `tier_id`, so the axis name and the column name differ; do not
introduce a third spelling. `-1` is the **unlimited** convention, and it is
load-bearing rather than cosmetic: the enforcement condition is
`limit.LimitValue >= 0`, so a `-1` row skips the comparison entirely while
still existing. A missing row and a `-1` row therefore behave identically
at enforcement time but mean opposite things to a reader, which is a good
reason to write all five rows explicitly including `free` at `0`.

`app_bundle_key` is `"business_suite"`, passed as
`stringPointer(bundleKey)` where `bundleKey` is the `const bundleKey =
"business_suite"` declared at the top of `seedTierLimitsDefaults`.

**A compute row must be `cost_based`, not `count_based`.** The two values
in use are `cost_based` (for `ai_credits`) and `count_based` (for
`workflow_executions` and `voice_minutes`). Compute is awkward precisely
because a minute is not a unit of cost: on Hetzner's own published prices a
month of CX23 is €5.49 and a month of CCX63 is €853.49, a factor of 155.
A `count_based` ceiling on minutes would let a `solo` subscriber consume
155 times more money on one instance type than another while reporting the
same usage, which makes the ceiling a spend bound in name only.
`cost_based` bounds the thing that actually matters.

That choice creates a naming tension worth stating rather than hiding. The
key is `compute_minutes`, a time-shaped name inherited from a catalog node
this scenario did not create, while the enforceable quantity is cost. Two
resolutions exist:

1. Keep the key and define a compute minute as a **normalized cost unit**:
   one minute of a declared reference instance size, with every other size
   converted by its price ratio. The meter then reports minutes and bounds
   money, and the conversion table is provider data the adapter already has
   to declare.
2. Ask the catalog strategist to rename the stream to something
   cost-shaped. Cleaner, and it is operator catalog work with a rename
   cost across two existing edges.

Option 1 is the recommendation because it needs no catalog mutation, but
option 2 is the better name and should be raised when the node is created.

**Whatever unit is chosen must be documented next to the seed rows.** The
unit behind `ai_credits` value `500000000` is written down nowhere in the
repository, and repeating that omission for compute would make the numbers
unreviewable by the operator who has to approve them.

**Two mechanical facts that make this cheaper than it looks.** The seeder
is called unconditionally on every startup from `startup_seed.go:62`, and
the insert is `ON CONFLICT (tier_id, limit_type, limit_key, app_bundle_key)
DO NOTHING` (`api/seed_statements.go:35`). So adding rows to the slice does
take effect on an existing database, and re-running is a no-op for rows
that already exist. There is a `seedTierLimitCountSQL` constant at
`seed_statements.go:34` that nothing calls, which might suggest the seed is
count-gated; it is not.

### The class, and where enforcement runs

Hosted compute is named verbatim as a Class A, cost-bearing example in
`path:docs/concepts/PAID_FEATURES.md`, in the same list as LLM tokens and
TTS/STT seconds. Vrooli pays the provider in real money, per hour, whether
or not the client is honest. Therefore:

- Enforcement runs **server-side, before the machine boots**. Credit is
  reserved before any provider API call, which is `OT-P0-006`.
- The tier is resolved server-side and never read from the request. A
  client asserting a higher tier is still refused, which the intended
  `enforcement_boundary_test.go` behind `COMPUTEM-P0-006` asserts
  directly.
- A refused reservation short-circuits with **no provider call at all**.
  There is no optimistic local path and no outbox, because a Class B
  shape here would mean a machine that boots before anybody checked
  whether it could be paid for.

### The APIs to call, and the one never to call

All in
`scenarios/landing-page-business-suite/api/internal/commerce/reservation_service.go`.

| Step | Call | Line | Note |
|---|---|---|---|
| Reserve before provisioning | `ReserveCredits` | `:200` | Returns a reservation ID. Enforces the tier limit only under the four conditions above. |
| Settle measured usage on teardown | `FinalizeReservation` | `:338` | Records actual usage and closes the reservation atomically. |
| Release when provisioning fails | `ReleaseReservation` | `:460` | Marks the reservation released without recording usage. A no-op, logged as `reservation_release_noop`, if the reservation was already finalized, released, expired or absent. |
| Settle, multi-tenant | `FinalizeReservationForUser` | `:451` | **Prefer this.** Calls `verifyOwner` first, then delegates to `FinalizeReservation`. |
| Release, multi-tenant | `ReleaseReservationForUser` | `:504` | **Prefer this.** Same ownership check before delegating. |
| Never | `ReserveAndCharge` | `:72` | See below. |

`FinalizeReservation` and `ReleaseReservation` take a reservation ID and
nothing else, so **any caller holding any reservation ID can settle or
release any other tenant's reservation**. The `...ForUser` variants exist
precisely to close that, by calling `verifyOwner` (`:511`) before doing
anything. In a single-operator deployment the difference is invisible; in
the multi-tenant deployment this section exists to describe, the unchecked
variants are a cross-tenant defect waiting to happen. Use the checked ones
from the first line of code, because retrofitting them later means
auditing every call site.

**Never `ReserveAndCharge`.** It is the convenience helper, and every one
of its conveniences is a property this scenario needs: it creates no
reservation, so there is nothing to release; it takes no idempotency key,
so a retried provision double-charges; it has no release path, so a failed
provider call leaves the charge standing; and it does not refund on
provider failure. It is correct for a short synchronous call that either
works or does not. A machine that boots is neither.

### The lifecycle

Reserve, provision, re-reserve on a heartbeat, settle measured usage on
teardown, and release the reservation if provisioning fails rather than
burning it. Usage is metered from transitions this scenario caused, never
from an observer loop that watches what is running, because a dead
observer stops billing while the provider keeps charging.

### Two money seams, and their ordering

There are two independent spend controls and they are not the same thing.
Conflating them is the likeliest design error in this area.

| Seam | Direction | Owner | Question it answers | Behaviour when unavailable |
|---|---|---|---|---|
| `landing-page-business-suite` | **Inbound.** Charges a customer. | The wallet and the tier limits | "Has this customer paid, and do they have headroom?" | **Fail closed.** Provisioning refuses. |
| `treasury` | **Outbound.** Bounds an agent's spend. | The signed mandate | "Is this agent authorized to commit Vrooli's money at all?" | Agent-initiated provisioning refuses; operator-initiated continues. |

They apply on different paths. Treasury applies only where one of Vrooli's
own agents requests capacity. It does not apply when a human operator asks,
and it does not apply when a paying customer asks, because a customer
spending their own credit is not an agent spending Vrooli's money. LPBS
applies wherever Vrooli is the one paying the provider bill, which is the
paid path and only the paid path.

**The ordering is mandate first, reservation second**, on the agent path
where both apply. Three reasons:

1. The treasury check is local and cheap; the reservation writes a durable
   row and starts a ten-minute clock. Doing the cheap refusal first wastes
   nothing.
2. A reservation created and then abandoned because the mandate refused has
   to be released, and a release that is dropped holds credit against the
   customer until the reservation expires. Reserving second means there is
   nothing to unwind.
3. The mandate answers a question about authority, and authority questions
   belong before resource questions. An agent with no mandate should never
   have caused a row to be written in the customer's billing period.

**When the two disagree**, the refusal wins and the reason is reported
distinctly, never collapsed into one error:

- **Mandate refuses, wallet would have allowed.** No reservation is
  created because none was attempted. The caller is told the agent is not
  authorized, not that the customer is out of credit. Telling a paying
  customer they are out of credit when the real cause is an internal
  authorization limit is a support incident manufactured out of a bad
  error message.
- **Mandate allows, wallet refuses.** `ReserveCredits` returns the
  `insufficientCredits` sentinel error. The mandate is not consumed,
  because nothing was spent. The caller is told the customer is out of
  credit.
- **Both refuse.** Report the authority failure, because it is the earlier
  one and the one the agent can do nothing about.

The one ordering that must not be built is provision-then-check-either.
Both seams exist to prevent a machine from booting, and a check that runs
after the boot prevents nothing.

### The dependency posture that follows

`landing-page-business-suite` is the one required dependency that **fails
closed**. If it is unavailable, provisioning refuses. A machine that boots
unmetered is cost that grows hourly and cannot be recovered afterwards, so
refusing is the cheaper failure. Everything else degrades: `vrooli-bridge`
being unavailable still lets the instance be created, metered and expired,
with enrollment queued and the instance flagged as un-enrolled.

### Prerequisites, all upstream, none of them integration polish

| Prerequisite | File and line | Why it blocks the sale |
|---|---|---|
| A `compute-manager` deliverable node and a rewired `unlocks` edge in offer desk, and an offer promoted to `active` by an operator | offer desk catalog, no in-repo file | Until then the catalog attributes the stream to two neighbours and this scenario has no node at all. An agent must never promote; see [`GO-TO-MARKET.md`](GO-TO-MARKET.md). |
| Tier limit rows seeded for `compute_minutes` | `landing-page-business-suite/api/startup_seed.go:129-163` | Without them the meter never refuses anything. See [Seeding `subscription_tier_limits`](#seeding-subscription_tier_limits). |
| The reservation window is longer than an hour of compute, or heartbeat re-reservation covers it | `reservation_service.go:296-298` | `expiresAt := time.Now().Add(10 * time.Minute)` is hard-coded, and ten minutes is shorter than the smallest unit of compute anybody buys. A reservation that expires mid-instance means unbilled running cost. |
| Refunds work for app-scoped charges | `reservation_service.go:621` (sqlite) and `:635` (postgres) | Both `AdjustUsage` queries end `AND app_bundle_key IS NULL`, while the charge path writes rows that carry one. A refund against an app-scoped charge updates zero rows and returns nil. Silent is the problem, not the refund. |
| **The prepaid wallet is never drained** | `credit_wallet_service.go:132` and `:138` | `ConsumeCredits` and `ConsumeCreditsIdempotent` have **zero production callers**. The only references in the repository are the interface declaration, the two definitions, and tests. A customer who tops up buys a balance that nothing decrements, so wallet-funded compute would be free forever and the operator would find out from the provider invoice. |
| The reservation path is used instead of the convenience charge helper | `reservation_service.go:72` | `ReserveAndCharge` creates no reservation, takes no idempotency key, has no release path and does not refund on provider failure. |
| Out-of-credit is distinguishable from a server error at the client | `ai-gateway/api/internal/providers/metered.go:150-159` | See below. |
| `vrooli-bridge` publishes its onboarding public key | upstream, no endpoint exists | Not a monetization prerequisite, but it is the single new wire contract unattended enrollment needs, and unattended enrollment is what makes sold capacity worth more than a provider console. |

**On the 402 problem, precisely.** The reference metered client does
decode the response body: `json.NewDecoder(response.Body).Decode(&result)`
at `metered.go:150`, before any status check. What it discards is the
**error text**. The status check at `:157-159` returns
`fmt.Errorf("metered inference returned HTTP %d", response.StatusCode)`,
which throws away whatever the server said about why, and `:158` calls
`c.failed()` so a 402 advances the circuit breaker exactly like a 500
does. Combined with `:131`, where an open breaker refuses before the
request is even built, a customer who runs out of credit takes the
provider offline for the cooldown and is shown an outage.

The better pattern already exists in the fleet:
`scenarios/switchboard/api/internal/metering/lpbs.go:68-70` reads up to
4096 bytes of the body on any non-2xx and includes it in the error, so the
caller can distinguish a refusal from a fault. Copy that shape. Whatever
this scenario builds must surface the `insufficientCredits` condition as a
refusal the operator can act on, and must not count it as a provider
fault.

## Pricing Hypothesis

- **Model:** pass-through of the provider rate plus a margin, billed per
  `compute_minutes` with a **minimum billable unit**. Not a flat
  subscription for capacity, because the underlying cost is genuinely
  variable and hiding that inside a flat fee means either overcharging
  light users or absorbing an unbounded bill.
- **Willingness-to-pay evidence: none.** There is no evidence anyone will
  pay a margin for capacity when the free bring-your-own-key path exists
  and is deliberately not degraded. That is the central open question and
  it should not be glossed.
- **Comparable products:** hosted platform-as-a-service vendors price at a
  large multiple of raw provider cost and justify it with the operational
  layer. The named comparison set is **Fly.io, Render, Railway, Heroku,
  Northflank and DigitalOcean App Platform**. No price comparison against
  any of them has been retrieved, so the multiple is an impression rather
  than a measurement, and no asset may quote one until it is. The honest
  qualitative comparison is narrower than the category anyway, because the
  operational layer being sold here is trust plus mortality plus cost
  visibility, not a full application platform. Every one of those six also
  sells a deployment experience this scenario deliberately does not own.

### The minimum billable unit is a fleet property, not an adapter property

The minimum billable unit must be **at least the worst rounding behaviour
across every provider the router may select**, not the rounding behaviour
of whichever provider happens to serve a given request.

`OT-P1-004` promises that "a second provider changes no caller". The
moment that lands, a caller does not know which provider will serve it, so
a per-provider minimum billable unit would mean the price of the same
request depends on a routing decision the customer cannot see. That is
either an unpriceable product or an unhedged loss, depending on which way
the routing goes.

Today the worst case is Hetzner, verbatim from its billing FAQ: "We always
round up the hourly usage of a server. If you create a server just for a
few minutes, we will still bill you for one whole hour." The full rule is
`min(monthly_cap, ceil(hours) * hourly)`, and **the cap is per server, so
it gives no protection against churn**: fifty ten-minute servers cost fifty
provider hours and no cap applies to any of them.

DigitalOcean, the intended second adapter, bills per second with "a
minimum charge of 60 seconds or $0.01, whichever is higher" since
1 January 2026. That is a dual floor, and on the cheapest Droplets the
money floor binds first. It is much better than an hour, which is exactly
why it cannot be allowed to set the fleet-wide minimum: adding a
finer-grained provider must not silently reduce a minimum sized for a
coarser one still in the pool. **The minimum billable unit stays at 60
minutes until Hetzner leaves the routing pool**, and the rule to encode is
`max` over declared provider rounding, evaluated across the pool rather
than per request. Each adapter already has to declare its provider's
rounding behaviour as data, so the maximum is computable rather than
hand-maintained.

Citations for both providers, with retrieval dates, are in
[`../reference/provider-survey.md`](../reference/provider-survey.md).

### A worked margin example, with sourced numbers

The interesting risk is not the margin, it is the supplier moving under a
published price. Hetzner's June 2026 increase makes that concrete, using
only Hetzner's own published prices.

A CCX23 (cloud dedicated vCPU, Germany) cost **€31.49 per month** before
15 June 2026 and **€85.99 per month** after, a rise of 173.1 percent. Take
a price sheet written in May 2026 with a 30 percent margin:

| | Before 15 June 2026 | After |
|---|---|---|
| Provider cost | €31.49 | €85.99 |
| Our price, 30% margin over the May cost | €40.94 | €40.94 |
| Gross margin per instance-month | **+€9.45** | **−€45.05** |

A healthy 30 percent margin became a **loss of 110 percent of revenue**
overnight, on a price that was never republished. The increase was
announced on 27 May 2026 and took effect on 15 June at 08:00 CEST, which
is nineteen days of notice, delivered while nobody was watching for it.

Three details make this worse than it first reads, and all three are
sourced:

- **The increase applies to "new orders and rescales of existing
  servers".** Existing contracts keep their pricing, so a long-lived fleet
  sees nothing and a scenario whose entire product is short-lived
  instances sees all of it. This scenario is on the wrong side of that
  line by design. A resize codepath is also a repricing event.
- **"Shared rose by about a third" is false.** CX and CAX, both shared
  lines, rose 30 to 38 percent in the EU. CPX is also a shared vCPU line
  and it rose 143.9 to 175.4 percent in the EU and up to 209.0 percent in
  the USA. Sizing a buffer from the CX figure would under-provision by a
  factor of six on a line an automated system would plausibly pick for
  being cheap per vCPU.
- **Hetzner published no percentage at all.** Every figure above is
  arithmetic on its published per-plan prices. Cite it as derived.

The conclusion is not "add a bigger buffer". It is that **any published
price must carry a supplier-change clause and a revision mechanism**, and
that a multi-year fixed price would be a bet against a supplier that has
already moved 173 percent once with nineteen days of notice.

### Rounding, stated rate-independently

For short workloads the rounding, not the rate, is the margin. A
ten-minute instance costs one provider hour. Billing the customer ten
minutes recovers one sixth of what was paid, so **break-even on a
ten-minute job requires charging six times the provider's per-minute
equivalent** no matter what the rate is. This holds for every rate, which
is why the mitigation cannot be a pricing tweak.

Two mitigations exist and both are hypotheses. A minimum billable unit
passes the rounding through honestly, so the customer pays for the hour
that was actually bought. Warm pooling (`OT-P2-003`) amortizes one
provider hour across several short jobs, which is the only way to make the
rounding disappear rather than be passed on. Warm pooling is worth
building only once measured churn shows rounding dominating the bill, and
that measurement cannot be taken until real instances exist.

### The credit unit and `limit_type` decision

Recorded in [Seeding `subscription_tier_limits`](#seeding-subscription_tier_limits)
because that is where it takes effect. In summary: `cost_based`, because a
minute is not a unit of cost when instance types differ by a factor of 155
in price; and a compute minute defined as a normalized cost unit against a
declared reference instance size, with the conversion table coming from the
per-adapter billing facts each provider adapter already declares.

### The reserve estimate and the safety margin

An open policy decision with two defensible answers, and it must be
decided before code rather than discovered by it.

**Option A: reserve the full requested lifetime up front.** The customer's
headroom is checked against what they actually asked for, so a request
that cannot be paid for in full is refused before the machine boots, which
is the whole point of Class A enforcement. The cost is that credit is held
for the entire lifetime, so a subscriber with a modest balance can request
one long instance and be unable to request anything else, even though the
first instance might be destroyed in ten minutes.

**Option B: reserve one heartbeat window and re-reserve.** Credit is held
only for what has nearly been consumed, which is far friendlier to a
balance. The cost is that a long instance can be started by a customer who
could never have afforded it, and the refusal arrives mid-life, which is
the failure mode discussed under [out-of-credit
mid-instance](#the-refund-and-failure-matrix) and the worst one in this
document.

**Recommendation: A for the headroom check, B for the hold.** Check the
full requested lifetime against the limit at request time and refuse if it
does not fit, then hold only a heartbeat window plus a safety margin. This
gets the refusal at the only moment where refusing is free, without freezing
credit for hours. It requires the limits check and the reservation amount to
be separable, which `ReserveCredits` does not currently offer, so it is a
prerequisite rather than a configuration.

**The safety margin must exceed the reservation window**, because the
window is ten minutes and hard-coded (`reservation_service.go:296-298`).
A heartbeat that re-reserves at the ten-minute mark has zero margin for a
slow network, a restart, or a `landing-page-business-suite` that is briefly
unavailable. Re-reserve at a fraction of the window, and record each
renewal as a new row rather than mutating the previous one, so the history
survives.

### Cost drivers

- The provider hourly rate, which is not stable. See the worked example.
- **Hourly rounding on short-lived instances**, which is the dominant
  driver for this workload shape and is not mitigated by the monthly cap.
- **Stopped instances that still bill.** A stopped instance continues to
  bill at the full rate on most providers, which is why `OT-P0-007` makes
  "destroy is the only stop" a must-ship target and asserts structurally
  that no `Stop` method exists anywhere in the scenario. This is the
  clearest case in the document of a cost driver becoming a product
  decision: a pause button would be a control that costs full price and
  delivers nothing, and the only way to be sure nobody ships one is for the
  provider interface not to have the method. *The specific claim that this
  holds on five of the seven surveyed providers is currently unsourced;
  see the unsourced-claims table in
  [`../reference/provider-survey.md`](../reference/provider-survey.md).*
- Outbound traffic beyond the included allowance. Hetzner bills outbound
  only, which is why it was chosen first, and which is why a provider that
  also bills inbound would turn a fixed cost into one an attacker
  controls. *Also currently unsourced.*
- Zero local resource cost, since this scenario requires no resources and
  its state is SQLite.

## The Refund And Failure Matrix

Every row is a case where money and reality can disagree. None is
implemented, and the point of writing them down before any code exists is
that several have no good answer and the design has to choose which bad
one it accepts.

| Case | What happens today if nothing is designed | Intended handling |
|---|---|---|
| **Provisioning fails after the reserve** | The reservation sits pending for ten minutes and then expires. The customer sees held credit for a failure that was ours. | Call `ReleaseReservationForUser` on every failure path, including panics and context cancellation. Release is idempotent and logs `reservation_release_noop` when there was nothing to release, so calling it defensively is free. |
| **Partial failure: the instance was created but the response was lost** | Worst case in the whole scenario. The provider is billing for a machine no record mentions, and the reservation may or may not have been settled. | The intent record written before the provider call (`OT-P0-002`) is what makes this recoverable. Reconciliation matches the unaccounted instance back to its intent and its reservation. Never release a reservation on a timeout, because a timeout is not a failure; leave it pending and let reconciliation decide. |
| **An orphan is found that was already refunded** | Double credit. The customer is refunded once by the failure path and once by the reconciler. | Refunds key on the reservation ID and are idempotent at that key. The reconciler proposes; it never refunds directly. This is the same "report, never resolve" rule that governs orphan destruction. |
| **The provider bills more than we metered** | The difference is absorbed silently and shows up only as a margin that does not match the model. | Daily reconciliation (`OT-P1-003`) raises divergence as an alarm with the instance identified. Divergence is never auto-corrected into a charge, because the provider statement lags by an amount nobody can bound and a correction applied to lagging data corrects the wrong thing. |
| **Double settle: `FinalizeReservation` called twice** | Depends entirely on the upstream transaction, which this scenario does not own. | Settlement must be idempotent at the reservation ID, and the second call must be a logged no-op rather than an error, because a retried teardown is normal. Verify the upstream behaviour before relying on it; this has not been tested. |
| **A refund is issued against an app-scoped charge** | **It silently does nothing.** `AdjustUsage` filters `AND app_bundle_key IS NULL` at `reservation_service.go:621` and `:635`, so zero rows update and nil is returned. | Do not build a refund path on `AdjustUsage` until the filter is fixed upstream. Until then, settle the real measured quantity at teardown rather than over-reserving and correcting downwards. A refund path built on this today would appear to work in every test that does not check row counts. |
| **The customer runs out of credit mid-instance** | See below. This is the one that is genuinely bad. | See below. |

### Out of credit mid-instance

**Because destroy is the only stop, running out of credit mid-instance is
a customer-visible data-loss event.** There is no pause. The instance
either keeps running at Vrooli's expense or it is destroyed with whatever
is on its disk.

This is not a corner case introduced by an implementation shortcut. It is
the direct consequence of `OT-P0-007`, which is a good decision made for
good reasons: a stopped instance still bills, so a pause button costs full
price and delivers nothing. But the decision's cost lands here, and it
lands on the customer.

The design must therefore make it nearly impossible to arrive at:

- **Refuse early rather than stop late.** Check the full requested
  lifetime against the customer's headroom at request time, per the
  reserve-estimate recommendation above. A request that cannot be paid for
  in full never boots, and a refusal at request time costs nobody
  anything.
- **Warn before the boundary, not at it.** The heartbeat already knows the
  remaining balance and the remaining lifetime. It can say "this instance
  will be destroyed in N minutes for non-payment" while N is still large
  enough to act on. A warning that arrives with the destruction is not a
  warning.
- **Never destroy silently.** The destruction must produce an operator
  alarm and a customer-visible record naming credit exhaustion as the
  cause, distinct from expiry and distinct from an operator-requested
  destroy.
- **Prefer absorbing a bounded overrun to destroying.** A grace allowance,
  bounded in both money and time and applied once per billing period, is
  cheaper than the support cost and reputational cost of destroying a
  customer's running workload. It must be bounded in both dimensions,
  because a grace bounded only in time is an unbounded bill on a large
  instance.
- **Say so in the product surface, in advance.** A customer who was told
  at purchase that capacity is destroyed rather than paused when credit
  runs out has been sold a different product from one who discovers it.
  This belongs in the offer text, not only here.

**No public asset may describe the paid path without stating this
behaviour.** It is the sharpest difference between this product and every
platform-as-a-service comparator, all of which suspend rather than delete.

## Abuse, Acceptable Use, And The Shared-Account Risk

**One abusive customer can get the shared provider account terminated,
taking every customer's instance with it.** This is the largest
non-financial risk in the paid path and it has no technical mitigation on
its own.

Hetzner's Terms and Conditions §8.3 prohibits spam and sender-identity
falsification, and then states verbatim: "The operation of applications for
mining cryptocurrencies remains prohibited. These include, but are not
limited to, mining, farming and plotting of cryptocurrencies." The remedy
it reserves is: "We are entitled to lock the Customer's access to their
Hetzner services or account in the event of non-compliance." §5.2 reserves
the same remedy for (d)DoS and open relays, and does so "without prior
notice".

The account is the unit of enforcement, not the server. And §7.1, the
clause that makes the whole model possible, is explicit that "the Customer
nevertheless remains the sole contractual partner" and "continues to be
solely and fully liable". Permission to let third parties use the capacity
comes bundled with full liability for what they do with it, and §9.2 adds
liability for all direct and indirect damages plus legal defence costs.

Crypto mining is the specific named case and it is also the most likely
one, because mining is the highest-value use of anonymous rented compute
and short-lived instances are exactly what a miner wants.

Four things follow, and none of them is optional before a first paying
customer:

1. **A published acceptable-use policy**, written to be at least as strict
   as the strictest provider in the routing pool, since a customer cannot
   be expected to read seven providers' terms and a router may send them
   anywhere. It must name crypto mining, (d)DoS sourcing and open relays
   explicitly rather than gesturing at "illegal use".
2. **An abuse runbook** that exists before it is needed: how a report
   arrives, who can destroy an instance without waiting for consensus, what
   the customer is told, and what is preserved for the provider. The mean
   time to first response is the number the provider will judge, and a
   runbook written during an incident is written too late.
3. **Detection that does not depend on the customer being honest.** Sustained
   full CPU on a small instance with negligible network is the mining
   signature, and it is cheap to look for. This is a hypothesis about a
   detector, not a detector.
4. **An accepted answer to "what happens when the account is locked".**
   Under a single shared account the answer is that every customer's
   instance stops at once, including the honest ones, and no code in this
   scenario can prevent it.

`../internal/SECURITY.md` records the same risk from the security side and
should be read alongside this.

## Tenancy Model

The choice is unresolved and it determines the blast radius of everything
in the previous section.

| Model | Blast radius of one abuse incident | Operational cost | Margin impact |
|---|---|---|---|
| **One shared Vrooli provider account** | **Total.** Account locked, every customer's instance stops, including customers with no involvement. | Lowest. One credential, one billing relationship, one reconciliation. | Best. Volume pricing applies across the whole fleet. |
| **Sub-account or project per customer** | Bounded to the offending customer, if the provider's sub-account boundary is genuinely an enforcement boundary rather than an organisational one. | Highest. Credential lifecycle per customer, N reconciliations, N quota ceilings, and a provisioning flow that can fail because a sub-account could not be created. | Worse. Volume pricing may not aggregate across sub-accounts. |
| **Customer-supplied credential** | None. But this is the free bring-your-own-key path, and a customer who has a provider credential is not the paid customer. | Lowest. | Not applicable; there is no margin because there is no cost. |

The paid audience is defined as "a subscriber who wants managed capacity
without holding a cloud account at all", which rules out the third row for
that audience by construction.

**The shared account is the default and it must be an explicit, recorded
operator acceptance rather than a default nobody chose.** The correct
question to put to the operator is not "shared or per-customer" but "is a
total outage for every paying customer, caused by one customer we cannot
fully vet, an acceptable risk at the revenue this is expected to produce".
At low revenue the answer is plausibly yes, and it stops being yes at some
volume nobody can currently name.

**A prerequisite nobody has checked:** whether the provider's sub-account
boundary is an *enforcement* boundary for terms violations, or only a
billing and organisational convenience. If a provider locks the parent
account for a sub-account's mining, the middle row buys nothing and costs a
great deal. This question is not answered by the terms survey and must be
answered before the middle row is chosen.

## Churn Policy

**An instance destroyed before the minimum billable unit elapses is
non-refundable, and this must be stated at purchase.**

The reason is arithmetic rather than commercial. The provider has already
charged a full hour by the time the instance is ten minutes old; refunding
the customer for the fifty minutes they did not use means paying for them
out of margin. A refund policy that is generous about early destruction
converts every impatient customer into a direct loss, and an automated
system creates and destroys instances far more often than a human does.

Three consequences:

- The minimum billable unit is charged **at creation**, not at teardown.
  Settling less than the minimum at teardown reintroduces exactly the loss
  the minimum exists to prevent.
- Extending an instance past the minimum bills the additional time in the
  same units, so extension is not a way to buy cheaper minutes.
- The policy is stated in the offer text before purchase. A customer who
  destroys an instance after two minutes and is billed an hour has not been
  treated unfairly if they were told; they have been if they were not.

The one exception is a destruction caused by us: a provisioning failure, a
reconciler error, or an instance destroyed for credit exhaustion without
the warning described above. Those are refunded in full, and they are
distinguishable from customer-initiated destruction because the intent
record says who asked.

## Commercial Metrics

The measurements that would tell an operator whether the paid path is
working. None can be taken yet, and the first three are also the inputs to
any defensible price.

| Metric | Definition | Why it matters | Source once built |
|---|---|---|---|
| **Gross margin per compute-minute** | Revenue recognised per compute-minute minus the provider cost attributed to that minute, over a full billing period. | The single number the business rests on. A positive margin computed against a meter that disagrees with the supplier is not a margin. | Metered usage joined to the provider statement, per `OT-P1-003`. |
| **Rounding waste** | Provider-billed time minus customer-billed time, as a fraction of provider-billed time. | Decides whether warm pooling (`OT-P2-003`) is worth building or whether a minimum billable unit alone suffices. | The lifecycle's own transition timestamps against the adapter's declared rounding rule. Computable without a provider statement, which makes it the cheapest of these to obtain. |
| **Meter-to-statement divergence** | Absolute difference between metered usage and the provider's own billing, per billing period, per provider. | Gate on any pricing conversation. This is the check that catches a metering bug before a customer does. | `OT-P1-003`. |
| **Release rate** | Reservations released as a fraction of reservations created. | A rising release rate means provisioning is failing more often, and each failure is a customer who asked for capacity and did not get it. A release rate near zero when failures are known to occur means the release path is not being called, which is worse. | Reservation status counts from `landing-page-business-suite`. |
| **Grace-allowance usage** | Instances that entered the credit-exhaustion grace window, and how many were destroyed for non-payment. | Any destruction for non-payment is a product failure by the standard set above. This counts them. | The lifecycle's destruction reason. |
| **Free-path share** | Instances created against an operator's own credential versus a Vrooli-owned account. | The honest denominator for the whole hypothesis. If it stays near 100 percent after the paid path ships, the paid path has no customer. | The instance record's credential source. |

## Validation Plan

- **Demand signal needed:** an operator who has the free path working and
  still asks for Vrooli-provisioned capacity. Anything short of that is
  interest in the capability rather than in paying for it.
- **The question that matters most:** whether not holding a cloud account
  is worth a margin. If a buyer is willing to open a Hetzner account, the
  free path is strictly better for them and the paid path has no
  customer. The paid path exists for the buyer who will not, or cannot,
  hold provider credentials at all.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** set against the project-level monetization
  taxonomy, not here.

### Revisit triggers

`path:docs/monetization/catalogs/CATALOG.md` requires a catalog revisit
trigger to be machine-evaluable and warns against phrasings like "revisit
when we feel ready". The triggers below are stated in that form. They are
document-local until a catalog record exists for this scenario, at which
point the operator who creates the node decides which of them the record
carries.

| Trigger | Evaluable as |
|---|---|
| The reserve, provision, settle spine holds against the fake provider | The `COMPUTEM-P0-006` suite passes, including a refused reservation that makes zero provider calls. |
| At least one real instance has completed the full lifecycle | Count of instance records with a `destroyed` transition, a settled reservation, and a non-null enrollment, is `>= 1`. |
| The meter agrees with the supplier | Meter-to-statement divergence is **below 2 percent of the provider-billed total, over a full billing period, on the provider serving the majority of instance-hours**. Two percent is chosen as roughly one hour in fifty, which is the granularity Hetzner's own rounding imposes; it is a starting figure to be replaced by a measured one, not a derived bound. |
| Rounding waste is understood | Rounding waste has been computed over at least 30 completed instances, so the warm-pooling decision has a denominator. |
| Tier limits actually refuse | A reservation exceeding a seeded `compute_minutes` limit returns the `insufficientCredits` sentinel in an integration test, and `undeclared_streams` from `offer-desk offers meters` no longer lists `compute_minutes`. |
| Demand exists | Count of identified buyers who have the free path available, understand it, and still ask to be billed, is `>= 1`. |

Before the first four hold there is no basis for a price, because nobody
knows what a compute minute actually costs to deliver here.

## Current Status

`hypothesis`, and nothing is implemented. As of 2026-09-03 the scenario
contains generated template code only. The free/metered/gated split, the
Class A placement, the manifest shape, the seed spec and the prerequisite
list above are argued from `path:docs/concepts/PAID_FEATURES.md`, the
schema, the upstream source cited inline, the PRD and the design brief.
The prices, the bundle and the decision to sell at all remain operator
canon and are not decided here. No willingness-to-pay evidence exists.

Three claims in this document are sourced only by a retrieved-and-cited
artifact rather than by code, and are listed here so they are re-checkable:
the Hetzner rounding rule, the June 2026 price move, and the absence of any
published billing-latency SLA. All three, with retrieval dates, are in
[`../reference/provider-survey.md`](../reference/provider-survey.md).
Three further claims used elsewhere in this scenario are **not** sourced at
all and are marked as such in that document's unsourced-claims table.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md): orientation workflow
- [`../../PRD.md`](../../PRD.md): product requirements, including `OT-P0-006`, `OT-P0-007`, `OT-P1-004` and `OT-P2-001`
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md): channel, launch motion, the launch-gating checklist and the operator-only promotion rule
- [`../reference/provider-survey.md`](../reference/provider-survey.md): resale clauses, rounding rules, the price move and the billing-latency finding, with retrieval dates
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md): why Hetzner is first, why there is no `Stop`, and the per-service annex method rule
- [`../internal/SECURITY.md`](../internal/SECURITY.md): account termination as a fleet-wide availability failure, and the missing per-tenant isolation
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md): cost-relevant timing budgets, all targets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md): telemetry needed for business validation
- Paid-features contract: `path:docs/concepts/PAID_FEATURES.md`
- Project-level monetization strategy: `path:docs/monetization/README.md`
- Tier definitions and the `plan_tier` vocabulary: `path:docs/monetization/strategy/TIERS.md`
- Catalog lifecycle and revisit-trigger discipline: `path:docs/monetization/catalogs/CATALOG.md`
