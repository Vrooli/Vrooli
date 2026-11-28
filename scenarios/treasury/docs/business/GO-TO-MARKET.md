# Go To Market — Treasury

This document records how this scenario would reach the people who need
it, and what evidence would justify investing further. It is a hypothesis
document. Channel strategy at the portfolio level is operator canon; this
is a scenario-local proposal.

## Audience And Positioning

- **Audience:** operators running agents that need to spend money —
  self-hosters, solo founders, and small teams — who will not hand a
  treasury to a hosted vendor. Secondarily, scenario and tool authors who
  want to charge per call without first building accounts, card vaulting,
  subscriptions and invoicing.

- **Positioning:** *the spend-control layer you run yourself.* The
  agentic-payments category is crowded and funded, and every entrant
  custodies your money in exchange for convenience. This sits in the one
  position they structurally cannot occupy, because their business model
  is the custody.

- **Main claim:** an agent can be given spending power without being given
  a credential. Authority becomes a signed object with a cap, a
  counterparty scope and an expiry — and the agent that read the merchant
  page never holds the decision.

- **The sharper claim, for the technical reader:** the approval gate is
  not a permission check an injected prompt might argue past. The
  agent-facing service **declares no method** that can change policy.
  That is a proto-descriptor assertion, not a runtime rule. This is the
  claim most likely to land with the people who already understand why
  prompt injection makes ordinary authorization insufficient.

- **Proof needed:** a working end-to-end loop — an agent proposes, a human
  approves on a phone, money moves, the ledger records it, and the evidence
  replays afterwards — plus a demonstration that a compromised agent
  cannot raise its own limit. The second half is the one that converts
  skeptics, and it demos in about fifteen seconds.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Self-hosting and homelab communities | The buyer who refuses hosted custody is already visibly self-hosting everything else. This is the highest-fit, lowest-cost channel. | A one-command local setup, a short demo of the injection-resistance claim, and honest limits documentation. | Installs that proceed past setup to a first issued mandate. |
| Agent-building developer communities | People shipping autonomous agents hit the "it cannot buy anything" ceiling directly and remember it. | A worked example of an agent buying an API top-up under a mandate; the two Connect services explained. | Inbound questions about rails, which indicate someone got far enough to plan a real purchase. |
| Technical writing on the injection problem | The structural argument — absence of an RPC beats a permission check — is a genuinely useful idea independent of this product, which makes it worth reading rather than an ad. | One well-argued piece with the proto-descriptor test as its concrete artefact. | Referrals from people who came for the argument. |
| x402 and agentic-payments ecosystem | Being a self-hostable facilitator plus a policy layer is a distinct contribution to an ecosystem currently dominated by hosted services. | A working self-hosted facilitator and a clear statement of what is and is not custodied. | Ecosystem listings and integrations that appear without being asked for. |
| Vrooli bundle | Existing operators get spend governance as part of a money story that already includes `money-ledger` and `offer-desk`. | Bundle wiring only; the scenario needs no separate marketing here. | Attach rate among operators who already run the other two. |
| Paid acquisition | Rejected for now. | n/a | The buyer is a skeptical technical self-hoster, which is the audience least reachable and least persuadable by ads. |

## Launch Motion

1. **Ship the free spine and say so loudly.** The mandate contract,
   budgets, approval, evidence and the manual rail move no money and cost
   nothing to run. An operator can adopt the whole safety story before
   trusting the project with a single payment. This is the strongest
   available first step precisely because it asks for no trust.
2. **Publish the injection-resistance argument** with the descriptor test
   as its artefact. Lead with the idea, not the product.
3. **Add x402 in both directions.** Cent-scale amounts make the first real
   transaction genuinely low-stakes, and the earning half is a second
   reason to install that has nothing to do with spending.
4. **Add the scoped card rail**, which is the point the product becomes
   useful for ordinary purchases rather than machine-native ones.
5. **Only then discuss pricing.** Every chargeable item is convenience over
   a free capability, so the paid tier is legible rather than extractive —
   but only if the free tier visibly works first.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Give an agent spending power without giving it a credential." | All | The mandate contract; `TRS-P0-001`. | hypothesis |
| "The agent cannot turn off its own approval gate — the method does not exist." | Technical / security-minded | The proto-descriptor assertion; `TRS-P0-004`. | hypothesis |
| "Your agents, your card, your keys, your policy, your box." | Self-hosters | The whole architecture; `TRS-P0-010`. | hypothesis |
| "Every refusal tells you which constraint refused it." | Operators | Evidence records covering declines and expiries; `TRS-P0-009`. | hypothesis |
| "The same rail that spends can earn." | Scenario and tool authors | x402 in both directions; `TRS-P1-002`. | hypothesis |
| "It fails closed. If we cannot verify who is asking, nothing is spent." | Security-minded | `TRS-P0-005`. | hypothesis |

**Messages deliberately not used:** anything implying an agent can buy
anything autonomously. Identity-bound purchases — anything requiring a
government ID, a company registration, or a biometric — cannot be
completed by a machine, and claiming otherwise would set up the exact
disappointment that loses a technical audience permanently. The honest
framing is that the agent prepares everything and hands back the one step
only a human can take.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Does the free spine get *used*, not just installed? | Self-hosting communities | Operators who issue a mandate and let an agent spend under it, rather than continuing to buy by hand. | If installs are high and mandates near zero, the ceiling is trust in agents rather than absence of a wallet — and this scenario is not the bottleneck it assumed. |
| Is self-hosting a requirement or a preference? | Direct conversation | Whether operators who evaluate this also seriously evaluated a hosted alternative. | If it is only a preference, the incumbents win on convenience and the positioning needs rethinking before more investment. |
| Does the injection argument travel on its own? | Technical writing | Readers arriving from the argument rather than from product channels. | If the idea travels, lead with it permanently. If not, the technical positioning is weaker than assumed. |
| Does earning drive adoption independently? | Developer communities | Installs whose first action is metering an endpoint rather than issuing a mandate. | If earning pulls its own users, it deserves equal billing rather than being framed as the second half of spending. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the positioning decisions and their reasons
- Project-level strategy is operator canon: `path:docs/monetization/strategy/STRATEGY.md`.
