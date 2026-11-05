# Flows — Switchboard

## Purpose Of This Document

This document is the canonical description of this scenario's temporal behavior:
what happens in what order, what state a thing moves through, and where a flow
can stop. Use it to answer:

- What is the sequence when a message arrives from outside?
- Which decisions happen before an agent reads anything?
- What states can a turn be in, and which are terminal?
- Which flows are deliberately not modelled yet?

Structure lives in `ARCHITECTURE.md`; ownership in `DOMAINS.md`. This document
owns *order and time*.

## Flow Inventory

| Flow | Trigger | Ends when | Criticality |
|---|---|---|---|
| **F1 — Inbound turn** | A message arrives on any adapter | A reply is delivered, or a refusal is stated, or the message is recorded and deliberately unanswered | P0 |
| **F2 — Trust resolution** | F1 needs a scope | An effective scope or an empty set is returned | P0 |
| **F3 — Channel registration** | API boot, or a descriptor changes | Every descriptor is live, unavailable-with-reason, or unimplemented | P0 |
| **F4 — Reply egress** | A turn produces output | The adapter reports delivered, or terminal failure is recorded on the thread | P0 |
| **F5 — Group turn arbitration** | F1 on a thread with more than one human | The turn proceeds, or the agent stays silent, or it refuses out loud | P1 |
| **F6 — Approval gate** | A run requests a scope it lacks | The owner grants once, denies, or the request expires | P1 |
| **F7 — Channel attachment** | The operator attaches an agent to a channel | The binding is live, or a requirement is stated as unmet | P1 |
| **F8 — Agent authoring** | The operator describes a wanted agent | A descriptor is written to `prompt-manager` after explicit confirmation | P1 |

## Flow Details

### F1 — Inbound turn (P0)

The spine of the scenario. Every ordering decision in it exists to make a
specific failure impossible.

1. **Receive.** The adapter takes a transport-native message and converts it to
   the envelope. Transport errors are the adapter's problem and never surface
   above it in transport-native form.
2. **De-duplicate.** `(channel_id, remote_message_id)` is checked. A redelivery
   stops here. *This is before any run and before any metered call, which is the
   whole point: every one of these transports retries, and a duplicate that
   reaches step 7 is a duplicate charge.*
3. **Append.** The message joins its thread, with its roster and media
   references. The thread is durable from this moment, so a crash after this
   point loses nothing and re-answers nothing.
4. **Bind.** The address resolves to exactly one agent. An ambiguous binding is a
   refusal, never a guess.
5. **Arbitrate** (F5 when the thread has more than one human). Is this
   agent-authored? Is the agent addressed? Is there turn budget and spend cap
   left?
6. **Resolve trust** (F2). Produces an effective scope or an empty set.
7. **Dispatch.** `agent-manager.StartRun` with the resolved scope, attenuated
   one-way. *The agent reads the message for the first time here — after every
   permission decision has already been made.*
8. **Egress** (F4). The reply leaves through the adapter that received it.

**Where it can stop, and what the sender sees.** At step 2, silently — a
duplicate is not an event. At step 4, 5 or 6, with a stated reason on the thread;
never silently, because an agent that goes quiet is indistinguishable from one
that is broken. At step 5 in a group when the agent simply was not addressed,
silently by design — but the message is still recorded.

**Acknowledgement discipline.** The transport is acknowledged after step 3, not
before. If SQLite is unreachable the message is *not* acknowledged, so the
transport redelivers rather than the message being lost.

### F2 — Trust resolution (P0)

Runs entirely before step 7 of F1 and touches no model.

1. Resolve the sender to a contact. Unrecognised means `stranger`; there is no
   fourth outcome and no default-allow.
2. Determine the thread ceiling. One-to-one: the sender's own tier. Group: **the
   lowest tier present in the roster.**
3. Effective scope is the narrowest of sender tier, thread ceiling, and the
   agent's capability grant.
4. Strip any owner-only scope unless the sender is the owner. This strip is
   structural — an owner-only scope has no representation reachable from a lower
   tier, so there is no configuration that produces one.
5. Empty scope means refuse, with a statement of what was withheld and what
   would unblock it.
6. Otherwise attenuate one-way into a child scope and hand it to F1 step 7.

**The rule that makes groups safe:** adding a lower-tier participant narrows the
room; it never widens that participant.

### F3 — Channel registration (P0)

Runs at boot and on descriptor change.

1. Read every file under `data/channels/`.
2. Validate against the descriptor schema. **An invalid descriptor fails boot
   loudly, naming the file and the field** — a silently skipped channel is a
   channel the operator believes is running.
3. Evaluate `requires` against host facts held by `vrooli-bridge` and
   `tunnel-manager`. Unsatisfied means unavailable *with a stated reason*, which
   the console renders as a prompt to satisfy the requirement rather than as a
   selectable dead option.
4. Bind to a registered adapter by identifier. No adapter means listed but
   unimplemented — visible, not hidden.
5. Live channels accept agent bindings.

### F4 — Reply egress (P0)

The reply returns through the adapter that ingressed the message, on the same
thread. Media that exceeds the descriptor's declared limits is rejected before
send, with the limit named. A transient failure backs off per the descriptor's
declared policy; a terminal failure is recorded on the thread and surfaced to the
owner. A reply is never re-routed to a different channel to work around a
failure — that would deliver a private answer somewhere the sender did not choose.

