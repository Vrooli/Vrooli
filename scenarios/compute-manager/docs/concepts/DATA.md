# Data — Compute Manager

## Purpose Of This Document

This document is the data contract: what this scenario stores, which domain is
the authority over each table, what it deliberately does not store, and how the
data is expected to age. Read it before adding a table or a column, so a fact
gets exactly one owner.

> **Status: designed, not implemented.** No schema file exists yet. The tables
> below are the intended shape, recorded ahead of the build so the first
> migration is a decision that was already made rather than one improvised at
> the keyboard.

## Storage Overview

One scenario-owned SQLite database, reached through `api-core/storage` and
resolved from the scenario's own data directory. No external database, no cache,
no object store, and no local service of any kind.

The volumes here are small by construction. A single-operator fleet holds tens
of instances, and even a modest resale product holds thousands rather than
millions. The design constraint is not throughput; it is that **every row that
represents money must be durable and attributable**, because an instance this
database forgets is an instance that keeps billing.

## Data Ownership

| Table | Owning domain | Authority for |
|---|---|---|
| `instance_intents` | intent | That a request was made, and its idempotency key |
| `instances` | instance | The lifecycle state and identity of every instance this scenario created |
| `provider_receipts` | provider | What a provider returned for a call, kept as evidence |
| `reservations` | meter | The link between an instance and the credit hold that permits it |
| `reconcile_findings` | reconcile | Divergence between what we believe and what the provider has |
| `bridge_key_cache` | enroll | The bridge onboarding public key and when it was last refreshed |

Where a fact could sit in two places it sits in one. The intent-to-reservation
link lives on `reservations.intent_id`, not on the intent, because a single
intent can hold a chain of renewals and the many-side owns the pointer. The
instance-to-reservation link lives on `reservations.instance_id` for the same
reason.

Two ownership rules follow from the fleet boundary:

- **Bridge owns node trust.** This database stores a bridge machine identifier
  as a pointer and nothing else. No node key, no pairing state, no scope list is
  copied here. A join by pointer survives a rename; a join by name does not.
- **The business suite owns money.** This database stores reservation
  identifiers and measured quantities. It stores no wallet balance, no plan, no
  price, no card and no invoice. Asking this database what a customer owes is a
  question it is designed to be unable to answer.

## Schema Map

### `instance_intents`

The first write on every provisioning path.

| Column | Notes |
|---|---|
| `id` | Intent identifier |
| `idempotency_key` | Unique. A replay returns the original intent rather than creating a second instance |
| `requested_by` | Operator or tenant identity, resolved server-side, never read from a request body |
| `provider` | Identifier of the adapter selected |
| `spec_json` | Requested size, region, image and lifetime |
| `reservation_id` | The credit hold obtained before the provider was called |
| `state` | `open`, `fulfilled`, `abandoned` |
| `instance_id` | Set once the instance exists; null until then |
| `created_at`, `resolved_at` | |

An intent with `state = open` and no `instance_id` older than the provider's
call timeout is exactly the shape reconciliation looks for when matching an
unaccounted instance back to a request.

### `instances`

| Column | Notes |
|---|---|
| `id` | This scenario's identifier |
| `provider` | Adapter identifier |
| `provider_instance_id` | The provider's own identifier |
| `state` | `requested`, `creating`, `running`, `draining`, `destroyed`, `orphaned`, `unknown` |
| `region`, `size`, `image` | As created, not as requested, so drift is visible |
| `address` | Reachable address, passed to bridge as a locator |
| `bridge_machine_id` | Pointer to the bridge Machine record; null while enrollment is queued |
| `tenant` | Who the capacity is for. Denormalised from the intent so a per-tenant ceiling does not have to join back through it, and so a reconciler-adopted row still has an owner |
| `tags_json` | Owner, purpose and expiry, applied at creation |
| `expires_at` | Enforced by the sweeper and by the instance-side timer |
| `created_at`, `running_at`, `destroyed_at` | The transitions usage is measured from |

**No table stores a price, a rate or a currency**, and that is deliberate.
Elapsed cost is derived at read time from `running_at`, the current clock and
the rate the instance's adapter declares. Storing a rate would create a second
authority on cost that drifts the first time a provider changes its pricing,
which Hetzner did in June 2026. The `cost_divergence` finding compares that
derived figure against what the provider's own billing reports, which is the
only comparison that means anything.

Usage derives from `running_at` and `destroyed_at`, which are transitions this
scenario caused. It is never derived from a loop that observes what is running,
because a dead observer stops billing while the provider keeps charging.

### `provider_receipts`

