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
