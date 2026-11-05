# Context-brief calibration fixture

This committed fixture contains 30 answerable operator-intent prompts and 8
no-answer prompts. Answerable rows carry the expected provider and item-title
label used by the live sweep. It is a regression corpus for gate changes, not a
claim that every provider is currently indexed.

The live federated sweep ran against 33 registered Search Hub providers on
2026-09-07. It used the effective score rule (rerank score when non-zero,
otherwise score), and classified a brief as delivered when the top score met
the candidate threshold. The seeded item floor remained `0.20`.

| MinRerankScore | Predicted | True positive | False positive | Precision | Recall |
|---:|---:|---:|---:|---:|---:|
| 0.01 | 11 | 10 | 1 | 0.909 | 0.333 |
| 0.03 | 9 | 8 | 1 | 0.889 | 0.267 |
| 0.05 | 9 | 8 | 1 | 0.889 | 0.267 |
| 0.08 | 9 | 8 | 1 | 0.889 | 0.267 |
| 0.10 | 8 | 7 | 1 | 0.875 | 0.233 |
| 0.15 | 7 | 7 | 0 | 1.000 | 0.233 |
| 0.20 | 7 | 7 | 0 | 1.000 | 0.233 |
| 0.25 | 7 | 7 | 0 | 1.000 | 0.233 |
| 0.30 | 6 | 6 | 0 | 1.000 | 0.200 |
| 0.35 | 5 | 5 | 0 | 1.000 | 0.167 |
| 0.40 | 5 | 5 | 0 | 1.000 | 0.167 |
| 0.50 | 4 | 4 | 0 | 1.000 | 0.133 |

The chosen threshold is `MinRerankScore=0.01`, `MinItemRerankScore=0.20`:
precision is 0.909 and recall is 0.333. The target pair (precision above 0.80
and recall above 0.70) was not met; the best measured pair is retained without
lowering the target. The low recall is a Search Hub federated-ranking finding,
not a reason to weaken the gate further.

Any threshold change must update this report and the corpus test. This report
records 38 live prompts, the provider snapshot, the sample count and the date.
