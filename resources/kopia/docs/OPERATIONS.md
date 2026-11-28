# Kopia Resource — Operations

Every capability is reachable **only** through `resource-kopia`. Do not call the
`kopia` binary directly (the single exception is disaster recovery — see
[`RECOVERY.md`](RECOVERY.md)).

Output is human-readable by default; pass `--json` to emit kopia's native JSON
(consumers parse it; the wrapper invents no schema of its own).

## Provisioning & health

```bash
vrooli setup --resources kopia     # install the pinned, checksum-verified binary
resource-kopia status              # binary present; version matches pin (exit 0)
resource-kopia version             # wrapper version + pinned + installed kopia
```

`version` warns on stderr if the installed kopia drifts below the pin (e.g. a
brew/winget install that ignored the exact version).

## Repositories (`repo`) — one repository == one Vrooli destination

Encryption is always on; the passphrase is generated and stored in the
credential authority on first `create`. S3 credentials and SFTP private keys
are stored under the same per-repository authority identity. SFTP password
authentication is unsupported because Kopia accepts that password only on
argv; provision `sftp-private-key` through the credential authority instead.

Provision S3 access keys or SFTP private keys through the control-plane
credential owner before creating or reconnecting a repository. Values are read
from stdin and do not appear in process arguments:

```bash
vrooli credentials provision --identity vrooli/kopia/offsite --field s3-access-key-id
vrooli credentials provision --identity vrooli/kopia/offsite --field s3-secret-access-key
vrooli credentials provision --identity vrooli/kopia/offsite-sftp --field sftp-private-key
```

For SFTP, `--known-hosts` supplies the host-key trust file path and is stored
as non-secret backend metadata. Keep it available on each host that must
reconnect; the private key stays in credential authority and can be
re-provisioned during clean-node recovery.

```bash
# Filesystem backend
resource-kopia repo create --name nightly --backend filesystem --path /var/backups/nightly

# SFTP backend (private key is read from the credential authority)
resource-kopia repo create --name offsite-sftp --backend sftp \
  --host backup.example --port 22 --username vrooli \
  --remote-path /backups/vrooli --known-hosts ~/.ssh/known_hosts

# S3 / MinIO backend (provision credentials through the credential authority first)
resource-kopia repo create --name offsite --backend s3 \
  --bucket vrooli-backups --endpoint minio:9000 --disable-tls

resource-kopia repo connect    --name offsite              # reconnect using registered params
resource-kopia repo status     --name offsite --json       # encryption algo, format version
resource-kopia repo stats      --name offsite --json       # size, content/object counts, dedup
resource-kopia repo damage-scan --name local --json        # registered filesystem repository
resource-kopia repo damage-scan --path /mnt/Elements/vrooli-backups/repositories/elements-local.kopia --json # explicit read-only mount; no repo credential needed
resource-kopia repo repair-format-plan --name local --json # non-mutating format-blob recovery plan only
resource-kopia repo cache-probe --name local --blob-id q... --json # isolated cache; test this exact q-blob temp in a disposable overlay; never publishes it
resource-kopia repo index-recovery-scan --name local --json # scan recoverable indexes; captures the Kopia logger summary; does not commit
resource-kopia repo index-recovery-commit --name local --confirm-commit commit-index-recovery:local --json # explicit name-bound index commit; never deletes indexes
resource-kopia repo orphan-temp-blob-commit --name local --blob-id q... --candidate-sha256 <reviewed-hash> --confirm-commit restore-orphan-blob:local:q... --json # exact-hash orphan temp recovery; preserves candidate and never replaces target
resource-kopia repo restore-from-temp --name local --repository-path /mnt/backup/repositories/local.kopia --relative-temp-path q/ab/<id>.f.tmp --temp-sha256 <sha256> --confirm-restore 'restore-temp:local:/mnt/backup/repositories/local.kopia:q/ab/<id>.f.tmp:<sha256>' --json # typed exact-path publication; DBM enforces authority
resource-kopia repo quarantine-orphan-temp-blob --name local --relative-temp-path s/f43/<blob>.f.tmp[.<hex-suffix>] --confirm-quarantine quarantine-orphan-temp:local:s/f43/<blob>.f.tmp[.<hex-suffix>] --json # reversible same-volume move for one unpaired unreadable temp; does not delete its bytes
resource-kopia repo list                                   # registered repos on this host
resource-kopia repo validate   --name offsite              # provider connectivity/integrity
resource-kopia repo disconnect --name offsite              # detach (registry entry kept)
```

