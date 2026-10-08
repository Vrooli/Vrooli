# Host pressure: the floor, the slice, the setpoint, the authority

On 2026-09-02 three agent sessions built the repository at once on a 32-core
host and drove the 15-minute load past 1,499. Nothing could stop it: builds
had no width cap, agent sessions had no ceiling, their slice was excluded
from systemd-oomd, and the watchdog's findings died in a journal. This page
names the four layers that now stand between a session and the host, and
where each one lives.

## Layer A: the toolchain floor

Every toolchain spawn inherits a bounded width from `envkit.Toolchain`
(`packages/envkit-go/toolchain.go`): `-p=<width>` appended to an inherited
`GOFLAGS` only when no `-p` token is present, `GOMAXPROCS` and pnpm's
concurrency settings only when absent. The width is the `BuildWidth` tuning
lever (`min(4, max(1, NumCPU/4))`, `VROOLI_TUNING_BUILD_WIDTH`). The floor is
applied at the lifecycle step environment, `internal/shell.Command`, both CLI
installer paths, the agent launcher, and every scenario site that spawns a
toolchain; `mk/toolchain.mk` carries it into every Makefile and CI. The
ast-grep rule `no-raw-toolchain-spawn` admits no allowlist entries, and the
root Makefile refuses `./...` as a build goal. See `build-and-validation.md`.

## Layer B: contained sessions

`platformgo.Containment` (`packages/platform-go/servicedef.go`) is the ceiling
vocabulary: CPU share, memory high and max, task ceiling, parent slice. The
`agent_session_containment` safeguard converges `vrooli-agents.slice` under
the user manager with typed operator config (see
`configuration/host/safeguards.md`), and every coding-agent session is born
inside a scope under it: `systemd-run --scope` (Linux, with a cgroup-write
fallback), an rlimit shim (macOS), a Job Object with quotas (Windows).
The exec-replace branch moves the launcher itself into the scope before
`exec`. Supervisors keep `CPUWeight=400`, `MemoryMin=128M`,
`OOMScoreAdjust=-500`; the slice's `ManagedOOMMemoryPressure=kill` lets
systemd-oomd act on agents first.

| Platform | Evidence tier | What holds |
| --- | --- | --- |
| Linux | host-verified | slice limits, scope placement, `TasksMax` stops a fork bomb, `cgroup.freeze` |
| macOS | fixture-verified | `RLIMIT_NPROC`/`RLIMIT_AS` per process through the shim; `Nice` and `SoftResourceLimits` in launchd plists |
| Windows | fixture-verified | Job Object memory, active-process and CPU-rate limits; task priority |

## Layer C: one setpoint, attributed pressure, one authority

`internal/setpoint` is the only parser of
`scenarios/infrastructure-manager/setpoint/reliability-setpoint.json`. Every
bar carries its threshold, unit and authored sustain; two bars cannot share a
`cell_ref`; the emergency watchdog, autoheal's `system-host-pressure` check
and the runtime supervisor's pressure gate all read it. A breach is a warning
until the authored sustain has elapsed, then critical; no consumer shortens
the window in code (`docs/infra-health/governance/editing.md`).

`hostpressure.Attribution` folds the process list the watchdog already
collects into the top parents by child count and by delta. The watchdog
writes its whole report to `~/.vrooli/state/emergency-watchdog/last-report.json`
on every run; autoheal's `system-emergency-watchdog-report` check reads it,
opens one incident per finding, and titles a fork storm by its parent.

The authority is one gated action: `contain-storm` freezes the attributed
session scope, and only that. It refuses any pid outside a `vrooli-agent-*`
scope under `vrooli-agents.slice`, refuses when the runtime recovery gate is
closed, records a `runtime_recovery_decisions` row, and is reversed by
`vrooli agent thaw`. The watchdog senses; autoheal decides; nothing else
kills. `contain_storm: propose_only` in autoheal's operator config keeps the
action available to the operator and never auto-runs it.

Delivery: `coverage-delivery-reach` counts an incident as reached only when
notification-hub recorded a `delivered` attempt (see
`configuration/notifications.md` for the recipient and channel settings).

## Layer D: visible editors

Every session records an editor lease (tree, scope, pid, claims) that expires
only on proof of death; `vrooli agent list` shows them, `vrooli status`
counts them, `--claim` names an overlapping holder and continues. See
`agent-sessions.md`.

## Commands

```bash
vrooli tuning list | grep BuildWidth              # the width lever
systemctl --user show vrooli-agents.slice -p MemoryMax -p TasksMax -p CPUWeight
vrooli setup status --json --phase readiness      # agent_session_containment applied?
vrooli agent list                                 # live sessions
vrooli agent thaw <session-or-scope>              # reverse a freeze
vrooli-autoheal storm status                      # frozen scopes and decisions
cat ~/.vrooli/state/emergency-watchdog/last-report.json | jq .attribution
```
