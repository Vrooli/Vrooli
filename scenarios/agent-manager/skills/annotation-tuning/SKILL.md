---
name: "annotation-tuning"
description: "Add, change, gate and schedule Agent Manager annotation labels through data packs, without changing code or invalidating unrelated observations."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["agent-manager","annotations","taxonomy","decision-models","capacity","self-improvement"]
  icon: "sliders"
  status: "active"
  revision: 1
  createdAt: "2026-11-18T00:00:00Z"
  updatedAt: "2026-11-18T00:00:00Z"
  requires:
    scenarios: ["agent-manager"]
    commands: ["agent-manager annotations packs", "agent-manager annotations labels", "agent-manager annotations coverage", "agent-manager annotations show", "agent-manager annotations run", "agent-manager annotations evaluate", "agent-manager annotations scheduler", "agent-manager annotations benchmark", "agent-manager annotations gold", "agent-manager usage annotations", "agent-manager measures get", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Annotation tuning

Use this when adding or changing what Agent Manager looks for in runs. A label is data in a
pack; the evaluator is chosen by the label. Adding a label must not require a code change,
and changing one label must not invalidate the others.

### 1. Choose the label and its evaluator

| Evaluator | Use when | Cost |
|---|---|---|
| `deterministic` | A rule over existing facts answers it | free |
| `signal` | An existing operator signal already answers it | already paid |
| `decision` | The answer is semantic, bounded, and benefits from calibrated probabilities | cheap |
| `composite` | It is a logical combination of other labels | free |
| `extract` | Rare free-form evidence is required | expensive; on demand only |

Prefer deterministic. Do not add an AI label for something a rule can answer.

### 2. Write the question

- `choice`: 2–255 labelled options, each with a description.
- `score`: 2–10 ordered levels, level 0 first.
- `noul`: a true/false statement.
- Keep the state bounded: operator message openings, agent text ends, tool summaries,
  timings and deterministic flags. Never send a full transcript.
- Write the label so a human can disagree with it. "The agent is spinning" is not a
  question; "How much is the agent repeating work without new information?" is.

### 3. Define the evidence and candidates

- Evidence selectors are declarative: `operator_messages`, `agent_text`, `tool_summary`,
  `timings`, `deterministic_flags`, `existing_signals`, `workload`, `terminal`.
- Candidates are CEL over the documented `RunFeatures`. Example:
  `duration_min >= 8 || failed_commands >= 2 || repeated_paths >= 3`.
- `require` states when the run has evidence at all, for example
  `agent_text_count > 0 || tool_calls > 0`. Without it an empty digest still produces a
  confident-looking model answer. A failed `require` is skipped as `insufficient_evidence`.
- Affect labels must judge a person, not a workflow: add
  `workload_kind != 'workflow_node'`, because a workflow node's user-role message is its task
  brief. Check what the operator text really is before trusting an operator-derived label.
- Sampling is data: full for candidate runs, a small random rate for the baseline.

### 4. Gate before showing

A label surfaces as a rate or finding only after it meets its gate on a gold set that is
disjoint from the tuning set:

- precision ≥ 0.80, recall ≥ 0.70, at least the declared `min_gold` rows;
- a fresh validation set, not the tuning set;
- the gate is per label, per model, per definition hash.
- Unproven labels are stored and may be shown as evidence with their measured numbers.
  They never produce a rate.

Draft a blind review set (the packet is wider than the model digest on purpose), label it,
then measure with stored probabilities (no new inference):

```bash
agent-manager annotations gold --label agent.spinning --window 30d --out gold.jsonl
# edit gold.jsonl: set "positive" on every row from the evidence, without seeing the model's answer
agent-manager annotations benchmark --gold gold.jsonl --label agent.spinning  # threshold defaults to the label's declared value
```

`annotations gold` selects runs already observed under the label's current definition and
emits bounded state per run, never the stored probability: anchoring on the model would make
the measurement circular. Its default size is the label's `min_gold` (capped at 100), so the
draft can actually clear the gate. Every row starts `"positive":null`, and the benchmark refuses an
unfilled row rather than silently dropping it. Rows are JSONL
(`{"run_id":"…","label":"…","positive":true}`); the command fetches each run's stored
annotation, prints TP/FP/FN/TN and precision/recall, and prints the `gate.measured` block
(with its `definition_hash`) to paste under the label's `gate` in the pack. Re-thresholding
is free because the probability vectors are durable.

A score label declares its friction direction: `max_score` for low-is-bad
(`operator.valence`), `min_score` for high-is-bad (`agent.spinning`). Its confidence is the
probability mass in that region, not the winning level's probability, so a confident
three-level score can still clear the gate.

For affect labels, start with valence and frustration/satisfaction. Six-way emotion is
optional; small local decision models are weak at it. Grow the gold set from the operator's
own messages before expecting a stable mood series.

### 5. Compatibility and invalidation

- Adding a label invalidates nothing.
- Rewording with the same options is `compatible`: older observations remain usable, but the
  gate is re-measured.
- Changing options/levels/evidence semantics is `breaking`: only that label's observations
  are not reused; the gate resets.
- Changing the model keeps other models' rows; the gate is per model.
- Deprecate, do not delete. Historical observations stay queryable.

### 6. Schedule and cost

Embedding and annotation have separate configs. Use the shared scheduler:

- `passive`: only on-demand work (search/friction) runs; good for a laptop.
- `idle`: background work only when the host is idle and inside windows.
- `balanced`: windows plus budgets.
- Backfill horizon is explicit; older data is annotated only on demand.
- Batch by model affinity to avoid hot loads. One `DecideBatch` call can answer many labels
  for many states; store the probability vectors so re-thresholding is free.
- `agent-manager usage annotations --window 7d` is the operator read for coverage and gate
  state; `agent-manager annotations evaluate` runs a bounded pass on demand.

### 7. Measure the label

- `annotations coverage` for observed / compatible / not_observed / unavailable.
- `annotations evaluate` for a bounded live pass; `annotations benchmark --gold <file>` for a
  labelled gold set. The benchmark uses stored probabilities, so re-thresholding runs no model.
- `annotations run <run-id>` for one evidence run.
- Journal the result with `vrooli-memory journal note --kind work-record`.

### Anti-gaming

- Never tune thresholds on the window you report.
- Never add a label to fish for a finding; add it because a question matters.
- Never drop gold rows that disagree to pass the gate.
- Never present a label from a single run as a pattern.
- Never use a stronger model only for the reported sample.
