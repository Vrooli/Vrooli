# Reliability invariants

- Critical-primary and critical-secondary plans fail closed unless every
  selected target is explicitly classified critical.
- Critical-secondary requires at least two non-overlapping destination roots;
  a destination that overlaps a critical source is rejected.
- Plan reads expose `destinations_physically_independent` and
  `shared_risk_warnings`; unknown volume identity is never rendered as proven
  physical independence.
- A recovery drill selects only a successful persisted snapshot and invokes
  verified restore into scratch. It never restores over the live target.
- A drill failure cannot become `verified`; the linked restore record is the
  source of truth for repository-readability evidence.
- Scratch cleanup is part of the terminal evidence boundary: a restore, verify,
  or audit whose temporary tree cannot be removed is recorded as failed with an
  actionable cleanup error, never silently published as healthy.
- Drill requests with the same idempotency key return one persisted record;
  active target×destination units are not started twice.
- Scheduled primary and secondary drill units are evaluated independently;
  failure of one destination does not suppress another destination's attempt.
- Plaintext credential values and restored content are not stored in drill,
  target, plan, or destination evidence.
- Snapshot identity is never inferred from the whole repository. A snapshot
  belongs to a target only when its exact `dbm.owner`/`dbm.name` tags match, or
  when a legacy snapshot's source path ends in `:/<owner>/<name>`. Unrelated
  snapshots are filtered, never treated as an integrity failure.
- Capacity admission and retention forecasting are separate. The immediate
  write uses a first-copy estimate or measured incremental growth plus a
  reserve; the number of GFS tiers is a snapshot count, never a byte
  multiplier. Advisory policy warns on the forecast; strict policy gates on it.
- A read-only repository operation (snapshot listing, usage, browse) proves
  device identity, mount, bundle identity, path containment, and repository
  readability. It does not require a clean or writable filesystem. A write
  requires the full write verdict.
- Native filesystem state `unknown` never becomes `clean` in evidence. It does
  not block either: an unobservable state is not evidence of corruption. A
  mounted, writable destination whose bounded write/read/fsync probe passes is
  admitted as a **degraded write with a warning**. A dirty or needs-check state,
  a failed probe, a read-only mount, or write protection still blocks.
- Copy count is never a write-admission input. How many devices hold a copy is a
  drive-loss risk reported by coverage; blocking a backup because it is the only
  copy would leave the data with fewer copies, not more.
- The write probe leaves no artifact on success or failure, and a read-only
  readiness command never runs it.
- One pure `EvaluateWriteVerdict` decides every write, and preflight, the write
  gate, and retention all consume it. There is no caller-supplied degraded
  policy, so preflight cannot admit a write the gate then silently blocks, and a
  destination that accepted a snapshot cannot refuse to prune it.
- A second destination on the same device is not another copy, but that is a
  reporting fact owned by coverage's device classification, not a per-run gate.
- Two destinations may not share one bundle root or overlapping repository
  paths. A bundle nested inside another destination's bundle root is allowed
  because the manifests and repository directories remain distinct.
- Repository probes in preflight, health, usage, and snapshot listing use the
  read path. Only an actual snapshot write uses the write gate.
- The host-volume lease handshake is bounded, and releasing or cancelling the
  lease kills the whole lease process group. A lease holder must never outlive
  the operation that acquired it; an orphan holds an exclusive flock that
  blocks every later write on the volume.
- Retention and any other repository mutation use the same write policy as the
  snapshot that just landed. A degraded destination must not accept a snapshot
  and then refuse to prune it.
- Abandoned per-run staging directories older than `StaleStagingAge` are removed
  at startup. Staging may hold a full source copy, so it is never left behind
  indefinitely.
- Protection is counted by **distinct storage device**, not by destination rows.
  Two or more devices with a successful copy is protected. One device is a
  single-copy warning, never an issue. Zero successful copies is the issue.
  Separate devices on the same host count as protection; off-host is a stronger
  level, not the entry condition.
- Filesystem check/repair behaviour, state-signal availability, suitability, and
  the operator remedy live once in `packages/driveprofile`. Readiness and the
  control plane must read it; neither may invent a filesystem switch of its own.
- A DBM bundle may share a drive folder with unrelated data, but DBM never
  deletes foreign files. A managed name (`README.txt`, `RECOVERY.txt`, the
  manifest, `repositories/`) occupied by an incompatible type is refused rather
  than overwritten.
- A repair is offered only when the platform's tool can resolve the diagnosed
  cause. On Linux, NTFS repair is a dirty-flag reset: it may be offered for a
  read-only mount, never for unreadable metadata. The system must not run a
  repair it cannot perform and must never report such a run as "repaired".
- A new filesystem destination whose filesystem this host cannot repair (for
  example NTFS on Linux) is refused unless the operator explicitly acknowledges
  it. A drive that is safe until the first unsafe shutdown is not admitted
  silently.
- On a Linux-only host the backup filesystem is ext4. The migration path keeps
  the bundle path and destination id, and re-binds only the volume identity, so
  plans do not have to be rebuilt.
- Automatic exception recording is limited to EIO-unreadable
  `leftover_temp`/session-marker entries. Snapshot data, manifests, and any
  ambiguous entry still require a verified decision. Destructive filesystem
  repair still requires verified human approval until a standing grant exists.
