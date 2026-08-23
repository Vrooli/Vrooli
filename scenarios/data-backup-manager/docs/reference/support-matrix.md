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
| Destination read-only cause | attributed from kernel evidence (`/proc/fs/ntfs3/*/volinfo`, `/sys/class/block/*/ro`, fstab); unattributed stays `unknown` | mount options only; cause reported `unknown` until a native adapter lands | mount options only; cause reported `unknown` until a native adapter lands |
| Destination repair | check/repair/unmount/mount via the control plane; udisks2 with no elevation, privilege broker otherwise | diskutil argv built and unit-tested; native runtime evidence required before claim | Repair-Volume argv built and unit-tested; native runtime evidence required before claim |
| Postgres/Redis/Qdrant/object storage | resource CLI integration, each independently gated | partial until resource CLI runtime evidence | partial until resource CLI runtime evidence |
| Symlink/permission fidelity | declared by source and restore tests | platform-specific evidence required | platform-specific evidence required |
| Scheduler/credential catalog | supported by portable Go/catalog contracts | build only until lifecycle evidence | build only until lifecycle evidence |

The credential-store escrow scheduler has the same explicit contract at the
control-plane boundary: Linux `systemd-user`, macOS `launchd-user`, and Windows
per-user Task Scheduler are supported adapters; other platforms return
`supported=false`, `state=degraded`, and the manual refresh remediation. A
cross-build does not upgrade a build-only cell to a native runtime claim.

NTFS, exFAT, FAT32, APFS, HFS+, and ext4 are not interchangeable. FAT32 is
warning-only because its 4 GiB file limit and removable installer-media role are
unsafe defaults for a primary repository. macOS has no read/write NTFS driver at
all, so an NTFS destination can never be writable there regardless of repair.

DBM never formats, partitions, or clears a volume, and it carries no host-repair
implementation of its own. Volume remediation — check, repair, unmount, and
mount read/write — is executed by the control plane (`vrooli host volume`),
which DBM plans, confirms and reports. Each step is a separate confirmed action:
the confirmation phrase names the device and its UUID, and `repair-filesystem`
additionally requires an explicit data-loss acknowledgement because it can
discard inconsistent filesystem metadata.

A successful repair is not proof the backups are intact. `ntfsfix` clears the
volume dirty flag and fixes a small set of known problems; it is not a `chkdsk`
replacement. Verify the repository itself (`kopia snapshot verify`) after any
repair — content-addressed verification proves far more than a filesystem check.

Two limits are stated rather than papered over. The privilege broker is Linux +
systemd only, and it offers check and repair but not mount or unmount: it runs
with mount-namespace isolation, so a mount performed there would not propagate
to the host. On a Linux host without udisks2 those two steps return a typed
unsupported result naming the exact operator command instead of appearing to
succeed.

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
