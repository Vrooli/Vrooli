# Runbook — Data Backup Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.


Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process
naming, ports, health checks, and logs.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| Backup overdue / failing | `/health` (flags overdue/failed), last-success per target, run history | Inspect the failing run's logs; re-run the plan on demand; verify the destination is reachable | If a target has no recent success, treat as a recovery risk — escalate. |
| Destination unreachable / not writable | Destination dry-run, credential-authority backend credentials, network/path | Restore authority access or provision the required field; re-validate destination; pause dependent plans | Escalate before the next scheduled run if it cannot be fixed in time. |
| Storage cap reached (alert+block) | Per-destination usage vs cap | Add capacity, tighten retention policy, or add a destination — **never** delete backups manually to free space | Escalate; blocked writes mean new backups are not landing. |
| Verified-restore failing | Verify-mode run output, checksums, scratch restore logs | Treat the affected target as un-recoverable until verify passes; do **not** remove its committed copy from git | Escalate immediately — this is the gate protecting recoverability. |
| UI blank or stale | UI port, `ui/dist` freshness, server logs | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `data-backup-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

### Linux NTFS destination

The control plane mounts Linux NTFS through UDisks' recognized `ntfs` helper
type, which resolves to `ntfs-3g` on the supported host. Verify the live mount
with `vrooli host volume inspect`; `fuseblk` is the mount driver, while DBM
readiness resolves the on-disk format as NTFS. A mount through `ntfs3` fails
readiness because the kernel driver's write fault can panic the host. Do not
back up through that driver.

`ntfs-3g` avoids that kernel fault, but this host does not expose NTFS
clean/dirty state while mounted through FUSE. If DBM reports filesystem state
or overall readiness as UNKNOWN, keep backup writes blocked. A successful mount
or filesystem repair is not a clean-state check, repository verification, or
backup/restore proof. Never format or clear a drive as a routine remount fix.

## Disaster Recovery — Backup / Restore / Verify

This is the scenario's reason to exist. The operating model has three
loops: **back up** registered targets on schedule or on demand,
**verify** that backups can restore, and **restore** when needed.

### Catalog orientation

The manager's domain is TARGET (a registered source) × DESTINATION (an
encrypted kopia repository) × PLAN (a many-to-many binding with schedule
and retention) → RUN → RESTORE. Scenarios self-register their targets;
the manager's SQLite catalog is a cache and run history, rebuilt from
re-registration on boot — it is not the source of truth.

```bash
data-backup-manager targets list
data-backup-manager destinations list
data-backup-manager plans list
data-backup-manager runs list
data-backup-manager restores list
```

### Coverage — register default protection

Discovery only *suggests* durable state worth protecting; a plan can only back
up *registered* targets. On a fresh install that gap lets an operator build a
plan that looks protective while it covers a single self-registered target.
The coverage surface closes it: it reports what is registered vs
recommended-but-unregistered vs sensitive, and bulk-registers the recommended
non-sensitive defaults in one action.

```bash
data-backup-manager coverage report
data-backup-manager coverage accept-defaults --dry-run
data-backup-manager coverage accept-defaults
data-backup-manager coverage review-sensitive
```

Sensitive suggestions (credential/token files such as `codex/auth`,
`claude-code/credentials`) are **never** registered through bulk acceptance.
Use `coverage review-sensitive`; the signed callback revalidates the candidate
set before applying the verified operator decision and publishes a result notice.
The decision offers a standing rule that covers future sensitive candidates, so
the question stops recurring; `coverage sensitive-policy` shows who consented and
when, and `coverage revoke-sensitive-policy` turns it off.

Credential-bearing targets are review-only. The plaintext `~/.vrooli/secrets.json`
legacy file is never suggested; the encrypted `~/.vrooli/secrets.enc.json` store
and credential-authority recovery bundle are the supported recovery surfaces.
After exporting credentials, `vrooli credentials doctor` reports the receipt and
whether every configured identity is covered. A detached Kopia bundle names the
credential identity and field and can be imported on a replacement host with:

```sh
printf '%s' "$PASSPHRASE" | vrooli credentials provision \
  --identity vrooli/kopia/<repository> --field repository-passphrase
