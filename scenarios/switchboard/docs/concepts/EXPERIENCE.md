# Experience Design

## Purpose Of This Document

Define the product's user experience: what it is trying to make a person feel
and understand, the information architecture that carries that, the visual and
motion language, and the rules every surface obeys. It is the prose companion
to the machine-readable contract in `experience/`, which holds the same intent
as typed claims. Where the two disagree, `experience/` wins — it is the one a
validator reads.

This document does not describe the `ui/` source tree. That is
`path:docs/concepts/UI-ARCHITECTURE.md`.

## The Design Thesis

**The thread is the product. Everything else is a console around it.**

Switchboard is used in two completely different postures, and conflating them
is the failure mode this design exists to avoid:

| Posture | Who | Where | What it must feel like |
|---|---|---|---|
| **Conversing** | anyone with a handle | mostly a phone, mostly not in this app | A message thread. Nothing else. |
| **Governing** | the owner | a desktop, deliberately | An instrument panel: dense, factual, honest about failure |

The conversing posture is judged against iMessage, Telegram and WhatsApp — apps
people use for hours a day and whose conventions they have fully internalised.
Any invention there reads as a bug. The governing posture is judged against
good operations software, where invention is welcome if it makes a risk legible.

So the rule is: **be conventional in the thread, opinionated in the console.**

## What The Product Must Make True

Three things the design is accountable for, in order:

1. **A person can tell what the agent is allowed to do, before it does it.**
   Every refusal, gate and grant is written for someone judging a risk, never
   as a permission error. This is the only part of the product that can cause
   harm, so it gets the most design attention and the most direct language.
2. **A person can tell who wrote a message.** Authorship is functional, not
   decorative: the loop breaker in `OT-P1-004` depends on it, and a person must
   be able to see what the loop breaker sees.
3. **A person can tell which channel they are looking at, without reading it.**
   A cross-channel list where the channel is only in the detail pane forces a
   click to answer the first question anyone has.

## Information Architecture

Nine surfaces. Every one of them earns its place by answering a question no
other surface answers.

| Surface | Route | The question it answers |
|---|---|---|
| Welcome | `/welcome` | Can I try this right now? |
| Overview | `/` | Is anything waiting on me? |
| Conversations | `/conversations`, `/conversations/:threadId` | What is being said, and may the agent answer? |
| Call | `/call/:threadId` | *(live voice, full viewport)* |
| Agents | `/agents` | Which agents exist and where are they reachable? |
| Agent | `/agents/:agentId` | What is this one allowed to do? |
| New agent | `/agents/new` | How do I make another? |
| Channels | `/channels` | What can I attach, and how much work is it? |
| Contacts | `/contacts` | Who can reach my agents, and at what tier? |
| Settings | `/settings` | What changes behaviour for everything? |

**Overview is not a dashboard.** It is a queue of things that need a person,
and it is empty most of the time. When it is empty it says so in a sentence
rather than rendering zeroed charts, because zeroed charts read as a broken
page and a sentence reads as good news.

**Call is a route, not a modal.** A call holds the whole viewport, survives a
refresh, and can be deep-linked. Ending a call returns to the thread with the
transcript already written into it.

## Layout System

One responsive skeleton, three densities.

```
wide     ┌──────┬───────────────┬──────────────────────────┐
(≥1440)  │ rail │ list          │ detail                   │
         └──────┴───────────────┴──────────────────────────┘

desktop  ┌──────┬──────────────────────────────────────────┐
(≥1024)  │ rail │ list + detail, detail dominant           │
         └──────┴──────────────────────────────────────────┘

mobile   ┌─────────────────────────────────────────────────┐
(<768)   │ one pane, stacked, bottom navigation            │
         └─────────────────────────────────────────────────┘
```

Rules that hold at every width:

- **The detail pane is the largest thing on screen.** The list is navigation.
- **Chrome is pinned; only content scrolls.** The composer never scrolls away,
  and the call's end button is reachable in every state including error states.
  A call you cannot hang up is the worst failure this product has.
- **Mobile is a stack, not a squeeze.** List and detail are separate screens
  with a real back affordance, because a three-pane layout crushed into 375px
  is how messaging apps become unusable.
- **The composer is anchored to the safe area,** above the home indicator.

## Visual Language

The palette is functional. Three encodings do all the work, and each encodes
something structurally true rather than something decorative.

### Channel identity is an edge, never a fill

Each channel descriptor carries an accent. It is applied as a 2px leading edge
on a thread row and as the composer's focus ring — never as a filled chip
background. A list of filled, differently-coloured chips is unreadable at
roster density, and it also makes every new channel a colour-conflict problem.
The accent comes from the descriptor, so adding a channel adds a colour without
touching the console.

### Agent identity is the descriptor's own colour triple

`prompt-manager` agent descriptors already carry `appearance.body`,
`appearance.head` and `appearance.accent`. The avatar renders from those exact
values, so the roster is scannable by shape and the console never invents a
second identity for an agent that already has one. Deriving an avatar from a
name hash instead would make the console and the descriptor disagree.

### Trust tier is a rank, not a category

