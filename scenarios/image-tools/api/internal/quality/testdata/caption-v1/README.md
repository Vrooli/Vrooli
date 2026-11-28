# Caption and visual-question corpus v1

This reference set is frozen before any new candidate run. It uses five
scenario-authored, deterministic graphics already present in Image Tools or the
owner run evidence. It has three difficulty bands and probes simple geometry,
product-like artwork, receipt text, low-contrast text in blur, and a generic
stylized illustration. It contains no client uploads, private media, or real
person's likeness.

The `caption` field is one human-written reference sentence. The `question` and
`answer` fields provide a second, prompt-conditioned task on the same image.
Candidate outputs must be retained verbatim and bound to this corpus version,
the exact model revision, runtime/backend, and source image SHA-256. Do not edit
references after inspecting candidate output; version a new corpus if a
reference is factually wrong.

## Frozen comparison policy

- Score captions and question answers in separate manifests with the `text`
  profile. Use the `image-tools-caption-v1` dataset ID and require the local
  `qwen3-vl:4b` judge for a promotion decision.
- Apply a per-case `text.similarity >= 0.35` floor as a lexical regression
  guard, plus per-case judge floors of `judge.fidelity >= 0.70`,
  `judge.instruction >= 0.75`, and `judge.artifacts >= 0.80`. The mean of each
  judge dimension must also be at least `0.80`. Similarity is normalized
  Levenshtein agreement, not semantic truth; the judge rubric must reward
  equivalent wording while penalizing unsupported claims and missed
  discriminating facts.
- Promotion also requires at least a `0.01` mean improvement in
  `judge.fidelity` over the paired baseline. This gives the report's explicit
  `promotion_eligible` bit a precommitted win condition instead of treating
  threshold compliance alone as evidence that a replacement is better.
- A candidate is promotable only after complete reports for all ten tasks,
  no case-level failures, and a paired baseline with the exact same references,
  rubric, budgets, and evaluator version. Keep human visual review as a
  separate observation; a local model judge is not calibrated human preference.
- The initial paired comparison is `smolvlm-256m` (current selected default)
  against `moondream2` (already installed). Do not install another model merely
  to satisfy this protocol. Evaluate a newer rights-clean model only after its
  runtime path, model revision, complete-job resource admission, and storage
  budget are established. Candidate outputs are not populated in this corpus;
  each model gets its own immutable result manifest.

This corpus is a compact capability/regression gate, not a broad natural-image
benchmark. The current local samples establish safe reproducibility and
reference metric behavior; broad public-photo generalization remains untested.
