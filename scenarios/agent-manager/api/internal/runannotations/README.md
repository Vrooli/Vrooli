# runannotations

Data-driven run annotations for Agent Manager. A **label pack** declares what to
look for; evaluators produce observations. Adding or changing a label is data.
Changing an evaluator kind is code.

This package is WS-D of the Agent Manager Friction Intelligence effort. It is
deliberately a library plus a JSON HTTP/CLI surface: no proto changes, no shared
migration edits, no new dependencies.

## Vocabulary

- **Pack** (`name`, `version`, `runtime`, `labels[]`) — the vocabulary. Pack
  version is reporting only and is never a cache key.
- **Label** — one thing to observe: `id`, `group`, `scope`
  (`message|episode|run`), `method`, optional `question`, evidence selectors,
  candidate rule, sampling, gate, search facet, friction block and compatibility
  edges.
- **Definition hash** — sha256 of the canonical semantic JSON of one label:
  id, scope, method, question text/options/levels/offset, rule, source, prior
  rule, fallback, evidence selectors and candidate rule. Description, gate,
  search, friction, sampling, compatibility and optional are intentionally
  excluded, so tuning a gate or rewording a description does not invalidate an
  observation.
- **Observation** — one stored answer keyed by
  `(scope_kind, scope_id, label_id, definition_hash, model, model_version)`.
- **State** — `observed`, `compatible`, `not_observed`, `unavailable`. Unknown
  is never stored as zero or false.

## Compatibility

Resolution order when evaluating a label:

1. **Exact hash** — an existing `observed`/`compatible` row for the current
   definition hash is reused as a compute cache.
2. **Declared compatible predecessor** — the pack lists a prior hash in
   `compatibility.compatible_with`. The newest such row is copied to the current
   hash with state `compatible` and reason `reused compatible predecessor …`.
3. **Breaking change** — no declared predecessor. The label is evaluated fresh.
   `compatibility.change` defaults to `breaking`; reuse is never inferred.

`supersedes[]` names other label ids (for example a rename) and is metadata for
consumers, not a reuse mechanism.

## Pack schema

```json
{
  "pack": {
    "name": "default",
    "version": "2026-11-18.1",
    "runtime": {
      "mode": "passive",
      "default_role": "decide.fast",
      "escalate_role": "decide.accurate",
      "escalation": {"enabled": true, "min_confidence": 0.55, "max_labels": 16},
      "batch": {"max_states": 32, "max_questions": 64},
      "backfill_horizon_days": 30,
      "budgets": {"max_runs_per_pass": 50, "max_decision_states_per_pass": 64}
    },
    "labels": [
      {
        "id": "agent.spinning",
        "group": "agent",
        "description": "…",
        "scope": "run",
        "method": "decision",
        "question": {
          "type": "score",
          "instructions": "…",
          "levels": ["0: …", "4: …"],
          "offset": 0
        },
        "evidence": [{"source": "tool_summary", "field": "repeated_paths"}],
        "candidates": "duration_min >= 8 || failed_commands >= 2 || repeated_paths >= 3",
        "sampling": {"rate": 0.35, "stratum": "candidate"},
        "gate": {"precision": 0.80, "recall": 0.70, "min_gold": 40},
        "search": {"facet": "label:agent.spinning", "min_probability": 0.6},
        "friction": {"enabled": true, "min_probability": 0.6, "cost_unit": "run", "importance": "high"},
        "compatibility": {"change": "breaking", "compatible_with": []}
      }
    ]
  }
}
```

### Question types

- `choice` — 2–255 distinct options.
- `score` — 2–10 level descriptions. `offset` maps the level index to the
  domain scale: levels 0..4 with `offset: -2` produce a `-2..+2` score.
- `noul` — `true_description` and `false_description`; the answer is the
  probability the statement holds.

### Methods

- `deterministic` — CEL over `RunFeatures` (below). A rule may return a string,
  bool or number. With `fallback: "decision"` and a question, an undecided rule
  asks the decision model; a decided rule never asks.
- `signal` — reads an existing operator signal (`source`, optional
  `source_version`). No signal rows means `unavailable`, never "not observed".
- `decision` — builds questions from the pack and asks ai-gateway
  `DecideBatch` under `PROFILE_LOCAL_ONLY` and
  `PRIVACY_CLASS_CONFIDENTIAL`.
- `composite` — CEL over `labels` (map<string,double>) and `present`
  (map<string,bool>), evaluated after other labels. Numeric result.
- `extract` — declared but returns `unavailable` (`extract_not_implemented_in_slice`).
  Generative extraction is deliberately not implemented in this slice.

### Priors

