# Operations

`redis` is organized as a native `managed-service` resource. The Linux
artifact is acquired from a digest-pinned OCI source and launched directly;
Docker is not a runtime prerequisite.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative runtime, lifecycle, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Redis-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.
- There is no resource-local shell lifecycle or test implementation. The Go
  CLI and declarative resource contract are the supported surface.

Do not turn `cli/main.go` into the implementation surface. If the resource needs specialized runtime shaping, richer status interpretation, Redis-specific probes, or environment derivation, grow `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, or `cli/internal/env` first.

## Operator Checklist

- Keep runtime image, ports, volumes, and health checks declared in `resource.json`.
- Keep mutable runtime state in canonical resource storage paths rather than repo-local ad hoc paths.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.

## Prefix Backup Contract

Use `resource-redis dump --prefix <prefix> --output <archive>` to create a
best-effort JSON archive. Use `resource-redis restore --prefix <prefix>
--input <archive>` to restore entries. The archive records Redis `DUMP` values
and remaining TTLs. Quiesce writers when a consistent snapshot is required.

## Shell Deletion Gate

The retired `lib/`, `config/defaults.sh`, `config/messages.sh`, and
`test/integration-test.sh` files were removed after an inventory confirmed
that their `redis::` functions were only referenced inside the deleted tree.
No supported command, scenario caller, or Go test imported `common.sh`. Port,
volume, health, image, and configuration behavior remain declared in
`resource.json`; Go CLI tests cover the installed operator surface.
