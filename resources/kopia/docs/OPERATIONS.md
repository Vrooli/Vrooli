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
credential authority on first `create`. S3 credentials, when supplied, are
stored under the same per-repository authority identity.

```bash
# Filesystem backend
resource-kopia repo create --name nightly --backend filesystem --path /var/backups/nightly

# S3 / MinIO backend (creds stored in the credential authority; --disable-tls for local MinIO)
resource-kopia repo create --name offsite --backend s3 \
  --bucket vrooli-backups --endpoint minio:9000 --disable-tls \
  --access-key "$MINIO_ACCESS_KEY" --secret-access-key "$MINIO_SECRET_KEY"

resource-kopia repo connect    --name offsite              # reconnect using registered params
resource-kopia repo status     --name offsite --json       # encryption algo, format version
resource-kopia repo stats      --name offsite --json       # size, content/object counts, dedup
resource-kopia repo repair-format-plan --name local --json # non-mutating format-blob recovery plan only
resource-kopia repo cache-probe --name local --blob-id q... --json # isolated cache; test this exact q-blob temp in a disposable overlay; never publishes it
resource-kopia repo index-recovery-scan --name local --json # scan recoverable indexes; captures the Kopia logger summary; does not commit
resource-kopia repo index-recovery-commit --name local --confirm-commit commit-index-recovery:local --json # explicit name-bound index commit; never deletes indexes
resource-kopia repo orphan-temp-blob-commit --name local --blob-id q... --candidate-sha256 <reviewed-hash> --confirm-commit restore-orphan-blob:local:q... --json # exact-hash orphan temp recovery; preserves candidate and never replaces target
resource-kopia repo quarantine-orphan-temp-blob --name local --relative-temp-path s/f43/<blob>.f.tmp.<suffix> --confirm-quarantine quarantine-orphan-temp:local:s/f43/<blob>.f.tmp.<suffix> --json # reversible same-volume move for one unpaired unreadable temp; does not delete its bytes
resource-kopia repo list                                   # registered repos on this host
resource-kopia repo validate   --name offsite              # provider connectivity/integrity
resource-kopia repo disconnect --name offsite              # detach (registry entry kept)
```

`orphan-temp-blob-commit` mutates one registered filesystem repository. Use it
only after a Notification Hub decision names the exact blob, candidate hash,
and repository path. The command requires exactly one adjacent `.tmp.*`
candidate, verifies its hash before and after same-volume staging, atomically
links it only when the target remains absent, and preserves the candidate file.
It does not repair other repository errors or establish mounted filesystem
health; rerun snapshot enumeration and DBM destination readiness afterward.

`quarantine-orphan-temp-blob` is a reversible namespace move for a content
temp that blocks shard enumeration and cannot be read. It accepts only a
repository-relative `s/...f.tmp.<suffix>` path, verifies that the matching
final `.f` object is absent and that no other adjacent temp exists, then moves
the named entry into a unique sibling quarantine directory on the same volume.
It neither reads nor deletes the temp contents. Afterward rerun Kopia stats and
DBM destination usage; retain the quarantine until the backup set has been
independently verified.

`repo connect` can also re-register a pre-existing repository by re-supplying
the backend flags (`--backend`/`--path` or `--backend s3 --bucket …`), useful
when re-attaching on a new host. The passphrase must already exist in the
credential authority.

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
`AWS_ACCESS_KEY_ID`, and `AWS_SECRET_ACCESS_KEY` environment variables, sourced
at call time from the credential authority. They never appear as CLI flags
(enforced by a unit test). If the authority is unavailable or a passphrase is
missing, the command fails closed — it never falls back to a default or empty
passphrase.
