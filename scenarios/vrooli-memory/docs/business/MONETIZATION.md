# Monetization — Vrooli Memory

> **Status: draft hypothesis — a real monetization path exists.**
>
> An earlier revision of this document marked the scenario `not-applicable`
> on the grounds that it is internal infrastructure. That was wrong. The
> harness-agnostic design — prompt block, CLI, and generated file projection —
> means this capability **works without Vrooli**, for anyone running more than
> one coding agent. That is a product, and there is a market for it.

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story — but so is naming a real one
when it exists.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

- **Direct product: candidate, offered free.** The strongest standalone
  candidate the fleet has produced, because its value does not depend on any
  other Vrooli scenario being present. The intended posture is a **free local
  tool**, not a paid one.
- **Internal capability: yes, and first.** It replaces per-harness memory and
  the hand-curated index for this project. Internal use produces the evidence
  for the external case — dogfooding is the validation plan.
- **Revenue line: indirect — inference demand, not a licence.** See below.

## The Monetization Path: Inference Demand, Not a Licence

The revenue mechanism is **not** a paid tier of this scenario. It is the
inference this scenario consumes, routed through ai-gateway's existing
three-way fallback:

```
  1. Local models on the user's machine     → free, preferred, no revenue
  2. User's own provider API key (BYOK)     → free to us, no revenue
  3. Vrooli-provided hosted inference       → the revenue surface
```

A user whose machine can run local models never pays, and that is the intended
default. A user with their own key never pays. A user with neither — modest
hardware, no provider account, wants it to just work — falls through to a paid
option.

**Why memory is an unusually good demand driver.** In most tools, inference is
per-user-action and stops when the user stops. Here, **compaction is recurring
background inference proportional to how much has been remembered.** The load
does not go away when the user is idle, and it grows with the corpus. A memory
system generates inference demand indefinitely, which is a structurally
different revenue shape from a per-request tool.

**What exists and what does not.** The routing is real and shipped in
ai-gateway: `OT-P0-002` keeps provider credentials in the provider resource
(so BYOK is the existing model, and ai-gateway handles no secrets itself),
`OT-P1-004` enforces local-only / local-first / cheap-first / max-cost policies,
and `OT-P1-008` does capacity-aware local routing with policy-respecting remote
fallback. **The Vrooli subscription as a third option is settled strategy but
unbuilt runtime.** `path:../../docs/monetization/evidence/FINANCIAL_MODEL.md` (Tier 1)
already states that paid subscriptions include the integrated gateway with a
credit allowance and that *"that IS the core reason to pay rather than running
the OSS apps with bring-your-own keys."* What does not exist is enforcement:
nothing tracks an allowance, identifies a subscriber, or refuses a call when an
allowance is exhausted. That gating work belongs to ai-gateway, not here.

**What this scenario owes the path.** Nothing but adoption. It builds no
billing, no entitlement checks, no tenancy. It routes inference through
ai-gateway as designed and lets platform policy decide where the call lands.
The honest counterweight: the users most likely to fall through to paid
inference are those whose machines cannot run local models, which correlates
with lower-spec hardware and plausibly with lower willingness to pay. Conversion
is not obviously aligned with ability to pay, and that should be measured rather
than assumed.

## Customer / Buyer

- **Primary user:** a developer running two or more coding agent harnesses —
  Claude Code, Codex, Cursor, Copilot, Gemini CLI, Windsurf, Aider. Multi-tool
  use is the norm rather than the exception, and each tool's memory is a silo.
- **Buyer (hosted inference):** the same developer, at the moment their machine
  cannot run local models and they have no provider key of their own. This is
  the only buyer in the primary plan.
- **Pain, in order of how sharply it is felt:**
  1. Memory does not follow you between tools. Something explained to one agent
     is unknown to the next.
  2. Each harness's memory is shallow — a flat file that grows until it is
     mostly noise, with no retrieval beyond the model reading the whole thing.
  3. Memory does not survive tool churn. Switching agents means starting over.
  4. *(Team-scale duplication — every engineer's agent re-learning the same
     things — is a real pain but is deliberately out of the primary plan. See
     the expansion note under Packaging.)*
- **Existing alternatives:**

  | Alternative | What it is | Why this is different |
  |---|---|---|
  | mem0, MemOS, Letta, EverMind | Memory layers for agents **you build**. | Different buyer — an app developer, not a coding-agent user. They sell an SDK; this sells a working setup for tools you already run. |
  | OptMem | Free, elegant, single-context. Regex recall, agent-performed compaction, one linear timeline. | Semantic recall instead of regex; background compaction instead of agent labor; multi-domain instead of one timeline. Its existence proves demand; its limits are the product gap. |
  | Each harness's built-in memory | Free, native, zero setup. | **The real competitor.** Free and already there. The pitch has to be that it stops working at scale and never crosses tools — not that it is bad. |
  | Doing nothing / re-explaining | Free, universal. | The true default. Most of the market does not perceive this as a problem yet. |

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Free standalone app + hosted inference fallback | **primary plan** | Local-first install: a binary plus the prompt-block installer. Works with any harness that reads a memory file, which is all of them. Fully functional and free on local models or BYOK; revenue only when a user falls through to hosted inference. Requires decoupling from the Vrooli lifecycle — see Preconditions. |
| Bundle component | plausible | Component of a broader developer-tooling SKU. |
| Add-on | unlikely | The capability is more foundational than an add-on to something else. |
| Team / shared memory | **speculative expansion — may never be built** | See below. Deliberately *not* the revenue case. |
| Service/consulting assist | deferred | Not a starting motion. |

