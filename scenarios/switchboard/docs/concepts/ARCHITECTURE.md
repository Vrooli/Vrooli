# Architecture — Switchboard

## Purpose Of This Document

This document is the canonical description of this scenario's shape: which
surfaces exist, where the boundaries are, how data crosses them, and what may
be added where. Use it to answer:

- What are the surfaces, and which one owns a given responsibility?
- Where does an inbound message enter, and what does it become?
- Which infrastructure is shared, and on what grounds?
- Where does new behavior go, and in what order is it added?

Product ownership lives in `DOMAINS.md`; persistence in `DATA.md`; temporal
behavior in `FLOWS.md`; dependency contracts in `INTEGRATIONS.md`. This document
does not restate those — it is the frame they hang on.

## Scenario Shape

A Tier 1 local scenario generated from the `react-vite` template: a Go API over
Connect-RPC with proto-owned wire contracts, a typed Go CLI with full headless
parity over that API, and a React + Vite + TypeScript operator console. There is
no separate worker process; adapter connections, the delivery of replies, and
budget accounting all run inside the API process against SQLite.

What makes this scenario's shape unusual is that **it has an inbound edge that
nothing else in Vrooli has**. Every other scenario is entered by a person, an
agent, or another scenario, through a surface Vrooli controls. This one is
additionally entered by the outside world, on transports Vrooli does not own,
from senders that have not authenticated. That asymmetry is why the layering
below is strict rather than conventional: everything above the adapter layer
must be unable to tell which transport a message came from, and everything above
the trust guard must be unable to act before a scope exists.

```
outside world
     |  (many transports, none of them ours)
     v
+------------------+   channel-native types stop here
| channel adapters |   one per channel, bound to a descriptor by id
+------------------+
     |  message envelope (the only shape above this line)
     v
+------------------+
| conversations    |   idempotency, threads, roster, media
+------------------+
     |
     v
+------------------+
| agents  + trust  |   address -> agent; sender -> tier -> effective scope
+------------------+
     |  a scope, or a refusal
     v
+------------------+
| turns            |   arbitration, budgets, dispatch, approvals, egress
+------------------+
     |  StartRun(scope)                    ^  reply, back out the same adapter
     v                                     |
  agent-manager  ---->  program-runtime, audio-tools, every other scenario
```

## System Boundaries

| Boundary | What crosses it | What must not cross it |
|---|---|---|
| Adapter ↔ core | The message envelope, and nothing else | Channel-native types, channel identifiers used as branch conditions, transport error shapes |
| Core ↔ `agent-manager` | An agent identifier, a resolved scope, thread context, and a result | Any request to widen a scope; any execution this scenario performs itself |
| Core ↔ `prompt-manager` | An agent identifier and a read of its descriptor and grant | A written or cached copy of a descriptor this scenario does not own |
| Core ↔ `vrooli-bridge` | This scenario's own cataloged CLI verb, dispatched to a named machine | Any channel vocabulary — the bridge must never learn what a channel is |
| Core ↔ LPBS | A reservation, an execution, a finalisation | Any locally invented notion of credit, entitlement, or plan |
| Owner ↔ non-owner | Everything a resolved scope permits | Owner-only scope, which is unreachable from a lower tier by construction |

The last row is the one that matters most and is the reason the trust guard is a
domain rather than a middleware: a boundary enforced in a request filter can be
bypassed by any new call path, while a boundary enforced by scope resolution
before dispatch cannot.

## Contracts And Data Flow

**Inbound.** An adapter receives a transport-native message and converts it to
the envelope. `conversations` de-duplicates on `(channel_id, remote_message_id)`
and appends to a thread — this happens before any run starts and before any
metered call is possible, so a webhook retry costs nothing. `agents` resolves the
address to exactly one binding, refusing an ambiguous one rather than choosing.
`trust` resolves the sender's tier, the thread's ceiling, and the agent's grant
into one effective scope. `turns` arbitrates — is this addressed to us, is it
agent-authored, is there budget — and dispatches to `agent-manager` under that
scope.

**Outbound.** The reply returns through the adapter that received it, on the same
thread. This is a deliberate constraint rather than an implementation detail:
thread identity, reply-to identifiers, and typing state live in that adapter's
session, and none survives a hop through a one-shot notification path.

**Descriptor flow.** Descriptors are validated JSON files under `data/channels/`.
They are loaded and validated at boot, and boot fails loudly naming the file and
field when one is invalid. The file is the source of truth; the table is a cache.

**Cross-node flow.** A host-bound channel — iMessage on a Mac — is reached by
dispatching this scenario's own cataloged CLI verb to that machine through
`vrooli-bridge` durable dispatch. The remote instance runs the same code against
its own local adapter. The bridge carries the call and never learns what a
channel is, which is the same shape `notification-hub` and `device-control`
already chose, for the same reason: a second registry would split the single
answer to "what do I control".

## Shared Infrastructure

Shared infrastructure is allowed only when the code is business-vocabulary-free
and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting and test-substitutable. | Middleware, repositories, **budget windows and approval expiry**. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

The clock seam carries more weight here than in a typical scenario: hourly turn
budgets and approval expiry are both time-derived safety behaviors, and both must
be provable in a test without sleeping.

