# textmetrics

`textmetrics` is the shared deterministic measurement tier for prose
generation. It computes lexical diversity, repetition, compression, sentence
burstiness, Flesch–Kincaid, Dale–Chall, Gunning Fog, MATTR, type-token ratio,
lexicon spans, and pairwise lexical similarity. It intentionally does not
claim to compute perplexity or semantic similarity: no log-probability or
embedding surface is available in the current gateway contract.

The package has no runtime dependency on prose-studio. Identical UTF-8 input
and lexicon produces identical values and JSON representation, which makes it
suitable for persisted candidate measurements and reproducible comparisons.
