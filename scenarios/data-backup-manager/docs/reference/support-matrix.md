# Platform support matrix

This matrix is a claim boundary, not a compile target list. A cross-build proves
that the package compiles; it does not prove that a platform can safely inspect,
write, or restore a destination. A cell is advertised as supported only when a
native runtime result or a deterministic adapter fixture exists.

| Feature slice | Linux | macOS | Windows |
|---|---|---|---|
| API/CLI build | supported; native CI/runtime | cross-build only until native runner evidence | cross-build only until native runner evidence |
| Filesystem capture/restore | supported | adapter tests; native restore evidence required before claim | adapter tests; native restore evidence required before claim |
| SQLite capture/restore | supported | adapter tests; native restore evidence required before claim | adapter tests; native restore evidence required before claim |
| Kopia filesystem repository | supported through `resource-kopia` | partial; repository/runtime evidence required | partial; repository/runtime evidence required |
| S3/MinIO repository | supported when resource and credentials are ready | partial; resource runtime evidence required | partial; resource runtime evidence required |
| Volume identity | Linux `lsblk` metadata (UUID, label, model, serial when exposed) | `diskutil info` adapter | identity remains uncertain until a native adapter is installed |
| Destination repair | read-only diagnosis and subdirectory preparation only | diagnosis/plan only | diagnosis/plan only |
| Postgres/Redis/Qdrant/object storage | resource CLI integration, each independently gated | partial until resource CLI runtime evidence | partial until resource CLI runtime evidence |
| Symlink/permission fidelity | declared by source and restore tests | platform-specific evidence required | platform-specific evidence required |
| Scheduler/credential catalog | supported by portable Go/catalog contracts | build only until lifecycle evidence | build only until lifecycle evidence |

NTFS, exFAT, FAT32, APFS, HFS+, and ext4 are not interchangeable. FAT32 is
warning-only because its 4 GiB file limit and removable installer-media role are
unsafe defaults for a primary repository. DBM never formats, repairs,
partitions, clears, or silently remounts a volume.

## Evidence commands

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./api
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ./api
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./api
vrooli scenario test data-backup-manager
storage-manager validate scenario data-backup-manager
```

Native runtime claims must additionally record the host, date, fixture or
repository used, and the restore result in the execution ledger. A partial
resource-backed source result is visible in preflight and must not be rolled up
into a full-platform claim.