`repo damage-scan` is the Kopia owner path for filesystem repository
classification. It lists the registered repository tree without modifying it
and emits typed classifications for unreadable blobs and leftover upload temps.
DBM consumes this result during backup-failure diagnosis; an unreadable
committed blob remains blocking even when a temp-blob exception exists. The
scan is read-only and does not itself establish that a damaged object is safe
to repair. The explicit `--path` form is useful on a second host with a
read-only mount; it requires a clean absolute path containing Kopia's root
marker and reads no repository credential. Provide exactly one of `--name` or
`--path`.

`repo cache-probe --blob-id <exact-q-blob-id>` also emits a `blob` object in
JSON with the expected repository-relative path, exact ID, state (`missing`,
`present`, `unreadable`, or `unknown`), adjacent temp paths, and an `orphan_temp`
classification. It recognizes both suffixless `.f.tmp` and hex-suffixed
`.f.tmp.<hex>` names. This is read-only location evidence; candidate validity
still requires the separate disposable-overlay check reported as
`orphan_temp_blob_candidate`.

`orphan-temp-blob-commit` is the legacy q-blob publishing command. DBM's
current standing-authorized route uses `restore-from-temp` only after a typed
missing-blob probe, exact adjacent candidate binding, absent target, source
read-only evidence and passing disposable-overlay snapshot enumeration. The
operation preserves the source and does not overwrite an existing target. Do
not use the legacy command to bypass DBM's decision and receipt checks.

`restore-from-temp` is the typed path-based operation for one exact Kopia
`.f.tmp` entry. Supply its canonical repository path, repository-relative
temp path and independently reviewed SHA-256; the confirmation phrase binds
repository name, canonical path, relative path and hash. It
resolves the registered repository, rejects symlinked shard components, checks
the source bytes, and publishes by same-volume link only if the final `.f`
object is absent. The temp is preserved. An exact retry returns a replay receipt
only when both source and final object still match the approved hash; a
conflicting final object is never overwritten. This does not grant authority:
DBM must enforce the applicable decision or standing authorization and
revalidate repository and volume identity before invoking it.

`quarantine-orphan-temp-blob` is a reversible namespace move for a content
temp that blocks shard enumeration and cannot be read. It accepts only a
repository-relative `s/...f.tmp` path, optionally followed by a dot and a
hexadecimal unique suffix, verifies that the matching
final `.f` object is absent and that no other adjacent temp exists, then moves
the named entry into a stable, operation-keyed sibling quarantine directory on
the same volume. Repeating the exact approved command returns the same
quarantine receipt without moving or overwriting bytes again.
It neither reads nor deletes the temp contents. Afterward rerun Kopia stats and
DBM destination usage; retain the quarantine until the backup set has been
independently verified.

## Repair authority boundary

Resource-kopia supplies the typed repository operations; it does not decide
whether an operator approved them. `--confirm-commit` and `--confirm-quarantine`
bind a command to its repository and exact object/path. Those phrases are
integrity checks, not verified-human consent.

For DBM production recovery, its `backup_failure` decision owner is the authority
gate. An exact orphan q-blob may be restored under the operator's bounded
2026-11-04 standing authorization only when DBM revalidates the repository and
strong volume identity, one exact adjacent candidate and its hash, an absent
final object, a read-only source and a passing disposable-overlay snapshot list.
DBM then commits that hash, verifies snapshot enumeration, queues one result
notice and retries the affected plans. If any predicate fails, there is no
standing authorization for that attempt.

