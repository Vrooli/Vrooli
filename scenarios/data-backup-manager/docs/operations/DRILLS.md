# Recovery drills

Recovery drills are scheduled or operator-triggered proofs that a successful
backup can be restored. They select the newest successful target×destination
snapshot and call the existing verified-restore path, which restores only to a
temporary scratch directory, checks the repository, computes evidence, and
cleans the scratch directory.

Drills are durable records. A record remains `requested`, `running`,
`verified`, or `failed` across API restarts and links to the restore record
that produced the evidence. Retry requests should carry a stable
`idempotency_key`; a repeated key returns the original record. An in-flight
drill is never started twice.

New plans default to the weekly `recovery_drill_schedule` value `168h`. Startup
bootstraps existing enabled plans with an empty cadence to that same weekly
default; explicit non-empty cadences are preserved. Use the explicit
`recovery_drill_schedule: manual` value to keep an enabled plan manual-only.
The scheduler independently evaluates
every target × destination unit in an enabled plan, so a failed primary does not
suppress a secondary drill. It checks due work immediately at startup and then
once per minute. The Drills page lists every configured target × destination
unit alongside its cadence and latest receipt, including units with no receipt.
Failed-drill notifications collect for up to 15 seconds. Multiple failures in
that window produce one summary message; each drill remains available as an
individual record on the Drills page.

Before starting a drill, use:

```text
vrooli scenario cli data-backup-manager drills preview --plan <plan-id>
vrooli scenario cli data-backup-manager drills run --plan <plan-id> --idempotency-key <stable-key>
vrooli scenario cli data-backup-manager drills get <drill-id>
vrooli scenario cli data-backup-manager drills list --plan <plan-id>
```

If preview reports no successful snapshot, run the selected backup plan first.
A failed drill is recovery evidence, not a healthy backup result; inspect its
linked restore and `next_action` before retrying.

Drills never write to live target locations and never serialize passphrases,
credential values, or restored file contents.

## Treasury portable key-only bundle transport

Treasury writes each recovery set as a private directory at
`<SCENARIO_DATA_DIR>/portable-recovery/<wallet-id>.recovery/`. It contains
`wallet.bundle` (encrypted key-only ciphertext) and `witness.json` (public
wallet/book metadata authenticated with the operator-held recovery secret).
Treasury stages `revocation.json`, signed by the retiring wallet key, and
removes the live `wallet.bundle`. Its rotation path must wait until the owner
publisher has restore-verified that exact tree as the newest validated
snapshot; only then may Treasury mark the old wallet revoked or delete its
scalar. If no healthy writable destination or publisher is configured,
rotation stays pending and the old key remains active. DBM snapshots the whole
directory and treats all of its contents as opaque bytes.

On the source node, export the pair, register that directory under its stable
owner/name, and bind it to a dedicated manual plan and independently reachable
encrypted destination:

```bash
treasury wallet export-portable-recovery --id <wallet-id> --network <network> --secret-stdin
data-backup-manager targets register --owner treasury --name treasury-wallet-recovery-<wallet-id> --kind filesystem --locator '<SCENARIO_DATA_DIR>/portable-recovery/<wallet-id>.recovery'
data-backup-manager plans create --name treasury-wallet-recovery-<wallet-id> --targets <target-id> --destinations <destination-id> --schedule '' --keep-latest 10 --allow-incomplete-coverage
data-backup-manager runs trigger --plan <plan-id>
data-backup-manager runs get <run-id>
data-backup-manager restores verify --target <target-id> --destination <destination-id> --snapshot <snapshot-id>
```

The owner publisher must query the repository's stable `dbm://treasury/treasury-wallet-recovery-<wallet-id>`
history and select only its newest snapshot. It restore-verifies that the
snapshot contains the signed `revocation.json` and no live `wallet.bundle`
before returning durable publication evidence to Treasury. A restore consumer
refuses a tombstone and never falls back to an older active bundle. Keep enough
snapshots to preserve that newest lineage; the normal retention and
repository-availability checks still apply.

Use a destination whose repository is independently reachable from both
machines, such as an already configured S3-compatible or SFTP repository. Keep
repository credentials in Credential Authority on each node. A filesystem
destination is suitable only when its mounted volume is physically separate
from the source host and is also available to the recovery node. A second path
on the same host is not off-host protection.

On the recovery node, import/attach the existing repository with its exact
non-secret catalog metadata, then use Scenario-to-Cloud to select and restore
the newest stable-target snapshot into a new private staging directory:

```bash
data-backup-manager destinations import-existing --id <destination-id> --name <destination-name> --backend s3 --location '<existing-s3-location>' --created-at <destination-created-at-rfc3339>
scenario-to-cloud wallet-recovery restore --wallet-id <wallet-id> --network <network> --destination-id <destination-id> --destination-name <destination-name> --backend s3 --location '<existing-s3-location>' --created-at <destination-created-at-rfc3339> --staging-dir '<new-private-empty-directory>' --secret-stdin
```

Use `--backend sftp` and that repository's existing non-secret location when
SFTP is the configured independent destination. Its credentials must already
be available to DBM through Credential Authority. Never put credentials or the
operator's recovery secret in command arguments. Restore to a private empty
staging directory, never over Treasury's live data directory. The alternative
is to restore the snapshot into a clean replacement Treasury
`<SCENARIO_DATA_DIR>` before starting Treasury; its restore command then reads
the private artifact automatically.

The S3/SFTP attach, destination readiness, source run, post-rotation tombstone
snapshot, newest-snapshot verification, clean-node restore, restart and signed
control proof are required evidence for an off-host recovery claim. A
successful filesystem source unit test or scratch-only recovery drill proves
byte handling only; neither proves independent placement or second-node
retrieval. Do not classify an existing repository as usable while DBM reports
it unavailable, and do not substitute an emulator or fixture for the fleet
node in the recovery result.

For credential recovery, this DBM receipt is only the restore half of the
proof. Secrets Manager must separately inspect and decrypt its authority bundle
on an isolated replacement authority, verify synthetic item usability, and
activate a new recovery epoch. Activation closes old use sessions and places
restored grants into review until current membership and provider state have
been reconciled. A checksum-only drill must remain incomplete for that use
case. Keep the DBM restore receipt and the authority usability receipt as
separate records.

The plan surface also shows the selected protection tier, drill cadence, and
destination topology warnings. Treat a plan with
`destinations_physically_independent=false` as requiring operator review before
calling its secondary protection independent.
