# Substrate Space — Host, Kernel And Device Condition

## This Space

| | |
|---|---|
| Projection | substrate |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `PARTIAL` — eleven of nineteen cells are NOW, seven are IN-REACH, and SB6 remains the one open-loop crash-accounting gap; SB17 is joined by workload reconciliation |
| Leg unit | host subsystem |

## Why This Projection Exists

The infra-health cascade ladder orders findings inner-to-outer: sensor-channel
integrity, **host/process substrate**, capability availability, efficiency, then
measurement improvement. The substrate tier had no projection, so the one layer
the ladder says to resolve *first* was the only layer the instrument could not
read. Host and device faults were reachable solely through the availability
projection, where every check is expressed as an uptime percentage — a unit in
which a GPU falling off the bus, a machine-check storm and a slow scenario
restart are indistinguishable.

Substrate readings are therefore **ordered severities**, not percentages:
`0` = OK, `1` = WARNING, `2` = CRITICAL. A cell backed by more than one check
takes the worst severity among them and names the contributing check, so a
verdict points at a signal rather than at a projection. A check reporting
`NOT_APPLICABLE` or an unspecified status contributes no severity and is never
read as OK.

## Coverage Grid

| ID | Question | Owner | Status | Gap opened on | Notes |
|---|---|---|---|---|---|
| SB1 | Is the host free of unresolved kernel and device error signals? | vrooli-autoheal | NOW | | `host-kernel-error-signals` classifies Xid, MCE, AER/PCIe, DRM and I/O signals from the kernel log. |
| SB2 | Is crash evidence from the last boot captured and readable? | vrooli-autoheal | NOW | | `system-panic-evidence` and `system-pstore-evidence`. Reads kdump and pstore; a coverage gap here means a panic can occur without leaving readable evidence. |
| SB3 | Is machine-check telemetry readable? | vrooli-autoheal | NOW | | `system-mce-recent`. Depends on rasdaemon; an unreadable MCE channel is a blind spot, not a clean bill of health. |
| SB4 | Are accelerator devices present and usable by their claimants? | vrooli-autoheal | NOW | | `system-gpu` and `resource-gpu-access`. Device presence and claimant access are distinct failures and both belong here. |
| SB5 | Do the running kernel, module inventory and driver bindings agree? | vrooli-autoheal | NOW | | `host-kernel-module-drift`, `host-device-driver-binding`, `host-runtime-integrity`. Drift here precedes device-access failures that look like application faults. |
| SB18 | Do critical findings have a remediation path? | vrooli-autoheal | IN-REACH | 2026-08-25 | `coverage-remediation-reach` reads critical registry results and registered actions; a missing action is a critical shortfall rather than an implicit OK. |
| SB19 | Do critical findings reach a human channel? | notification-hub | IN-REACH | 2026-08-25 | `coverage-delivery-reach` joins incident IDs to durable delivery-attempt IDs. Its current host reading is explicitly unreadable until the notification-hub projection is configured. |
| SB6 | Are userspace process crashes counted and attributed to an owner? | vrooli-autoheal | MISSING | 2026-08-20 | No core-dump sensor exists anywhere in the platform. `core_pattern`, `systemd-coredump` and `coredumpctl` are unread, so a process crash loop is invisible unless it also fails a liveness check. |
| SB7 | Did the host boot cleanly? | vrooli-autoheal | NOW | | `system-boot-history`. An unclean boot re-dates every other substrate reading and is checked before them. |
| SB8 | Is the host watchdog armed and are declared host capabilities present? | vrooli-autoheal | NOW | | `os-watchdog` and `host-capability-drift`. The watchdog is the last line under everything above it, so its own liveness is part of the substrate. |
| SB9 | Is every attached device identified by vendor, model and driver rather than a bare numeric id? | vrooli-autoheal | IN-REACH | | Sensor exists: `system-monitor` device graph, `internal/devicegraph/hwids.go` and `sysfs_bus.go`. An unidentified device cannot be reasoned about — it has no owner, no expected driver and no known failure mode. |
| SB10 | Do storage devices report predictive-failure indicators before they fail? | vrooli-autoheal | IN-REACH | | Sensor exists: `internal/devicegraph/smart.go`, which separates "no SMART reader installed", "the host refused the read" and a real reading rather than reporting a blocked probe as a healthy drive. This is the only substrate cell that can warn *before* a fault. |
| SB11 | Are host temperatures readable and below their trip points? | vrooli-autoheal | IN-REACH | | Sensor exists: `internal/devicegraph/sysfs_thermal.go`, enumerating every hwmon sensor and thermal zone with its trip points. Thermal throttling presents as a performance regression with every availability signal green. |
| SB12 | Are correctable and uncorrectable memory errors counted? | vrooli-autoheal | IN-REACH | | Sensor exists: `internal/devicegraph/sysfs_edac.go`, per controller and per DIMM. It grades "no EDAC controller registered" as unmeasurable rather than as zero errors, which is the distinction this cell exists to preserve. |
| SB13 | Do hardware network interfaces carry traffic without error or drop growth? | vrooli-autoheal | IN-REACH | | Sensor exists: `internal/devicegraph/sysfs_net.go`, link state plus `rx/tx_errors`, `rx/tx_dropped`, `rx_crc_errors` and `collisions`, counting and naming virtual interfaces so their exclusion is visible rather than silent. |
| SB14 | Does sustained CPU pressure stay below the operator bar? | vrooli-autoheal | NOW | | `system-host-pressure` joins CPU PSI `some.avg10`; load average is not substituted for pressure. |
| SB15 | Do host memory and swap stay in band, with evicted services reclaimed? | vrooli-autoheal | NOW | | `system-host-pressure` joins memory and swap; stranded memory is distinct from total swap used. |
| SB16 | Do process count and process-creation rate stay below the operator bar? | vrooli-autoheal | NOW | | `system-host-pressure` joins process count and fork rate; fork rate is unread on darwin and windows by decision. |
| SB17 | Is every workload accounted for under the declared host workload posture? | vrooli-autoheal | NOW | | `system-host-pressure` joins workload ownership reporting. `host_workload_posture` changes reporting, grading, and recorded detail, never classification. |

