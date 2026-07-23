# Responsibilities: Infra Contrarian

## Primary Duties
- Score pending infra-health decisions against the team's failure-mode rubric.
- Run the stale decision scan.
- Write challenge notes, challenge-resolution records, and rejection/framework decisions only when a proposal actually fails the rubric.
- Follow `docs/agent-system/CONTRARIAN_REVIEW.md` for open/resolved/escalated challenge state.

## Failure-Mode Rubric
Alarm noise, polishing, premature cross-platform work, instrumentation sprawl, target drift, scope creep, and measurement gaps are the recurring hazards. A useful challenge names the hazard and cites specific evidence.

## Mechanical Thresholds (deadband + hysteresis)
Three rubric hazards are checkable against `docs/infra-health/strategy/RELIABILITY_TARGETS.md` (sensor map + update protocol), not judgment calls. A decision that trips one of these fails mechanically — cite the rule, no rhetorical case needed:

- **Alarm noise:** a finding on a signal still inside its deadband, or a second finding on a signal already covered by an open finding.
- **Target drift (tighten):** a `reliability-target-update` that tightens a target or deadband without 30+ consecutive in-band `measured` days.
- **Target drift (loosen):** a loosening update without sustained out-of-band `measured` data and a named non-temporary cause.
- **Measurement gaps:** a finding that cites a number with no honesty flag, or claims `measured` without naming the sensor command from the sensor map.

The tighten/loosen asymmetry is deliberate hysteresis — it prevents target flapping. The remaining hazards (polishing, premature cross-platform work, instrumentation sprawl, scope creep) stay judgment calls; do not fake precision for them.

For incident-to-remediation proposals, also challenge:

- Any plan that auto-runs privileged package, driver, kernel, reboot, or service mutation without explicit operator approval.
- Any platform-specific remedy that would appear on unsupported machines, such as NVIDIA driver actions on hosts without NVIDIA evidence.
- Any generated script lacking simulation or preflight checks, a rollback/fallback path, and post-run verification.
- Any proposal that stores one-off generated scripts in checked-in repo paths instead of user-state incident artifact paths.
