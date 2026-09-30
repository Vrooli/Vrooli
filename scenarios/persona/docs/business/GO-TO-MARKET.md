# Go To Market — Persona

Scenario-specific positioning and launch motion. Portfolio strategy is
operator-curated canon in `docs/monetization/`; this records how this
scenario reaches the people it is for.

## Purpose Of This Document

Use this document to answer:

- Who is this for and how is it positioned against alternatives?
- Which channels reach them?
- What is the launch motion?
- What messaging is true, and what would be overclaiming?
- Which experiments would falsify the positioning?

## Audience And Positioning

**Positioning statement.** Persona gives an agent a declared identity to
act as — an address it controls, a route to the codes it cannot receive,
and a pointer to the documents it must never hold — with every action
traceable to the human who authorised it, running entirely on your own
machine.

**Against hosted KYA products** (Skyfire, Vouched, and the DIF KYA-OS
ecosystem): they model agent identity well and hold your material. The
differentiator is custody, not modelling. Do not claim better identity
semantics; claim that the passport never leaves the box.

**Against a password manager plus discipline** — the real incumbent for
most operators today: a password manager has no delegation chain, no
attribution, no OTP contract an agent can call, and no concept of a step
a machine must not take. Persona is not a better vault; it is the thing
that makes a vault usable by an agent without handing the agent the
vault.

**Against doing nothing**: the honest alternative for many. The cost of
nothing is that agents either cannot complete outbound work at all, or
complete it using the operator's own personal identity with no record of
having done so. The second is worse than the first.

## Channels

| Channel | Fit | Notes |
|---|---|---|
| Vrooli ecosystem (bundled with agent purchasing) | **Strongest** | The natural motion. Persona is what makes `treasury` usable for anything beyond machine-native payment. |
| Self-hosting and homelab communities | **Strong** | Custody-sensitive by disposition, and already running their own mail. The audience most likely to understand the wedge without explanation. |
| Agent-builder and AI-engineering communities | Medium | Large and interested, but mostly building demos where a real identity is not yet the blocker. |
| KYA / decentralised-identity standards community | Medium | Credible venue for the attestation work at `PSN-P1-007`, and a source of correction on where the standard is going. Not a buyer channel. |
| Compliance and audit buyers | Weak for now | The attribution story is genuinely relevant, but the scenario needs a real deployment history before that conversation is honest. |

## Launch Motion

1. **Ship with a consumer, not alone.** Persona's value is invisible in
   isolation; it becomes obvious the moment an agent cannot finish a
   signup. Launch alongside the first flow that visibly needs it.
2. **Lead with the wall, not the wallet.** The demo that lands is an
   agent getting to the end of an enrolment, stopping at the ID check,
   and handing the operator one pre-filled action — because everyone
   who has tried this has hit exactly that.
3. **Publish the custody split as the argument.** The table of what this
   scenario deliberately does not hold is more persuasive than any
   feature list, and it is checkable in the source.
4. **Be first to say what it will never do.** No CAPTCHA solving, no
   synthetic identity, no impersonation. In a category with obvious
   grey-area demand, refusing loudly is a positioning asset rather than
   a limitation.
5. **Only then** consider standalone KYA positioning, which is a
   different product motion and a separate bet.

## Messaging

**True and defensible:**

- The passport never enters this scenario; it stays in `document-manager`
  with its own custody journal, and is released only into a named
  handoff.
- Every action names the authorising human, the persona, and the run —
  including the actions that were refused.
- If the verification authority is unreachable, the answer is no. There
  is no flag to change that.
- A blocked step becomes one pre-filled action for a person, not a stack
  trace.

**Overclaiming — do not say:**

- "Fully autonomous purchasing." Identity-bound enrolment is not
  automatable and this scenario is built on admitting that.
- "KYC" or "identity verification." It verifies nothing about a human;
  it records who authorised what.
- "Compliance-ready" or any regulatory badge. The attribution record may
  support a compliance process; it does not constitute one.
- "Secure by default" as a bare claim. The specific, checkable claims
  above are stronger than the generic one.

## Validation Experiments

| Experiment | Falsifies | Success Signal |
|---|---|---|
| Show the custody-split table to five target operators without other context | That custody is the wedge | They name it unprompted as the reason to look further |
| Run one real Tier-C enrolment end to end with handoff | That the handoff model matches reality | The human step takes under two minutes and requires no context reconstruction |
| Offer standalone before building P1 | That it is a standalone product | Any inbound interest without `treasury` in the pitch |
| Post the "what it will never do" list publicly | That refusing is a positioning asset | Positive engagement rather than requests for the grey-area features |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — value promise and appendix
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — the claims that must stay true
