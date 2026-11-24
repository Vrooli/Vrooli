# Evaluating a detector on a pinned public corpus

`scenarios/prose-studio/scripts/raid_detector_eval.py` evaluates detector scores
against a pinned labeled CSV. It does not fetch data or run a detector. It
fails closed when the dataset identity, schema, model labels, domains, attack
classes, score coverage, or source-group split is incomplete or inconsistent.

## Data boundary

Keep the downloaded public benchmark CSV, score CSV, and temporary scratch
directory outside both the Vrooli repository and `~/private/presence/`. The
scratch SQLite database contains IDs, source-group IDs, labels, group metadata,
and scores; it is removed at exit. The report contains aggregates only and does
not include sample IDs or text. A capacity and disk reservation is required
before acquiring large files or running this evaluation. Keep the public
benchmark staging path separate from the private voice corpus.

## Inputs

The pinned JSON manifest has schema version 1 and these fields:

- `dataset_id`, immutable 40/64 character lowercase hex `revision`, `split`,
  `file_name`, and the exact CSV `file_sha256`.
- `score_file_name` and `score_file_sha256` bind the score input, while
  `detector` records its nonempty `id` and `revision`.
- `columns`: exact semantic mappings for `sample_id`, `source_id`,
  `adv_source_id`, `model`, `domain`, and `attack`.
- `human_model`, the complete `ai_models` inventory, observed `domains`, and
  supported `attack_classes`. `empty_attack_is_clean` must explicitly record
  whether the pinned schema defines an empty attack cell as clean; otherwise
  empty attack labels fail closed. `clean_attack_value` names that declared
  clean class. `threshold_provenance` records how the threshold was selected
  from calibration data and locked before evaluation.
  These inventories must come from inspection of the pinned file; the tool
  rejects unknown/missing model, domain, and attack labels.
- `license`: the dataset card's declared license, HTTPS source URL, and
  `per_source_rights` (`unresolved` or `reviewed`). A repository license tag
  does not itself establish rights for every source text.
- `threshold` fixed using calibration data before evaluation,
  `threshold_provenance` describing the calibration method, and deterministic
  source-group bootstrap settings (`replicates` 100–10,000, integer `seed`).

The score CSV must have exactly one row for every data `sample_id`, with
`score` and `partition` (`calibration` or `evaluation`). Scores must be finite;
higher scores mean AI. No sample ID may be duplicated, omitted, or added. A
source ID or adversarial source ID appearing in both partitions is rejected.

## Run

After host memory admission and a disk reservation at the required 120 GiB
free-space floor, stage the pinned inputs in a public benchmark directory. Use
a pre-created scratch directory on the same admitted storage:

```sh
python3 scenarios/prose-studio/scripts/raid_detector_eval.py \
  --manifest /path/to/raid-manifest.json \
  --data /path/to/raid_train.csv \
  --scores /path/to/detector-scores.csv \
  --work-dir /path/to/admitted-scratch \
  --report /path/to/aggregate-report.json
```

The report contains overall precision, recall, false-positive rate, F1,
balanced accuracy, AUROC, average precision, and source-group bootstrap 95%
intervals. It also reports confusion counts, prevalence, and precision/recall
and uncertainty by domain, generator model (including human), and attack class.
Undefined one-class metrics remain `null`; they are not replaced with zero.
Calibration rows are used to report class counts and validate the partition and
source separation; the manifest threshold is treated as frozen. The provenance
string is reported but cannot be independently verified by the evaluator. This
tool does not set the threshold, endorse a detector, or turn a score into an
authorship verdict.

A synthetic fixture test lives alongside the evaluator. It checks hash pinning,
immutable revisions, required labels, supported attacks, exact score coverage,
source and adversarial-source leakage, grouped metrics, bootstrap output, and
aggregate-only reporting. It does not download or retain benchmark rows.
