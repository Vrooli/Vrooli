# Go To Market — Switchboard

## Purpose Of This Document

Who this is for, what it claims, where those people are, and how the claim is
tested. Positioning only — pricing and bundle membership are canon under
`path:docs/monetization/`.

## Audience And Positioning

**The one-sentence claim:** your agent, on your phone number, on your machine.

**Positioning against the category.** Hosted iMessage and SMS agent products
exist and are roughly fourteen months old. Every one of them routes the owner's
conversations through a vendor's servers. Vrooli runs on the owner's own
hardware, reaches Apple through the owner's own Mac over the owner's own fleet
link, and keeps the thread on the owner's disk. *"Your agent, your machine, your
messages"* is a sentence the hosted competitors structurally cannot say — and it
is the exact objection those products' own users voice publicly.

**The second differentiator, which is larger but slower to explain.** Competitors
sell a chat interface with a tool list. This sells a chat interface onto a
self-improving ecosystem: every scenario ever shipped becomes something the agent
can already do, with no integration written. The message is a front door; the
house is the product.

| Segment | What they want | What blocks them today |
|---|---|---|
| Private-by-default operators | An agent that does not phone home | Nothing — this is the beachhead |
| Power users of thin chatbots | An agent that can actually act | Nothing technical; a credibility problem |
| Owners exposing an agent to others | Safe delegation to family, a team, customers | `SWBD-PROB-001` — runtime injection defence is unowned |
| Owners without a Mac | iMessage without buying hardware | Only a hosted relay solves it, and that forfeits the core claim |

## Channels

| Channel | Why it fits | Notes |
|---|---|---|
| The product itself | An agent in a group thread is seen by everyone in the room | Group support is the only organic distribution this product has, which is a commercial argument for building it, not just a feature request |
| Developer and self-hosting communities | The privacy objection is already articulated there, loudly | The claim lands without education |
| Vrooli's own surfaces | Existing operators already have the ecosystem the agent reaches | Lowest-friction adoption path |
| Demonstration over description | The product is hard to describe and immediate to show | A recorded thread where the agent refuses correctly is more persuasive than one where it succeeds |

## Launch Motion

Sequenced so that each step is provable before the next is claimed.

1. **Prove the loop internally.** In-app plus Telegram, one agent, one real
   thread. No claims made externally.
2. **Prove the governance.** A second person, a lower tier, an observed refusal.
   Until this works, the product is a single-user toy and should be described as
   one.
3. **Attach iMessage.** The differentiator becomes demonstrable — and it is only
   a differentiator because it runs on the owner's own Mac.
4. **Open to owners exposing agents to others.** Gated on `SWBD-PROB-001` being
   closed. Not before.
5. **Number provisioning as the paid convenience.** The first genuinely gated
   capability, and the honest one.

## Messaging

**Say:**

- Your agent, on your phone number, on your machine.
- It reaches everything Vrooli can do, not a list of integrations.
- You choose who reaches it and how far they get.
- It refuses out loud, and tells you what would unblock it.

**Do not say:**

- "Secure" or "private" as adjectives without the mechanism. The claim is
  specific — the thread is on your disk and the Mac is yours — and specificity is
  what makes it credible against competitors who also say "private".
- Anything implying safe exposure to untrusted senders while `SWBD-PROB-001` is
  open. The trust ceiling holds; defence *within* a tier does not exist yet, and
  overclaiming it is the fastest way to earn a real incident.
- Latency or voice-quality claims. Speech currently runs on CPU because
  `engine_id` is never sent (`SWBD-PROB-002`).
- "Works with iMessage" without the Mac requirement stated in the same breath.

**Tone.** Plain and exact. The product's most important sentences are refusals,
and the marketing should sound like the product: state what is withheld and what
would unblock it, rather than reassuring.

## Validation Experiments

| Hypothesis | Experiment | Falsified if |
|---|---|---|
| Custody is the deciding purchase reason | Ask new owners why they chose it, unprompted | Privacy is rarely mentioned unprompted |
| The zero-setup path converts better than channel-first | Compare completion of the `first-agent` journey against a channel-first variant | Channel-first converts at least as well |
| Groups drive acquisition | Track new owners whose first exposure was somebody else's group thread | Effectively none arrive that way |
| Depth is felt, not just claimed | Measure how many distinct scenarios a retained owner's agent actually reaches | Retained owners use one or two capabilities, in which case a thin chatbot is a sufficient competitor |
| iMessage justifies the Mac requirement | Attach rate for iMessage among owners who have a Mac available | Most owners stop at Telegram |

The fourth row is the one worth taking seriously: if it fails, the ecosystem
argument is wrong and the product is competing on custody alone.

## Cross-References

- `docs/business/MONETIZATION.md` — packaging and meter classes
- `path:docs/monetization/` — canon: strategy, catalog, pricing (read-only)
- `docs/internal/PROBLEMS.md` — `SWBD-PROB-001` and `SWBD-PROB-002`, both of which constrain claims
- `path:docs/concepts/ECOSYSTEM.md` — role, interfaces, and the objective served
- `PRD.md` — the operational targets behind each launch step
