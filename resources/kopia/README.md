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

All credentials are sourced from the **vault** resource via `resource-vault`,
never from config files or committed env:

- repository passphrase → `secret/resources/kopia/repo/<name>/passphrase`
  (auto-generated 32+ chars on `repo create`; the resource fails closed if it
  is missing — it never uses a default/empty passphrase)
- S3 credentials → `secret/resources/kopia/s3/<name>/{access_key,secret_key}`
  (only for `--backend s3`)

See [`config/secrets.yaml`](config/secrets.yaml) and
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
