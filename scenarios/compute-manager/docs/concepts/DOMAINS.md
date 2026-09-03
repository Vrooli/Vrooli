# Domains — Compute Manager

## Purpose Of This Document

This document is the domain map: the bounded contexts this scenario owns, what
each one is responsible for, and which data it is the authority over. Read it
before adding a package, a table, or an API surface, so new work lands in the
domain that already owns the concept instead of creating a second owner for it.

> **Status: designed, not implemented.** The scenario was generated from the
> `react-vite` template and currently contains only template code. Every domain
> below is a decision recorded ahead of the build. The `Source Paths` column
> names where each domain will live, not where it is.

The load-bearing rule for this scenario: **it owns capacity and cost, and
nothing else.** Node trust, public exposure, deployment, subscriptions and
agent spend limits all have existing owners elsewhere in the fleet. When a
change here starts to look like one of those, it belongs in the other scenario.

## Domain Inventory

| Domain | Responsibility | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|
| intent | Records that a request was made, with its idempotency key, before any provider is contacted | `instance_intents` | crud | validation | intent, idempotency key | api/internal/intent |
| instance | Owns the instance lifecycle state machine and the durable instance record | `instances` | orchestration | mutation, crud | instance, lifecycle | api/internal/instance |
| provider | The adapter interface, its implementations, and each provider's declared billing facts | `provider_receipts` | provider | infrastructure | provider, adapter, rounding | api/internal/provider |
| meter | Reserves credit before provisioning, re-reserves on a heartbeat, settles on teardown, and is the only domain that writes money | `reservations` | service | mutation, validation | reservation, settlement, meter key | api/internal/meter, api/handlers/meter, cli/domains/meter |
| reconcile | Compares provider inventory against local records in both directions and reports divergence | `reconcile_findings` | reporting | aggregation, validation | orphan, unaccounted, divergence | api/internal/reconcile |
| expiry | Destroys instances past their lifetime and renders the instance-side drain timer | none | mutation | orchestration | expiry, lifetime, drain | api/internal/expiry |
| enroll | Hands a booted instance to the bridge onboarding contract so it becomes a trusted node | `bridge_key_cache` | orchestration | service | enrollment, first boot, locator | api/internal/enroll, api/handlers/enroll, cli/domains/enroll |
| presentation | Renders inventory, elapsed cost and remaining lifetime, and owns the cross-domain read model the dashboard consumes | none | query | aggregation | inventory, elapsed cost, expiring soon | api/handlers/inventory, ui/src/features/inventory |
| health | Liveness and readiness for the scenario itself, and the capability descriptor surface | none | reporting | infrastructure | health, readiness | api/handlers/health, api/handlers/capabilities, ui/src/features/health |

## Domain Details

### intent

The genuine first write on every provisioning path, and the reason both
orphaned instances and stranded credit holds are recoverable rather than
invisible.

A request is persisted with its idempotency key **before** the business suite
is called and therefore before the provider client is reachable. The intent
starts in `reserving`, gains the reservation id once credit is held, and moves
to `refused` if credit is denied. A refusal leaves a cheap prunable row, which
is the price of never stranding a hold that no local record points at.

If a create call succeeds while its response is lost, the intent
row survives and reconciliation can match it to the instance that actually
exists. Replaying the same key returns the original intent instead of creating
a second machine.

This ordering is the single most important invariant in the scenario. Most
orphaned-instance problems in the wild come from calling first and recording
afterwards.

Owns `OT-P0-002`.

### instance

The lifecycle state machine and the durable record it advances.

States are `requested`, `creating`, `running`, `draining` and `destroyed`.
Two further states, `orphaned` and `unknown`, are outcomes reconciliation
assigns rather than states anything requests.

The instance record is this scenario's own object. It is deliberately **not**
called a machine, because `vrooli-bridge` already owns a Machine, meaning
durable operator intent for a node. One bridge Machine may point at the
instance currently backing it; the two are joined by pointer, never by name.