```

Coverage reads no file contents and registers locators only. Because plan
creation is guarded (see below), running `coverage accept-defaults` before
`plans create` is the normal first-backup order.

### Why a backup run used to be interrupted or blocked (fixed 2026-11-15)

- **Health probe timeouts restarted the service mid-run.** `/health` computed the
  backup posture (every target's last run, every enabled plan) on each probe. While
  a large backup was writing that took 5 seconds or more, the probe timed out, and
  the service was restarted in the middle of the run, three times in 25 minutes.
  The probe is now answered from the last posture result (15 second freshness, a
  background refresh, never older than 10 minutes), so it returns at once.
- **One unreadable source stopped every target of a plan.** A source that could
  not be enumerated (for example a root-owned subdirectory) left the destination's
  capacity "unknown", which preflight reported as "destination filesystem readiness
  is unknown" and applied to every target at that destination. The unreadable
  target is now blocked on its own (`source_missing`) and capacity is judged over
  the targets that will be written.

### Default plans: "registered" means backed up

A plan can name its targets by rule instead of by list. `--target-selector
registered` means every registered target, minus `--exclude-targets`, evaluated
every time the plan is read (runs, the scheduler, coverage, health). A target an
owner declares is registered by the standing rule's sweep (secrets included, under
the operator's answer to Hub ask c0d25f0a) and is in the plan on its next run with
nobody editing anything. Explicit target lists keep working.
`plans update --targets` turns a selector plan back into a list; `add-destination`
and `update` keep the rule.

Two default plans use it (created 2026-11-15):

| Plan | Targets | Destination | State |
|---|---|---|---|
| `declared-targets` | every registered target | `elements-local` (external drive) | disabled until the Elements drive passes readiness (DB5); enable it with `plans update --id <id> --enabled true` |
| `declared-targets-local` | every registered target except the left-out list below | `baseline-safety` (this machine's home disk) | enabled, every 24h, keeps the latest 3 snapshots |

Why two: the home disk cannot hold everything. The local plan is sized so its
first copy keeps the home disk under 92% used (budget = headroom to 92% minus a
15 GiB margin, measured 2026-11-15 at 89.5% used; secrets and the AI tool homes
first, then the smallest targets). The Elements plan has no exclusions. It is left
disabled because enabling it now would fail every target on the same blocked drive
and turn the health list from a few issues into hundreds.

**The local copy is on the same disk as the source.** It protects against
deletion and corruption, not against losing the drive. Only a target with a proven
independent second copy (the Elements plan, once it runs) is protected against
drive loss; `coverage report` states the count.

Left out of the local plan (kept by `declared-targets` on Elements once it runs;
a few also have their own baseline-safety plan):

| Target | Size | Where |
|---|---|---|
| vrooli / state | 141.0 GiB | `~/.vrooli/state` |
| web-console / session-rollouts | 133.1 GiB | `~/.vrooli/state/vrooli/web-console/sessions` |
| vrooli / data | 129.7 GiB | `~/.vrooli/data` |
| research-desk / research-home | 69.9 GiB | `~/.vrooli/research` |
| songgen / models | 51.6 GiB | `~/.local/share/vrooli/resources/songgen/models` |
| research-desk / research_archive_rehearsals | 45.1 GiB | `~/.vrooli/archive/research/rehearsal` |
| ollama / models | 42.1 GiB | `~/.local/share/vrooli/resources/ollama/models` |
| image-tools / data | 42.0 GiB | `~/.vrooli/data/vrooli/image-tools` |
| image-tools / models | 37.8 GiB | `~/.vrooli/data/vrooli/image-tools/models` |
| plan-manager / plan_artifacts | 25.8 GiB | `~/.vrooli/plan-artifacts` |
| vrooli / plan_artifacts | 25.8 GiB | `~/.vrooli/plan-artifacts` |
| browser-automation-studio / data | 25.7 GiB | `~/.vrooli/data/vrooli/browser-automation-studio` |
| agent-manager / effort_evidence | 25.6 GiB | `~/.vrooli/evidence` |
| vrooli / evidence | 25.6 GiB | `~/.vrooli/evidence` |
| browser-automation-studio / recordings | 20.3 GiB | `~/.vrooli/data/vrooli/browser-automation-studio/recordings` |
| ace-step / models | 15.1 GiB | `~/.local/share/vrooli/resources/ace-step/checkpoints` |
| stable-audio-3 / models | 9.7 GiB | `~/.local/share/vrooli/resources/stable-audio-3/models` |
| ollaya / models | 8.9 GiB | `~/.local/share/vrooli/resources/ollaya/models` |
| agent-manager / data | 6.5 GiB | `~/.vrooli/data/vrooli/agent-manager` |
| agent-manager / legacy_history | 6.5 GiB | `~/.vrooli/data/vrooli/agent-manager/legacy-history` |
| browser-automation-studio / captures | 5.2 GiB | `~/.vrooli/data/vrooli/browser-automation-studio/captures` |
| system-monitor / data | 5.2 GiB | `~/.vrooli/data/vrooli/system-monitor` |
| postgres / instance_runtime | 5.2 GiB | `~/.local/share/vrooli/resources/postgres/instances/main` |
| postgres / main_data | 5.2 GiB | `~/.local/share/vrooli/resources/postgres/instances/main/data` |
| agent-manager / run_state | 4.9 GiB | `~/.vrooli/state/vrooli/agent-manager/runs` |
| image-tools / adapters | 3.9 GiB | `~/.vrooli/data/vrooli/image-tools/adapters` |
| vrooli / backups | 3.5 GiB | `~/.vrooli/backups` |
| vrooli / agent-worktree | 2.8 GiB | `~/Vrooli/.vrooli/work` |

To bring one in, remove its id from the plan's exclusions with
`plans update --id <local plan id> --exclude-targets <remaining ids>`; check
`df /` first. `coverage report` lists registered targets bound to no enabled plan.

### Destination onboarding

Discovery and readiness analysis are read-only. A removable drive is not
used until an operator reviews it and creates a destination explicitly.
Prefer a dedicated repository subdirectory such as
`<mount>/vrooli-backups` instead of the mount root:

```bash
data-backup-manager discovery destinations
data-backup-manager destinations readiness --location "/mnt/backup-drive"
data-backup-manager destinations prepare-plan --location "/mnt/backup-drive" --action create-subdir --subdir vrooli-backups
```

`prepare-execute` defaults to dry-run. Real execution is opt-in and requires
the exact confirmation phrase plus device-identity revalidation. Volume
recovery is split into `unmount`, `check-filesystem`, optional
`repair-filesystem` when the check reports inconsistency, and
`mount-read-write`. The control plane owns host mutations; DBM plans,
confirms, and records each typed outcome. Formatting, relabeling, and clearing
files remain outside this recovery flow.

When the volume is unmounted, include its UUID and/or serial from a read-only
inspection. On Linux those identifiers can resolve the current device without
the old path; other platforms require their native resolver before this flow is
available. Device paths can change after a replug; stable identifiers make a
stale plan fail closed. The mount step uses the
plan's requested mount location even though the pre-step inspection correctly
reports no current mountpoint.

### SFTP destination

An operator-authorized SSH host can hold a Kopia repository. The destination
location carries only connection metadata; private-key contents stay in the
credential authority:

```text
sftp://<username>@<host>[:port]/<absolute-repository-path>?known_hosts=<url-encoded-local-file>
```

Provision `sftp-private-key` for `vrooli/kopia/<destination-name>` before
creating the destination. Kopia receives it through a temporary owner-only
keyfile, embeds the connection key in its protected repository config for
later operations, and the credential authority remains the canonical source
for rebuild. Host-key verification uses the supplied `known_hosts` file.
An SFTP hostname and key alone do not prove a distinct Vrooli host or physical
failure domain; topology remains unknown until a trusted host owner provides
that evidence.

```bash
vrooli credentials provision --identity "vrooli/kopia/destination-name" --field sftp-private-key
data-backup-manager destinations create --name "remote-backup" --backend sftp --location 'sftp://backup@example.invalid:22/repository?known_hosts=%2Fhome%2Foperator%2F.ssh%2Fknown_hosts'
```

### S3-compatible destination

An operator-selected S3-compatible repository can use one of these non-secret
location forms:

```text
s3://<bucket>/<prefix>?region=<region>
s3+https://<endpoint>/<bucket>/<prefix>?region=<region>
```

Use `s3+http://` only for an explicitly chosen local-compatible endpoint. Do
not put access keys or tokens in the location. Provision the repository's
`s3-access-key-id` and `s3-secret-access-key` fields for its
`vrooli/kopia/<destination-name>` credential-authority identity before creating
the destination; resource-kopia reads those fields itself. An S3 backend name
or URI does not prove provider hosting or physical independence. Until the
endpoint owner supplies hosting and failure-domain evidence, DBM reports
independence as unknown.

