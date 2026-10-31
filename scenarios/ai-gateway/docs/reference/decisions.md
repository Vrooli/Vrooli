# Decisions (decide)

`InferenceService.Decide` asks typed questions about a **state** and gets one
answer per question from a closed set, with calibrated probabilities. It is
served by non-generative **decision models**: one forward pass, no text
generation, no prompt to engineer beyond the question and option descriptions.

## When To Use Decide

| Need | Use |
|---|---|
| Pick one of 2–255 labels, with how sure the model is | `decide`, `choice` question |
| Place a text on a 2–10 level rubric | `decide`, `score` question |
| Yes/no with a probability (gate, threshold, triage) | `decide`, `noul` question |
| Many texts, same questions | `inference decide-batch`, or the `ai-gateway.decide-batch` program |
| The answer needs reasoning, counting, arithmetic or prose | typed inference (`inference run`) |
| A schema-shaped object, free text, or a long document | typed inference (`inference run`) |

Decision models read instructions literally. Phrase questions positively and
describe every option; negations and implications are where they go wrong.

## Contract

Request (`DecideRequest`): `role` (default `decide.fast` on the CLI), exactly one
of `state` (text) or `state_json` (a JSON object or array whose structure carries
meaning, such as `{"assistant": "...", "operator": "..."}`), `questions`, and an
optional `profile`.

| Question type | Declares | Answer |
|---|---|---|
| `DECISION_QUESTION_TYPE_CHOICE` | `options[{label, description}]`, 2–255 distinct labels | `choice`, `confidence`, `probabilities` per label in declared order |
| `DECISION_QUESTION_TYPE_SCORE` | `levels[]`, 2–10 descriptions, level 0 first | `score` = Σ i·pᵢ (fractional), `confidence`, `probabilities` per level `"0"…` |
| `DECISION_QUESTION_TYPE_NOUL` | optional `true_description` / `false_description` | `noul` = P(statement holds) |

Limits: 1–64 questions per request (the intersection of every candidate's
limits, so no candidate refuses what another accepts); ids are 1–100 of
`[A-Za-z0-9_.-]`; state at most 256 KiB; batches at most 256 states.

Response (`DecideResponse`): `answers` in question order; `method`
(`DECISION_MODEL`, or `LLM_ENUM` for the fallback); `provider`, `model` (the model
that answered, for example `laya:en` behind the `laya` router); `resource_role`
(which upstream); `locality` (`local` means the state never left the host);
`latency_ms` (gateway-measured, model load included); `usage`; and
`candidates`, one row per catalog candidate with `status`
(`ANSWERED`, `SKIPPED`, `FAILED`, `NOT_ATTEMPTED`) and a stable `reason`.

`confidence` is System One's normalized top probability, (K·p_max − 1)/(K − 1);
when a provider omits it the gateway derives it from the distribution. A
threshold tuned on one `model` does not transfer to another.

The gateway refuses an answer set that misses a question, answers with the
wrong type, names a label outside the declared set, reports a distribution that
does not cover exactly the declared options or does not sum to 1 (±0.02), or a
score/noul out of range. That candidate is marked `FAILED: invalid_answers` and
the walk continues.

## Roles And Candidates

Roles are `kind: "decision"` entries in `config/inference-role-catalog.json`,
walked in order by the same `ResourceRepository` that serves typed inference.

| Role | Candidates (in order) |
|---|---|
| `decide.fast` | `ollaya/decide.local` (Laya) → `ollaya/decide.jev.edenai` → `ollaya/decide.jev.typesafe` → `ollaya/decide.clef.cloudflare` → `ai-gateway/classify.fast` |
| `decide.accurate` | `ollaya/decide.local.accurate` (Winnow-E4B) → Jev (Eden AI) → Jev (TypeSafe) → Clef → `ollaya/decide.local` → `ai-gateway/classify.fast` |

`ollaya` candidates are roles of the [`ollaya` resource](../../../../resources/ollaya/README.md),
which owns the local daemon, the hosted System One client and its credentials.
A candidate is **skipped**, not failed, when:

| Reason | Meaning | Fix |
|---|---|---|
| `unconfigured` | hosted key not provisioned, or local model not pulled | `vrooli credentials provision --identity vrooli/edenai --field api-key`; `resource-ollaya models pull [--model winnow:e4b]` |
| `unreachable` | the daemon or host is down | `vrooli resource status ollaya` |
| `locality_forbidden` | `profile=PROFILE_LOCAL_ONLY` and the candidate is hosted (or the reverse) | intended |
| `unsupported_question_types` | the LLM fallback only answers choice questions | restore a decision model |

The `ai-gateway/classify.fast` candidate runs the typed classify path (schema
gate, its own local-first walk, local validation and one repair) with an
object-of-enums schema. Its answers carry the choice only: an LLM label is not a
calibrated distribution, so no `confidence` or `probabilities` are invented.

Terminal errors: `INVALID_REQUEST` (the request, naming the construct),
`CONTEXT_OVERFLOW` (every dispatched candidate would have truncated the state;
decisions on partial state are refused), `UNAVAILABLE` (nothing answered; read
`candidates`).

## CLI

```bash
cat > questions.json <<'EOF'
[{"id":"kind","type":"DECISION_QUESTION_TYPE_CHOICE","instructions":"What is the operator doing?",
  "options":[{"label":"correction","description":"Correcting or redirecting the agent"},
             {"label":"approval","description":"Approving the work"},
             {"label":"question","description":"Asking for information"}]},
 {"id":"stop","type":"DECISION_QUESTION_TYPE_NOUL","instructions":"The operator wants the agent to stop."}]
EOF
ai-gateway inference decide --questions questions.json --state "No, stop. Revert the migration file."
ai-gateway inference decide-batch --questions questions.json --items items.json   # [{"state":"..."}, ...]
```

Measured on the reference host (RTX 4070 Ti SUPER, Laya on CUDA, 2026-10-30):
one warm decision is 11–15 ms inside the gateway; the first call after the
daemon unloads Laya pays a ~3.5 s load; 20 operator messages in one
`decide-batch` took 0.43 s end to end.

## Program

`ai-gateway.decide-batch` (`.vrooli/program-runtime/decide-batch.{json,py}`)
answers up to 4 questions about up to 32 states in one governed
`ai-gateway/inference/decide-batch` call. Questions use a compact form
(`{id, type: choice|score|noul, options: {label: description} | [labels], levels, true, false}`);
`signals.results` keeps each item's answers, method, provider and locality,
probabilities are lists in the order `inputs.questions` declares, and
`signals.candidates` shows which hosted candidates were skipped and why.
