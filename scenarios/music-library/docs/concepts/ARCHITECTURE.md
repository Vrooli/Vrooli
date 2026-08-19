# Architecture — Music Library

Domain ownership is in [`DOMAINS.md`](DOMAINS.md); the decisions behind these
choices, and the alternatives rejected, are in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Why this is content-based, and why that is an advantage

A library with one listener has **no collaborative signal** — there are no other
users to draw similarity from. That removes the mechanism behind the behaviour this
product exists to fix: recommendation systems trained on aggregate engagement are
conservative by construction, regularising a handful of plays in a new sub-genre
away as noise against years of history.

So the model is content-based, defined over attributes computed from the audio
itself. This is also the only thing that *can* work here: a track generated ten
seconds ago has no interaction history and never will, so no identity-based method
can rank it. One mechanism therefore serves both halves of the product.

## The preference model

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

## The loop

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

## Why generation protects the recommender

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

## Signals, and one honest caveat

Completion, replay, and skip are the dense signal; comparisons are the sparse but
high-quality one. Because tracks are structurally segmented, the *position* of a
skip can in principle be attributed to the section that was playing — attribute-level
feedback at no cost to the listener.

That idea is **unproven**. No prior art for within-track section attribution was
found, and skip data is confounded by listeners abandoning early regardless of
content and by not-right-now skips that say nothing about taste. It is therefore
implemented as a hypothesis: attribution requires the same section to be abandoned
repeatedly across plays, is corrected for position bias, and must demonstrate
improved prediction against a model without it before it is enabled by default.

## Storage posture

Source files are **pointed at and never touched** — not moved, renamed, retagged, or
deleted. Managed storage holds only generated audio and derived artifacts, under a
budget with least-recently-used eviction. Producing many generated candidates and
keeping few is intended behaviour.
