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
    commands: ["agent-manager annotations packs", "agent-manager annotations labels", "agent-manager annotations coverage", "agent-manager annotations show", "agent-manager annotations run", "agent-manager annotations evaluate", "agent-manager annotations scheduler", "agent-manager annotations benchmark", "agent-manager annotations gold", "agent-manager annotations calibration", "agent-manager annotations provenance", "agent-manager annotations series", "agent-manager usage annotations", "agent-manager measures get", "vrooli-memory journal note"]
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
- Mood labels must judge a person, not a workflow: use `scope: "message"`, `population:
  "human_operator"` and `require: human_message_count > 0`. A workflow node's, a scheduled
  run's and an agent's delegation message are briefs (`runtime.provenance`); check what the
  operator text really is before trusting an operator-derived label.
- Sampling is data: full for candidate runs, a small random rate for the baseline.

### 4. Gate before showing

A label surfaces as a rate or finding only after a **hold-out** measurement clears its gate
for the model that answered:

- precision ≥ 0.80, recall ≥ 0.70, at least the declared `min_gold` rows;
- the hold-out is disjoint from the tuning set by run; the gate is per
  `(definition hash, model, resource role)`, so a proof never carries to another model;
- unproven labels are stored and shown as evidence with their measured numbers, never as a rate.

Draft a blind review set (the packet is wider than the model digest on purpose), label it
without seeing the model's answer, fit on tuning, prove on hold-out:

```bash
agent-manager annotations gold --label work.blocked_by_tool --window 30d --out gold.jsonl
# set "positive" on every row from the evidence; split rows into tuning and hold-out files by run
agent-manager annotations benchmark --gold tuning.jsonl --label work.blocked_by_tool --model laya:en --resource-role decide.local
agent-manager annotations benchmark --gold holdout.jsonl --split holdout --fit-from tuning.jsonl --label work.blocked_by_tool --model laya:en --resource-role decide.local
agent-manager annotations calibration --label work.blocked_by_tool --gold tuning.jsonl --gold holdout.jsonl
```

Gold files live in the private gold root (`runtime.gold_root`, default `gold/` under the scenario's state directory; the
API reports it as `gold_root` on `annotations packs`): relative `--out`, `--gold`, `--fit-from` and `--runs-from` resolve
against it, files are written 0600, and a measurement's `gold_file` is root-relative. After editing `runtime.provenance`,
run `agent-manager annotations provenance` so coverage and the mood population reflect the new policy.

`annotations gold` emits bounded state per run, never the stored probability: anchoring on
the model would make the measurement circular. Every row starts `"positive":null`, and the
benchmark refuses an unfilled row. The tuning run fits a temperature and threshold; the
hold-out run measures with them and prints the `gate.measurements[]` entry to paste under
the label's `gate`. Re-thresholding is free because the probability vectors are durable.
A hold-out needs at least 8 positives to mean anything; if the corpus cannot supply them
(a rare label), delete the label or mark it `optional: true` rather than leave it unproven.

A score label declares its friction direction (`max_score` low-is-bad, `min_score`
high-is-bad); its confidence is the probability mass in that region. Prefer `noul`:
small decision models compress ordinal scales.

For mood labels, read every message (the corpus is small), label per message, and split by
message. Grow the gold set from the operator's own words before expecting a stable series.

### 5. Compatibility and invalidation

- Adding a label invalidates nothing.
- Rewording with the same options is `compatible`: older observations remain usable, but the
  gate is re-measured.
- Changing options/levels/evidence semantics is `breaking`: only that label's observations
  are not reused; the gate resets.
- Changing the model keeps other models' rows; the gate is per model.
- A label that cannot reach a hold-out is deleted or marked `optional: true`. Historical
  observations of a removed label stay in the store but leave every report.

### 6. Schedule and cost

Embedding and annotation have separate configs. Use the shared scheduler:

- `passive`: only on-demand work (search/friction) runs; good for a laptop.
- `idle`: background work only when the host is idle and inside windows.
- `balanced`: windows plus budgets.
- Backfill horizon is explicit; older data is annotated only on demand.
- Batches group by the model each label's role resolves to, to avoid hot loads. One `DecideBatch` call can answer many labels
  for many states; store the probability vectors so re-thresholding is free.
- `agent-manager usage annotations --window last_7d` is the operator read for coverage, gate
  state and the model each proof is for; `agent-manager annotations evaluate` runs a bounded pass on demand.

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
