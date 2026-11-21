# Annotation packs

`default.json` is Agent Manager's initial label vocabulary. The API loads it at
startup through `runannotations.DefaultPackPath()`; override the directory with
`AGENT_MANAGER_ANNOTATIONS_DIR` for tests or experiments.

This directory is **data**, not code. Adding, rewording or removing a label here
does not require a Go change. Each label's semantic definition is hashed, so a
pack edit invalidates only the label that changed.

## Add a label

Copy a label block, give it a unique `id`, choose `scope: "run"`, pick a
`method`, and add:

- `evidence` selectors from the table in
  `api/internal/runannotations/README.md`;
- `candidates` CEL when the expensive evaluator should run only on some runs;
- a `gate` (`precision`, `recall`, `min_gold`) — required, and never proven by
  default;
- a `search` facet (`label:<id>`) and threshold;
- a `friction` block when the label may inform friction reporting.

Then run:

```
cd scenarios/agent-manager/api && go test ./internal/runannotations/...
agent-manager annotations labels
```

The pack loader validates unique ids, question shapes, option/level counts,
evidence selectors, CEL compilation and gate ranges. A bad pack fails loudly.

## Change a label without invalidating everything

1. Edit only the labels whose meaning changed.
2. Compute the old definition hash (`agent-manager annotations labels --json`).
3. If the new question still means the same thing (reworded instructions, same
   options/levels), list the old hash:

   ```json
   "compatibility": {"change": "compatible", "compatible_with": ["<old-hash>"]}
   ```

   Existing observations are reused and stored with state `compatible`.
4. If the options, levels, scale or rule changed meaning, leave
   `compatible_with` empty (`"change": "breaking"`). Only that label is
   re-evaluated. Other labels' rows are untouched.

Pack `version` is reporting only. Do not expect a version bump to invalidate
caches.

## Gates and honesty

Every label has a gate. `measured` is absent until a hold-out measurement exists;
absent means **unmeasured**, not proven. Only `proven` gates may produce a rate.
Coverage is always reported separately. An empty window means unknown coverage,
not "no friction".

`operator.emotion` is `optional: true` and sampled at 0.2. It is skipped unless
`--include-optional` / `include_optional` is passed, and its six-way accuracy is
not claimed. `operator.valence` (-2..+2), `operator.frustration` and
`operator.satisfaction` are the primary affect labels.

## Budgets and sampling

`runtime.batch.max_states` (32) and `max_questions` (64) bound one `DecideBatch`
call. `runtime.budgets.max_runs_per_pass` and
`max_decision_states_per_pass` bound one batch. Sampling is deterministic:
`sha256(run_id|label_id)` decides inclusion, so re-running the same pack selects
the same runs. Do not annotate every attached web-console session by default.
