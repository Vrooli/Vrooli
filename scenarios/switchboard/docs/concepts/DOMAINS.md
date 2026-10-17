# Domains — Switchboard

## Purpose Of This Document

This document is the canonical map of the bounded contexts this scenario owns.
Use it to answer:

- Which domains exist, and what does each one own outright?
- Which domain owns a given piece of data, so that two domains never both claim it?
- In what order must the domains be built, given what each one reads?
- What was deliberately *not* made a domain here, and where does it live instead?

A domain owns its data, its schema, and the rules that interpret it. A domain
that reads another domain has a dependency, and the dependency decides the build
order. Nothing in this scenario reads a domain that is built later.

## Domain Inventory

| Domain | Owns | Reads | Build order |
|---|---|---|---|
| `channels` | Channel descriptors, per-host availability, adapter registration and connection lifecycle | nothing | 1 |
| `conversations` | Threads, messages, participants, media references, ingress de-duplication | `channels` | 2 |
| `agents` | Agent references, address-to-agent bindings, the roster projection | `channels` | 3 |
| `trust` | Contacts, tier assignments, effective-scope resolution | `agents`, `conversations` | 4 |
| `turns` | Turn arbitration, budgets and spend caps, run dispatch, approval gates, reply egress | all of the above | 5 |

The chain is strictly linear except for `turns`, which is the composition point
and is therefore last. This is deliberate: `turns` is where a message becomes an
action, and every safety decision must already be answerable before it runs.

## Domain Details

### `channels`

**Owns.** The channel descriptor registry, loaded from validated JSON files
under `data/channels/` at boot. Per-host availability, computed by evaluating
each descriptor's `requires` block against facts held by `vrooli-bridge` and
`tunnel-manager`. The adapter registry that binds a descriptor to a running
adapter by identifier, and the connection lifecycle of each adapter.

**The invariant that defines it.** The descriptor file is the source of truth
and the database table is a cache rebuilt at boot. Adding a channel is a new
descriptor plus a new adapter, and touches nothing else. No code above this
domain branches on a channel identifier; a capability question is answered by
reading the descriptor.

The descriptor's field contract is specified in
`docs/reference/configuration.md` under *Channel descriptors*.

**Why it is first.** Every other domain asks it a question — what this channel
supports, what its limits are, whether it can run here at all — and it asks
nothing of anyone.

**Boundary.** It does not know what an agent is, what a thread means, or who is
allowed to say anything. It moves bytes and describes what a transport can do.

### `conversations`

**Owns.** Threads, the messages inside them, the participant roster, references
to media, and the ingress de-duplication index keyed on channel identifier plus
remote message identifier.

**The invariant that defines it.** A redelivered message dies here. Every one of
these transports retries, and a duplicate that reaches `turns` is a duplicate
charge against a metered wallet, so de-duplication is an ingress concern and
never a downstream one.

**Boundary.** It records who said what and when. It has no opinion about whether
a message should be answered, and it never starts a run.

### `agents`

**Owns.** The binding between an address — a handle, a number, a room, a thread —
and exactly one agent. The roster projection the console renders.

**The invariant that defines it.** An agent is held *by reference* into
`prompt-manager`, never copied. This scenario stores an identifier and a binding;
it stores no descriptor it does not own. An ambiguous binding is rejected rather
than resolved by picking one.

**Boundary.** It does not define what an agent is, what it may do, or how it is
authored. Those are `prompt-manager` concerns, and duplicating any of them here
would create a fork the first time somebody edits one copy.

### `trust`

**Owns.** Contacts, their tier assignments — owner, trusted, known, stranger —
and the resolution that produces an effective scope for one turn.

**The invariant that defines it.** Effective scope is the narrowest of the sender
tier, the thread ceiling, and the agent's capability grant, computed *before* the
agent reads the message. Empty means refuse. An owner-only scope is unreachable
from any tier below owner by construction, not by configuration, so no
misconfiguration and no prompt can reach it.

**The rule that makes groups safe.** A thread's ceiling is the lowest tier in its
roster. Adding a lower-tier participant narrows the room; it never widens that
participant.

**Boundary.** It decides *what may be attempted*. It does not execute anything and
it does not decide whether the agent is in the mood to speak — that is `turns`.