Provision each field through the credential owner, which prompts for the
value, then create the destination with the CLI location set to the URI:

```bash
vrooli credentials provision --identity "vrooli/kopia/destination-name" --field s3-access-key-id
vrooli credentials provision --identity "vrooli/kopia/destination-name" --field s3-secret-access-key
data-backup-manager destinations create --name "cloud-backup" --backend s3 --location 's3+https://storage.example.invalid/bucket/prefix?region=region-1'
```

Escrow the credential recovery bundle separately through the control plane's
`credential-escrow` capability, using its operator-reviewed preview/apply flow
and the operator-selected sink. Keep its passphrase away from both the bundle
and the old host.

For an interruption-safe sequence, persist the plans with
`destinations recovery-start`, inspect progress with `recovery-get`, and use
`recovery-resume` to validate or continue from the recorded cursor. Resume
defaults to dry-run; a real repair still requires the exact step confirmation
and `--acknowledge-data-loss`.

The catalog and bundle record the destination path relative to the volume root.
A changed mountpoint or Windows drive letter is resolved only after the current
volume identity is observed. For a filesystem bundle, DBM also records the
Kopia repository `uniqueID` in the bundle manifest. On relocation, it compares
the repository marker's ID with that anchor before reconnecting Kopia or
repairing generated path projections. This detects a different or replaced
repository at the expected path; it does not replace a restore drill as proof
that the expected snapshot history is recoverable. A legacy bundle without an
anchor can be anchored only while its catalog paths still match the bundle's
original absolute paths. A moved legacy bundle without an anchor fails closed.

