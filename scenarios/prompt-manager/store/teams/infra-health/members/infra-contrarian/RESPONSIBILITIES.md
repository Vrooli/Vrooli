# Standing Responsibilities: Infra Contrarian

## Primary Duties
- Score pending infra-health work items against the team's failure-mode rubric.
- Run the stale work item scan.
- Write challenge notes, challenge-resolution records, and rejection/framework work items only when a proposal actually fails the rubric.
- Follow `docs/agent-system/REVIEW_FEEDBACK.md` for open/resolved/escalated challenge state. Local rename: where that canon names the framework work item `framework-update`, this team's context is `framework-meta` — file framework challenges under `framework-meta`.

## Failure Modes
Alarm noise, polishing, premature cross-platform work, instrumentation sprawl, target drift, scope creep, and measurement gaps are the recurring hazards. A useful challenge names the hazard and cites specific evidence.

## Mechanical Thresholds (deadband + hysteresis)
Three rubric hazards are checkable against `docs/infra-health/strategy/RELIABILITY_TARGETS.md` (sensor map + update protocol), not judgment calls. A work item that trips one of these fails mechanically — cite the rule, no rhetorical case needed:

- **Alarm noise:** a finding on a signal still inside its deadband, or a second finding on a signal already covered by an open finding.
- **Target drift (tighten):** a `reliability-target-update` that tightens a target or deadband without 30+ consecutive in-band `measured` days.
- **Target drift (loosen):** a loosening update without sustained out-of-band `measured` data and a named non-temporary cause.
- **Measurement gaps:** a finding that cites a number with no honesty flag, or claims `measured` without naming the sensor command from the sensor map.
- **Unit mismatch:** a finding that cites the event-weighted `actions uptime` aggregate as evidence for a per-scenario claim; per-check `actions trends` is the per-scenario sensor, the aggregate is the alarm-flood sensor only.
- **Dead-sensor evidence:** a finding whose evidence comes from a ghost check (target no longer exists), from a saturated check's repeat events (only the transition and its single durable incident count), or from a shelved check (see `RELIABILITY_TARGETS.md` § Sensor integrity).
- **Roster creep:** a target, sensor cell, finding, or proposal that enumerates capability members (scenario names in a list) instead of naming a derivation query (SDA core-set closure; the capability owner's load-bearing declared members) — see operating-model rule 6. Also fails: member-level performance thresholds recorded in team docs instead of the member's own `.vrooli/` declaration, and any capability-architecture prescription (how search/testing/guidance should be built) inside an infra-health finding.
- **Cascade violation:** an outer-layer finding (capability availability, performance/efficiency trend) raised while an inner layer (sensor-channel integrity, host/process substrate) holds an unresolved out-of-band excursion covering the same evidence path.

The tighten/loosen asymmetry is deliberate hysteresis — it prevents target flapping. The remaining hazards (polishing, premature cross-platform work, instrumentation sprawl, scope creep) stay judgment calls; do not fake precision for them.

For incident-to-remediation proposals, also challenge:

- Any plan that auto-runs privileged package, driver, kernel, reboot, or service mutation without explicit operator approval.
- Any platform-specific remedy that would appear on unsupported machines, such as NVIDIA driver actions on hosts without NVIDIA evidence.
- Any generated script lacking simulation or preflight checks, a rollback/fallback path, and post-run verification.
- Any proposal that stores one-off generated scripts in checked-in repo paths instead of user-state incident artifact paths.
