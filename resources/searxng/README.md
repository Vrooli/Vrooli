# SearXNG Resource

Managed SearXNG metasearch runtime for privacy-respecting local search workflows.

## Intent

- Resource ID: `searxng`
- Category: `search`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Give scenarios a local search endpoint without binding directly to one provider.
- Run privacy-sensitive research and retrieval workflows with aggregated results.
- Support AI agents that need search access inside the local resource graph.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for SearXNG-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add SearXNG-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to SearXNG
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer SearXNG status interpretation
- `cli/internal/health`: SearXNG-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install searxng

# Check status through the shared control plane
resource-searxng status
```

Default endpoint:

- Search API: `http://localhost:8280`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for search or config workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/searxng/docs/OPERATIONS.md) as the architecture boundary for future migrations.