Before repository creation/deletion, preflight, usage/capacity probes, a
backup snapshot, a snapshot restore/verification/audit, retention, or recovery
metadata repair, DBM resolves the mounted destination again and holds a
per-destination operation lock through the complete Kopia call. Backup-failure
diagnosis, stalled-notice evidence and health posture checks also re-resolve
the destination before repository reads, so a stale Kopia connection cannot
clear or reclassify a failure. A changed mountpoint is repaired before use,
while ambiguous identity, an unhealthy/read-only volume, or a path that no
longer resolves to the same repository prevents the operation. If readiness
changes after a successful snapshot, DBM keeps that snapshot and reports
retention as failed instead of applying policy to an unverified repository.
The DBM lock serializes operations within one API process, and waiters honor
request cancellation. For destinations bound to a stable volume identity, DBM
also holds the control-plane host volume lease from the final identity check
through the complete Kopia call. Supported host check, repair, unmount and
remount actions take the same cross-process lease, so they wait until DBM
releases it. This protects only cooperating owner paths: physical unplug,
kernel mount replacement, another tool that bypasses the owner, or an external
filesystem change can still race a path-based Kopia call. Do not describe the
lease as pinning the operating-system mount.
If a journal reports an `in_flight` step, stop and inspect the volume before
creating a new plan; DBM will not repeat an ambiguous filesystem action
automatically.

Readiness copies an explicit filesystem `dirty` or `needs-check` signal from
the mounted-volume metadata into the stable `destination_dirty` failure and
refuses the affected protection tier. When the platform exposes no such bit,
readiness reports health as unknown rather than claiming clean. Run the
recovery sequence above and rerun `destinations readiness` before retrying a
backup. DBM never guesses a device, repairs an unattributed volume, or
formats/clears a destination.

Note: `prepare-plan --json` wraps the plan as `{"plan": {…}}`, but
`prepare-execute --plan-json` expects the **inner** plan object. Extract it and
pass the exact `confirmation_phrase`:

First run `destinations prepare-plan --json`, extract the inner `plan` object
and its exact `confirmation_phrase` from the response, then pass those values
to `destinations prepare-execute --plan-json <plan-json> --confirm <phrase>`.
Start with `--dry-run true`; set it to false only after reviewing the plan.

After review, create the destination at the bundle root (a slug-safe name):

```bash
data-backup-manager destinations create --name elements-local --backend filesystem --location "/mnt/backup-drive/vrooli-backups"
```

This materializes a self-describing bundle at `<mount>/vrooli-backups`:
`README.txt`, `RECOVERY.txt`, `vrooli-backup-destination.json`, and the vanilla
kopia repository under `repositories/elements-local.kopia`. Open `README.txt` on
the drive to confirm it explains itself; the `repository_location` reported by
`destinations get` is the path to hand to plain kopia for standalone recovery.
For filesystem creation, DBM rechecks readiness and the same strong device
identity before preparing the bundle, creating the repository, and writing its
generated metadata. Unknown readiness cannot be overridden by acknowledging a
known readiness failure.

