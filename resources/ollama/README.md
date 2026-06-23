# Ollama Resource

Managed Ollama runtime for local model serving and inference workloads.

## Intent

- Resource ID: `ollama`
- Category: `ai`
- Driver: `docker-service`
- Portability tier: `partial`

## Use Cases

- Serve local models for private or offline scenario workflows.
- Provide local chat, generation, and embedding endpoints to scenarios.
- Reduce dependence on hosted AI providers for development and internal tooling.

## Architecture

This resource follows the `docker-service` structure. Ollama runs exclusively as a
Docker container — there is no host-systemd install and no `lib/` shell layer.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `model-policy.json` is the declarative authority for Ollama roles, concrete model catalog entries, capacity estimates, and estimate provenance.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Ollama-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Ollama-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Ollama
- `cli/internal/runtime`: runtime and model-state shaping helpers
- `cli/internal/status`: richer Ollama status interpretation
- `cli/internal/health`: Ollama-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers
- `cli/internal/ensure`: model auto-provisioning triggered by scenario dependencies

## Usage

```bash
# Install or update the resource contract
vrooli resource install ollama

# Check status through the shared control plane
resource-ollama status

# Default API endpoint
curl http://localhost:11434/api/tags
```

## Model provisioning

`model-policy.json` defines the shared model roles that scenarios should use
instead of hard-coding concrete model names:

| Role | Current model | Purpose |
|---|---|---|
| `embedding.default` | `nomic-embed-text:latest` | semantic search embeddings |
| `chat.small` | `llama3.2:3b` | low-memory local generation |
| `chat.default` | `qwen3:4b` | default local chat/synthesis |
| `summarize.default` | `qwen3:4b` | text distillation and summaries |
| `rerank.llm_fallback` | `qwen3:4b` | fallback reranking when the reranker resource is unavailable |
| `code.local` | `gemma4:12b` | local code-specialized generation (tool-calling capable) |

The catalog includes static capacity estimates today. Each estimate carries
provenance and confidence so later `/api/show`, `/api/ps`, and measured-profile
ingestion can update the same fields without changing the schema.

Scenarios declare the Ollama models they need in their `.vrooli/service.json`
under the `ollama` dependency block:

```json
"ollama": {
  "type": "ollama",
  "enabled": true,
  "required": false,
  "startup_policy": "try_start",
  "model_roles": [
    "embedding.default",
    {"role": "chat.default", "reason": "answer synthesis"}
  ]
}
```

Direct concrete models remain an escape hatch, not the preferred path. Use
`models` only with exception metadata:

```json
"models": [
  {
    "name": "gemma4",
    "tag": "12b",
    "reason": "code-specialized local generation",
    "owner": "agent-manager",
    "review_after": "2026-09-01"
  }
]
```

The singular `model` field is deprecated and is kept only so old manifests can
surface a warning instead of failing abruptly.

On `vrooli scenario start`, the orchestrator passes the Ollama dependency block
to `resource-ollama ensure --config-base64 <base64-json>`. The ensure verb:

1. Resolves `model_roles` through `model-policy.json`.
2. Emits warnings for deprecated `model` usage and direct `models` exceptions.
3. Lists installed tags via `GET /api/tags` (fast, ~10ms).
4. Computes the missing set.
5. Streams `POST /api/pull` for each missing model, relaying progress to the
   lifecycle console.
6. Exits 0 once every requested model is present (or reports which pulls
   failed while keeping the scenario start best-effort via the usual
   `startup_policy` semantics).

Direct invocation (e.g. while debugging):

```bash
resource-ollama ensure --config-base64 $(echo -n '{"model_roles":["chat.default"]}' | base64)
```

All log lines from the ensure path are prefixed with `ollama-ensure:` so
`grep` over `vrooli logs` surfaces the auto-provisioning flow cleanly.

## Policy metadata

`resource-ollama policy` is the programmatic interface for role and model facts
stored in `model-policy.json`. Use it from scripts and shared helpers instead
of copying dimensions, context windows, or capacity estimates into consumers.

```bash
resource-ollama policy resolve --role embedding.default --json
resource-ollama policy resolve --role embedding.default --field embedding_dimensions
resource-ollama policy resolve --model nomic-embed-text:latest --json
resource-ollama policy roles --json
resource-ollama policy models --json
resource-ollama policy constraints --json
```