`prior_rule` is a deterministic CEL expression evaluated before an expensive
evaluator. A true prior is stored as an observation with method
`deterministic` and reason `deterministic_prior`. Priors are supported only for
`noul` labels in this slice (`objective_lost`, `blocked_by_tool`,
`misunderstanding`).

### Evidence selectors

| Source | Fields |
|---|---|
| `operator_messages` | `text`, `count`, `last_text` |
| `agent_text` | `text`, `count`, `last_text` |
| `tool_summary` | `calls`, `failures`, `failed_commands`, `repeated_paths`, `failed_tools`, `last_error` |
| `timings` | `duration_min`, `time_to_first_tool_min` |
| `deterministic_flags` | `stepback_triggers`, `tool_findings`, `compaction_count` |
| `existing_signals` | `label`, `probability` (requires `label`) |
| `workload` | `kind`, `key` |
| `terminal` | `status`, `class`, `stop_reason`, `exit_code` |

Selectors name bounded digest fields. Selecting `operator_messages` also attaches
the last few message event ids to the observation as evidence refs.

### `RunFeatures` (CEL environment)

`run_id`, `workload_kind`, `workload_key`, `harness_kind`, `run_mode`, `status`,
`terminal_class`, `stop_reason`, `exit_code` (int), `duration_min` (double),
`compaction_count`, `operator_message_count`, `agent_text_count`, `tool_calls`,
`tool_failures`, `failed_commands`, `repeated_paths` (all int),
`stepback_triggers` (list<string>), `tool_findings` (list<string>),
`existing_signals` (map<string,double>), `last_operator_message`,
`last_agent_text`, `last_tool_error`.

Cross-type numeric comparison is enabled, so `duration_min >= 8` works.
Evaluation is cost-limited (default 10,000). Test packs can compile with a lower
limit to prove enforcement.

## Digest

`BuildDigestBounded(facts, max)` produces valid JSON no larger than `max`
(default 1,800 characters). It starts with 3 operator messages (280 chars), 2
agent messages (220 chars), bounded tool errors and failed tools, then shrinks
text deterministically until the JSON fits. Counts, workload, timing, terminal
facts, deterministic flags and signal probabilities survive text shrinking. A
digest never contains an unbounded transcript.

## Storage

`Store.EnsureSchema` applies `schema.sql` idempotently. The table is
`run_annotations`; the unique key is
`(scope_kind, scope_id, label_id, definition_hash, model, model_version)`.
`Save` upserts and refuses to move a row backwards in time. Query helpers:

- `Current` — newest row for an exact hash (compute cache).
- `LatestForLabel` / `LatestForScope` — newest rows for one scope.
- `Compatible` — newest row whose hash is in the declared predecessor list.
- `Coverage` — latest state per `(scope,label)` in a window.

`value_json` stores the full `Value`, including choice/score/noul, the option or
level list, and the complete probability distribution. `probability` is NULL
when the method reports no calibrated distribution (`deterministic`, `LLM_ENUM`).
`model`/`model_version` are recorded so a model change leaves other rows intact.

## Evaluators

- **Deterministic** CEL is cost-limited and returns observed values with
  probability 1 (rule assertion).
- **Signal** aggregates an existing `operator_signals` label for the run. The
  SQLite reader picks the run's most recent classified signal version.
- **Decision** batches across runs: up to `runtime.batch.max_states` states and
  `max_questions` questions per `DecideBatch` call. The full probability vector
  is stored. `LLM_ENUM` is stored with `method=LLM_ENUM`, no probability and no
  confidence, and reason `llm_enum_no_probabilities`. Escalation to
  `escalate_role` re-asks only low-confidence states and only replaces an answer
  with a more confident one.
- **Composite** runs after decisions, up to three passes, so dependencies
  declared later in the pack can still resolve.

Candidate rules and sampling bound expensive work. `sampling.rate` is applied
deterministically from `sha256(run_id|label_id)` so a replay selects the same
runs. Labels with `optional: true` are skipped unless the caller asks for them
(`operator.emotion` is the first such label).

## HTTP

Plain JSON, consistent with the untagged surface (`/api/v1/runtime/friction`).
Errors are `{"error": "..."}`.

| Method | Path | Result |
|---|---|---|
| GET | `/api/v1/annotations/packs` | `{"packs":[{name,version,pack_hash,labels,mode,label_hashes}]}` |
| GET | `/api/v1/annotations/labels?pack=&window=` | pack identity + label definitions (`id`, `definition_hash`, `compatibility`, `method`, `scope`, `gate`, `search`, `friction`) and coverage |
| GET | `/api/v1/annotations/coverage?window=` | contract coverage: window, pack, `eligible_runs`, `observed_runs`, `coverage`, `weighting`, per-label rows, summary, notes |
| GET | `/api/v1/annotations/run/{id}` | `run_id`, `pack`, `coverage:{observed,labels}`, `labels:[{id,state,value,probability,…,gate}]` |
| POST | `/api/v1/annotations/run/{id}` | annotate one run (`force`, `include_optional`, `labels`, `max_labels`, `dry_run`) |
| POST | `/api/v1/annotations/evaluate` | bounded batch (`since`, `until`, `limit`, `run_ids`, `labels`, `include_optional`, `force`, `dry_run`) |

