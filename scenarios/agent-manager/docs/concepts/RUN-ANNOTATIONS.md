# Run annotations

Run annotations are Agent Manager's data-driven, inferred layer over run history. They
answer questions the deterministic tool and runtime lenses cannot: did a failing tool block the work, what kind of work was this, and was the operator dissatisfied,
pleased or impatient in the words they wrote?

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
| `signal` | An existing operator signal (for example `explicit_negative`) | already paid |
| `decision` | ai-gateway `DecideBatch` (choice/score/noul) under local-only, confidential | cheap |
| `composite` | CEL over other labels | free |
| `extract` | Declared, returns `unavailable`; generative extraction is on demand only | expensive |

A label may declare a deterministic `rule` with `fallback: "decision"`: the rule answers
when it can, and the model only sees the residual. `prior_rule` lets a deterministic fact
(for example a recorded tool finding) answer a `noul` label before inference runs.

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

## Roles, escalation and limits

A label may declare `role` (and `escalate_role`); both default to `runtime.default_role` and
`runtime.escalate_role`. Neither is part of the definition hash: the model that answered is recorded on each
observation (`model`, the serving `resource_role`, and the `requested_role` that was asked) and proven separately.
Labels are batched by `(role, escalate_role)`, one `DecideBatch` per role, and a stored answer asked of a different
role than the label routes to now is asked again (a row written before roles were recorded counts as the pack
default).

Escalation re-asks only low-confidence answers and **never the same model twice**. When the escalation role is the
role already asked, or resolves to the model that already answered (learned from answers, not configured), the call
is skipped and the fact is remembered for `runtime.escalation.same_model_ttl` (a Go duration such as `10m`). A call
that is made and answered by the same model teaches the engine the same thing, so it happens at most once per ttl.
`evaluate` reports `escalation: {eligible_states, calls, skipped_states, skipped_calls, same_model_answers, adopted,
call_seconds}`, so the saving is measured.

When the gateway exposes a role's model limits (`context_tokens`, `max_options`, `score_levels`, `max_questions`),
questions per call and the digest size follow the resolved model: the pack's `batch.max_questions` and `digest_chars`
stay the cap, a question with more options than the model declares is `unavailable` (`question_exceeds_role_limits`)
instead of asked, and the overflow split/shrink remains the safety net. A candidate reporting `limits_exceeded` is
handled like `context_overflow`.

