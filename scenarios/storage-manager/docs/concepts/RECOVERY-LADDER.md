# Recovery ladder

Storage-manager recovery is class-based. The controller measures and applies
one bounded batch at a time, while the host recovery lock prevents concurrent
deleters.

| Rung | Authority | Autonomous contents |
|---|---|---|
| R0 | `class` | `safe` roots such as temporary work and agent scratch |
| R1 | `class` + proof | `regenerable` roots whose bytes are derived, recreatable, exactly contained, and lease-free |
| R2 | `owner_budget` | `safe_with_owner` entries with `regenerable: true` and a declared retention budget |
| operator line / R3 | `standing_approval` | conditional providers explicitly approved for this host |
| forbidden | never | providers marked `forbidden` |

The policy profile controls ordinary retention age and enablement. It does not
change the autonomous authority boundary. A recovery request may override the
ordinary approval mode only for a provider admitted by the current rung.

Each action is bounded, re-stat'ed before deletion, and recorded with the run
id, provider, rung, bytes reclaimed, and free-space measurements. Recovery
stops when the free-space target is met, the rung budget is exhausted, the
operator line is reached, or the run is interrupted.