The JSON form is the stable contract. A resolved role includes the selected
model, required and provided capabilities, embedding dimensions when the model
supports embeddings, context-window tokens when the model supports generation,
capacity estimates, provenance, schema version, and the policy file path. The
scalar `--field` form is only for simple shell paths.

### Embedding retargeting

Embedding roles are stable inputs; resolved model names and dimensions are
policy facts. When an embedding role changes, stored vectors must be treated as
stale even if the new model has the same dimension, because equal dimensions do
not imply the same vector space.

Use the dry-run planner before changing production stores:

```bash
resource-ollama policy retarget-plan \
  --role embedding.default \
  --old-model <previous-resolved-model> \
  --old-dimensions <previous-dimensions> \
  --old-schema-version <previous-policy-schema-version> \
  --store qdrant:<collection> \
  --store postgres:<table>.<embedding_column> \
  --json
```

The planner classifies the change as no-op, same-shape re-embed, or
incompatible shape. Incompatible changes require shadow storage with the new
dimension, regeneration, validation, and cutover. The command is intentionally
dry-run only; destructive apply requires store-specific backups and validation.

Postgres-backed pgvector tables should pair each embedding column with
`embedding_metadata` rows carrying `embedding_role`, `embedding_model`,
`embedding_dimensions`, `embedding_policy_schema_version`,
`source_content_hash`, and `generated_at`. SQL schemas must not hard-code
current Ollama dimensions; the metadata records what generated existing rows
and lets the retarget planner find stale vectors.

## Resource limits and concurrency

Defaults are tuned for a single-host workstation:

| Setting | Default | Where set | Override path |
|---|---|---|---|
| Container memory cap | `12g` | `runtime.memory_limit` in `resource.json` (passed as `docker run --memory`) | Edit `resource.json` and `vrooli scenario restart ollama` |
| Concurrent requests in-flight | `4` | `runtime.env.OLLAMA_NUM_PARALLEL` | Same — edit and restart |
| Models kept resident | `3` | `runtime.env.OLLAMA_MAX_LOADED_MODELS` | Same |

The 12 GiB cap is intended to keep one 7-8B model resident plus headroom; raise
it on hosts with more RAM, lower it on smaller boxes. Keep `OLLAMA_NUM_PARALLEL`
in step with the gateway semaphore (see below) — they are deliberately tied.

## Capacity planning

Use the planner before adding a new Ollama dependency or when reviewing model
churn across scenarios:

```bash
resource-ollama capacity plan --scenario prompt-manager
resource-ollama capacity plan --all-scenarios --json
```

The planner reads scenario `.vrooli/service.json` Ollama dependency blocks,
resolves `model_roles` through `model-policy.json`, adds documented direct
model exceptions, samples the shared host inventory, and best-effort reads
Ollama `/api/tags` plus `/api/ps`. It reports distinct model demand, installed
and loaded models, estimated disk/RAM/VRAM usage, policy estimate provenance,
runtime settings, and warnings for direct models, low-confidence estimates, and
likely unload/reload churn. It exits non-zero when resident model estimates
exceed the configured runtime or host policy budget.

## Gateway access (callers)

Scenarios MUST reach Ollama through the resource CLI, not by constructing
`/api/embeddings` / `/api/generate` URLs directly. The CLI fronts the daemon
with a host-wide cross-process semaphore sized to `OLLAMA_NUM_PARALLEL`, so the
fleet of scenarios cannot overwhelm the daemon even when individual scenarios
forget to bound their own fan-out:

```bash
resource-ollama gateway embed    --role embedding.default --json --input "hello"
resource-ollama gateway generate --role chat.default      --json --prompt "say hi"
resource-ollama gateway chat     --role summarize.default --json --system "Be concise" --prompt "summarize this"
```

`--role` and `--model` are mutually exclusive. Use `--role` for normal
scenario runtime calls so model changes stay centralized in `model-policy.json`.
Keep `--model` only for explicit direct-model exceptions and tests.

If `resource-ollama` is not on `$PATH` or the daemon is unhealthy, the gateway
fails fast with a structured error. There is no HTTP fallback by design.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for model workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- New logic lands in Go under `cli/internal/...`; the deployment/management axis stays in the Docker driver and `resource.json`.
- Keep user-facing model and API guidance in the existing docs set, and use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/ollama/docs/OPERATIONS.md) as the architecture boundary for future migrations.
