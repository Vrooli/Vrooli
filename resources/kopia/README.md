# Kopia Resource

A Vrooli **resource** that wraps the [kopia](https://kopia.io) content-addressed
backup engine behind the canonical `resource-kopia` CLI. It is the encrypted
backup *substrate* for the future `data-backup-manager` scenario — the engine
plus a thin, well-documented wrapping seam, nothing more.

## Why wrap kopia

Per Vrooli's [wrap-not-use](../../docs/README.md) principle, agents and
scenarios must never call the `kopia` binary directly — only `resource-kopia`.
Kopia gives us, for free, the capabilities we would otherwise hand-roll:

- content-addressed **deduplication** + compression
- **client-side encryption, on by default** (AES256-GCM-HMAC-SHA256) — there is
  no plaintext-repo code path
- incremental snapshots + GFS retention policies
- multi-destination repositories (filesystem and S3/MinIO today)
- integrity verification and repository statistics

## Capabilities (all via `resource-kopia`)

| Group | Commands |
|---|---|
| Lifecycle | `info` `status` `install` `uninstall` `start` `stop` `restart` `logs` |
| Engine | `version` (wrapper + pinned/installed kopia version, drift warning) |
| `repo` | `create` `connect` `status` `stats` `list` `disconnect` `validate` |
| `snapshot` | `create` `list` `restore` `verify` `delete` |
| `policy` | `set` `show` `list` |
| `maintenance` | `run` `set` |

A Vrooli **destination** maps one-to-one to a kopia **repository**, identified
by `--name`. Backends in this version: `filesystem` and `s3` (MinIO-compatible
via `--endpoint`).

## Secrets

Credentials are sourced through the control-plane credential authority and
injected only into the owning runtime process, never from config files or
committed environment files:

- repository passphrase → logical identity `vrooli/kopia`, field
  `repository-passphrase`
  (auto-generated 32+ chars on `repo create`; the resource fails closed if it
  is missing — it never uses a default/empty passphrase)
- S3 credentials → logical identity `vrooli/kopia`, fields `s3-access-key-id`
  and `s3-secret-access-key`
  (only for `--backend s3`)

See [`resource.json`](resource.json) and
[`resources/vault/docs/SECRETS-STANDARD.md`](../vault/docs/SECRETS-STANDARD.md).

## Disaster recovery

The wrapper adds **no proprietary encoding** — repositories are vanilla kopia
repositories. With only the plain `kopia` binary + the passphrase (and S3 creds
for S3 repos), an operator can restore with Vrooli completely down. See
[`docs/RECOVERY.md`](docs/RECOVERY.md); the property is proven by an automated
integration test.

## Quick start

```bash
vrooli setup --resources kopia          # install the pinned kopia binary
resource-kopia status                   # binary present, version matches pin
resource-kopia version                  # wrapper + pinned/installed kopia version

resource-kopia repo create --name nightly --backend filesystem --path /var/backups/nightly
resource-kopia policy set --repo nightly --path /data/pg --keep-daily 7 --keep-weekly 4 --compression zstd
resource-kopia snapshot create --repo nightly --path /data/pg
resource-kopia snapshot list --repo nightly --path /data/pg --json
```

`snapshot create` accepts optional self-identifying metadata that is passed
straight through to kopia (and never carries a secret):

```bash
resource-kopia snapshot create --repo nightly --path /data/pg \
  --description "nightly pg backup" \
  --override-source dbm://acme/db \
  --tags app:dbm --tags run:r1
```

These let a standalone `kopia snapshot list --json` attribute a snapshot to its
owner without the calling application running. Data Backup Manager uses them to
stamp `dbm.*` tags on every backup.

### Destination bundles (Data Backup Manager)

When Data Backup Manager creates a filesystem destination, it does **not** point
kopia at the operator-facing folder directly. Instead it builds a self-describing
*bundle*: the folder holds `README.txt`, `RECOVERY.txt`, and
`vrooli-backup-destination.json`, and the actual vanilla kopia repository lives
under `repositories/<slug>.kopia`. The concrete repository path is recorded in
the manifest's `repository_path` field. The repository itself is an ordinary
kopia repository — connect to that nested path (not the bundle root) for
standalone recovery.

Full command reference: [`docs/OPERATIONS.md`](docs/OPERATIONS.md).

## Architecture

`external-cli` driver. The kopia binary is provisioned (version-pinned,
checksum-verified on Linux) via the manifest's `install.platforms` block; the
wrapping CLI is Go-native under `cli/internal/...` (no shell `lib/*.sh`):

```
cli/internal/
  app/         CLI wiring (lifecycle + repo/snapshot/policy/maintenance groups)
  kexec/       the SINGLE seam that execs the kopia binary (+ test fake)
  vault/       secret sourcing via resource-vault (+ test fake)
  repoctx/     resolve a repo name -> config + secret env
  registry/    name -> repository metadata (state root; no repo-local data/)
  repo/ snapshot/ policy/ maintenance/   command handlers (flag -> kopia argv)
  discovery/ install/ version/           binary location + version pin
  cmdutil/ invariant/                    shared helpers + safety scanners
```

Runtime state resolves to host-scoped storage-class paths
(`<state|config|cache>/vrooli/resources/kopia/...`), never repo-local `data/`.
