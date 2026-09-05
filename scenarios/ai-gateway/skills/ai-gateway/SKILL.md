---
name: "ai-gateway"
description: "Use AI Gateway for every model call: pick the role and profile for the need, keep the schema inside the enforceable subset, choose single or batch, choose a program's ai.* helper or the CLI, read cost and usage from the response, respect privacy classes, and record what the call taught in the ai-gateway-usage scope."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["ai-gateway", "inference", "typed-inference", "routing", "role", "profile", "privacy", "embedding", "batch", "cost", "learning-spine"]
  icon: "terminal"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  learning:
    scope: "ai-gateway-usage"
    capture: "every attempt"
  requires:
    scenarios: ["ai-gateway", "program-runtime", "vrooli-memory", "prompt-manager"]
    commands: ["ai-gateway inference run", "ai-gateway inference run-batch", "ai-gateway inference embed", "ai-gateway routing preview", "ai-gateway routing execute", "ai-gateway routing evidence-list", "ai-gateway routing evidence-show", "ai-gateway routing health", "ai-gateway inventory roles", "ai-gateway inventory smoke", "ai-gateway gateway validate", "ai-gateway conformance scan", "vrooli-memory recall wake", "vrooli-memory recall recall", "vrooli-memory journal note", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Tools focus: AI Gateway

Use AI Gateway for every model call a scenario or an agent makes: it is the only sanctioned path to Ollama and OpenRouter, and the only place route evidence is recorded. This skill holds the judgment `--help` does not: which role and profile fit the need, when a schema will be refused, when batch beats single, when a program's `ai.*` helper is the layer, and how to read cost from the response. Flags are in the CLI (`ai-gateway <group> <command> --help`); vocabularies are in `path:scenarios/ai-gateway/docs/reference/roles-profiles-policies.md` and `path:scenarios/ai-gateway/docs/reference/typed-inference-schema-subset.md`.

Required reading:
- `prompt-manager skill read vrooli-memory` — the mechanics of the learning spine this skill uses.
- `path:scenarios/program-runtime/docs/guides/program-construction.md` §"Typed inference and contract aliases" — the `ai.*` helpers that call this gateway from a program.

### 1. Scope

**In scope:** choosing role, profile, privacy class, and request kind; typed inference and embeddings; previewing and reading routes; reading usage and cost; what to journal.

**Out of scope:** editing a resource's `model-policy.json` (the resource owns it); adding a provider; regulating the gateway itself (`ai-gateway-improve`); calling a provider directly (a conformance violation).

### 2. Before acting

1. `vrooli-memory recall wake --scope ai-gateway-usage`. [S1]
2. When the call names a role or a provider: `vrooli-memory recall recall "<role or provider>" --scope ai-gateway-usage --limit 5`. [S1]
3. Apply what the entries say before choosing a leaf.

### 3. The decision tree

```
I need a model to do something
│
├─ Am I inside a program-runtime program?
│   └─ yes → ai.classify / ai.extract / ai.judge / ai.write (labels= or schema=); texts=[...] for a batch   [S3, program-runtime]
│            The helper is the same governed route; do not shell out to this CLI from a program.
│
├─ What is the output?
│   ├─ one label from a closed set        → role classify.fast, --schema {"type":"string","enum":[...]}    (§3.1)
│   ├─ fields the caller reads            → role extract.structured, an object schema                       (§3.1)
│   ├─ a verdict about a text             → role judge.default, a small enum or boolean schema             (§3.1)
│   ├─ prose                              → role write.default (write.diverse when variety is the point)   (§3.1)
│   ├─ vectors                            → ai-gateway inference embed --texts texts.json                  [S1]
│   └─ a chat or summary with no schema   → ai-gateway routing execute --role chat.default|summarize.default --input-text "<t>"  [S1]
│
├─ Typed inference (inference run | run-batch | embed) takes role and schema only: the role catalog picks the
│   candidates. Profile, privacy class, cost ceiling, and caller attribution are not flags on it.           (§3.1)
│
└─ Routing calls (routing execute | routing preview | gateway validate) take the routing flags:
    ├─ Which profile?                                                                                       (§3.2)
    ├─ Which privacy class?                                                                                 (§3.3)
    └─ Do I need to know the route before paying for it?  → ai-gateway routing preview --role <r> --profile <p> --privacy <c>  [S1]
```

#### 3.1 Typed inference

`inference run` accepts `--source`, `--schema`, `--instruction`, `--role`, `--temperature`, and `--max-output-tokens`. `inference run-batch` accepts `--items`, `--schema`, `--instruction`, and `--role` only: it takes no sampling flags at all, so a batch always runs at its role's declared sampling. There is no `--profile`, `--privacy`, `--max-cost-usd`, `--scenario`, or `--operation` on them: the role's catalog entry orders the candidates (local first for `classify.fast`), and attribution comes from the caller's identity header, not a flag. A caller that needs a profile or a cost ceiling on typed work uses `routing execute --kind extract` and validates the schema itself, or files W1 against ai-gateway for the flags.

```
Typed inference
├─ Write the schema; check it against the subset (type, enum, const, required, properties, items, pattern, minimum, maximum only)   [S0]
├─ Put intent in --instruction; a schema `description` is metadata, never an instruction                                          [S0]
├─ How many sources?
│   ├─ 1            → ai-gateway inference run --source "<text>" --schema s.json --instruction "<i>" --role <role>                [S1]
│   └─ 2 or more    → ai-gateway inference run-batch --items items.json --schema s.json --instruction "<i>" --role <role>        [S1]
│                     items.json is a JSON array of {"source": "<text>"}; order is preserved
└─ Read: value_json only when validated is true; usage.input_tokens, usage.output_tokens, usage.cost_micros (USD micro-units; local = 0);
         applied.temperature_sent with applied.temperature_support                                                                  [S0]
```

Cost and usage: `--json` returns `usage.input_tokens`, `usage.output_tokens`, and `usage.cost_micros`; divide by 1,000,000 for USD. A local route reports `0`. The human output prints one `applied=` line; read `temperature_sent` together with `temperature_support`, because a value sent to a provider that declares `ignored` had no effect.

Sampling: `classify.fast`, `extract.structured`, and `judge.default` are deterministic and refuse `--temperature` with `INVALID_REQUEST`; only `write.default` and `write.diverse` accept it. Omit `--temperature` to get the role's declared sampling; `0.0` is a request, not an omission.

#### 3.2 Profile (routing execute, routing preview, gateway validate)

| Need | Profile | What it refuses |
|---|---|---|
| Default for internal data | `local-first` (the default) | hosted fallback for sensitive data unless the caller allows it |
| Data must not leave the host | `local-only` | every hosted provider; fails closed when local capacity is rejected |
| Best answer, cost second | `quality-first` | routes that violate privacy or a `--max-cost-usd` ceiling |
| Cheapest acceptable | `cheap-first` | routes that lack the role's capability |
| Only hosted models | `remote-only` | local providers |
| Confidential or secret content | `privacy-sensitive` | unapproved hosted providers; never falls back remote for capacity |

Add `--max-cost-usd` when the caller has a budget; the route refuses instead of overspending. Add `--scenario <slug> --operation <label>` on every routing call from a scenario: evidence and future cost measures group on them. `inference run` accepts none of these; see §3.1.

#### 3.3 Privacy class (routing execute, routing preview, gateway validate)

| Content | `--privacy` | Consequence |
|---|---|---|
| Public text | `public` | any eligible provider |
| Internal project text (default) | `internal` | local-first keeps it local when a local role is eligible |
| Credentials, customer data, unreleased plans | `confidential` | hosted providers only when the profile explicitly approves them |
| Secrets | `secret` | fails closed unless a local route is eligible; never remote |

### 4. Reading routes

| Need | Command | Read it as |
|---|---|---|
| Why did a request go where it went | `ai-gateway routing evidence-list --scenario <slug> --limit 20`, then `routing evidence-show <event-id>` | `policy_reasons` in order; `selected_locality`; `breaker_state`; `capacity_verdict` |
| Which providers are suppressed right now | `ai-gateway routing health` | `effective: open` or `half_open` means the provider is skipped or probing |
| Which roles exist per provider | `ai-gateway inventory roles [--provider ollama]` | `status=available` is executable; `declared` is gateway-owned typed roles |
| Is a provider alive | `ai-gateway inventory smoke --provider <p>` | one bounded call; do not loop it |
| Would this request be accepted | `ai-gateway gateway validate --role <r> --profile <p> --privacy <c>` | a refusal names the construct |
| Does my scenario bypass the gateway | `ai-gateway conformance scan --scenario <slug>` | each finding names the file and the rule |

### 5. After acting, always

One entry per call site attempt, success or not:

```
vrooli-memory journal note "<two lines>" --scope ai-gateway-usage --kind <role-note|provider-note> \
  --trigger "<scenario/operation: what the model had to produce>" \
  --approach "<role; profile; privacy; single|batch; schema shape>" \
  --evidence "<event_id or request_id; provider/model; validated; cost_micros>" \
  --outcome "<validated|validation_failed|refused:<construct>|unavailable:<provider>; next time: <one line>>"
```

Entry kinds: `role-note` when the lesson is about a role or schema; `provider-note` when it is about a provider, breaker, or capacity. Curation leaves: pin a role-note on its third confirmation; supersede a note whose profile advice stopped routing; propose a rule when one operation label keeps needing one facet (`prompt-manager skill read vrooli-memory` §2). `run vrooli-memory.scope-bootstrap` creates starter rules for this scope. [S3]

### 6. In-use settings

| Symptom | Move | Journal |
|---|---|---|
| `validated: false` with a schema-shaped answer | Move the intent from the schema `description` into `--instruction` | the schema and the instruction that worked |
| Schema refused as unsupported | Remove the named construct (`additionalProperties`, `format`, `$ref`, `oneOf`, length bounds) | the construct removed |
| Route went remote for internal data | On a routing call, add `--profile local-only` or `--privacy confidential`; on typed inference, pick a role whose catalog entry is local-first (`inventory roles --provider ollama`) | the profile or role and the evidence event id |
| Cost too high on a repeated call | On a routing call, add `--max-cost-usd` or `--profile cheap-first`; on typed inference, switch to `run-batch` or a cheaper role | the ceiling or role and the resolved model |
| Many identical schema calls | Switch to `run-batch` with one items file | the item count and wall time |
| Local role keeps timing out | `routing health` first; if the breaker is open, use `local-first` and let the fallback happen; do not force `remote-only` for internal data | the breaker state and cooldown |

### 7. Debug order

1. `vrooli scenario status ai-gateway`.
2. `ai-gateway routing health` — breaker state per provider and role.
3. `ai-gateway routing preview --role <r> --profile <p> --privacy <c>` — is any route eligible.
4. `ai-gateway inventory roles --provider <p>` — is the role `available` on that provider.
5. `ai-gateway routing evidence-list --scenario <slug> --limit 5` — what happened to the last calls.

### 8. Safety

- Never call `resource-ollama` or `resource-openrouter` directly from a scenario; the conformance scan reports it.
- Never put a secret in `--source`, `--input-text`, or `--instruction`; evidence redacts prompt bodies, not your shell history.
- `secret` and `confidential` content stays on `privacy-sensitive` or `local-only`.
- `--max-cost-usd` is a ceiling, not a budget: a refusal is the correct outcome when nothing fits.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `INVALID_REQUEST` naming `sampling.temperature` | `--temperature` on a deterministic role | the role | Drop the flag, or use `write.default` |
| `INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING` | Overridable role, but no candidate honors temperature | `applied.temperature_support` | Accept the role's sampling; journal a provider-note |
| `INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA` | A construct outside the subset | the error names it | Remove it; the gate never degrades silently |
| `INFERENCE_ERROR_CODE_VALIDATION_FAILED` with usage present | The provider answered off-schema | `value_json` | Tighten `--instruction`; retry once at most; usage was still charged |
| `INFERENCE_ERROR_CODE_UNAVAILABLE`, or `ai.*` fails closed naming a bridge or provider | Provider down or breaker open; the gateway does not fall back outside the profile | `routing health` | Wait for cooldown or pick a profile with an eligible provider; never call the provider directly |
| `capability_mismatch` | Attachment modality not declared by the selected role | `inventory roles` capabilities | Use a role with the modality (`vision.default`, `locate.visual`) |
| Local route rejected for capacity | Broker verdict `insufficient_capacity` or `advisory_reclaim_unavailable` | `routing evidence-show <id>` `capacity_verdict` | `local-first` falls back remote if the privacy class allows; `local-only` and `privacy-sensitive` fail closed by design |
| `routing execute` ignores temperature | The routing CLI has no `--temperature` flag | `docs/reference/cli-commands.md` | Use `inference run` for sampled typed output |
| Measures binding in a program says `proto: syntax error ... unexpected token` | `window` passed as a string | the call | Pass `window={"token": "TIME_WINDOW_TOKEN_LAST_7D"}` |