What a small window keeps is the label's own choice. A run-scope decision label lists its `evidence` sections most
important first (`tool_summary`, `agent_text`, `deterministic_flags`, `operator_messages`, `existing_signals`). When the
state must shrink for the model's window, sections no asked label names are dropped first, least important first; the
named sections then shrink together, each starting after the one ranked below it, so none is starved. A label naming
nothing gets the end of the run (last tool error and failed tools, then the agent's closing words) ahead of early-run
detail. The order is part of the definition hash (with a digest-format version for the renderer), so a change to either
makes stored answers and measurements of that label stale rather than mistaking them for answers to the new input.
Message-scope labels read their own message digest and are unaffected.

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
- model-affinity grouping and dwell to avoid hot loads; the affinity is derived from the decision roles the
  pack's labels resolve to (`Service.Affinity`), so re-routing a label changes it and no separate pack-wide
  setting can drift from the roles
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

The benchmark reads the durable probability vectors on the server, so changing the threshold or the
temperature runs no model. Gold rows are JSONL (`{"run_id":"…","label":"work.blocked_by_tool","positive":true}` or
`{"run_id":"…","labels":{…}}`); the output is a `gate.measurements[]` entry to record in the pack.

Gold lives in one private directory, the gold root: `runtime.gold_root` in the pack, default `gold/` under the
scenario's durable state directory (the one holding its database, `scenarios/agent-manager/data/gold` on this host; `GET
/api/v1/annotations/packs` reports the resolved path as `gold_root`). Mood gold contains the operator's own words, so the
root is never a tracked path. `--gold`, `--fit-from`, `--runs-from` and `--out` resolve a relative path against the
root (an absolute path is used as written), `gold` writes files 0600 in a 0700 directory, and a measurement's
`gold_file` is recorded relative to the root so it survives moving the state directory.

A gold row names one subject: a run (`run_id`), or for a message-scope label one message of a run (`run_id` plus
`scope_id`, the message's event id; `annotations gold` writes both and the review packet is that message with the agent
text before it). A tuning and a hold-out file are disjoint by subject, so two messages of one run may split across them.
A gold row whose run or message has no stored answer under the current definition for the chosen candidate is accepted
and reported as `unobserved` (with the first subjects named), never dropped and never counted as a negative; `unavailable`
is a stored answer with nothing to score. `annotations evaluate --runs-from <gold file>` answers exactly those runs,
exactly those messages, and only the labels the file judges (unless `--labels` is given); with `--pin-resource-role`
the answers land in the trials table for `benchmark --pinned`. A composite over labels has no gold and is refused by the
benchmark: measure its inputs.

A measurement is for one `(definition_hash, model, resource_role)`. Rows whose stored answers span more than one are
refused with the groups named; `--model` and `--resource-role` pick the candidate. Deterministic, signal and composite
answers measure as `model: "rule"` with an empty resource role.

Calibration is fitted from the stored vectors, never from a new model call:

- A binary (`noul`) answer is tempered as `sigmoid(logit(p) / T)`; a choice or score answer's log-probabilities are
  divided by `T` and renormalized, and a score label's event is the mass in its declared friction region. `T`
  minimizes the negative log-likelihood of the gold outcomes. ECE (ten equal-width bins) and Brier are reported
  before and after.
- The threshold is chosen from the calibrated curve: the best F1 among thresholds meeting the label's precision and
  recall targets, else the best F1 overall, placed at the midpoint to the next lower score.
- `--split tuning` (default) fits on the rows and records a `tuning` measurement, which fits but never proves.
  `--split holdout --fit-from <tuning file>` evaluates the tuning file's temperature and threshold on a disjoint
  file without refitting, and records a `holdout` measurement. A hold-out sharing a run with the tuning file is
  refused. A hold-out of a decision label needs `--fit-from`; a rule needs none.
- At read time the temperature and threshold of the best measurement for an observation's own
  `(definition_hash, model, resource_role)` are applied to that observation only (`calibrated_probability`,
  `temperature`, `friction_positive`); the stored vector is unchanged and another model's observations are read raw.

```json
{"definition_hash":"…","model":"winnow:e4b","resource_role":"decide.local.accurate",
 "precision":0.0,"recall":0.0,"gold":0,"positives":0,"threshold":0.6,"temperature":1.0,
 "ece":0.0,"brier":0.0,"split":"holdout","gold_file":"<label>.holdout.jsonl","measured_at":"RFC3339"}
```

`annotations calibration --label <id>` prints the reliability table (bins, rows, mean predicted, observed rate)
with ECE and Brier for the stored vectors and the fitted temperature, from gold and stored vectors only.

A legacy `gate.measured` block is read once at pack load and migrated to a `legacy` measurement with an unknown
model: it stays visible and can never prove anything.

### Bake-offs

`annotations evaluate --pin-resource-role <role> --runs-from <gold file> [--role <role>]` re-annotates exactly the
gold runs with one declared candidate: the gateway answers with that candidate or refuses, never falling back. The
answers go to a separate table (`run_annotation_trials`), keyed by the answering model and resource role, so they
never overwrite or mix with production observations, take no escalation, and teach the engine nothing about what a
role resolves to. `annotations benchmark --pinned --model M --resource-role R` measures them and
`annotations show <run> --pinned` reads them. Rule, signal and composite labels are skipped (they do not depend on
the model); the candidate rule and sampling do not apply to named runs, the evidence guard still does.

One measure, `usage.annotation_gate`, projects coverage, gate and rate for every consumer: the Usage page, `usage
annotations`, and (through `inferredAnnotations` on `usage friction`) the friction digest. Each label row carries its
state, the model and resource role the proof is for, hold-out precision and recall, ECE, gold size and whether its
probabilities are `calibrated`, `raw` or from a `rule`; a model row follows for every model that answered. A label has
a rate only over answers from a proven model.

## Gates and honesty

A label is **proven for an observation** only when a `split: "holdout"` measurement exists for that observation's
own `(definition_hash, model, resource_role)` and clears the label's targets (precision, recall and at least
`min_gold` rows, on a gold set disjoint from tuning). Everything that reports a rate, a facet, a measure or a
digest line applies the proof of the observation's own model: coverage splits a label's observations by model
(`models[]`), and the rate counts only the observations whose model is proven. A proof never carries to another
model, another resource role or a changed definition; a measurement for an earlier definition stays in the pack as
history and proves nothing. Gate states: `proven`, `unproven` (hold-out below target), `insufficient_gold`,
`tuning_only`, `legacy_unattributed`, `unmeasured`. A label's coverage state is `proven` when every observed answer
is proven, `partial` when some are, and `unproven` otherwise; unproven answers are shown as evidence with their
measured numbers and never produce a rate. Coverage is always printed; an empty window is unknown, not zero friction.

A composite over other labels (its rule reads `labels['x']`, `present['x']` or `message_labels['x']`) has no gold of
its own and cannot declare gate measurements. Its label-level gate state is `derived`, and each observation of it is
proven exactly when every observation it read is proven for its own `(definition_hash, model, resource_role)`; if any
input is unproven the composite is unproven. The observation's `gate.inputs` carries the proof key and state of each
input (transitively, for a composite over composites), so the reason is visible. A composite over facts only (no label
reference) keeps the rule path: a hold-out under `model: "rule"`. One function decides this
(`CompiledPack.StatusFor`); coverage, views, series and message statistics all call it.

Coverage counts only rows for the loaded definition. A row left from an earlier revision is
reported as `stale` and excluded from observed counts, friction positives and the run coverage
ratio, so changing one label cannot make an unrelated one look measured. Re-annotating under
the new definition (or declaring `compatible_with`) is what turns those rows back into
evidence.

A score label declares its friction direction: `friction.max_score` for a signed low-is-bad score, `friction.min_score`
for a high-is-bad one. An enabled score label without a direction is rejected at pack load: confidence alone must never
turn a neutral or positive score into friction. The confidence for a score label is the probability mass in the declared
region, not the winning level's probability; choice and noul labels keep their top probability. The per-observation
`friction_probability` in `annotations show` is that mass, so the CLI, UI and digest apply the same rule. The default
pack asks no score questions: small decision models compress ordinal scales, so its labels are noul.

## Whose words

Most runs in the corpus are opened by a workflow brief or an agent's delegation, so reading every operator-source
message as the operator's mood would measure the briefs. `runtime.provenance` declares which text is a person's:
`brief_workloads` are brief throughout, and `seed_brief_workloads` are brief only for the run's opening message. The
policy is entirely pack data (the live pack declares `workflow_node`, `scheduled` and `adhoc`); the engine carries no
word list, so a pack that declares none reads every operator-source message as typed. Pack CEL can read `human_message_count`,
`brief_message_count` and `last_human_message`; coverage reports a `provenance` block (runs with human text, runs of
briefs only, runs with no operator text, human and brief messages) and a label that declares
`population: "human_operator"` is reported against the runs (or messages) a person wrote, not every annotated run.

Authorship comes from the data a run carries, not from a workload proxy. A message is not a person's when its run's
workload is a brief workload, when it is the seed of a seed-brief workload, when another run spawned the run
(`parent_run_id` set, not an imported transcript: an agent opened it, so an agent wrote every operator-source message,
whichever token delivered it), when the run's opening message says an agent launched it
(`delegating_openings`, the live pack declares `Agent Manager lineage:`), or when the text carries a structural marker only an agent
or supervisor writes (`agent_markers`: an epoch heading `# E37`, `Read E1 D34`, `FEEDBACK.md`, a supervisor directive; the live pack's
list was read off the corpus and none matched the 429 messages a person wrote). `delegated_by_lineage: true` turns the
lineage rule on. CEL reads `spawned_by_run` and `imported_session`. On the corpus read by hand (656
operator-source messages) this keeps all 429 a person wrote and drops 218 of the 227 an agent wrote; the 9 left are
relays inside a person's own imported session. Messages Agent Manager accepts record their sender (`sender`: person or agent, `sender_id`: the principal's subject),
stamped by orchestration from the verified caller's identity at every place a user message is accepted. A recorded
sender is authoritative and outranks all of the rules above in both directions: a person typing into a run an agent
spawned is a person, an agent steering a run is not. Text no authenticated caller sent (an imported transcript, a wake,
harness text) and messages stored before senders were recorded have none, and the lineage and marker rules decide them.

