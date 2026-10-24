# Ollama Models Guide

How models are selected, pulled, and managed for the Ollama resource.

Ollama is supervised as a managed service, so there is no host package install
for the `ollama` CLI. Pull models through the service API or
`resource-ollama ensure`.

## Models are governed by roles, not hard-coded lists

The authoritative model set lives in [`model-policy.json`](../model-policy.json),
which maps logical **roles** (e.g. `chat.default`, `code.local`,
`embedding.default`) to concrete model tags plus fallbacks and capacity
estimates. This indirection keeps scenarios decoupled from specific model
versions — when a better model ships, the policy is updated in one place and
every consumer follows.

Inspect the current roles and the models they resolve to (do not hard-code tags
elsewhere — read them from here):

```bash
resource-ollama policy roles                  # role -> resolved model
resource-ollama policy models                 # catalog entries + capacity estimates
resource-ollama policy resolve --role code.local
resource-ollama capacity                      # plan demand against host/runtime budget
resource-ollama models inventory --json        # named sizes/digests + policy reachability
```

- Scenarios should declare model **roles** in their ollama dependency config
  rather than naming a model. `resource-ollama ensure` then pulls any missing
  resolved models automatically.
- Use the Ollama API for ad-hoc local experimentation; anything a scenario
  depends on belongs in the policy.

### Explicit-only roles and content posture

A role normally describes a capability, not an alignment posture. The
`chat.uncensored` role is the deliberate exception: it resolves to an
abliterated community build whose refusal behaviour was reduced by a weight
edit. It exists so a caller can ask for unrestricted local generation by name.

- It is **explicit-only**. `default_eligible` is false, it sits last in
  `role_preference_order`, and no other role falls back to it.
- It requires only `generate`, `chat`, and `summarize`. The selected model
  advertises tool calling but no vision; the role does not guarantee tool use,
  so probe a model build before relying on it.
- Sampling is pinned to `temperature 0.2` / `top_p 0.9` with a 4096-token
  output cap for instruction-following and agent work. Raise temperature at
  the call site when the task needs creative variation.
- The declared 20480-token context is the measured host fit: 11.13 GiB VRAM
  with no CPU offload and ~99 tok/s. 24K needs CPU offload and previously
  caused a CUDA OOM under the old `OLLAMA_NUM_PARALLEL=2` default, so the
  resource now runs with `OLLAMA_NUM_PARALLEL=1` and
  `OLLAMA_MAX_LOADED_MODELS=2`.
- The default was chosen by the 2026-10-21 abliterated bake-off: Granite 4.1
  8B scored 13/14 on instruction, formatting, and arithmetic checks, with a
  live tool smoke pass and ~99 tok/s at 20K. The Qwen3.5-9B, Hermes-9B,
  MiMo-9B, Qwen3-14B, and Qwen3.6-27B candidates scored lower or did not fit
  the host budget.
- Its footprint is a first-class capacity rung. Adding a role that names a new
  model without materializing the matching `acceleration.claim.profile` step
  makes resource loading fail.
- Outputs are unreviewed. Keep this role out of customer-facing scenarios and
  default routing; a fallback to a safety-tuned model would silently change
  behaviour, so the role declares no fallbacks.

### Role-owned request levers

Besides the model, a role may declare three request levers. All are optional and
omission is a documented state, not an oversight.

| Key | Effect when the caller sends nothing |
|---|---|
| `sampling_defaults` | clamped temperature/top_p/top_k written into runtime config |
| `max_tokens` | becomes the request's `num_predict` |
| `sampling_support` | declares how the role's models treat an explicit control |

`--max-tokens` and `--temperature` override the role. `--temperature` uses `-1`
as its "unset" sentinel because `0.0` is a legitimate, meaningful temperature,
so "absent" cannot be encoded as a zero value.

**Without a role `max_tokens`, a caller who sends nothing is uncapped**:
`num_predict` is omitted from the request and generation is bounded only by the
model's context window. That is a real state, not a bug — but a role that wants
a ceiling has to say so. The resolved budget (flag, else role cap, else nothing)
is what `validateContextWindow` reserves, so the guard refuses by name rather
than letting the server slide its context window and truncate the prompt
silently.

`sampling_support` takes `honored`, `ignored`, `rejected`, or `unknown` per
parameter; an absent key means `unknown`, which consumers treat as `ignored`.
These are **declarations, never probes**: a provider that accepts a control and
silently discards it is indistinguishable at the call site from one that honours
it, so a successful call is not evidence of support.

## Model management

```bash
curl http://localhost:11434/api/tags
curl http://localhost:11434/api/show -d '{"name":"<model>"}'
curl http://localhost:11434/api/delete -d '{"name":"<model>"}'
curl http://localhost:11434/api/pull -d '{"name":"<model>","stream":false}'
```

Model files live in `${RESOURCE_DATA_DIR}/models` and persist across managed
service restarts.

### Storage identity and attribution

Use `resource-ollama models inventory --json` as the authoritative model census.
It reads `name`, `digest`, and logical `size` from Ollama's `/api/tags` API and
marks whether each installed tag is reachable from a policy role or fallback.
Every row is explicitly regenerable because the weights can be re-pulled from
the Ollama registry, with that reason included in the output. Do not infer
model identity by walking the `blobs/` directory: Ollama shares blobs between
models, so filesystem paths cannot safely attribute bytes to a single tag.

Storage Manager exposes the same rows at `GET /api/v1/storage/inventory` under
`ollama_models`. It also measures the configured model root and reports a
physical remainder and path when bytes cannot be attributed by the service
inventory.

## Performance by model size

Inference speed scales with parameter count and hardware, not the specific
model, so size is the useful axis when choosing a role target:

| Model Size | CPU (16 cores) | RTX 4090 | RTX 3080 |
|------------|----------------|----------|----------|
| 3–4B       | 15–25 tok/s    | 80–120 tok/s | 60–90 tok/s |
| 7–9B       | 5–12 tok/s     | 40–70 tok/s  | 25–45 tok/s |
| 12–14B     | 2–6 tok/s      | 20–35 tok/s  | 12–25 tok/s |
| 27B+       | 0.5–2 tok/s    | 8–15 tok/s   | 4–8 tok/s |

Smaller roles (`chat.small`, `classify.routing`) favour latency; larger roles
(`code.local`, `chat.default`) favour quality. The policy's `disk/ram/vram`
estimates and `resource-ollama capacity` enforce the declared host/runtime budget.

## Advanced configuration

### Memory / concurrency

Concurrency and loaded-model limits are declared in
[`resource.json`](../resource.json) under `managed_service.environment`
(`OLLAMA_NUM_PARALLEL`, `OLLAMA_MAX_LOADED_MODELS`). Edit those and
`vrooli resource restart ollama`.

### Custom models

Create custom models through the Ollama API or the managed resource CLI; keep
the source Modelfile under `${RESOURCE_DATA_DIR}` so it remains in the resource
data boundary.

## Next steps

- [Installation Guide](INSTALLATION.md) — setup and configuration
- [Embedding Models](EMBEDDING_MODELS.md) — embedding model guidance
- [Operations runbook](OPERATIONS.md)
- [Resource docs](README.md) — current usage guidance
