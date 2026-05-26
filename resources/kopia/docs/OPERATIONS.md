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

Encryption is always on; the passphrase is generated and stored in vault on
first `create`. S3 credentials, when supplied, are also stored in vault.

```bash
# Filesystem backend
resource-kopia repo create --name nightly --backend filesystem --path /var/backups/nightly

# S3 / MinIO backend (creds stored in vault; --disable-tls for local MinIO)
resource-kopia repo create --name offsite --backend s3 \
  --bucket vrooli-backups --endpoint minio:9000 --disable-tls \
  --access-key "$MINIO_ACCESS_KEY" --secret-access-key "$MINIO_SECRET_KEY"

resource-kopia repo connect    --name offsite              # reconnect using registered params
resource-kopia repo status     --name offsite --json       # encryption algo, format version
resource-kopia repo stats      --name offsite --json       # size, content/object counts, dedup
resource-kopia repo list                                   # registered repos on this host
resource-kopia repo validate   --name offsite              # provider connectivity/integrity
resource-kopia repo disconnect --name offsite              # detach (registry entry kept)
```

`repo connect` can also re-register a pre-existing repository by re-supplying
the backend flags (`--backend`/`--path` or `--backend s3 --bucket …`), useful
when re-attaching on a new host. The passphrase must already exist in vault.

## Snapshots (`snapshot`)

```bash
resource-kopia snapshot create  --repo offsite --path /data/pg --json   # --json includes the id
resource-kopia snapshot list    --repo offsite --path /data/pg --json
resource-kopia snapshot restore --repo offsite --snapshot <id> --target /restore/pg
resource-kopia snapshot verify  --repo offsite --snapshot <id> --verify-files-percent 100
resource-kopia snapshot delete  --repo offsite --snapshot <id>          # passes kopia --delete
```

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
```

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
at call time from vault. They never appear as CLI flags (enforced by a unit
test). If vault is unavailable or a passphrase is missing, the command fails
closed — it never falls back to a default or empty passphrase.
