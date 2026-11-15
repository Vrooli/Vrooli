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