Creation persists a `creating` catalog intent before it invokes resource-kopia.
If the engine call outcome is ambiguous, or its confirmed-success state cannot
be persisted, the destination remains quarantined and unavailable to backup,
restore, retention, update, or deletion. Do not retry creation automatically,
issue another repository-create call, or infer ownership from its name/path;
preserve the row and files for explicit owner reconciliation. A durable
`created` row can be finalized automatically only after the existing bundle
manifest and Kopia repository unique ID match that destination before any
resource-kopia rebind. If that identity anchor is absent or conflicting, keep
the destination blocked for owner-led recovery. Do not delete the row or
repository to clear the error. Pending filesystem creation records a stable
volume identity and relative locator and uses the host-volume lease; the lease
still cannot prevent physical unplug or non-cooperating filesystem changes.

#### First real backup (data-safe order)

1. Run the disposable proof first (see "Gated backup proof" in
   [cli-commands](../reference/cli-commands.md)) under a `/tmp` root to confirm
   create → run → verify → restore → `diff` is clean and the bundle metadata
   contains no secret value.
2. Register default coverage so the plan protects all known non-regenerable
   durable state, not just one self-registered target:
   `coverage report` → `coverage accept-defaults --dry-run` →
   `coverage accept-defaults`. Plan creation is blocked while non-sensitive
   recommendations remain unregistered (override with
   `plans create --allow-incomplete-coverage` only on purpose).
3. Only then create the real external-drive destination and run a plan. Verify
   (`restores verify`) before trusting it; verify is non-destructive.

### Escrow handoff and combined evidence

Credential escrow remains owned by the control plane and its credential
authority. data-backup-manager owns durable target registration, encrypted
Kopia destinations, backup runs, restores, and recovery drills. Onboarding
connects the two through provider-neutral status and evidence; it never imports
this scenario's packages or runs its backup engine.

Treat the combined readiness view as complete only when both evidence chains
are current:

1. Credential escrow has a verified encrypted root-copy receipt and a verified
   recovery-bundle receipt whose source generation covers every configured
   credential.
2. This manager has a writable, encrypted, separate destination; an enabled
   plan; a successful snapshot; and a scratch-only verified recovery drill with
   checksum evidence.

The recovery bundle is not an ordinary backup target. Target registration
rejects the exact bundle locator recorded in the recovery receipt; import it
through the credential recovery owner instead. A destination that is read-only,
unmounted, inside a protected root, or physically unresolved remains pending or
degraded with remediation and must not be described as durable protection.

### Clean-node rehearsal

`scripts/clean_node_rehearsal.py` restores a reviewed set of exact snapshots on
a replacement node and then starts each named scenario through the control
plane. It requires a credential-recovery receipt from the credential owner,
bound by SHA-256 and `source_generation`; that receipt must say
`status: verified`. Import the detached bundle into the replacement node's
credential authority first with `vrooli credentials recovery restore --input
<bundle-path>`, then inspect `vrooli credentials doctor --format json`. Enter
the separately held recovery passphrase only through the credential owner's
secure prompt. The runner never reads the source host's credential store or
copies secret values. Do not put credentials in the manifest.

The JSON manifest has `schema_version: 1`, `rehearsal_id`, `source_generation`,
`credential_recovery_receipt` (`path`, `sha256`), `destinations` (each with
non-secret source-catalog fields `destination_id`, `name`, `backend`, `location`,
`created_at`, `cap_bytes`, and `cap_policy`), `targets` (each with `owner`, `name`, `kind`, `locator`,
`destination_id`, `snapshot_id`, and absolute `restore_location`), and
`scenarios` (scenario names whose restored data must be checked). The runner
requires at least one scenario name, so an empty health gate cannot produce a
successful receipt. Each target's receipt includes the scratch-verification
record ID and checksum plus the restore record ID, status, and snapshot binding.
imports each existing destination into the empty DBM catalog without creating
the repository, then re-registers targets by stable owner/name and uses those
new local target IDs for scratch verification and restore. Filesystem imports
require the existing DBM bundle, stable mounted-volume identity, matching
Kopia repository marker and credential reference. S3/SFTP imports reconnect
with their non-secret source-catalog descriptor and require every selected
snapshot ID to exist before the catalog row is persisted. Missing credential
authority values fail the reconnect; values never appear in the manifest or
receipt. The imported destination ID must match the manifest. The receipt is written atomically with
mode 0600 and retains only identifiers, outcome codes, health status, and
receipt digests.

