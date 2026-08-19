# Architecture — Music Library

How the scenario is put together, and why.

## Purpose Of This Document

Use this document to answer:

- Why is the recommender built this way rather than the conventional way?
- What does the preference model actually produce, and what depends on it?
- Where are the boundaries, especially the one protecting the listener's trust?
- What is designed versus built?

Domain ownership is in [`DOMAINS.md`](DOMAINS.md); workflows in
[`FLOWS.md`](FLOWS.md); storage in [`DATA.md`](DATA.md); the decisions behind these
choices, and the alternatives rejected, in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Scenario Shape

### Why this is content-based, and why that is an advantage

A library with one listener has **no collaborative signal** — there are no other
users to draw similarity from. That removes the mechanism behind the behaviour this
product exists to fix: recommendation systems trained on aggregate engagement are
conservative by construction, regularising a handful of plays in a new sub-genre
away as noise against years of history.

So the model is content-based, defined over attributes computed from the audio
itself. This is also the only thing that *can* work here: a track generated ten
seconds ago has no interaction history and never will, so no identity-based method
can rank it. One mechanism therefore serves both halves of the product.

### The preference model

A Bayesian preference model over frozen embeddings supplied by `music-tools`,
trained on pairwise comparisons. The essential property is that it returns **both
an expected preference and an uncertainty** at any point in the space. Three
requirements depend on that second output:

- **Informative elicitation** — choose the next comparison that most reduces
  posterior uncertainty, so few judgements go a long way. This matters acutely
  because one person produces every label.
- **Honest exploration** — high uncertainty marks genuinely unexplored territory,
  which is what the exploration control acts on.
- **Truthful explanation** — "I expect you'll like this" and "I have no idea and
  want to find out" are different statements, and the listener is told which one
  applies.

A point-estimate rating scheme cannot supply any of these. The embedding backbone
stays frozen: personalisation lives in the light head, which trains in minutes on
commodity hardware, rather than in a fine-tune the reference machine cannot hold.

### Why generation protects the recommender

Closed recommendation loops degenerate: serving the model's argmax narrows what the
listener hears, which narrows the evidence, which narrows the model. The
best-established defences are **continuous exploration** and **growing the
candidate pool**.

Local generation grows the candidate pool at zero marginal cost, indefinitely. That
makes generation a structural defence against collapse rather than a second
feature — which is the strongest argument for pairing the two capabilities at all.
Alongside it, ranking is **calibrated** to the listener's own distribution rather
than always serving the peak, and served diversity is monitored so persistent
contraction raises a warning.

## System Boundaries

| Inside | Outside |
|---|---|
| Listening, queue, transport, library index | Model execution of any kind — `music-tools` |
| The preference model and everything that shapes it | Embeddings, boundaries, tags, generation — `music-tools` |
| Explanation, exploration policy, calibration | Speech — `audio-tools` |
| Generated-content disclosure and lineage | Publishing or distributing music anywhere |
| Offer decoration, strictly downstream | Any offer influence on ranking |

### The blindness boundary

`ranking` and `generation` must not be able to observe what is sold, promoted, or
commission-bearing. `offers` sits strictly downstream of a final ranking and may
only decorate it — it cannot reorder or filter.

This follows the lifestyle-bundle recommendation-blindness rule in monetisation
strategy, and it is enforced by **package boundary rather than convention**. The
authority the listener is paying for is the reason the bundle exists at all, so
this is an integrity control, not a policy preference.

## Contracts And Data Flow

```
listen ──▶ signals ──▶┐
compare ─▶ ratings ──▶├──▶ preference model ──▶ ranking ──▶ queue ──▶ listen
                      │         (mean + uncertainty)  │
constraints ─────────▶┘                               ├──▶ explanation
                                                      └──▶ generation ──▶ music-tools
                                                                 │
                                         decomposition ◀─────────┘
```

