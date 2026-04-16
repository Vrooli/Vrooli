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

## Usage

```bash
# Install or update the resource contract
vrooli resource install ollama

# Check status through the shared control plane
resource-ollama status

# Default API endpoint
curl http://localhost:11434/api/tags
```

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for model workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Keep user-facing model and API guidance in the existing docs set, and use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/ollama/docs/OPERATIONS.md) as the architecture boundary for future migrations.
