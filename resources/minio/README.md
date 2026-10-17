# MinIO Resource

Managed MinIO object storage runtime for local S3-compatible storage workflows.

## Intent

- Resource ID: `minio`
- Category: `storage`
- Driver: `managed-service`
- Portability tier: `full`

## Use Cases

- Store uploads, artifacts, and generated files behind an S3-compatible API.
- Share object storage across scenarios without depending on cloud buckets.
- Provide local object storage for development, testing, and internal workflows.

## Architecture

This resource uses the native, checksum-verified MinIO artifact through the shared
`managed-service` lifecycle. Docker is not required. The control plane defaults to
`managed-shared` on the control plane and `managed-private` in desktop bundles.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for MinIO-specific Go logic when the manifest and shared control plane are not enough.
- Historical `lib/` shell behavior has been retired; lifecycle and configuration
  behavior lives in the shared control plane and typed Go packages.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add MinIO-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to MinIO
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer MinIO status interpretation
- `cli/internal/health`: MinIO-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install minio

# Check status through the shared control plane
resource-minio status
```

Connection defaults:

- API: `http://localhost:9000`
- Console: `http://localhost:9001`

Readiness: `http://localhost:9000/minio/health/ready`.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for bucket or object workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Windows is supported at the managed-artifact and contract level; hardware-specific
  acceptance remains partial until a Windows runtime is available in CI.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/minio/docs/OPERATIONS.md) as the architecture boundary for future migrations.

## Maturity

M4 (2026-08-05): managed-service lifecycle, typed configuration, pinned native artifacts, readiness contract, and capability evidence are enforced by the fleet contract.
