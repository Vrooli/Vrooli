# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `docker-service` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `docker-service`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

## Use Cases

Replace these bullets with the real scenario-facing uses for this resource.

- Serve as the shared local runtime for scenarios that need a managed service dependency.
- Support repeatable local workflows without each scenario owning its own runtime lifecycle.
- Provide a foundation for cross-scenario service reuse across the Vrooli stack.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for lifecycle, install, invoke, freshness, exports, and runtime metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for resource-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/install`: install/bootstrap behavior unique to the resource
- `cli/internal/runtime`: runtime config/materialization logic
- `cli/internal/status`: richer status interpretation when generic lifecycle status is insufficient
- `cli/internal/health`: resource-specific probe helpers
- `cli/internal/env`: environment export and derived configuration helpers

## Next Steps

1. Replace the placeholder container image in `resource.json`.
2. Define real health checks and port mappings.
3. Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths; do not replace them with repo-local `data/` paths.
4. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.source_build`, `cli.distribution`, `cli.invoke`, and `cli.freshness`; do not add resource-local installer scripts.
5. Keep `cli/main.go` focused on bootstrap and delegation; put resource-specific logic in `cli/internal/...`.
6. Start with the standard delegated lifecycle surface. Only add custom CLI commands after the resource has a clear resource-local operator workflow that should not live in `vrooli resource ...`.
7. Define `environment_exports` for every scenario-facing variable this resource provides.
8. Run `vrooli resource validate {{RESOURCE_NAME}}` and `vrooli scenario validate-env <scenario>` before removing any compatibility paths.
9. Update `docs/OPERATIONS.md` with production runtime notes.
10. Add smoke/integration coverage for the real service behavior.

## Declaring hardware acceleration

A resource that runs work on a GPU or another accelerator declares it once, in
the `acceleration` block of `resource.json`. There is no separate `gpu` block,
no `requirements.gpu`, and no top-level `capacity` block — the control plane
reads this one declaration and nothing else.

```json
{
  "acceleration": {
    "backends": ["cuda", "cpu"],
    "require": "preferred",
    "cuda": {
      "min_compute": "8.9",
      "compose_overlay": "docker/docker-compose.gpu.yml",
      "env": { "MY_RESOURCE_DEVICE": "cuda" }
    },
    "cpu": {
      "env": { "MY_RESOURCE_DEVICE": "cpu" }
    },
    "claim": {
      "resource_kind": "vram",
      "preferred_bytes": 2147483648,
      "floor_bytes": 1073741824,
      "priority": "service",
      "yield_when_idle": true,
      "idle_grace_seconds": 900,
      "profile": {
        "steps": [
          { "label": "full", "amount_bytes": 2147483648 },
          { "label": "reduced", "amount_bytes": 1073741824 }
        ],
        "apply": { "verb": "capacity", "argv": ["degrade", "--to", "{label}"] },
        "upshift": true
      }
    }
  }
}
```

| Field | Meaning |
|---|---|
| `backends` | Ordered preference, drawn from `cuda`, `metal`, `rocm`, `vulkan`, `cpu`. The last entry is the floor the resource can always fall back to. Every name here needs a matching config block. |
| `require` | `required` fails install and start when no declared non-CPU backend is ready. `preferred` (the default) falls through to the next backend and reports mode drift. `none` makes the accelerator opportunistic, so falling back is not drift. |
| `<backend>.min_compute` | Minimum vendor compute capability. CUDA only. |
| `<backend>.compose_overlay` | Compose file the container drivers layer on when this backend is selected. Not allowed on `cpu`, which is the base compose file. |
| `<backend>.env` | Environment applied to the resource process when this backend is selected. |
| `<backend>.verify` | Overrides how the control plane reads observed placement. Leave it out; the default for the backend and the placement target is what every resource in the fleet uses. |
| `claim` | The capacity broker's reservation. A `vram` claim must declare `yield_when_idle` and a `profile` whose last step equals `floor_bytes`, otherwise the broker can never ask the resource to step down. |

Omit the whole block for a resource that does no accelerated work. Declaring
`claim` without a non-CPU backend is rejected: VRAM cannot be reserved on a
device the resource never asked for.
