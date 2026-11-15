# Platform support matrix

This matrix is generated from `platform-receipts.json`, the control-plane host
volume receipt ledger, and the DBM runtime receipt ledger under
`$VROOLI_HOME/state/` (or `~/.vrooli/state/` when `VROOLI_HOME` is unset). A cell
is qualified only by the latest native build or runtime receipt for that
platform and capability. Cross-builds and adapter fixtures do not qualify a
native platform. Missing receipts remain explicit rather than inheriting claims
from another platform.
Regenerate with `python3 scripts/generate_support_matrix.py`; validate with
`python3 scripts/generate_support_matrix.py --check`.

<!-- BEGIN GENERATED PLATFORM MATRIX -->
| Capability | Linux | macOS | Windows |
|---|---|---|---|
| API and CLI native build | qualified · `linux-api-build-swarminator-20261113-192125` (2026-11-13T19:21:25Z) | no native receipt | no native receipt |
| Volume identity | qualified · `volume-20261114t072330000000000z-6b7b8bd4155f` (2026-11-14T07:23:30.122812055Z) | no native receipt | no native receipt |
| Read-only destination check | no native receipt | no native receipt | no native receipt |
| Filesystem repair | no native receipt | no native receipt | no native receipt |
| Mount and remount | no native receipt | no native receipt | no native receipt |
| Backup to a repository | no native receipt | no native receipt | no native receipt |
| Scratch restore and verification | qualified · `dbm-1debd04b-a125-47fd-b0ca-f0f0fe5d5603` (2026-11-14T08:43:17.103198799Z) | no native receipt | no native receipt |
<!-- END GENERATED PLATFORM MATRIX -->

The notes below describe implementation boundaries and previous fixture or
operator-run behavior. They are not qualification receipts and do not upgrade
any generated cell.

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

Recovery re-resolves stored UUID/serial identity before using the last observed
device path. This protects resume after device enumeration changes. DBM now
owns a lifecycle-managed monitor: it checks an unmounted registered volume
through the control plane and remounts only when the read-only check reports
`consistent=yes`, then verifies mounted/read-write state. If safety is not
proven, it reports the mount failure and raises an identity-bound Notification
Hub decision for the exact destructive repair. A signed callback must carry a
verified human approval; DBM then revalidates the plan and device before using
the control-plane owner, and sends a durable result notice. Direct recovery
RPCs cannot supply the repair approval. Recovery episodes persist per
destination: repeated polls reuse one journal and decision, a mounted
observation closes the episode, and a later outage receives a new recovery ID;
a completed journal also advances when an intervening mount was not observed.
These are code and fixture/test claims,
with one observed Linux Elements recovery: the exact repair decision was
approved, the control plane repaired once, and the volume was mounted read/write
through `ntfs-3g`. The result notification was delivered. This is not broad
hardware certification. macOS and Windows remain unqualified for native runtime
recovery. On Windows, the volume inventory does not expose a portable
clean/dirty bit; a successful `Repair-Volume -Scan` result is an explicit
consistency signal, while an absent or nonzero result is not clean evidence.
Do not infer DBM backup readiness from mount success: with NTFS-3G
the current host exposes no clean/dirty signal, so readiness remains UNKNOWN
until the evidence gap is resolved or an explicit owner decision changes that
gate. No backup or restore was performed.

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