Owns `OT-P0-001`.

### provider

One adapter interface with four methods: create, describe, list and destroy.

There is no stop, and that is a decision rather than an omission. A stopped
instance still bills at the full rate on most providers surveyed, so a pause
control would cost full price for no value. Destroy is the only stop, and
`COMPUTEM-P0-007` specifies a structural test that will assert no such method
exists anywhere in the scenario.

Each adapter also declares its provider's billing facts as data instead of
hiding them: rounding behaviour, minimum billable unit, whether a stopped
instance bills, and whether inbound traffic counts against the transfer
allowance. Callers select a provider by identifier and never name one in code.

Owns `OT-P0-001`, `OT-P0-007` and `OT-P1-004`.

### meter

The reservation spine, and the reason provisioning can refuse.

Hosted compute is cost-bearing, so enforcement runs server-side before the
machine boots. The order is reserve, provision, re-reserve on a heartbeat,
settle on teardown, and release on failure. A refused reservation short-circuits
with no provider call and an out-of-credit outcome the caller can distinguish
from a server error.

This domain holds no wallet, no plan and no invoice. It calls the business
suite's existing reservation surface and stores only the reservation
identifiers it needs to settle.

Owns `OT-P0-006` and `OT-P1-002`.

### reconcile

The safety net, and deliberately the least powerful domain in the scenario.

It compares provider inventory against local records in both directions on a
schedule. Present at the provider and absent locally is recorded as
`unaccounted_at_provider`. Present locally and absent at the provider is
recorded as `destroyed_out_of_band`.

It **reports and never resolves.** A finding is a row that something else acts
on, not an action the sweep takes. This is what stops a reconciler bug from
destroying a running node, and by the same rule it never settles, releases or
adjusts a reservation. A `destroyed_out_of_band` finding is drained by the
meter domain, which closes the usage window, so exactly one domain writes
money. Marking precedes sweeping for the same reason.

Owns `OT-P0-003` and `OT-P1-003`.

### expiry

Two enforcement points for one guarantee.

The scenario sweeps for instances past their lifetime and destroys them. The
instance also carries a timer, installed in its first-boot configuration. The
second one exists because the first cannot run when this scenario is down, and
an unbounded instance costs money for as long as nobody notices.

The instance-side timer is a **bounded lease that the instance renews**, not a
copy of the full lifetime. That distinction is what makes extension possible at
all: this scenario holds no execution channel into a running machine, so a timer
written once at first boot could never be moved. As a renewing lease it fires by
default and is deferred by a renewal the instance collects on its own outbound
check, which keeps the drain guarantee and still lets an operator extend.

Owns `OT-P0-004`.

### enroll

Delegation, and the domain most likely to be misunderstood as more than it is.

It renders a first-boot configuration carrying the bridge's onboarding public
key, then creates the bridge Machine record with the instance address as a
locator and starts bridge onboarding. Because the instance already trusts that
key, passwordless access works on first contact, so **no secret crosses any
wire** and no interactive step is needed. The public key itself does travel,
from bridge to this scenario and from this scenario into the provider's
create call, which is what a public key is for.

The key must be embedded at create time and cannot be retrofitted afterwards,
so this domain caches it rather than fetching it on the provisioning path. A
warm cache is what makes the bridge dependency a degrade rather than a hard
stop: enrollment can queue and retry, but an instance created with no key in
its first-boot configuration can never be enrolled by the unattended path.
When the cache is cold and bridge is unreachable, provisioning refuses instead
of creating a machine it could never enroll.

**This domain contains no SSH implementation and never will.** First touch,
bootstrap and node trust belong to `vrooli-bridge`.

Owns `OT-P0-005` and `OT-P1-001`.

### presentation

The read model, and the only domain that spans the others.