An unreadable session `.f.tmp` may be quarantined only after DBM has a verified
Notification Hub answer and revalidates one exact orphan temp with the same
path, type and EIO diagnosis. The Kopia move preserves its bytes, writes no
final blob and stays on the same filesystem. Filesystem repair is performed by
the control-plane volume owner through DBM’s identity-bound verified decision
callback; resource-kopia never repairs or formats a filesystem. Other repository
repair, replacement, deletion or pruning is outside the q-blob standing class
and requires its own exact verified-human decision. See the [DBM recovery
runbook](../../../scenarios/data-backup-manager/docs/operations/RUNBOOK.md#backups-blocked-on-a-mounted-drive)
for the decision and callback flow.

`repo connect` can also re-register a pre-existing repository by re-supplying
the backend flags (`--backend filesystem --path …`, `--backend s3 --bucket …`,
or SFTP host/user/path options), useful
when re-attaching on a new host. The passphrase must already exist in the
credential authority. With backend flags, the CLI commits the new registry
entry only after Kopia reports a successful connect. If connect fails, the
prior registry path remains in place. That registry guarantee does not prove
Kopia's config file stayed untouched if the runner partially writes before
returning an error. Resolving a legacy passphrase sidecar can also migrate its
value into the credential authority before Runner; a later failed connect does
not roll back that migration. A successful Kopia connect followed by a registry write
failure is reported explicitly; it may require owner reconciliation. This
ordering does not serialize concurrent create/connect commands. S3 keys and
SFTP private keys are always resolved from the credential authority; the repo
CLI has no credential-bearing flags. Missing values fail before Kopia runs.
SFTP create/connect also requires `--known-hosts` host-key trust metadata.

## Snapshots (`snapshot`)

```bash
resource-kopia snapshot create  --repo offsite --path /data/pg --json   # --json includes the id
resource-kopia snapshot list    --repo offsite --path /data/pg --json
resource-kopia snapshot restore --repo offsite --snapshot <id> --target /restore/pg
resource-kopia snapshot verify  --repo offsite --snapshot <id> --verify-files-percent 100
resource-kopia snapshot delete  --repo offsite --snapshot <id>          # passes kopia --delete
resource-kopia snapshot expire  --repo offsite --path me@host:/data/pg   # dry run: what policy would drop
resource-kopia snapshot expire  --repo offsite --path me@host:/data/pg --delete
```

Kopia expires a source only while creating its next snapshot; `snapshot expire`
applies a tightened policy now. `--path` (and `policy set --path`) take the
exact source a snapshot was recorded under (`user@host:/path`, as shown by
`snapshot list --json`); a snapshot taken with `--override-source` is not
governed by a policy on its captured path.

## Policies (`policy`) — GFS retention + compression per source path

```bash
resource-kopia policy set --repo offsite --path /data/pg \
  --keep-latest 3 --keep-hourly 24 --keep-daily 7 \
  --keep-weekly 4 --keep-monthly 12 --keep-annual 2 \
  --compression zstd --snapshot-interval 1h

resource-kopia policy show --repo offsite --path /data/pg --json
resource-kopia policy show --repo offsite --global --json
resource-kopia policy list --repo offsite --json
```

Any `--keep-*` flag you omit is left untouched on the repository's policy.

## Maintenance (`maintenance`)

```bash
resource-kopia maintenance run --repo offsite           # quick maintenance
resource-kopia maintenance run --repo offsite --full    # full (prunes per retention)
resource-kopia maintenance set --repo offsite --enable-full true --full-interval 24h
resource-kopia maintenance info --repo offsite          # owner, schedule, recent runs
```

Kopia runs due maintenance automatically after snapshot writes, but only when
the writing `user@host` is the repository's maintenance owner (the identity
that created it). Expired snapshots free blob space only after two full cycles
because kopia holds a safety delay before deleting unreferenced blobs.

## Where state lives

| Class | Path | Holds |
|---|---|---|
| config | `<config-root>/vrooli/resources/kopia/repos/<name>/repository.config` | kopia per-repo config file |
| state | `<state-root>/vrooli/resources/kopia/registry.json` | name → repository metadata (no secrets) |
| cache | `<cache-root>/vrooli/resources/kopia/repos/<name>` | kopia content cache (rebuildable) |

Roots default to XDG bases and can be overridden with `KOPIA_CONFIG_DIR`,
`KOPIA_STATE_DIR`, `KOPIA_CACHE_DIR`. No runtime state is ever written into the
repo source tree.

## Secrets, never in argv

Passphrases and S3 keys travel to kopia only through the `KOPIA_PASSWORD`,
`AWS_ACCESS_KEY_ID`, and `AWS_SECRET_ACCESS_KEY` environment variables. SFTP
private keys are written to a temporary mode-0600 file and removed after each
Kopia invocation; `--keyfile` receives only that file path. SFTP repositories
embed credentials in their saved config so later operations can reconnect; the
resource forces that file to mode `0600` on POSIX and keeps the credential
authority as the canonical copy. If the authority is unavailable or a required
credential is missing, the command fails closed.