Each run's stored authorship records the hash of the policy it was computed under (`Provenance.Hash`). Editing the
policy makes the stored rows stale: coverage counts them in `provenance.runs_stale` and in no bucket, never mixing two
definitions of whose words a message was. `agent-manager annotations provenance` (POST `/annotations/provenance/refresh`)
recomputes them from facts alone, with no label evaluated and no model asked.

A label with `scope: "message"` is answered once per human operator message: the subject is the message (its event
id is the scope id, its timestamp is stored with the row, the parent run is `run_id`), and the state a model reads is
that message plus the agent text just before it. Workflow briefs are never subjects. Message-scope labels are decision,
deterministic or composite labels (a message composite reads that message's own labels); run-scope observations and
their hashes are untouched. A run composite reads `message_labels['id']`, a map of `count`, `positives`, `max` and `mean`
over the run's message observations of that label (the calibrated value where the observation's model is proven, the raw
one otherwise). The messages of a run are finished before its composites, and a composite is rewritten only when its
inputs changed: its stored fingerprint covers the run's facts, the value of each label it names, the summary of each
message label it names and the proof keys behind them.
`annotations series --label <id> --bucket day` reads a message label as a time series, with `proven_mean` separate
from `mean` until the model is proven.

## Mood

Mood is a message-scope series over a person's own words, not a run score. `operator.dissatisfied`, `operator.pleased`
and `operator.impatient` each ask one noul question per human message (`population: "human_operator"`). Everything
else is derived and stored, never asked: `operator.valence` is the message-scope composite pleased minus dissatisfied,
and the run-scope `operator.dissatisfied_run`, `operator.pleased_run` and `operator.impatient_run` are the highest
probability over a run's messages, so run mood flows through facets, friction and the digest like any other label.
They are text signals about one message, not a measurement of the person.

