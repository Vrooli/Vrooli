# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `compose-service` resource template on {{CURRENT_DATE}}.

## Intent

- Resource ID: `{{RESOURCE_NAME}}`
- Category: `{{RESOURCE_CATEGORY}}`
- Driver: `compose-service`
- Portability tier: `{{RESOURCE_PORTABILITY_TIER}}`

Use this template when the resource needs a coordinated runtime graph instead of a single container.

## Use Cases

Replace these bullets with the real scenario-facing uses for this resource.

- Serve as the shared local multi-service runtime for scenarios that need a coordinated service graph.
- Support repeatable local workflows without each scenario owning its own compose topology.
- Provide a foundation for cross-scenario multi-service reuse across the Vrooli stack.

## Architecture

This template keeps the generated CLI thin on purpose.

- `resource.json` is the declarative authority for lifecycle, install, invoke, freshness, and orchestration metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for compose-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json` and `compose.yaml`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Generated placeholder packages:

- `cli/internal/compose`: compose-specific graph and command helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: resource-specific readiness helpers
- `cli/internal/env`: environment/export helpers

## Next Steps

1. Keep runtime state outside the repo; use `${RESOURCE_*_DIR}` or platform-native equivalents when bind mounts are needed.
2. Keep the generated CLI contract manifest-driven: `resource.json` owns `cli.command`, `cli.source_build`, `cli.distribution`, `cli.invoke`, and `cli.freshness`; do not add resource-local installer scripts.
3. Keep `cli/main.go` focused on bootstrap and delegation; put compose-specific logic in `cli/internal/...`.
4. Replace placeholder images and ports in `compose.yaml`.
5. Document real dependency and readiness semantics in `docs/OPERATIONS.md`.

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