## Reading Contract

- Severity is ordered and comparable; it is **not** a score and must not be
  averaged across cells. The instrument grades each cell against its own bar.
- A cell whose contributing checks are all unreadable is reported as an
  unreadable cell, never omitted and never defaulted to OK.
- `SB6` is open-loop by declaration. It is counted and dated by the
  instrument's open-loop surface like any other `MISSING` cell; closing it
  means shipping a crash-accounting sensor, not adding a row here.
- `SB9`-`SB13` are `IN-REACH`, not `MISSING`, and the difference is
  load-bearing. Each names a sensor that is **already shipped and emitting** —
  `system-monitor`'s device graph runs on a 60s collector and grades every
  subsystem it walks. What is missing is the join, not the instrument. Declaring
  these `MISSING` would have counted five closable gaps as honest blindness,
  which the coverage model names directly: an open-loop cell that could already
  be served is *unowned work wearing honesty's clothes*.

## The `SB9`-`SB13` Join Is Not Free

These five cells cannot be closed by adding an entry to the instrument's sensor
map, because their sensor is in a different scenario **and speaks a different
unit**. This projection reads ordered severities (`0` OK, `1` WARNING, `2`
CRITICAL) from autoheal check statuses; the device graph grades subsystems on a
rung ladder and emits them as a metric series. Closing a cell means one of:

1. `vrooli-autoheal` registers a check that wraps the device-graph subsystem and
   projects its rung state onto a severity, or
2. `system-monitor` exposes the device graph through a typed read verb and the
   instrument grows a second substrate source that maps rungs to severities.

Until one of those ships, a bar on these cells is graded against nothing. That is
why their bars route to `instrumentation-gap` rather than to the plant: the work
is instrument work, and filing it as a runtime finding would send someone to fix
a machine that is not broken.

## Cross-references

- `scenarios/infrastructure-manager/docs/concepts/COVERAGE-MODEL.md` — the projection model this space instantiates.
- `docs/infra-health/operating/OPERATING_MODEL.md` — the Platform Under Control layer map.
- `internal/hostinventory/integrity_collector.go` — the control-plane collector behind SB1, SB2 and SB5.
- `scenarios/system-monitor/api/internal/devicegraph/` — the shipped device-layer sensor behind SB9-SB13, and
  `internal/collectors/devicegraph.go`, the 60s collector that publishes it.

## Change Log

- `2026-08-22` — `SB14`-`SB17` authored by the host-pressure recovery plan;
  `SB14`-`SB16` join portable pressure readings and `SB17` joins workload
  ownership reporting. `SB17` is gradeable under `whole_host` and remains
  reporting-only for unmanaged work under `vrooli_only`; changing posture is a
  setpoint-visible event, not a silent classification change.

- `2026-08-20` — `SB9`-`SB13` added (device identity, storage predictive health,
  thermal telemetry, memory error telemetry, network interface health). Authored
  during the reliability-denominator pass after a sweep found a shipped sensor for
  each one in `system-monitor`'s device graph; all five are therefore `IN-REACH`
  rather than the `MISSING` they were expected to be. Denominator confidence stays
  `PARTIAL` — the cell set grew by five without a single new sensor, which is
  evidence the space was under-declared, not that it is now complete.