### Team / shared memory — a potential expansion, not the plan

Recorded so the option is visible, not because it is intended.

What has been designed works well **for a single machine**, and that is the
thing worth getting right. Sharing across a team is a materially different
product, and harder than it first appears — not primarily for infrastructure
reasons but because **you would only want to share certain kinds of memory.**
Project gotchas and environment facts are worth sharing; personal preferences,
half-finished threads, and anything from an operator's personal domains are
not. That is a per-memory sharing policy, which is a real product surface on
top of the tenancy work.

It would also require reopening decision **D-005** (unified read, no
access-control partitioning), which is currently correct and settled for
single-machine use.

**Status: P2-at-most, and quite possibly never.** It should not appear in the
launch motion, it should not shape the architecture, and no P0/P1 work should
be justified by it. It is here as a documented direction of possible expansion
and nothing more.

## Pricing Hypothesis

- **Model:** free tool, indirect revenue. The scenario itself is free and fully
  functional on local models or BYOK. Revenue accrues at the platform level when
  a user's inference falls through to Vrooli-provided hosting. No per-seat, no
  licence, no feature gating of the memory capability itself.
- **Comparable products:** mem0, MemOS, Letta (agent-memory infra, SDK buyers);
  developer-tool subscriptions in the $10–30/seat/month range as the anchor for
  what a coding-adjacent tool commands.
- **Willingness-to-pay evidence: none captured.** This is a hypothesis. Its
  strength is that it never asks anyone to pay for the *tool* — only for
  inference they cannot otherwise run, at the moment they need it. Its weakness
  is stated above: fall-through correlates with weaker hardware, which may
  correlate with lower ability to pay.
- **Cost drivers:** embedding and summarization inference. Local models make the
  default genuinely free at no cost to us. **Compaction is the recurring cost,
  not writes** — it scales with corpus size and continues while the user is
  idle, so hosted-tier pricing must be modelled on retained corpus, not on
  activity.

## Preconditions (what has to be true before this can be sold)

Recorded because they are the actual work, and one of them contradicts a
settled architectural decision.

1. **Lifecycle decoupling.** The scenario currently assumes the Vrooli lifecycle
   (`.vrooli/service.json`, `make start`). A standalone distribution needs to run
   without it.
2. **ai-gateway packaged, not substituted.** Inference routes through
   ai-gateway by design (`INTEGRATIONS.md`), and that is also what carries the
   revenue path — so a standalone distribution should **ship the gateway**
   rather than bypass it. Replacing it with a direct local-model call would
   remove the monetization surface entirely.
3. **Allowance enforcement must exist in ai-gateway.** The strategy is settled
   (Tier 1 in the financial model); the runtime is not. Per-request max-cost
   ceilings ship today, but a per-user-per-period allowance does not. Platform
   dependency, not work this scenario can do.
4. **Internal proof first.** Nothing here should be pursued before the P0 loop
   is real and this project has been running on it long enough to know whether
   memory quality holds. Selling a memory product whose recall is mediocre is
   worse than not selling one.

## Validation Plan

- **Demand signal needed, in order:**
  1. Internal: the fleet actually runs on this and the hand-curated index stops
     being maintained. If that does not happen, nothing else matters.
  2. External interest in the standalone framing — measurable cheaply by
     publishing the approach before building any commercial surface.
  3. **Fall-through rate** — what share of real users cannot run local models
     and have no key of their own. This single number decides whether the
     inference path is a business or a rounding error, and it is measurable as
     soon as there are users.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** to be set against the project-level monetization
  taxonomy. Not inventing a number here.
- **Kill signal:** if internal dogfooding shows recall quality is not clearly
  better than a harness's built-in memory, the product case collapses regardless
  of market interest — the differentiator is quality at scale, not features.

## Current Status

`draft` — a real hypothesis with a named buyer, named comparables, and stated
preconditions. Not validated. The tool is free by intent; revenue is indirect,
accrues at the platform level, and depends on a hosted-inference tier that does
not exist yet. Team/shared memory is documented as possible expansion only and
is explicitly not the revenue case — decision **D-005 stays closed**.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-005 (unified read, no partitioning) stays closed under this plan; D-014 records the revenue mechanism
- [`../concepts/REPLACEMENT.md`](../concepts/REPLACEMENT.md) — the harness-agnostic design that makes a standalone product possible
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:../../docs/monetization/README.md`.
