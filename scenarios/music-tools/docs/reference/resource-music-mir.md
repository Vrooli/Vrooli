# Resource Spec — `music-mir`

The music-information-retrieval runtime this scenario depends on, specified at the
confidence it actually has.


Use this document to answer:

- What is `music-mir` for, and why is it a separate runtime?
- What is known about it, and what is merely assumed?
- What must be measured before it can be built?

## Status

**The current implementation has not been measured.** A local resource scaffold,
Music Tools job adapter, and request/cancellation contract now exist, but the
resource has not been installed, run, or profiled on the reference host. Every
figure elsewhere in the docs concerning structure analysis, beat tracking, or stem
separation is `vendor` or `estimated`. The current adapter only provides partial
acoustic estimates and low-confidence cut candidates. It now calls the
configurable local Whisper service for timestamped transcript segments when
available; these are hypotheses, not word-level alignment, and Whisper supplies
no confidence value. Semantic section labels and descriptions are still absent.

This document keeps the remaining qualification gap visible: composition is the
tier that has been measured; the MIR runtime has not.

## Why It Is A Separate Runtime

Recorded as a decision on 2026-08-19: the generation, music-information-retrieval,
and embedding stacks pin mutually incompatible Python versions, torch builds, and
NumPy majors. The shared sidecar provisioner (`pyenv-go`) syncs exactly one
virtualenv per scenario, so only one of the three can live in-scenario. The MIR
stack pins an exact torch build and an older NumPy major that cannot coexist with
the composition stack.

Cross-runtime coordination is by job queue and filesystem, never shared imports.

## Responsibilities

| Operation | Notes |
|---|---|
| Structure and section segmentation | |
| Beat and downbeat tracking | |
| Stem separation | A **hard dependency**, not a quality upgrade — see below |
| Tempo and key estimation | By vote across independent estimators, with disagreement retained |

Stem separation lives here and is never assumed available from the composition
runtime. The recorded reason: the generator's own documentation is internally
inconsistent about which variants support separation, and the variant that fits the
free-VRAM tier is the one most likely to lack it. The dependency stays regardless,
because a dedicated separator is also the better separator.

Tempo and key estimation retains estimator disagreement rather than resolving it.
Disagreement correlates with rubato, ambiguous tonality, and half/double-time
ambiguity, and is itself a useful attribute. The spike encountered exactly this
ambiguity in its own crude autocorrelation tempo checks, which is weak corroboration
that the policy is the right one.

## Licence Exposure

This is the resource where the licence lane split earns its keep. Several of the
strongest analysis models are non-commercial or share-alike, and at least one
useful tool declares no licence at all. An unknown licence defaults to the
**restricted** lane, never the permissive one.

`OT-P1-002` — a permissive-lane build that still produces a working analysis stack
— is the commercial gate, and it is gated on this resource more than on
`ace-step`, whose model is MIT.

Copyleft tools are invoked as separate processes, never linked in-process, so
their obligations do not reach the scenario binary.

## What Must Be Measured Before It Is Built

| Unknown | Why it matters |
|---|---|
| VRAM per operation | Every residency and rung statement in the docs is currently `estimated` |
| Throughput per operation | Analysis is the larger share of eventual GPU time, and none of it is profiled |
| Which models clear the permissive lane | Determines whether `OT-P1-002` is achievable at all |
| Whether separation and structure can share a resident set | Determines whether these are one exclusive lease or two |

The honest sequencing consequence: **this resource should be spiked the way
`ace-step` was, before its requirements are treated as costed.** A measured hour
here is worth more than another pass over the documents.

## Cross-References

- [`resource-ace-step.md`](resource-ace-step.md) — the sibling resource, measured
- [`model-registry.md`](model-registry.md) — per-model licence and lane
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — the three-runtime split, the separator dependency, the tempo vote
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — what is and is not measured
