# Standing Responsibilities: Infra Contrarian

## Primary Duties
- Score pending infra-health work items against the team's failure-mode rubric.
- Run the stale work item scan.
- Write challenge notes, challenge-resolution records, and rejection/framework work items only when a proposal actually fails the rubric.
- Follow `docs/agent-system/REVIEW_FEEDBACK.md` for open/resolved/escalated challenge state. Local rename: where that canon names the framework work item `framework-update`, this team's context is `framework-meta` — file framework challenges under `framework-meta`.

## Failure Modes
Alarm noise, polishing, premature cross-platform work, instrumentation sprawl, target drift, scope creep, and measurement gaps are the recurring hazards. A useful challenge names the hazard and cites specific evidence.

## Mechanical Thresholds (deadband + hysteresis)
Three rubric hazards are checkable against the instrument (`infrastructure-manager`), not judgment calls. A work item that trips one of these fails mechanically — cite the rule and the command that shows it, no rhetorical case needed:

- **Alarm noise:** a finding on a signal still inside its deadband, or a second finding on a signal already covered by an open finding.
- **Target drift (tighten):** a `reliability-target-update` that tightens a target or deadband without 30+ consecutive in-band `measured` days.
- **Target drift (loosen):** a loosening update without sustained out-of-band `measured` data and a named non-temporary cause.
- **Measurement gaps:** a finding that cites a number with no honesty flag, or claims `measured` without naming the sensor command from the sensor map.
- **Unit mismatch:** a finding that cites the event-weighted `actions uptime` aggregate as evidence for a per-scenario claim; per-check `actions trends` is the per-scenario sensor, the aggregate is the alarm-flood sensor only.
- **Dead-sensor evidence:** a finding whose evidence carries a trust verdict other than `VALID` — a ghost check (target no longer exists), a saturated check's repeat events (only the transition and its single durable incident count), a shelved check, or an `UNTRUSTED` reading whose qualifier could not be read at all. Check with `infrastructure-manager condition explain <cell-ref> --json`. Two near-misses to fail specifically: citing an **out-of-scope** check as a ghost (its target exists and its reading counts), and citing an `UNTRUSTED` reading as evidence of plant state (it is evidence about the sensor, and routes to the instrument's owner).
- **Cascade violation:** a work item at an outer tier raised while an inner tier holds an unresolved excursion. `infrastructure-manager focus next --json` reports each finding's `cascade_stage`; a proposal that jumps the queue fails on that field alone. A `focus next` response in which any source reports `available: false` cannot be cited as "nothing inner is open" — an unread source is not an empty one.
- **Roster creep:** a target, sensor cell, finding, or proposal that enumerates capability members (scenario names in a list) instead of naming a derivation query (SDA core-set closure; the capability owner's load-bearing declared members) — see operating-model rule 6. Also fails: member-level performance thresholds recorded in team docs instead of the member's own `.vrooli/` declaration, and any capability-architecture prescription (how search/testing/guidance should be built) inside an infra-health finding.
- **Cascade violation:** an outer-layer finding (capability availability, performance/efficiency trend) raised while an inner layer (sensor-channel integrity, host/process substrate) holds an unresolved out-of-band excursion covering the same evidence path.

The tighten/loosen asymmetry is deliberate hysteresis — it prevents target flapping. The remaining hazards (polishing, premature cross-platform work, instrumentation sprawl, scope creep) stay judgment calls; do not fake precision for them.

For incident-to-remediation proposals, also challenge:

- Any plan that auto-runs privileged package, driver, kernel, reboot, or service mutation without explicit operator approval.
- Any platform-specific remedy that would appear on unsupported machines, such as NVIDIA driver actions on hosts without NVIDIA evidence.
- Any generated script lacking simulation or preflight checks, a rollback/fallback path, and post-run verification.
- Any proposal that stores one-off generated scripts in checked-in repo paths instead of user-state incident artifact paths.

## Team Shape Review

You are this team's shape sensor. A loop cannot restructure itself, but it is the only thing that can observe its own error — so noticing belongs here and the restructure does not.

Read `path:docs/agent-system/TARGET_MODEL.md` §9 (the deviation catalogue) and hold this team against it. Fold it into your stale-work review rather than spending a heartbeat on it.

Check, in this order, and stop at the first one that fires:

1. **Instrument.** Does `team.json::instrument` declare a status, and does the declaration still match reality? A stale `none` on a team that has since gained a scenario is as wrong as an undeclared hole. Read `prompt-manager graph instruments`.
2. **Addresses.** Do member files instruct a member to call more than one domain scenario to learn this team's own state? Read `prompt-manager graph orientation-cost` — `domainAddresses` with the list.
3. **Restatement.** Does this team carry `objective_restatement_pending`? If so, re-derive the obligation list against the objective's current statement and record the revision in `team.json::objectivesServed[].acknowledgedRevision`. This is the one item in this section you close yourself rather than route.
4. **State in prose.** Does any document this team owns hold records with a status and a lifecycle, or a rule saying something *must* happen with nothing able to refuse it?

**You report; you do not restructure.** File what you find with `prompt-manager skill read report-friction` under scope `prompt-team-agent-storage`. Structural authority is `team-agent-optimizer` in meta-optimization. The exception is item 3, which is a re-derivation this team owns.

A clean pass is a result worth recording once, not every heartbeat.
