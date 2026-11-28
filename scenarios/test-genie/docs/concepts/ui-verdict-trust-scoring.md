# UI verdict trust corpus

`scripts/score_ui_verdicts.py` scores phase predictions against an independently adjudicated JSONL corpus and emits paired before/after results. Each corpus row is identified by `(case_id, phase)` and needs a gold `truth` (`defect` or `clean`), evidence pointer, and frozen scenario/build/route/state/viewport/snapshot identity. `snapshot_sha256` and evaluator `source_sha256` are lowercase SHA-256 digests. The same case ID may be scored in multiple phases; each phase gets its own truth and prediction. Do not duplicate one observation under several IDs to increase sample size.

Prediction files are JSONL rows of `{ "case_id", "phase", "verdict", ...identity }`, where verdict is `passed`, `failed`, or `not-evaluated`. Every corpus `(case_id, phase)` must have exactly one prediction in each snapshot. Extra, missing, duplicate, and phase-mismatched records stop scoring. Each prediction must repeat the exact subject scenario/build/route/state/viewport/snapshot identity and name its evaluator, evaluator build identity, source SHA-256, command, and execution mode. Before and after manifests contain phase-keyed evaluator identities with the same fields; stale or missing identities stop scoring.

`--assembly-receipt` is required. It must be the `ui-verdict-corpus-assembly/v1` receipt emitted by `assemble_ui_verdict_corpus.py`, and its scored corpus SHA-256, case count, reviewer packet hashes, and rubric must match the exact corpus input. This binds scored truths to the two blinded ballot files and optional adjudicator file recorded by that assembler.

Rates default to a **50 independently adjudicated snapshot-cluster minimum per phase**. Below that floor the report still shows counts and denominators, sets status to `insufficient-eligible-cases`, withholds FP/FN and coverage rates, writes JSON with `rates: null`, and exits 2. A phase only publishes rates once its own threshold is met. Missing required provenance is an input error and also exits nonzero. `--json-output` writes the machine-readable report used by EX6 evidence consumers; the Markdown report remains the human review artifact. For a reviewer handoff, use `--artifact-dir`: it writes fixed-name Markdown and JSON reports plus `ui-verdict-trust-artifacts.json`, a deterministic locator receipt with input, scorer, and output SHA-256 values and the full invocation. The command prints that receipt path even when it exits 2 for insufficient eligible cases, so reviewers can discover withheld reports. The scorer never treats one phase's cases as another phase's denominator.

The false-positive rate is clean cases failed divided by all clean cases. The false-negative rate is defective cases not failed divided by all defect cases; `not-evaluated` therefore cannot improve the rate and is also reported separately. Coverage is evaluated cases divided by all applicable cases. A phase with no examples of one truth class reports that class's rate as `n/a`, never zero; a phase with no labels is explicitly unmeasured, not zero. Do not compare rates as estimates of fleet performance: they describe this adjudicated corpus only. Claims sharing a capture are clustered, so they must not be described as independent captures or a probability sample.

The report includes an independence audit by phase. It counts distinct snapshot clusters and scenario clusters separately from labeled rows. A cluster qualifies only when the row carries frozen scenario/build/route/state/viewport/snapshot identity plus an `adjudication` object with `method: "blind-independent"`, `phase_blinded: true`, at least two reviewers, and a separate `rubric_ref`. These fields are auditable declarations, not cryptographic proof of reviewer independence; preserve review receipts with the corpus.

`build_ui_health_observation_corpus.py` freezes objective claims from ui-health's persisted browser observations. It collapses duplicate evaluator inputs and labels only explicit client expectations against recorded geometry. `ui-health-trust-replay` evaluates those frozen minimal observations through the current rule implementation without writing to the database. The paired before snapshot comes from finding codes stored with the original observations. This pair measures replay agreement on that rule corpus; it does not establish fleet-wide experience quality or movement if the evaluator itself did not change.

The current run has 108 rule claims but only 45 distinct evaluator inputs, all from `git-control-tower`; these are clustered by one scenario and are **not 45 proven independent samples**. The labels come from explicit client expectations and recorded geometry, but they have not been blind independently adjudicated. Three rules have 107 clean labels and one defect label, so the defect recall denominator is too small for a stable estimate. A read-only RCL audit found 2,505 candidate unique `(asset, version, story)` groups with an accessibility artifact and a supported assertion kind. That is only a candidate inventory: it has not been independently adjudicated or replayed as a Test Genie phase corpus, and most stored failures are infrastructure or harness outcomes. The reports do not provide post-change phase predictions. Do not report the candidate count as measurement or imply that this proves the 50-case showcase.

To measure another phase, freeze sanitized independent cases with contract/geometry evidence, keep only behavior verdicts (exclude infrastructure-blocked outcomes), and run both the immutable baseline and the changed phase evaluator against those same case IDs. A source-level replay alone is not a Test Genie phase run. If the phase cannot be executed safely against a fixed fixture, report its rates as unmeasured and record that exact gap.

The EX6 run's `experience-blind-review-v1` packet is generated from frozen objective geometry candidates. It masks source case IDs and evaluator outputs, includes two independent reviewer templates, and keeps the source crosswalk in a mode-0600 private sidecar. It has 45 unique source snapshot clusters, not 108 cases; the source lacks route, state, and served-build identity. It is therefore an operator review packet for candidate labels only, not a qualifying 50-case corpus. It must not be used to claim measured phase accuracy.

`scripts/replay_ui_health_rule_corpus.py` builds and runs the actual `ui-health` BrowserObservation rule evaluator against the same frozen JSONL input. It records binary, source, input and output hashes plus the full command in a manifest usable by the scorer. This proves a same-input rule replay only; it does not invoke or stand in for the aggregate Test Genie `experience` phase, which also runs static UI checks and browser render/interaction checks.

The before and after manifests use a phase-keyed shape:

```json
{
  "phases": {
    "experience": {
      "evaluator": "ui-health",
      "build_identity": "ui-health-build-123",
      "command": "vrooli scenario test demo --phases experience",
      "execution_mode": "fresh",
      "source_sha256": "<sha256 of evaluator source>"
    },
    "workflow": {
      "evaluator": "workflow-health",
      "build_identity": "workflow-health-build-123",
      "command": "vrooli scenario test demo --phases workflow",
      "execution_mode": "fresh",
      "source_sha256": "<sha256 of evaluator source>"
    }
  }
}
```

Run with case-matched predictions and manifests. The one-directory output is the supported operator/reviewer handoff; preserve the entire directory as the evidence artifact. It still returns 2 and writes the receipt when eligible labels are below threshold.

```sh
python3 scenarios/test-genie/scripts/score_ui_verdicts.py \
  --corpus path/to/cases.jsonl --before path/to/before.jsonl --after path/to/after.jsonl \
  --assembly-receipt path/to/assembly-receipt.json \
  --artifact-dir path/to/review-bundle \
  --before-evaluator path/to/before-evaluator.json \
  --after-evaluator path/to/after-evaluator.json \
  --phases experience,workflow,component-tests
```