`window` accepts `24h`, `7d`, `30d`, `all`, `0`, a Go duration or RFC3339.
Unknown run ids return 404.

Coverage rows are:

```json
{
  "id": "agent.spinning",
  "group": "agent",
  "state": "proven|unproven|not_observed|unavailable",
  "gate": {"state":"proven|unproven|insufficient_gold|unmeasured",
           "precision":0.82,"recall":0.74,"gold":31,"min_gold":25,"reason":"",
           "target_precision":0.65,"target_recall":0.55},
  "observed": 142,
  "eligible": 818,
  "evaluated": 818,
  "rate": 0.174,
  "baseline_state": "not_measured",
  "search_facet": "label:agent.spinning",
  "min_probability": 0.6,
  "friction": {"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high"}
}
```

`rate` is null unless the gate is proven. `baseline`/`ratio` are omitted until a
random-sample baseline is measured (`baseline_state: "not_measured"`).
`coverage_basis` says whether `eligible_runs` is the annotated population (no
census wired) or a candidate census supplied through the optional `RunCensus`. When no
annotations exist at all, coverage returns 200 with `labels: []`, `coverage: 0`
and a note; it never returns 500 or "no friction". Per-label `observed` is the
positive count at the label's probability threshold; `evaluated` is the
denominator (latest rows for the label).

Evaluate returns:

```json
{
  "window": {"since":"RFC3339","until":"RFC3339"},
  "submitted": 10, "observed": 10, "unavailable": 0, "coverage": 1.0,
  "labels": [/* same coverage rows */],
  "runs": [/* diagnostic per-run views */]
}
```

`dry_run` evaluates without writing. `until` bounds selection when a
`WindowSelector` is wired; explicit `run_ids` are used as given.

## CLI

```
agent-manager annotations packs [--json]
agent-manager annotations labels [--pack NAME] [--window 7d] [--json]
agent-manager annotations coverage [--window 7d] [--json]
agent-manager annotations show <run-id> [--json]
agent-manager annotations run <run-id> [--force] [--include-optional] [--labels a,b] [--max-labels N] [--json]
agent-manager annotations evaluate [--since RFC3339] [--limit N] [--run-id ID]... [--include-optional] [--force] [--json]
```

Human output always prints coverage and gate state. An empty window prints
`unknown, not zero`; an unmeasured gate prints `gate=unmeasured`. Only a proven
gate may produce a rate, and rates are the consumer's job (WS-E).

## Tests

From `scenarios/agent-manager/api`:

```
go test ./internal/runannotations/... ./internal/handlers/...
```

From `scenarios/agent-manager/cli`:

```
go test ./domains/annotations/...
```

Covered: pack load/validation, per-label hashes, compatible reuse vs breaking,
CEL candidates and cost limit, digest bounds and validity, store idempotency and
nullable probability, decision batching with probabilities and LLM_ENUM
distinction, deterministic fallback, priors, optional labels, signals absent vs
observed, extract unavailable, composite evaluation, HTTP routes, CLI parser and
renderers.

## Integration points left for the orchestrator

1. Construct the service once in the composition root (or wiring) and pass it to
   `wiring.RouteDependencies.Annotations`:

   ```go
   svc, err := runannotations.NewDefaultService(ctx, deps.DB, runannotations.DefaultPackPath(), nil)
   if err == nil { deps.Annotations = svc }
   ```

   `wiring/routes.go` already mounts the handlers when `deps.Annotations != nil`.
2. The repository feature source leaves `DeterministicFlags.StepbackTriggers`
   empty. The existing `internal/stepback` projection (or an orchestrator
   adapter) can populate it; until then `agent.objective_lost` falls through to
   its decision evaluator on candidate runs.
3. WS-C scheduler jobs should call `Service.Evaluate` with a bounded
   `since`/`limit`/`run_ids` set. The pack's `runtime.budgets` cap the work.
4. WS-E consumers should read `Coverage` and gate state, and may use
   `search.facet`/`min_probability` to tag catalog rows. This package does not
   mutate the conversation search catalog.
5. Gold measurement is not implemented here: `gate.measured` is pack data. An
   unmeasured or insufficient gate must not produce a rate.
