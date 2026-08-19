# Decisions — Music Library

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-19 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-19 | **The preference model is content-based over frozen embeddings. Collaborative filtering is not used.** | A single-listener library has no collaborative signal to learn from. Collaborative filtering is also the direct cause of the behaviour this scenario exists to fix: it is conservative, popularity-biased, and slow to follow a listener into a new sub-genre. | Novel and generated tracks are rankable from their attributes alone. Adaptation to a new interest takes examples, not seasons. No cross-listener signal is ever available. | Revisit only if the product gains genuinely multi-listener semantics. |
| 2026-08-19 | **The preference model must emit calibrated uncertainty, not a point estimate.** | Informative pair selection, honest exploration, and truthful explanation all read from uncertainty. A point-estimate rating cannot distinguish "predicted poor" from "unknown". | The model is a Bayesian formulation over the embedding space rather than an online point-update rating. Uncertainty is a first-class output with dependent requirements. | Revisit if a point-estimate method is shown to satisfy the dependent requirements. |
| 2026-08-19 | **Rejected plain Elo as the rating mechanism, while keeping pairwise comparison as the elicitation method.** | Elo is an online point-estimate approximation of a pairwise-preference model. It carries no uncertainty, so it cannot drive active pair selection or exploration. | Pairwise comparison remains the primary explicit signal — it is far more label-efficient than thumbs for a single rater. The rating that comes out of it is distributional. | Revisit if uncertainty is sourced elsewhere and a cheaper rating update is wanted. |
| 2026-08-19 | **Build the player interface rather than adopting an existing self-hosted music server.** | Mature self-hosted players exist and would remove real work — transcoding, clients, playlists. But the surfaces that justify this scenario (why a track was chosen, generated-versus-owned provenance, the editable taste profile, attribute steering) have no representation in their data models. | Full control of the differentiating surfaces. In exchange this scenario owns transcoding, gapless playback, library scan, and offline behaviour, none of which are novel but all of which are real. | Revisit only if the differentiating surfaces turn out to be expressible as an extension to an existing server. |
| 2026-08-19 | **Source audio is read-only and referenced in place; the scenario never imports or moves owned files.** | Listener libraries already exist and are already organised. Taking ownership of them creates import, dedupe, and migration burdens that earn nothing. | Track identity is derived from content, so files can move without losing history. Generated audio is the only audio this scenario writes, and it goes to a managed location. | Revisit if in-place references prove too fragile across platforms. |
| 2026-08-19 | **Ranking and generation are structurally blind to monetisation, enforced by package boundary.** | The lifestyle bundle carries a recommendation-blindness rule: the component producing a recommendation must not know what is sold or commission-bearing. Offer insertion is strictly post-processing over an already-final ranking. | `offers` may decorate a final ranking but cannot reorder or filter it. Enforcement is a boundary, not a convention, because the listener's trust is the reason the bundle exists. | Revisit only through a monetisation-strategy decision, never locally. |
| 2026-08-19 | **Generation exists partly as a defence against feedback-loop collapse.** | The degenerate-feedback-loop literature finds the strongest remedies are continuous exploration and growing the candidate pool at least linearly. Local generation grows the candidate pool at zero marginal cost, indefinitely. | Generation is not a separate novelty feature; it is load-bearing for recommendation quality. A build without it is measurably more prone to narrowing. | Revisit if candidate-pool growth is achieved another way. |
| 2026-08-19 | **Recommendations are calibrated to the listener's distribution rather than served from the argmax.** | Always serving the highest-scoring item is the mechanism that makes a recommender feel stuck on one genre. | The queue reflects the shape of the listener's taste, including its minority regions. Slightly lower average predicted score, materially better perceived range. | Revisit if calibration measurably suppresses genuine preference. |
| 2026-08-19 | **Section-level attribution of skips is a tested hypothesis, not a load-bearing mechanism.** | Structural segmentation makes it possible to attribute a skip to the section that was playing. No prior art was found for within-track attribution, and skips are known to be noisy — listeners skip because they dislike a track or merely because it is not the moment. | Attribution requires repetition across plays before it updates the profile, and is corrected for position bias. The product does not depend on it working. | Revisit once enough listening history exists to evaluate it honestly. |
| 2026-08-19 | **All audio understanding is delegated to `music-tools`. This scenario runs no models.** | Attribute extraction, embeddings, separation, and generation are capability-primitive concerns with their own hardware and licence constraints. | No model registry, no GPU claim, and no inference path here. Every attribute has a single source. | Revisit only if the boundary blocks a listener-facing requirement. |
| 2026-08-19 | **Taste is modelled per context rather than as one global profile.** | Listening preference is not stationary — the same listener wants different things for focus, exercise, and evening. A single profile averages these into something that fits none of them. | Profiles are context-scoped, which multiplies the sparse-feedback problem and makes label efficiency more important. | Revisit if context separation proves unlearnable from available signal. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the blindness boundary
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
