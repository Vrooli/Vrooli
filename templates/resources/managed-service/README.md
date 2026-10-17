# {{RESOURCE_DISPLAY_NAME}}

{{RESOURCE_DESCRIPTION}}

This scaffold was generated from the `managed-service` resource template on {{CURRENT_DATE}}.

The shared Vrooli managed-service driver owns start, stop, status, logs, artifact verification, and provider authority. Keep this CLI limited to resource-specific commands; do not add a resource-local supervisor or shell lifecycle path.

The `control-plane` target defaults to a Vrooli-owned `managed-shared` host;
the `desktop-bundle` target defaults to `managed-private`. A desktop app may
select shared reuse only with explicit consent. Choose `managed-shared` only
when the resource can isolate every app scope and its bootstrap material can
live in the OS credential store. Declare
`external_access_capabilities` for attach-only use. External write access does
not grant lifecycle authority. Use `managed-discovered` only for an explicit,
verified executable candidate. Vrooli must never adopt a running host process
or endpoint; after verification it may launch that candidate under its own
supervision, configuration, and state.

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