For remote backends, preserve the non-secret fields from
`data-backup-manager destinations list --json` in the rehearsal manifest and
store that manifest with the off-host recovery material. The S3 location may
contain endpoint, bucket, prefix and region; the SFTP location may contain
username, host, remote path and `known_hosts` path. Never add a private key,
access key, passphrase, or credential value. A remote descriptor must include
the exact source-catalog destination ID/name/creation time and at least one
snapshot ID the rehearsal needs; the attach operation reconnects and confirms
those snapshots before adding a destination row to the new catalog. Provider
credentials and SFTP keys must already be restored/provisioned through the
replacement node's credential authority. Physical/topology independence still
requires separate evidence.

```json
{
  "schema_version": 1,
  "rehearsal_id": "fleet-node-2-2026w46",
  "source_generation": "escrow-generation-id",
  "credential_recovery_receipt": {
    "path": "/var/lib/vrooli/recovery/credential-receipt.json",
    "sha256": "<64 lowercase hex characters>"
  },
  "destinations": [{
    "destination_id": "stable-id-from-bundle",
    "name": "nightly-local",
    "backend": "filesystem",
    "location": "/mnt/off-host/vrooli-backups/nightly",
    "created_at": "2026-01-01T00:00:00Z",
    "cap_bytes": 0,
    "cap_policy": "alert-block"
  }],
  "targets": [{
    "owner": "example-scenario",
    "name": "durable-state",
    "kind": "filesystem",
    "locator": "data",
    "destination_id": "stable-id-from-bundle",
    "snapshot_id": "snapshot-id-from-backup-catalog",
    "restore_location": "/var/lib/vrooli/rehearsal/restore/example-scenario"
  }],
  "scenarios": ["example-scenario"]
}
```

```bash
python3 scenarios/data-backup-manager/scripts/clean_node_rehearsal.py \
  --manifest /path/to/rehearsal-manifest.json \
  --receipt /var/lib/vrooli/rehearsal/node-2.json
```

Each selected snapshot is scratch-verified with `data-backup-manager restores
verify` before the runner restores it to the explicit destination path. After
all restores, the runner starts each scenario, waits
once for the in-flight lifecycle operation, and requires `scenario status` to
report both `running` and `healthy`. A failed step leaves a durable `failed`
receipt with completed target/scenario outcomes. This path produces rehearsal
evidence only when run on the clean replacement node; fixture tests do not
qualify a real backup, escrow, or restored scenario.
Every restore location must be absent or an empty real directory, must not
overlap another restore location or destination bundle, and is rejected if it
already contains files; this prevents a rehearsal from overwriting old data.

### Back up (scheduled + on-demand)

Plans run on cadence via the in-process scheduler. Operators (and
scenarios) can trigger a run manually:

```bash
data-backup-manager runs trigger --plan "plan-id"
data-backup-manager runs list
```

When a run is stopped by a destination problem that looks transient, the
backup-failure monitor asks for one automatic retry. A retry is recorded with
trigger `retry` (shown as "Automatic retry"), not `manual`, and `runs stats`
counts retries on their own line so they never lower the success rate of real
attempts. If the retry fails the same way, the next one for that plan is held
back: 5 minutes, then 15, 45, 2h15m and never more than 6 hours between
retries. A held retry creates no run. A scheduled or manual run, a success, or
a different failure starts the count again; an operator's own `runs trigger` is
never held. Before this, the monitor re-ran a blocked plan every minute and
left thousands of failed `manual` runs (2026-11-13 to 2026-11-15, eight
baseline-safety plans); those rows were relabelled once, as below. Run stats cover the newest 1000
attempts and only the retries inside that span, so retries cannot crowd real
attempts out of the window.

One-off, 2026-11-15 (record only; do not repeat): the 3,728 failed rows of that
loop were relabelled from `manual` to `retry` in one transaction, selected by
`trigger='manual' AND status='failed' AND error LIKE '%filesystem readiness is
unknown%' AND started_at` between 2026-11-13T07:11:45Z and 2026-11-15T15:13:37Z.
Before: 3,728 matching rows, 3,770 `manual`/failed in all. After: 0 matching,
42 `manual`/failed left, 3,728 `retry`/failed. Only the `trigger` column changed;
a copy of the database from just before the change is kept with the work notes.

