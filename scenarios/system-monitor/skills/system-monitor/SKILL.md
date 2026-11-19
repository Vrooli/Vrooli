---
name: "system-monitor"
description: "Check how this machine is doing: CPU, memory, disk and GPU usage, which processes or scenarios are consuming resources, active alerts, and recent history. Run or read a host investigation, generate a daily or weekly host report, and inspect GPU capacity claims."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["system-monitor", "cpu", "memory", "disk", "gpu", "host-health", "resource-usage", "processes", "alerts", "monitoring"]
  requires:
    scenarios: ["system-monitor"]
    commands: ["system-monitor metrics current", "system-monitor overview alerts"]
  origin:
    kind: "authored"
---

## System Monitor

Read the host's live state before guessing at it. Every command below is read-only
unless marked.

| You want | Command |
|---|---|
| Is the monitor itself healthy | `system-monitor overview health` |
| CPU, memory, disk right now | `system-monitor metrics current` (`metrics detailed` for more) |
| What is using CPU or swap | `system-monitor metrics processes`; over a window, grouped by owner: `metrics process-timeline` (`--rank swap` to surface the largest measured swap consumers with coverage counts) |
| Recent history | `system-monitor metrics timeline` |
| Active alerts | `system-monitor overview alerts` |
| GPU contention and who holds it | `system-monitor capacity overview`, `capacity claims` |
| Why was the host slow | `system-monitor investigations latest` or `investigations list`; start one with `investigations trigger` (writes; has a cooldown, see `investigations cooldown`) |
| Daily report | `system-monitor reports generate daily`, then `reports get` |
| Weekly per-owner host-headroom report | `program-runtime library run system-monitor.owner-headroom-weekly` (writes one weekly report and returns bounded owner resource rows plus actionable follow-ups when attribution or swap coverage is partial) |

Run `system-monitor <group> --help` for flags. Do not change settings,
maintenance or capacity policy (`settings`, `maintenance`, `capacity policy set`)
from this skill; those change host behaviour and belong to the operator.

Timeline owners may be scenario/resource names or explicit host identities such
as `desktop-app:chrome`, `agent-scope:vrooli-agent-codex-…`,
`systemd-unit:system.slice/dbus.service`, and `vrooli-service:…`. These labels
make process groups traceable; they do not authorize stopping a process. Resolve
an agent scope through `vrooli agent list` and use its owner/approval path before
preparing any reclaim decision.