**The message envelope is not shared infrastructure.** It is product vocabulary
and is owned by `channels`. Placing it in a generic bucket would invite
transport-specific fields into it, which is precisely the drift this scenario
must not have.

If shared infrastructure starts using product vocabulary, move that piece back
into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by growing
generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/switchboard/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

Finish one domain before starting the next. Do not build every API, then every
CLI, then every UI — horizontal layers defer the first working flow to the end,
and the UI then meets contract defects a finished domain would have surfaced in
its first slice. The domain order is fixed in `DOMAINS.md`.

**Adding a channel is a different and much smaller operation, and this is the
central architectural claim of the scenario:**

1. Add a validated descriptor at `data/channels/<id>.json`.
2. Add an adapter at `api/internal/channels/adapters/<id>.go` implementing the
   adapter interface.
3. Run the shared `channel-conformance` phase against it.

Nothing else changes. Not the envelope, not the router, not the trust guard, not
the console, not the funnel, not the test suite. If a change to a channel
requires editing anything above the adapter layer, the descriptor is missing a
field — add the field, do not add the branch.

## Architecture Maturity

| Aspect | State | Evidence |
|---|---|---|
| Charter and requirements | **Authored** | `PRD.md` with 40 operational targets; `requirements/` registry with `SWBD-*` identifiers linked by `prd_ref`; `vrooli scenario requirements validate` passes |
| Domain map | **Authored** | `DOMAINS.md`, five domains with a linear build order |
| Dependency decisions | **Authored** | `INTEGRATIONS.md` and `.vrooli/service.json` agree; nine scenario dependencies declared, zero resources at P0 |
| Design language | **Adopted** | `DESIGN.md`, `vrooli-default` kit, token contract intact |
| Experience contract | **Authored, draft** | Six product pages and three journeys under `experience/`, all `status: draft` — authored ahead of the build by intent |
| Proto contracts | **Not started** | No `packages/proto/schemas/switchboard/` tree exists yet |
| Domain implementation | **Not started** | Only the template's example `notes` domain exists |
| Evidence | **Stubbed** | Every requirement carries a `manual` validation stub explicitly marked "replace with a test-typed validation once the behavior exists". The registry validator reporting a complete traceability level reflects complete *stubs*, not tested behavior, and must not be read as evidence of a working system |

## Intentional Deviations

| Deviation | From what | Why |
|---|---|---|
| No resource dependencies at P0 | The common pattern of taking `postgres` for anything stateful | Neither is `supported` on macOS: `postgres` is `build-verified` there with no hardware run performed, and the platform matrix's two tables disagree about `redis` (generated matrix `build-verified`, narrative table `unsupported`). Taking either puts the Mac fleet node on unproven footing, and the Mac lane is the most valuable channel in the product. Neither is Docker-backed — both are `managed-service` — so the reason is macOS evidence, not a container runtime |
| Descriptor files, not database rows, as the source of truth | Ordinary schema-first persistence | Adding a channel must not require a migration or a code change. `channel-manager` already proved this shape and its loader is the reference |
| Trust resolution is a domain, not middleware | The usual auth-middleware placement | A middleware filter is bypassed by every new call path added later. Scope resolution before dispatch cannot be |
| Replies do not use `notification-hub` | Its existing role as the delivery spine | Its senders are one-shot by design and carry no thread identity. Sharing adapters is correct; sharing the delivery path is not |
| Experience specs authored before the UI | Building first and documenting after | This scenario was explicitly set up documentation-first. Pages carry `status: draft`, which the contract defines as intent authored ahead of the build with advisory reconciliation |
| The in-app chat is an adapter | Treating the native surface as privileged | If the core can tell in-app from Telegram, a feature will eventually become in-app-only by accident |

## Documentation Architecture

| Question | Document |
|---|---|
| What capability is this, and what must it do? | `PRD.md`, `requirements/` |
| What are the bounded contexts and their build order? | `docs/concepts/DOMAINS.md` |
| What is the shape and where do things go? | this document |
| How does behavior move over time? | `docs/concepts/FLOWS.md` |
| What is stored, owned, retained, deleted? | `docs/concepts/DATA.md` |
| What does this depend on and how does it fail? | `docs/concepts/INTEGRATIONS.md` |
| Why is it like this? | `docs/internal/DECISIONS.md` |
| What is the attack surface? | `docs/internal/SECURITY.md` |
| What is known-broken or unowned? | `docs/internal/PROBLEMS.md` |
| What has actually been done? | `docs/internal/PROGRESS.md` |
| How is it run and recovered? | `docs/operations/RUNBOOK.md`, `DEPLOYMENT.md`, `OBSERVABILITY.md` |
| How does it earn its keep? | `docs/business/MONETIZATION.md`, `GO-TO-MARKET.md` |
| What are the screens and journeys? | `experience/`, `DESIGN.md`, `docs/concepts/UI-ARCHITECTURE.md` |

## Cross-References

- `docs/concepts/DOMAINS.md` — the domains this shape hosts
- `docs/concepts/FLOWS.md` — the flows that cross these boundaries
- `docs/concepts/DATA.md` — what each domain persists
- `docs/concepts/INTEGRATIONS.md` — the dependency contract
- `docs/internal/SEAMS.md` — the substitutable interfaces
- `docs/internal/DECISIONS.md` — the decision log behind every deviation above
- `docs/START-HERE.md` — the initialization gates still open
