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

Configure an automated interval on a plan with `recovery_drill_schedule`, for
example `168h`. An empty value disables scheduling while leaving the manual
commands available. The scheduler independently evaluates every target ×
destination unit in an enabled plan, so a failed primary does not suppress a
secondary drill.

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

The plan surface also shows the selected protection tier, drill cadence, and
destination topology warnings. Treat a plan with
`destinations_physically_independent=false` as requiring operator review before
calling its secondary protection independent.
