# Ollama Models Guide

How models are selected, pulled, and managed for the Ollama resource.

Ollama runs as a Docker container, so there is no host `ollama` CLI. Pull models
inside the container with `docker exec ollama ollama pull <model>`.

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
```

- Scenarios should declare model **roles** in their ollama dependency config
  rather than naming a model. `resource-ollama ensure` then pulls any missing
  resolved models automatically.
- Use ad-hoc `docker exec ollama ollama pull <model>` only for local
  experimentation; anything a scenario depends on belongs in the policy.

## Model management

```bash
docker exec ollama ollama list                 # list installed models
docker exec ollama ollama show <model>         # show model details
docker exec ollama ollama rm <model>           # remove a model
docker exec ollama ollama pull <model>         # (re-)pull a model
```

Model files live in the `${RESOURCE_DATA_DIR}` → `/root/.ollama` bind-mount and
persist across container restarts.

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
estimates and `resource-ollama capacity` enforce the container's memory ceiling.

## Advanced configuration

### Memory / concurrency

Concurrency and loaded-model limits are declared in
[`resource.json`](../resource.json) under `runtime.env` (`OLLAMA_NUM_PARALLEL`,
`OLLAMA_MAX_LOADED_MODELS`). The container's hard memory ceiling is
`runtime.memory_limit: 12g`. Edit those and `vrooli resource restart ollama`.

### Custom models

```bash
# Create a custom model from a Modelfile placed in the data volume
docker exec ollama ollama create my-assistant -f /root/.ollama/Modelfile
```

## Next steps

- [Installation Guide](INSTALLATION.md) — setup and configuration
- [Embedding Models](EMBEDDING_MODELS.md) — embedding model guidance
- [Operations runbook](OPERATIONS.md)
- [Resource docs](README.md) — current usage guidance
