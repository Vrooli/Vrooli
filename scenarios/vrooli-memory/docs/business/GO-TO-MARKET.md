# Go To Market — Vrooli Memory

> **Status: draft hypothesis.** An earlier revision marked this
> `not-applicable`. That was wrong — the harness-agnostic design works without
> Vrooli, so there is a real external audience. See
> [`MONETIZATION.md`](MONETIZATION.md) for buyers, comparables, and the
> preconditions that gate the team tier.

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **Audience:** developers running two or more coding agent harnesses. The
  qualifying signal is multi-tool use, not team size — a solo developer using
  Claude Code and Cursor has the problem; a ten-person team standardized on one
  tool feels it far less.
- **Offer:** the tool is **free**. Revenue is indirect, through hosted inference
  for users who can run neither a local model nor their own API key. Nothing
  about the memory capability is feature-gated.
- **Positioning:** *memory that follows you between coding agents.* Positioned
  against tool lock-in and context loss, **not** against any individual
  harness's memory feature — that framing invites comparison with something
  free and native, which is a losing argument.
- **Main claim:** what one agent learns, every agent knows — and it keeps
  working at ten thousand memories.
- **Secondary claim:** your memory outlives the tool you are currently using.
- **Proof needed:** retrieval quality at a corpus size where a flat memory file
  has visibly stopped working. That is the demo, and it cannot be faked with a
  small corpus — which is why internal dogfooding has to come first.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Open-source release | This category's users adopt tools they can read and run locally before paying for anything. OptMem's reception is direct evidence for this channel. | Working repo, install script, prompt block, honest README about limits. | Installs and issues from people running more than one harness. |
| Architecture write-up | The frontier-agglomerative design, the facet decay-law argument, and the 1-in-200 measurement are genuinely novel and travel on technical merit. | The design record already written for this scenario. | Inbound interest in the standalone framing specifically. |
| Coding-agent communities | Users who already feel memory fragmentation congregate around each harness. | Setup guide per harness. | Whether people ask for the *team* version unprompted — the highest-value signal available. |
| Vrooli capability catalog | Existing internal surface; lowest-cost distribution. | Already covered by scenario docs. | Internal adoption. |

## Launch Motion

1. **Dogfood.** Run this project on it until the hand-curated index is no
   longer maintained. If that never happens, stop — the product case fails
   internally before it can fail externally.
2. **Publish the approach** before building any commercial surface. Cheapest
   possible read on whether the standalone framing lands.
3. **Open-source the single-user local engine.** The free tier is the
   distribution channel, not a giveaway.
4. **Measure fall-through.** What share of real users can run neither a local
   model nor their own provider key. That number, not team interest, decides
   whether the inference path is a business.
5. **Ship the gateway with it.** A standalone build must package ai-gateway
   rather than bypass it — bypassing removes the revenue surface entirely.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| Memory that follows you between coding agents. | Multi-harness developers | Working cross-harness projection | untested |
| Still works at ten thousand memories. | Anyone whose memory file has gone stale | Retrieval quality at scale vs. a flat file | untested — the differentiator, and the hardest to prove |
| It just works even if your machine can't run local models. | Developers on modest hardware | Hosted-inference fallback through ai-gateway | untested — this is the conversion message |
| *(Team shared memory)* | Engineering leads | None. Speculative expansion; not in the launch motion. | out of scope |
| Your memory outlives the tool. | Developers who have switched agents | Import/export across harnesses | untested |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Internal dogfood | This project | The curated index stops being maintained by hand | Proceed / stop. **Gates everything below.** |
| Publish architecture write-up | Technical write-up | Inbound interest in standalone use | Justifies packaging work |
| Open-source local engine | Repo | Installs from multi-harness users | Justifies the team-tier investigation |
| Fall-through measurement | Installed users | Share running neither local models nor BYOK | Decides whether hosted inference is a business or a rounding error |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