### `turns`

**Owns.** Turn arbitration — is this addressed to the agent, is it agent-authored,
is there budget left — the per-thread turn budget and spend cap, dispatch of the
run to `agent-manager` under the resolved scope, the approval gate and its expiry,
and egress of the reply through the adapter that received the message.

**The invariant that defines it.** A reply leaves through the adapter that
ingressed it. Thread identity, reply-to identifiers and typing state live in that
adapter's session, and routing a conversational reply through a one-shot
notification path loses all three.

**The two mechanisms that are cost controls, not manners.** An agent-authored
message never starts a turn, and an exhausted hourly budget fails closed. Without
both, two agents in one room exchange metered calls until someone notices.

**Boundary.** It does not execute the agent. `agent-manager` owns run execution,
approvals, and scope attenuation; this domain hands it a scope and receives a
result.

## Shared Concepts

These cross domains and are defined once so that no domain redefines them.

| Concept | Definition | Owner |
|---|---|---|
| **Message envelope** | The single normalised shape every adapter converts to and from. No component above the adapter layer ever receives a channel-native type. | `channels` defines it; every domain consumes it |
| **Address** | A channel-scoped identifier for a counterparty — a handle, an E.164 number, a workspace user, a room. Always qualified by channel; never globally unique on its own. | `agents` |
| **Scope** | A named set of permissions drawn from the vocabulary `agent-manager` already enforces. This scenario narrows and names scopes; it never defines a second vocabulary. | `trust` |
| **Descriptor** | The validated JSON declaration of what a channel is and can do. | `channels` |
| **Tier** | One of owner, trusted, known, stranger. Ordered, and comparison is total. | `trust` |

## Deferred Domains

Named here so nobody builds them by accident before they are earned.

| Deferred domain | Why deferred | Where it lives meanwhile | Revisit trigger |
|---|---|---|---|
| `billing` | Credits, entitlements and reservations are owned by `landing-page-business-suite`. Reinventing them here is explicitly forbidden by the paid-features contract. | LPBS | Never as a domain here; only as a client of LPBS when metering activates |
| `speech` | Transcription and synthesis are `audio-tools`, with `whisper`, `kyutai-stt` and `kokoro` behind it. | `audio-tools` | When call mode activates, and as a consumer only |
| `numbers` | Provisioning and carrier registration involve an operator handoff that no machine may complete. | `persona` owns the handoff contract | When SMS activates at P2 |
| `injection-defence` | A runtime guard against hostile inbound text has no working owner. `prompt-injection-arena` is stale and shaped as an offline tournament, not a guard. | Nowhere today — this is a real gap | Before any non-owner tier is exposed in production |

## Non-Domains

Things that look like they belong here and do not.

| Not a domain here | Why | Actual owner |
|---|---|---|
| Agent authoring, skills, actions, teams | An agent must be usable outside a message thread. A copy here forks on first edit. | `prompt-manager` |
| Run execution, approvals, scope attenuation, DAG workflows | Already built, already governed, already audited. | `agent-manager` |
| One-shot notification routing, quiet hours, device registry | A notification is not a conversation. Both directions can share adapters without sharing a domain. | `notification-hub` |
| Reaching another machine | The reach plane is one answer to "what do I control", and a second registry would split it. | `vrooli-bridge` |
| GitHub, Jira, calendar capability | These are tools an agent *uses*, not places a human talks. Only GitHub's comment surface is channel-shaped. | The scenario that already owns each |
| Social platform identities and posting cadence | Outbound identity management for social accounts; never receives a message addressed to an agent. | `channel-manager` |
| Governed script execution | A capability an agent may be granted, not something this scenario runs. | `program-runtime` |

## Cross-References

- `docs/concepts/ARCHITECTURE.md` — how these domains are wired into surfaces
- `docs/concepts/FLOWS.md` — the flows that cross these boundaries
- `docs/concepts/DATA.md` — the schema each domain owns
- `docs/concepts/INTEGRATIONS.md` — the scenarios named as non-domains above
- `docs/internal/DECISIONS.md` — the decisions that produced this split
- `PRD.md` — the operational targets each domain satisfies