Inventory, elapsed cost and remaining lifetime are the three facts the operator
surface leads with, and none of them belongs to a single product domain. Cost
is derived at read time from the instance's `running_at`, the current clock and
the rate its adapter declares, so no domain stores a price and no second
authority on cost exists. Expiring-soon and over-budget are computed here too.

It exists as a domain for the reason `tunnel-manager` gives for the same
decision: assigning cross-domain read tests to one product domain would make
the domain map misleading.

Owns `OT-P1-005`.

### health

Liveness, readiness and the capability descriptor surface.

Generated by the template and kept deliberately. It is listed here because a
domain map that omits code which exists on day one is incomplete rather than
aspirational.

Owns no operational target.

## Shared Concepts

**Instance.** A virtual machine this scenario created and is responsible for
destroying. Distinct from a bridge Machine.

**Intent.** The durable record that a request was made, written before the
provider is called, carrying the idempotency key that makes a retry safe.

**Reservation.** A hold on a tenant's credit taken before provisioning and
settled at teardown. Held by the business suite; this scenario stores only the
identifier.

**Unaccounted instance.** An instance the reconciler found at a provider that
this scenario has no record of. Named "unaccounted" rather than "orphan" in the
operator surface, because it is cost before it is a bug.

**Billing facts.** The per-provider constants an adapter declares: rounding,
minimum billable unit, stopped-instance behaviour, and inbound traffic
treatment. Product decisions depend on these, so they are data rather than
assumptions buried in an adapter.

## Deferred Domains

| Domain | Why deferred | Trigger to build |
|---|---|---|
| pooling | Warm instance reuse only pays for itself once measured churn shows provider hour rounding dominating the bill | Real usage data showing short-lived instances are a material cost |
| tenancy | Isolation between paying customers is a larger surface than a single-operator fleet needs | The customer purchase target `OT-P2-001` is promoted |
| placement | Choosing a region or provider by cost or latency needs more than one provider and real measurements | A second adapter exists and cost data has accumulated |
| image | Building and pinning custom machine images would speed boot but adds a build pipeline | First-boot configuration proves too slow or too fragile in practice |

### Pooling

Warm instance reuse, deferred deliberately and tracked as `OT-P2-003`.

Most providers round a partial hour up to a full one, so an instance created and
destroyed inside forty minutes still costs an hour. Reusing a warm instance
instead of creating a new one turns that rounding waste back into capacity. It
is worth building only once real usage data shows short-lived instances are a
material share of the bill, because a pool that nobody fills is a second
lifecycle to maintain for no saving.

It would live in the `instance` domain rather than becoming a domain of its own,
because a pooled instance is still an instance and the state machine already has
the states it needs.

## Non-Domains

Things that look like they might belong here and do not.

- **Node identity, pairing, scopes and dispatch.** Owned by `vrooli-bridge`. An
  instance becoming a trusted node is a bridge concern that this scenario
  triggers, not one it performs.
- **SSH, first touch and bootstrap.** Owned by `vrooli-bridge`. This scenario
  never opens a shell on anything.
- **Public hostnames, DNS and ingress.** Owned by `tunnel-manager`.
- **Deploying scenarios onto a machine.** Owned by `scenario-to-cloud` and
  `deployment-manager`. This scenario delivers a node, not a running workload.
- **Subscriptions, entitlements, wallets and invoices.** Owned by
  `landing-page-business-suite`. This scenario reserves and settles against it.
- **Bounding what an agent may spend.** Owned by `treasury`.
- **The sellable definition.** Owned by `offer-desk`.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md) — the scenario shape these domains sit in
- [FLOWS.md](FLOWS.md) — how the domains sequence during a provisioning run
- [DATA.md](DATA.md) — the tables each domain owns
- [INTEGRATIONS.md](INTEGRATIONS.md) — the scenarios these domains delegate to
- [../internal/SEAMS.md](../internal/SEAMS.md) — the substitution boundaries
