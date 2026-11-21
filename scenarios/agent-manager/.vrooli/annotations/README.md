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

Every label has a gate. `measurements` is empty until a hold-out measurement exists;
empty means **unmeasured**, not proven. A measurement is for one
`(definition_hash, model, resource_role)` and only a `split: "holdout"` entry can
prove it, for that model alone; only proven observations may enter a rate. Record the
entry `annotations benchmark` prints (see `docs/concepts/RUN-ANNOTATIONS.md`). Labels
may also declare `role` / `escalate_role` (default: `runtime`), `population:
"human_operator"`, and `scope: "message"` for one answer per human operator message.
Coverage is always reported separately. An empty window means unknown coverage,
not "no friction".

Mood is a message-scope series, not a run score: `operator.dissatisfied` (mild pushback counts, not only anger: it was named `operator.frustrated` until most gold positives proved to be mild),
`operator.pleased` and `operator.impatient` each answer one noul question per human
operator message (`require: human_message_count > 0`, `population: "human_operator"`).
`runtime.provenance` decides whose words count: workflow, scheduled, interactive and
ad hoc runs are driven by agents and briefs, so only imported transcripts of a person
speaking are mood subjects. Everything else is derived and stored, never asked:
`operator.valence` is a message-scope composite (pleased minus dissatisfied), and the
run-scope `operator.dissatisfied_run`, `operator.pleased_run` and `operator.impatient_run`
are composites over `message_labels` (the run's own message observations), so run mood
flows through facets, friction and the digest like any other label. A label that cannot reach a gold hold-out (too rare, or
not observable from the digest) is deleted or marked `optional: true`
(`agent.discovery_slow`, `work.blocked_by_tool`), never left to produce an unproven rate.

## Budgets and sampling

`runtime.batch.max_states` (32) and `max_questions` (64) bound one `DecideBatch`
call. `runtime.budgets.max_runs_per_pass` and
`max_decision_states_per_pass` bound one batch. Sampling is deterministic:
`sha256(run_id|label_id)` decides inclusion, so re-running the same pack selects
the same runs. Do not annotate every attached web-console session by default.
