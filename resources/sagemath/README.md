# SageMath Resource

Managed SageMath computation runtime for local mathematical and scientific workflows.

## Intent

- Resource ID: `sagemath`
- Category: `science`
- Driver: `docker-service`
- Portability tier: `partial`

## Use Cases

- Run symbolic math, numerical computation, and notebook-backed analysis locally.
- Support science, engineering, and research scenarios that need a math runtime.
- Provide a reusable computation service for workflows that should not depend on hosted tools.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for SageMath-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add SageMath-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to SageMath
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer SageMath status interpretation
- `cli/internal/health`: SageMath-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install sagemath

# Check status through the shared control plane
resource-sagemath status
```

Connection defaults:

- Jupyter: `http://localhost:8888`
- API: `http://localhost:8889`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for notebook or computation workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/sagemath/docs/OPERATIONS.md) as the architecture boundary for future migrations.
