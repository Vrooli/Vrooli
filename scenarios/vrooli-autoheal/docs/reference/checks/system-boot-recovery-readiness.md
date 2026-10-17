# Boot Recovery Readiness (system-boot-recovery-readiness)

Proves, while the host is healthy, that the autoheal boot path would work if the host rebooted now. It is observation-only: every failed precondition is repaired by `vrooli setup`, and the check offers no recovery actions because the scenario cannot perform any of them.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-boot-recovery-readiness` |
| Category | System |
| Interval | 3600 seconds (1 hour) |
| Platforms | All (lingering applies to Linux only) |
| Importance | Critical |
| Recovery actions | None |

## Why It Exists

Every earlier boot-protection surface answered "is the unit installed". None answered "would it start". On 2026-09-02 a supervisor unit rendered two weeks earlier crash-looped 495 times after boot because its argv no longer parsed, while every surface read green. A boot path that is only tested by rebooting fails at the one moment nobody is watching.

## Preconditions

Each precondition reports `ok`, `failed` or `undetermined` with a reason. They are listed in `details.preconditions` in this order.

| Name | Source | ok when | failed when | undetermined when |
|------|--------|---------|-------------|-------------------|
| `safeguards` | `vrooli setup status --json --phase readiness` through the shared CLI invoker (15 s budget) | `autoheal_watchdog`, `runtime_supervisor` and `emergency_watchdog` are all `applied` | any is missing or not applied (the last note is quoted) | the command cannot run, times out, or its output is not JSON |
| `loop-preflight` | the loop binary's `--self-test` (8 s budget) | the preflight reports `ok` with no failed check | any preflight check failed (each is named with its reason) | the binary cannot run or prints unparseable output |
| `unit-active` | the service manager, per core daemon unit from `platformgo.CoreUnits()` | every unit is `active`, `NRestarts` is `0`, `Result` is not `start-limit-hit` | a unit is not active, hit its start limit, or has restarted | a unit cannot be read |
| `loop-heartbeat` | `~/.vrooli/state/vrooli-autoheal/loop-status.json` and the loop binary on disk | `last_tick_at` is within 3 minutes and the running `binary_sha256` matches the on-disk binary | the loop never ticked, the tick is older than 3 minutes, or the running binary differs from the one on disk | the status file or the binary is unreadable |
| `lingering` | `loginctl show-user` (Linux, only when the `autoheal_watchdog` safeguard's `boot_policy` is `dedicated`) | lingering is enabled, or the policy does not require it | the policy is dedicated and lingering is off | the policy or the invoking user is unknown |
| `validator` | the `validator_verdict` each safeguard recorded | every verdict is `accepted` | any verdict is `rejected` (the validator output is quoted) | any verdict is `unavailable` or missing: unproven is never accepted |
| `containment` | the `agent_session_containment` safeguard in the same setup status report | the agent slice is applied with an accepted validator verdict | the safeguard is not applied or its verdict is not accepted | the safeguard could not inspect the slice, or is absent from the readiness phase |

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Every precondition holds |
| **Critical** | At least one precondition failed; the message names it and says `run vrooli setup` |
| **Undetermined** | No precondition failed but at least one could not be probed |

A failed precondition always outranks an undetermined one. Phase 11's nightly rehearsal reads this check and never lets a passing rehearsal override a failed precondition.

## Details Returned

```json
{
  "remediation": "vrooli setup",
  "evaluatedAt": "2026-09-02T18:30:00Z",
  "preconditions": [
    {"name": "safeguards", "state": "ok", "reason": "autoheal_watchdog, runtime_supervisor, emergency_watchdog applied"},
    {"name": "loop-preflight", "state": "ok", "reason": "6 preflight checks passed"},
    {"name": "unit-active", "state": "ok", "reason": "vrooli-autoheal.service, vrooli-runtime-supervisor.service active with zero restarts"},
    {"name": "loop-heartbeat", "state": "ok", "reason": "last tick 40s ago, running the on-disk binary"},
    {"name": "lingering", "state": "ok", "reason": "lingering enabled for matthalloran8 (dedicated policy)"},
    {"name": "validator", "state": "ok", "reason": "native validator accepted autoheal_watchdog, runtime_supervisor, emergency_watchdog"},
    {"name": "containment", "state": "ok", "reason": "vrooli-agents.slice applied with an accepted validator verdict; 0 live agent scope(s)"}
  ],
  "failedPreconditions": [],
  "undeterminedPreconditions": [],
  "findingKey": ""
}
```

`findingKey` is the comma-joined failed set, so the same broken precondition stays one incident across hourly runs.

## Where Else It Surfaces

- `vrooli setup status` prints a "Boot recovery" block from the autoheal API's typed readiness (`GetReadiness` → `boot_recovery`), or `boot recovery: unknown (autoheal API not reachable)` when the API does not answer.
- `GET /api/v1/checks/system-boot-recovery-readiness` returns the last result.

## Related Checks

- **os-watchdog**: observes the unit itself (installed, enabled, active, loop process present)
- **system-stale-service-binary**: a core unit that is not running is critical there every 5 minutes; this check confirms it hourly with the boot-path context
- **system-emergency-watchdog-report**: the last line of defense's findings

---

*Back to [Check Catalog](../check-catalog.md)*
