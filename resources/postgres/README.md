# PostgreSQL Resource

Managed PostgreSQL runtime for local scenario storage and multi-instance database workflows.

## Intent

- Resource ID: `postgres`
- Category: `storage`
- Driver: `managed-service`
- Portability tier: `Native on Linux, macOS, and Windows amd64; Windows on ARM unsupported`

## Use Cases

- Provide relational storage for scenarios that need SQL, transactions, and migrations.
- Run isolated local PostgreSQL instances for development or client-specific work.
- Support internal services that need a shared local database without external hosting.

## Architecture

This resource uses the managed-service structure, and Docker is never a runtime
prerequisite on any platform. The acquisition kind is declared per target
because the upstreams genuinely differ:

- **Linux (amd64, arm64)** extracts binaries, libraries, and the share tree from
  a digest-pinned official OCI source and launches them directly. The image is
  used because it carries the shared libraries the server links against.
- **macOS (amd64, arm64)** and **Windows (amd64)** stage a checksum-pinned
  upstream archive from `theseus-rs/postgresql-binaries`. The macOS archives are
  built from source on GitHub runners and load OpenSSL through `@loader_path`;
  the Windows archive repackages EnterpriseDB's official build and ships its
  ICU, OpenSSL, and libxml2 DLLs beside the executables. Both are relocatable
  and need no package manager or administrator rights.
- **Windows on ARM** is unsupported: upstream publishes no
  `aarch64-pc-windows-msvc` archive to pin.

Windows has no Unix-domain sockets, so the socket-directory argument and the
`pg_hba.conf` local-line bootstrap flag are omitted for that target. Those
differences are declared per target in `resource.json` rather than branched on
in Go. No macOS or Windows hardware run has been performed; see
[platform support](../../docs/reference/platform-support.md) for the evidence
tier of each row.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for PostgreSQL-specific Go logic when the manifest and shared control plane are not enough.
- Historical shell behavior has been retired; lifecycle and configuration
  behavior lives in the shared control plane and typed Go packages.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add PostgreSQL-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal packages:

- `cli/internal/install`: install/bootstrap helpers unique to PostgreSQL
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/config`: PostgreSQL resource configuration
- `cli/internal/content`: database content and maintenance operations
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install postgres

# Check status through the shared control plane
resource-postgres status
```

Connection defaults:

- Host: `localhost`
- Port: `5433`
- URL: `${POSTGRES_URL}` (uses the active governed credentials)

The readiness check performs a PostgreSQL startup handshake; liveness separately
checks that port 5433 accepts TCP connections. A `healthy` status does not prove
application authentication or queries succeed. See the readiness boundary in
[PRD.md](PRD.md).

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for instance, migration, or backup workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Historical shell-heavy workflows in `lib/` have been retired; new logic belongs in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/postgres/docs/OPERATIONS.md) as the architecture boundary for future migrations.

## Maturity

M4 (2026-08-16): Native tree acquisition, copy-first bootstrap, manifest
health checks, and capability evidence are enforced by the fleet contract.
