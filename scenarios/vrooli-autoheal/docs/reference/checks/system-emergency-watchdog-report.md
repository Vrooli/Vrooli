# Emergency Watchdog Report (system-emergency-watchdog-report)

Reads the emergency watchdog's last report and turns each finding into an incident. The watchdog senses; autoheal decides.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `system-emergency-watchdog-report` |
| Category | System |
| Interval | 60 seconds |
| Platforms | All |
| Recovery actions | None |

## What It Reads

`~/.vrooli/state/emergency-watchdog/last-report.json`, which the `vrooli-watchdog` binary writes atomically on every timer run (every 5 minutes), whether or not it found anything, so a reader can tell "no findings" from "no run". The file is the watchdog's whole report: readings, findings, evidence, the fork-rate attribution, the down-unit list, and any actions taken.

Before 2026-09-02 the watchdog printed its report to stdout, where only the journal saw it, and its unit-restart escalation lived in a shell script that nothing installed. The report file is the sink; the escalation now lives in the binary behind `--reclaim`.

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | The report is fresh and carries no findings |
| **Critical** | The report carries at least one finding (for example `unit-liveness`, `disk-space`, `fork-rate`, `cpu-pressure`); each finding is copied to `details.findings` with its evidence and, for fork-rate and CPU pressure, the attributed parent process |
| **Warning** (`reportState: undetermined`) | The report is missing, unreadable, carries no `captured_at`, or is older than three timer intervals (15 minutes) |

## Details Returned

```json
{
  "reportPath": "/home/user/.vrooli/state/emergency-watchdog/last-report.json",
  "reportState": "read",
  "reportAge": "2m10s",
  "capturedAt": "2026-09-02T18:40:02Z",
  "findings": [
    {"name": "unit-liveness", "reason": "declared units are not active: vrooli-runtime-supervisor.service (systemctl --user is-active inactive)", "evidence": ["declared units: vrooli-autoheal.service, vrooli-runtime-supervisor.service"]}
  ]
}
```

The incident service fingerprints each finding by name, so a finding that persists across runs is one incident and a new finding is a new one.

## Related Checks

- **system-stale-service-binary**: names a dead core unit within 5 minutes, before the watchdog's 10-minute liveness sustain elapses
- **system-boot-recovery-readiness**: the hourly proof that the boot path works
- **system-host-pressure**: the same pressure bars, read in-process

---

*Back to [Check Catalog](../check-catalog.md)*
