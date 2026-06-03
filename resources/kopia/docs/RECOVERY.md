# Kopia Resource — Disaster Recovery (standalone restore)

**The guarantee:** repositories created by `resource-kopia` are *vanilla kopia
repositories*. The wrapper adds no proprietary metadata or encoding. With only

1. the plain `kopia` binary,
2. the repository **passphrase**, and
3. (for S3 repos) the S3 **access/secret keys**,

you can connect and restore **with Vrooli completely down** — no `resource-kopia`,
no vault, no Vrooli control plane required.

This property is load-bearing and is proven by an automated test
(`TestIntegrationStandaloneDisasterRecovery`, plan §9.2 Test C), which performs
the exact steps below using the plain `kopia` binary and asserts a byte-for-byte
checksum match against the original tree.

## Step 0 — Recover the secrets

Under normal operation the passphrase and S3 keys live in vault:

- passphrase: `secret/resources/kopia/repo/<name>/passphrase`
- S3 access key: `secret/resources/kopia/s3/<name>/access_key`
- S3 secret key: `secret/resources/kopia/s3/<name>/secret_key`

For true disaster recovery, retrieve these from wherever your vault data is
backed up. **Keep an out-of-band copy of each repository passphrase** — without
it the encrypted repository cannot be opened by anyone, including you.

## Step 1 — Install plain kopia

Any kopia build at or above the pinned version works (the pin is in
`cli/internal/version/version.go`). For example, download the matching release
from <https://github.com/kopia/kopia/releases> and put `kopia` on your `PATH`.

## Step 2 — Connect to the repository

Use a throwaway config file so you do not disturb any existing state. The
passphrase is supplied via the `KOPIA_PASSWORD` environment variable so it never
lands in shell history or process listings.

### Filesystem repository

```bash
export KOPIA_PASSWORD='<the repository passphrase>'
kopia --config-file /tmp/dr.config repository connect filesystem \
  --path /var/backups/nightly
```

> **Data Backup Manager destination bundles.** If the folder you plugged in is a
> DBM destination *bundle* (it contains `README.txt`, `RECOVERY.txt`, and
> `vrooli-backup-destination.json`), the kopia repository is **not** at the
> folder root — it is nested under `repositories/<slug>.kopia`. Read the exact
> path from `vrooli-backup-destination.json`'s `repository_path` field and pass
> that to `--path`. The bundle's own `RECOVERY.txt` repeats these steps with the
> concrete path filled in.

### S3 / MinIO repository

```bash
export KOPIA_PASSWORD='<the repository passphrase>'
export AWS_ACCESS_KEY_ID='<s3 access key>'
export AWS_SECRET_ACCESS_KEY='<s3 secret key>'
kopia --config-file /tmp/dr.config repository connect s3 \
  --bucket vrooli-backups --endpoint minio:9000 --disable-tls   # drop --disable-tls for real TLS
```

## Step 3 — Find the snapshot and restore

```bash
kopia --config-file /tmp/dr.config snapshot list --json     # pick a snapshot "id"
kopia --config-file /tmp/dr.config snapshot restore <id> /restore/target
```

## Step 4 — Verify integrity

```bash
kopia --config-file /tmp/dr.config snapshot verify --verify-files-percent 100
```

## Notes

- Encryption is mandatory and on by default; the algorithm is recorded in the
  repository format and reported by `kopia repository status --json`.
- The `--config-file` flag is the only addressing the wrapper relies on — it is
  a stock kopia global flag, not a Vrooli invention.
- Production consumers must still go through `resource-kopia`. Calling `kopia`
  directly is sanctioned **only** for this recovery path (and the test that
  proves it works).
