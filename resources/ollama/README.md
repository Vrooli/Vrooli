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

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Ollama-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

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

Scenarios declare the Ollama models they need in their `.vrooli/service.json`
under the `ollama` dependency block:

```json
"ollama": {
  "type": "ollama",
  "enabled": true,
  "required": false,
  "startup_policy": "try_start",
  "models": ["qwen3:4b", {"name": "nomic-embed-text", "tag": "latest"}]
}
```

On `vrooli scenario start`, the orchestrator sees the extra `models` key,
confirms this resource advertises `supports_ensure` in `resource.json`, and
calls `resource-ollama ensure --config-base64 <base64-json>`. The ensure verb:

1. Lists installed tags via `GET /api/tags` (fast, ~10ms).
2. Computes the missing set.
3. Streams `POST /api/pull` for each missing model, relaying progress to the
   lifecycle console.
4. Exits 0 once every requested model is present (or reports which pulls
   failed while keeping the scenario start best-effort via the usual
   `startup_policy` semantics).

Direct invocation (e.g. while debugging):

```bash
resource-ollama ensure --config-base64 $(echo -n '{"models":["qwen3:4b"]}' | base64)
```

All log lines from the ensure path are prefixed with `ollama-ensure:` so
`grep` over `vrooli logs` surfaces the auto-provisioning flow cleanly.

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

## Gateway access (callers)

Scenarios MUST reach Ollama through the resource CLI, not by constructing
`/api/embeddings` / `/api/generate` URLs directly. The CLI fronts the daemon
with a host-wide cross-process semaphore sized to `OLLAMA_NUM_PARALLEL`, so the
fleet of scenarios cannot overwhelm the daemon even when individual scenarios
forget to bound their own fan-out:

```bash
resource-ollama gateway embed    --model nomic-embed-text --json --input "hello"
resource-ollama gateway generate --model llama3.2:1b      --json --prompt "say hi"
```

If `resource-ollama` is not on `$PATH` or the daemon is unhealthy, the gateway
fails fast with a structured error. There is no HTTP fallback by design.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for model workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Keep user-facing model and API guidance in the existing docs set, and use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/ollama/docs/OPERATIONS.md) as the architecture boundary for future migrations.
