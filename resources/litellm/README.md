# LiteLLM Resource

Managed LiteLLM proxy runtime for routing AI traffic across multiple upstream providers.

## Intent

- Resource ID: `litellm`
- Category: `ai`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Route scenario traffic across multiple upstream AI providers behind one endpoint.
- Expose an OpenAI-compatible gateway for tools that expect a single provider surface.
- Add fallback, cost-control, and provider-switching without changing every scenario client.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for LiteLLM-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add LiteLLM-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to LiteLLM
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer LiteLLM status interpretation
- `cli/internal/health`: LiteLLM-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install litellm

# Check status through the shared control plane
resource-litellm status

# Default proxy endpoint
curl http://localhost:11435/health
```

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for provider routing workflows.
- Keep runtime config and logs rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/litellm/docs/OPERATIONS.md) as the architecture boundary for future migrations.
