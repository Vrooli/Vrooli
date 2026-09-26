# Go To Market — Music Tools

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

> **This scenario has no go-to-market of its own, and that is a conclusion rather
> than a gap.** It is a capability primitive with no price, no SKU, and no buyer —
> see [`MONETIZATION.md`](MONETIZATION.md). What follows is therefore short on
> purpose: it records the claim that consuming products inherit, the evidence that
> claim needs, and the two things that would bound any market built on it. It was
> rewritten on 2026-09-18 to replace generated scaffold that said "define during
> PRD review" in every field.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

**Audience: other Vrooli scenarios and the agents driving them.** There is no
external audience and none is expected. The first consumer is `music-library`; the
launch video pipeline is the consumer that exists *today*, by way of an agent
running the `music-generation` skill.

**Positioning, internally:** the music layer that consuming scenarios compose
instead of reimplementing, in the same relationship `image-tools` has to
`backdrop-studio` and `asset-studio`.

**The claim that travels outward**, carried by products rather than by this
scenario: *original music with no licence to resolve, generated on hardware you
already own, at no per-generation cost.* That claim is unusually strong in this
modality, because hosted music generation is exactly where per-generation
royalties and contested training-data provenance sit. It is also the claim the
whole architecture exists to protect, which is why there is no hosted rung.

Two bounds on it, both recorded as decisions and neither hypothetical:

- **It needs a GPU.** There is no CPU rung for composition and there will not be
  one. An operator without a capable card gets no composition at all — unlike
  `image-tools`, which degrades to CPU and still serves that user.
- **The permissive licence is a vendor claim.** ACE-Step 1.5 is MIT, and the
  publisher asserts commercially clean training data. That assertion is unaudited.
  It is the load-bearing claim under everything above, and it is recorded at
  `vendor` confidence, not as fact.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal scenario reuse | Consuming scenarios adopt this instead of building their own music path or licensing stock audio. | A working composition operation; the `ace-step` resource. | Count of consuming scenarios; retirement of the `music-generation` skill into this scenario. |
| Agent-driven marketing production | Launch videos and marketing assets get original scored audio with no licence step. | The take pool, so a track is ready when wanted rather than generated on demand. | Marketing assets shipped with generated rather than bundled audio. |
| Consuming products' own GTM | The zero-marginal-cost claim survives contact with a paying customer. | A proven permissive-lane build (`OT-P1-002`). | Not this scenario's to measure. |

No external channel. This scenario is not listed, packaged, or marketed on its own.

## Launch Motion

1. Build the `ace-step` resource against the measured spike evidence in
   [`../reference/resource-ace-step.md`](../reference/resource-ace-step.md).
2. Ship composition end to end — style, batch, takes, selection — and retire the
   `music-generation` skill into it.
3. Add the pool, so a consumer draws a ready take rather than waiting on a batch.
4. Prove the permissive-lane build (`OT-P1-002`), which is the commercial gate for
   every downstream product.
5. Analysis and transformation follow. Neither has a waiting consumer today.

Note what is *not* in this list: any packaging, pricing, or external-channel step.
If one ever appears, this document is the wrong place for it — the scenario would
first have to stop being a primitive.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| No licence to resolve on generated output. | Consuming scenarios | MIT model; per-artifact provenance recording model, lane, and rung | hypothesis — rests on an unaudited vendor claim |
| Zero marginal cost per generation. | Consuming scenarios | No royalty, no per-request billing, no third party that can change terms | true of licensing; **not** of electricity or wall-clock, which products should quote as measured throughput |
| Quality comes from selection, not from one lucky generation. | Consuming scenarios, agents | 2026-09-18 spike: no single take reliably usable, a batch of ten contained several | validated on one operator, one genre, one session |
| A track is ready when you want it. | Agents producing marketing assets | Pool with replenish-on-consume | unbuilt |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Does a second consumer adopt it? | Internal reuse | One scenario beyond the launch-video path composes music through this scenario. | If no second consumer appears, the scenario is over-built for its real demand and the analysis and transformation tiers should stay unstarted. |
| Does the permissive-lane build actually work? | Consuming products' GTM | `OT-P1-002` passes: a permissive build satisfies composition and analysis end to end. | If it cannot, no product built on this is shippable, and the lane split needs rework before anything downstream commits. |
| Is generated audio actually used, or quietly replaced? | Agent-driven production | Marketing assets ship with generated audio rather than falling back to bundled tracks. | Sustained fallback means the quality bar is not met and the `sft` variant and prompt work matter more than any further feature. |
| How often does the hardware bound bite? | Internal reuse | Count of consumers blocked by having no capable GPU. | A non-zero count is the recorded revisit trigger for the BYOK rung decision. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the no-hosted-rung and hardware-reach decisions
- [`../reference/resource-ace-step.md`](../reference/resource-ace-step.md) — the measured evidence behind the claims here
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
