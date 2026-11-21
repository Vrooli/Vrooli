# Run annotations

Run annotations are Agent Manager's data-driven, inferred layer over run history. They
answer questions the deterministic tool and runtime lenses cannot: was the agent spinning,
did it take too long to find the right thing, did it lose its objective, what kind of work
was this, and what did the operator's message signal?

The deterministic backbone stays authoritative. Annotations are **evidence**, not verdicts:
a label produces a rate or a friction finding only after it passes its accuracy gate.

## Vocabulary

- **Pack** — a JSON vocabulary at `.vrooli/annotations/default.json` (`name`, `version`,
  `runtime`, `labels[]`). The pack version is reporting only.
- **Label** — one thing to observe: `id`, `scope` (`message|episode|run`), `method`,
  optional `question`, evidence selectors, candidate rule, evidence `require` guard, sampling,
  gate, search facet, friction block and compatibility edges.
- **Definition hash** — sha256 of a label's canonical semantic definition. Observations are
  keyed by `(scope, id, definition_hash, model, model_version)`.
- **Observation** — one stored answer with its probability vector, confidence, method,
  provider/model, evidence event ids, an evidence fingerprint and state.
- **State** — `observed`, `compatible`, `not_observed`, `unavailable`. Unknown is never
  stored as zero or false.

## Evaluators

| Method | Use | Cost |
|---|---|---|
| `deterministic` | A CEL rule over `RunFeatures` (workload, timings, tool summary, messages, terminal) | free |
| `signal` | An existing operator signal (`explicit_negative`, frustration, satisfaction) | already paid |
| `decision` | ai-gateway `DecideBatch` (choice/score/noul) under local-only, confidential | cheap |
| `composite` | CEL over other labels | free |
| `extract` | Declared, returns `unavailable`; generative extraction is on demand only | expensive |

A label may declare a deterministic `rule` with `fallback: "decision"`: the rule answers
when it can, and the model only sees the residual. `prior_rule` lets a deterministic fact
(for example a step-back trigger) answer a `noul` label before inference runs.

## Compatibility: what a change invalidates

| Change | Effect |
|---|---|
| Add a label | nothing |
| Reword a question, same options | that label's gate resets; old observations reusable if declared `compatible_with` |
| Change options/levels/evidence semantics | only that label's observations are not reused; gate resets |
| Change the model | observations are keyed by model; other models' rows remain |
| Bump the pack version | nothing; the hash is per label |
| Deprecate a label | historical observations remain queryable |

Resolution: exact hash → declared compatible predecessor (state `compatible`) → evaluate
fresh. Breaking changes never reuse.

## Decision calls

One `DecideBatch` call answers up to 32 states × 64 questions. Questions come from the
pack; states are bounded digests (operator message openings, agent text ends, tool summary,
timings, deterministic flags), never full transcripts. Probabilities are stored so
thresholds and gates can be re-tuned offline without re-running inference. `LLM_ENUM`
answers (no calibrated probabilities) are stored distinctly and never treated as
probabilities.

## Scheduling and cost

Embedding and annotation have separate configs and share the capacity-aware scheduler
(`packages/capacity/schedule`):

- modes: `off`, `passive` (on-demand only), `idle`, `balanced`, `eager`
- model-affinity grouping and dwell to avoid hot loads; the pack declares
  `runtime.model_affinity`, so changing the decision provider does not silently keep an old group
- rolling-hour budgets for seconds, jobs and model loads
- broker admission with no-reclaim for background work
- explicit backfill horizon (annotation default 30 days; retrieval default 90 days)
- rotation-aware selection: a bounded pass prefers runs this pack has not answered yet,
  then the least recently answered, so the newest rows cannot starve older runs inside
  the horizon
- checkpoint and crash recovery

Agent Manager's annotation scheduler defaults to `passive`; set
`AGENT_MANAGER_ANNOTATIONS_SCHEDULE_MODE=idle|balanced|eager` to enable background passes.
`/api/v1/annotations/scheduler` reports queue depth, deferral reasons and budget.
`runtime.digest_chars` bounds the state JSON sent to the decision model. When a model still
reports `context_overflow`, the engine splits the question set and retries with a smaller
digest once before marking the label unavailable, so a small-context model degrades coverage
instead of failing the run. A transport failure (timeout, gateway restart) is retried once in
smaller state groups, so one slow item cannot discard the rest of a batch. The default pack
sizes `batch.max_states` and `batch.max_questions` for the declared local model rather than
for a large hosted one.

## Measuring a gate

`annotations gold` writes a blind draft: runs already observed under the current definition,
their bounded state, and no stored answer. The reviewer's state is wider than the model's
(6,000 chars, more messages and a longer last error): blindness protects the stored answer, not
the evidence, and a label can only be measured when a person can actually judge it. It defaults to the label's declared `min_gold`
(capped at 100), so one command produces a set large enough to measure; every row starts
`"positive":null`. A reviewer
sets it from the evidence, and `annotations benchmark` refuses a file that still contains an
unfilled row, so a draft can never become a gate by accident.

