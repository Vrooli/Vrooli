# Integrations — Compute Manager

## Purpose Of This Document

This document records what this scenario depends on, why, and what it does when
each dependency is unavailable. It is the prose half of a contract whose machine
half is the `dependencies` block in `.vrooli/service.json`. **The two must agree.**
A dependency named here and absent there is intent that never reached the
lifecycle, and a dependency declared there and unexplained here is a coupling
nobody chose.

> **Status: partially implemented.** Provider, business-suite metering and
> bridge integration seams are implemented behind interfaces. Live provider
> proof and some upstream operational contracts remain open.

## Dependency Inventory

| Dependency | Kind | Required | Startup policy | On failure |
|---|---|---|---|---|
| `landing-page-business-suite` | scenario | yes | `must_start` | **Fail closed.** Provisioning refuses |
| `vrooli-bridge` | scenario | yes | `try_start` | Degrade **while the onboarding key is cached**; refuse to create when it is not |
| `treasury` | scenario | no | `ignore` | Agent-initiated provisioning refuses; operator path continues |
| `offer-desk` | scenario | no | `ignore` | No runtime effect |
| Cloud provider HTTP API | third party | yes, per adapter | n/a | Provisioning fails; existing instances are unaffected |

## Vrooli Resources

**None.**

This scenario runs no local service. Its state is a scenario-owned SQLite
database and its only outbound reach is HTTPS. There is no queue, no cache, no
object store and no daemon to supervise.

This is worth stating explicitly because a scenario that provisions
infrastructure looks like it ought to need infrastructure. It does not. The
machines it creates live at a provider, not here.

## Scenario Dependencies

### landing-page-business-suite — the one that refuses

The credit reservation and settlement surface. Hosted compute is cost-bearing,
so a reservation must be obtained server-side **before** a provider is called,
and settled when the instance is destroyed.

This is the single dependency that fails closed rather than degrading, and the
reasoning is worth keeping: a machine that boots without a reservation is cost
that grows by the hour and cannot be recovered after the fact. There is no
compensating action later. Refusing to provision is recoverable; provisioning
unmetered is not.

The surface to call is `POST /api/v1/usage/reservations`, then
`POST /api/v1/usage/reservations/{id}/finalize` or `.../release`. In Go those
are `ReserveCredits`, `FinalizeReservation` and `ReleaseReservation`, and the
ownership-checked `FinalizeReservationForUser` and `ReleaseReservationForUser`
are the right ones once capacity is sold to more than one buyer. What this
scenario calls settling maps to the upstream verb finalize.

Four known upstream defects are prerequisites rather than integration work, and
this scenario must not paper over them:

- The reservation window is hard-coded to ten minutes, which is shorter than an
  hour of compute. Until it is parameterised, a heartbeat re-reserves.
- Refunds silently do nothing for app-scoped charges, because the adjustment
  query filters to rows with no application key, and every metered scenario
  writes usage under a bundle key.
- The convenience helper `ReserveAndCharge` creates no reservation row, takes no
  idempotency key and has no release path, and does not refund when the work
  fails. **Use the reservation path, never that helper.**
- The prepaid wallet never drains: the credit-consumption functions have no
  production callers, so a customer top-up buys a balance nothing decrements.
  This does not block provisioning, but it does block a top-up product.

### vrooli-bridge — the one that degrades

Node identity, pairing, first touch and dispatch. This scenario creates a bridge
Machine record with the instance address as a locator, then starts bridge
onboarding.

It degrades rather than refuses because blocking capacity on the trust plane
would make capacity unavailable for a reason unrelated to capacity. An instance
whose enrollment is queued is still created, still metered, still expiring on
schedule, and visibly flagged as not yet enrolled.

**This scenario contains no SSH implementation.** First touch is bridge's, and
it works without a password because the instance's first-boot configuration
already carries bridge's onboarding public key.

Bridge now publishes its onboarding public key through an owner-gated wire
contract. The remaining gap for unattended enrollment (`OT-P0-005`) is
provider-live proof that a real host reaches the online state.

### treasury — conditional

Bounds what an agent may spend. It applies only on the path where Vrooli's own
agents request capacity, and not at all when a human operator or a paying
customer does.

When it is unavailable, agent-initiated provisioning refuses and operator-
initiated provisioning continues. This asymmetry is deliberate: an unverifiable
agent should not be able to spend, but an operator at a keyboard has already
demonstrated the authority the mandate exists to check.

### offer-desk — catalog only

Holds the sellable definition of provisioned capacity. A stream node named
`compute_minutes` already exists in the live catalog at status `idea`, which
means the intent to sell compute is recorded but nothing has been promoted.
Its two edges come from elsewhere: `vrooli-bridge` unlocks it and `scenario-to-cloud`
enables it. **This scenario has no node in offer-desk at all.**

