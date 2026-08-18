# Integrations — Persona

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Posture

Persona is **not standalone**, and that is a deliberate design outcome
rather than an accident of convenience. The scenario's central promise —
hold bindings and policy, never the sensitive payload — only works if
something else holds the payload. Every scenario dependency below exists
because the alternative was to duplicate a custody, verification, or
storage concern that already has a owner and an audit story.

It needs **no external Vrooli resources** beyond embedded SQLite, and
that remains true through P2. Every heavier capability is satisfied by
an existing scenario rather than by a new resource.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| `agent-manager` | scenario | **yes** | access | Live identity-verification endpoint; `persona_id` in `Claims.Meta`; `persona.act-as:<id>` scope | **Fails closed.** Act-as is refused and journaled. Read-only surfaces stay available. |
| `document-manager` | scenario | **yes** (from `PSN-P0-008`) | documents | Document reference plus release-into-handoff | Binding and release refused; existing bindings remain listable. |
| `secrets-manager` | scenario | **yes** | channels | Mailbox and provider credential retrieval | Code retrieval refused for affected adapters; other channels unaffected. |
| `prompt-manager` | scenario | no (P1) | access | Act-as grants as member configuration | Falls back to locally held ACL entries; grant-derived access is unavailable, never widened. |
| `device-control` | scenario | no | channels | Lease a phone, read a code via bounded action | That one adapter is unavailable; other OTP adapters continue. |
| `notification-hub` | scenario | no | handoffs | Handoff delivery relay | Built-in handoff queue serves the operator directly. No handoff is lost. |
| `browser-automation-studio` | scenario | no | handoffs, channels | Consumer that drives flows using a resolved persona | No effect on this scenario; BAS simply has no persona to use. |
| SMS / phone provider | third-party | no (P1/P2) | channels | Provider API behind the OTP adapter contract | That adapter reports unavailable; never a silent fallback to another persona's route. |

## Vrooli Resources

**None, by design — including at P2.** The scenario stores small
relational records and needs no shared database, cache, vector store, or
object store. Postgres is explicitly not warranted: this is a
single-operator scenario with low write volume and no concurrent-writer
pressure, and SQLite's guarantees are sufficient for every table above.

The one thing that would change this assessment is a workload with many
concurrent external writers, which this scenario does not have — code
retrieval is agent-initiated and serialised by the flow that requested
it. If that ever changes, revisit this section before adding a resource.

## Scenario Dependencies

Declared in `.vrooli/service.json`. Grouped by why they exist:

**Custody delegation** — the reason the scenario stays small.

- `document-manager` holds identity documents. It already runs
  sensitivity classification as a fail-closed choke point *before* tier
  selection, writes a per-document custody receipt, and keeps an
  append-only custody journal. Rebuilding that here would produce a
  second, weaker custody story for the most sensitive data in the
  system.
- `secrets-manager` holds mailbox and provider credentials. A persona
  records *which* credential to use; it never holds the secret.

**Verification** — the reason the delegation chain is trustworthy.

- `agent-manager` owns run identity: `Claims.Subject`, the attenuated
  scope list, and `Attenuate()`'s one-way narrowing. This scenario adds
  one outer link (`persona_id` in `Claims.Meta`, plus the
  `persona.act-as:<id>` scope namespace) and consumes the rest. It never
  reimplements token parsing, scope intersection, or attenuation.
  Verification is a live call, so this is a **hard runtime dependency on
  the act-as path** — an accepted availability cost, chosen over a
  degraded-evidence alternative.

**Policy source** — the reason there is no second org chart.

- `prompt-manager` owns teams, members, and contracts. Which members may
  act as which persona is ordinary member configuration and is read from
  there rather than mirrored. Optional at P0 because the local ACL
  covers a single-operator install.

**Optional capability** — enhancements that never become requirements.

- `device-control` is one OTP adapter among several, using its existing
  lease-and-bounded-action pattern unchanged.
- `notification-hub` relays handoffs to the right device. The built-in
  queue is the floor and always works.

**Consumers, not dependencies** — one-way, and this scenario has no
knowledge of them.

- `treasury` resolves a persona before executing a mandate. Persona
  knows nothing about budgets, rails, or money.
- `browser-automation-studio` drives flows using a resolved persona.

## Third-Party Services

| Service | Purpose | Required? | Status |
|---|---|---|---|
| SMS / phone-number provider | Deliver and retrieve codes on a number the persona owns | No | **P1/P2.** Deliberately excluded from P0 so the scenario ships with zero paid dependencies; the email adapter satisfies `PSN-P0-006` alone. |
| Email provider | Host the controlled address | No | The address may be self-hosted, a plus-addressed alias, or a provider mailbox. The contract is IMAP/SMTP-shaped access, not a named vendor. |

No third-party package is added outside the governed
`scenario-dependency-analyzer` path, and no dependency is added merely
because the fenced example domain happens to use it.

## Failure Modes

| Failure | Detection | Degradation | Recovery |
|---|---|---|---|
| `agent-manager` unreachable | Verification call errors | **Act-as refused and journaled.** Reads, handoff completion by a human, and the operator console stay available. | Automatic once reachable; no state repair needed. |
| Verification returns invalid | Explicit invalid result | Act-as refused, refusal journaled with the reason | Operator inspects the run; never overridden by a flag. |
| `document-manager` unreachable | Binding/release call errors | Binding and release refused; existing bindings listable | Automatic; releases are idempotent by handoff id. |
| `secrets-manager` unreachable | Credential fetch errors | Affected channel adapters report unavailable | Automatic; no code value was cached to leak. |
| `prompt-manager` unreachable | Grant fetch errors | Local ACL only — **narrower, never wider** | Automatic. |
| `notification-hub` absent or down | Delivery attempt fails or hub not declared | Handoff stays queued and visible in the console | Operator works the queue directly; no handoff is lost. |
| OTP adapter fails | Retrieval times out or errors | That adapter reports unavailable | **Never silently falls back to another persona's route** — a wrong-persona code is worse than no code. |
| Controlled mailbox rejects auth | Channel test fails; staleness finding raised (P1) | Email OTP unavailable for that persona | Operator re-authorises the mailbox credential. |

The unifying rule: **every degradation is narrower, never wider.** No
failure path grants access that the healthy path would refuse.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`DATA.md`](DATA.md) — what is stored here versus referenced elsewhere
- [`FLOWS.md`](FLOWS.md) — where dependencies sit in each workflow
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — threat model
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — operator response