### F5 — Group turn arbitration (P1)

1. **Agent-authored?** Record it and stop. An agent never auto-replies to an
   agent. Without this, two agents in one room exchange metered calls until
   someone notices.
2. **Addressed?** In any thread with more than one human, the default is
   `speak_when: mentioned`. Not addressed means stay silent — but still record.
3. **Turn budget?** Exhausted means stay silent and notify the owner *once*, not
   on every suppressed turn.
4. **Spend cap?** Exhausted means refuse out loud, because unlike a budget this
   one the owner can choose to raise.

Steps 1 and 3 are cost controls, not manners.

### F6 — Approval gate (P1)

A run may request a scope it lacks. The request goes to the **owner only**,
never to the thread it arose in, and never to a non-owner participant. An
unanswered request expires; it does not wait indefinitely, because a thread
waiting on a prompt nobody will answer is indistinguishable from a broken one. A
grant is one-time and time-boxed.

### F7 — Channel attachment (P1)

The operator picks an agent, sees channels ordered by the `setup.friction` each
descriptor declares, completes the declared steps, and receives a live binding.
Unmet requirements are shown as the single thing that would fix them. A channel
requiring an operator handoff no machine may complete — carrier registration
being the case that exists — becomes a typed `persona` handoff rather than a
failure.

### F8 — Agent authoring (P1)

A plain-language description is drafted into a complete descriptor, rendered in
full for confirmation — including the capability grant, never summarised away —
and only then written to `prompt-manager` through a typed action. The model
drafts; it does not write.

## State Machines

### Turn lifecycle

```
[*] -> idle
idle              -> queued            : inbound accepted (after de-duplication)
queued            -> refused           : effective scope empty
queued            -> running           : scope granted
running           -> awaiting-approval : run requests a scope it lacks
awaiting-approval -> running           : owner grants once
awaiting-approval -> refused           : denied, or expired
running           -> replying          : output ready
replying          -> idle              : delivered
replying          -> retrying          : transient adapter error
retrying          -> replying          : backoff elapsed
retrying          -> dead              : terminal — recorded on thread, surfaced to owner
refused           -> idle
```

`refused` and `dead` are both terminal for the turn and both are *visible*.
Neither is silent.

### Channel lifecycle

```
[*] -> declared          : descriptor file present
declared -> invalid      : schema validation fails  (boot fails loudly)
declared -> unavailable  : `requires` unsatisfied on this host  (reason stated)
declared -> unimplemented: no adapter registered for this id
declared -> live         : validated, requirements met, adapter registered
live -> degraded         : adapter connected but reporting transport errors
degraded -> live         : recovery
live -> unavailable      : a host fact changed (a Mac node went offline)
```

### Contact tier

```
stranger -> known    : the owner records a reply relationship
known    -> trusted  : explicit operator assignment
trusted  -> owner    : NOT REACHABLE. Owner is established by identity, never by promotion.
any      -> stranger : explicit operator demotion
```

The missing transition into `owner` is the point of the diagram.

## Maturity Ladder

| Level | Meaning | Where this scenario is |
|---|---|---|
| L0 | Flows named | ✅ F1–F8 named above |
| L1 | Flows described with stop conditions | ✅ this document |
| L2 | State machines declared | ✅ three above |
| L3 | Flows implemented | ❌ no domain code exists |
| L4 | Flows covered by tests tagged to requirements | ❌ every requirement carries a `manual` stub |

This scenario is deliberately at **L2**: the flows are specified and nothing is
built. Read the level honestly — a complete document is not a working flow.

## Production Shape

When these flows run in production:

- F1 runs in the API process, per inbound message, with no queue between adapter
  and core. Throughput expectations are conversational, not bulk.
- F3 runs once at boot and on descriptor change, never per message.
- F5's budget windows and F6's expiry are derived from `api/internal/clock/`, so
  both are provable in tests without sleeping.
- Cross-node flows dispatch this scenario's own CLI verb to a remote instance
  through `vrooli-bridge` durable dispatch, so a briefly offline Mac delays a
  delivery rather than dropping it.
- Nothing in F1 is idempotent *after* step 2, which is why step 2 is not
  optional and not deferrable to a later phase.

## Deferred / Unmodeled Flows

| Flow | Why deferred | Blocked on |
|---|---|---|
| Voice call turn-taking, barge-in, endpointing | Needs a working speech path first, and dictation currently never sends `engine_id`, so transcription has been running on CPU with every signal green | `audio-tools` engine-selection fix |
| Runtime injection defence | No owner. `prompt-injection-arena` is stale and shaped as an offline tournament, not a runtime guard | A redesign of that scenario from its own operational targets |
| Carrier registration | Contains steps no machine may take | `persona` handoff contract, at P2 |
| Media transcoding | No evidence any target channel needs it yet | A descriptor that declares a format the source cannot produce |
| Multi-agent collaboration inside one thread | Deliberately out of scope. Two agents in a room is currently a *hazard* to bound, not a capability to build | Not planned |

## Cross-References

- `docs/concepts/ARCHITECTURE.md` — the boundaries these flows cross
- `docs/concepts/DOMAINS.md` — the domain that owns each step
- `docs/concepts/DATA.md` — what each step persists
- `docs/internal/SECURITY.md` — the threat model F2 answers
- `docs/internal/DECISIONS.md` — why the ordering in F1 is what it is
- `PRD.md` — the operational targets these flows satisfy