Four tiers — stranger, known, trusted, owner — are ordered, and the room
ceiling is a *minimum across them*. So they are drawn as an ascending ladder of
filled steps, not as four arbitrary colours. A person must be able to see that
`known` is below `trusted` without reading either word, or the ceiling rule
cannot be understood.

`stranger` is styled as the ordinary default, not as an alarm. Every
unrecognised sender is a stranger by design; styling that as a fault would make
the normal case look broken.

### Everything else is neutral

Semantic colour is reserved for genuine severity: a failed delivery, an expired
gate, a channel that stopped working. Nothing else competes with those. Numbers
that change — budgets, counts, durations — use tabular figures so they do not
jump as they update.

### Density

- **Thread:** comfortable. Generous line height, 65–75 character measure,
  messages grouped by consecutive author with a date divider between days.
- **Console lists:** dense. Sticky headers, a clear selected row, bulk
  affordances where a bulk action exists.
- **Governance controls:** deliberately loose. The grant section on an agent
  gets more space than its information density warrants, because spacing is how
  a design says *slow down* without a warning dialog.

## Motion

Motion is used in exactly four places, each carrying information:

1. **A message arriving** — a short rise and settle, so a new message is noticed
   without the transcript jumping.
2. **A state transition that changes what is possible** — a composer becoming
   disabled, a gate resolving, a channel going offline.
3. **The listening indicator in call mode** — the only ambient animation in the
   product, because it is the only place where continuous liveness is the
   information.
4. **Optimistic send** — a message appears immediately and settles when
   acknowledged; a failure animates back to a retryable state rather than
   vanishing.

Everything else is instant. Under `prefers-reduced-motion` all four degrade to
state changes without movement, and the listening indicator becomes a static
indicator that still distinguishes listening from not.

## The State Contract

Every asynchronous surface declares its states, and every state is designed
rather than inherited. The canonical vocabulary is in
`path:../../DESIGN.md`; the per-page declarations are in `experience/pages/`.

Four rules that are easy to get wrong and expensive to get wrong:

**Empty is not an error.** An empty region is perceivably distinct from a failed
one and from a loading one. Where empty means nothing needs attention, it says
so plainly.

**Partial keeps working.** One failed region names what failed and leaves every
other region interactive. A page does not become unusable because a sidebar
could not load.

**Stale is labelled, never hidden.** A cached figure carries its age. No number
is shown without its freshness when the read is stale.

**A disabled control states why.** A composer disabled because the adapter is
offline says that, and says what would re-enable it. A disabled control with no
reason is indistinguishable from a bug.

## Failure And Refusal Copy

This product refuses things constantly, by design. Refusal copy is therefore a
first-class design surface, not error-handling debris.

Every refusal answers three questions in this order:

1. **What was withheld** — the specific capability, never "permission denied".
2. **Why** — the tier, the ceiling, or the budget that caused it.
3. **What would unblock it** — an action, and who can take it.

> Not in this room — Sam is `known`, and account detail needs `trusted`.
> Matt, I can send it to you directly.

That is the target register: plain, specific, addressed to a person, offering
the next move. Not:

> ~~Error: insufficient permissions (403)~~

The same rule applies to a group. `OT-P1-002` forbids the partial answer, so
the refusal itself must carry the information — the agent says what it will not
say, rather than quietly answering less.

## Accessibility

Treated as correctness, not as a checklist.

- **The transcript is a `log`.** Arriving messages are announced politely and
  never steal focus from the composer.
- **Authorship is never colour alone.** Nor is trust tier, delivery state,
  channel identity, or severity. Every one has a second cue.
- **Focus returns.** Answering or dismissing a capability gate returns focus to
  the composer. Closing a dialog returns focus to what opened it.
- **Call controls are large and always reachable.** Minimum tap target at every
  viewport, and present in every call state.
- **Both themes are designed.** Dark is not an inversion. Every claim about
  distinguishability carries a `dark-parity` sibling in `experience/`.

## How This Is Enforced

| Layer | Where | What it catches |
|---|---|---|
| Typed claims | `experience/pages/*.json`, `experience/components/*.json` | Stated intent, tiered `machine` / `manual` / `aspirational` |
| Spec validation | `experience-manager spec validate switchboard` | Reference resolution, region bindings, required-state coverage |
| Journeys | `experience/journeys/*.json` | That the surfaces compose into a path someone can actually walk |
| Case scaffolding | `bas/` | Reads `bindings` as the selector source of truth |

Claims are currently `aspirational` and `manual` because the UI does not exist
yet and machine-tier claims need stable selectors. Promote a claim to `machine`
when the surface it describes is built and its binding is stable — that
promotion is the intended way this contract tightens over time, and it is
tracked as depth rather than as debt.

## Cross-References

- `path:../../experience/README.md` — the depth ladder and validation command
- `path:../../DESIGN.md` — the design contract and UX-state vocabulary
- `path:./UI-ARCHITECTURE.md` — the `ui/` source tree and slot taxonomy
- `path:../reference/component-library-gaps.md` — what the shared component
  library does not yet provide for this UX
- `path:../../PRD.md` — the operational targets each page claims against
