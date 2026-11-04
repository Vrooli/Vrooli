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
| `repo` | `create` `connect` `status` `stats` `list` `disconnect` `validate` `repair-format-plan` `cache-probe` `index-recovery-scan` `index-recovery-commit` `orphan-temp-blob-commit` `quarantine-orphan-temp-blob` |
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

- repository passphrase → per-repository logical identity `vrooli/kopia/<name>`, field
  `repository-passphrase`
  (auto-generated 32+ chars on `repo create`; the resource fails closed if it
  is missing — it never uses a default/empty passphrase)
- S3 credentials → the same per-repository identity `vrooli/kopia/<name>`, fields `s3-access-key-id`
  and `s3-secret-access-key`
  (only for `--backend s3`)

See [`resource.json`](resource.json) and the platform credential-authority
documentation in `docs/configuration/secrets.md`.

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

`repo repair-format-plan --name <registered-filesystem-repo> --json` asks
Kopia for a dry-run of filesystem repository format-blob recovery. This is not
a general repository or snapshot-manifest repair. The command only accepts a
registered filesystem repository and cannot apply changes; any repair write
requires a separate reviewed owner action. It fails closed if Kopia returns no
reviewable plan.

`repo index-recovery-scan --name <registered-filesystem-repo> --json` scans
filesystem repository pack blobs for recoverable index entries with Kopia's
`index recover --no-commit`. It never commits recovered indexes or deletes
existing indexes. Its output is diagnostic evidence for a separate reviewed
recovery action, not proof that snapshots can be restored. The command captures
Kopia's successful stderr summary as well as stdout, because Kopia reports the
recovery counts through its logger; an empty captured report is treated as a
failed scan rather than a success-shaped result.

`repo cache-probe --name <registered-filesystem-repo> --json` reconnects the
repository through an ephemeral Kopia config and empty cache, then retries
snapshot enumeration. It does not alter repository bytes or the registered
cache; temporary local probe state is removed afterward. A pass means the
registered cache may be implicated. The same failure points toward repository
storage or a broader host/driver problem. This is diagnostic evidence, not a
repair or restore proof.

Pass `--blob-id <exact-q-blob-id>` to check a specific missing object in the
disposable overlay even when another concurrent manifest read reports a
different first error.

When a missing Kopia `q` blob has exactly one adjacent stale `.tmp` file, the
probe also creates a disposable filesystem overlay, copies that candidate
under the expected blob name in the overlay, and retries snapshot enumeration.
The original candidate and repository remain untouched. This qualifies a
possible recovery candidate only; the probe never publishes it.

`repo index-recovery-commit --name <registered-filesystem-repo>
--confirm-commit commit-index-recovery:<registered-filesystem-repo>` is the
separate write action. It is name-bound, requires an explicit confirmation
phrase, and runs Kopia's `index recover --commit` without enabling
`--delete-indexes`. Use it only after an exact repository recovery decision;
then verify snapshot enumeration and destination usage. Index recovery does
not prove filesystem health or snapshot restorability.

`repo orphan-temp-blob-commit --name <repo> --blob-id <exact-q-blob-id>
--candidate-sha256 <reviewed-sha256>
--confirm-commit restore-orphan-blob:<repo>:<exact-q-blob-id> --json` publishes
one exact adjacent Kopia temp candidate to an absent blob path. It is restricted
to registered filesystem repositories, requires exactly one adjacent candidate
with the reviewed hash, stages and rechecks the bytes, atomically links without
replacing an existing target, and preserves the candidate. This mutates the
repository and must follow an exact operator decision; it is not a general
repository repair and does not prove filesystem health or restore integrity.

`repo quarantine-orphan-temp-blob --name <repo>
--relative-temp-path <s/...f.tmp.suffix>
--confirm-quarantine quarantine-orphan-temp:<repo>:<relative-temp-path> --json`
moves one exact unpaired `s`-blob temp out of the repository shard namespace
to a unique sibling quarantine directory on the same volume. It checks that the
only adjacent temp is the exact named source and that the corresponding final
blob is absent. It does not read, rewrite, or delete the temp's bytes; the move
is reversible. Use this only for an exact orphan temp that blocks shard
enumeration; it is not a substitute for filesystem repair or restore proof.

## Architecture

`external-cli` driver. The kopia binary is provisioned (version-pinned,
checksum-verified on Linux) via the manifest's `install.platforms` block; the
wrapping CLI is Go-native under `cli/internal/...` (no shell `lib/*.sh`):

```
cli/internal/
  app/         CLI wiring (lifecycle + repo/snapshot/policy/maintenance groups)
  kexec/       the SINGLE seam that execs the kopia binary (+ test fake)
  credentials/ credential-authority sourcing (+ test fake)
  repoctx/     resolve a repo name -> config + secret env
  registry/    name -> repository metadata (state root; no repo-local data/)
  repo/ snapshot/ policy/ maintenance/   command handlers (flag -> kopia argv)
  discovery/ install/ version/           binary location + version pin
  cmdutil/ invariant/                    shared helpers + safety scanners
```

Runtime state resolves to host-scoped storage-class paths
(`<state|config|cache>/vrooli/resources/kopia/...`), never repo-local `data/`.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