A stored `unavailable` row is never reused as a cached answer: a gateway restart or a busy
lock must not freeze a label for that run forever. `observed`, `compatible` and deterministic
`not_observed` rows are reused only while their evidence fingerprint matches the bounded state
the run has now. The fingerprint covers the digest and any operator signals, so an appended
message, a new tool failure or a newly arrived signal recomputes the answer instead of serving
a stale one. Rows written before the fingerprint existed carry an empty value and are
recomputed once.

A label whose `require` guard fails after an earlier answer exists is superseded with an
explicit `not_observed` row (reason `insufficient_evidence`). That keeps coverage truthful
when the evidence behind an old answer disappears — for example when a classifier stops
treating a harness block as operator text.

The benchmark reads the durable probability vectors, so changing the threshold runs no model.
Gold rows are JSONL (`{"run_id":"…","label":"agent.spinning","positive":true}` or
`{"run_id":"…","labels":{…}}`); the output is the `gate.measured` block to paste into the pack.
For a score label the benchmark applies the declared friction direction, so gold `positive`
means the friction event, not merely a confident answer. The emitted block includes
`definition_hash` and the decision `threshold`. The threshold defaults to the label's declared
friction or search threshold, so a gate always measures the number consumers apply; if the
scored rows span more than one revision the benchmark refuses to emit a gate.

The friction read carries the same gated coverage: `usage friction` and the
`agent-manager/usage/friction` binding include an `inferredAnnotations` section, so the
friction digest reports label coverage beside the deterministic fingerprints without a
second call. A label has a rate only when its gate is proven.

## Gates and honesty

A label surfaces as a rate or finding only when its gate is proven on a gold set disjoint
from tuning: precision ≥ 0.80, recall ≥ 0.70, at least `min_gold` rows, per label, per model,
per definition hash. `gate.measured` carries the `definition_hash` it measured; a measurement
without the current hash is dropped at compile time, so changing a label can never inherit its
old proof. Unproven, insufficient-gold and unmeasured labels are shown as
evidence with their measured numbers; they never produce a rate. Coverage is always printed;
an empty window is unknown, not zero friction.

Coverage counts only rows for the loaded definition. A row left from an earlier revision is
reported as `stale` and excluded from observed counts, friction positives and the run coverage
ratio, so changing one label cannot make an unrelated one look measured. Re-annotating under
the new definition (or declaring `compatible_with`) is what turns those rows back into
evidence.

A score label declares its friction direction. A signed low-is-bad score uses
`friction.max_score` (for example `-1` for `operator.valence`, so praise can never be
friction). A high-is-bad score uses `friction.min_score` (for example `2` for
`agent.spinning`, so mild repetition is not friction). An enabled score label without a
direction is rejected at pack load: confidence alone must never turn a neutral or positive
score into friction.

The confidence for a score label is the probability mass in the declared region, not the
winning level's probability. On a five-level scale the top level is rarely above 0.6 even
when the model is certain the score is in the friction direction; the mass answers the
question the gate actually asks. Choice and noul labels keep their top probability. The
per-observation `friction_probability` in `annotations show` (and in the coverage selectors)
is that mass, so the CLI, UI and digest apply the same rule.

## Affect

The primary affect labels are `operator.valence` (score -2..+2),
`operator.frustration` and `operator.satisfaction` (noul). Frustration and satisfaction read
the stored operator signal first; when no signal exists for the run they fall back to their
declared decision question instead of staying unavailable, and the observation records
`method=decision` so the two sources stay distinguishable. They are text signals about the
operator's message, not a measurement of the person. Six-way `operator.emotion` is optional:
small local decision models are weak at it (~0.33–0.60 on public emotion sets) and it stays
unproven until its gate passes. Local Laya answers binary/ordinal affect well; the Ollaya
rights policy must classify the model before ai-gateway will use it.

## Surfaces

```bash
agent-manager annotations packs
agent-manager annotations labels
agent-manager annotations coverage --window 7d
agent-manager annotations show <run-id>
agent-manager annotations run <run-id>
agent-manager annotations evaluate --since <RFC3339> --limit 10
agent-manager annotations scheduler
agent-manager annotations gold --label agent.spinning --window 30d --out gold.jsonl
agent-manager annotations benchmark --gold gold.jsonl --label agent.spinning --threshold 0.6
agent-manager usage annotations --window 7d
```

`usage friction` appends the inferred-annotations section after the tool findings and
operator interventions. Label filters use the existing conversation-search tags
(`--tags label:agent.spinning`); the projection is refreshed after an annotation pass and
never replaces the annotation store as the source of truth.

Measures: `usage.annotation_coverage` and `usage.annotation_rate`.

## Skills

- `friction-analysis` — the four-lens investigation method.
- `annotation-tuning` — add/change/gate/schedule a label.