Append-only evidence. One row per provider call that changed something, holding
the request identifier, the response status, and the provider's own identifier
for whatever it created. Never holds a credential, and never holds a full
response body.

### `reservations`

| Column | Notes |
|---|---|
| `id` | Reservation identifier returned by the business suite |
| `intent_id` | The intent this hold was taken for. Written before the instance exists |
| `instance_id` | Null between reserving and creating, which is the window reconciliation cares about |
| `supersedes` | The reservation this renewal replaces, null for the first in a chain |
| `meter_key` | `compute_minutes` |
| `state` | `held`, `superseded`, `settled`, `released`, `expired` |
| `held_at`, `settled_at`, `quantity` | |

The upstream reservation window is currently ten minutes, which is shorter than
an hour of compute. Until that is parameterised upstream, a heartbeat
re-reserves before expiry, and each renewal is a new row rather than a mutation
so the history survives.

Two things follow, and both are easy to get wrong. **The prior hold is released
only after the successor is confirmed held**, so there is never a moment with no
hold at all, which is what the fail-closed premise requires. And the prior row
moves to `superseded` rather than `released`, so a six-hour instance does not
accumulate thirty-six concurrent holds. Settlement targets the newest row in the
chain, and `supersedes` is what makes the chain reconstructable when it has to
be audited.

### `reconcile_findings`

| Column | Notes |
|---|---|
| `id`, `observed_at` | |
| `kind` | `unaccounted_at_provider`, `destroyed_out_of_band`, `state_divergence`, `cost_divergence` |
| `provider`, `provider_instance_id`, `instance_id` | Whichever side is known |
| `status` | `open`, `acknowledged`, `quarantined`, `resolved` |
| `detail_json` | What differed |

Findings are reported, never auto-resolved by the sweep that raised them.
Marking precedes any sweep so a reconciler defect cannot destroy a running
node, and by the same rule the sweep never settles or releases a reservation.
A `destroyed_out_of_band` finding is actionable: the meter domain drains it and
closes the usage window, so settlement has exactly one owner.

## Migrations And Compatibility

Maturity is `greenfield`, so no migration debt exists and none is expected until
the first real deployment. Once the scenario reaches `pilot`, the ordinary
migration hygiene rules apply and this section must be rewritten to describe
the real forward path.

Two compatibility commitments are worth fixing now, because reversing them later
is expensive:

- `idempotency_key` is unique and stable. A future schema may add columns to
  `instance_intents` but must never make a key reusable.
- `provider_instance_id` is opaque. No code parses it, formats it, or infers a
  region or account from it.

## Import / Export

There is no bulk import path, and there should not be one. An instance row that
did not come from this scenario calling a provider is a row that describes
capacity nobody is responsible for destroying.

Export is read-only and exists for two purposes: an operator reconciling a
provider invoice by hand, and evidence retention. The export carries instance
identity, lifecycle timestamps, measured quantity and provider identifiers. It
carries no credential and no customer identity beyond the tenant reference.

## Retention And Deletion

| Data | Retention |
|---|---|
| `instances` (destroyed) | Retained. The record of what was billed must outlive the machine |
| `instance_intents` (resolved) | Retained alongside their instance |
| `provider_receipts` | Retained while the instance is retained, then pruned |
| `reconcile_findings` (resolved) | Pruned on a schedule; open findings are never pruned |
| `reservations` (settled) | Retained, as the settlement evidence |

The lesson taken from elsewhere in the fleet is that an append-only evidence
table with no retention policy becomes half the database. A pruning job is part
of the first implementation slice, not a follow-up, and the budget declared in
the service manifest is the tripwire.

Deletion of an instance row is not a supported operation. An instance that
should not have existed is recorded as such, because deleting the evidence of a
charge does not undo the charge.

## Privacy Notes

- **No credential is ever written here.** Provider API tokens resolve through
  the credential authority at call time and never reach this database, the
  process environment, or a command line.
- **No payment data is ever written here.** Card details, balances and invoices
  belong to the business suite.
- Tenant identity is stored as a reference, resolved server-side from the
  authenticated caller, and never read from a request body.
- Instance addresses are operational data rather than personal data, but they
  are still real network locations and are excluded from any shared diagnostic
  bundle by default.

## Cross-References

- [DOMAINS.md](DOMAINS.md) — which domain owns each table
- [ARCHITECTURE.md](ARCHITECTURE.md) — where storage sits in the scenario shape
- [FLOWS.md](FLOWS.md) — the order these rows are written in
- [INTEGRATIONS.md](INTEGRATIONS.md) — the data deliberately owned elsewhere
