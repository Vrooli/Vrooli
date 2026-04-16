# Browserless Resource

Managed Browserless Chrome runtime for browser automation and UI-driven workflows.

## Intent

- Resource ID: `browserless`
- Category: `automation`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Run headless browser automation for UI testing and workflow execution.
- Capture pages, PDFs, and rendered content that require a real browser.
- Provide a shared browser backend for scenarios and automation tools.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Browserless-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Browserless-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Browserless
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer Browserless status interpretation
- `cli/internal/health`: Browserless-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install browserless

# Check status through the shared control plane
resource-browserless status
```

Default endpoint:

- Browserless API: `http://localhost:4110`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for browser, adapter, or pool-management workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/browserless/docs/OPERATIONS.md) as the architecture boundary for future migrations.