That matters for what clearing the gap actually takes. The gap check only scans
`unlocks` edges, so the live `deliverable_meter_gap` is attributed to
`vrooli-bridge`, and declaring a meter manifest here clears the separate
`undeclared_stream` diagnostic only. Closing the gap needs a compute-manager
deliverable node and a rewired `unlocks` edge, which is operator work in the
catalog rather than a file in this repository.

Nothing at runtime reads offer-desk. Its absence affects publishing an offer,
never provisioning a machine.

## Third-Party Services

One cloud provider HTTP API per adapter. The adapter surface is four methods, so
the coupling is small by design and a second provider changes no caller.

**Hetzner Cloud is the first adapter**, chosen for reasons that are contractual
before they are technical:

1. Its standard terms permit granting third parties use rights, with the account
   holder remaining the sole contracting party. Only three of seven providers
   surveyed permit reselling at all. Amazon, Scaleway for virtual machines, Fly
   and Vultr without written permission all forbid it.
2. It bills outbound traffic only. Amazon's small-instance product counts inbound
   traffic against the same allowance, which for a publicly reachable machine
   turns a fixed cost into one an attacker controls.
3. Inbound UDP on arbitrary ports is unencumbered.
4. Its included traffic allowance is roughly twenty times DigitalOcean's.

Costs to price in: it raised prices in June 2026, shared instances by about a
third and dedicated instances by roughly 120 to 175 percent, so it is inexpensive but not
price-stable. It rounds a partial hour up to a full hour, which makes short-lived
instances a margin problem. Its terms also prohibit cryptocurrency mining, which
is a real abuse exposure for anyone reselling general-purpose compute.

DigitalOcean is the intended second adapter, for per-second billing and
geographic diversification. One constraint travels with that choice and should
be recorded before it surprises anyone: DigitalOcean requires every end customer
to individually accept DigitalOcean's own terms, so it can carry Vrooli's own
capacity and capacity for a customer who holds their own account, but it cannot
carry a white-labelled sale to a subscriber who holds no cloud account at all.
That audience is servable on Hetzner only.

A method note worth carrying forward: **a provider's general terms are not the
whole contract.** Scaleway's reselling prohibition exists only in a per-service
annex that explicitly overrides the general terms on conflict. Check the
service-specific conditions, not just the headline agreement.

Credentials for every provider resolve through the credential authority at call
time. No provider token is written to this scenario's database, its process
environment, or a command line. See
[../reference/configuration.md](../reference/configuration.md).

## Future Consumers

Nothing below is built. These are the consumers the contract is shaped for, and
they are recorded here so the adapter surface is not narrowed in a way that
forecloses them.

### Ephemeral capacity for validation

Tracked as `OT-P2-002`. The cross-operating-system gate in `vrooli-bridge`
selects one eligible node per target operating system and fails the gate when a
target has no eligible node. Today that means the gate is bounded by the
standing fleet, so an operating system nobody runs cannot be validated at all.

This scenario could supply a short-lived node for the duration of a gate run.
The requirement it creates upstream is that the gate accept a node that did not
exist when the gate started, which is a change to the gate rather than to this
scenario. Until that lands, the capability is a design commitment and nothing
more.

Two properties of this scenario make the shape work without special cases. An
ephemeral gate node is an ordinary instance with a short lifetime, so it is
expiry-enforced twice and reconciled like any other. And because it is metered
on the same path, burst capacity for validation shows up in the same cost
figures as everything else rather than hiding in a separate budget.

## Failure Modes

| Failure | Effect | Response |
|---|---|---|
| Business suite unreachable | No reservation can be obtained | Refuse provisioning. Existing instances continue and settle when it returns |
| Bridge unreachable | New instances cannot enroll | Create and meter anyway; queue enrollment; flag the instance as not enrolled |
| Bridge onboarding-key endpoint absent | Unattended enrollment impossible | Blocked upstream. Do not substitute a password path |
| Treasury unreachable | Agent spend cannot be authorised | Refuse agent-initiated requests only |
| Provider API returns an error | Provisioning fails | Mark the intent abandoned; release the reservation; leave nothing behind |
| **Provider call succeeds, response lost** | An instance exists that we may not know about | The intent was written first, so reconciliation matches it. This is the failure the intent-before-action rule exists for |
| Provider API rate limits | Provisioning slows | Back off; never retry a create without the original idempotency key |
| Reconciler finds an unaccounted instance | Cost with no owner | Report it. Never destroy automatically. Mark, then let an operator sweep |
| Reconciler itself is down | Divergence accumulates silently | Instances still drain, because expiry is enforced on the instance as well |
| Provider billing disagrees with our meter | Possible revenue or cost error | Alarm. Provider billing is a reconciliation signal and never a control, because it lags by hours to more than a day |

## Cross-References

- [DOMAINS.md](DOMAINS.md) — the domains that own these integrations
- [ARCHITECTURE.md](ARCHITECTURE.md) — where the boundaries sit
- [FLOWS.md](FLOWS.md) — the order these calls happen in
- [DATA.md](DATA.md) — what is stored here versus owned elsewhere
- [../reference/configuration.md](../reference/configuration.md) — endpoints and credentials
