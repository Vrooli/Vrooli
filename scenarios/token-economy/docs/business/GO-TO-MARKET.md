# Go To Market — Token Economy

This document records how the scenario would reach the people it is for, and
what evidence would justify investing further. It is a hypothesis document, not
a commitment; bundle placement and pricing are operator-curated canon
(`path:docs/monetization/README.md`).

## Purpose Of This Document

Use this document to answer:

- Who is this for, and how would they describe their problem?
- Where do those people already are?
- What is the launch motion, and what is deliberately not attempted?
- What experiments would confirm or kill the hypothesis?

## Audience And Positioning

**Audience.** Parents who already run a chore-and-reward arrangement informally
— a whiteboard, a jar, a note on the fridge — and want it to keep track of
itself. Secondarily: parents currently paying for an allowance app who resent
that it required a bank account and a card for a nine-year-old.

**Positioning, in one line.** *A household economy that runs on your own
machine, where the rewards can be anything you decide.*

**The two things that are actually different:**

1. **Rewards without a price.** Every alternative pays out in dollars because
   every alternative is a bank product. This one redeems against a catalog the
   household writes — screen time, a trip, choosing dinner, a chore traded
   between siblings. Those are the rewards parents actually use, and no
   card-based product can express them.
2. **Nothing leaves the machine.** A behavioral record of a child — what they
   did, what they wanted, when — never reaches a third party, because there is
   no third party. That is the same wedge `document-manager` makes for
   documents, and it is credible here precisely because the product needs no
   processor, no analytics vendor, and no cloud sync to function.

**What we do not claim.** Not a banking product, not financial education, not
parental controls. The scenario records that screen time was redeemed; it does
not manage a device. Overreaching into any of those turns a small honest
product into a worse version of an existing one.

## Channels

| Channel | Fit | Why | Cost |
|---|---|---|---|
| Self-hoster and homelab communities | **strongest** | The audience already runs a machine, already values local-first, and already distrusts hosted services holding family data. The privacy claim lands without explanation here. | Low. Organic, community-shaped. |
| Vrooli ecosystem / bundle | **strong** | Existing operators get the composition story free: any scenario becomes an earning surface. This is the only channel where the compound-value claim is legible. | Low. Bundle placement is operator canon. |
| Parenting communities and forums | **medium, and delicate** | The pain is real here but the audience is not technical. Self-hosting is a hard prerequisite, and leading with it loses the room. | Medium. Requires a genuinely easy install story that does not exist yet. |
| App stores | **poor fit today** | The product assumes a household instance. A store listing implies a hosted service, which is the thing we are deliberately not building. | High, and it would require abandoning the central claim. |
| Paid acquisition | **not attempted** | Buying users before the retention question is answered would purchase a false signal. | High. |

## Launch Motion

Deliberately small and sequential, because the central question is behavioral
rather than commercial.

1. **One household, unassisted.** The operator's own, running the P0 loop.
   The success condition is behavioral: does the child open the holder view
   without being told to? If the loop only works when an adult drives it, the
   product is a chore and the design is wrong.
2. **Three to five self-hosting households.** Recruited from the ecosystem, not
   from a market. What matters here is whether the catalog fills up with
   rewards the product's authors never imagined — that is the signal that
   "rewards without a price" is real and not a designer's preference.
3. **Write up the mechanism, not the product.** The append-only journal, the
   two-audience design, and the isolation boundary are interesting to the
   self-hosting audience on their own terms. Content that teaches lands better
   than content that sells in this channel.
4. **Only then consider packaging.** Bundle placement and pricing are requested
   after retention exists, never before.

**What the launch does not do:** no waitlist, no landing page before the loop
works, no pricing page before willingness to pay is measured, and no marketing
to children at any point.

## Messaging

**To a self-hoster:** "Your household's chore-and-reward system, running on your
own box. No bank, no card, no KYC, and no company holding a record of what your
kid did this week."

**To a parent already paying for an allowance app:** "You linked a bank account
and issued a card to a nine-year-old so they could earn five dollars. This does
the same job without any of that — and it can reward things that don't have a
price."

**To a Vrooli operator:** "Any scenario you already run can become a way to
earn. And the policy model it uses is the same one `treasury` uses for real
money, tested where being wrong is free."

**Tone rules.** Plain and non-condescending. Never gamified, never urgent,
never implying that a household without this is failing at parenting. The copy
never suggests tokens are money or convert to it — that is the constraint the
product is built on, and messaging that blurs it would undermine the thing that
keeps multi-holder balances clear of money transmission.

## Validation Experiments

| Experiment | Question | Signal that confirms | Signal that kills |
|---|---|---|---|
| Own household, four weeks | Does the loop survive contact with a real child? | The child opens the holder view unprompted and initiates redemptions. | Only the adult ever touches it; the product is a chore with extra steps. |
| Catalog content across 3–5 households | Is "rewards without a price" real? | Catalogs fill with non-monetary rewards the authors never anticipated. | Every household writes cash amounts, in which case a card product is genuinely better and this one should not be a consumer product. |
| Approval-queue latency | Does the gated posture work socially, or does it just create friction? | Approvals are decided in hours and the pending state is understood without complaint. | Requests rot in the queue; the posture defaults to instant and the feature was theater. |
| Earning-adapter uptake (post-`TKE-P1-009`) | Does composition matter, or is it an engineer's argument? | Operators wire their own scenarios as earning surfaces unprompted. | Nobody connects anything; earning stays manual, which is fine but removes the ecosystem advantage from the pitch. |
| Willingness to pay | Would anyone pay for reach and backup? | Unprompted asks for cross-device access or hosted relay. | Indifference — in which case the product stays free and earns its keep internally. |

**The honest framing:** the internal role (rehearsal surface for `treasury`'s
policy model) is already sufficient justification to build this. The consumer
hypothesis is upside, and it should be allowed to fail cleanly rather than be
propped up.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging, pricing hypothesis, validation thresholds
- [`../../PRD.md`](../../PRD.md) — value promise and the competitive scan in the appendix
- [`../concepts/DATA.md`](../concepts/DATA.md) — the privacy posture underpinning the second differentiator
- [`../../DESIGN.md`](../../DESIGN.md) — the two-audience adaptation
- Project-level monetization strategy: `path:docs/monetization/README.md`
