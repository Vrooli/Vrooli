# Operations

`comfyui` is currently a shell-driven `docker-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, and health metadata.
- `cli.enabled` is currently `false`, so there is no active Go CLI contract for this resource yet.
- `cli.sh` and `lib/` own the current operational behavior.

If this resource is migrated to the standard native CLI model later:

- keep the generated CLI thin
- place ComfyUI-specific logic under `cli/internal/...`
- do not treat `cli/main.go` as the implementation surface

## Operator Checklist

- Keep runtime image, ports, volumes, and health checks declared in `resource.json`.
- Keep mutable runtime state in canonical resource storage paths rather than repo-local ad hoc paths.
- Avoid expanding shell behavior unless it is necessary for the current shell-driven surface.
- When native migration starts, move behavior from `lib/` into focused Go packages rather than recreating shell-era sprawl.