Before target fan-out, each run performs a read-only plan-wide preflight. It
checks repository connection and enumerates snapshot metadata before source
capture; a repository that opens but has missing or unreadable snapshot
manifests is blocked before a new write. A shared destination, repository, or
credential failure is recorded once as a grouped incident, and affected work
is blocked without capturing source data. Inspect
the run JSON for `failure_code`, `next_action`, and `preflight_incidents`.
Never retry a `credential_missing`, `destination_dirty`, `destination_read_only`,
or `destination_unmounted` incident repeatedly; follow its next action first.

The `/health` backup dependency checks freshness and quick connection status
for repositories used by enabled plans. It does not enumerate repository
manifests; that deeper, potentially slower check runs in preflight before each
backup write. A healthy `/health` result therefore does not certify that every
registered or disabled repository can restore.

Plans expose an independent `protection_tier` (`full_primary`,
`critical_primary`, or `critical_secondary`). Configure critical plans with
their own target and destination membership; do not infer secondary health
from the full-primary plan.

For an attached drive, use `destinations readiness --location <mountpoint>`.
The response includes stable device identity, confidence, evidence source, and
non-destructive repair steps. Safe remount is automatic only after the
control-plane volume owner proves a positive offline consistency check. If a
filesystem repair may be needed, DBM presents one identity-bound Notification
Hub decision. Only the verified-human callback can perform the exact approved
repair after revalidating device identity and the prepared operation; inspect
the result notice and readiness receipt before retrying backups. The control
plane owns all host actions. DBM never formats, partitions, clears, or repairs
a volume without that verified decision, and never remounts after an
inconclusive check.

The current incident class of a dirty/unmounted NTFS volume and an unavailable
repository credential is two independent blockers. A power outage is not
inferred from a dirty volume alone.

Source capture is per kind: filesystem (direct), SQLite (`VACUUM INTO`),
Postgres (`pg_dump`), Redis (prefix `SCAN`+`DUMP`, best-effort —
non-transactional), Qdrant (snapshot API), object-storage (S3/MinIO
mirror). All artifacts land in an encrypted destination; none are
written under the scenario source tree.

### Verify (the recoverability gate)

Verified restore is first-class. Verify mode test-restores a target to a
scratch location and checksums the result, proving the backup is
actually recoverable:

```bash
data-backup-manager restores verify --target "target-id" --destination "destination-id" --snapshot "snapshot-id"
```

**Rule:** committed runtime data is removed from git only after its
target has a passing verified restore. A target that cannot verify is
treated as un-recoverable until it does.

### Restore (privileged)

Restore rehydrates source data to an operator-chosen location. It is a
privileged operation (see `../internal/SECURITY.md`):

```bash
data-backup-manager restores restore --target "target-id" --destination "destination-id" --snapshot "snapshot-id" --location "/tmp/restore-check"
```

### Standalone restore (Vrooli down)

Because destinations are plain kopia repositories, recovery does not
depend on this scenario being up. With the repo passphrase from the credential
authority,
an operator can restore directly with the kopia CLI even if the manager,
API, or the whole Vrooli stack is unavailable. This is a deliberate
disaster-recovery property — keep the passphrase recoverable
independently of the data it protects.

For a filesystem destination, the drive itself explains how: read
`RECOVERY.txt` at the bundle root, and connect plain kopia to the **nested**
repository path (`repository_location` /
`<bundle-root>/repositories/<slug>.kopia` — recorded in
`vrooli-backup-destination.json`), **not** the bundle root. The manifest also
records the credential identity/field `secret_ref` (a reference, never the
passphrase value); import it with the command in `RECOVERY.txt`.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Registered source targets | Scheduled/on-demand plan run into an encrypted kopia destination | `restores restore` (whole target in v1); `restores verify` to prove recoverability | implemented; must be validated with a real destination/plan/run per installation |
| Manager catalog (SQLite) | Not separately backed up — reconstructable from scenario re-registration on boot | Rebuilt automatically as scenarios re-register | by design |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration

## Workspace checkpoint preparation

1. Inspect the intended destination and run `runs preflight --plan <id> --json`.
   Temporary destinations are not eligible for durable protection. A passing
   preflight is readiness evidence, not proof of a fresh backup or restoration.
