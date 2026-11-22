# UI verdict trust corpus

`scripts/score_ui_verdicts.py` scores phase predictions against an adjudicated JSONL corpus and emits paired before/after rates. The case set is the fixed denominator. Each row needs a stable `case_id`, one applicable `phase`, a gold `truth` (`defect` or `clean`), and a concise `evidence` pointer that explains the adjudication. Rows may also carry a `rule_code` and `snapshot_sha256`; the report breaks rates out by rule and identifies shared snapshot clusters. Do not duplicate one observation under several IDs to increase sample size.

Prediction files are JSONL rows of `{ "case_id", "phase", "verdict" }`, where verdict is `passed`, `failed`, or `not-evaluated`. Every corpus case must have exactly one prediction in each snapshot. Extra, missing, duplicate, and phase-mismatched records stop scoring. For a measurement claim, each prediction must also carry its `evaluator` identity. Supply `--before-evaluator` and `--after-evaluator` manifests; each manifest requires `evaluator`, `build_identity`, `command`, `execution_mode` (`fresh`, `cached`, or `replay`), and `source_sha256`. The scorer checks every row against the manifest and prints both evaluator identities in the report. Historical predictions without this provenance remain scoreable only as descriptive records and cannot pass this provenance check.

The false-positive rate is clean cases failed divided by all clean cases. The false-negative rate is defective cases not failed divided by all defect cases; `not-evaluated` therefore cannot improve the rate and is also reported separately. Coverage is evaluated cases divided by all applicable cases. A phase with no examples of one truth class reports that class's rate as `n/a`, never zero; a phase with no labels is explicitly unmeasured, not zero. Do not compare rates as estimates of fleet performance: they describe this adjudicated corpus only. Claims sharing a capture are clustered, so they must not be described as independent captures or a probability sample.

The report includes an independence audit by phase. It counts distinct snapshot clusters and scenario clusters separately from labeled claim rows. A cluster qualifies as independently adjudicated only when the row carries frozen scenario/build/route/state/viewport/snapshot identity plus an `adjudication` object with `method: "blind-independent"`, `phase_blinded: true`, at least two reviewers, and a separate `rubric_ref`. These fields are an auditable declaration, not cryptographic proof of reviewer independence; preserve review receipts with the corpus. Use `--minimum-independent-cases 50 --phases <actual-phase>` to make the scorer exit nonzero unless every named phase has 50 qualifying snapshot clusters. The default remains descriptive so legacy candidate corpora can be honestly audited.

`build_ui_health_observation_corpus.py` freezes objective claims from ui-health's persisted browser observations. It collapses duplicate evaluator inputs and labels only explicit client expectations against recorded geometry. `ui-health-trust-replay` evaluates those frozen minimal observations through the current rule implementation without writing to the database. The paired before snapshot comes from finding codes stored with the original observations. This pair measures replay agreement on that rule corpus; it does not establish fleet-wide experience quality or movement if the evaluator itself did not change.

The current run has 108 rule claims but only 45 distinct evaluator inputs, all from `git-control-tower`; these are clustered by one scenario and are **not 45 proven independent samples**. The labels come from explicit client expectations and recorded geometry, but they have not been blind independently adjudicated. Three rules have 107 clean labels and one defect label, so the defect recall denominator is too small for a stable estimate. A read-only RCL audit found 2,505 candidate unique `(asset, version, story)` groups with an accessibility artifact and a supported assertion kind. That is only a candidate inventory: it has not been independently adjudicated or replayed as a Test Genie phase corpus, and most stored failures are infrastructure or harness outcomes. The reports do not provide post-change phase predictions. Do not report the candidate count as measurement or imply that this proves the 50-case showcase.

To measure another phase, freeze sanitized independent cases with contract/geometry evidence, keep only behavior verdicts (exclude infrastructure-blocked outcomes), and run both the immutable baseline and the changed phase evaluator against those same case IDs. A source-level replay alone is not a Test Genie phase run. If the phase cannot be executed safely against a fixed fixture, report its rates as unmeasured and record that exact gap.

The EX6 run's `experience-blind-review-v1` packet is generated from frozen objective geometry candidates. It masks source case IDs and evaluator outputs, includes two independent reviewer templates, and keeps the source crosswalk in a mode-0600 private sidecar. It has 45 unique source snapshot clusters, not 108 cases; the source lacks route, state, and served-build identity. It is therefore an operator review packet for candidate labels only, not a qualifying 50-case corpus. It must not be used to claim measured phase accuracy.

`scripts/replay_ui_health_rule_corpus.py` builds and runs the actual `ui-health` BrowserObservation rule evaluator against the same frozen JSONL input. It records binary, source, input and output hashes plus the full command in a manifest usable by the scorer. This proves a same-input rule replay only; it does not invoke or stand in for the aggregate Test Genie `experience` phase, which also runs static UI checks and browser render/interaction checks.

Example descriptive score:

```sh
python3 scenarios/test-genie/scripts/score_ui_verdicts.py \
  --corpus path/to/cases.jsonl --before path/to/before.jsonl \
  --after path/to/after.jsonl --output path/to/report.md
```

For a measured score, add paired evaluator manifests and include the same evaluator identity in every row from that snapshot:

```sh
python3 scenarios/test-genie/scripts/score_ui_verdicts.py \
  --corpus path/to/cases.jsonl --before path/to/before.jsonl \
  --after path/to/after.jsonl --output path/to/report.md \
  --before-evaluator path/to/before-evaluator.json \
  --after-evaluator path/to/after-evaluator.json \
  --minimum-independent-cases 50 --phases experience
```