Generated candidates re-enter through the same decomposition path and are ranked by
the same model with no provenance advantage. Disclosure is a presentation concern;
it never touches the score.

Two contract invariants:

- **The explanation is produced by the pass that produced the ranking**, never
  reconstructed afterwards. A reconstructed explanation can disagree with the
  decision it claims to describe, which defeats the purpose.
- **A profile records the embedding model it was fitted against.** If that model
  changes upstream, the profile is stale — a preference model fitted over one
  embedding space cannot be read against another.

### Signals, and one honest caveat

Completion, replay, and skip are the dense signal; comparisons are the sparse but
high-quality one. Because tracks are structurally segmented, the *position* of a
skip can in principle be attributed to the section that was playing —
attribute-level feedback at no cost to the listener.

That idea is **unproven**. No prior art for within-track section attribution was
found, and skip data is confounded by listeners abandoning early regardless of
content and by not-right-now skips that say nothing about taste. It is therefore
implemented as a hypothesis: attribution requires the same section to be abandoned
repeatedly across plays, is corrected for position bias, and must demonstrate
improved prediction against a model without it before it is enabled by default.

## Shared Infrastructure

- **`music-tools`** — the entire capability layer. No model runs here.
- **`qdrant`** — embedding index for retrieval and similarity.
- **api-core BlobStore** — generated audio and transcode cache.
- **SQLite by default**, Postgres optional for larger libraries.

### Storage posture

Source files are **pointed at and never touched** — not moved, renamed, retagged, or
deleted. Managed storage holds only generated audio and derived artifacts, under a
budget with least-recently-used eviction. Producing many generated candidates and
keeping few is intended behaviour.

## Extension Rules

- **New signal type** — it enters through `signals` and must state its confounds.
  A signal without a stated failure mode is not admissible evidence.
- **New surface** — if it presents a recommendation, it presents the explanation
  with it. These do not ship separately.
- **New attribute** — it comes from `music-tools`. Computing an attribute locally
  would create a second source of truth for the same fact.
- **Anything touching offers** — it lives downstream of a final ranking. If a change
  requires `ranking` to know about offers, the change is wrong, not the boundary.
- **Never** write to a source root, add a behavioural telemetry path, or let a
  generated track reach a surface without its disclosure.

## Architecture Maturity

| Aspect | State |
|---|---|
| Documented | Yes — this document, `DOMAINS.md`, `FLOWS.md`, `DATA.md` |
| Implemented | **No.** No domain exists in code; the template example domain is still present |
| Dependencies declared | **No.** `.vrooli/service.json` declares none |
| Blindness boundary enforced | **No.** Designed and documented; no package boundary or test exists |
| Core assumption validated | **No.** Whether generated music survives repeat listening is untested |

This scenario is at `generated`. See
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Intentional Deviations

| Deviation | Conventional approach | Why |
|---|---|---|
| No collaborative filtering | The default for recommendation | Unavailable with one listener, and it is the cause of the behaviour being fixed |
| Uncertainty-bearing model | Point-estimate ratings are cheaper | Elicitation, exploration, and explanation all read from uncertainty |
| Built rather than adopted player | Mature self-hosted players exist | The differentiating surfaces have no representation in their data models |
| Calibrated rather than argmax ranking | Serve the highest score | Argmax is the mechanism that makes a recommender feel stuck |
| Generation as infrastructure | Generation as a feature | It is the candidate-pool growth that keeps the loop from collapsing |

## Documentation Architecture

- [`DOMAINS.md`](DOMAINS.md) — bounded contexts, ownership, the blindness boundary
- [`FLOWS.md`](FLOWS.md) — ordered states and illegal transitions
- [`DATA.md`](DATA.md) — ownership, retention, privacy
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependencies and degraded behaviour
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — sensitivity and threat model
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — rejected alternatives

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — domain ownership
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why each choice was made
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — current gaps
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — why blindness is required