2. Select a permanent repository. Preserve old repository bytes and credential
   identities when moving a plan; do not initialize over a missing repository.
   Keep any newly prepared plan manual/disabled until capture is authorized.
3. Register an explicit `workspace-checkpoint` target with an absolute source
   directory. Inspect the profile limits under `docs/internal/SEAMS.md` first.
   Registration does not capture files. Do not use a live application directory
   as evidence of an atomic snapshot; this profile records observed stability.
4. Before a future authorized run, supply a stable source and allow sufficient
   space for a full staged copy plus engine storage. Stage outside the source.
5. Verification must restore the envelope AND materialize its tree into fresh
   scratch storage, then compare the result with the capture manifest. A damaged
   or unsupported checkpoint must fail. Never point validation at a live target.

The workspace profile does not yet certify this machine's full Vrooli checkout:
external Git storage, non-current owners/groups, hard links, ACLs/xattrs, special
files and unsupported platforms are refused. Do not weaken those checks to make
an unsupported tree report success. There is no tested production-engine receipt
for this new source kind; mocked validation does not establish one.

## Backups blocked on a mounted drive

DBM watches enabled plans. When a run is stopped by a destination-level reason it
diagnoses read-only and raises one Notification Hub decision. Available choices
depend on the typed damage and the host repair path.
Do not hand-run kopia repair, delete or rename repository files to clear it.

1. Read the decision: it states the cause, days stopped, and items with no other
   copy. Its technical details list every unreadable repository path.
2. Skipping unreadable paths records an exception only for each exact unreadable
   leftover temp listed in the decision. `runs preflight --plan <id>` then passes
   only while all unreadable entries remain covered. Exception files are under
   the scenario state root, `backup-exceptions/`.
3. For a complete set of unreadable EIO session temps, the decision can also
   offer Kopia quarantine only when every entry is an exact, unique orphan
   `s/...f.tmp` session-marker path and the destination has stable filesystem,
   UUID or serial identity. One verified answer covers only the listed set.
   DBM revalidates the same device, repository root, and typed damage scan while
   holding the host volume lease; resource-kopia moves each entry to a
   same-volume quarantine, preserves its bytes, and writes no final blob. DBM
   records progress per path, rescans for unreadable entries, and can safely
   resume after interruption. Changed identity/root/set remains applying for
   review without continuing the effect.
4. For a precise missing q-blob, DBM uses the operator's standing 2026-11-04
   approval only when resource-kopia proves one exact disposable temp candidate,
   the final blob is absent, the source repository stays read-only, and an overlay
   snapshot listing passes. DBM calls resource-kopia's typed `repo restore-from-temp`
   operation with the repository identity, exact relative path and proven hash;
   it verifies the returned source/target and preservation receipt, rescans the
   repository, sends a result notice, then retries affected plans. If any proof
   is absent or changes, DBM makes no repository write and uses the normal
   decision path. The CLI confirmation phrase binds the operation; it is not
   human authorization.
5. Filesystem repair takes the drive offline, repairs and remounts through the
   control plane; it is refused while a backup is running.
6. A verified fresh-repository answer currently records a durable request and
   waits for destination selection; DBM does not guess a backend or path. The
   reusable migration workflow requires capacity proof, fresh captures and
   scratch restore/drill receipts for all 14 targets before changing plan
   bindings, and the old repository remains preserved. Production integration
   of that workflow is still in progress, so an accepted answer does not yet
   start migration or change the plan.
7. Declining or not answering leaves backups blocked. The stalled-backup notice
   repeats daily; the question returns after a day (declined) and is asked again at
   once if the hub expires it unanswered after its week-long deadline.
8. The question never piles up. Notification Hub keeps one open decision per
   destination (`backup-failure:<destination-id>`; a drive repair question is
   `volume-repair:<recovery-id>`). DBM closes it by itself, with the reason shown
   to you, when backups succeed again, the problem changes into one with nothing to
   offer, the destination is removed, or no plan is switched on for it; if the
   problem changes into a different one the same card is updated instead. If you
   see a DBM decision that is plainly stale, DBM is not running (the hub also
   closes an unconfirmed decision about 6 hours after DBM last confirmed it).
   After a deploy, a decision published by an older DBM is replaced by its keyed
   copy and the old one is closed; this happens on DBM's next sweep, no action needed.

