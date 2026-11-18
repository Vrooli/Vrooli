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
- Native filesystem state `unknown` never becomes `clean` in evidence. A
  degraded write requires all three: explicit degraded policy, a passed bounded
  write/read/fsync probe, and a separate verified copy for every selected
  target. Any probe failure blocks even when native state claims clean.
- The write probe leaves no artifact on success or failure, and a read-only
  readiness command never runs it.
- Preflight owns the degraded-write decision and the executor consumes it.
  The write gate must not re-derive readiness with different inputs; otherwise
  preflight can admit a write the gate then silently blocks.
- Redundancy is per target: a destination may write degraded only when every
  target that will write has a verified copy on a different proven device. A
  second destination on the same device is not another copy. A legacy
  destination without persisted identity may prove independence from the
  current mounted filesystem, but only for the run making the decision.
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
