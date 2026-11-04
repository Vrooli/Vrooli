# Progress — Data Backup Manager

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/data-backup-manager/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-11-03 | Codex | Declared DBM's recovery-state storage surfaces. | The recovery worker writes durable journals/episodes and Notification Hub decision records under the canonical state root. Added `volume-recovery` and `volume-recovery-decisions` entries with measured budgets after the comprehensive storage phase identified an uncovered journal. After the inventory cache refreshed, Storage Manager reported both entries and all 471,319,544 observed DBM bytes accounted for. A fresh Test Genie storage phase remains pending. |
| 2026-11-03 | Codex | Fixed the UI's vulnerable Express transitive dependencies and reduced new handler files below the tidiness limit. | Used Scenario Dependency Analyzer to add DBM-only exact approvals and pnpm overrides for `body-parser@1.20.9` and `proxy-addr@2.0.8`, resolving GHSA-27r3-jxmh-p828 and GHSA-jqcg-44mw-7w3h. Split protobuf conversion and notification delivery helpers out of the oversized destination handler files. Test Genie run `20261103-221404-826c4f93` passed contracts, unit, and security; tidiness still fails because total duplication debt is 734 over its recorded baseline. The first wait timed out during provider preparation, but the server-owned run later reached terminal state. Elements is mounted read/write, its offline check is consistent, but current mounted NTFS state remains unknown and its Kopia repository still fails on missing manifest data plus EIO. |
| 2026-11-03 | Codex | Rechecked Elements through the control-plane volume owner. | With no user process holding the exact identity-bound mount, the control plane unmounted `/dev/sda1`, UDisks2 `Filesystem.Check` returned `consistent=yes` and `changed=false`, and it remounted read/write at `/media/matthalloran8/Elements`. This is fresh offline consistency evidence, but mounted NTFS-3G still exposes no current clean/dirty state. No Kopia repository files were changed. |
| 2026-11-03 | Codex | Completed recovery callback follow-up and refreshed DBM validation. | Fixed compile issues found while validating endpoint descriptor extraction and the recovery response tuple. Test Genie run `20261103-224150-efbc82e6` passes unit, contracts and security; tidiness still fails the complexity budget by 12 (`105` vs `93`). Restarted through `make restart`; API health remains degraded from 13 failed backup posture outcomes. Elements' offline NTFS check is consistent, but mounted state remains UNKNOWN and the Kopia repository still fails on missing manifest data plus `EIO`. No repository repair/write occurred; independent off-host Treasury backup and restore remain open. |
| 2026-11-03 | Codex | Reconciled DBM's remaining Elements blockers and capacity decision. | All 13 overdue Elements targets still have `repository_invalid` failures from one run; no source capture occurred and their last-success evidence remains stale. Read-only inspection found 13 distinct repository `.tmp` paths returning `EIO`; no zero-length repository files were found. `resource-kopia repo validate --name elements-local` exited successfully but is not equivalent to snapshot enumeration; `snapshot list` still fails on a missing manifest. For the full-plan apparent estimate (~325 GB), Elements has ~1.62 TB free: 3 retained copies fit, 7 do not. `baseline-safety` is same-host and fails the 3-copy capacity estimate. The Notification Hub retention decision remains pending, so no plan metadata changed. No repo files or backup data were modified. |
| 2026-11-03 | Codex | Repeated the identity-bound offline check and added a non-committing Kopia index scan. | Control plane unmounted the exact Elements UUID/serial, UDisks2 returned `consistent=yes`, `changed=false`, and the control plane remounted it read/write at its recorded root. Direct `ntfsinfo` could not open the device due OS permission limits; current NTFS-3G readiness stays UNKNOWN. Snapshot list and destination usage still fail on missing blob `q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144` and repository `EIO`. Added `resource-kopia repo index-recovery-scan`, which runs upstream `index recover --no-commit` for a registered filesystem repo only; focused repo/app tests pass. Live scan session `88237` was still running at 23:39Z, having scanned 6,699 of ~17,161 pack blobs and recovered 690,175 index entries in its read-only pass. No indexes were committed; result pending. |
| 2026-11-04 | Codex | Completed the repository index scan, fixed scan output capture, applied the operator's retention decision, and prepared a gated index commit command. | Kopia's scan completed successfully in non-committing mode: 4,024,881 content entries recovered from 17,232 pack blobs; no repository indexes were committed/deleted. The wrapper's first run reported a false empty-output error because successful stderr was not captured; `kexec.Call.CaptureStderr` now opts the scan into stderr capture and the command sets info logging/no progress. A separate resource-owned commit command is registered, requires the exact repository-name confirmation phrase, uses `index recover --commit`, and does not enable index deletion; the lifecycle rebuild succeeded, but the command was not run. Notification Hub answer `retain3` was applied to `elements-daily-runtime` with the explicit incomplete-coverage override; plan verified `keep_latest=3`, still disabled, no backup triggered. Decision `ad0a46dc-fac9-4704-92d5-4a600190b625` was delivered for the index commit; it remains pending. Snapshot list/usage still need recheck after approval. |
| 2026-11-04 | Codex | Committed the operator-approved additive index recovery and verified it did not restore repository usability. | Verified-human answer `commit` to Notification Hub decision `ad0a46dc-fac9-4704-92d5-4a600190b625` authorized only additive index recovery for `elements-local`; the resource-owned command committed recovered indexes and explicitly did not delete indexes. Fresh Kopia snapshot enumeration still fails on missing manifest blob `q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144`, and DBM usage still reports that blob plus `EIO` on repository path `s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838`. The exact volume again passed identity-bound offline check (`consistent=yes`, `changed=false`) and remounted read/write; mounted NTFS state remains UNKNOWN. SMART/kernel diagnostics are unavailable without host privilege (`sudo -n smartctl` requests a password; `dmesg` is restricted). Retention is `keep_latest=3`, the 14-target plan remains disabled, and 13 outcomes remain failed/overdue; no new backup was triggered. Result notification `c3bc384a-70cc-47c3-a3bc-97b435c13a23` was delivered by web push receipt `cf77d26a-388a-414c-8575-84623c060c0c`. |
| 2026-11-04 | Codex | Isolated Kopia's registered cache and narrowed the repository failure to storage or repository data. | Added and rebuilt `resource-kopia repo cache-probe`, which reconnects a registered filesystem repository with a temporary config/cache, lists snapshots, and removes diagnostic state without touching the registered cache or repository. Focused `go test ./internal/repo ./internal/app` passes. A live probe again fails on missing manifest blob `q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144`; no diagnostic temp directories remain. The exact WD Elements device reports USB mass-storage transport, SCSI state `running`, and read/write block nodes, while DBM still sees NTFS dirty state UNKNOWN. Mac Bridge node discovery found no attached drive; its relay rejected read scope as ungranted, so no remote call was dispatched. No backup, restore, repair, or drive mutation occurred. |
| 2026-11-03 | Codex | Verified Treasury DBM backup and scratch restore; clarified plan and destination-readiness contracts. | The encrypted `baseline-safety` repository accepted a manual Treasury snapshot (424,778,594 logical bytes; 291,406,853 physical bytes), and `restores verify` returned VERIFIED with a recorded checksum. A manual-only plan retains three snapshots and is disabled after the proof. The destination is on the same host disk, so this is pipeline evidence, not off-host recovery protection. Removed the unsupported `plans create --enabled` option from the canonical CLI manifest and refreshed the installed CLI through `make setup`; disable plans through `plans update --enabled false`. `AnalyzeDestination` now supplies registered filesystem roots to its already-configured check. Elements remains blocked by unreadable Kopia metadata and mounted NTFS filesystem state UNKNOWN. |
| 2026-05-26 | matthalloran8 | Scaffolded scenario from the `react-vite` template. | Archived |
| 2026-05-26 | matthalloran8 | Authored PRD and requirements: runtime-state backup with self-registration, six source kinds, kopia-backed encrypted destinations, many-to-many plans, in-process scheduler, and verified restore. | Archived |
| 2026-05-26 | matthalloran8 | Wrote companion `kopia` resource plan (`plan-manager plans get kopia-resource-implementation-validation-plan`) — the engine this scenario wraps. | Archived |
| 2026-05-26 | matthalloran8 | Locked architecture decisions (see `DECISIONS.md`): kopia wrap, Source/Destination/Plan model, encryption-on default, alert+block storage limits, verified-restore gate, separate-root rule, no n8n. | Archived |
| 2026-05-26 | matthalloran8 | Filled INTERNAL/OPERATIONS/BUSINESS docs to reflect the locked design. | Archived |
| 2026-05-26 | matthalloran8 | API+CLI implementation pass. | Archived |
| 2026-05-26 | matthalloran8 | Discovery domain (onboarding suggestions, Track B #1+#3). | Archived |

| 2026-06-03 | matthalloran8 | Coverage domain (default-coverage activation). | Archived |

| 2026-06-03 | matthalloran8 | First real backup to the Elements drive + capturer/transport fixes. | Archived |

| 2026-06-03 | matthalloran8 | Async run execution + persisted lifecycle + startup reconciliation (perf/observability plan, Phase 1). | Archived |

| 2026-06-03 | matthalloran8 | Bounded-concurrency target fan-out (perf/observability plan, Phase 2). | Archived |

| 2026-06-03 | matthalloran8 | In-place snapshot for filesystem sources (perf/observability plan, Phase 3). | Archived |

| 2026-06-03 | matthalloran8 | Per-target kopia overhead — Phase 4 investigated & descoped (perf/observability plan). | Archived |

| 2026-06-03 | matthalloran8 | Run metrics surface (perf/observability plan, Phase 5). | Archived |

| 2026-06-03 | matthalloran8 | Cadence / freshness view (perf/observability plan, Phase 6). | Archived |

| 2026-06-03 | matthalloran8 | Deferred follow-ups: repo-stats fix, dedup metric, next-scheduled view, async restores, overhead spike | Archived |

| 2026-06-03 | matthalloran8 | Generic snapshot audit (new `audits` domain) + DBM-UI-001 closed | Archived |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-05-26 — Discovery domain (onboarding suggestions, Track B #1+#3).**: Well-known scanner covers `~/.vrooli` runtime state (`plans`/`state`/`config`/`secrets.json` fs + `runtime.db` sqlite; scenario stores deferred via an additive `rootKind`).

- **2026-06-03 — First real backup to the Elements drive + capturer/transport fixes.**: `restores` `checksumDir` made symlink-safe; (3) `sources/sqlite.go` single-file artifact unrestorable into a directory target — now staged as a directory (`sqlite/snapshot.db`); (4) transport: `runs trigger`/`restores verify`/`restores restore` CLIs now use an unlimited-timeout Connect client and the API sets `WriteTimeout: 6h`, so multi-minute synchronous backup/restore RPCs no longer disconnect mid-kopia (client `Client.Timeout`/server 30s `WriteTimeout` were killing the exec'd kopia and wedging runs in PENDING). Filed separately: sync-execution/PENDING-reconciliation robustness gap and `resource-kopia repo stats --json` (breaks `destinations usage`).

- **2026-06-03 — Async run execution + persisted lifecycle + startup reconciliation (perf/observability plan, Phase 1).**: **Validated on the real Elements harness:** trigger returns in ~2s (pending); live `capturing`→`snapshotting` with outcomes accruing durably; full 16-target run completed 16/16 in ~45s; a hard `kill -9` mid-`capturing` left an orphan that startup reconciliation closed to `failed` (zero non-terminal runs remain); the migration preserved all 18 pre-existing runs and reconciled the 2 prior-session PENDING orphans.

- **2026-06-03 — Run metrics surface (perf/observability plan, Phase 5).**: **Dedup/physical-bytes deliberately deferred** (no silent drop — documented in PROBLEMS.md): kopia `snapshot create --json` exposes only logical size (empty `stats`, no uploaded bytes) in this build, and `repo stats --json` is separately broken, so a dedup ratio can't be computed reliably yet.

- **2026-06-03 — Cadence / freshness view (perf/observability plan, Phase 6).**: `next_scheduled_at` deferred with a note (needs a scheduler `lastFire` seam — see PROBLEMS.md).

- **2026-06-03 — Deferred follow-ups: repo-stats fix, dedup metric, next-scheduled view, async restores, overhead spike**: **Deferred follow-ups: repo-stats fix, dedup metric, next-scheduled view, async restores, overhead spike** (plan `data-backup-manager-deferred-follow-ups-...`). **(E)** Per-target overhead: spike measured ~0.2–0.5s connect/target with kopia's on-disk cache already persisting across spawns and targets already fanning out at concurrency 4 (~2s/run total) → **re-deferred** (kopia server mode is a large change and the per-target override-source identity likely can't survive its snapshot API; rationale in PROBLEMS.md, no scaffolding left).

## 2026-11-04 — isolated overlay proves one exact manifest candidate

Resource-kopia's isolated cache probe found one adjacent 75,403-byte temp file
whose SHA-256 matches the absent Elements `q` manifest blob. A disposable
filesystem overlay containing that candidate at the expected path passed Kopia
snapshot enumeration; the original repository and temp candidate remained
untouched. A separate owner command now stages, rehashes, and atomically links
that one blob only when the final path is absent. The Notification Hub decision
for that exact write is pending (`8939a7e8-8ffa-4604-92d9-24701256638f`). The
Elements plan remains disabled; the separate EIO paths, 13 failed/overdue
targets, and off-host verified restore remain unresolved.

## 2026-11-04 — first exact blob repaired; a second manifest gap surfaced

Notification Hub approved the first exact missing blob repair. Resource-kopia
published it with the reviewed hash, preserved its `.tmp` source, and did not
replace a target. A follow-up notification was sent. Snapshot listing advanced
to a different missing manifest blob; a second exact orphan-temp candidate also
passes in the disposable overlay. Its separate Notification Hub ask is pending.
Destination readiness remains UNKNOWN for mounted NTFS dirty-state observability,
and destination usage still fails on manifest loading plus an unrelated EIO.

## 2026-11-04 — second approved blob repaired; third missing blob surfaced

The second hash-bound Notification Hub decision was approved by a verified
human and its exact blob was published without replacing a target. The source
temp remains, and a follow-up notification was sent. Snapshot listing advanced
to missing blob `q24decdf22b7ca74f9f48f99ca06937d0-sf40366ce03934d79144`; the
next isolated-cache probe is running. Destination usage still hits a separate
EIO. The Elements plan remains disabled.

## 2026-11-04 — third orphan candidate identified

A slower isolated-cache probe found one more exact missing manifest candidate
and passed snapshot enumeration in the overlay. The exact q947 repair ask is
pending. The prior direct list error named another missing object (q24...), so
there may be multiple gaps or nondeterministic traversal order. Separate EIO,
UNKNOWN mounted NTFS state, and failed/overdue targets remain; the plan is still
disabled.

## 2026-11-04 — fourth approved blob restored; targeted diagnostic added

A verified-human Notification Hub decision approved the exact q06 candidate;
resource-kopia restored it with source temp preserved and no replacement, then
a follow-up notification was sent. A new optional `cache-probe --blob-id` flag
tests a known missing object directly in the disposable overlay. The persistent
q24 error is being tested with that flag; no fifth repository write is
approved.

## 2026-11-04 — targeted q24 candidate confirmed

The explicit `cache-probe --blob-id` path found the exact adjacent q24 candidate
and passed isolated-overlay snapshot listing. Its separate Notification Hub
approval is pending. Baseline enumeration in that probe also surfaced a q3d
missing blob; no q24 repository write has run yet.

## 2026-11-04 — q24 and q3d repaired; qe090 candidate awaits decision

The operator approved both exact repairs and resource-kopia published each
hash-verified blob without replacing targets or deleting source temp files.
Follow-up notifications were sent. The next exact orphan candidate passes the
targeted isolated overlay and its Notification Hub decision is pending. A
separate repository EIO and UNKNOWN NTFS readiness still block backup usage.

## 2026-11-04 — qe090 repaired; qcae awaits approval

The operator approved the exact qe090 candidate, which was restored with its
source preserved; execution and post-check result notifications were sent.
Kopia now surfaces qcae, whose targeted disposable overlay passes with its
unique candidate. Its separate Notification Hub decision is pending. The plan
remains disabled and the independent EIO remains.

## 2026-11-04 — qcae repaired; qf020 awaits approval

The verified-human decision approved qcae; it was restored with its source temp
preserved and no replacement, then a follow-up notification was sent. The next
qf020 candidate passes targeted overlay snapshot listing and its own decision
is pending. Separate EIO and backup-readiness gates remain.

## 2026-11-04 — standing authorization and snapshot enumeration recovery

The operator's verified-human Other note explicitly approved this exact blob
restore and all future ones. The commission now records that standing approval
for only unique hash-verified adjacent temp q-blob candidates whose target is
absent and whose isolated overlay passes snapshot listing. No more per-blob
questions are needed within that scope. Nine blobs have been restored; Kopia
snapshot enumeration passes (192 total, 189 DBM-tagged). The control plane also
ran a clean offline check and remounted the exact Elements drive. Destination
usage remains blocked by an unrelated `s/...tmp` EIO; mounted NTFS dirty state
remains UNKNOWN. Keep the plan disabled until those gates plus target backup
and independent restore proof pass.

## 2026-11-04 — exact content-temp quarantine failed safely

Kopia snapshot enumeration now passes, but repo stats and DBM usage still fail
on one `s/...f.tmp` entry with EIO. A new resource-kopia exact quarantine
operation was built to preserve an unpaired temp outside the shard namespace;
the exact rename also returned EIO, leaving the source unchanged and no
quarantine residue. The control plane independently passed an offline NTFS
check (`consistent=yes`, `changed=false`) and remounted the device. DBM still
reports 13 failed target postures; the Elements plan remains disabled. Resolve
the storage-level EIO before backup writes.

## 2026-11-04 — recovery monitor activated; repository EIO remains

Restarted Data Backup Manager through the control-plane lifecycle so the
volume-recovery monitor and Notification Hub callback wiring are active. The
startup scan observes the Elements volume mounted read/write and does not
modify it. Vrooli Events confirms the enabled signed resolution subscription
for DBM. Destination usage still fails on the exact unpaired `s/...f.tmp`
entry with EIO; unprivileged SMART inspection is unavailable and readable
kernel logs provide no matching device evidence. The Elements plan remains
disabled, with 13 failed/overdue target postures. Keep the temp intact and
continue read-only device/repository diagnosis before writes.

## 2026-11-04 — backup preflight now blocks unreadable repository usage

A live read-only plan preflight reported `ready=true` even though DBM destination
usage and Kopia `repo stats` failed on the exact Elements EIO path. Preflight
previously checked repository status and snapshot enumeration but did not check
whether repository usage could be read; the plan has no storage cap, so a run
could have entered source capture without reliable usage/capacity evidence.
Added a repository stats probe after the status and snapshot-list checks. The
API builds and the supported lifecycle restart deploys it. Re-running plan
preflight now returns a grouped `repository_invalid` incident for all 14
selected targets before source capture. No backup, restore, prune, or drive
mutation ran. Keep the plan disabled until the EIO and NTFS readiness/capacity
gates are resolved.

## 2026-11-04 — lower-bound capacity estimate measured

A metadata-only size scan of all 14 Elements-plan locators observed at least
328,472,976,155 bytes. Elements has 1,621,584,723,968 bytes free; three copies
of that lower bound pass the DBM readiness capacity check. The source scan was
not complete because protected files under `.vrooli/state` returned permission
denied, so this is not a final capacity qualification. Repository usage still
fails with EIO and preflight blocks before source capture.

## 2026-11-04 03:12Z — source preflight and transport evidence

Source preflight now measures local filesystem and SQLite targets before destination readiness. It uses the filesystem capturer's tree walk, treats incomplete enumeration as unknown capacity, and passes the complete source estimate plus plan retention into the shared destination capacity check. The live read-only Elements preflight now reports two independent blockers: repository usage fails on the exact Kopia `.tmp` EIO path, and the `vrooli/state` target cannot enumerate one mode-0600 Agent Manager receipt directory. The selected target is blocked before capture; capacity remains unknown rather than passing on the earlier lower bound. API build and supported lifecycle restart succeeded; no tests were run.

The operator answered Notification Hub ask `d085cf69-1c61-4e4c-ac8e-d78b746dd32a` to hold filesystem repair and obtain SMART/transport or alternate-host diagnostics. The drive was not changed. SMART access is denied because `/dev/sda` is `root:disk` mode `0660`. Read-only SCSI sysfs values showed two more command errors during the observation interval (`ioerr_cnt` `0x1787` → `0x1789`; request count `0x80606` → `0x807ad`), with state running, no blocked device, and zero timeouts. This supports a transport/device concern but does not identify the failing LBA or establish drive health. Repository temp remains preserved; Elements plan remains disabled.

03:17Z follow-up: A paired direct `stat` of the exact temp and DBM usage probe reproduced EIO without changing SCSI counters (`iorequest_cnt=0x807ad`, `ioerr_cnt=0x1789` before and after). The earlier two-error rise is not attributable to that exact object. Bug report `knw-1793762204365449738` routes the inaccessible Agent Manager receipt directory to its owner; no mode change was made.

## 2026-11-04 03:27Z — receipt traversal repaired; preflight narrowed

The exact Agent Manager `retry-preflight` directory was owned by the current
operator and had mode `0600`, which made it impossible for even its owner to
traverse. Changed only that directory to owner-only `0700`; contents and all
other paths were left untouched. A fresh read-only plan preflight now reports
only the grouped repository-usage `repository_invalid` incident across the 14
Elements targets. It reports no source-enumeration or capacity incident, so
the selected source set is fully enumerable and passes current destination
capacity checks with retention three. The earlier lower-bound-only capacity
qualification is superseded by this complete traversal, but repository usage
still fails on EIO and prevents backup capture.

Fresh `system-monitor metrics devices --json` at 03:27Z again identifies the
Elements drive as `/dev/sda` on USB/SCSI and reports SMART as unmeasurable due
to permission denied. No raw device permission was changed. Preserve the EIO
temp and keep the Elements plan disabled pending the operator-directed SMART,
transport, or alternate-host diagnosis.

## 2026-11-04 03:50Z — one-shot SMART path added; root broker update awaits operator

The `smartctl` host-tool hint claimed `vrooli setup` provisions raw-device
permissions, but `vrooli setup explain smartctl` returns “no host requirement
named smartctl.” No persistent permission path exists. Added a control-plane
`vrooli host volume smart` action backed by the privilege broker. Its request
requires an exact whole-disk serial; the broker rechecks serial and disk type
with `lsblk`, then invokes only `smartctl -j -H -A`. No scenario receives
persistent block-device access. The CLI and owning Go packages compile, and
`make install` installed the updated CLI.

The active root broker still advertises the old action set. `vrooli setup
broker --sudo-mode ask` attempted the supported update but stopped at the sudo
password prompt (`sudo: a password is required`). No ACL, drive, or repository
change occurred. The operator answered SMART access ask
`60c85fc6-bb92-4167-8809-dc6a1cd79f51` with verified-human `authorize_smart_read`;
the remaining operator action is to run the setup command in an authenticated
terminal. Elements preflight still reports repository-usage EIO only; plan
remains disabled.

## 2026-11-04 03:58Z — refreshed readiness and operator handoff

Re-read destination readiness using the fully traversable selected source set
(`328,472,976,155` bytes, retention three): destination presence, read/write
mount, bounded directory inspection and capacity pass. Overall readiness stays
`UNKNOWN` because the current NTFS-3G/FUSE mount does not expose its
dirty/needs-check state. Fresh destination usage still fails with `EIO` at the
same preserved unpaired Kopia content temp; the 14-target plan remains
inactive and 13 outcomes remain failed/overdue. No backup, restore, prune,
repository mutation or drive repair ran.

The exact-serial, fixed-command SMART owner path is compiled and installed,
but the active root broker has not loaded its new action. The supported broker
setup stopped at the sudo password prompt, so the already-approved SMART read
still cannot execute in this session. Notification Hub decision
`8d842870-6d1b-4676-8809-dc6a1cd79d50` was sent through the authorized agent
session and delivered by Web Push (notification
`4b6dba77-1daa-414c-9327-bd59495e65c7`, receipt
`b0331b17-afc1-4dfe-b814-b8910e8c685e`). It asks the operator to update the
broker in an authenticated terminal, use an authorized second computer, or
hold recovery; Notification Hub also provides its built-in Other response and
dismiss action. The action ledger is
`/home/matthalloran8/.vrooli/research/ATTENTION.md`.

## 2026-11-04 04:05Z — SMART result serialization repaired at the broker owner

The operator selected the setup route; the root broker now advertises
`volume.smart.inspect`. The exact-serial read executed and returned SMART
`passed=true`, but live inspection found that the broker's 8-KiB prefix of raw
smartctl JSON was incomplete and could not be parsed. Replaced raw-prefix
truncation with a bounded valid-JSON summary carrying device model/protocol,
health, temperature, exit status, ATA attributes, and available error/self-test
summaries. The control-plane packages compiled and `make install` installed the
updated CLI. The active broker still has the old serializer; updating the
setup-managed broker requires sudo authentication not available to this
session. Notification `588ef5a2-5b4c-4bf6-924a-9cfd09cdfaae` was delivered by
Web Push receipt `9eeed729-812a-42c1-95ae-22a98ee451f8` with the exact rerun
command and full action-ledger path. No filesystem or repository write ran.

## 2026-11-04 04:09Z — partial SMART findings from the truncated response

A second authorized identity-bound SMART read confirmed `smart_status.passed`;
complete attribute objects available before the broker's 8-KiB truncation show
35°C, zero reallocated sectors (ID 5), zero reallocation events (ID 196), and
zero current-pending sectors (ID 197). Later attributes and ATA error/self-test
logs are not present in the truncated result, so this does not qualify the
whole SMART report or rule out USB transport/filesystem damage. The bounded
serializer is built and installed in the control-plane owner, but the separate
setup-managed root broker remains on its older serializer. The EIO, NTFS
readiness UNKNOWN, inactive plan and overdue targets are unchanged; no disk or
repository write ran.

## 2026-11-04 04:14Z — SMART query-window SCSI counter change

Bracketed one approved SMART read with read-only `/sys/block/sda/device`
observations. `ioerr_cnt` changed from `0x1793` to `0x1794` while
`iorequest_cnt` changed from `0x80b11` to `0x80b15`; after the probe,
`iodone_cnt` equaled `iorequest_cnt` and device state remained `running`.
The probe window therefore contains one additional SCSI error across four
requests. Concurrent host I/O prevents attributing that error specifically to
smartctl or to the preserved Kopia object, but it reinforces that the USB/SCSI
path must be treated as suspect. No matching kernel log records were readable.
Stop repeating SMART reads until the full broker summary is deployed and the
transport path is inspected. Preserve the temp and keep the plan inactive.

## 2026-11-04 04:17Z — USB path identified; fresh plan preflight

Read-only USB topology places the WD Elements disk on USB bus 2, port 3 at
5 Gbit/s through the kernel `usb-storage` driver (`/dev/sda` via SCSI), rather
than UAS. This identifies the physical link/bridge for a host-owner cable/port
inspection; it does not prove which component caused the SCSI error. Fresh
DBM plan preflight at 04:16Z reports exactly one grouped
`repository_invalid` incident for all 14 targets because destination usage
cannot be measured. Source traversal/capacity do not appear as incidents. The
repository EIO and NTFS dirty-state UNKNOWN remain the blockers; no backup or
repair ran.

## 2026-11-04 04:22Z — broker deployment and restore-topology audit

The first authenticated setup update did land: `/usr/local/lib/vrooli/vrooli-privilege-broker`
was written at 03:57Z and its live root process (PID 1828097) exposes the SMART
capability. The compact serializer fix was built/installed afterward: the
current user CLI is timestamped 04:10Z and has a different binary digest from
the root broker. This confirms why the broker still returns the old 8-KiB raw
prefix and why the supported broker setup must be run again from an authenticated
terminal.

DBM owner reads confirm Treasury target `bba8b1d6-bfdf-4c21-ab48-177411eaa276`
has a verified scratch-restore record (`56e32b5b-8da5-472e-85cb-ea2f5c4207db`),
but both its source `/home/matthalloran8/.vrooli/data/vrooli/treasury` and
`baseline-safety` destination are on `/dev/nvme0n1p2`. This is valid same-host
verification evidence, not an independent/off-host restore. Elements remains
blocked on repository usage EIO; no current off-host restore evidence exists.

## 2026-11-04 04:23Z — no broker deployment or repository-state change

The live root broker binary and process remain the 03:57Z build (PID 1828097);
its digest still differs from the 04:10Z user CLI containing the bounded SMART
serializer. Fresh DBM preflight at 04:23Z again reports only destination
repository-usage `repository_invalid` for all 14 Elements targets, with last
known-good usage at 03:17:42Z. Capacity/source enumeration remains clear.
Destination usage still fails on the exact preserved temp with EIO. No
independent host or repository action is available without an authenticated
broker refresh or host-owner transport inspection; no disk/repository write
was performed.

## 2026-11-04 04:32Z — SMART decoder fixed; broker refresh still required

The operator ran `vrooli setup broker --sudo-mode ask` from an authenticated
terminal, and the root broker now exposes `volume.smart.inspect`. The first
SMART request through that broker exposed a decoder mismatch: this host's
smartctl encodes `local_time` as an object (`time_t` and `asctime`), while the
broker expected a string. Updated the control-plane decoder to accept either
representation, keep malformed-output details bounded, and retain health,
attribute, error-log and self-test summaries within the response size limit.
The focused broker, host-handler and volume-remediation packages compile, and
`make install` completed. No tests were run.

The owner source and user CLI were updated at 23:30 local time, but the active
root broker binary still dates from the preceding 23:28 setup. Run
`vrooli setup broker --sudo-mode ask` again from the authenticated terminal to
deploy the parser fix; then rerun the already-approved serial-bound read-only
SMART query once and inspect the complete bounded summary before any further
diagnostic probe. The Elements EIO remains preserved, plan preflight still
gates all 14 targets, and no backup, restore, prune or filesystem repair ran.

The broker-refresh update was sent through Notification Hub as `a111d74c-9d87-4078-ba84-ee956700c312` and reached Web Push receipt `84b57a8b-68c7-4bec-8f5c-5fbedf228762` at 04:35:53Z. It requests the setup refresh and asks the operator to tell the supervisor when complete.

## 2026-11-04 04:44Z — full SMART summary and transport check

The operator refreshed the root privilege broker. Its live binary is now newer
than the corrected `internal/privilegebroker/smart.go`, and the already-approved
identity-bound SMART action returned verified evidence. SMART health passed at
35°C. Reallocated sectors (ID 5), reallocation events (196), pending sectors
(197), offline uncorrectable sectors (198), and UDMA CRC errors (199) all have
raw count zero; all 17 returned ATA attributes fit the bounded summary.

Read-only transport observations identify a Western Digital Elements 2620 USB
mass-storage device at 5 Gbit/s through `usb-storage`/SCSI. Device state is
`running`, not blocked, with zero timeouts. `ioerr_cnt` remained `0x1799` while
request/completion counts advanced from `0x80bf3` to `0x80cc5` (286 requests);
no matching storage/USB kernel-log record appeared during the SMART window.
This does not explain or rule out the earlier intermittent failure. Fresh DBM
preflight at 04:43:17Z still has one `repository_invalid` destination-usage
incident covering all 14 targets; the resource-owned usage query reproduces
`EIO` on the exact preserved temp object. Readiness still reports NTFS
filesystem state UNKNOWN. No mount/unmount, SMART retry, backup, restore,
prune, repository write or repair ran in this turn.

The remaining useful diagnosis requires the operator to safely eject the drive
through the OS and choose either a read-only check on another authorized
computer or a same-host cable/direct-port isolation check. That decision was sent as Notification Hub ask
`3c351a99-050f-4e29-b8c7-b396c179eaf2`, which includes the Hub `Other` answer
and recommends an alternate-host test. Web Push receipt
`b22de504-c4f5-43d8-b9ce-49b096631504` confirms delivery at 04:46:54Z. The ask
is pending until the active commission deadline. The plan remains disabled.
