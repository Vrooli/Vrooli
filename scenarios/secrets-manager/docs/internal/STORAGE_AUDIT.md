# Secrets Manager storage audit

## Storage roles

| Runtime | Metadata store | Secret-value authority |
| --- | --- | --- |
| Managed shared service | Lifecycle-provisioned PostgreSQL metadata | Credential authority (host key service or encrypted authority store) |
| Tier 2 desktop bundle | Private SQLite metadata under `APP_DATA_DIR` | Desktop secret service or explicit operator input |

The desktop database is metadata only. It does not become an alternate secret-value
store for the managed-service flow.

## Desktop storage contract

The API resolves the database with `api-core/storage` using
`ProfileDesktop`, `APP_DATA_DIR` as the controlled root, and the
variant-aware scenario namespace. The resulting database path is:

`APP_DATA_DIR/data/vrooli/<namespace>/secrets-manager.sqlite`

The API creates the data directory with mode `0700` and restricts the database
file to mode `0600`. The resolver keeps live and shadow variants separate.

## State preservation

The desktop bootstrap migrates the former
`APP_DATA_DIR/runtime/api/secrets-manager.sqlite` location only when the
resolver-owned destination does not exist. It checkpoints SQLite WAL state
before moving the database. An existing destination is authoritative and is
not overwritten.

## Validation

- `go test . -run '^TestDesktopDatabase' -count=1` validates metadata access,
  file permissions, and legacy-state migration.
- `storage-manager validate scenario secrets-manager` passes. The API now opens
  a `*database.RoutedDB`, applies its PostgreSQL schema through
  `database.EnsureSchemas`, mounts `apihttp.TestModeMiddleware`, and registers
  `devrouting.Register`. Test Genie can install a per-run test pool without
  restarting the scenario or risking the primary database.
- Seven advisory `DIRECT_SQL_IN_HANDLERS` findings remain. They identify legacy
  handler SQL that should move behind domain repositories, but they do not
  bypass the routed test-pool seam or block the fail-closed isolation gate.

## Managed-service boundary

When credential-authority status is unavailable, the Secrets Manager API returns
an unavailable response. It does not scan local files or environment variables
for a fallback result. Provisioning writes only through the credential-authority
client and accepts values from the guarded stdin path.