The Usage page (Friction tab, Operator mood) draws each message label per day from `annotations series`: a solid line
for messages read by a proven model and a dashed line for every message, which is unproven evidence until the two
agree. It states the human-message population, says "no proven model yet" instead of drawing a proven line, and treats
a window with no human message as unknown, not neutral. Label sets and scopes come from the pack; the UI holds no list.

## Surfaces

```bash
agent-manager annotations packs
agent-manager annotations labels
agent-manager annotations coverage --window 7d
agent-manager annotations show <run-id>
agent-manager annotations run <run-id>
agent-manager annotations evaluate --since <RFC3339> --limit 10
agent-manager annotations scheduler
agent-manager annotations gold --label work.blocked_by_tool --window 30d --out work.blocked_by_tool.draft.jsonl
agent-manager annotations benchmark --gold tuning.jsonl --label work.blocked_by_tool --model laya:en --resource-role decide.local
agent-manager annotations benchmark --gold holdout.jsonl --split holdout --fit-from tuning.jsonl --label work.blocked_by_tool --model laya:en --resource-role decide.local
agent-manager annotations calibration --label work.blocked_by_tool --gold tuning.jsonl --gold holdout.jsonl
agent-manager annotations evaluate --pin-resource-role decide.local.accurate --runs-from holdout.jsonl --labels work.blocked_by_tool
agent-manager annotations series --label operator.dissatisfied --window 30d --bucket day
agent-manager usage annotations --window last_7d
```

`usage friction` appends the inferred-annotations section after the tool findings and
operator interventions. Label filters use the existing conversation-search tags
(`--tags label:work.blocked_by_tool`); the projection is refreshed after an annotation pass and
never replaces the annotation store as the source of truth.

Measure: `usage.annotation_gate` (`agent-manager measures get usage.annotation_gate --window last_30d`).

## Skills

- `friction-analysis` — the four-lens investigation method.
- `annotation-tuning` — add/change/gate/schedule a label.
